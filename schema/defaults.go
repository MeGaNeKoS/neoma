package schema

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/MeGaNeKoS/neoma/core"
)

// DefaultResultPath represents a single default value located at a specific
// field path within a struct hierarchy.
type DefaultResultPath struct {
	Path  []int
	Value any
}

// DefaultResult holds a collection of default values discovered for a type,
// each associated with the struct field index path where the default applies.
type DefaultResult struct {
	Paths []DefaultResultPath
}

func (r *DefaultResult) every(current reflect.Value, path []int, v any, f func(reflect.Value, any)) {
	if len(path) == 0 {
		f(current, v)
		return
	}

	current = reflect.Indirect(current)
	if current.Kind() == reflect.Invalid {
		// Indirect may have resulted in no value, for example an optional field
		// that's a pointer may have been omitted; just ignore it.
		return
	}

	switch current.Kind() {
	case reflect.Struct:
		r.every(current.Field(path[0]), path[1:], v, f)
	case reflect.Slice:
		for j := 0; j < current.Len(); j++ {
			r.every(current.Index(j), path, v, f)
		}
	case reflect.Map:
		for _, k := range current.MapKeys() {
			r.every(current.MapIndex(k), path, v, f)
		}
	default:
		panic("unsupported kind in default traversal: " + current.Kind().String())
	}
}

// Every calls f for each default value path, traversing into the given value
// to locate the target field. Slices and maps are traversed recursively.
func (r *DefaultResult) Every(v reflect.Value, f func(reflect.Value, any)) {
	for i := range r.Paths {
		r.every(v, r.Paths[i].Path, r.Paths[i].Value, f)
	}
}

func (r *DefaultResult) everyPB(current reflect.Value, path []int, pb *core.PathBuffer, v any, f func(reflect.Value, any)) {
	switch reflect.Indirect(current).Kind() {
	case reflect.Slice, reflect.Map:
		// Ignore these. We only care about the leaf nodes.
	default:
		if len(path) == 0 {
			f(current, v)
			return
		}
	}

	current = reflect.Indirect(current)
	if current.Kind() == reflect.Invalid {
		return
	}

	switch current.Kind() {
	case reflect.Struct:
		pops := 0
		field := current.Type().Field(path[0])
		if !field.Anonymous {
			pops++
			if pathTag := field.Tag.Get("path"); pathTag != "" && pb.Len() == 0 {
				pb.Push("path")
				pb.Push(pathTag)
				pops++
			} else if query := field.Tag.Get("query"); query != "" && pb.Len() == 0 {
				pb.Push("query")
				pb.Push(query)
				pops++
			} else if header := field.Tag.Get("header"); header != "" && pb.Len() == 0 {
				pb.Push("header")
				pb.Push(header)
				pops++
			} else {
				pb.Push(jsonFieldName(field))
			}
		}
		r.everyPB(current.Field(path[0]), path[1:], pb, v, f)
		for i := 0; i < pops; i++ {
			pb.Pop()
		}
	case reflect.Slice:
		for j := 0; j < current.Len(); j++ {
			pb.PushIndex(j)
			r.everyPB(current.Index(j), path, pb, v, f)
			pb.Pop()
		}
	case reflect.Map:
		for _, k := range current.MapKeys() {
			if k.Kind() == reflect.String {
				pb.Push(k.String())
			} else {
				pb.Push(fmt.Sprintf("%v", k.Interface()))
			}
			r.everyPB(current.MapIndex(k), path, pb, v, f)
			pb.Pop()
		}
	default:
		panic("unsupported kind in default traversal: " + current.Kind().String())
	}
}

// EveryPB is like Every but also tracks the JSON path using a PathBuffer,
// which is useful for reporting the location of applied defaults.
func (r *DefaultResult) EveryPB(pb *core.PathBuffer, v reflect.Value, f func(reflect.Value, any)) {
	for i := range r.Paths {
		pb.Reset()
		r.everyPB(v, r.Paths[i].Path, pb, r.Paths[i].Value, f)
	}
}

// ApplyDefaults sets zero-valued fields in v to their default values as
// described by the given DefaultResult.
func ApplyDefaults(defaults *DefaultResult, v reflect.Value) {
	defaults.Every(v, func(item reflect.Value, def any) {
		if item.IsZero() {
			if item.Kind() == reflect.Pointer {
				item.Set(reflect.New(item.Type().Elem()))
				item = item.Elem()
			}
			item.Set(reflect.Indirect(reflect.ValueOf(def)))
		}
	})
}

// FindDefaults walks the given type and returns all fields that have a
// "default" struct tag, along with their parsed default values.
func FindDefaults(registry core.Registry, t reflect.Type) *DefaultResult {
	result := &DefaultResult{}
	findDefaultsInType(registry, t, nil, result, make(map[reflect.Type]struct{}))
	return result
}

func findDefaultsInType(registry core.Registry, t reflect.Type, path []int, result *DefaultResult, visited map[reflect.Type]struct{}) {
	t = deref(t)

	switch t.Kind() {
	case reflect.Struct:
		if _, ok := visited[t]; ok {
			return
		}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			if slices.Contains([]string{"Status", "Body"}, f.Name) {
				continue
			}

			fi := append([]int{}, path...)
			fi = append(fi, i)

			if d := f.Tag.Get("default"); d != "" {
				if f.Type.Kind() == reflect.Pointer && f.Type.Elem().Kind() == reflect.Struct {
					panic("pointers to structs cannot have default values")
				}
				s := registry.Schema(f.Type, true, "")
				if s == nil {
					continue
				}
				val := ConvertType(f.Type.Name(), f.Type, JsonTagValue(registry, f.Name, s, d))
				if val != nil {
					result.Paths = append(result.Paths, DefaultResultPath{fi, val})
				}
			}

			// Always recurse into embedded structs and named fields.
			visited[t] = struct{}{}
			findDefaultsInType(registry, f.Type, fi, result, visited)
			delete(visited, t)
		}
	case reflect.Slice:
		findDefaultsInType(registry, t.Elem(), path, result, visited)
	case reflect.Map:
		findDefaultsInType(registry, t.Elem(), path, result, visited)
	default:
	}
}

func jsonFieldName(field reflect.StructField) string {
	name := strings.ToLower(field.Name)
	if jsonName := field.Tag.Get("json"); jsonName != "" {
		name = strings.Split(jsonName, ",")[0]
	}
	return name
}
