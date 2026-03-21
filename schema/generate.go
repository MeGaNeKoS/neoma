package schema

import (
	"encoding"
	"encoding/json"
	"fmt"
	"math/bits"
	"net"
	"net/netip"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/MeGaNeKoS/neoma/core"
)

// DefaultArrayNullable controls whether generated schemas for slice types
// are nullable by default. Set to false to make arrays non-nullable.
var DefaultArrayNullable = true

var (
	ipType                = reflect.TypeFor[net.IP]()
	ipAddrType            = reflect.TypeFor[netip.Addr]()
	rawMessageType        = reflect.TypeFor[json.RawMessage]()
	schemaProviderType    = reflect.TypeFor[core.SchemaProvider]()
	schemaTransformerType = reflect.TypeFor[core.SchemaTransformer]()
	textUnmarshalerType   = reflect.TypeFor[encoding.TextUnmarshaler]()
	timeType              = reflect.TypeFor[time.Time]()
	urlType               = reflect.TypeFor[url.URL]()
)

var deref = core.Deref

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

func intTag(f reflect.StructField, tag string) *int {
	if v := f.Tag.Get(tag); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return &i
		} else {
			panic(fmt.Errorf("invalid int tag '%s' for field '%s': %v (%w)", tag, f.Name, v, err))
		}
	}
	return nil
}

func floatTag(f reflect.StructField, tag string) *float64 {
	if v := f.Tag.Get(tag); v != "" {
		if i, err := strconv.ParseFloat(v, 64); err == nil {
			return &i
		} else {
			panic(fmt.Errorf("invalid float tag '%s' for field '%s': %v (%w)", tag, f.Name, v, err))
		}
	}
	return nil
}

func stringTag(f reflect.StructField, tag string, def string) string {
	if v := f.Tag.Get(tag); v != "" {
		return v
	}
	return def
}

func ensureType(r core.Registry, fieldName string, s *core.Schema, value string, v any) {
	if s.Ref != "" {
		s = r.SchemaFromRef(s.Ref)
		if s == nil {
			// We may not have access to this type, e.g., custom schema provided
			// by the user with remote refs. Skip validation.
			return
		}
	}

	switch s.Type {
	case core.TypeBoolean:
		if _, ok := v.(bool); !ok {
			panic(fmt.Errorf("invalid boolean tag value '%s' for field '%s': %w", value, fieldName, core.ErrSchemaInvalid))
		}
	case core.TypeInteger, core.TypeNumber:
		if _, ok := v.(float64); !ok {
			panic(fmt.Errorf("invalid number tag value '%s' for field '%s': %w", value, fieldName, core.ErrSchemaInvalid))
		}

		if s.Type == core.TypeInteger {
			fv, _ := v.(float64)
			if fv != float64(int(fv)) {
				panic(fmt.Errorf("invalid integer tag value '%s' for field '%s': %w", value, fieldName, core.ErrSchemaInvalid))
			}
		}
	case core.TypeString:
		if _, ok := v.(string); !ok {
			panic(fmt.Errorf("invalid string tag value '%s' for field '%s': %w", value, fieldName, core.ErrSchemaInvalid))
		}
	case core.TypeArray:
		arr, ok := v.([]any)
		if !ok {
			panic(fmt.Errorf("invalid array tag value '%s' for field '%s': %w", value, fieldName, core.ErrSchemaInvalid))
		}

		if s.Items != nil {
			for i, item := range arr {
				b, _ := json.Marshal(item)
				ensureType(r, fieldName+"["+strconv.Itoa(i)+"]", s.Items, string(b), item)
			}
		}
	case core.TypeObject:
		obj, ok := v.(map[string]any)
		if !ok {
			panic(fmt.Errorf("invalid object tag value '%s' for field '%s': %w", value, fieldName, core.ErrSchemaInvalid))
		}

		for name, prop := range s.Properties {
			if val, exists := obj[name]; exists {
				b, _ := json.Marshal(val)
				ensureType(r, fieldName+"."+name, prop, string(b), val)
			}
		}
	}
}

