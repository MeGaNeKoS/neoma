package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"regexp"
	"sort"
	"strings"

)

const (
	TypeBoolean = "boolean"
	TypeInteger = "integer"
	TypeNumber  = "number"
	TypeString  = "string"
	TypeArray   = "array"
	TypeObject  = "object"
)


var ErrSchemaInvalid = errors.New("schema is invalid")

type Discriminator struct {
	PropertyName string            `yaml:"propertyName"`
	Mapping      map[string]string `yaml:"mapping,omitempty"`
}

func (d *Discriminator) MarshalJSON() ([]byte, error) {
	return MarshalJSON([]JSONFieldInfo{
		{"propertyName", d.PropertyName, OmitNever},
		{"mapping", d.Mapping, OmitEmpty},
	}, nil)
}

type Schema struct {
	Type                 string              `yaml:"type,omitempty"`
	Nullable             bool                `yaml:"-"`
	Title                string              `yaml:"title,omitempty"`
	Description          string              `yaml:"description,omitempty"`
	Ref                  string              `yaml:"$ref,omitempty"`
	Format               string              `yaml:"format,omitempty"`
	ContentEncoding      string              `yaml:"contentEncoding,omitempty"`
	Default              any                 `yaml:"default,omitempty"`
	Examples             []any               `yaml:"examples,omitempty"`
	Items                *Schema             `yaml:"items,omitempty"`
	AdditionalProperties any                 `yaml:"additionalProperties,omitempty"`
	Properties           map[string]*Schema  `yaml:"properties,omitempty"`
	Enum                 []any               `yaml:"enum,omitempty"`
	Minimum              *float64            `yaml:"minimum,omitempty"`
	ExclusiveMinimum     *float64            `yaml:"exclusiveMinimum,omitempty"`
	Maximum              *float64            `yaml:"maximum,omitempty"`
	ExclusiveMaximum     *float64            `yaml:"exclusiveMaximum,omitempty"`
	MultipleOf           *float64            `yaml:"multipleOf,omitempty"`
	MinLength            *int                `yaml:"minLength,omitempty"`
	MaxLength            *int                `yaml:"maxLength,omitempty"`
	Pattern              string              `yaml:"pattern,omitempty"`
	PatternDescription   string              `yaml:"patternDescription,omitempty"`
	MinItems             *int                `yaml:"minItems,omitempty"`
	MaxItems             *int                `yaml:"maxItems,omitempty"`
	UniqueItems          bool                `yaml:"uniqueItems,omitempty"`
	Required             []string            `yaml:"required,omitempty"`
	MinProperties        *int                `yaml:"minProperties,omitempty"`
	MaxProperties        *int                `yaml:"maxProperties,omitempty"`
	ReadOnly             bool                `yaml:"readOnly,omitempty"`
	WriteOnly            bool                `yaml:"writeOnly,omitempty"`
	Deprecated           bool                `yaml:"deprecated,omitempty"`
	Extensions           map[string]any      `yaml:",inline"`
	DependentRequired    map[string][]string `yaml:"dependentRequired,omitempty"`

	OneOf []*Schema `yaml:"oneOf,omitempty"`
	AnyOf []*Schema `yaml:"anyOf,omitempty"`
	AllOf []*Schema `yaml:"allOf,omitempty"`
	Not   *Schema   `yaml:"not,omitempty"`

	Discriminator *Discriminator `yaml:"discriminator,omitempty"`

	PatternRe     *regexp.Regexp  `yaml:"-"`
	RequiredMap   map[string]bool `yaml:"-"`
	PropertyNames []string        `yaml:"-"`
	Hidden        bool            `yaml:"-"`

	MsgEnum              string                       `yaml:"-"`
	MsgMinimum           string                       `yaml:"-"`
	MsgExclusiveMinimum  string                       `yaml:"-"`
	MsgMaximum           string                       `yaml:"-"`
	MsgExclusiveMaximum  string                       `yaml:"-"`
	MsgMultipleOf        string                       `yaml:"-"`
	MsgMinLength         string                       `yaml:"-"`
	MsgMaxLength         string                       `yaml:"-"`
	MsgPattern           string                       `yaml:"-"`
	MsgMinItems          string                       `yaml:"-"`
	MsgMaxItems          string                       `yaml:"-"`
	MsgMinProperties     string                       `yaml:"-"`
	MsgMaxProperties     string                       `yaml:"-"`
	MsgRequired          map[string]string            `yaml:"-"`
	MsgDependentRequired map[string]map[string]string `yaml:"-"`
}

