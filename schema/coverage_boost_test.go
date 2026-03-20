package schema_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateFormatDateTimeNano(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString, Format: "date-time"}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)
	res := &core.ValidateResult{}

	schema.Validate(r, s, pb, core.ModeWriteToServer, "2024-01-15T10:30:00.123456789Z", res)
	assert.Empty(t, res.Errors, "RFC3339Nano should be valid")
}

func TestValidateFormatDateTimeHTTP(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString, Format: "date-time-http"}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, "Mon, 15 Jan 2024 10:30:00 GMT", res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, "not-http-date", res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateFormatTimeWithTimezone(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString, Format: "time"}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)
	res := &core.ValidateResult{}

	schema.Validate(r, s, pb, core.ModeWriteToServer, "10:30:00+01:00", res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, "bad-time", res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateFormatIDNEmail(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString, Format: "idn-email"}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)
	res := &core.ValidateResult{}

	schema.Validate(r, s, pb, core.ModeWriteToServer, "user@example.com", res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, "not-an-email", res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateFormatIDNHostname(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString, Format: "idn-hostname"}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)
	res := &core.ValidateResult{}

	schema.Validate(r, s, pb, core.ModeWriteToServer, "example.com", res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, strings.Repeat("a", 256), res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateFormatHostnameInvalid(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString, Format: "hostname"}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)
	res := &core.ValidateResult{}

	schema.Validate(r, s, pb, core.ModeWriteToServer, "invalid hostname!", res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateFormatIPGeneric(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString, Format: "ip"}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, "192.168.1.1", res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, "::1", res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, "not-an-ip", res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateFormatIPv4WithIPv6(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString, Format: "ipv4"}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)
	res := &core.ValidateResult{}

	schema.Validate(r, s, pb, core.ModeWriteToServer, "::1", res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateFormatIPv6Invalid(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString, Format: "ipv6"}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)
	res := &core.ValidateResult{}

	schema.Validate(r, s, pb, core.ModeWriteToServer, "not-ipv6", res)
	assert.NotEmpty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, "192.168.1.1", res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateFormatIPv6Is4In6(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString, Format: "ipv6"}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)
	res := &core.ValidateResult{}

	schema.Validate(r, s, pb, core.ModeWriteToServer, "::ffff:192.168.1.1", res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateFormatURIReference(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString, Format: "uri-reference"}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)
	res := &core.ValidateResult{}

	schema.Validate(r, s, pb, core.ModeWriteToServer, "/relative/path", res)
	assert.Empty(t, res.Errors)
}

func TestValidateFormatIRI(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString, Format: "iri"}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)
	res := &core.ValidateResult{}

	schema.Validate(r, s, pb, core.ModeWriteToServer, "https://example.com/path", res)
	assert.Empty(t, res.Errors)
}

func TestValidateFormatIRIReference(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString, Format: "iri-reference"}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)
	res := &core.ValidateResult{}

	schema.Validate(r, s, pb, core.ModeWriteToServer, "/path", res)
	assert.Empty(t, res.Errors)
}

func TestValidateFormatURITemplate(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString, Format: "uri-template"}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, "https://example.com/{id}", res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, "/path/{unclosed", res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateFormatRelativeJSONPointer(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString, Format: "relative-json-pointer"}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, "0/foo/bar", res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, "not a pointer", res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateFormatJSONPointerInvalid(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString, Format: "json-pointer"}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)
	res := &core.ValidateResult{}

	schema.Validate(r, s, pb, core.ModeWriteToServer, "not a json pointer", res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateFormatDurationInvalid(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString, Format: "duration"}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)
	res := &core.ValidateResult{}

	schema.Validate(r, s, pb, core.ModeWriteToServer, "5h30m", res)
	assert.Empty(t, res.Errors)
}

func TestValidateUUIDFormats(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString, Format: "uuid"}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{"standard", "550e8400-e29b-41d4-a716-446655440000", true},
		{"urn prefix", "urn:uuid:550e8400-e29b-41d4-a716-446655440000", true},
		{"braces", "{550e8400-e29b-41d4-a716-446655440000}", true},
		{"no hyphens", "550e8400e29b41d4a716446655440000", true},
		{"invalid urn prefix", "xxx:uuid:550e8400-e29b-41d4-a716-446655440000", false},
		{"invalid braces", "(550e8400-e29b-41d4-a716-446655440000)", false},
		{"wrong length", "550e8400-e29b-41d4", false},
		{"invalid hex in no-hyphen", "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz", false},
		{"bad dash positions", "550e8400xe29bx41d4xa716x446655440000", false},
		{"invalid hex in standard", "zzzzzzzz-zzzz-zzzz-zzzz-zzzzzzzzzzzz", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := &core.ValidateResult{}
			pb.Reset()
			schema.Validate(r, s, pb, core.ModeWriteToServer, tt.value, res)
			if tt.valid {
				assert.Empty(t, res.Errors)
			} else {
				assert.NotEmpty(t, res.Errors)
			}
		})
	}
}

func TestValidateBooleanTypeError(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeBoolean}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)
	res := &core.ValidateResult{}

	schema.Validate(r, s, pb, core.ModeWriteToServer, "not-a-bool", res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateNumberTypeError(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeNumber}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)
	res := &core.ValidateResult{}

	schema.Validate(r, s, pb, core.ModeWriteToServer, "not-a-number", res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateIntegerTypeError(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeInteger}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)
	res := &core.ValidateResult{}

	schema.Validate(r, s, pb, core.ModeWriteToServer, "hello", res)
	assert.NotEmpty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, 1.5, res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateIntegerWithIntTypes(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeInteger}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	intValues := []any{
		int(5), int8(5), int16(5), int32(5), int64(5),
		uint(5), uint8(5), uint16(5), uint32(5), uint64(5),
		float32(5), float64(5),
	}
	for _, v := range intValues {
		res := &core.ValidateResult{}
		pb.Reset()
		schema.Validate(r, s, pb, core.ModeWriteToServer, v, res)
		assert.Empty(t, res.Errors, "expected %T(%v) to pass integer validation", v, v)
	}
}

func TestValidateNumberWithMinMaxExclusive(t *testing.T) {
	r := newRegistry()
	eMin := 0.0
	eMax := 10.0
	s := &core.Schema{
		Type:             core.TypeNumber,
		ExclusiveMinimum: &eMin,
		ExclusiveMaximum: &eMax,
	}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	// ExclusiveMinimum: 0 is not strictly greater than 0
	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, float64(0), res)
	assert.NotEmpty(t, res.Errors)

	// ExclusiveMaximum: 10 is not strictly less than 10
	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, float64(10), res)
	assert.NotEmpty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, float64(5), res)
	assert.Empty(t, res.Errors)
}

func TestValidateStringTypeError(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)
	res := &core.ValidateResult{}

	schema.Validate(r, s, pb, core.ModeWriteToServer, 42, res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateStringBytesInput(t *testing.T) {
	r := newRegistry()
	minLen := 3
	s := &core.Schema{Type: core.TypeString, MinLength: &minLen}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)
	res := &core.ValidateResult{}

	schema.Validate(r, s, pb, core.ModeWriteToServer, []byte("hello"), res)
	assert.Empty(t, res.Errors)
}

