package schema_test

import (
	"encoding"
	"encoding/json"
	"net"
	"net/netip"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)


func TestFromTypeTime(t *testing.T) {
	r := newRegistry()
	s := r.Schema(reflect.TypeFor[time.Time](), false, "")
	require.NotNil(t, s)
	assert.Equal(t, core.TypeString, s.Type)
	assert.Equal(t, "date-time", s.Format)
}

func TestFromTypeURL(t *testing.T) {
	r := newRegistry()
	s := r.Schema(reflect.TypeFor[url.URL](), false, "")
	require.NotNil(t, s)
	assert.Equal(t, core.TypeString, s.Type)
	assert.Equal(t, "uri", s.Format)
}

func TestFromTypeIP(t *testing.T) {
	r := newRegistry()
	s := r.Schema(reflect.TypeFor[net.IP](), false, "")
	require.NotNil(t, s)
	assert.Equal(t, core.TypeString, s.Type)
	assert.Equal(t, "ipv4", s.Format)
}

func TestFromTypeIPAddr(t *testing.T) {
	r := newRegistry()
	s := r.Schema(reflect.TypeFor[netip.Addr](), false, "")
	require.NotNil(t, s)
	assert.Equal(t, core.TypeString, s.Type)
	assert.Equal(t, "ip", s.Format)
}

func TestFromTypeRawMessage(t *testing.T) {
	r := newRegistry()
	s := r.Schema(reflect.TypeFor[json.RawMessage](), false, "")
	require.NotNil(t, s)
	assert.Empty(t, s.Type)
}

func TestFromTypeByteSlice(t *testing.T) {
	r := newRegistry()
	s := r.Schema(reflect.TypeFor[[]byte](), false, "")
	require.NotNil(t, s)
	assert.Equal(t, core.TypeString, s.Type)
	assert.Equal(t, "base64", s.ContentEncoding)
}

func TestFromTypeUint8(t *testing.T) {
	r := newRegistry()
	s := r.Schema(reflect.TypeFor[uint8](), false, "")
	require.NotNil(t, s)
	assert.Equal(t, core.TypeInteger, s.Type)
	assert.NotNil(t, s.Minimum)
}

func TestFromTypeUint16(t *testing.T) {
	r := newRegistry()
	s := r.Schema(reflect.TypeFor[uint16](), false, "")
	require.NotNil(t, s)
	assert.Equal(t, core.TypeInteger, s.Type)
}

func TestFromTypeUint32(t *testing.T) {
	r := newRegistry()
	s := r.Schema(reflect.TypeFor[uint32](), false, "")
	require.NotNil(t, s)
	assert.Equal(t, core.TypeInteger, s.Type)
}

func TestFromTypeUint64(t *testing.T) {
	r := newRegistry()
	s := r.Schema(reflect.TypeFor[uint64](), false, "")
	require.NotNil(t, s)
	assert.Equal(t, core.TypeInteger, s.Type)
	assert.Equal(t, "int64", s.Format)
}

func TestFromTypeInt8(t *testing.T) {
	r := newRegistry()
	s := r.Schema(reflect.TypeFor[int8](), false, "")
	require.NotNil(t, s)
	assert.Equal(t, core.TypeInteger, s.Type)
	assert.Equal(t, "int32", s.Format)
}

func TestFromTypeInt16(t *testing.T) {
	r := newRegistry()
	s := r.Schema(reflect.TypeFor[int16](), false, "")
	require.NotNil(t, s)
	assert.Equal(t, core.TypeInteger, s.Type)
}

func TestFromTypeFloat32(t *testing.T) {
	r := newRegistry()
	s := r.Schema(reflect.TypeFor[float32](), false, "")
	require.NotNil(t, s)
	assert.Equal(t, core.TypeNumber, s.Type)
	assert.Equal(t, "float", s.Format)
}

func TestFromTypeInterface(t *testing.T) {
	r := newRegistry()
	s := r.Schema(reflect.TypeFor[any](), false, "")
	require.NotNil(t, s)
	assert.Empty(t, s.Type)
}

func TestFromTypePointerNullable(t *testing.T) {
	r := newRegistry()
	s := r.Schema(reflect.TypeFor[*int](), false, "")
	require.NotNil(t, s)
	assert.Equal(t, core.TypeInteger, s.Type)
	assert.True(t, s.Nullable)
}

