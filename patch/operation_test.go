package patch

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Parse: RFC 7396 Merge Patch ---

func TestParseMergePatchSetField(t *testing.T) {
	ops, err := Parse(ContentTypeMergePatch, []byte(`{"price": 1.23}`))
	require.NoError(t, err)
	require.Len(t, ops, 1)
	assert.Equal(t, OpReplace, ops[0].Op)
	assert.Equal(t, []string{"price"}, ops[0].Path)
	assert.InDelta(t, 1.23, ops[0].Value, 0.001)
}

func TestParseMergePatchDeleteField(t *testing.T) {
	ops, err := Parse(ContentTypeMergePatch, []byte(`{"name": null}`))
	require.NoError(t, err)
	require.Len(t, ops, 1)
	assert.Equal(t, OpRemove, ops[0].Op)
	assert.Equal(t, []string{"name"}, ops[0].Path)
	assert.Nil(t, ops[0].Value)
}

func TestParseMergePatchNested(t *testing.T) {
	ops, err := Parse(ContentTypeMergePatch, []byte(`{"address": {"zip": "10002"}}`))
	require.NoError(t, err)
	require.Len(t, ops, 1)
	assert.Equal(t, OpReplace, ops[0].Op)
	assert.Equal(t, []string{"address", "zip"}, ops[0].Path)
	assert.Equal(t, "10002", ops[0].Value)
}

func TestParseMergePatchNestedDelete(t *testing.T) {
	ops, err := Parse(ContentTypeMergePatch, []byte(`{"a": {"b": "d", "c": null}}`))
	require.NoError(t, err)
	require.Len(t, ops, 2)

	opMap := make(map[string]Operation)
	for _, op := range ops {
		opMap[op.Path[len(op.Path)-1]] = op
	}

	assert.Equal(t, OpReplace, opMap["b"].Op)
	assert.Equal(t, []string{"a", "b"}, opMap["b"].Path)
	assert.Equal(t, "d", opMap["b"].Value)

	assert.Equal(t, OpRemove, opMap["c"].Op)
	assert.Equal(t, []string{"a", "c"}, opMap["c"].Path)
}

func TestParseMergePatchMultipleFields(t *testing.T) {
	ops, err := Parse(ContentTypeMergePatch, []byte(`{"price": 9.99, "name": "new", "old": null}`))
	require.NoError(t, err)
	require.Len(t, ops, 3)

	opMap := make(map[string]Operation)
	for _, op := range ops {
		opMap[op.Path[0]] = op
	}

	assert.Equal(t, OpReplace, opMap["price"].Op)
	assert.Equal(t, OpReplace, opMap["name"].Op)
	assert.Equal(t, OpRemove, opMap["old"].Op)
}

func TestParseMergePatchArrayValue(t *testing.T) {
	// RFC 7396: arrays are leaf values, replaced wholesale.
	ops, err := Parse(ContentTypeMergePatch, []byte(`{"tags": ["a", "b"]}`))
	require.NoError(t, err)
	require.Len(t, ops, 1)
	assert.Equal(t, OpReplace, ops[0].Op)
	assert.Equal(t, []string{"tags"}, ops[0].Path)
	assert.Equal(t, []any{"a", "b"}, ops[0].Value)
}

func TestParseMergePatchNonObjectReplaces(t *testing.T) {
	// Non-object patch replaces entire document.
	ops, err := Parse(ContentTypeMergePatch, []byte(`"just a string"`))
	require.NoError(t, err)
	require.Len(t, ops, 1)
	assert.Equal(t, OpReplace, ops[0].Op)
	assert.Nil(t, ops[0].Path)
	assert.Equal(t, "just a string", ops[0].Value)
}

func TestParseMergePatchInvalidJSON(t *testing.T) {
	_, err := Parse(ContentTypeMergePatch, []byte(`{`))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidPatch))
}

// --- Parse: RFC 6902 JSON Patch ---

func TestParseJSONPatchAdd(t *testing.T) {
	ops, err := Parse(ContentTypeJSONPatch, []byte(`[{"op":"add","path":"/c","value":"d"}]`))
	require.NoError(t, err)
	require.Len(t, ops, 1)
	assert.Equal(t, OpAdd, ops[0].Op)
	assert.Equal(t, []string{"c"}, ops[0].Path)
	assert.Equal(t, "d", ops[0].Value)
}

func TestParseJSONPatchRemove(t *testing.T) {
	ops, err := Parse(ContentTypeJSONPatch, []byte(`[{"op":"remove","path":"/a"}]`))
	require.NoError(t, err)
	require.Len(t, ops, 1)
	assert.Equal(t, OpRemove, ops[0].Op)
	assert.Equal(t, []string{"a"}, ops[0].Path)
}

func TestParseJSONPatchReplace(t *testing.T) {
	ops, err := Parse(ContentTypeJSONPatch, []byte(`[{"op":"replace","path":"/a","value":"c"}]`))
	require.NoError(t, err)
	require.Len(t, ops, 1)
	assert.Equal(t, OpReplace, ops[0].Op)
	assert.Equal(t, []string{"a"}, ops[0].Path)
	assert.Equal(t, "c", ops[0].Value)
}

