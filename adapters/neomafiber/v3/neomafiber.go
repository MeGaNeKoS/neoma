package neomafiber

import (
	"bytes"
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
	"github.com/gofiber/fiber/v3"
)

var MultipartMaxMemory int64 = 8 * 1024

// Unwrap extracts the underlying Fiber context from a Neoma context. If passed a
// context from a different adapter it will panic. Keep in mind the limitations
// of the underlying Fiber/fasthttp libraries and how that impacts
// memory-safety: https://docs.gofiber.io/#zero-allocation. Do not keep
// references to the underlying context or its values!
func Unwrap(ctx core.Context) fiber.Ctx {
	c, ok := core.UnwrapContext(ctx).(*fiberWrapper)
	if !ok {
		panic("not a neomafiber context")
	}
	return c.Unwrap()
}

type fiberAdapter struct {
	tester requestTester
	router router
}

type fiberWrapper struct {
	op     *core.Operation
	status int
	orig   fiber.Ctx
	ctx    context.Context
}

func (c *fiberWrapper) Unwrap() fiber.Ctx {
	return c.orig
}

func (c *fiberWrapper) Operation() *core.Operation {
	return c.op
}

func (c *fiberWrapper) Context() context.Context {
	return c.ctx
}

func (c *fiberWrapper) Method() string {
	return c.orig.Method()
}

func (c *fiberWrapper) Host() string {
	return c.orig.Hostname()
}

func (c *fiberWrapper) RemoteAddr() string {
	return c.orig.RequestCtx().RemoteAddr().String()
}

func (c *fiberWrapper) URL() url.URL {
	u, _ := url.Parse(c.orig.OriginalURL())
	return *u
}

func (c *fiberWrapper) Param(name string) string {
	return c.orig.Params(name)
}

func (c *fiberWrapper) Query(name string) string {
	return c.orig.Query(name)
}

func (c *fiberWrapper) Header(name string) string {
	return c.orig.Get(name)
}

func (c *fiberWrapper) EachHeader(cb func(name, value string)) {
	headers := c.orig.GetReqHeaders()
	for name, values := range headers {
		for _, value := range values {
			cb(name, value)
		}
	}
}

func (c *fiberWrapper) BodyReader() io.Reader {
	if c.orig.App().Server().StreamRequestBody {
		return c.orig.Request().BodyStream()
	}
	return bytes.NewReader(c.orig.Body())
}

func (c *fiberWrapper) GetMultipartForm() (*multipart.Form, error) {
	ct := string(c.orig.Request().Header.ContentType())
	boundary := ""
	for _, part := range strings.Split(ct, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "boundary=") {
			boundary = strings.TrimPrefix(part, "boundary=")
			break
		}
	}
	if boundary == "" {
		return c.orig.MultipartForm()
	}
	reader := multipart.NewReader(bytes.NewReader(c.orig.Body()), boundary)
	return reader.ReadForm(MultipartMaxMemory)
}

func (c *fiberWrapper) SetReadDeadline(deadline time.Time) error {
	// Note: for this to work properly you need to do two things:
	// 1. Set the Fiber app's `StreamRequestBody` to `true`
	// 2. Set the Fiber app's `BodyLimit` to some small value like `1`
	// Fiber will only call the request handler for streaming once the limit is
	// reached. This is annoying but currently how things work.
	return c.orig.RequestCtx().Conn().SetReadDeadline(deadline)
}

func (c *fiberWrapper) SetStatus(code int) {
	c.status = code
	c.orig.Status(code)
}

func (c *fiberWrapper) Status() int {
	return c.status
}

func (c *fiberWrapper) AppendHeader(name string, value string) {
	c.orig.Append(name, value)
}

func (c *fiberWrapper) SetHeader(name string, value string) {
	c.orig.Set(name, value)
}

func (c *fiberWrapper) GetResponseHeader(name string) string {
	return c.orig.GetRespHeader(name)
}

func (c *fiberWrapper) DeleteResponseHeader(name string) {
	c.orig.Response().Header.Del(name)
}

func (c *fiberWrapper) BodyWriter() io.Writer {
	return c.orig
}

func (c *fiberWrapper) TLS() *tls.ConnectionState {
	return c.orig.RequestCtx().TLSConnectionState()
}

func (c *fiberWrapper) Version() core.ProtoVersion {
	return core.ProtoVersion{
		Proto: c.orig.Protocol(),
	}
}

func (c *fiberWrapper) MatchedPattern() string {
	return c.orig.Route().Path
}

type router interface {
	Add(methods []string, path string, handler any, handlers ...any) fiber.Router
}

type requestTester interface {
	Test(*http.Request, ...fiber.TestConfig) (*http.Response, error)
}

type contextWrapperValue struct {
	Key   any
	Value any
}

type contextWrapper struct {
	values []*contextWrapperValue
	context.Context
}

var _ context.Context = &contextWrapper{}

func (c *contextWrapper) Value(key any) any {
	raw := c.Context.Value(key)
	if raw != nil {
		return raw
	}
	for _, pair := range c.values {
		if pair.Key == key {
			return pair.Value
		}
	}
	return nil
}

func (a *fiberAdapter) Handle(op *core.Operation, handler func(core.Context)) {
	path := op.Path
	path = strings.ReplaceAll(path, "{", ":")
	path = strings.ReplaceAll(path, "}", "")
	a.router.Add([]string{op.Method}, path, func(c fiber.Ctx) error {
		var values []*contextWrapperValue
		c.RequestCtx().VisitUserValuesAll(func(key, value any) {
			values = append(values, &contextWrapperValue{
				Key:   key,
				Value: value,
			})
		})
		handler(&fiberWrapper{
			op:   op,
			orig: c,
			ctx: &contextWrapper{
				values:  values,
				Context: c.Context(),
			},
		})
		return nil
	})
}

func (a *fiberAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp, err := a.tester.Test(r)
	if resp != nil && resp.Body != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}
	if err != nil {
		panic(err)
	}
	h := w.Header()
	for k, v := range resp.Header {
		for item := range v {
			h.Add(k, v[item])
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func NewAdapter(r *fiber.App) core.Adapter {
	return &fiberAdapter{tester: r, router: r}
}

func NewAdapterWithGroup(r *fiber.App, g fiber.Router) core.Adapter {
	return &fiberAdapter{tester: r, router: g}
}

func New(r *fiber.App, config core.Config) core.API {
	return neoma.NewAPI(config, NewAdapter(r))
}