func convertType(fieldName string, t reflect.Type, v any) any {
	if v == nil {
		return v
	}

	tv := reflect.TypeOf(v)
	if tv == t {
		return v
	}

	// Directly convert equal underlying types, avoiding traversal.
	// e.g., json.RawMessage -> []byte.
	if tv.ConvertibleTo(t) {
		return reflect.ValueOf(v).Convert(t).Interface()
	}

	val := reflect.ValueOf(v)

	if tv.Kind() == reflect.Slice {
		// Slices can't be cast due to the different layouts. Instead, we make a
		// new instance of the destination slice, and convert each value in
		// the original to the new type.
		tmp := reflect.MakeSlice(t, 0, val.Len())
		for i := 0; i < val.Len(); i++ {
			item := val.Index(i)
			if item.Kind() == reflect.Interface {
				// E.g. []any and we want the underlying type.
				item = item.Elem()
			}

			item = reflect.Indirect(item)
			typ := deref(t.Elem())
			if !item.Type().ConvertibleTo(typ) {
				panic(fmt.Errorf("unable to convert %v to %v for field '%s': %w", item.Interface(), t.Elem(), fieldName, core.ErrSchemaInvalid))
			}

			value := item.Convert(typ)
			if t.Elem().Kind() == reflect.Pointer {
				// Special case: if the field is a pointer, we need to get a pointer
				// to the converted value.
				ptr := reflect.New(value.Type())
				ptr.Elem().Set(value)
				value = ptr
			}

			tmp = reflect.Append(tmp, value)
		}

		return tmp.Interface()
	}

	if !tv.ConvertibleTo(deref(t)) {
		panic(fmt.Errorf("unable to convert %v to %v for field '%s': %w", tv, t, fieldName, core.ErrSchemaInvalid))
	}

	converted := val.Convert(deref(t))
	if t.Kind() == reflect.Pointer {
		// Special case: if the field is a pointer, we need to get a pointer
		// to the converted value.
		tmp := reflect.New(t.Elem())
		tmp.Elem().Set(converted)
		converted = tmp
	}

	return converted.Interface()
}

// JsonTagValue parses a struct tag value string into a Go value according
// to the schema's type. Strings are returned as-is, while other types are
// parsed as JSON.
func JsonTagValue(r core.Registry, fieldName string, s *core.Schema, value string) any {
	if s.Ref != "" {
		s = r.SchemaFromRef(s.Ref)
		if s == nil {
			return nil
		}
	}

	// Special case: strings don't need quotes.
	if s.Type == core.TypeString {
		return value
	}

	// Special case: array of strings with comma-separated values and no quotes.
	if s.Type == core.TypeArray && s.Items != nil && s.Items.Type == core.TypeString && value[0] != '[' {
		var values []string
		for s := range strings.SplitSeq(value, ",") {
			values = append(values, strings.TrimSpace(s))
		}
		return values
	}

	var v any
	if err := json.Unmarshal([]byte(value), &v); err != nil {
		panic(fmt.Errorf("invalid %s tag value '%s' for field '%s': %w", s.Type, value, fieldName, err))
	}

	ensureType(r, fieldName, s, value, v)

	return v
}

func jsonTag(r core.Registry, f reflect.StructField, s *core.Schema, name string) any {
	t := f.Type
	if value := f.Tag.Get(name); value != "" {
		return convertType(f.Name, t, JsonTagValue(r, f.Name, s, value))
	}
	return nil
}

// ConvertType converts a value to the target type, handling slices, pointers,
// and scalar conversions. It panics if the conversion is not possible.
func ConvertType(fieldName string, t reflect.Type, v any) any {
	return convertType(fieldName, t, v)
}

func targetSchema(fs *core.Schema) *core.Schema {
	if fs.Type == core.TypeArray {
		return fs.Items
	}
	return fs
}

type fieldInfo struct {
	Parent reflect.Type
	Field  reflect.StructField
}

func getFields(typ reflect.Type, visited map[reflect.Type]struct{}) []fieldInfo {
	fields := make([]fieldInfo, 0, typ.NumField())
	var embedded []reflect.StructField

	if _, ok := visited[typ]; ok {
		return fields
	}
	visited[typ] = struct{}{}

	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if !f.IsExported() {
			continue
		}

		if f.Anonymous && f.Tag.Get("json") == "" {
			embedded = append(embedded, f)
			continue
		}

		fields = append(fields, fieldInfo{typ, f})
	}

	for _, f := range embedded {
		newTyp := f.Type
		for newTyp.Kind() == reflect.Pointer {
			newTyp = newTyp.Elem()
		}
		if newTyp.Kind() == reflect.Struct {
			fields = append(fields, getFields(newTyp, visited)...)
		}
	}

	return fields
}

