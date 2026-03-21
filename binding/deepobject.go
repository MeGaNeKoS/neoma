package binding

import (
	"net/url"
	"reflect"
	"strings"

	"github.com/MeGaNeKoS/neoma/core"
)

// ParseDeepObjectQuery extracts key-value pairs from query parameters encoded
// in OpenAPI deepObject style (e.g., "filter[name]=value").
func ParseDeepObjectQuery(query url.Values, name string) map[string]string {
	result := make(map[string]string)
	for key, values := range query {
		if strings.Contains(key, "[") {
			keys := strings.Split(key, "[")
			if keys[0] != name {
				continue
			}
			k := strings.Trim(keys[1], "]")
			result[k] = values[0]
		}
	}
	return result
}

// SetDeepObjectValue populates a struct or map field from deep object query
// data, recording validation errors in res. It returns the parsed key-value
// pairs as a map for further validation.
func SetDeepObjectValue(pb *core.PathBuffer, res *core.ValidateResult, f reflect.Value, data map[string]string) map[string]any {
	t := f.Type()
	result := make(map[string]any)
	switch t.Kind() {
	case reflect.Map:
		if t.Key().Kind() != reflect.String {
			// Unsupported map key type; return empty result rather than panicking
			// since this is a user-controlled input path.
			return result
		}
		f.Set(reflect.MakeMap(t))
		for k, v := range data {
			key := reflect.New(t.Key()).Elem()
			key.SetString(k)
			value := reflect.New(t.Elem()).Elem()
			if err := setFieldValue(value, v); err != nil {
				pb.Push(k)
				res.Add(pb, v, err.Error())
				pb.Pop()
			} else {
				f.SetMapIndex(key, value)
				result[k] = value.Interface()
			}
		}
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			fieldName := field.Name
			if name := jsonName(field); name != "" {
				fieldName = name
			}

			fv := f.Field(i)
			if val, ok := data[fieldName]; ok {
				if err := setFieldValue(fv, val); err != nil {
					pb.Push(fieldName)
					res.Add(pb, val, err.Error())
					pb.Pop()
				} else {
					result[fieldName] = fv.Interface()
				}
			} else {
				if val := field.Tag.Get("default"); val != "" {
					// SetFieldValue may fail if the default value cannot be parsed
					// into the target type, but this is a programming error in the
					// struct definition, not a user input error, so we suppress it.
					_ = setFieldValue(fv, val)
					result[fieldName] = fv.Interface()
				}
			}
		}
	default:
	}
	return result
}
