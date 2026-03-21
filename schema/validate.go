package schema

import (
	"fmt"
	"math"
	"net/mail"
	"net/netip"
	"net/url"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
	"unsafe"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/validation"
)

// ValidateStrictCasing controls whether property name matching during
// validation is case-sensitive. When false (the default), property names
// are matched case-insensitively.
var ValidateStrictCasing = false

var rxHostname = regexp.MustCompile(`^([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])(\.([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9]))*$`)
var rxURITemplate = regexp.MustCompile("^([^{]*({[^}]*})?)*$")
var rxJSONPointer = regexp.MustCompile("^(?:/(?:[^~/]|~0|~1)*)*$")
var rxRelJSONPointer = regexp.MustCompile("^(?:0|[1-9][0-9]*)(?:#|(?:/(?:[^~/]|~0|~1)*)*)$")
var rxBase64 = regexp.MustCompile(`^[a-zA-Z0-9+/_-]+=*$`)

// ModelValidator validates Go values against their auto-generated JSON
// Schemas. It maintains a reusable registry, path buffer, and result to
// reduce allocations across repeated validations.
type ModelValidator struct {
	registry core.Registry
	pb       *core.PathBuffer
	result   *core.ValidateResult
}

// Validate validates a value against the schema for the given type, returning
// any validation errors.
func (v *ModelValidator) Validate(typ reflect.Type, value any) []error {
	v.pb.Reset()
	v.result.Reset()

	s := v.registry.Schema(typ, true, typ.Name())
	if s == nil {
		return nil
	}

	Validate(v.registry, s, v.pb, core.ModeReadFromServer, value, v.result)

	if len(v.result.Errors) > 0 {
		return v.result.Errors
	}
	return nil
}

// NewModelValidator returns a new ModelValidator with a default registry.
func NewModelValidator() *ModelValidator {
	return &ModelValidator{
		registry: NewMapRegistry(DefaultSchemaNamer),
		pb:       core.NewPathBuffer([]byte(""), 0),
		result:   &core.ValidateResult{},
	}
}