func TestParseJSONPatchMove(t *testing.T) {
	ops, err := Parse(ContentTypeJSONPatch, []byte(`[{"op":"move","from":"/a","path":"/b"}]`))
	require.NoError(t, err)
	require.Len(t, ops, 1)
	assert.Equal(t, OpMove, ops[0].Op)
	assert.Equal(t, []string{"b"}, ops[0].Path)
	assert.Equal(t, []string{"a"}, ops[0].From)
}

func TestParseJSONPatchCopy(t *testing.T) {
	ops, err := Parse(ContentTypeJSONPatch, []byte(`[{"op":"copy","from":"/a","path":"/b"}]`))
	require.NoError(t, err)
	require.Len(t, ops, 1)
	assert.Equal(t, OpCopy, ops[0].Op)
	assert.Equal(t, []string{"b"}, ops[0].Path)
	assert.Equal(t, []string{"a"}, ops[0].From)
}

func TestParseJSONPatchTest(t *testing.T) {
	ops, err := Parse(ContentTypeJSONPatch, []byte(`[{"op":"test","path":"/a","value":"b"}]`))
	require.NoError(t, err)
	require.Len(t, ops, 1)
	assert.Equal(t, OpTest, ops[0].Op)
	assert.Equal(t, []string{"a"}, ops[0].Path)
	assert.Equal(t, "b", ops[0].Value)
}

func TestParseJSONPatchMultiple(t *testing.T) {
	ops, err := Parse(ContentTypeJSONPatch, []byte(`[
		{"op":"replace","path":"/name","value":"new"},
		{"op":"remove","path":"/old"},
		{"op":"add","path":"/tags/-","value":"x"}
	]`))
	require.NoError(t, err)
	require.Len(t, ops, 3)
	assert.Equal(t, OpReplace, ops[0].Op)
	assert.Equal(t, OpRemove, ops[1].Op)
	assert.Equal(t, OpAdd, ops[2].Op)
}

func TestParseJSONPatchNestedPath(t *testing.T) {
	ops, err := Parse(ContentTypeJSONPatch, []byte(`[{"op":"replace","path":"/a/b/c","value":1}]`))
	require.NoError(t, err)
	require.Len(t, ops, 1)
	assert.Equal(t, []string{"a", "b", "c"}, ops[0].Path)
}

func TestParseJSONPatchInvalidJSON(t *testing.T) {
	_, err := Parse(ContentTypeJSONPatch, []byte(`[`))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidPatch))
}

// --- Parse: Content-Type dispatch ---

func TestParseUnsupportedContentType(t *testing.T) {
	_, err := Parse("application/json", []byte(`{}`))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnsupportedContentType))
}

// --- JSON Pointer (RFC 6901) ---

func TestParseJSONPointerSimple(t *testing.T) {
	assert.Equal(t, []string{"a", "b", "c"}, parseJSONPointer("/a/b/c"))
}

func TestParseJSONPointerRoot(t *testing.T) {
	assert.Nil(t, parseJSONPointer(""))
}

func TestParseJSONPointerEscapedSlash(t *testing.T) {
	// RFC 6901: ~1 → /
	assert.Equal(t, []string{"a/b"}, parseJSONPointer("/a~1b"))
}

func TestParseJSONPointerEscapedTilde(t *testing.T) {
	// RFC 6901: ~0 → ~
	assert.Equal(t, []string{"a~b"}, parseJSONPointer("/a~0b"))
}

func TestParseJSONPointerBothEscapes(t *testing.T) {
	// ~01 → ~1 (not /), because ~0 is decoded to ~ first, then ~1 stays as-is.
	// Wait, the order is: replace ~1 first, then ~0.
	// "/a~01b" → split → "a~01b" → replace ~1→/ → "a~0/b" → replace ~0→~ → "a~/b"
	// Hmm actually let me re-check. RFC 6901 says: ~0 represents ~, ~1 represents /.
	// Decoding: first replace ~1 with /, then replace ~0 with ~.
	// So "~01" → after ~1→/: "~0/" wait no, "~01" has no "~1" substring...
	// "~01" → check for ~1: the "~0" part doesn't contain ~1, the "1" is just "1".
	// Actually "~01" as a string: characters are ~, 0, 1.
	// Looking for "~1": not found (we have "~0" then "1").
	// Looking for "~0": found at pos 0 → replace with "~" → "~1".
	// Wait, that's wrong. We replace ~1 first.
	// "~01": looking for "~1" → substring "~1" found at pos 1? No.
	// chars: [~][0][1] → "~0" starts at 0, "01" starts at 1. "~1" is at... not present.
	// So after ~1 pass: "~01" unchanged. After ~0 pass: "~1" → but wait "~1" is the literal string now.
	// Hmm. Actually the result should be: ~01 → ~ followed by 1 → "~1" as literal.
	// But our implementation does strings.ReplaceAll for ~1 first then ~0. Let's verify.
	assert.Equal(t, []string{"~1"}, parseJSONPointer("/~01"))
}

func TestParseJSONPointerArrayIndex(t *testing.T) {
	assert.Equal(t, []string{"tags", "0"}, parseJSONPointer("/tags/0"))
}

func TestParseJSONPointerAppend(t *testing.T) {
	// "-" references past-the-end for appending.
	assert.Equal(t, []string{"tags", "-"}, parseJSONPointer("/tags/-"))
}