func TestValidateStringMaxLength(t *testing.T) {
	r := newRegistry()
	maxLen := 5
	s := &core.Schema{Type: core.TypeString, MaxLength: &maxLen}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)
	res := &core.ValidateResult{}

	schema.Validate(r, s, pb, core.ModeWriteToServer, "toolong", res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateStringPattern(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString, Pattern: "^[A-Z]+$"}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)
	res := &core.ValidateResult{}

	schema.Validate(r, s, pb, core.ModeWriteToServer, "lowercase", res)
	assert.NotEmpty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, "UPPERCASE", res)
	assert.Empty(t, res.Errors)
}

func TestValidateArrayTypeError(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeArray, Items: &core.Schema{Type: core.TypeString}}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)
	res := &core.ValidateResult{}

	schema.Validate(r, s, pb, core.ModeWriteToServer, "not-an-array", res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateArrayMinMaxItems(t *testing.T) {
	r := newRegistry()
	minItems := 2
	maxItems := 4
	s := &core.Schema{
		Type:     core.TypeArray,
		Items:    &core.Schema{Type: core.TypeString},
		MinItems: &minItems,
		MaxItems: &maxItems,
	}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, []any{"a"}, res)
	assert.NotEmpty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, []any{"a", "b", "c", "d", "e"}, res)
	assert.NotEmpty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, []any{"a", "b", "c"}, res)
	assert.Empty(t, res.Errors)
}

func TestValidateArrayTypedSlices(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeArray, Items: &core.Schema{Type: core.TypeString}}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, []string{"a", "b"}, res)
	assert.Empty(t, res.Errors)

	intSchema := &core.Schema{Type: core.TypeArray, Items: &core.Schema{Type: core.TypeInteger}}
	intSchema.PrecomputeMessages()
	res.Reset()
	pb.Reset()
	schema.Validate(r, intSchema, pb, core.ModeWriteToServer, []int{1, 2, 3}, res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, intSchema, pb, core.ModeWriteToServer, []int8{1, 2}, res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, intSchema, pb, core.ModeWriteToServer, []int16{1, 2}, res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, intSchema, pb, core.ModeWriteToServer, []int32{1, 2}, res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, intSchema, pb, core.ModeWriteToServer, []int64{1, 2}, res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, intSchema, pb, core.ModeWriteToServer, []uint{1, 2}, res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, intSchema, pb, core.ModeWriteToServer, []uint16{1, 2}, res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, intSchema, pb, core.ModeWriteToServer, []uint32{1, 2}, res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, intSchema, pb, core.ModeWriteToServer, []uint64{1, 2}, res)
	assert.Empty(t, res.Errors)

	numSchema := &core.Schema{Type: core.TypeArray, Items: &core.Schema{Type: core.TypeNumber}}
	numSchema.PrecomputeMessages()
	res.Reset()
	pb.Reset()
	schema.Validate(r, numSchema, pb, core.ModeWriteToServer, []float32{1.1, 2.2}, res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, numSchema, pb, core.ModeWriteToServer, []float64{1.1, 2.2}, res)
	assert.Empty(t, res.Errors)
}

func TestValidateObjectTypeError(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeObject}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)
	res := &core.ValidateResult{}

	schema.Validate(r, s, pb, core.ModeWriteToServer, "not-an-object", res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateObjectMinMaxProperties(t *testing.T) {
	r := newRegistry()
	minProps := 2
	maxProps := 3
	s := &core.Schema{
		Type:          core.TypeObject,
		MinProperties: &minProps,
		MaxProperties: &maxProps,
	}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, map[string]any{"a": 1}, res)
	assert.NotEmpty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, map[string]any{"a": 1, "b": 2, "c": 3, "d": 4}, res)
	assert.NotEmpty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, map[string]any{"a": 1, "b": 2}, res)
	assert.Empty(t, res.Errors)
}

func TestValidateObjectAdditionalPropertiesFalse(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{
		Type: core.TypeObject,
		Properties: map[string]*core.Schema{
			"name": {Type: core.TypeString},
		},
		PropertyNames:        []string{"name"},
		AdditionalProperties: false,
		RequiredMap:          map[string]bool{},
	}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, map[string]any{"name": "Alice", "extra": "bad"}, res)
	assert.NotEmpty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, map[string]any{"name": "Alice"}, res)
	assert.Empty(t, res.Errors)
}

func TestValidateObjectAdditionalPropertiesSchema(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{
		Type: core.TypeObject,
		Properties: map[string]*core.Schema{
			"name": {Type: core.TypeString},
		},
		PropertyNames:        []string{"name"},
		AdditionalProperties: &core.Schema{Type: core.TypeInteger},
		RequiredMap:          map[string]bool{},
	}
	s.PrecomputeMessages()
	s.AdditionalProperties.(*core.Schema).PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, map[string]any{"name": "Alice", "age": float64(30)}, res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, map[string]any{"name": "Alice", "age": "not-int"}, res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateObjectWriteOnlyInReadMode(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{
		Type: core.TypeObject,
		Properties: map[string]*core.Schema{
			"password": {Type: core.TypeString, WriteOnly: true},
		},
		PropertyNames: []string{"password"},
		Required:      []string{},
		RequiredMap:   map[string]bool{},
	}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeReadFromServer, map[string]any{"password": "secret"}, res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateObjectReadOnlyRequiredInWriteMode(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{
		Type: core.TypeObject,
		Properties: map[string]*core.Schema{
			"id": {Type: core.TypeString, ReadOnly: true},
		},
		PropertyNames: []string{"id"},
		Required:      []string{"id"},
		RequiredMap:   map[string]bool{"id": true},
	}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, map[string]any{}, res)
	assert.Empty(t, res.Errors)
}

func TestValidateObjectWriteOnlyRequiredInReadMode(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{
		Type: core.TypeObject,
		Properties: map[string]*core.Schema{
			"secret": {Type: core.TypeString, WriteOnly: true},
		},
		PropertyNames: []string{"secret"},
		Required:      []string{"secret"},
		RequiredMap:   map[string]bool{"secret": true},
	}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeReadFromServer, map[string]any{}, res)
	assert.Empty(t, res.Errors)
}

func TestValidateObjectNullableOptionalField(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{
		Type: core.TypeObject,
		Properties: map[string]*core.Schema{
			"opt": {Type: core.TypeString, Nullable: true},
		},
		PropertyNames: []string{"opt"},
		RequiredMap:   map[string]bool{},
	}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, map[string]any{"opt": nil}, res)
	assert.Empty(t, res.Errors)
}

func TestValidateObjectDependentRequired(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{
		Type: core.TypeObject,
		Properties: map[string]*core.Schema{
			"a": {Type: core.TypeString},
			"b": {Type: core.TypeString},
		},
		PropertyNames:     []string{"a", "b"},
		RequiredMap:       map[string]bool{},
		DependentRequired: map[string][]string{"a": {"b"}},
	}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, map[string]any{"a": "val"}, res)
	assert.NotEmpty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, map[string]any{"a": "val", "b": "val2"}, res)
	assert.Empty(t, res.Errors)
}

