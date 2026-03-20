package patch

import (
	"errors"
	"testing"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- RFC 7396 JSON Merge Patch ---

func TestMergePatchSetField(t *testing.T) {
	result, err := Apply(ContentTypeMergePatch,
		[]byte(`{"a":"b"}`),
		[]byte(`{"a":"c"}`),
	)
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":"c"}`, string(result))
}

func TestMergePatchAddField(t *testing.T) {
	result, err := Apply(ContentTypeMergePatch,
		[]byte(`{"a":"b"}`),
		[]byte(`{"b":"c"}`),
	)
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":"b","b":"c"}`, string(result))
}

func TestMergePatchDeleteField(t *testing.T) {
	// RFC 7396: null means delete.
	result, err := Apply(ContentTypeMergePatch,
		[]byte(`{"a":"b"}`),
		[]byte(`{"a":null}`),
	)
	require.NoError(t, err)
	assert.JSONEq(t, `{}`, string(result))
}

func TestMergePatchDeleteOneOfMany(t *testing.T) {
	result, err := Apply(ContentTypeMergePatch,
		[]byte(`{"a":"b","b":"c"}`),
		[]byte(`{"a":null}`),
	)
	require.NoError(t, err)
	assert.JSONEq(t, `{"b":"c"}`, string(result))
}

func TestMergePatchReplaceArrayWholesale(t *testing.T) {
	// RFC 7396: arrays are replaced, not merged element-by-element.
	result, err := Apply(ContentTypeMergePatch,
		[]byte(`{"a":["b"]}`),
		[]byte(`{"a":"c"}`),
	)
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":"c"}`, string(result))
}

func TestMergePatchSetArray(t *testing.T) {
	result, err := Apply(ContentTypeMergePatch,
		[]byte(`{"a":"c"}`),
		[]byte(`{"a":["b"]}`),
	)
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":["b"]}`, string(result))
}

func TestMergePatchNested(t *testing.T) {
	result, err := Apply(ContentTypeMergePatch,
		[]byte(`{"a":{"b":"c"}}`),
		[]byte(`{"a":{"b":"d","c":null}}`),
	)
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":{"b":"d"}}`, string(result))
}

func TestMergePatchReplaceArrayOfObjects(t *testing.T) {
	result, err := Apply(ContentTypeMergePatch,
		[]byte(`{"a":[{"b":"c"}]}`),
		[]byte(`{"a":[1]}`),
	)
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":[1]}`, string(result))
}

func TestMergePatchReplaceTopLevelArray(t *testing.T) {
	result, err := Apply(ContentTypeMergePatch,
		[]byte(`["a","b"]`),
		[]byte(`["c","d"]`),
	)
	require.NoError(t, err)
	assert.JSONEq(t, `["c","d"]`, string(result))
}

func TestMergePatchObjectToArray(t *testing.T) {
	result, err := Apply(ContentTypeMergePatch,
		[]byte(`{"a":"b"}`),
		[]byte(`["c"]`),
	)
	require.NoError(t, err)
	assert.JSONEq(t, `["c"]`, string(result))
}

func TestMergePatchExistingNullPreserved(t *testing.T) {
	// RFC 7396 Appendix A test case 13: existing null is preserved
	// when the patch does not mention that field.
	result, err := Apply(ContentTypeMergePatch,
		[]byte(`{"e":null}`),
		[]byte(`{"a":1}`),
	)
	require.NoError(t, err)
	assert.JSONEq(t, `{"e":null,"a":1}`, string(result))
}

func TestMergePatchArrayToObject(t *testing.T) {
	result, err := Apply(ContentTypeMergePatch,
		[]byte(`[1,2]`),
		[]byte(`{"a":"b","c":null}`),
	)
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":"b"}`, string(result))
}

func TestMergePatchNestedNullDelete(t *testing.T) {
	result, err := Apply(ContentTypeMergePatch,
		[]byte(`{}`),
		[]byte(`{"a":{"bb":{"ccc":null}}}`),
	)
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":{"bb":{}}}`, string(result))
}

func TestMergePatchInvalidJSON(t *testing.T) {
	_, err := Apply(ContentTypeMergePatch,
		[]byte(`{"a":"b"}`),
		[]byte(`{`),
	)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidPatch))
}

// --- RFC 6902 JSON Patch ---

func TestJSONPatchAdd(t *testing.T) {
	result, err := Apply(ContentTypeJSONPatch,
		[]byte(`{"a":"b"}`),
		[]byte(`[{"op":"add","path":"/c","value":"d"}]`),
	)
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":"b","c":"d"}`, string(result))
}

