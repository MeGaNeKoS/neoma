package negotiate_test

import (
	"bytes"
	"testing"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/negotiate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)


func defaultNegotiator() *negotiate.Negotiator {
	return negotiate.NewNegotiator(negotiate.DefaultFormats(), "application/json", false)
}

func noFallbackNegotiator() *negotiate.Negotiator {
	return negotiate.NewNegotiator(negotiate.DefaultFormats(), "application/json", true)
}


func TestNewNegotiatorDefault(t *testing.T) {
	n := defaultNegotiator()
	require.NotNil(t, n)
}

func TestNewNegotiatorNoDefault(t *testing.T) {
	n := negotiate.NewNegotiator(negotiate.DefaultFormats(), "", false)
	require.NotNil(t, n)

	ct, err := n.Negotiate("")
	require.NoError(t, err)
	assert.NotEmpty(t, ct)
}


func TestNegotiateEmptyAccept(t *testing.T) {
	n := defaultNegotiator()
	ct, err := n.Negotiate("")
	require.NoError(t, err)
	assert.Equal(t, "application/json", ct)
}

func TestNegotiateExactMatch(t *testing.T) {
	n := defaultNegotiator()
	ct, err := n.Negotiate("application/json")
	require.NoError(t, err)
	assert.Equal(t, "application/json", ct)
}

func TestNegotiateWildcard(t *testing.T) {
	n := defaultNegotiator()
	ct, err := n.Negotiate("*/*")
	require.NoError(t, err)
	assert.Equal(t, "application/json", ct)
}

func TestNegotiateNoMatchFallback(t *testing.T) {
	n := defaultNegotiator()
	ct, err := n.Negotiate("text/xml")
	require.NoError(t, err)
	assert.Equal(t, "application/json", ct)
}

func TestNegotiateNoMatchNoFallback(t *testing.T) {
	n := noFallbackNegotiator()
	_, err := n.Negotiate("text/xml")
	assert.ErrorIs(t, err, core.ErrUnknownAcceptContentType)
}


func TestMarshalJSON(t *testing.T) {
	n := defaultNegotiator()
	var buf bytes.Buffer
	err := n.Marshal(&buf, "application/json", map[string]string{"hello": "world"})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), `"hello"`)
	assert.Contains(t, buf.String(), `"world"`)
}

func TestMarshalUnknownType(t *testing.T) {
	n := defaultNegotiator()
	var buf bytes.Buffer
	err := n.Marshal(&buf, "text/xml", nil)
	assert.ErrorIs(t, err, core.ErrUnknownContentType)
}

func TestMarshalSuffixFallback(t *testing.T) {
	n := defaultNegotiator()
	var buf bytes.Buffer
	err := n.Marshal(&buf, "application/problem+json", map[string]int{"code": 1})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), `"code"`)
}


func TestUnmarshalJSON(t *testing.T) {
	n := defaultNegotiator()
	var dest map[string]string
	err := n.Unmarshal("application/json", []byte(`{"key":"value"}`), &dest)
	require.NoError(t, err)
	assert.Equal(t, "value", dest["key"])
}

func TestUnmarshalEmptyContentType(t *testing.T) {
	n := defaultNegotiator()
	var dest map[string]any
	err := n.Unmarshal("", []byte(`{"a":1}`), &dest)
	require.NoError(t, err)
	assert.InDelta(t, float64(1), dest["a"], 0.01)
}

func TestUnmarshalWithCharset(t *testing.T) {
	n := defaultNegotiator()
	var dest map[string]any
	err := n.Unmarshal("application/json; charset=utf-8", []byte(`{"b":2}`), &dest)
	require.NoError(t, err)
	assert.InDelta(t, float64(2), dest["b"], 0.01)
}

func TestUnmarshalSuffixFallback(t *testing.T) {
	n := defaultNegotiator()
	var dest map[string]any
	err := n.Unmarshal("application/problem+json", []byte(`{"c":3}`), &dest)
	require.NoError(t, err)
	assert.InDelta(t, float64(3), dest["c"], 0.01)
}

func TestUnmarshalUnknownType(t *testing.T) {
	n := defaultNegotiator()
	var dest any
	err := n.Unmarshal("text/xml", []byte("<x/>"), &dest)
	assert.ErrorIs(t, err, core.ErrUnknownContentType)
}


func TestDefaultFormatsContainsJSON(t *testing.T) {
	fmts := negotiate.DefaultFormats()
	_, ok := fmts["application/json"]
	assert.True(t, ok, "should contain application/json")
	_, ok = fmts["json"]
	assert.True(t, ok, "should contain shorthand json key")
}

func TestDefaultJSONFormatRoundTrip(t *testing.T) {
	f := negotiate.DefaultJSONFormat()

	var buf bytes.Buffer
	require.NoError(t, f.Marshal(&buf, map[string]int{"n": 42}))

	var out map[string]int
	require.NoError(t, f.Unmarshal(buf.Bytes(), &out))
	assert.Equal(t, 42, out["n"])
}