func TestValidateObjectCaseInsensitiveMatch(t *testing.T) {
	old := schema.ValidateStrictCasing
	schema.ValidateStrictCasing = false
	defer func() { schema.ValidateStrictCasing = old }()

	r := newRegistry()
	s := &core.Schema{
		Type: core.TypeObject,
		Properties: map[string]*core.Schema{
			"name": {Type: core.TypeString},
		},
		PropertyNames:        []string{"name"},
		Required:             []string{"name"},
		RequiredMap:          map[string]bool{"name": true},
		AdditionalProperties: false,
	}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, map[string]any{"Name": "Alice"}, res)
	assert.Empty(t, res.Errors)
}

func TestValidateObjectStrictCasing(t *testing.T) {
	old := schema.ValidateStrictCasing
	schema.ValidateStrictCasing = true
	defer func() { schema.ValidateStrictCasing = old }()

	r := newRegistry()
	s := &core.Schema{
		Type: core.TypeObject,
		Properties: map[string]*core.Schema{
			"name": {Type: core.TypeString},
		},
		PropertyNames:        []string{"name"},
		Required:             []string{"name"},
		RequiredMap:          map[string]bool{"name": true},
		AdditionalProperties: false,
	}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, map[string]any{"Name": "Alice"}, res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateMapAnyBasic(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{
		Type: core.TypeObject,
		Properties: map[string]*core.Schema{
			"name": {Type: core.TypeString},
		},
		PropertyNames: []string{"name"},
		Required:      []string{"name"},
		RequiredMap:   map[string]bool{"name": true},
	}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, map[any]any{"name": "Alice"}, res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, map[any]any{}, res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateMapAnyMinMaxProperties(t *testing.T) {
	r := newRegistry()
	minProps := 1
	maxProps := 2
	s := &core.Schema{
		Type:          core.TypeObject,
		MinProperties: &minProps,
		MaxProperties: &maxProps,
	}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, map[any]any{}, res)
	assert.NotEmpty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, map[any]any{"a": 1, "b": 2, "c": 3}, res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateMapAnyAdditionalPropertiesFalse(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{
		Type: core.TypeObject,
		Properties: map[string]*core.Schema{
			"name": {Type: core.TypeString},
		},
		PropertyNames:        []string{"name"},
		AdditionalProperties: false,
		RequiredMap:          map[string]bool{},
	}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, map[any]any{"name": "Alice", "extra": "bad"}, res)
	assert.NotEmpty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, map[any]any{"name": "Alice", 42: "bad"}, res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateMapAnyAdditionalPropertiesSchema(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{
		Type:                 core.TypeObject,
		Properties:           map[string]*core.Schema{},
		PropertyNames:        []string{},
		AdditionalProperties: &core.Schema{Type: core.TypeInteger},
		RequiredMap:          map[string]bool{},
	}
	s.PrecomputeMessages()
	s.AdditionalProperties.(*core.Schema).PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, map[any]any{99: float64(42), "extra": float64(7)}, res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, map[any]any{"bad": "string"}, res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateMapAnyWriteOnlyReadMode(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{
		Type: core.TypeObject,
		Properties: map[string]*core.Schema{
			"secret": {Type: core.TypeString, WriteOnly: true},
		},
		PropertyNames: []string{"secret"},
		RequiredMap:   map[string]bool{},
	}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeReadFromServer, map[any]any{"secret": "value"}, res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateMapAnyDependentRequired(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{
		Type: core.TypeObject,
		Properties: map[string]*core.Schema{
			"a": {Type: core.TypeString},
			"b": {Type: core.TypeString},
		},
		PropertyNames:     []string{"a", "b"},
		RequiredMap:       map[string]bool{},
		DependentRequired: map[string][]string{"a": {"b"}},
	}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, map[any]any{"a": "val"}, res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateMapAnyNullableOptional(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{
		Type: core.TypeObject,
		Properties: map[string]*core.Schema{
			"opt": {Type: core.TypeString, Nullable: true},
		},
		PropertyNames: []string{"opt"},
		RequiredMap:   map[string]bool{},
	}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, map[any]any{"opt": nil}, res)
	assert.Empty(t, res.Errors)
}

func TestValidateMapAnyReadOnlyWriteMode(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{
		Type: core.TypeObject,
		Properties: map[string]*core.Schema{
			"id": {Type: core.TypeString, ReadOnly: true},
		},
		PropertyNames: []string{"id"},
		Required:      []string{"id"},
		RequiredMap:   map[string]bool{"id": true},
	}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, map[any]any{}, res)
	assert.Empty(t, res.Errors)
}

func TestValidateDiscriminator(t *testing.T) {
	r := newRegistry()
	catSchema := &core.Schema{
		Type: core.TypeObject,
		Properties: map[string]*core.Schema{
			"type":  {Type: core.TypeString},
			"meows": {Type: core.TypeBoolean},
		},
		PropertyNames: []string{"meows", "type"},
		RequiredMap:   map[string]bool{},
	}
	catSchema.PrecomputeMessages()

	dogSchema := &core.Schema{
		Type: core.TypeObject,
		Properties: map[string]*core.Schema{
			"type":  {Type: core.TypeString},
			"barks": {Type: core.TypeBoolean},
		},
		PropertyNames: []string{"barks", "type"},
		RequiredMap:   map[string]bool{},
	}
	dogSchema.PrecomputeMessages()

	r.Schema(reflect.TypeFor[struct {
		Type  string `json:"type"`
		Meows bool   `json:"meows"`
	}](), true, "Cat")
	r.Schema(reflect.TypeFor[struct {
		Type  string `json:"type"`
		Barks bool   `json:"barks"`
	}](), true, "Dog")

	s := &core.Schema{
		OneOf: []*core.Schema{
			{Ref: "#/components/schemas/Cat"},
			{Ref: "#/components/schemas/Dog"},
		},
		Discriminator: &core.Discriminator{
			PropertyName: "type",
			Mapping: map[string]string{
				"cat": "#/components/schemas/Cat",
				"dog": "#/components/schemas/Dog",
			},
		},
	}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, map[string]any{"type": "cat", "meows": true}, res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, map[string]any{"meows": true}, res)
	assert.NotEmpty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, map[string]any{"type": "bird"}, res)
	assert.NotEmpty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, map[string]any{"type": 42}, res)
}

func TestValidateDiscriminatorMapAny(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{
		OneOf: []*core.Schema{
			{Type: core.TypeObject},
		},
		Discriminator: &core.Discriminator{
			PropertyName: "type",
			Mapping:      map[string]string{},
		},
	}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, map[any]any{"type": "cat"}, res)
}

func TestValidateDiscriminatorNilValue(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{
		OneOf: []*core.Schema{
			{Type: core.TypeObject},
		},
		Discriminator: &core.Discriminator{
			PropertyName: "type",
			Mapping:      map[string]string{},
		},
	}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, "not-a-map", res)
}

func TestValidateOneOfMatchesMultiple(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{
		OneOf: []*core.Schema{
			{Type: core.TypeNumber},
			{Type: core.TypeNumber},
		},
	}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, float64(1), res)
	assert.NotEmpty(t, res.Errors, "should fail when multiple oneOf schemas match")
}