func (s *Schema) MarshalJSON() ([]byte, error) {
	var typ any = s.Type
	if s.Nullable {
		typ = []string{s.Type, "null"}
	}

	var contentMediaType string
	if s.Format == "binary" {
		contentMediaType = "application/octet-stream"
	}

	props := s.Properties
	for _, ps := range props {
		if ps.Hidden {
			props = make(map[string]*Schema, len(s.Properties))
			for k, v := range s.Properties {
				if !v.Hidden {
					props[k] = v
				}
			}
			break
		}
	}

	return MarshalJSON([]JSONFieldInfo{
		{"type", typ, OmitEmpty},
		{"title", s.Title, OmitEmpty},
		{"description", s.Description, OmitEmpty},
		{"$ref", s.Ref, OmitEmpty},
		{"format", s.Format, OmitEmpty},
		{"contentMediaType", contentMediaType, OmitEmpty},
		{"contentEncoding", s.ContentEncoding, OmitEmpty},
		{"default", s.Default, OmitNil},
		{"examples", s.Examples, OmitEmpty},
		{"items", s.Items, OmitEmpty},
		{"additionalProperties", s.AdditionalProperties, OmitNil},
		{"properties", props, OmitEmpty},
		{"enum", s.Enum, OmitEmpty},
		{"minimum", s.Minimum, OmitEmpty},
		{"exclusiveMinimum", s.ExclusiveMinimum, OmitEmpty},
		{"maximum", s.Maximum, OmitEmpty},
		{"exclusiveMaximum", s.ExclusiveMaximum, OmitEmpty},
		{"multipleOf", s.MultipleOf, OmitEmpty},
		{"minLength", s.MinLength, OmitEmpty},
		{"maxLength", s.MaxLength, OmitEmpty},
		{"pattern", s.Pattern, OmitEmpty},
		{"patternDescription", s.PatternDescription, OmitEmpty},
		{"minItems", s.MinItems, OmitEmpty},
		{"maxItems", s.MaxItems, OmitEmpty},
		{"uniqueItems", s.UniqueItems, OmitEmpty},
		{"required", s.Required, OmitEmpty},
		{"dependentRequired", s.DependentRequired, OmitEmpty},
		{"minProperties", s.MinProperties, OmitEmpty},
		{"maxProperties", s.MaxProperties, OmitEmpty},
		{"readOnly", s.ReadOnly, OmitEmpty},
		{"writeOnly", s.WriteOnly, OmitEmpty},
		{"deprecated", s.Deprecated, OmitEmpty},
		{"oneOf", s.OneOf, OmitEmpty},
		{"anyOf", s.AnyOf, OmitEmpty},
		{"allOf", s.AllOf, OmitEmpty},
		{"not", s.Not, OmitEmpty},
		{"discriminator", s.Discriminator, OmitEmpty},
	}, s.Extensions)
}