func TestFromTypeArray(t *testing.T) {
	r := newRegistry()
	s := r.Schema(reflect.TypeFor[[3]string](), false, "")
	require.NotNil(t, s)
	assert.Equal(t, core.TypeArray, s.Type)
	require.NotNil(t, s.MinItems)
	assert.Equal(t, 3, *s.MinItems)
	require.NotNil(t, s.MaxItems)
	assert.Equal(t, 3, *s.MaxItems)
}

func TestFromTypeMap(t *testing.T) {
	r := newRegistry()
	s := r.Schema(reflect.TypeFor[map[string]string](), false, "")
	require.NotNil(t, s)
	assert.Equal(t, core.TypeObject, s.Type)
	assert.NotNil(t, s.AdditionalProperties)
}


type customTextType struct{}

func (c *customTextType) UnmarshalText([]byte) error { return nil }

var _ encoding.TextUnmarshaler = (*customTextType)(nil)

func TestFromTypeTextUnmarshaler(t *testing.T) {
	r := newRegistry()
	s := r.Schema(reflect.TypeFor[customTextType](), false, "")
	require.NotNil(t, s)
	assert.Equal(t, core.TypeString, s.Type)
}


func TestFromFieldDoc(t *testing.T) {
	r := newRegistry()

	type S struct {
		Name string `json:"name" doc:"The user name"`
	}
	f, _ := reflect.TypeFor[S]().FieldByName("Name")
	s := schema.FromField(r, f, "")
	require.NotNil(t, s)
	assert.Equal(t, "The user name", s.Description)
}

func TestFromFieldTimeFormatDate(t *testing.T) {
	r := newRegistry()

	type S struct {
		Birthday time.Time `json:"birthday" timeFormat:"2006-01-02"`
	}
	f, _ := reflect.TypeFor[S]().FieldByName("Birthday")
	s := schema.FromField(r, f, "")
	require.NotNil(t, s)
	assert.Equal(t, "date", s.Format)
}

func TestFromFieldTimeFormatTime(t *testing.T) {
	r := newRegistry()

	type S struct {
		StartTime time.Time `json:"start_time" timeFormat:"15:04:05"`
	}
	f, _ := reflect.TypeFor[S]().FieldByName("StartTime")
	s := schema.FromField(r, f, "")
	require.NotNil(t, s)
	assert.Equal(t, "time", s.Format)
}

func TestFromFieldHeader(t *testing.T) {
	r := newRegistry()

	type S struct {
		Modified time.Time `header:"Last-Modified"`
	}
	f, _ := reflect.TypeFor[S]().FieldByName("Modified")
	s := schema.FromField(r, f, "")
	require.NotNil(t, s)
	assert.Equal(t, "date-time-http", s.Format)
}

func TestFromFieldDefault(t *testing.T) {
	r := newRegistry()

	type S struct {
		Color string `json:"color" default:"red"`
	}
	f, _ := reflect.TypeFor[S]().FieldByName("Color")
	s := schema.FromField(r, f, "")
	require.NotNil(t, s)
	assert.Equal(t, "red", s.Default)
}

func TestFromFieldExample(t *testing.T) {
	r := newRegistry()

	type S struct {
		Name string `json:"name" example:"Alice"`
	}
	f, _ := reflect.TypeFor[S]().FieldByName("Name")
	s := schema.FromField(r, f, "")
	require.NotNil(t, s)
	assert.Len(t, s.Examples, 1)
	assert.Equal(t, "Alice", s.Examples[0])
}

func TestFromFieldReadOnly(t *testing.T) {
	r := newRegistry()

	type S struct {
		ID string `json:"id" readOnly:"true"`
	}
	f, _ := reflect.TypeFor[S]().FieldByName("ID")
	s := schema.FromField(r, f, "")
	require.NotNil(t, s)
	assert.True(t, s.ReadOnly)
}

func TestFromFieldWriteOnly(t *testing.T) {
	r := newRegistry()

	type S struct {
		Password string `json:"password" writeOnly:"true"`
	}
	f, _ := reflect.TypeFor[S]().FieldByName("Password")
	s := schema.FromField(r, f, "")
	require.NotNil(t, s)
	assert.True(t, s.WriteOnly)
}

func TestFromFieldDeprecated(t *testing.T) {
	r := newRegistry()

	type S struct {
		Old string `json:"old" deprecated:"true"`
	}
	f, _ := reflect.TypeFor[S]().FieldByName("Old")
	s := schema.FromField(r, f, "")
	require.NotNil(t, s)
	assert.True(t, s.Deprecated)
}