// Validate checks a value against a schema, appending any violations to res.
// It resolves $ref references, enforces type constraints, format validation,
// and composition keywords (oneOf, anyOf, allOf, not).
func Validate(r core.Registry, s *core.Schema, path *core.PathBuffer, mode core.ValidateMode, v any, res *core.ValidateResult) {
	if s == nil {
		return
	}
	for s.Ref != "" {
		s = r.SchemaFromRef(s.Ref)
		if s == nil {
			return
		}
	}

	if s.OneOf != nil {
		if s.Discriminator != nil {
			validateDiscriminator(r, s, path, mode, v, res)
		} else {
			validateOneOf(r, s, path, mode, v, res)
		}
	}

	if s.AnyOf != nil {
		validateAnyOf(r, s, path, mode, v, res)
	}

	if s.AllOf != nil {
		for _, sub := range s.AllOf {
			Validate(r, sub, path, mode, v, res)
		}
	}

	if s.Not != nil {
		subRes := &core.ValidateResult{}
		Validate(r, s.Not, path, mode, v, subRes)
		if len(subRes.Errors) == 0 {
			res.Add(path, v, core.MsgExpectedNotMatchSchema)
		}
	}

	if s.Nullable && v == nil {
		return
	}

	switch s.Type {
	case core.TypeBoolean:
		if _, ok := v.(bool); !ok {
			res.Add(path, v, core.MsgExpectedBoolean)
			return
		}
	case core.TypeNumber, core.TypeInteger:
		var num float64

		switch v := v.(type) {
		case float64:
			num = v
		case float32:
			num = float64(v)
		case int:
			num = float64(v)
		case int8:
			num = float64(v)
		case int16:
			num = float64(v)
		case int32:
			num = float64(v)
		case int64:
			num = float64(v)
		case uint:
			num = float64(v)
		case uint8:
			num = float64(v)
		case uint16:
			num = float64(v)
		case uint32:
			num = float64(v)
		case uint64:
			num = float64(v)
		default:
			if s.Type == core.TypeInteger {
				res.Add(path, v, core.MsgExpectedInteger)
			} else {
				res.Add(path, v, core.MsgExpectedNumber)
			}
			return
		}

		if s.Type == core.TypeInteger && num != math.Trunc(num) {
			res.Add(path, v, core.MsgExpectedInteger)
		}

		if s.Minimum != nil {
			if num < *s.Minimum {
				res.Add(path, v, s.MsgMinimum)
			}
		}
		if s.ExclusiveMinimum != nil {
			if num <= *s.ExclusiveMinimum {
				res.Add(path, v, s.MsgExclusiveMinimum)
			}
		}
		if s.Maximum != nil {
			if num > *s.Maximum {
				res.Add(path, v, s.MsgMaximum)
			}
		}
		if s.ExclusiveMaximum != nil {
			if num >= *s.ExclusiveMaximum {
				res.Add(path, v, s.MsgExclusiveMaximum)
			}
		}
		if s.MultipleOf != nil {
			if remainder := math.Mod(num, *s.MultipleOf); math.Abs(remainder) > 1e-9 && math.Abs(remainder-*s.MultipleOf) > 1e-9 {
				res.Add(path, v, s.MsgMultipleOf)
			}
		}
	case core.TypeString:
		str, ok := v.(string)
		if !ok {
			if b, ok := v.([]byte); ok {
				str = *(*string)(unsafe.Pointer(&b))
			} else {
				res.Add(path, v, core.MsgExpectedString)
				return
			}
		}

		if s.MinLength != nil {
			if utf8.RuneCountInString(str) < *s.MinLength {
				res.Add(path, str, s.MsgMinLength)
			}
		}

		if s.MaxLength != nil {
			if utf8.RuneCountInString(str) > *s.MaxLength {
				res.Add(path, str, s.MsgMaxLength)
			}
		}

		if s.PatternRe != nil {
			if !s.PatternRe.MatchString(str) {
				res.Add(path, v, s.MsgPattern)
			}
		}

		if s.Format != "" {
			validateFormat(path, str, s, res)
		}

		if s.ContentEncoding == "base64" {
			if !rxBase64.MatchString(str) {
				res.Add(path, str, core.MsgExpectedBase64String)
			}
		}
	case core.TypeArray:
		switch arr := v.(type) {
		case []any:
			handleArray(r, s, path, mode, res, arr)
		case []string:
			handleArray(r, s, path, mode, res, arr)
		case []int:
			handleArray(r, s, path, mode, res, arr)
		case []int8:
			handleArray(r, s, path, mode, res, arr)
		case []int16:
			handleArray(r, s, path, mode, res, arr)
		case []int32:
			handleArray(r, s, path, mode, res, arr)
		case []int64:
			handleArray(r, s, path, mode, res, arr)
		case []uint:
			handleArray(r, s, path, mode, res, arr)
		case []uint16:
			handleArray(r, s, path, mode, res, arr)
		case []uint32:
			handleArray(r, s, path, mode, res, arr)
		case []uint64:
			handleArray(r, s, path, mode, res, arr)
		case []float32:
			handleArray(r, s, path, mode, res, arr)
		case []float64:
			handleArray(r, s, path, mode, res, arr)
		default:
			res.Add(path, v, core.MsgExpectedArray)
			return
		}
	case core.TypeObject:
		switch vv := v.(type) {
		case map[string]any:
			handleMapString(r, s, path, mode, vv, res)
		case map[any]any:
			handleMapAny(r, s, path, mode, vv, res)
		default:
			res.Add(path, v, core.MsgExpectedObject)
			return
		}
	}

	if len(s.Enum) > 0 {
		found := slices.Contains(s.Enum, v)
		if !found {
			res.Add(path, v, s.MsgEnum)
		}
	}
}