func (s *Schema) PrecomputeMessages() {
	s.MsgEnum = fmt.Sprintf(MsgExpectedOneOf,
		strings.Join(mapTo(s.Enum, func(v any) string { return fmt.Sprintf("%v", v) }), ", "))

	if s.Minimum != nil {
		s.MsgMinimum = fmt.Sprintf(MsgExpectedMinimumNumber, *s.Minimum)
	}
	if s.ExclusiveMinimum != nil {
		s.MsgExclusiveMinimum = fmt.Sprintf(MsgExpectedExclusiveMinimumNumber, *s.ExclusiveMinimum)
	}
	if s.Maximum != nil {
		s.MsgMaximum = fmt.Sprintf(MsgExpectedMaximumNumber, *s.Maximum)
	}
	if s.ExclusiveMaximum != nil {
		s.MsgExclusiveMaximum = fmt.Sprintf(MsgExpectedExclusiveMaximumNumber, *s.ExclusiveMaximum)
	}
	if s.MultipleOf != nil {
		s.MsgMultipleOf = fmt.Sprintf(MsgExpectedNumberBeMultipleOf, *s.MultipleOf)
	}
	if s.MinLength != nil {
		s.MsgMinLength = fmt.Sprintf(MsgExpectedMinLength, *s.MinLength)
	}
	if s.MaxLength != nil {
		s.MsgMaxLength = fmt.Sprintf(MsgExpectedMaxLength, *s.MaxLength)
	}
	if s.Pattern != "" {
		s.PatternRe = regexp.MustCompile(s.Pattern)
		if s.PatternDescription != "" {
			s.MsgPattern = fmt.Sprintf(MsgExpectedBePattern, s.PatternDescription)
		} else {
			s.MsgPattern = fmt.Sprintf(MsgExpectedMatchPattern, s.Pattern)
		}
	}
	if s.MinItems != nil {
		s.MsgMinItems = fmt.Sprintf(MsgExpectedMinItems, *s.MinItems)
	}
	if s.MaxItems != nil {
		s.MsgMaxItems = fmt.Sprintf(MsgExpectedMaxItems, *s.MaxItems)
	}
	if s.MinProperties != nil {
		s.MsgMinProperties = fmt.Sprintf(MsgExpectedMinProperties, *s.MinProperties)
	}
	if s.MaxProperties != nil {
		s.MsgMaxProperties = fmt.Sprintf(MsgExpectedMaxProperties, *s.MaxProperties)
	}

	if s.Required != nil {
		if s.MsgRequired == nil {
			s.MsgRequired = map[string]string{}
		}
		for _, name := range s.Required {
			s.MsgRequired[name] = fmt.Sprintf(MsgExpectedRequiredProperty, name)
		}
	}

	if s.DependentRequired != nil {
		if s.MsgDependentRequired == nil {
			s.MsgDependentRequired = map[string]map[string]string{}
		}
		for name, dependents := range s.DependentRequired {
			for _, dependent := range dependents {
				if s.MsgDependentRequired[name] == nil {
					s.MsgDependentRequired[name] = map[string]string{}
				}
				s.MsgDependentRequired[name][dependent] = fmt.Sprintf(
					MsgExpectedDependentRequiredProperty, dependent, name)
			}
		}
	}

	s.PropertyNames = make([]string, 0, len(s.Properties))
	for name := range s.Properties {
		s.PropertyNames = append(s.PropertyNames, name)
	}
	sort.Strings(s.PropertyNames)

	s.RequiredMap = map[string]bool{}
	for _, name := range s.Required {
		s.RequiredMap[name] = true
	}

	if s.Items != nil {
		s.Items.PrecomputeMessages()
	}
	for _, prop := range s.Properties {
		prop.PrecomputeMessages()
	}
	for _, sub := range s.OneOf {
		sub.PrecomputeMessages()
	}
	for _, sub := range s.AnyOf {
		sub.PrecomputeMessages()
	}
	for _, sub := range s.AllOf {
		sub.PrecomputeMessages()
	}
	if sub := s.Not; sub != nil {
		sub.PrecomputeMessages()
	}
}

type OmitType int

const (
	OmitNever OmitType = iota
	OmitEmpty
	OmitNil
)

type JSONFieldInfo struct {
	Name  string
	Value any
	Omit  OmitType
}

type SchemaProvider interface {
	Schema(r Registry) *Schema
}

type SchemaTransformer interface {
	TransformSchema(r Registry, s *Schema) *Schema
}

type Registry interface {
	Schema(t reflect.Type, allowRef bool, hint string) *Schema
	SchemaFromRef(ref string) *Schema
	TypeFromRef(ref string) reflect.Type
	Map() map[string]*Schema
	RegisterTypeAlias(t reflect.Type, alias reflect.Type)
}

func IsEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Pointer:
		return v.IsNil()
	default:
		return false
	}
}

func IsNilValue(v any) bool {
	if v == nil {
		return true
	}
	vv := reflect.ValueOf(v)
	switch vv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return vv.IsNil()
	default:
		return false
	}
}

func MarshalJSON(fields []JSONFieldInfo, extensions map[string]any) ([]byte, error) {
	value := make(map[string]any, len(extensions)+len(fields))

	for _, v := range fields {
		if v.Omit == OmitNil && IsNilValue(v.Value) {
			continue
		}
		if v.Omit == OmitEmpty {
			if IsEmptyValue(reflect.ValueOf(v.Value)) {
				continue
			}
		}
		value[v.Name] = v.Value
	}

	maps.Copy(value, extensions)

	return json.Marshal(value)
}


func mapTo[T any, U any](s []T, f func(T) U) []U {
	result := make([]U, len(s))
	for i, v := range s {
		result[i] = f(v)
	}
	return result
}