func TestValidateAnyOfNoMatch(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{
		AnyOf: []*core.Schema{
			{Type: core.TypeString},
			{Type: core.TypeInteger},
		},
	}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, true, res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateRef(t *testing.T) {
	r := newRegistry()
	type Inner struct {
		Value string `json:"value"`
	}
	r.Schema(reflect.TypeFor[Inner](), true, "Inner")

	s := &core.Schema{Ref: "#/components/schemas/Inner"}
	pb := core.NewPathBuffer([]byte{}, 0)
	res := &core.ValidateResult{}

	schema.Validate(r, s, pb, core.ModeWriteToServer, map[string]any{"value": "hello"}, res)
	assert.Empty(t, res.Errors)
}

func TestEnsureTypeBooleanPanics(t *testing.T) {
	r := newRegistry()
	type S struct {
		Flag bool `json:"flag" default:"wrong"`
	}
	assert.Panics(t, func() {
		r.Schema(reflect.TypeFor[S](), false, "S")
	})
}

func TestEnsureTypeIntegerPanics(t *testing.T) {
	r := newRegistry()
	type S struct {
		Count int `json:"count" default:"hello"`
	}
	assert.Panics(t, func() {
		r.Schema(reflect.TypeFor[S](), false, "S")
	})
}

func TestEnsureTypeIntegerFloatPanics(t *testing.T) {
	r := newRegistry()
	type S struct {
		Count int `json:"count" default:"1.5"`
	}
	assert.Panics(t, func() {
		r.Schema(reflect.TypeFor[S](), false, "S")
	})
}

func TestEnsureTypeArrayPanics(t *testing.T) {
	r := newRegistry()
	type S struct {
		Tags []int `json:"tags" default:"not-an-array"`
	}
	assert.Panics(t, func() {
		r.Schema(reflect.TypeFor[S](), false, "S")
	})
}

func TestEnsureTypeArrayItemValidation(t *testing.T) {
	r := newRegistry()
	type S struct {
		Tags []int `json:"tags" default:"[1,2,3]"`
	}
	s := r.Schema(reflect.TypeFor[S](), false, "S")
	require.NotNil(t, s)
	prop := s.Properties["tags"]
	require.NotNil(t, prop)
	require.NotNil(t, prop.Default)
}

func TestEnsureTypeObjectPanics(t *testing.T) {
	r := newRegistry()
	type S struct {
		Meta map[string]string `json:"meta" default:"not-json"`
	}
	assert.Panics(t, func() {
		r.Schema(reflect.TypeFor[S](), false, "S")
	})
}

func TestEnsureTypeStringPanics(t *testing.T) {
	r := newRegistry()
	type S struct {
		Name string `json:"name" default:"42"`
	}
	s := r.Schema(reflect.TypeFor[S](), false, "S")
	require.NotNil(t, s)
	require.NotNil(t, s.Properties["name"])
	assert.Equal(t, "42", s.Properties["name"].Default)
}

func TestConvertTypeSlice(t *testing.T) {
	v := schema.ConvertType("test", reflect.TypeFor[[]int](), []any{float64(1), float64(2)})
	assert.Equal(t, []int{1, 2}, v)
}

func TestConvertTypeSlicePointer(t *testing.T) {
	v := schema.ConvertType("test", reflect.TypeFor[[]*int](), []any{float64(1)})
	result := v.([]*int)
	require.Len(t, result, 1)
	assert.Equal(t, 1, *result[0])
}

func TestConvertTypePointer(t *testing.T) {
	v := schema.ConvertType("test", reflect.TypeFor[*int](), float64(42))
	assert.Equal(t, 42, *v.(*int))
}

func TestConvertTypeSliceIncompatiblePanics(t *testing.T) {
	assert.Panics(t, func() {
		schema.ConvertType("test", reflect.TypeFor[[]int](), []any{"not-a-number"})
	})
}

func TestConvertTypeIncompatiblePanics(t *testing.T) {
	assert.Panics(t, func() {
		schema.ConvertType("test", reflect.TypeFor[int](), "not-a-number")
	})
}

func TestJsonTagValueString(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString}
	v := schema.JsonTagValue(r, "test", s, "hello")
	assert.Equal(t, "hello", v)
}

func TestJsonTagValueStringArray(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{
		Type:  core.TypeArray,
		Items: &core.Schema{Type: core.TypeString},
	}
	v := schema.JsonTagValue(r, "test", s, "a, b, c")
	assert.Equal(t, []string{"a", "b", "c"}, v)
}

func TestJsonTagValueStringArrayJSON(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{
		Type:  core.TypeArray,
		Items: &core.Schema{Type: core.TypeString},
	}
	v := schema.JsonTagValue(r, "test", s, `["a","b"]`)
	arr, ok := v.([]any)
	require.True(t, ok)
	assert.Len(t, arr, 2)
}

func TestJsonTagValueRef(t *testing.T) {
	r := newRegistry()
	type Inner struct {
		Value string `json:"value"`
	}
	r.Schema(reflect.TypeFor[Inner](), true, "Inner")

	s := &core.Schema{Ref: "#/components/schemas/Inner"}
	v := schema.JsonTagValue(r, "test", s, `{"value":"hello"}`)
	assert.NotNil(t, v)
}

func TestJsonTagValueRefNotFound(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Ref: "#/components/schemas/NotExist"}
	v := schema.JsonTagValue(r, "test", s, "anything")
	assert.Nil(t, v)
}

func TestJsonTagValueInvalidJSON(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeInteger}
	assert.Panics(t, func() {
		schema.JsonTagValue(r, "test", s, "not-json")
	})
}

func TestBoolTagInvalidPanics(t *testing.T) {
	r := newRegistry()
	type S struct {
		Flag string `json:"flag" nullable:"maybe"`
	}
	assert.Panics(t, func() {
		r.Schema(reflect.TypeFor[S](), false, "S")
	})
}

func TestIntTagInvalidPanics(t *testing.T) {
	r := newRegistry()
	type S struct {
		Name string `json:"name" minLength:"abc"`
	}
	assert.Panics(t, func() {
		r.Schema(reflect.TypeFor[S](), false, "S")
	})
}

func TestFloatTagInvalidPanics(t *testing.T) {
	r := newRegistry()
	type S struct {
		Score float64 `json:"score" minimum:"abc"`
	}
	assert.Panics(t, func() {
		r.Schema(reflect.TypeFor[S](), false, "S")
	})
}

func TestGetFieldsEmbedded(t *testing.T) {
	r := newRegistry()

	type Base struct {
		ID string `json:"id"`
	}
	type Child struct {
		Base
		Name string `json:"name"`
	}

	s := r.Schema(reflect.TypeFor[Child](), false, "Child")
	require.NotNil(t, s)
	assert.Contains(t, s.PropertyNames, "id")
	assert.Contains(t, s.PropertyNames, "name")
}

func TestGetFieldsEmbeddedPointer(t *testing.T) {
	r := newRegistry()

	type Base struct {
		ID string `json:"id"`
	}
	type Child struct {
		*Base
		Name string `json:"name"`
	}

	s := r.Schema(reflect.TypeFor[Child](), false, "Child")
	require.NotNil(t, s)
	assert.Contains(t, s.PropertyNames, "id")
	assert.Contains(t, s.PropertyNames, "name")
}

