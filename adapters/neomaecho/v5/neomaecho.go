// Package neomaecho provides a neoma adapter for the Echo v5 web framework.
package neomaecho

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
	"github.com/labstack/echo/v5"
)

// MultipartMaxMemory is the maximum memory in bytes used for parsing multipart forms.
var MultipartMaxMemory int64 = 8 * 1024

// Unwrap extracts the underlying *echo.Context from a neoma Context.
// It panics if the context was not created by this adapter.
func Unwrap(ctx core.Context) *echo.Context {
	c, ok := core.UnwrapContext(ctx).(*echoCtx)
	if !ok {
		panic("not a neomaecho context")
	}
	return c.Unwrap()
}

type echoCtx struct {
	op     *core.Operation
	orig   *echo.Context
	status int
}

func (c *echoCtx) Unwrap() *echo.Context {
	return c.orig
}

func (c *echoCtx) Operation() *core.Operation {
	return c.op
}

func (c *echoCtx) Context() context.Context {
	return c.orig.Request().Context()
}

func (c *echoCtx) Method() string {
	return c.orig.Request().Method
}

func (c *echoCtx) Host() string {
	return c.orig.Request().Host
}

func (c *echoCtx) RemoteAddr() string {
	return c.orig.Request().RemoteAddr
}

func (c *echoCtx) URL() url.URL {
	return *c.orig.Request().URL
}

func (c *echoCtx) Param(name string) string {
	return c.orig.Param(name)
}

func (c *echoCtx) Query(name string) string {
	return c.orig.QueryParam(name)
}

func (c *echoCtx) Header(name string) string {
	return c.orig.Request().Header.Get(name)
}

func (c *echoCtx) EachHeader(cb func(name, value string)) {
	for name, values := range c.orig.Request().Header {
		for _, value := range values {
			cb(name, value)
		}
	}
}

func (c *echoCtx) BodyReader() io.Reader {
	return c.orig.Request().Body
}

func (c *echoCtx) GetMultipartForm() (*multipart.Form, error) {
	err := c.orig.Request().ParseMultipartForm(MultipartMaxMemory)
	return c.orig.Request().MultipartForm, err
}

func (c *echoCtx) SetReadDeadline(deadline time.Time) error {
	return core.SetReadDeadline(c.orig.Response(), deadline)
}

func (c *echoCtx) SetStatus(code int) {
	c.status = code
	c.orig.Response().WriteHeader(code)
}

func (c *echoCtx) Status() int {
	return c.status
}

func (c *echoCtx) AppendHeader(name, value string) {
	c.orig.Response().Header().Add(name, value)
}

func (c *echoCtx) SetHeader(name, value string) {
	c.orig.Response().Header().Set(name, value)
}

func (c *echoCtx) GetResponseHeader(name string) string {
	return c.orig.Response().Header().Get(name)
}

func (c *echoCtx) DeleteResponseHeader(name string) {
	c.orig.Response().Header().Del(name)
}

func (c *echoCtx) BodyWriter() io.Writer {
	return c.orig.Response()
}

func (c *echoCtx) TLS() *tls.ConnectionState {
	return c.orig.Request().TLS
}

func (c *echoCtx) Version() core.ProtoVersion {
	r := c.orig.Request()
	return core.ProtoVersion{
		Proto:      r.Proto,
		ProtoMajor: r.ProtoMajor,
		ProtoMinor: r.ProtoMinor,
	}
}

func (c *echoCtx) MatchedPattern() string {
	return c.orig.Path()
}

type router interface {
	Add(method, path string, handler echo.HandlerFunc, middlewares ...echo.MiddlewareFunc) echo.RouteInfo
}

type echoAdapter struct {
	http.Handler
	router router
}

func (a *echoAdapter) Handle(op *core.Operation, handler func(core.Context)) {
	path := op.Path
	path = strings.ReplaceAll(path, "{", ":")
	path = strings.ReplaceAll(path, "}", "")
	a.router.Add(op.Method, path, func(c *echo.Context) error {
		ctx := &echoCtx{op: op, orig: c}
		handler(ctx)
		return nil
	})
}

func (a *echoAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.Handler.ServeHTTP(w, r)
}

// NewAdapter creates a new neoma Adapter wrapping the given Echo instance.
func NewAdapter(r *echo.Echo) core.Adapter {
	return &echoAdapter{Handler: r, router: r}
}

// NewAdapterWithGroup creates a new neoma Adapter that registers routes on the given Echo Group.
func NewAdapterWithGroup(r *echo.Echo, g *echo.Group) core.Adapter {
	return &echoAdapter{Handler: r, router: g}
}

// New creates a new neoma API using the given Echo instance and configuration.
func New(r *echo.Echo, config core.Config) core.API {
	return neoma.NewAPI(config, NewAdapter(r))
}
