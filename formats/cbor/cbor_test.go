package cbor_test

import (
	"bytes"
	"testing"

	"github.com/MeGaNeKoS/neoma/formats/cbor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultCBORFormatMarshalUnmarshal(t *testing.T) {
	type Msg struct {
		Name string `json:"name"`
	}

	var buf bytes.Buffer
	require.NoError(t, cbor.DefaultCBORFormat.Marshal(&buf, &Msg{Name: "test"}))
	assert.NotEmpty(t, buf.Bytes())

	var out Msg
	require.NoError(t, cbor.DefaultCBORFormat.Unmarshal(buf.Bytes(), &out))
	assert.Equal(t, "test", out.Name)
}