func handleArray[T any](r core.Registry, s *core.Schema, path *core.PathBuffer, mode core.ValidateMode, res *core.ValidateResult, arr []T) {
	if s.MinItems != nil {
		if len(arr) < *s.MinItems {
			res.Add(path, arr, s.MsgMinItems)
		}
	}
	if s.MaxItems != nil {
		if len(arr) > *s.MaxItems {
			res.Add(path, arr, s.MsgMaxItems)
		}
	}

	if s.UniqueItems {
		seen := make(map[any]struct{}, len(arr))
		for _, item := range arr {
			if _, ok := seen[item]; ok {
				res.Add(path, arr, core.MsgExpectedArrayItemsUnique)
			}
			seen[item] = struct{}{}
		}
	}

	for i, item := range arr {
		path.PushIndex(i)
		Validate(r, s.Items, path, mode, item, res)
		path.Pop()
	}
}

func handleMapAny(r core.Registry, s *core.Schema, path *core.PathBuffer, mode core.ValidateMode, m map[any]any, res *core.ValidateResult) {
	if s.MinProperties != nil {
		if len(m) < *s.MinProperties {
			res.Add(path, m, s.MsgMinProperties)
		}
	}
	if s.MaxProperties != nil {
		if len(m) > *s.MaxProperties {
			res.Add(path, m, s.MsgMaxProperties)
		}
	}

	for _, k := range s.PropertyNames {
		v := s.Properties[k]
		if v == nil {
			continue
		}

		readOnly := v.ReadOnly
		writeOnly := v.WriteOnly
		for v.Ref != "" {
			v = r.SchemaFromRef(v.Ref)
			if v == nil {
				break
			}
		}
		if v == nil {
			continue
		}

		if mode == core.ModeReadFromServer && writeOnly && m[k] != nil && !reflect.ValueOf(m[k]).IsZero() {
			res.Add(path, m[k], "write only property is non-zero")
			continue
		}

		if _, ok := m[k]; !ok {
			if !s.RequiredMap[k] {
				continue
			}
			if (mode == core.ModeWriteToServer && readOnly) ||
				(mode == core.ModeReadFromServer && writeOnly) {
				continue
			}
			res.Add(path, m, s.MsgRequired[k])
			continue
		}

		if m[k] == nil && (!s.RequiredMap[k] || s.Nullable) {
			continue
		}

		if m[k] != nil && s.DependentRequired[k] != nil {
			for _, dependent := range s.DependentRequired[k] {
				if m[dependent] != nil {
					continue
				}

				res.Add(path, m, s.MsgDependentRequired[k][dependent])
			}
		}

		path.Push(k)
		Validate(r, v, path, mode, m[k], res)
		path.Pop()
	}

	if addl, ok := s.AdditionalProperties.(bool); ok && !addl {
		for k := range m {
			var kStr string
			if s, ok := k.(string); ok {
				kStr = s
			} else {
				kStr = fmt.Sprint(k)
			}
			if _, ok := s.Properties[kStr]; !ok {
				path.Push(kStr)
				res.Add(path, m, core.MsgUnexpectedProperty)
				path.Pop()
			}
		}
	}

	if addl, ok := s.AdditionalProperties.(*core.Schema); ok {
		for k, v := range m {
			var kStr string
			if s, ok := k.(string); ok {
				kStr = s
			} else {
				kStr = fmt.Sprint(k)
			}
			path.Push(kStr)
			Validate(r, addl, path, mode, v, res)
			path.Pop()
		}
	}
}