// FromField generates a schema for a struct field, applying struct tag
// constraints such as doc, format, default, enum, minimum, maximum, pattern,
// and others.
func FromField(registry core.Registry, f reflect.StructField, hint string) *core.Schema {
	fs := registry.Schema(f.Type, true, hint)
	if fs == nil {
		return fs
	}

	// Support int64 as string for JavaScript safety (#698). When the json tag
	// contains ",string" and the field is an int64 or uint64, the JSON encoder
	// will emit the value as a quoted string. Reflect that in the schema so
	// that OpenAPI consumers know to expect a string representation.
	if j := f.Tag.Get("json"); strings.Contains(j, ",string") {
		bt := core.BaseType(f.Type)
		if bt.Kind() == reflect.Int64 || bt.Kind() == reflect.Uint64 {
			fs.Type = core.TypeString
			// Keep the existing format (int64) so consumers know the
			// semantic meaning of the string value.
		}
	}

	fs.Description = stringTag(f, "doc", fs.Description)
	if fs.Format == "date-time" && f.Tag.Get("header") != "" {
		// Special case: this is a header and uses a different date/time format.
		// Note that it can still be overridden by the format or timeFormat
		// tags later.
		fs.Format = "date-time-http"
	}
	if format := f.Tag.Get("format"); format != "" {
		targetSchema(fs).Format = format
	}
	if timeFmt := f.Tag.Get("timeFormat"); timeFmt != "" {
		s := targetSchema(fs)
		switch timeFmt {
		case "2006-01-02":
			s.Format = "date"
		case "15:04:05":
			s.Format = "time"
		default:
			s.Format = timeFmt
		}
	}
	fs.ContentEncoding = stringTag(f, "encoding", fs.ContentEncoding)
	if defaultValue := jsonTag(registry, f, fs, "default"); defaultValue != nil {
		fs.Default = defaultValue
	}

	if value, ok := f.Tag.Lookup("example"); ok {
		if e := JsonTagValue(registry, f.Name, fs, value); e != nil {
			fs.Examples = []any{e}
		}
	}

	if enum := f.Tag.Get("enum"); enum != "" {
		s := targetSchema(fs)
		var enumValues []any
		for e := range strings.SplitSeq(enum, ",") {
			enumValues = append(enumValues, JsonTagValue(registry, f.Name, s, e))
		}
		s.Enum = enumValues
	}

	fs.Nullable = boolTag(f, "nullable", fs.Nullable)
	if fs.Nullable && fs.Ref != "" {
		refSchema := registry.SchemaFromRef(fs.Ref)
		if refSchema != nil && refSchema.Type == "object" {
			panic(fmt.Errorf("nullable is not supported for field '%s' which is type '%s'", f.Name, fs.Ref))
		}
	}

	if v := floatTag(f, "minimum"); v != nil {
		targetSchema(fs).Minimum = v
	}
	if v := floatTag(f, "exclusiveMinimum"); v != nil {
		targetSchema(fs).ExclusiveMinimum = v
	}
	if v := floatTag(f, "maximum"); v != nil {
		targetSchema(fs).Maximum = v
	}
	if v := floatTag(f, "exclusiveMaximum"); v != nil {
		targetSchema(fs).ExclusiveMaximum = v
	}
	if v := floatTag(f, "multipleOf"); v != nil {
		targetSchema(fs).MultipleOf = v
	}
	if v := intTag(f, "minLength"); v != nil {
		targetSchema(fs).MinLength = v
	}
	if v := intTag(f, "maxLength"); v != nil {
		targetSchema(fs).MaxLength = v
	}
	if v := f.Tag.Get("pattern"); v != "" {
		targetSchema(fs).Pattern = v
	}
	if v := f.Tag.Get("patternDescription"); v != "" {
		targetSchema(fs).PatternDescription = v
	}
	if v := intTag(f, "minItems"); v != nil {
		fs.MinItems = v
	}
	if v := intTag(f, "maxItems"); v != nil {
		fs.MaxItems = v
	}
	if v := intTag(f, "minProperties"); v != nil {
		fs.MinProperties = v
	}
	if v := intTag(f, "maxProperties"); v != nil {
		fs.MaxProperties = v
	}
	fs.UniqueItems = boolTag(f, "uniqueItems", fs.UniqueItems)
	fs.ReadOnly = boolTag(f, "readOnly", fs.ReadOnly)
	fs.WriteOnly = boolTag(f, "writeOnly", fs.WriteOnly)
	fs.Deprecated = boolTag(f, "deprecated", fs.Deprecated)

	// Support oneOf, anyOf, allOf struct tags (#761). Each tag is a
	// comma-separated list of type names that are resolved as $ref
	// references using the registry prefix.
	if v := f.Tag.Get("oneOf"); v != "" {
		fs.OneOf = buildCompositionRefs(v)
		// When composition keywords are present, clear the type so it does
		// not conflict with the oneOf/anyOf/allOf constraint.
		fs.Type = ""
		fs.Format = ""
	}
	if v := f.Tag.Get("anyOf"); v != "" {
		fs.AnyOf = buildCompositionRefs(v)
		fs.Type = ""
		fs.Format = ""
	}
	if v := f.Tag.Get("allOf"); v != "" {
		fs.AllOf = buildCompositionRefs(v)
		fs.Type = ""
		fs.Format = ""
	}

	fs.PrecomputeMessages()

	fs.Hidden = boolTag(f, "hidden", fs.Hidden)

	return fs
}

