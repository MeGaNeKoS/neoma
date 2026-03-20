package errors_test

import (
	"context"
	"crypto/tls"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubCtx struct {
	method string
}

func (s *stubCtx) Operation() *core.Operation                  { return nil }
func (s *stubCtx) Context() context.Context                    { return context.Background() }
func (s *stubCtx) TLS() *tls.ConnectionState                   { return nil }
func (s *stubCtx) Version() core.ProtoVersion                  { return core.ProtoVersion{} }
func (s *stubCtx) Method() string                              { return s.method }
func (s *stubCtx) Host() string                                { return "" }
func (s *stubCtx) RemoteAddr() string                          { return "" }
func (s *stubCtx) URL() url.URL                                { return url.URL{} }
func (s *stubCtx) Param(string) string                         { return "" }
func (s *stubCtx) Query(string) string                         { return "" }
func (s *stubCtx) Header(string) string                        { return "" }
func (s *stubCtx) EachHeader(func(string, string))             {}
func (s *stubCtx) BodyReader() io.Reader                       { return nil }
func (s *stubCtx) GetMultipartForm() (*multipart.Form, error)  { return nil, nil }
func (s *stubCtx) SetReadDeadline(time.Time) error             { return nil }
func (s *stubCtx) SetStatus(int)                               {}
func (s *stubCtx) Status() int                                 { return 0 }
func (s *stubCtx) SetHeader(string, string)                    {}
func (s *stubCtx) AppendHeader(string, string)                 {}
func (s *stubCtx) GetResponseHeader(string) string             { return "" }
func (s *stubCtx) DeleteResponseHeader(string)                 {}
func (s *stubCtx) BodyWriter() io.Writer                       { return io.Discard }
func (s *stubCtx) MatchedPattern() string                      { return "" }

func TestNoopContentTypePassthrough(t *testing.T) {
	f := errors.NewNoopHandler()
	assert.Equal(t, "text/html", f.ErrorContentType("text/html"))
	assert.Equal(t, "application/cbor", f.ErrorContentType("application/cbor"))
}

func TestProblemDetailGetType(t *testing.T) {
	em := &errors.ProblemDetail{Type: "https://example.com/errors/400"}
	assert.Equal(t, "https://example.com/errors/400", em.GetType())
}

func TestProblemDetailGetTypeAboutBlank(t *testing.T) {
	em := &errors.ProblemDetail{Type: "about:blank"}
	assert.Equal(t, "about:blank", em.GetType())
}

func TestRFC9457HandlerGetTypeBaseURI(t *testing.T) {
	f := errors.NewRFC9457HandlerWithConfig("/errors", nil)
	assert.Equal(t, "/errors", f.GetTypeBaseURI())
}

func TestRFC9457HandlerGetTypeBaseURIEmpty(t *testing.T) {
	f := errors.NewRFC9457Handler()
	assert.Empty(t, f.GetTypeBaseURI())
}

func TestRFC9457HandlerNewErrorWithContextInstanceFunc(t *testing.T) {
	f := errors.NewRFC9457HandlerWithConfig("/errors", func(ctx core.Context) string {
		return "/traces/abc123"
	})
	se := f.NewErrorWithContext(nil, http.StatusBadRequest, "bad")
	var em *errors.ProblemDetail
	require.ErrorAs(t, se, &em)
	assert.Empty(t, em.Instance)
}

func TestRFC9457HandlerNewErrorWithContextInstanceFuncNonNilCtx(t *testing.T) {
	f := errors.NewRFC9457HandlerWithConfig("/errors", func(ctx core.Context) string {
		return "/traces/" + ctx.Method()
	})
	ctx := &stubCtx{method: "POST"}
	se := f.NewErrorWithContext(ctx, http.StatusBadRequest, "bad input")
	var em *errors.ProblemDetail
	require.ErrorAs(t, se, &em)
	assert.Equal(t, "/traces/POST", em.Instance)
	assert.Equal(t, "/errors/400", em.Type)
}

func TestRFC9457HandlerTypeURIDefault(t *testing.T) {
	f := errors.NewRFC9457Handler()
	se := f.NewError(http.StatusNotFound, "not found")
	var em *errors.ProblemDetail
	require.ErrorAs(t, se, &em)
	assert.Equal(t, "about:blank", em.Type)
}

func TestRFC9457HandlerTypeURIWithBase(t *testing.T) {
	f := errors.NewRFC9457HandlerWithConfig("/errors", nil)
	se := f.NewError(http.StatusBadRequest, "bad")
	var em *errors.ProblemDetail
	require.ErrorAs(t, se, &em)
	assert.Equal(t, "/errors/400", em.Type)
}
