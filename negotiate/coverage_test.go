package negotiate_test

import (
	"bytes"
	"testing"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/negotiate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalMalformedContentType(t *testing.T) {
	n := defaultNegotiator()
	var buf bytes.Buffer
	err := n.Marshal(&buf, "application/json;+xml", nil)
	assert.ErrorIs(t, err, core.ErrUnknownContentType)
}

func TestUnmarshalMalformedContentType(t *testing.T) {
	n := defaultNegotiator()
	var dest any
	err := n.Unmarshal("application/json;+xml", []byte("{}"), &dest)
	assert.ErrorIs(t, err, core.ErrUnknownContentType)
}

func TestNegotiatePreferJSON(t *testing.T) {
	n := defaultNegotiator()
	ct, err := n.Negotiate("application/json, text/html")
	require.NoError(t, err)
	assert.Equal(t, "application/json", ct)
}

func TestNegotiateQValuePreference(t *testing.T) {
	n := defaultNegotiator()
	ct, err := n.Negotiate("application/json;q=0.9, application/cbor;q=1.0")
	require.NoError(t, err)
	assert.Equal(t, "application/json", ct)
}

func TestNegotiateEmptyFormats(t *testing.T) {
	n := negotiate.NewNegotiator(map[string]core.Format{}, "", false)
	ct, err := n.Negotiate("")
	assert.Error(t, err)
	_ = ct
}

func TestMarshalCBORNotRegistered(t *testing.T) {
	n := defaultNegotiator()
	var buf bytes.Buffer
	err := n.Marshal(&buf, "application/cbor", nil)
	assert.ErrorIs(t, err, core.ErrUnknownContentType)
}

func TestUnmarshalSuffixCBOR(t *testing.T) {
	n := defaultNegotiator()
	var dest any
	err := n.Unmarshal("application/problem+cbor", []byte{}, &dest)
	assert.ErrorIs(t, err, core.ErrUnknownContentType)
}


func TestDefaultJSONFormatMarshal(t *testing.T) {
	f := negotiate.DefaultJSONFormat()
	require.NotNil(t, f.Marshal)
	require.NotNil(t, f.Unmarshal)

	var buf bytes.Buffer
	err := f.Marshal(&buf, map[string]string{"key": "value"})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), `"key":"value"`)
}

func TestDefaultJSONFormatUnmarshal(t *testing.T) {
	f := negotiate.DefaultJSONFormat()

	var dest map[string]string
	err := f.Unmarshal([]byte(`{"name":"test"}`), &dest)
	require.NoError(t, err)
	assert.Equal(t, "test", dest["name"])
}

func TestDefaultJSONFormatNoHTMLEscape(t *testing.T) {
	f := negotiate.DefaultJSONFormat()

	var buf bytes.Buffer
	err := f.Marshal(&buf, map[string]string{"url": "https://example.com?a=1&b=2"})
	require.NoError(t, err)
	assert.NotContains(t, buf.String(), `\u0026`)
	assert.Contains(t, buf.String(), `&`)
}
