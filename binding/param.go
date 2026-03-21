package binding

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/schema"
)

const styleDeepObject = "deepObject"

var (
	cookieType       = reflect.TypeFor[http.Cookie]()
	fmtStringerType  = reflect.TypeFor[fmt.Stringer]()
	paramWrapperType = reflect.TypeFor[ParamWrapper]()
	stringType       = reflect.TypeFor[string]()
	stringSliceType  = reflect.TypeFor[[]string]()
	timeType         = reflect.TypeFor[time.Time]()
	urlType          = reflect.TypeFor[url.URL]()
)


// ParamWrapper is implemented by types that wrap a parameter value and provide
// the actual reflect.Value that should receive the parsed data.
type ParamWrapper interface {
	Receiver() reflect.Value
}

// ParamReactor is implemented by types that need to be notified after a
// parameter is bound, receiving whether the parameter was present and its
// parsed value.
type ParamReactor interface {
	OnParamSet(isSet bool, parsed any)
}

// ParamFieldInfo describes a single parameter field, including its type, name,
// location (path/query/header/cookie), default value, and serialization style.
type ParamFieldInfo struct {
	Type       reflect.Type
	Name       string
	Loc        string
	Required   bool
	Default    string
	TimeFormat string
	Explode    bool
	Style      string
	Schema     *core.Schema
	IsPointer  bool // true when the original field is a pointer type (#393)
}

// ParamLocation pairs a ParamFieldInfo with its explode setting for OpenAPI
// parameter documentation.
type ParamLocation struct {
	Explode *bool
	PFI     *ParamFieldInfo
}


// GetParamValue extracts the raw string value for a parameter from the request
// context based on its location (path, query, header, or cookie), falling back
// to the default value if empty.
func GetParamValue(p ParamFieldInfo, ctx core.Context, cookies map[string]*http.Cookie) string {
	var value string
	switch p.Loc {
	case "path":
		value = ctx.Param(p.Name)
	case "query":
		value = ctx.Query(p.Name)
	case "header":
		value = ctx.Header(p.Name)
	case "cookie":
		if c, ok := cookies[p.Name]; ok {
			value = c.Value
		}
	}
	if value == "" {
		value = p.Default
	}
	return value
}


func boolTag(f reflect.StructField, tag string, def bool) bool {
	if v := f.Tag.Get(tag); v != "" {
		switch v {
		case "true":
			return true
		case "false":
			return false
		default:
			panic(fmt.Errorf("invalid bool tag '%s' for field '%s': %v", tag, f.Name, v))
		}
	}
	return def
}

func documentParam(op *core.Operation, pl *ParamLocation) {
	pfi := pl.PFI
	for _, existing := range op.Parameters {
		if existing.Name == pfi.Name && existing.In == pfi.Loc {
			return
		}
	}

	desc := ""
	if pfi.Schema != nil {
		// Some tools will not show the description if it is only on the schema.
		desc = pfi.Schema.Description
	}

	op.Parameters = append(op.Parameters, &core.Param{
		Name:        pfi.Name,
		Description: desc,
		In:          pfi.Loc,
		Explode:     pl.Explode,
		Required:    pfi.Required,
		Schema:      pfi.Schema,
		Style:       pfi.Style,
	})
}

func findParams[I any](registry core.Registry, op *core.Operation, fieldsOptionalByDefault bool) *FindResult[*ParamFieldInfo] {
	t := reflect.TypeFor[I]()
	return findInType(t, nil, func(f reflect.StructField, path []int) *ParamFieldInfo {
		if f.Anonymous {
			return nil
		}

		pl, ok := parseParamLocation(f, fieldsOptionalByDefault)
		if !ok {
			return nil
		}

		pfi := pl.PFI
		pfi.Schema = schema.FromField(registry, f, "")

		// While discouraged, make it possible to make query/header params required.
		if _, ok = f.Tag.Lookup("required"); ok {
			pfi.Required = boolTag(f, "required", false)
		}

		if pfi.Type == timeType {
			timeFormat := time.RFC3339Nano
			if pfi.Loc == "header" {
				timeFormat = http.TimeFormat
			}
			if v := f.Tag.Get("timeFormat"); v != "" {
				timeFormat = v
			}
			pfi.TimeFormat = timeFormat
		}

		if boolTag(f, "hidden", false) {
			if pfi.Loc != "form" {
				desc := ""
				if pfi.Schema != nil {
					desc = pfi.Schema.Description
				}
				op.HiddenParameters = append(op.HiddenParameters, &core.Param{
					Name:        pfi.Name,
					In:          pfi.Loc,
					Required:    pfi.Required,
					Schema:      pfi.Schema,
					Style:       pfi.Style,
					Explode:     pl.Explode,
					Description: desc,
				})
			}
		} else if pfi.Loc != "form" {
			documentParam(op, pl)
		}

		return pfi
	}, false, "Body")
}

func parseParamLocation(f reflect.StructField, fieldsOptionalByDefault bool) (*ParamLocation, bool) {
	pfi := &ParamFieldInfo{Type: f.Type}

	if reflect.PointerTo(f.Type).Implements(paramWrapperType) {
		if pw, ok := reflect.New(f.Type).Interface().(ParamWrapper); ok {
			pfi.Type = pw.Receiver().Type()
		}
	}

	if def := f.Tag.Get("default"); def != "" {
		pfi.Default = def
	}

	result := &ParamLocation{PFI: pfi}

	switch {
	case f.Tag.Get("path") != "":
		pfi.Loc = "path"
		pfi.Name = f.Tag.Get("path")
		pfi.Required = true
	case f.Tag.Get("query") != "":
		raw := f.Tag.Get("query")
		split := strings.Split(raw, ",")
		pfi.Loc = "query"
		pfi.Name = split[0]
		// If in is query then explode defaults to true. Parsing is much
		// easier if we use comma-separated values, so we disable explode by default.
		if slices.Contains(split[1:], "explode") {
			pfi.Explode = true
		}
		if slices.Contains(split[1:], styleDeepObject) {
			pfi.Style = styleDeepObject
		}
		result.Explode = &pfi.Explode
	case f.Tag.Get("header") != "":
		pfi.Loc = "header"
		pfi.Name = f.Tag.Get("header")
	case f.Tag.Get("form") != "":
		pfi.Loc = "form"
		pfi.Name = f.Tag.Get("form")
		pfi.Required = !fieldsOptionalByDefault
	case f.Tag.Get("cookie") != "":
		pfi.Loc = "cookie"
		pfi.Name = f.Tag.Get("cookie")
		if f.Type == cookieType {
			// Special case: parsed from a string input to an http.Cookie struct.
			pfi.Type = stringType
		}
	default:
		return nil, false
	}

	// Support pointer types for nullable query/header/cookie params (#393).
	// For path params, pointers are not supported because path segments are
	// always present by definition.
	if f.Type.Kind() == reflect.Pointer {
		if pfi.Loc == "path" {
			panic("pointers are not supported for path parameters")
		}
		pfi.IsPointer = true
		pfi.Type = f.Type.Elem()
	}

	return result, true
}
