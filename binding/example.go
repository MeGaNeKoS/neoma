package binding

import (
	"reflect"
	"strings"
	"time"

	"github.com/MeGaNeKoS/neoma/core"
)

var exampleProviderType = reflect.TypeFor[core.ExampleProvider]()

func anyFieldHasExample(t reflect.Type) bool {
	t = core.Deref(t)
	if t.Kind() != reflect.Struct {
		return false
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		if f.Tag.Get("example") != "" {
			return true
		}
		ft := core.Deref(f.Type)
		if ft.Kind() == reflect.Struct && ft != timeType {
			if anyFieldHasExample(ft) {
				return true
			}
		}
	}
	return false
}

func buildExampleFromType(t reflect.Type) any {
	t = core.Deref(t)
	if t.Kind() != reflect.Struct {
		return nil
	}

	v := reflect.New(t)
	if buildExampleValue(v.Elem()) {
		return v.Interface()
	}
	return nil
}

func buildExampleMap(v reflect.Value, t reflect.Type) bool {
	valType := core.Deref(t.Elem())
	if valType.Kind() != reflect.Struct {
		return false
	}

	elem := reflect.New(valType)
	if buildExampleValue(elem.Elem()) {
		m := reflect.MakeMap(t)
		key := reflect.New(t.Key()).Elem()
		if t.Key().Kind() == reflect.String {
			key.SetString("example")
		}
		if t.Elem().Kind() == reflect.Pointer {
			m.SetMapIndex(key, elem)
		} else {
			m.SetMapIndex(key, elem.Elem())
		}
		v.Set(m)
		return true
	}
	return false
}

func buildExampleSlice(v reflect.Value, t reflect.Type) bool {
	elemType := core.Deref(t.Elem())
	if elemType.Kind() != reflect.Struct {
		return false
	}

	elem := reflect.New(elemType)
	if buildExampleValue(elem.Elem()) {
		slice := reflect.MakeSlice(t, 1, 1)
		if t.Elem().Kind() == reflect.Pointer {
			slice.Index(0).Set(elem)
		} else {
			slice.Index(0).Set(elem.Elem())
		}
		v.Set(slice)
		return true
	}
	return false
}

func buildExampleStruct(v reflect.Value, t reflect.Type) bool {
	anySet := false

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		fv := v.Field(i)
		fieldType := field.Type

		isPtr := fieldType.Kind() == reflect.Pointer
		if isPtr {
			fieldType = fieldType.Elem()
		}

		exampleTag := field.Tag.Get("example")
		if exampleTag != "" {
			if isPtr {
				if fv.IsNil() {
					fv.Set(reflect.New(fieldType))
				}
				if setExampleFromTag(fv.Elem(), fieldType, exampleTag) {
					anySet = true
				}
			} else if setExampleFromTag(fv, fieldType, exampleTag) {
				anySet = true
			}
			continue
		}

		actual := fv
		if isPtr {
			if actual.IsNil() {
				bt := core.BaseType(fieldType)
				if bt.Kind() == reflect.Struct {
					actual.Set(reflect.New(fieldType))
					actual = actual.Elem()
				} else {
					continue
				}
			} else {
				actual = actual.Elem()
			}
		}

		switch {
		case buildExampleValue(actual):
			anySet = true
		case isPtr && fv.IsNil():
		case isPtr && !anyFieldHasExample(fieldType):
			fv.Set(reflect.Zero(field.Type))
		}
	}

	return anySet
}

func buildExampleValue(v reflect.Value) bool {
	if !v.IsValid() || !v.CanSet() {
		return false
	}

	t := v.Type()

	if reflect.PointerTo(t).Implements(exampleProviderType) {
		ptr := reflect.New(t)
		ptr.Elem().Set(v)
		result := ptr.MethodByName("Example").Call(nil)
		if len(result) > 0 && !result[0].IsNil() {
			ex := result[0].Interface()
			rv := reflect.ValueOf(ex)
			if rv.Type().AssignableTo(t) {
				v.Set(rv)
			} else if rv.Type().ConvertibleTo(t) {
				v.Set(rv.Convert(t))
			}
			return true
		}
	}

	switch t.Kind() {
	case reflect.Struct:
		return buildExampleStruct(v, t)
	case reflect.Slice:
		return buildExampleSlice(v, t)
	case reflect.Map:
		return buildExampleMap(v, t)
	default:
	}

	return false
}

func setExampleFromTag(v reflect.Value, t reflect.Type, tag string) bool {
	if t == timeType {
		parsed, err := time.Parse(time.RFC3339, tag)
		if err != nil {
			parsed, err = time.Parse(time.RFC3339Nano, tag)
			if err != nil {
				parsed, err = time.Parse("2006-01-02", tag)
				if err != nil {
					return false
				}
			}
		}
		v.Set(reflect.ValueOf(parsed))
		return true
	}

	if t.Kind() == reflect.Slice {
		if t.Elem().Kind() == reflect.String {
			parts := strings.Split(tag, ",")
			slice := reflect.MakeSlice(t, len(parts), len(parts))
			for i, p := range parts {
				slice.Index(i).SetString(strings.TrimSpace(p))
			}
			v.Set(slice)
			return true
		}
		return false
	}

	if err := parseScalar(v, tag); err != nil {
		return false
	}
	return true
}