func TestFromFieldPattern(t *testing.T) {
	r := newRegistry()

	type S struct {
		Code string `json:"code" pattern:"^[A-Z]{3}$" patternDescription:"three uppercase letters"`
	}
	f, _ := reflect.TypeFor[S]().FieldByName("Code")
	s := schema.FromField(r, f, "")
	require.NotNil(t, s)
	assert.Equal(t, "^[A-Z]{3}$", s.Pattern)
	assert.Equal(t, "three uppercase letters", s.PatternDescription)
}

func TestFromFieldEncoding(t *testing.T) {
	r := newRegistry()

	type S struct {
		Data string `json:"data" encoding:"base64"`
	}
	f, _ := reflect.TypeFor[S]().FieldByName("Data")
	s := schema.FromField(r, f, "")
	require.NotNil(t, s)
	assert.Equal(t, "base64", s.ContentEncoding)
}

func TestFromFieldMinMax(t *testing.T) {
	r := newRegistry()

	type S struct {
		Score float64 `json:"score" minimum:"0" maximum:"100" exclusiveMinimum:"-1" exclusiveMaximum:"101" multipleOf:"0.5"`
	}
	f, _ := reflect.TypeFor[S]().FieldByName("Score")
	s := schema.FromField(r, f, "")
	require.NotNil(t, s)
	assert.NotNil(t, s.Minimum)
	assert.NotNil(t, s.Maximum)
	assert.NotNil(t, s.ExclusiveMinimum)
	assert.NotNil(t, s.ExclusiveMaximum)
	assert.NotNil(t, s.MultipleOf)
}

func TestFromFieldMinMaxItems(t *testing.T) {
	r := newRegistry()

	type S struct {
		Tags []string `json:"tags" minItems:"1" maxItems:"10" uniqueItems:"true"`
	}
	f, _ := reflect.TypeFor[S]().FieldByName("Tags")
	s := schema.FromField(r, f, "")
	require.NotNil(t, s)
	assert.NotNil(t, s.MinItems)
	assert.NotNil(t, s.MaxItems)
	assert.True(t, s.UniqueItems)
}

func TestFromFieldMinMaxProperties(t *testing.T) {
	r := newRegistry()

	type S struct {
		Meta map[string]string `json:"meta" minProperties:"1" maxProperties:"5"`
	}
	f, _ := reflect.TypeFor[S]().FieldByName("Meta")
	s := schema.FromField(r, f, "")
	require.NotNil(t, s)
	assert.NotNil(t, s.MinProperties)
	assert.NotNil(t, s.MaxProperties)
}

func TestFromFieldHidden(t *testing.T) {
	r := newRegistry()

	type S struct {
		Secret string `json:"secret" hidden:"true"`
	}
	f, _ := reflect.TypeFor[S]().FieldByName("Secret")
	s := schema.FromField(r, f, "")
	require.NotNil(t, s)
	assert.True(t, s.Hidden)
}


func TestValidateFormatDateTime(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString, Format: "date-time"}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, "2024-01-15T10:30:00Z", res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, "not-a-date", res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateFormatDate(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString, Format: "date"}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, "2024-01-15", res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, "nope", res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateFormatTime(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString, Format: "time"}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, "10:30:00", res)
	assert.Empty(t, res.Errors)
}

func TestValidateFormatEmail(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString, Format: "email"}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, "user@example.com", res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, "not-email", res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateFormatHostname(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString, Format: "hostname"}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, "example.com", res)
	assert.Empty(t, res.Errors)
}

func TestValidateFormatIPv4(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString, Format: "ipv4"}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, "192.168.1.1", res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, "not-ip", res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateFormatIPv6(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString, Format: "ipv6"}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, "::1", res)
	assert.Empty(t, res.Errors)
}

func TestValidateFormatURI(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString, Format: "uri"}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, "https://example.com/path", res)
	assert.Empty(t, res.Errors)
}

func TestValidateFormatUUID(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString, Format: "uuid"}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, "550e8400-e29b-41d4-a716-446655440000", res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, "not-a-uuid", res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateFormatRegex(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString, Format: "regex"}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, "^[a-z]+$", res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, "[invalid(", res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateFormatJSONPointer(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString, Format: "json-pointer"}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, "/foo/bar/0", res)
	assert.Empty(t, res.Errors)
}

func TestValidateFormatDuration(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString, Format: "duration"}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, "5s", res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, "not-duration", res)
	assert.NotEmpty(t, res.Errors)
}