func handleMapString(r core.Registry, s *core.Schema, path *core.PathBuffer, mode core.ValidateMode, m map[string]any, res *core.ValidateResult) {
	if s.MinProperties != nil {
		if len(m) < *s.MinProperties {
			res.Add(path, m, s.MsgMinProperties)
		}
	}
	if s.MaxProperties != nil {
		if len(m) > *s.MaxProperties {
			res.Add(path, m, s.MsgMaxProperties)
		}
	}

	for _, k := range s.PropertyNames {
		v := s.Properties[k]
		if v == nil {
			continue
		}

		readOnly := v.ReadOnly
		writeOnly := v.WriteOnly
		for v.Ref != "" {
			v = r.SchemaFromRef(v.Ref)
			if v == nil {
				break
			}
		}
		if v == nil {
			continue
		}

		if mode == core.ModeReadFromServer && writeOnly && m[k] != nil && !reflect.ValueOf(m[k]).IsZero() {
			res.Add(path, m[k], "write only property is non-zero")
			continue
		}

		actualKey := k
		_, ok := m[k]
		if !ok && !ValidateStrictCasing {
			for actual := range m {
				if strings.EqualFold(actual, k) {
					// Case-insensitive match found, so this is not an error.
					actualKey = actual
					ok = true
					break
				}
			}
		}

		if !ok {
			if !s.RequiredMap[k] {
				continue
			}
			if (mode == core.ModeWriteToServer && readOnly) ||
				(mode == core.ModeReadFromServer && writeOnly) {
				continue
			}
			res.Add(path, m, s.MsgRequired[k])
			continue
		}

		if m[actualKey] == nil && (!s.RequiredMap[k] || s.Nullable) {
			continue
		}

		if m[actualKey] != nil && s.DependentRequired[k] != nil {
			for _, dependent := range s.DependentRequired[k] {
				if m[dependent] != nil {
					continue
				}

				res.Add(path, m, s.MsgDependentRequired[k][dependent])
			}
		}

		path.Push(k)
		Validate(r, v, path, mode, m[actualKey], res)
		path.Pop()
	}

	if addl, ok := s.AdditionalProperties.(bool); ok && !addl {
	addlPropLoop:
		for k := range m {
			if _, ok := s.Properties[k]; !ok {
				if !ValidateStrictCasing {
					for propName := range s.Properties {
						if strings.EqualFold(propName, k) {
							// Case-insensitive match found, so this is not an error.
							continue addlPropLoop
						}
					}
				}

				path.Push(k)
				res.Add(path, m, core.MsgUnexpectedProperty)
				path.Pop()
			}
		}
	}

	if addl, ok := s.AdditionalProperties.(*core.Schema); ok {
		for k, v := range m {
			if _, ok := s.Properties[k]; ok {
				continue
			}

			path.Push(k)
			Validate(r, addl, path, mode, v, res)
			path.Pop()
		}
	}
}

func validateAnyOf(r core.Registry, s *core.Schema, path *core.PathBuffer, mode core.ValidateMode, v any, res *core.ValidateResult) {
	matches := 0
	subRes := &core.ValidateResult{}
	for _, sub := range s.AnyOf {
		Validate(r, sub, path, mode, v, subRes)
		if len(subRes.Errors) == 0 {
			matches++
		}
		subRes.Reset()
	}

	if matches == 0 {
		res.Add(path, v, core.MsgExpectedMatchAtLeastOneSchema)
	}
}

func validateDiscriminator(r core.Registry, s *core.Schema, path *core.PathBuffer, mode core.ValidateMode, v any, res *core.ValidateResult) {
	var kk any
	found := true

	if vv, ok := v.(map[string]any); ok {
		kk, found = vv[s.Discriminator.PropertyName]
	}

	if vv, ok := v.(map[any]any); ok {
		kk, found = vv[s.Discriminator.PropertyName]
	}

	if !found {
		path.Push(s.Discriminator.PropertyName)
		res.Add(path, v, core.MsgExpectedPropertyNameInObject)
		return
	}

	if kk == nil {
		// Either v is not a map or the property is set to null. Return so that
		// type and enum checks on the field can complete elsewhere.
		return
	}

	key, ok := kk.(string)
	if !ok {
		path.Push(s.Discriminator.PropertyName)
		return
	}

	ref, found := s.Discriminator.Mapping[key]
	if !found {
		validateOneOf(r, s, path, mode, v, res)
		return
	}

	Validate(r, r.SchemaFromRef(ref), path, mode, v, res)
}

