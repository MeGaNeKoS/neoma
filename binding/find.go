package binding

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/MeGaNeKoS/neoma/core"
)


// FindResultPath pairs a struct field index path with its associated value
// found during type traversal.
type FindResultPath[T comparable] struct {
	Path  []int
	Value T
}

// FindResult holds the results of a recursive type scan, mapping struct field
// paths to their discovered values.
type FindResult[T comparable] struct {
	Paths []FindResultPath[T]
}


// Every traverses the value v and calls f for each field that matched during
// the original type scan, recursing into slices and maps.
func (r *FindResult[T]) Every(v reflect.Value, f func(reflect.Value, T)) {
	for i := range r.Paths {
		r.every(v, r.Paths[i].Path, r.Paths[i].Value, f)
	}
}

// EveryPB is like Every but also tracks the JSON/parameter path in pb for use
// in validation error reporting.
func (r *FindResult[T]) EveryPB(pb *core.PathBuffer, v reflect.Value, f func(reflect.Value, T)) {
	for i := range r.Paths {
		pb.Reset()
		r.everyPB(v, r.Paths[i].Path, pb, r.Paths[i].Value, f)
	}
}

func (r *FindResult[T]) every(current reflect.Value, path []int, v T, f func(reflect.Value, T)) {
	if len(path) == 0 {
		f(current, v)
		return
	}

	current = reflect.Indirect(current)
	if current.Kind() == reflect.Invalid {
		// Indirect may have resulted in no value, for example an optional field
		// that is a pointer may have been omitted; just ignore it.
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
		// Unsupported kind encountered during traversal; skip silently since
		// this can happen with user-provided types that have unexpected shapes.
	}
}

func (r *FindResult[T]) everyPB(current reflect.Value, path []int, pb *core.PathBuffer, v T, f func(reflect.Value, T)) {
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
		// Indirect may have resulted in no value, for example, an optional field
		// may have been omitted; just ignore it.
		return
	}

	switch current.Kind() {
	case reflect.Struct:
		field := current.Type().Field(path[0])
		pops := 0
		if !field.Anonymous {
			// The path name can come from one of four places: path parameter,
			// query parameter, header parameter, or body field.
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
				// The body is always in a field called "Body", which turns into
				// "body" in the path buffer, so we do not need to push it
				// separately like the params fields above.
				pb.Push(jsonName(field))
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
		// Unsupported kind encountered during path-buffer traversal; skip
		// silently since this can happen with user-provided types.
	}
}


func findInType[T comparable](t reflect.Type, onType func(reflect.Type, []int) T, onField func(reflect.StructField, []int) T, recurseFields bool, ignore ...string) *FindResult[T] {
	result := &FindResult[T]{}
	walkType(t, []int{}, result, onType, onField, recurseFields, make(map[reflect.Type]struct{}), ignore...)
	return result
}

func getHint(parent reflect.Type, name string, other string) string {
	if parent.Name() != "" {
		return parent.Name() + name
	}
	return other
}

func jsonName(field reflect.StructField) string {
	name := strings.ToLower(field.Name)
	if jsonName := field.Tag.Get("json"); jsonName != "" {
		name = strings.Split(jsonName, ",")[0]
	}
	return name
}

func walkType[T comparable](t reflect.Type, path []int, result *FindResult[T], onType func(reflect.Type, []int) T, onField func(reflect.StructField, []int) T, recurseFields bool, visited map[reflect.Type]struct{}, ignore ...string) {
	t = core.Deref(t)
	zero := reflect.Zero(reflect.TypeFor[T]()).Interface()

	ignoreAnonymous := false
	if onType != nil {
		if v := onType(t, path); v != zero {
			result.Paths = append(result.Paths, FindResultPath[T]{path, v})

			// Found what we were looking for in the type, no need to go deeper.
			// We do still want to potentially process each non-anonymous field,
			// so only skip anonymous ones.
			ignoreAnonymous = true
		}
	}

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
			if slices.Contains(ignore, f.Name) {
				continue
			}
			if ignoreAnonymous && f.Anonymous {
				continue
			}
			fi := append([]int{}, path...)
			fi = append(fi, i)
			if onField != nil {
				if v := onField(f, fi); v != zero {
					result.Paths = append(result.Paths, FindResultPath[T]{fi, v})
				}
			}
			if f.Anonymous || recurseFields || core.BaseType(f.Type).Kind() != reflect.Struct {
				// Always process embedded structs and named fields which are not
				// structs. If recurseFields is true, then we also process named
				// struct fields recursively.
				visited[t] = struct{}{}
				walkType[T](f.Type, fi, result, onType, onField, recurseFields, visited, ignore...)
				delete(visited, t)
			}
		}
	case reflect.Slice:
		walkType[T](t.Elem(), path, result, onType, onField, recurseFields, visited, ignore...)
	case reflect.Map:
		walkType[T](t.Elem(), path, result, onType, onField, recurseFields, visited, ignore...)
	default:
		// unsupported kind, skip
	}
}