func TestGetFieldsOverride(t *testing.T) {
	r := newRegistry()

	type Base struct {
		Name string `json:"name" minLength:"1"`
	}
	type Child struct {
		Base
		Name string `json:"name" minLength:"5"`
	}

	s := r.Schema(reflect.TypeFor[Child](), false, "Child")
	require.NotNil(t, s)
	require.NotNil(t, s.Properties["name"])
	assert.Equal(t, 5, *s.Properties["name"].MinLength)
}

func TestFromFieldNullableObjectPanics(t *testing.T) {
	r := newRegistry()

	type Inner struct {
		X string `json:"x"`
	}
	type S struct {
		Inner Inner `json:"inner" nullable:"true"`
	}

	assert.Panics(t, func() {
		r.Schema(reflect.TypeFor[S](), false, "S")
	})
}

func TestFromFieldCustomFormat(t *testing.T) {
	r := newRegistry()

	type S struct {
		Val string `json:"val" format:"custom-format"`
	}

	f, _ := reflect.TypeFor[S]().FieldByName("Val")
	s := schema.FromField(r, f, "")
	require.NotNil(t, s)
	assert.Equal(t, "custom-format", s.Format)
}

func TestFromFieldCustomTimeFormat(t *testing.T) {
	r := newRegistry()

	type S struct {
		Val string `json:"val" timeFormat:"custom"`
	}

	f, _ := reflect.TypeFor[S]().FieldByName("Val")
	s := schema.FromField(r, f, "")
	require.NotNil(t, s)
	assert.Equal(t, "custom", s.Format)
}

func TestFromFieldInt64AsString(t *testing.T) {
	r := newRegistry()

	type S struct {
		BigID int64 `json:"big_id,string"`
	}

	f, _ := reflect.TypeFor[S]().FieldByName("BigID")
	s := schema.FromField(r, f, "")
	require.NotNil(t, s)
	assert.Equal(t, core.TypeString, s.Type)
}

func TestFromFieldUint64AsString(t *testing.T) {
	r := newRegistry()

	type S struct {
		BigID uint64 `json:"big_id,string"`
	}

	f, _ := reflect.TypeFor[S]().FieldByName("BigID")
	s := schema.FromField(r, f, "")
	require.NotNil(t, s)
	assert.Equal(t, core.TypeString, s.Type)
}

func TestFromFieldOmitemptyPointerNotNullable(t *testing.T) {
	r := newRegistry()

	type S struct {
		Val *string `json:"val,omitempty"`
	}

	s := r.Schema(reflect.TypeFor[S](), false, "S")
	require.NotNil(t, s)
	require.NotNil(t, s.Properties["val"])
	assert.False(t, s.Properties["val"].Nullable)
}

func TestFromFieldOmitzeroOptional(t *testing.T) {
	r := newRegistry()

	type S struct {
		Val string `json:"val,omitzero"`
	}

	s := r.Schema(reflect.TypeFor[S](), false, "S")
	require.NotNil(t, s)
	assert.NotContains(t, s.Required, "val")
}

func TestFromFieldRequired(t *testing.T) {
	r := newRegistry()

	type S struct {
		Val string `json:"val,omitempty" required:"true"`
	}

	s := r.Schema(reflect.TypeFor[S](), false, "S")
	require.NotNil(t, s)
	assert.Contains(t, s.Required, "val")
}

func TestFromFieldHiddenNotRequired(t *testing.T) {
	r := newRegistry()

	type S struct {
		Secret string `json:"secret" hidden:"true"`
	}

	s := r.Schema(reflect.TypeFor[S](), false, "S")
	require.NotNil(t, s)
	assert.NotContains(t, s.Required, "secret")
}

type customSchemaType struct{}

func (c *customSchemaType) Schema(_ core.Registry) *core.Schema {
	return &core.Schema{Type: core.TypeString, Format: "custom"}
}

func TestFromTypeSchemaProvider(t *testing.T) {
	r := newRegistry()
	s := r.Schema(reflect.TypeFor[customSchemaType](), false, "")
	require.NotNil(t, s)
	assert.Equal(t, core.TypeString, s.Type)
	assert.Equal(t, "custom", s.Format)
}

type transformedType struct {
	Name string `json:"name"`
}

func (tt *transformedType) TransformSchema(_ core.Registry, s *core.Schema) *core.Schema {
	s.Description = "transformed"
	return s
}

func TestFromTypeSchemaTransformer(t *testing.T) {
	r := newRegistry()
	s := r.Schema(reflect.TypeFor[transformedType](), false, "")
	require.NotNil(t, s)
	assert.Equal(t, "transformed", s.Description)
}

func TestSchemaAdditionalPropertiesTag(t *testing.T) {
	r := newRegistry()

	type S struct {
		_    struct{} `additionalProperties:"true"`
		Name string   `json:"name"`
	}

	s := r.Schema(reflect.TypeFor[S](), false, "S")
	require.NotNil(t, s)
	assert.True(t, s.AdditionalProperties.(bool))
}

func TestSchemaNullableTag(t *testing.T) {
	r := newRegistry()

	type S struct {
		_    struct{} `nullable:"true"`
		Name string   `json:"name"`
	}

	s := r.Schema(reflect.TypeFor[S](), false, "S")
	require.NotNil(t, s)
	assert.True(t, s.Nullable)
}

func TestDependentRequiredInvalidFieldPanics(t *testing.T) {
	r := newRegistry()

	type S struct {
		A string `json:"a" dependentRequired:"nonexistent"`
	}

	assert.Panics(t, func() {
		r.Schema(reflect.TypeFor[S](), false, "S")
	})
}

func TestFromFieldOneOfTag(t *testing.T) {
	r := newRegistry()

	type S struct {
		Val any `json:"val" oneOf:"TypeA,TypeB"`
	}

	f, _ := reflect.TypeFor[S]().FieldByName("Val")
	s := schema.FromField(r, f, "")
	require.NotNil(t, s)
	require.NotNil(t, s.OneOf)
	assert.Len(t, s.OneOf, 2)
	assert.Equal(t, "#/components/schemas/TypeA", s.OneOf[0].Ref)
	assert.Equal(t, "#/components/schemas/TypeB", s.OneOf[1].Ref)
	assert.Empty(t, s.Type, "type should be cleared when oneOf is set")
}

func TestFromFieldAnyOfTag(t *testing.T) {
	r := newRegistry()

	type S struct {
		Val any `json:"val" anyOf:"TypeA,TypeB"`
	}

	f, _ := reflect.TypeFor[S]().FieldByName("Val")
	s := schema.FromField(r, f, "")
	require.NotNil(t, s)
	require.NotNil(t, s.AnyOf)
	assert.Len(t, s.AnyOf, 2)
}

func TestFromFieldAllOfTag(t *testing.T) {
	r := newRegistry()

	type S struct {
		Val any `json:"val" allOf:"TypeA"`
	}

	f, _ := reflect.TypeFor[S]().FieldByName("Val")
	s := schema.FromField(r, f, "")
	require.NotNil(t, s)
	require.NotNil(t, s.AllOf)
	assert.Len(t, s.AllOf, 1)
}