func validateFormat(path *core.PathBuffer, str string, s *core.Schema, res *core.ValidateResult) {
	switch s.Format {
	case "date-time":
		found := false
		for _, format := range []string{time.RFC3339, time.RFC3339Nano} {
			if _, err := time.Parse(format, str); err == nil {
				found = true
				break
			}
		}
		if !found {
			res.Add(path, str, core.MsgExpectedRFC3339DateTime)
		}
	case "date-time-http":
		if _, err := time.Parse(time.RFC1123, str); err != nil {
			res.Add(path, str, core.MsgExpectedRFC1123DateTime)
		}
	case "date":
		if _, err := time.Parse(time.DateOnly, str); err != nil {
			res.Add(path, str, core.MsgExpectedRFC3339Date)
		}
	case "duration":
		if _, err := time.ParseDuration(str); err != nil {
			res.Add(path, str, fmt.Sprintf(core.MsgExpectedDuration, err))
		}
	case "time":
		found := false
		for _, format := range []string{time.TimeOnly, "15:04:05Z07:00"} {
			if _, err := time.Parse(format, str); err == nil {
				found = true
				break
			}
		}
		if !found {
			res.Add(path, str, core.MsgExpectedRFC3339Time)
		}
	case "email", "idn-email":
		if _, err := mail.ParseAddress(str); err != nil {
			res.Add(path, str, fmt.Sprintf(core.MsgExpectedRFC5322Email, err))
		}
	case "idn-hostname", "hostname":
		if len(str) >= 256 || !rxHostname.MatchString(str) {
			res.Add(path, str, core.MsgExpectedRFC5890Hostname)
		}
	case "ipv4", "ipv6", "ip":
		addr, err := netip.ParseAddr(str)

		switch s.Format {
		case "ipv4":
			if err != nil || !addr.Is4() {
				res.Add(path, str, core.MsgExpectedRFC2673IPv4)
			}
		case "ipv6":
			if err != nil || !addr.Is6() || addr.Is4In6() {
				res.Add(path, str, core.MsgExpectedRFC2373IPv6)
			}
		default: // case "ip".
			if err != nil {
				res.Add(path, str, core.MsgExpectedRFCIPAddr)
			}
		}
	case "uri", "uri-reference", "iri", "iri-reference":
		if _, err := url.Parse(str); err != nil {
			res.Add(path, str, fmt.Sprintf(core.MsgExpectedRFC3986URI, err))
		}
	case "uri-template":
		u, err := url.Parse(str)
		if err != nil {
			res.Add(path, str, fmt.Sprintf(core.MsgExpectedRFC3986URI, err))
			return
		}
		if !rxURITemplate.MatchString(u.Path) {
			res.Add(path, str, core.MsgExpectedRFC6570URITemplate)
		}
	case "uuid":
		if err := validation.ValidateUUID(str); err != nil {
			res.Add(path, str, fmt.Sprintf(core.MsgExpectedRFC4122UUID, err))
		}
	case "json-pointer":
		if !rxJSONPointer.MatchString(str) {
			res.Add(path, str, core.MsgExpectedRFC6901JSONPointer)
		}
	case "relative-json-pointer":
		if !rxRelJSONPointer.MatchString(str) {
			res.Add(path, str, core.MsgExpectedRFC6901RelativeJSONPointer)
		}
	case "regex":
		if _, err := regexp.Compile(str); err != nil {
			res.Add(path, str, fmt.Sprintf(core.MsgExpectedRegexp, err))
		}
	}
}

func validateOneOf(r core.Registry, s *core.Schema, path *core.PathBuffer, mode core.ValidateMode, v any, res *core.ValidateResult) {
	found := false
	subRes := &core.ValidateResult{}
	for _, sub := range s.OneOf {
		Validate(r, sub, path, mode, v, subRes)
		if len(subRes.Errors) == 0 {
			if found {
				res.Add(path, v, "expected value to match exactly one schema but matched multiple")
			}
			found = true
		}
		subRes.Reset()
	}
	if !found {
		res.Add(path, v, core.MsgExpectedMatchExactlyOneSchema)
	}
}
