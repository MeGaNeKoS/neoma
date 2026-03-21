// Package neomachi provides a neoma adapter for the chi v5 HTTP router.
package neomachi

import (
	"context"
	"crypto/tls"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/neoma"
	"github.com/go-chi/chi/v5"
)

// MultipartMaxMemory is the maximum memory in bytes used for parsing multipart forms.
var MultipartMaxMemory int64 = 8 * 1024

// Unwrap extracts the underlying *http.Request and http.ResponseWriter from a neoma Context.
// It panics if the context was not created by this adapter.
func Unwrap(ctx core.Context) (*http.Request, http.ResponseWriter) {
	c, ok := core.UnwrapContext(ctx).(*chiContext)
	if !ok {
		panic("not a neomachi context")
	}
	return c.r, c.w
}

type chiContext struct {
	op     *core.Operation
	r      *http.Request
	w      http.ResponseWriter
	status int
}

func (c *chiContext) Operation() *core.Operation {
	return c.op
}

func (c *chiContext) Context() context.Context {
	return c.r.Context()
}

func (c *chiContext) Method() string {
	return c.r.Method
}

func (c *chiContext) Host() string {
	return c.r.Host
}

func (c *chiContext) RemoteAddr() string {
	return c.r.RemoteAddr
}

func (c *chiContext) URL() url.URL {
	return *c.r.URL
}

func (c *chiContext) Param(name string) string {
	v := c.r.PathValue(name)
	if c.r.URL.RawPath == "" {
		return v
	}
	u, err := url.PathUnescape(v)
	if err != nil {
		return v
	}
	return u
}

func (c *chiContext) Query(name string) string {
	return c.r.URL.Query().Get(name)
}

func (c *chiContext) Header(name string) string {
	return c.r.Header.Get(name)
}

func (c *chiContext) EachHeader(cb func(name, value string)) {
	for name, values := range c.r.Header {
		for _, value := range values {
			cb(name, value)
		}
	}
}

func (c *chiContext) BodyReader() io.Reader {
	return c.r.Body
}

func (c *chiContext) GetMultipartForm() (*multipart.Form, error) {
	err := c.r.ParseMultipartForm(MultipartMaxMemory)
	return c.r.MultipartForm, err
}

func (c *chiContext) SetReadDeadline(deadline time.Time) error {
	return core.SetReadDeadline(c.w, deadline)
}

func (c *chiContext) SetStatus(code int) {
	c.status = code
	c.w.WriteHeader(code)
}

func (c *chiContext) Status() int {
	return c.status
}

func (c *chiContext) AppendHeader(name string, value string) {
	c.w.Header().Add(name, value)
}

func (c *chiContext) SetHeader(name string, value string) {
	c.w.Header().Set(name, value)
}

func (c *chiContext) GetResponseHeader(name string) string {
	return c.w.Header().Get(name)
}

func (c *chiContext) DeleteResponseHeader(name string) {
	c.w.Header().Del(name)
}

func (c *chiContext) BodyWriter() io.Writer {
	return c.w
}

func (c *chiContext) TLS() *tls.ConnectionState {
	return c.r.TLS
}

func (c *chiContext) Version() core.ProtoVersion {
	return core.ProtoVersion{
		Proto:      c.r.Proto,
		ProtoMajor: c.r.ProtoMajor,
		ProtoMinor: c.r.ProtoMinor,
	}
}

func (c *chiContext) MatchedPattern() string {
	rctx := chi.RouteContext(c.r.Context())
	if rctx != nil {
		return rctx.RoutePattern()
	}
	return ""
}

// NewContext creates a new neoma Context wrapping the given http.Request and http.ResponseWriter.
func NewContext(op *core.Operation, r *http.Request, w http.ResponseWriter) core.Context {
	return &chiContext{op: op, r: r, w: w}
}

type chiAdapter struct {
	router chi.Router
}

func (a *chiAdapter) Handle(op *core.Operation, handler func(core.Context)) {
	a.router.MethodFunc(op.Method, op.Path, func(w http.ResponseWriter, r *http.Request) {
		handler(&chiContext{op: op, r: r, w: w})
	})
}

func (a *chiAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.router.ServeHTTP(w, r)
}

// NewAdapter creates a new neoma Adapter wrapping the given chi Router.
func NewAdapter(r chi.Router) core.Adapter {
	return &chiAdapter{router: r}
}

// New creates a new neoma API using the given chi Router and configuration.
func New(r chi.Router, config core.Config) core.API {
	return neoma.NewAPI(config, NewAdapter(r))
}