func TestJSONPatchRemove(t *testing.T) {
	result, err := Apply(ContentTypeJSONPatch,
		[]byte(`{"a":"b","c":"d"}`),
		[]byte(`[{"op":"remove","path":"/c"}]`),
	)
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":"b"}`, string(result))
}

func TestJSONPatchReplace(t *testing.T) {
	result, err := Apply(ContentTypeJSONPatch,
		[]byte(`{"a":"b"}`),
		[]byte(`[{"op":"replace","path":"/a","value":"c"}]`),
	)
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":"c"}`, string(result))
}

func TestJSONPatchMove(t *testing.T) {
	result, err := Apply(ContentTypeJSONPatch,
		[]byte(`{"a":"b","c":"d"}`),
		[]byte(`[{"op":"move","from":"/a","path":"/e"}]`),
	)
	require.NoError(t, err)
	assert.JSONEq(t, `{"c":"d","e":"b"}`, string(result))
}

func TestJSONPatchCopy(t *testing.T) {
	result, err := Apply(ContentTypeJSONPatch,
		[]byte(`{"a":"b"}`),
		[]byte(`[{"op":"copy","from":"/a","path":"/c"}]`),
	)
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":"b","c":"b"}`, string(result))
}

func TestJSONPatchTest(t *testing.T) {
	// Test operation succeeds: patch applies.
	result, err := Apply(ContentTypeJSONPatch,
		[]byte(`{"a":"b"}`),
		[]byte(`[{"op":"test","path":"/a","value":"b"},{"op":"replace","path":"/a","value":"c"}]`),
	)
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":"c"}`, string(result))
}

func TestJSONPatchTestFailsRollback(t *testing.T) {
	// Test operation fails: entire patch is rolled back (atomic).
	_, err := Apply(ContentTypeJSONPatch,
		[]byte(`{"a":"b"}`),
		[]byte(`[{"op":"test","path":"/a","value":"wrong"}]`),
	)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidPatch))
}

func TestJSONPatchSequentialOperations(t *testing.T) {
	// RFC 6902: operations are applied sequentially.
	result, err := Apply(ContentTypeJSONPatch,
		[]byte(`{"a":"b"}`),
		[]byte(`[{"op":"add","path":"/c","value":[]},{"op":"add","path":"/c/-","value":"x"}]`),
	)
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":"b","c":["x"]}`, string(result))
}

func TestJSONPatchArrayInsert(t *testing.T) {
	result, err := Apply(ContentTypeJSONPatch,
		[]byte(`{"a":[1,3]}`),
		[]byte(`[{"op":"add","path":"/a/1","value":2}]`),
	)
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":[1,2,3]}`, string(result))
}

func TestJSONPatchInvalidJSON(t *testing.T) {
	_, err := Apply(ContentTypeJSONPatch,
		[]byte(`{"a":"b"}`),
		[]byte(`[`),
	)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidPatch))
}

func TestJSONPatchInvalidOperation(t *testing.T) {
	_, err := Apply(ContentTypeJSONPatch,
		[]byte(`{"a":"b"}`),
		[]byte(`[{"op":"unsupported"}]`),
	)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidPatch))
}

func TestJSONPatchRemoveNonexistent(t *testing.T) {
	// RFC 6902: target location MUST exist for remove.
	_, err := Apply(ContentTypeJSONPatch,
		[]byte(`{"a":"b"}`),
		[]byte(`[{"op":"remove","path":"/nonexistent"}]`),
	)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidPatch))
}

// --- ApplyTo (generic) ---

type Thing struct {
	ID    string   `json:"id"`
	Price float64  `json:"price,omitempty"`
	Tags  []string `json:"tags,omitempty"`
}

func TestApplyToMergePatch(t *testing.T) {
	thing := &Thing{ID: "1", Price: 10.0, Tags: []string{"old"}}

	err := ApplyTo(ContentTypeMergePatch, thing, []byte(`{"price": 20.0}`))
	require.NoError(t, err)
	assert.Equal(t, 20.0, thing.Price)
	assert.Equal(t, "1", thing.ID)         // unchanged
	assert.Equal(t, []string{"old"}, thing.Tags) // unchanged
}