func TestFromFieldArrayMinMax(t *testing.T) {
	r := newRegistry()

	type S struct {
		Scores []float64 `json:"scores" minimum:"0" maximum:"100"`
	}

	f, _ := reflect.TypeFor[S]().FieldByName("Scores")
	s := schema.FromField(r, f, "")
	require.NotNil(t, s)
	require.NotNil(t, s.Items)
	assert.NotNil(t, s.Items.Minimum)
	assert.NotNil(t, s.Items.Maximum)
}

func TestFromFieldArrayEnum(t *testing.T) {
	r := newRegistry()

	type S struct {
		Tags []string `json:"tags" enum:"a,b,c"`
	}

	f, _ := reflect.TypeFor[S]().FieldByName("Tags")
	s := schema.FromField(r, f, "")
	require.NotNil(t, s)
	require.NotNil(t, s.Items)
	assert.Equal(t, []any{"a", "b", "c"}, s.Items.Enum)
}

func TestMapRegistryTypeFromRef(t *testing.T) {
	r := newRegistry()
	type S struct {
		X string `json:"x"`
	}
	r.Schema(reflect.TypeFor[S](), true, "S")

	typ := r.TypeFromRef("#/components/schemas/S")
	require.NotNil(t, typ)
	assert.Equal(t, "S", typ.Name())
}

func TestMapRegistryMap(t *testing.T) {
	r := newRegistry()
	type S struct {
		X string `json:"x"`
	}
	r.Schema(reflect.TypeFor[S](), true, "S")

	m := r.Map()
	assert.Contains(t, m, "S")
}

func TestMapRegistryMarshalYAML(t *testing.T) {
	r := newRegistry()
	type S struct {
		X string `json:"x"`
	}
	r.Schema(reflect.TypeFor[S](), true, "S")

	v, err := r.MarshalYAML()
	require.NoError(t, err)
	assert.NotNil(t, v)
}

func TestMapRegistryDuplicateNamePanics(t *testing.T) {
	r := schema.NewMapRegistry(func(t reflect.Type, hint string) string {
		return "SameName"
	})

	type A struct {
		X string `json:"x"`
	}
	r.Schema(reflect.TypeFor[A](), true, "")

	type B struct {
		Y int `json:"y"`
	}
	assert.Panics(t, func() {
		r.Schema(reflect.TypeFor[B](), true, "")
	})
}

func TestMapRegistryAllowRefFalse(t *testing.T) {
	r := newRegistry()
	type S struct {
		X string `json:"x"`
	}
	_ = r.Schema(reflect.TypeFor[S](), true, "S")
	s := r.Schema(reflect.TypeFor[S](), false, "S")
	require.NotNil(t, s)
	assert.Equal(t, core.TypeObject, s.Type)
}

func TestFindDefaultsNested(t *testing.T) {
	r := newRegistry()

	type Inner struct {
		Color string `json:"color" default:"red"`
	}
	type Outer struct {
		Inner Inner `json:"inner"`
	}

	defaults := schema.FindDefaults(r, reflect.TypeFor[Outer]())
	require.NotNil(t, defaults)
	assert.NotEmpty(t, defaults.Paths)

	v := Outer{}
	rv := reflect.ValueOf(&v).Elem()
	schema.ApplyDefaults(defaults, rv)
	assert.Equal(t, "red", v.Inner.Color)
}

func TestFindDefaultsSlice(t *testing.T) {
	r := newRegistry()

	type Item struct {
		Color string `json:"color" default:"blue"`
	}
	type Outer struct {
		Items []Item `json:"items"`
	}

	defaults := schema.FindDefaults(r, reflect.TypeFor[Outer]())
	require.NotNil(t, defaults)
	assert.NotEmpty(t, defaults.Paths)

	v := Outer{Items: []Item{{}, {Color: "red"}}}
	rv := reflect.ValueOf(&v).Elem()
	schema.ApplyDefaults(defaults, rv)
	assert.Equal(t, "blue", v.Items[0].Color)
	assert.Equal(t, "red", v.Items[1].Color) // Not overwritten
}

func TestFindDefaultsMap(t *testing.T) {
	r := newRegistry()

	type Item struct {
		Color string `json:"color" default:"green"`
	}
	type Outer struct {
		Items map[string]Item `json:"items"`
	}

	defaults := schema.FindDefaults(r, reflect.TypeFor[Outer]())
	require.NotNil(t, defaults)
}

func TestFindDefaultsPointerField(t *testing.T) {
	r := newRegistry()

	type S struct {
		Count *int `json:"count" default:"5"`
	}

	defaults := schema.FindDefaults(r, reflect.TypeFor[S]())
	require.NotNil(t, defaults)

	v := S{}
	rv := reflect.ValueOf(&v).Elem()
	schema.ApplyDefaults(defaults, rv)
	require.NotNil(t, v.Count)
	assert.Equal(t, 5, *v.Count)
}

func TestFindDefaultsPointerStructPanics(t *testing.T) {
	r := newRegistry()

	type Inner struct {
		X string `json:"x"`
	}
	type S struct {
		Inner *Inner `json:"inner" default:"{}"`
	}

	assert.Panics(t, func() {
		schema.FindDefaults(r, reflect.TypeFor[S]())
	})
}

func TestEveryPB(t *testing.T) {
	r := newRegistry()

	type Input struct {
		Limit int `query:"limit" default:"10"`
	}

	defaults := schema.FindDefaults(r, reflect.TypeFor[Input]())
	require.NotNil(t, defaults)

	v := Input{}
	rv := reflect.ValueOf(&v).Elem()
	pb := core.NewPathBuffer([]byte{}, 0)

	var paths []string
	defaults.EveryPB(pb, rv, func(item reflect.Value, def any) {
		paths = append(paths, pb.String())
		if item.IsZero() {
			item.Set(reflect.ValueOf(def))
		}
	})
	assert.NotEmpty(t, paths)
	assert.Equal(t, 10, v.Limit)
}

func TestEveryPBWithHeader(t *testing.T) {
	r := newRegistry()

	type Input struct {
		Token string `header:"X-Token" default:"default-token"`
	}

	defaults := schema.FindDefaults(r, reflect.TypeFor[Input]())
	require.NotNil(t, defaults)

	v := Input{}
	rv := reflect.ValueOf(&v).Elem()
	pb := core.NewPathBuffer([]byte{}, 0)

	var paths []string
	defaults.EveryPB(pb, rv, func(item reflect.Value, def any) {
		paths = append(paths, pb.String())
	})
	assert.NotEmpty(t, paths)
}

func TestEveryPBWithPath(t *testing.T) {
	r := newRegistry()

	type Input struct {
		ID string `path:"id" default:"default-id"`
	}

	defaults := schema.FindDefaults(r, reflect.TypeFor[Input]())
	require.NotNil(t, defaults)

	v := Input{}
	rv := reflect.ValueOf(&v).Elem()
	pb := core.NewPathBuffer([]byte{}, 0)

	var paths []string
	defaults.EveryPB(pb, rv, func(item reflect.Value, def any) {
		paths = append(paths, pb.String())
	})
	assert.NotEmpty(t, paths)
}