func TestValidateFormatBase64(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{Type: core.TypeString, ContentEncoding: "base64"}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, "SGVsbG8=", res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, "not valid base64!!!", res)
	assert.NotEmpty(t, res.Errors)
}


func TestValidateOneOf(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{
		OneOf: []*core.Schema{
			{Type: core.TypeString},
			{Type: core.TypeInteger},
		},
	}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, "hello", res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, true, res)
	assert.NotEmpty(t, res.Errors) // bool doesn't match either
}

func TestValidateAnyOf(t *testing.T) {
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
	schema.Validate(r, s, pb, core.ModeWriteToServer, "hello", res)
	assert.Empty(t, res.Errors)
}

func TestValidateAllOf(t *testing.T) {
	r := newRegistry()
	lo := 0.0
	hi := 100.0
	s := &core.Schema{
		AllOf: []*core.Schema{
			{Type: core.TypeNumber, Minimum: &lo},
			{Type: core.TypeNumber, Maximum: &hi},
		},
	}
	s.PrecomputeMessages()
	for _, sub := range s.AllOf {
		sub.PrecomputeMessages()
	}
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, float64(50), res)
	assert.Empty(t, res.Errors)
}

func TestValidateNot(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{
		Not: &core.Schema{Type: core.TypeString},
	}
	s.PrecomputeMessages()
	s.Not.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, float64(42), res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, "hello", res)
	assert.NotEmpty(t, res.Errors)
}


func TestMapRegistryTypeAlias(t *testing.T) {
	r := newRegistry()
	r.RegisterTypeAlias(reflect.TypeFor[int64](), reflect.TypeFor[int32]())

	s := r.Schema(reflect.TypeFor[int64](), false, "")
	require.NotNil(t, s)
	assert.Equal(t, core.TypeInteger, s.Type)
	assert.Equal(t, "int32", s.Format) // Should use int32's format
}

func TestMapRegistrySchemaFromRefNotFound(t *testing.T) {
	r := newRegistry()
	s := r.SchemaFromRef("#/components/schemas/NonExistent")
	assert.Nil(t, s)
}

func TestMapRegistrySchemaFromRefWrongPrefix(t *testing.T) {
	r := newRegistry()
	s := r.SchemaFromRef("#/wrong/prefix/Foo")
	assert.Nil(t, s)
}

func TestMapRegistryMarshalJSON(t *testing.T) {
	r := newRegistry()
	r.Schema(reflect.TypeFor[struct{ X int }](), false, "TestStruct")
	b, err := r.MarshalJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, b)
}

func TestMapRegistryWithConfig(t *testing.T) {
	config := schema.RegistryConfig{
		AllowAdditionalPropertiesByDefault: true,
		FieldsOptionalByDefault:            true,
	}
	r := schema.NewMapRegistryWithConfig(schema.DefaultSchemaNamer, config)
	assert.Equal(t, config, r.RegistryConfig())

	type S struct {
		Name string `json:"name"`
	}
	s := r.Schema(reflect.TypeFor[S](), false, "S")
	require.NotNil(t, s)
	assert.True(t, s.AdditionalProperties.(bool))
	assert.Empty(t, s.Required) // Fields optional by default
}


func TestFindAndApplyDefaults(t *testing.T) {
	r := newRegistry()

	type Input struct {
		Limit  int    `json:"limit" default:"10"`
		Format string `json:"format" default:"json"`
	}

	defaults := schema.FindDefaults(r, reflect.TypeFor[Input]())
	require.NotNil(t, defaults)
	assert.Len(t, defaults.Paths, 2)

	input := Input{}
	v := reflect.ValueOf(&input).Elem()
	schema.ApplyDefaults(defaults, v)

	assert.Equal(t, 10, input.Limit)
	assert.Equal(t, "json", input.Format)
}

func TestApplyDefaultsDoesNotOverwrite(t *testing.T) {
	r := newRegistry()

	type Input struct {
		Limit int `json:"limit" default:"10"`
	}

	defaults := schema.FindDefaults(r, reflect.TypeFor[Input]())

	input := Input{Limit: 50}
	v := reflect.ValueOf(&input).Elem()
	schema.ApplyDefaults(defaults, v)

	assert.Equal(t, 50, input.Limit) // Should not overwrite
}


func TestValidateMultipleOf(t *testing.T) {
	r := newRegistry()
	mult := 3.0
	s := &core.Schema{Type: core.TypeNumber, MultipleOf: &mult}
	s.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, float64(9), res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, float64(10), res)
	assert.NotEmpty(t, res.Errors)
}


