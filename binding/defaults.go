package binding

import (
	"reflect"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/schema"
)

func findDefaults[I any](registry core.Registry) *FindResult[any] {
	t := reflect.TypeFor[I]()
	return findInType(t, nil, func(sf reflect.StructField, i []int) any {
		if d := sf.Tag.Get("default"); d != "" {
			if sf.Type.Kind() == reflect.Pointer && sf.Type.Elem().Kind() == reflect.Struct {
				panic("pointers to structs cannot have default values")
			}
			s := registry.Schema(sf.Type, true, "")
			if s == nil {
				return d
			}
			return schema.ConvertType(sf.Name, sf.Type, schema.JsonTagValue(registry, sf.Name, s, d))
		}
		return nil
	}, true)
}