func TestEveryNilPointer(t *testing.T) {
	r := newRegistry()

	type Inner struct {
		Color string `json:"color" default:"red"`
	}
	type Outer struct {
		Inner *Inner `json:"inner"`
	}

	defaults := schema.FindDefaults(r, reflect.TypeFor[Outer]())

	v := Outer{} // Inner is nil
	rv := reflect.ValueOf(&v).Elem()
	schema.ApplyDefaults(defaults, rv)
}

func TestEnsureTypeObjectInvalid(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeObject}
	assert.Panics(t, func() {
		schema.JsonTagValue(r, "test", s, "[1,2,3]")
	})
}

func TestEnsureTypeObjectWithProperties(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{
		Type: core.TypeObject,
		Properties: map[string]*core.Schema{
			"name": {Type: core.TypeString},
		},
	}
	v := schema.JsonTagValue(r, "test", s, `{"name":"test"}`)
	require.NotNil(t, v)
}

func TestFromTypeUnsupportedKind(t *testing.T) {
	r := newRegistry()
	s := r.Schema(reflect.TypeFor[chan int](), false, "")
	assert.Nil(t, s)
}

func TestSchemaPointerToArrayDecays(t *testing.T) {
	r := newRegistry()
	s := r.Schema(reflect.TypeFor[*[]string](), false, "")
	require.NotNil(t, s)
	assert.Equal(t, core.TypeArray, s.Type)
}

func TestSchemaPointerToSliceDecays(t *testing.T) {
	r := newRegistry()
	s := r.Schema(reflect.TypeFor[*[3]int](), false, "")
	require.NotNil(t, s)
	assert.Equal(t, core.TypeArray, s.Type)
}

func TestSchemaUintMinZero(t *testing.T) {
	r := newRegistry()
	s := r.Schema(reflect.TypeFor[uint](), false, "")
	require.NotNil(t, s)
	assert.Equal(t, core.TypeInteger, s.Type)
	require.NotNil(t, s.Minimum)
	assert.InDelta(t, 0.0, *s.Minimum, 0.01)
}

func TestModelValidatorReuse(t *testing.T) {
	type S struct {
		Name string `json:"name" minLength:"1"`
	}

	v := schema.NewModelValidator()

	errs := v.Validate(reflect.TypeFor[S](), map[string]any{"name": "ok"})
	assert.Nil(t, errs)

	errs = v.Validate(reflect.TypeFor[S](), map[string]any{"name": ""})
	assert.NotNil(t, errs)
}

func TestEnsureTypeRefNil(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Ref: "#/components/schemas/Unknown"}

	v := schema.JsonTagValue(r, "test", s, "anything")
	assert.Nil(t, v)
}

func TestDefaultArrayNullable(t *testing.T) {
	old := schema.DefaultArrayNullable
	defer func() { schema.DefaultArrayNullable = old }()

	schema.DefaultArrayNullable = false
	r := newRegistry()
	s := r.Schema(reflect.TypeFor[[]string](), false, "")
	require.NotNil(t, s)
	assert.False(t, s.Nullable)

	schema.DefaultArrayNullable = true
	r2 := newRegistry()
	s2 := r2.Schema(reflect.TypeFor[[]string](), false, "")
	require.NotNil(t, s2)
	assert.True(t, s2.Nullable)
}

func TestSchemaFromRefWrongPrefix(t *testing.T) {
	r := newRegistry()
	s := r.SchemaFromRef("#/wrong/prefix/Foo")
	assert.Nil(t, s)
}

func TestValidateNumberMinMax(t *testing.T) {
	r := newRegistry()
	lo := 5.0
	hi := 10.0
	s := &core.Schema{Type: core.TypeNumber, Minimum: &lo, Maximum: &hi}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, float64(3), res)
	assert.NotEmpty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, float64(15), res)
	assert.NotEmpty(t, res.Errors)
}

func TestMapRegistryMarshalJSONContainsSchema(t *testing.T) {
	r := newRegistry()
	type Person struct {
		Name string `json:"name"`
	}
	r.Schema(reflect.TypeFor[Person](), true, "Person")

	b, err := r.MarshalJSON()
	require.NoError(t, err)

	var m map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(b, &m))
	assert.Contains(t, m, "Person")
}

func TestJsonFieldNameWithoutTag(t *testing.T) {
	r := newRegistry()

	type Inner struct {
		NoTag string // No json tag, so jsonFieldName lowercases the field name
	}
	type Outer struct {
		Inner Inner `json:"inner"`
	}

	_ = r.Schema(reflect.TypeFor[Outer](), false, "Outer")

	type WithDefault struct {
		Inner struct {
			NoTag string `default:"hello"`
		} `json:"inner"`
	}

	defaults := schema.FindDefaults(r, reflect.TypeFor[WithDefault]())
	require.NotNil(t, defaults)
	assert.NotEmpty(t, defaults.Paths, "should find default for NoTag field")

	val := WithDefault{}
	v := reflect.ValueOf(&val).Elem()
	schema.ApplyDefaults(defaults, v)
	assert.Equal(t, "hello", val.Inner.NoTag)
}

func TestJsonFieldNameWithJsonTag(t *testing.T) {
	r := newRegistry()

	type WithJsonTag struct {
		Labeled string `json:"custom_label" default:"tagged"`
	}

	defaults := schema.FindDefaults(r, reflect.TypeFor[WithJsonTag]())
	require.NotNil(t, defaults)
	assert.NotEmpty(t, defaults.Paths)

	val := WithJsonTag{}
	v := reflect.ValueOf(&val).Elem()
	schema.ApplyDefaults(defaults, v)
	assert.Equal(t, "tagged", val.Labeled)
}

func TestDefaultsEveryWithSlice(t *testing.T) {
	r := newRegistry()

	type Item struct {
		Color string `json:"color" default:"red"`
	}

	defaults := schema.FindDefaults(r, reflect.TypeFor[[]Item]())
	require.NotNil(t, defaults)
	assert.NotEmpty(t, defaults.Paths, "should find defaults inside slice element type")

	items := []Item{{}, {Color: "blue"}}
	v := reflect.ValueOf(&items).Elem()
	defaults.Every(v, func(item reflect.Value, def any) {
		if item.IsZero() {
			item.Set(reflect.Indirect(reflect.ValueOf(def)))
		}
	})
	assert.Equal(t, "red", items[0].Color, "zero-valued item should get default")
	assert.Equal(t, "blue", items[1].Color, "non-zero item should be unchanged")
}

func TestDefaultsEveryWithMap(t *testing.T) {
	r := newRegistry()

	type Item struct {
		Color string `json:"color" default:"green"`
	}

	defaults := schema.FindDefaults(r, reflect.TypeFor[map[string]Item]())
	require.NotNil(t, defaults)
	assert.NotEmpty(t, defaults.Paths, "should find defaults inside map value type")
}

