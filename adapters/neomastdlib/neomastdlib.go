// Package neomastdlib provides a neoma adapter for Go's standard library net/http router.
package neomastdlib

import (
	"context"
	"crypto/tls"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/neoma"
)

// MultipartMaxMemory is the maximum memory in bytes used for parsing multipart forms.
var MultipartMaxMemory int64 = 8 * 1024

type stdlibContext struct {
	op     *core.Operation
	r      *http.Request
	w      http.ResponseWriter
	status int
}

func (c *stdlibContext) Operation() *core.Operation { return c.op }
func (c *stdlibContext) Context() context.Context    { return c.r.Context() }
func (c *stdlibContext) Method() string              { return c.r.Method }
func (c *stdlibContext) Host() string                { return c.r.Host }
func (c *stdlibContext) RemoteAddr() string          { return c.r.RemoteAddr }
func (c *stdlibContext) URL() url.URL                { return *c.r.URL }

func (c *stdlibContext) Param(name string) string {
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

func (c *stdlibContext) Query(name string) string {
	return c.r.URL.Query().Get(name)
}

func (c *stdlibContext) Header(name string) string { return c.r.Header.Get(name) }

func (c *stdlibContext) EachHeader(cb func(name, value string)) {
	for name, values := range c.r.Header {
		for _, value := range values {
			cb(name, value)
		}
	}
}

func (c *stdlibContext) BodyReader() io.Reader { return c.r.Body }

func (c *stdlibContext) GetMultipartForm() (*multipart.Form, error) {
	err := c.r.ParseMultipartForm(MultipartMaxMemory)
	return c.r.MultipartForm, err
}

func (c *stdlibContext) SetReadDeadline(deadline time.Time) error {
	return core.SetReadDeadline(c.w, deadline)
}

func (c *stdlibContext) SetStatus(code int) {
	c.status = code
	c.w.WriteHeader(code)
}

func (c *stdlibContext) Status() int                          { return c.status }
func (c *stdlibContext) SetHeader(name, value string)         { c.w.Header().Set(name, value) }
func (c *stdlibContext) AppendHeader(name, value string)      { c.w.Header().Add(name, value) }
func (c *stdlibContext) GetResponseHeader(name string) string { return c.w.Header().Get(name) }
func (c *stdlibContext) DeleteResponseHeader(name string)     { c.w.Header().Del(name) }
func (c *stdlibContext) BodyWriter() io.Writer                { return c.w }
func (c *stdlibContext) TLS() *tls.ConnectionState            { return c.r.TLS }

func (c *stdlibContext) Version() core.ProtoVersion {
	return core.ProtoVersion{
		Proto:      c.r.Proto,
		ProtoMajor: c.r.ProtoMajor,
		ProtoMinor: c.r.ProtoMinor,
	}
}

func (c *stdlibContext) MatchedPattern() string {
	if pat := c.r.Pattern; pat != "" {
		return pat
	}
	return c.op.Path
}

// Unwrap extracts the underlying *http.Request and http.ResponseWriter from a neoma Context.
// It panics if the context was not created by this adapter.
func Unwrap(ctx core.Context) (*http.Request, http.ResponseWriter) {
	c, ok := core.UnwrapContext(ctx).(*stdlibContext)
	if !ok {
		panic("not a neomastdlib context")
	}
	return c.r, c.w
}

type stdlibAdapter struct {
	mux *http.ServeMux
}

func (a *stdlibAdapter) Handle(op *core.Operation, handler func(core.Context)) {
	pattern := op.Method + " " + convertPath(op.Path)
	a.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		handler(&stdlibContext{op: op, r: r, w: w})
	})
}

func (a *stdlibAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mux.ServeHTTP(w, r)
}

// NewContext creates a new neoma Context wrapping the given http.Request and http.ResponseWriter.
func NewContext(op *core.Operation, r *http.Request, w http.ResponseWriter) core.Context {
	return &stdlibContext{op: op, r: r, w: w}
}

// NewAdapter creates a new neoma Adapter wrapping the given http.ServeMux.
func NewAdapter(mux *http.ServeMux) core.Adapter {
	return &stdlibAdapter{mux: mux}
}

// New creates a new neoma API using the given http.ServeMux and configuration.
func New(mux *http.ServeMux, config core.Config) core.API {
	return neoma.NewAPI(config, NewAdapter(mux))
}

func convertPath(path string) string {
	path = strings.TrimRight(path, "/")
	if path == "" {
		path = "/"
	}
	return path
}
