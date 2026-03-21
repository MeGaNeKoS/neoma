// Package neomagin provides a neoma adapter for the Gin web framework.
package neomagin

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
	"github.com/gin-gonic/gin"
)

// MultipartMaxMemory is the maximum memory in bytes used for parsing multipart forms.
var MultipartMaxMemory int64 = 8 * 1024

// Unwrap extracts the underlying *gin.Context from a neoma Context.
// It panics if the context was not created by this adapter.
func Unwrap(ctx core.Context) *gin.Context {
	c, ok := core.UnwrapContext(ctx).(*ginCtx)
	if !ok {
		panic("not a neomagin context")
	}
	return c.Unwrap()
}

type ginCtx struct {
	op     *core.Operation
	orig   *gin.Context
	status int
}

func (c *ginCtx) Unwrap() *gin.Context {
	return c.orig
}

func (c *ginCtx) Operation() *core.Operation {
	return c.op
}

func (c *ginCtx) Context() context.Context {
	return c.orig.Request.Context()
}

func (c *ginCtx) Method() string {
	return c.orig.Request.Method
}

func (c *ginCtx) Host() string {
	return c.orig.Request.Host
}

func (c *ginCtx) RemoteAddr() string {
	return c.orig.Request.RemoteAddr
}

func (c *ginCtx) URL() url.URL {
	return *c.orig.Request.URL
}

func (c *ginCtx) Param(name string) string {
	return c.orig.Param(name)
}

func (c *ginCtx) Query(name string) string {
	return c.orig.Query(name)
}

func (c *ginCtx) Header(name string) string {
	return c.orig.GetHeader(name)
}

func (c *ginCtx) EachHeader(cb func(name, value string)) {
	for name, values := range c.orig.Request.Header {
		for _, value := range values {
			cb(name, value)
		}
	}
}

func (c *ginCtx) BodyReader() io.Reader {
	return c.orig.Request.Body
}

func (c *ginCtx) GetMultipartForm() (*multipart.Form, error) {
	err := c.orig.Request.ParseMultipartForm(MultipartMaxMemory)
	return c.orig.Request.MultipartForm, err
}

func (c *ginCtx) SetReadDeadline(deadline time.Time) error {
	return core.SetReadDeadline(c.orig.Writer, deadline)
}

func (c *ginCtx) SetStatus(code int) {
	c.status = code
	c.orig.Status(code)
}

func (c *ginCtx) Status() int {
	return c.status
}

func (c *ginCtx) AppendHeader(name string, value string) {
	c.orig.Writer.Header().Add(name, value)
}

func (c *ginCtx) SetHeader(name string, value string) {
	c.orig.Header(name, value)
}

func (c *ginCtx) GetResponseHeader(name string) string {
	return c.orig.Writer.Header().Get(name)
}

func (c *ginCtx) DeleteResponseHeader(name string) {
	c.orig.Writer.Header().Del(name)
}

func (c *ginCtx) BodyWriter() io.Writer {
	return c.orig.Writer
}

func (c *ginCtx) TLS() *tls.ConnectionState {
	return c.orig.Request.TLS
}

func (c *ginCtx) Version() core.ProtoVersion {
	return core.ProtoVersion{
		Proto:      c.orig.Request.Proto,
		ProtoMajor: c.orig.Request.ProtoMajor,
		ProtoMinor: c.orig.Request.ProtoMinor,
	}
}

func (c *ginCtx) MatchedPattern() string {
	return c.orig.FullPath()
}

// NewContext creates a new neoma Context wrapping the given *gin.Context.
func NewContext(op *core.Operation, c *gin.Context) core.Context {
	return &ginCtx{op: op, orig: c}
}

// Router is the common interface for gin.Engine and gin.RouterGroup.
type Router interface {
	Handle(string, string, ...gin.HandlerFunc) gin.IRoutes
}

type ginAdapter struct {
	http.Handler
	router Router
}

func (a *ginAdapter) Handle(op *core.Operation, handler func(core.Context)) {
	// Convert {param} to :param
	path := op.Path
	path = strings.ReplaceAll(path, "{", ":")
	path = strings.ReplaceAll(path, "}", "")
	a.router.Handle(op.Method, path, func(c *gin.Context) {
		ctx := &ginCtx{op: op, orig: c}
		handler(ctx)
	})
}

func (a *ginAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.Handler.ServeHTTP(w, r)
}

// NewAdapter creates a new neoma Adapter wrapping the given gin.Engine.
func NewAdapter(r *gin.Engine) core.Adapter {
	return &ginAdapter{Handler: r, router: r}
}

// NewAdapterWithGroup creates a new neoma Adapter that registers routes on the given gin.RouterGroup.
func NewAdapterWithGroup(r *gin.Engine, g *gin.RouterGroup) core.Adapter {
	return &ginAdapter{Handler: r, router: g}
}

// New creates a new neoma API using the given gin.Engine and configuration.
func New(r *gin.Engine, config core.Config) core.API {
	return neoma.NewAPI(config, NewAdapter(r))
}