func TestDefaultsEveryPBWithMapAndSlice(t *testing.T) {
	r := newRegistry()

	type Item struct {
		Size int `json:"size" default:"10"`
	}
	type Container struct {
		Items []Item `json:"items"`
	}

	defaults := schema.FindDefaults(r, reflect.TypeFor[Container]())
	require.NotNil(t, defaults)

	container := Container{
		Items: []Item{{}, {Size: 5}},
	}
	v := reflect.ValueOf(&container).Elem()
	pb := core.NewPathBuffer(make([]byte, 0, 128), 0)

	var paths []string
	defaults.EveryPB(pb, v, func(item reflect.Value, def any) {
		paths = append(paths, string(pb.Bytes()))
		if item.Kind() != reflect.Invalid && item.IsZero() {
			item.Set(reflect.Indirect(reflect.ValueOf(def)))
		}
	})

	assert.Equal(t, 10, container.Items[0].Size, "zero-valued item should get default")
	assert.Equal(t, 5, container.Items[1].Size, "non-zero item should be unchanged")
	assert.NotEmpty(t, paths, "EveryPB should have visited at least one path")
}

func TestEnsureTypeWithObjectViaJsonTagValue(t *testing.T) {
	r := newRegistry()

	type Inner struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	s := r.Schema(reflect.TypeFor[Inner](), false, "Inner")
	require.NotNil(t, s)
	assert.Equal(t, core.TypeObject, s.Type)

	result := schema.JsonTagValue(r, "TestField", s, `{"name":"Alice","age":30}`)
	require.NotNil(t, result)

	obj, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Alice", obj["name"])
	assert.InDelta(t, 30.0, obj["age"], 0.001)
}

func TestEnsureTypeWithArray(t *testing.T) {
	r := newRegistry()

	type Outer struct {
		Tags []string `json:"tags" default:"[\"a\",\"b\"]"`
	}

	defaults := schema.FindDefaults(r, reflect.TypeFor[Outer]())
	require.NotNil(t, defaults)
	assert.NotEmpty(t, defaults.Paths)

	val := Outer{}
	v := reflect.ValueOf(&val).Elem()
	schema.ApplyDefaults(defaults, v)
	assert.Equal(t, []string{"a", "b"}, val.Tags)
}

func TestEnsureTypeWithArrayOfIntegers(t *testing.T) {
	r := newRegistry()

	type Outer struct {
		Scores []int `json:"scores" default:"[1,2,3]"`
	}

	defaults := schema.FindDefaults(r, reflect.TypeFor[Outer]())
	require.NotNil(t, defaults)
	assert.NotEmpty(t, defaults.Paths)

	val := Outer{}
	v := reflect.ValueOf(&val).Elem()
	schema.ApplyDefaults(defaults, v)
	assert.Equal(t, []int{1, 2, 3}, val.Scores)
}

type plainRegistry struct {
	inner *schema.MapRegistry
}

func (p *plainRegistry) Schema(t reflect.Type, allowRef bool, hint string) *core.Schema {
	return p.inner.Schema(t, allowRef, hint)
}

func (p *plainRegistry) SchemaFromRef(ref string) *core.Schema {
	return p.inner.SchemaFromRef(ref)
}

func (p *plainRegistry) TypeFromRef(ref string) reflect.Type {
	return p.inner.TypeFromRef(ref)
}

func (p *plainRegistry) Map() map[string]*core.Schema {
	return p.inner.Map()
}

func (p *plainRegistry) RegisterTypeAlias(t reflect.Type, alias reflect.Type) {
	p.inner.RegisterTypeAlias(t, alias)
}

func TestGetRegistryConfigFallback(t *testing.T) {
	inner := schema.NewMapRegistry(schema.DefaultSchemaNamer)
	pr := &plainRegistry{inner: inner}

	type Example struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	s := schema.FromType(pr, reflect.TypeFor[Example]())
	require.NotNil(t, s)
	assert.Equal(t, core.TypeObject, s.Type)

	assert.Contains(t, s.Required, "name")
	assert.Contains(t, s.Required, "age")
	assert.False(t, s.AdditionalProperties.(bool))
}

func TestDefaultsEveryPBWithMapValues(t *testing.T) {
	r := newRegistry()

	type Val struct {
		Label string `json:"label" default:"default-label"`
	}
	type MapContainer struct {
		Items map[string]Val `json:"items"`
	}

	defaults := schema.FindDefaults(r, reflect.TypeFor[MapContainer]())
	require.NotNil(t, defaults)

	container := MapContainer{
		Items: map[string]Val{
			"first":  {},
			"second": {Label: "custom"},
		},
	}
	v := reflect.ValueOf(&container).Elem()
	pb := core.NewPathBuffer(make([]byte, 0, 128), 0)

	defaults.EveryPB(pb, v, func(item reflect.Value, def any) {
	})
}

func TestFindDefaultsInMap(t *testing.T) {
	r := newRegistry()

	type MapItem struct {
		Priority int `json:"priority" default:"5"`
	}

	defaults := schema.FindDefaults(r, reflect.TypeFor[map[string]MapItem]())
	require.NotNil(t, defaults)
	assert.NotEmpty(t, defaults.Paths, "should discover defaults inside map value types")
}

func TestEnsureTypeWithRefSchema(t *testing.T) {
	r := newRegistry()

	type RefTarget struct {
		Name string `json:"name"`
	}
	refSchema := r.Schema(reflect.TypeFor[RefTarget](), true, "RefTarget")
	require.NotNil(t, refSchema)
	assert.NotEmpty(t, refSchema.Ref)

	result := schema.JsonTagValue(r, "TestRef", refSchema, `{"name":"hello"}`)
	require.NotNil(t, result)
	obj, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "hello", obj["name"])
}

func TestEnsureTypeWithRefNilResolution(t *testing.T) {
	r := newRegistry()

	s := &core.Schema{Ref: "#/components/schemas/NonExistent"}

	result := schema.JsonTagValue(r, "TestRefNil", s, `"anything"`)
	assert.Nil(t, result)
}

func TestEnsureTypeBooleanSuccess(t *testing.T) {
	r := newRegistry()

	s := &core.Schema{Type: core.TypeBoolean}
	s.PrecomputeMessages()

	result := schema.JsonTagValue(r, "BoolField", s, "true")
	assert.Equal(t, true, result)

	result = schema.JsonTagValue(r, "BoolField", s, "false")
	assert.Equal(t, false, result)
}

func TestDefaultsEveryMapTraversal(t *testing.T) {
	r := newRegistry()

	type Item struct {
		Color string `json:"color" default:"red"`
	}

	defaults := schema.FindDefaults(r, reflect.TypeFor[map[string]Item]())
	require.NotNil(t, defaults)

	items := map[string]Item{
		"first":  {},
		"second": {Color: "blue"},
	}

	v := reflect.ValueOf(&items).Elem()
	var visited int
	defaults.Every(v, func(item reflect.Value, def any) {
		visited++
	})
	assert.Equal(t, 2, visited, "should visit both map entries")
}

func TestDefaultsEveryPBIntMapKey(t *testing.T) {
	r := newRegistry()

	type Item struct {
		Count int `json:"count" default:"1"`
	}

	defaults := schema.FindDefaults(r, reflect.TypeFor[map[int]Item]())
	require.NotNil(t, defaults)

	items := map[int]Item{
		1: {},
		2: {Count: 5},
	}

	v := reflect.ValueOf(&items).Elem()
	pb := core.NewPathBuffer(make([]byte, 0, 128), 0)

	var paths []string
	defaults.EveryPB(pb, v, func(item reflect.Value, def any) {
		paths = append(paths, string(pb.Bytes()))
	})
	assert.NotEmpty(t, paths, "should have visited map entries with int keys")
}
