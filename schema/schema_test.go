package schema_test

import (
	"reflect"
	"testing"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRegistry() *schema.MapRegistry {
	return schema.NewMapRegistry(schema.DefaultSchemaNamer)
}


func TestSchemaBasicTypes(t *testing.T) {
	r := newRegistry()

	tests := []struct {
		name     string
		typ      reflect.Type
		expected string
	}{
		{"string", reflect.TypeFor[string](), core.TypeString},
		{"int", reflect.TypeFor[int](), core.TypeInteger},
		{"int32", reflect.TypeFor[int32](), core.TypeInteger},
		{"int64", reflect.TypeFor[int64](), core.TypeInteger},
		{"float32", reflect.TypeFor[float32](), core.TypeNumber},
		{"float64", reflect.TypeFor[float64](), core.TypeNumber},
		{"bool", reflect.TypeFor[bool](), core.TypeBoolean},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := r.Schema(tt.typ, false, "")
			require.NotNil(t, s)
			assert.Equal(t, tt.expected, s.Type)
		})
	}
}


func TestSchemaStructTags(t *testing.T) {
	r := newRegistry()

	type MyStruct struct {
		Name  string `json:"name" minLength:"1" maxLength:"50" doc:"The user name"`
		Age   int    `json:"age" minimum:"0" maximum:"150"`
		Email string `json:"email" format:"email"`
	}

	s := r.Schema(reflect.TypeFor[MyStruct](), false, "MyStruct")

	require.NotNil(t, s)
	assert.Equal(t, core.TypeObject, s.Type)
	assert.NotNil(t, s.Properties["name"])
	assert.NotNil(t, s.Properties["age"])
	assert.NotNil(t, s.Properties["email"])

	nameSchema := s.Properties["name"]
	assert.Equal(t, core.TypeString, nameSchema.Type)
	assert.NotNil(t, nameSchema.MinLength)
	assert.Equal(t, 1, *nameSchema.MinLength)
	assert.NotNil(t, nameSchema.MaxLength)
	assert.Equal(t, 50, *nameSchema.MaxLength)
	assert.Equal(t, "The user name", nameSchema.Description)

	ageSchema := s.Properties["age"]
	assert.Equal(t, core.TypeInteger, ageSchema.Type)
	assert.NotNil(t, ageSchema.Minimum)
	assert.InDelta(t, 0.0, *ageSchema.Minimum, 0.001)
	assert.NotNil(t, ageSchema.Maximum)
	assert.InDelta(t, 150.0, *ageSchema.Maximum, 0.001)

	emailSchema := s.Properties["email"]
	assert.Equal(t, "email", emailSchema.Format)
}


func TestSchemaValidationValid(t *testing.T) {
	r := newRegistry()

	type Item struct {
		Name string `json:"name" minLength:"1"`
		Age  int    `json:"age" minimum:"0"`
	}

	s := r.Schema(reflect.TypeFor[Item](), false, "Item")

	pb := core.NewPathBuffer([]byte(""), 0)
	res := &core.ValidateResult{}

	input := map[string]any{
		"name": "Alice",
		"age":  float64(25),
	}

	schema.Validate(r, s, pb, core.ModeWriteToServer, input, res)
	assert.Empty(t, res.Errors, "expected no validation errors for valid input")
}

func TestSchemaValidationInvalid(t *testing.T) {
	r := newRegistry()

	type Item struct {
		Name string `json:"name" minLength:"3"`
		Age  int    `json:"age" minimum:"18"`
	}

	s := r.Schema(reflect.TypeFor[Item](), false, "Item")

	pb := core.NewPathBuffer([]byte(""), 0)
	res := &core.ValidateResult{}

	input := map[string]any{
		"name": "ab",
		"age":  float64(10),
	}

	schema.Validate(r, s, pb, core.ModeWriteToServer, input, res)
	assert.NotEmpty(t, res.Errors, "expected validation errors for invalid input")
	assert.GreaterOrEqual(t, len(res.Errors), 2, "expected at least 2 errors (name minLength and age minimum)")
}

func TestSchemaValidationMissingRequired(t *testing.T) {
	r := newRegistry()

	type Item struct {
		Name string `json:"name"`
	}

	s := r.Schema(reflect.TypeFor[Item](), false, "Item")

	pb := core.NewPathBuffer([]byte(""), 0)
	res := &core.ValidateResult{}

	input := map[string]any{}

	schema.Validate(r, s, pb, core.ModeWriteToServer, input, res)
	assert.NotEmpty(t, res.Errors, "expected error for missing required field")
}


