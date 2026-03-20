package negotiate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSelectQValue(t *testing.T) {
	assert.Equal(t, "b", SelectQValue("a; q=0.5, b;q=1.0,c; q=0.3", []string{"a", "b", "d"}))
}

func TestSelectQValueBest(t *testing.T) {
	assert.Equal(t, "b", SelectQValue("a; q=1.0, b;q=1.0,c; q=0.3", []string{"b", "a"}))
}

func TestSelectQValueSimple(t *testing.T) {
	assert.Equal(t, "b", SelectQValue("a; q=0.5, b,c; q=0.3", []string{"a", "b", "c"}))
}

func TestSelectQValueSingle(t *testing.T) {
	assert.Equal(t, "b", SelectQValue("b", []string{"a", "b", "c"}))
}

func TestSelectQValueNoMatch(t *testing.T) {
	assert.Empty(t, SelectQValue("a; q=1.0, b;q=1.0,c; q=0.3", []string{"d", "e"}))
}

func TestSelectQValueFast(t *testing.T) {
	assert.Equal(t, "b", SelectQValueFast("a; q=0.5, b;q=1.0,c; q=0.3", []string{"a", "b", "d"}))
}

func TestSelectQValueFastCBOR(t *testing.T) {
	assert.Equal(t, "application/cbor", SelectQValueFast("application/ion;q=0.6,application/json;q=0.5,application/yaml;q=0.5,text/*;q=0.2,application/cbor;q=0.9,application/msgpack;q=0.8,*/*", []string{"application/json", "application/cbor"}))
}

func TestSelectQValueFastHTML(t *testing.T) {
	assert.Equal(t, "text/html", SelectQValueFast("text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7", []string{"text/html", "application/json", "application/cbor"}))
}

func TestSelectQValueFastYAML(t *testing.T) {
	assert.Equal(t, "application/yaml", SelectQValueFast("application/yaml", []string{"application/json", "application/yaml", "application/cbor"}))
}

func TestSelectQValueFastBest(t *testing.T) {
	assert.Equal(t, "b", SelectQValueFast("a; q=1.0, b;q=1.0,c; q=0.3", []string{"b", "a"}))
}

func TestSelectQValueFastNoMatch(t *testing.T) {
	assert.Empty(t, SelectQValueFast("a; q=1.0, b;q=1.0,c; q=0.3", []string{"d", "e"}))
}

func TestSelectQValueFastMalformed(t *testing.T) {
	assert.Empty(t, SelectQValueFast("a;,", []string{"d", "e"}))
	assert.Equal(t, "a", SelectQValueFast(",a ", []string{"a", "b"}))
	assert.Empty(t, SelectQValueFast("a;;", []string{"a", "b"}))
	assert.Empty(t, SelectQValueFast(";,", []string{"a", "b"}))
	assert.Equal(t, "a", SelectQValueFast("a;q=invalid", []string{"a", "b"}))
}

var benchResult string

func BenchmarkSelectQValue(b *testing.B) {
	_ = benchResult
	header := "application/ion;q=0.6,application/json;q=0.5,application/yaml;q=0.5,text/*;q=0.2,application/cbor;q=0.9,application/msgpack;q=0.8,*/*"
	allowed := []string{"application/json", "application/yaml", "application/cbor"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchResult = SelectQValue(header, allowed)
	}
}

func BenchmarkSelectQValueFast(b *testing.B) {
	_ = benchResult
	header := "application/ion;q=0.6,application/json;q=0.5,application/yaml;q=0.5,text/*;q=0.2,application/cbor;q=0.9,application/msgpack;q=0.8,*/*"
	allowed := []string{"application/json", "application/yaml", "application/cbor"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchResult = SelectQValueFast(header, allowed)
	}
}