func TestValidateUniqueItems(t *testing.T) {
	r := newRegistry()
	s := &core.Schema{
		Type:        core.TypeArray,
		UniqueItems: true,
		Items:       &core.Schema{Type: core.TypeString},
	}
	s.PrecomputeMessages()
	s.Items.PrecomputeMessages()
	pb := core.NewPathBuffer([]byte{}, 0)

	res := &core.ValidateResult{}
	schema.Validate(r, s, pb, core.ModeWriteToServer, []any{"a", "b", "c"}, res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, []any{"a", "b", "a"}, res)
	assert.NotEmpty(t, res.Errors)
}


func TestSchemaOmitemptyOptional(t *testing.T) {
	r := newRegistry()

	type S struct {
		Required string `json:"required"`
		Optional string `json:"optional,omitempty"`
		Ignored  string `json:"-"`
	}

	s := r.Schema(reflect.TypeFor[S](), false, "S")
	require.NotNil(t, s)
	assert.Contains(t, s.Required, "required")
	assert.NotContains(t, s.Required, "optional")
	assert.NotContains(t, s.PropertyNames, "-")
	assert.NotContains(t, s.Properties, "-")
}


func TestSchemaDependentRequired(t *testing.T) {
	r := newRegistry()

	type S struct {
		A string `json:"a" dependentRequired:"b"`
		B string `json:"b"`
	}

	s := r.Schema(reflect.TypeFor[S](), false, "S")
	require.NotNil(t, s)
	assert.NotNil(t, s.DependentRequired)
	assert.Equal(t, []string{"b"}, s.DependentRequired["a"])
}


func TestConvertType(t *testing.T) {
	result := schema.ConvertType("test", reflect.TypeFor[int](), float64(42))
	assert.Equal(t, 42, result)
}

func TestConvertTypeNil(t *testing.T) {
	result := schema.ConvertType("test", reflect.TypeFor[string](), nil)
	assert.Nil(t, result)
}


func TestDefaultSchemaNamerGeneric(t *testing.T) {
	// Unnamed type with brackets in hint (simulating generics)
	name := schema.DefaultSchemaNamer(reflect.TypeFor[struct{}](), "MyType[SubType]")
	assert.Equal(t, "MyTypeSubType", name)
}

func TestDefaultSchemaNamerSlice(t *testing.T) {
	name := schema.DefaultSchemaNamer(reflect.TypeFor[struct{}](), "[]int")
	assert.Contains(t, name, "List")
	assert.Contains(t, name, "Int")
}


func TestApplyDefaultsPointerField(t *testing.T) {
	r := newRegistry()

	type Input struct {
		Color *string `json:"color" default:"blue"`
	}

	defaults := schema.FindDefaults(r, reflect.TypeFor[Input]())

	input := Input{}
	v := reflect.ValueOf(&input).Elem()
	schema.ApplyDefaults(defaults, v)

	require.NotNil(t, input.Color)
	assert.Equal(t, "blue", *input.Color)
}


func TestNewModelValidator(t *testing.T) {
	validator := schema.NewModelValidator()
	require.NotNil(t, validator)

	type Example struct {
		Name string `json:"name" maxLength:"5"`
		Age  int    `json:"age" minimum:"0"`
	}

	var validInput any
	require.NoError(t, json.Unmarshal([]byte(`{"name":"Alice","age":30}`), &validInput))
	errs := validator.Validate(reflect.TypeOf(Example{}), validInput)
	assert.Nil(t, errs)

	var invalidInput any
	require.NoError(t, json.Unmarshal([]byte(`{"name":"TooLongName","age":30}`), &invalidInput))
	errs = validator.Validate(reflect.TypeOf(Example{}), invalidInput)
	assert.NotNil(t, errs)
}

func TestNewModelValidatorReuse(t *testing.T) {
	validator := schema.NewModelValidator()

	type S struct {
		Value int `json:"value" minimum:"10"`
	}

	var v1 any
	require.NoError(t, json.Unmarshal([]byte(`{"value":20}`), &v1))
	assert.Nil(t, validator.Validate(reflect.TypeOf(S{}), v1))

	// Second validation (invalid) to ensure Reset() works.
	var v2 any
	require.NoError(t, json.Unmarshal([]byte(`{"value":5}`), &v2))
	errs := validator.Validate(reflect.TypeOf(S{}), v2)
	assert.NotNil(t, errs)
}