func buildCompositionRefs(tagValue string) []*core.Schema {
	var refs []*core.Schema
	for part := range strings.SplitSeq(tagValue, ",") {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		ref := schemaRefPrefix + name
		refs = append(refs, &core.Schema{Ref: ref})
	}
	return refs
}

// FromType generates a JSON Schema from a Go type. It handles primitives,
// slices, maps, structs, and types that implement core.SchemaProvider or
// core.SchemaTransformer.
func FromType(r core.Registry, t reflect.Type) *core.Schema {
	s := schemaFromType(r, t)
	if s == nil {
		return nil
	}
	t = deref(t)

	ptrT := reflect.PointerTo(t)
	if t.Implements(schemaTransformerType) || ptrT.Implements(schemaTransformerType) {
		if st, ok := reflect.New(t).Interface().(core.SchemaTransformer); ok {
			s = st.TransformSchema(r, s)
		}

		// The schema may have been modified, so recompute the error messages.
		s.PrecomputeMessages()
	}
	return s
}

func schemaFromType(r core.Registry, t reflect.Type) *core.Schema {
	isPointer := t.Kind() == reflect.Pointer

	s := core.Schema{}
	t = deref(t)

	ptrT := reflect.PointerTo(t)
	if t.Implements(schemaProviderType) || ptrT.Implements(schemaProviderType) {
		// Special case: type provides its own schema. Do not try to generate.
		sp, _ := reflect.New(t).Interface().(core.SchemaProvider)
		custom := sp.Schema(r)
		custom.PrecomputeMessages()
		return custom
	}

	switch t {
	case timeType:
		return &core.Schema{Type: core.TypeString, Nullable: isPointer, Format: "date-time"}
	case urlType:
		return &core.Schema{Type: core.TypeString, Nullable: isPointer, Format: "uri"}
	case ipType:
		return &core.Schema{Type: core.TypeString, Nullable: isPointer, Format: "ipv4"}
	case ipAddrType:
		return &core.Schema{Type: core.TypeString, Nullable: isPointer, Format: "ip"}
	case rawMessageType:
		return &core.Schema{}
	}

	if t.Implements(textUnmarshalerType) || ptrT.Implements(textUnmarshalerType) {
		// Special case: types that implement encoding.TextUnmarshaler are able to
		// be loaded from plain text, and so should be treated as strings.
		// This behavior can be overridden by implementing core.SchemaProvider
		// and returning a custom schema.
		return &core.Schema{Type: core.TypeString, Nullable: isPointer}
	}

	minZero := 0.0
	switch t.Kind() {
	case reflect.Bool:
		s.Type = core.TypeBoolean
	case reflect.Int:
		s.Type = core.TypeInteger
		if bits.UintSize == 32 {
			s.Format = "int32"
		} else {
			s.Format = "int64"
		}
	case reflect.Int8, reflect.Int16, reflect.Int32:
		s.Type = core.TypeInteger
		s.Format = "int32"
	case reflect.Int64:
		s.Type = core.TypeInteger
		s.Format = "int64"
	case reflect.Uint:
		s.Type = core.TypeInteger
		if bits.UintSize == 32 {
			s.Format = "int32"
		} else {
			s.Format = "int64"
		}
		s.Minimum = &minZero
	case reflect.Uint8, reflect.Uint16, reflect.Uint32:
		// Unsigned integers can't be negative.
		s.Type = core.TypeInteger
		s.Format = "int32"
		s.Minimum = &minZero
	case reflect.Uint64:
		// Unsigned integers can't be negative.
		s.Type = core.TypeInteger
		s.Format = "int64"
		s.Minimum = &minZero
	case reflect.Float32:
		s.Type = core.TypeNumber
		s.Format = "float"
	case reflect.Float64:
		s.Type = core.TypeNumber
		s.Format = "double"
	case reflect.String:
		s.Type = core.TypeString
	case reflect.Slice, reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 {
			// Special case: []byte will be serialized as a base64 string.
			s.Type = core.TypeString
			s.ContentEncoding = "base64"
		} else {
			s.Type = core.TypeArray
			s.Nullable = DefaultArrayNullable
			s.Items = r.Schema(t.Elem(), true, t.Name()+"Item")

			if t.Kind() == reflect.Array {
				l := t.Len()
				s.MinItems = &l
				s.MaxItems = &l
			}
		}
	case reflect.Map:
		s.Type = core.TypeObject
		s.AdditionalProperties = r.Schema(t.Elem(), true, t.Name()+"Value")
	case reflect.Struct:
		var required []string
		requiredMap := map[string]bool{}
		var propNames []string
		fieldSet := map[string]struct{}{}
		props := map[string]*core.Schema{}
		dependentRequiredMap := map[string][]string{}
		for _, info := range getFields(t, make(map[reflect.Type]struct{})) {
			f := info.Field

			if _, ok := fieldSet[f.Name]; ok {
				// This field was overridden by an ancestor type, so we
				// should ignore it.
				continue
			}

			fieldSet[f.Name] = struct{}{}

			// Controls whether the field is required or not. All fields start as
			// required (unless the registry says otherwise), then can be made
			// optional with the omitempty JSON tag, omitzero JSON tag, or it
			// can be overridden manually via the required tag.
			fieldRequired := !getRegistryConfig(r).FieldsOptionalByDefault

			name := f.Name
			if j := f.Tag.Get("json"); j != "" {
				if n := strings.Split(j, ",")[0]; n != "" {
					name = n
				}
				if strings.Contains(j, "omitempty") {
					fieldRequired = false
				}
				if strings.Contains(j, "omitzero") {
					fieldRequired = false
				}
			}
			if name == "-" {
				continue
			}

			if _, ok := f.Tag.Lookup("required"); ok {
				fieldRequired = boolTag(f, "required", false)
			}

			if dr := f.Tag.Get("dependentRequired"); strings.TrimSpace(dr) != "" {
				dependentRequiredMap[name] = strings.Split(dr, ",")
			}

			fs := FromField(r, f, t.Name()+f.Name+"Struct")
			if fs != nil {
				props[name] = fs
				propNames = append(propNames, name)

				if fs.Hidden {
					fieldRequired = false
				}

				if fieldRequired {
					required = append(required, name)
					requiredMap[name] = true
				}

				// Special case: pointer with omitempty and not manually set to
				// nullable, which will never get null sent over the wire.
				if f.Type.Kind() == reflect.Pointer && strings.Contains(f.Tag.Get("json"), "omitempty") && f.Tag.Get("nullable") != "true" {
					fs.Nullable = false
				}
			}
		}
		s.Type = core.TypeObject

		var errs []string
		depKeys := make([]string, 0, len(dependentRequiredMap))
		for field := range dependentRequiredMap {
			depKeys = append(depKeys, field)
		}
		sort.Strings(depKeys)
		for _, field := range depKeys {
			dependents := dependentRequiredMap[field]
			for _, dependent := range dependents {
				if _, ok := props[dependent]; ok {
					continue
				}
				errs = append(errs, fmt.Sprintf("dependent field '%s' for field '%s' does not exist", dependent, field))
			}
		}
		if errs != nil {
			panic(fmt.Errorf("%s", strings.Join(errs, "; ")))
		}

		additionalProps := getRegistryConfig(r).AllowAdditionalPropertiesByDefault
		if f, ok := t.FieldByName("_"); ok {
			if _, ok = f.Tag.Lookup("additionalProperties"); ok {
				additionalProps = boolTag(f, "additionalProperties", false)
			}

			if _, ok = f.Tag.Lookup("nullable"); ok {
				// Allow overriding nullability per struct.
				s.Nullable = boolTag(f, "nullable", false)
			}
		}

		s.AdditionalProperties = additionalProps
		s.Properties = props
		s.PropertyNames = propNames
		s.Required = required
		s.DependentRequired = dependentRequiredMap
		s.RequiredMap = requiredMap
		s.PrecomputeMessages()
	case reflect.Interface:
		// Interfaces mean any object.
	default:
		return nil
	}

	switch s.Type {
	case core.TypeBoolean, core.TypeInteger, core.TypeNumber, core.TypeString:
		// Scalar types which are pointers are nullable by default. This can be
		// overridden via the nullable:"false" field tag in structs.
		s.Nullable = isPointer
	}

	return &s
}