func TestApplyToJSONPatch(t *testing.T) {
	thing := &Thing{ID: "1", Price: 10.0}

	err := ApplyTo(ContentTypeJSONPatch, thing, []byte(`[{"op":"replace","path":"/price","value":99.9}]`))
	require.NoError(t, err)
	assert.InDelta(t, 99.9, thing.Price, 0.01)
	assert.Equal(t, "1", thing.ID) // unchanged
}

func TestApplyToDeleteField(t *testing.T) {
	thing := &Thing{ID: "1", Price: 10.0, Tags: []string{"a", "b"}}

	// RFC 7396: null deletes the field (sets to zero value after unmarshal).
	err := ApplyTo(ContentTypeMergePatch, thing, []byte(`{"tags": null}`))
	require.NoError(t, err)
	assert.Nil(t, thing.Tags)
	assert.Equal(t, 10.0, thing.Price) // unchanged
}

func TestApplyToUnsupportedContentType(t *testing.T) {
	thing := &Thing{ID: "1"}

	err := ApplyTo("application/json", thing, []byte(`{}`))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnsupportedContentType))
	assert.Equal(t, "1", thing.ID) // unchanged on error
}

func TestApplyToInvalidPatch(t *testing.T) {
	thing := &Thing{ID: "1", Price: 10.0}

	err := ApplyTo(ContentTypeMergePatch, thing, []byte(`{`))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidPatch))
	assert.Equal(t, 10.0, thing.Price) // unchanged on error
}

func TestApplyToNestedStruct(t *testing.T) {
	type Address struct {
		City string `json:"city"`
		Zip  string `json:"zip"`
	}
	type Person struct {
		Name    string  `json:"name"`
		Address Address `json:"address"`
	}

	p := &Person{Name: "Alice", Address: Address{City: "NYC", Zip: "10001"}}

	err := ApplyTo(ContentTypeMergePatch, p, []byte(`{"address":{"zip":"10002"}}`))
	require.NoError(t, err)
	assert.Equal(t, "Alice", p.Name)
	assert.Equal(t, "10002", p.Address.Zip)
}

// --- Content-Type dispatch ---

func TestUnsupportedContentType(t *testing.T) {
	_, err := Apply("application/json", []byte(`{}`), []byte(`{}`))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnsupportedContentType))
}

func TestEmptyContentType(t *testing.T) {
	_, err := Apply("", []byte(`{}`), []byte(`{}`))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnsupportedContentType))
}

// --- Equal ---

func TestEqualIdentical(t *testing.T) {
	assert.True(t, Equal(
		[]byte(`{"a":"b","c":1}`),
		[]byte(`{"c":1,"a":"b"}`),
	))
}

func TestEqualDifferent(t *testing.T) {
	assert.False(t, Equal(
		[]byte(`{"a":"b"}`),
		[]byte(`{"a":"c"}`),
	))
}

// --- MakeOptionalSchema ---

func TestMakeOptionalSchemaRemovesRequired(t *testing.T) {
	s := &core.Schema{
		Type: "object",
		Properties: map[string]*core.Schema{
			"id":   {Type: "string"},
			"name": {Type: "string"},
		},
		Required: []string{"id", "name"},
	}
	opt := MakeOptionalSchema(s)
	assert.Equal(t, "object", opt.Type)
	assert.Contains(t, opt.Properties, "id")
	assert.Contains(t, opt.Properties, "name")
	assert.Empty(t, opt.Required)
}

func TestMakeOptionalSchemaNestedRequired(t *testing.T) {
	s := &core.Schema{
		Type: "object",
		Properties: map[string]*core.Schema{
			"nested": {
				Type:     "object",
				Required: []string{"deep"},
				Properties: map[string]*core.Schema{
					"deep": {Type: "string"},
				},
			},
		},
		Required: []string{"nested"},
	}
	opt := MakeOptionalSchema(s)
	assert.Empty(t, opt.Required)
	assert.Empty(t, opt.Properties["nested"].Required)
}

func TestMakeOptionalSchemaComposite(t *testing.T) {
	s := &core.Schema{
		AnyOf: []*core.Schema{{Type: "string"}, {Type: "number"}},
		AllOf: []*core.Schema{{Type: "object"}},
		Not:   &core.Schema{Type: "null"},
	}
	opt := MakeOptionalSchema(s)
	assert.Len(t, opt.AnyOf, 2)
	assert.Len(t, opt.AllOf, 1)
	require.NotNil(t, opt.Not)
}

func TestMakeOptionalSchemaNil(t *testing.T) {
	assert.Nil(t, MakeOptionalSchema(nil))
}
