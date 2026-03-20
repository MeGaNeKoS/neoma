package conditional

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MeGaNeKoS/neoma/neomatest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasConditional(t *testing.T) {
	p := Params{}
	assert.False(t, p.HasConditionalParams())

	p = Params{IfMatch: []string{"test"}}
	assert.True(t, p.HasConditionalParams())

	p = Params{IfNoneMatch: []string{"test"}}
	assert.True(t, p.HasConditionalParams())

	p = Params{IfModifiedSince: time.Now()}
	assert.True(t, p.HasConditionalParams())

	p = Params{IfUnmodifiedSince: time.Now()}
	assert.True(t, p.HasConditionalParams())
}

func TestIfMatch(t *testing.T) {
	p := Params{}

	r, _ := http.NewRequest(http.MethodGet, "https://example.com/resource", nil)
	w := httptest.NewRecorder()
	ctx := neomatest.NewContext(nil, r, w)

	p.IfMatch = []string{`"abc123"`, `W/"def456"`}
	p.Resolve(ctx)

	status, _ := p.Check("abc123", time.Time{})
	assert.Equal(t, 0, status)

	status, _ = p.Check("def456", time.Time{})
	assert.Equal(t, 0, status)

	status, _ = p.Check("bad", time.Time{})
	assert.Equal(t, http.StatusNotModified, status)

	status, _ = p.Check("", time.Time{})
	assert.Equal(t, http.StatusNotModified, status)

	r, _ = http.NewRequest(http.MethodPut, "https://example.com/resource", nil)
	w = httptest.NewRecorder()
	ctx = neomatest.NewContext(nil, r, w)

	p.IfMatch = []string{`"abc123"`, `W/"def456"`}
	p.Resolve(ctx)

	status, _ = p.Check("abc123", time.Time{})
	assert.Equal(t, 0, status)

	status, msg := p.Check("bad", time.Time{})
	assert.Equal(t, http.StatusPreconditionFailed, status)
	assert.NotEmpty(t, msg)
}

func TestIfNoneMatch(t *testing.T) {
	p := Params{}

	r, _ := http.NewRequest(http.MethodGet, "https://example.com/resource", nil)
	w := httptest.NewRecorder()
	ctx := neomatest.NewContext(nil, r, w)

	p.IfNoneMatch = []string{`"abc123"`, `W/"def456"`}
	p.Resolve(ctx)

	status, _ := p.Check("bad", time.Time{})
	assert.Equal(t, 0, status)

	status, _ = p.Check("", time.Time{})
	assert.Equal(t, 0, status)

	status, _ = p.Check("abc123", time.Time{})
	assert.Equal(t, http.StatusNotModified, status)

	status, _ = p.Check("def456", time.Time{})
	assert.Equal(t, http.StatusNotModified, status)

	r, _ = http.NewRequest(http.MethodPut, "https://example.com/resource", nil)
	w = httptest.NewRecorder()
	ctx = neomatest.NewContext(nil, r, w)

	p.IfNoneMatch = []string{`"abc123"`, `W/"def456"`}
	p.Resolve(ctx)

	status, _ = p.Check("abc123", time.Time{})
	assert.Equal(t, http.StatusPreconditionFailed, status)

	status, _ = p.Check("bad", time.Time{})
	assert.Equal(t, 0, status)

	p.IfNoneMatch = []string{"*"}
	status, _ = p.Check("", time.Time{})
	assert.Equal(t, 0, status)

	status, _ = p.Check("abc123", time.Time{})
	assert.Equal(t, http.StatusPreconditionFailed, status)
}

func TestIfModifiedSince(t *testing.T) {
	p := Params{}

	now, err := time.Parse(time.RFC3339, "2021-01-01T12:00:00Z")
	require.NoError(t, err)
	before, err := time.Parse(time.RFC3339, "2020-01-01T12:00:00Z")
	require.NoError(t, err)
	after, err := time.Parse(time.RFC3339, "2022-01-01T12:00:00Z")
	require.NoError(t, err)

	r, _ := http.NewRequest(http.MethodGet, "https://example.com/resource", nil)
	w := httptest.NewRecorder()
	ctx := neomatest.NewContext(nil, r, w)

	p.IfModifiedSince = now
	p.Resolve(ctx)

	status, _ := p.Check("", before)
	assert.Equal(t, http.StatusNotModified, status)

	status, _ = p.Check("", now)
	assert.Equal(t, http.StatusNotModified, status)

	status, _ = p.Check("", after)
	assert.Equal(t, 0, status)

	r, _ = http.NewRequest(http.MethodPut, "https://example.com/resource", nil)
	w = httptest.NewRecorder()
	ctx = neomatest.NewContext(nil, r, w)

	p.IfModifiedSince = now
	p.Resolve(ctx)

	status, _ = p.Check("", before)
	assert.Equal(t, http.StatusPreconditionFailed, status)
}

func TestIfUnmodifiedSince(t *testing.T) {
	p := Params{}

	now, err := time.Parse(time.RFC3339, "2021-01-01T12:00:00Z")
	require.NoError(t, err)
	before, err := time.Parse(time.RFC3339, "2020-01-01T12:00:00Z")
	require.NoError(t, err)
	after, err := time.Parse(time.RFC3339, "2022-01-01T12:00:00Z")
	require.NoError(t, err)

	r, _ := http.NewRequest(http.MethodGet, "https://example.com/resource", nil)
	w := httptest.NewRecorder()
	ctx := neomatest.NewContext(nil, r, w)

	p.IfUnmodifiedSince = now
	p.Resolve(ctx)

	status, _ := p.Check("", before)
	assert.Equal(t, 0, status)

	status, _ = p.Check("", now)
	assert.Equal(t, 0, status)

	status, _ = p.Check("", after)
	assert.Equal(t, http.StatusNotModified, status)

	r, _ = http.NewRequest(http.MethodPut, "https://example.com/resource", nil)
	w = httptest.NewRecorder()
	ctx = neomatest.NewContext(nil, r, w)

	p.IfUnmodifiedSince = now
	p.Resolve(ctx)

	status, _ = p.Check("", after)
	assert.Equal(t, http.StatusPreconditionFailed, status)
}