func TestRegistryReference(t *testing.T) {
	r := newRegistry()

	type Inner struct {
		Value string `json:"value"`
	}
	type Outer struct {
		Inner Inner `json:"inner"`
	}

	s := r.Schema(reflect.TypeFor[Outer](), false, "Outer")
	require.NotNil(t, s)
	assert.Equal(t, core.TypeObject, s.Type)

	innerProp := s.Properties["inner"]
	require.NotNil(t, innerProp)
	assert.Equal(t, "#/components/schemas/Inner", innerProp.Ref)

	innerSchema := r.SchemaFromRef("#/components/schemas/Inner")
	require.NotNil(t, innerSchema)
	assert.Equal(t, core.TypeObject, innerSchema.Type)
}


func TestSchemaEnum(t *testing.T) {
	r := newRegistry()

	type WithEnum struct {
		Color string `json:"color" enum:"red,green,blue"`
	}

	s := r.Schema(reflect.TypeFor[WithEnum](), false, "WithEnum")

	require.NotNil(t, s)
	colorProp := s.Properties["color"]
	require.NotNil(t, colorProp)
	assert.Equal(t, []any{"red", "green", "blue"}, colorProp.Enum)
}

func TestSchemaEnumValidation(t *testing.T) {
	r := newRegistry()

	s := &core.Schema{
		Type: core.TypeString,
		Enum: []any{"red", "green", "blue"},
	}
	s.PrecomputeMessages()

	pb := core.NewPathBuffer([]byte(""), 0)
	res := &core.ValidateResult{}

	schema.Validate(r, s, pb, core.ModeWriteToServer, "red", res)
	assert.Empty(t, res.Errors)

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, "yellow", res)
	assert.NotEmpty(t, res.Errors)
}


func TestSchemaNullable(t *testing.T) {
	r := newRegistry()

	s := r.Schema(reflect.TypeFor[*string](), false, "")
	require.NotNil(t, s)
	assert.True(t, s.Nullable, "pointer to string should be nullable")

	s = r.Schema(reflect.TypeFor[string](), false, "")
	require.NotNil(t, s)
	assert.False(t, s.Nullable, "plain string should not be nullable")
}

func TestSchemaNullableValidation(t *testing.T) {
	r := newRegistry()

	s := &core.Schema{
		Type:     core.TypeString,
		Nullable: true,
	}
	s.PrecomputeMessages()

	pb := core.NewPathBuffer([]byte(""), 0)
	res := &core.ValidateResult{}

	schema.Validate(r, s, pb, core.ModeWriteToServer, nil, res)
	assert.Empty(t, res.Errors, "nil should be valid for nullable schema")

	s2 := &core.Schema{
		Type:     core.TypeString,
		Nullable: false,
	}
	s2.PrecomputeMessages()

	res.Reset()
	pb.Reset()
	schema.Validate(r, s2, pb, core.ModeWriteToServer, nil, res)
	assert.NotEmpty(t, res.Errors, "nil should be invalid for non-nullable string schema")
}


func TestDefaultSchemaNamer(t *testing.T) {
	type MyType struct{}
	name := schema.DefaultSchemaNamer(reflect.TypeFor[MyType](), "fallback")
	assert.Equal(t, "MyType", name)

	name = schema.DefaultSchemaNamer(reflect.TypeFor[struct{}](), "SomeHint")
	assert.Equal(t, "SomeHint", name)
}


func TestSchemaArray(t *testing.T) {
	r := newRegistry()

	s := r.Schema(reflect.TypeFor[[]string](), false, "")
	require.NotNil(t, s)
	assert.Equal(t, core.TypeArray, s.Type)
	assert.NotNil(t, s.Items)
	assert.Equal(t, core.TypeString, s.Items.Type)
}


func TestSchemaMap(t *testing.T) {
	r := newRegistry()

	s := r.Schema(reflect.TypeFor[map[string]int](), false, "")
	require.NotNil(t, s)
	assert.Equal(t, core.TypeObject, s.Type)
	assert.NotNil(t, s.AdditionalProperties)
}


func TestModelValidator(t *testing.T) {
	type Person struct {
		Name string `json:"name" minLength:"2"`
		Age  int    `json:"age" minimum:"0"`
	}

	v := schema.NewModelValidator()

	errs := v.Validate(reflect.TypeFor[Person](), map[string]any{
		"name": "Jo",
		"age":  float64(5),
	})
	assert.Nil(t, errs)

	errs = v.Validate(reflect.TypeFor[Person](), map[string]any{
		"name": "J",
		"age":  float64(-1),
	})
	assert.NotNil(t, errs)
	assert.Len(t, errs, 2)
}
