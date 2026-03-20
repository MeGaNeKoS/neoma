// Package adapters_test runs shared verification tests against every adapter.
package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MeGaNeKoS/neoma/adapters/neomachi/v5"
	neomaechov4 "github.com/MeGaNeKoS/neoma/adapters/neomaecho/v4"
	neomaechov5 "github.com/MeGaNeKoS/neoma/adapters/neomaecho/v5"
	neomafiberv2 "github.com/MeGaNeKoS/neoma/adapters/neomafiber/v2"
	neomafiberv3 "github.com/MeGaNeKoS/neoma/adapters/neomafiber/v3"
	"github.com/MeGaNeKoS/neoma/adapters/neomagin/v1"
	"github.com/MeGaNeKoS/neoma/adapters/neomastdlib"
	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/neoma"
	"github.com/MeGaNeKoS/neoma/neomatest"
	"github.com/gin-gonic/gin"
	"github.com/go-chi/chi/v5"
	fiberv2 "github.com/gofiber/fiber/v2"
	fiberv3 "github.com/gofiber/fiber/v3"
	echov4 "github.com/labstack/echo/v4"
	echov5 "github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
)

type contextKey struct{}

type TestInput struct {
	Group   string `path:"group"`
	Verbose bool   `query:"verbose"`
	Auth    string `header:"Authorization"`
	Body    struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
}

type TestOutput struct {
	MyHeader string `header:"MyHeader"`
	Body     struct {
		Message string `json:"message"`
	}
}

func testHandler(_ context.Context, input *TestInput) (*TestOutput, error) {
	resp := &TestOutput{}
	resp.MyHeader = "my-value"
	resp.Body.Message = fmt.Sprintf(
		"Hello, %s <%s>! (%s, %v, %s)",
		input.Body.Name, input.Body.Email,
		input.Group, input.Verbose, input.Auth,
	)
	return resp, nil
}

func testAdapter(t *testing.T, api core.API) {
	t.Helper()

	methods := []string{http.MethodPut, http.MethodPost}

	for _, method := range methods {
		neoma.Register(api, core.Operation{
			OperationID: method + "-test",
			Method:      method,
			Path:        "/{group}",
		}, testHandler)
	}

	for _, method := range methods {
		testAPI := neomatest.Wrap(t, api)
		resp := testAPI.Do(method, "/foo?verbose=true",
			"Host: localhost",
			"Authorization: Bearer abc123",
			strings.NewReader(`{"name": "Daniel", "email": "daniel@example.com"}`),
		)

		assert.Equal(t, http.StatusOK, resp.Code, "status code")
		assert.Equal(t, "my-value", resp.Header().Get("MyHeader"), "response header MyHeader")
		assert.Contains(t, resp.Body.String(), `"message"`, "body contains message field")
		assert.Contains(t, resp.Body.String(), "Hello, Daniel <daniel@example.com>!", "body greeting")
		assert.Contains(t, resp.Body.String(), "foo", "body contains path param")
		assert.Contains(t, resp.Body.String(), "true", "body contains verbose=true")
		assert.Contains(t, resp.Body.String(), "Bearer abc123", "body contains auth header")
	}
}

func TestAdapters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := func() core.Config {
		return neoma.DefaultConfig("Test", "1.0.0")
	}

	// wrap adds middleware that verifies TLS, Version, context.WithValue, Unwrap,
	// GetResponseHeader, DeleteResponseHeader, and MatchedPattern.
	//
	// expectedPatternContains is the route pattern fragment we expect
	// MatchedPattern to contain. For most adapters this is something like
	// "/{group}" or "/:group". The stdlib adapter prepends the HTTP method
	// (e.g. "PUT /{group}") so we check with Contains rather than Equal.
	//
	// expectedProto is the expected value of Version().Proto for the adapter.
	// Most adapters report "HTTP/1.1" from httptest, but Fiber v2 reports "http"
	// (the scheme) because fasthttp does not expose the full protocol string.
	wrap := func(
		api core.API,
		expectedProto string,
		unwrapper func(ctx core.Context),
		expectedPatternContains string,
	) core.API {
		api.UseMiddleware(
			func(ctx core.Context, next func(core.Context)) {
				assert.Nil(t, ctx.TLS(), "TLS should be nil in test")

				v := ctx.Version()
				assert.Equal(t, expectedProto, v.Proto, "Proto")

				got := ctx.MatchedPattern()
				assert.NotEmpty(t, got, "MatchedPattern should not be empty")
				assert.Contains(t, got, expectedPatternContains,
					"MatchedPattern should contain the route pattern")

				ctx.SetHeader("X-Test-Temp", "temporary")
				assert.Equal(t, "temporary", ctx.GetResponseHeader("X-Test-Temp"),
					"GetResponseHeader should read back the header we just set")

				ctx.DeleteResponseHeader("X-Test-Temp")
				assert.Empty(t, ctx.GetResponseHeader("X-Test-Temp"),
					"DeleteResponseHeader should remove the header")

				ctx = core.WithContext(ctx, context.WithValue(ctx.Context(), contextKey{}, "from-middleware"))
				next(ctx)
			},
			func(ctx core.Context, next func(core.Context)) {
				assert.NotPanics(t, func() { unwrapper(ctx) }, "Unwrap should not panic")
				assert.Equal(t, "from-middleware", ctx.Context().Value(contextKey{}),
					"context value should propagate through WithContext")
				next(ctx)
			},
		)
		return api
	}

	for _, tc := range []struct {
		name   string
		newAPI func() core.API
	}{
		{
			name: "chi",
			newAPI: func() core.API {
				r := chi.NewMux()
				adapter := neomachi.NewAdapter(r)
				api := neoma.NewAPI(config(), adapter)
				return wrap(api, "HTTP/1.1",
					func(ctx core.Context) { neomachi.Unwrap(ctx) },
					"/{group}",
				)
			},
		},
		{
			name: "echo-v4",
			newAPI: func() core.API {
				e := echov4.New()
				adapter := neomaechov4.NewAdapter(e)
				api := neoma.NewAPI(config(), adapter)
				return wrap(api, "HTTP/1.1",
					func(ctx core.Context) { neomaechov4.Unwrap(ctx) },
					"/:group",
				)
			},
		},
		{
			name: "echo-v5",
			newAPI: func() core.API {
				e := echov5.New()
				adapter := neomaechov5.NewAdapter(e)
				api := neoma.NewAPI(config(), adapter)
				return wrap(api, "HTTP/1.1",
					func(ctx core.Context) { neomaechov5.Unwrap(ctx) },
					"/:group",
				)
			},
		},
		{
			name: "gin",
			newAPI: func() core.API {
				r := gin.New()
				adapter := neomagin.NewAdapter(r)
				api := neoma.NewAPI(config(), adapter)
				return wrap(api, "HTTP/1.1",
					func(ctx core.Context) { neomagin.Unwrap(ctx) },
					"/:group",
				)
			},
		},
		{
			name: "fiber-v2",
			newAPI: func() core.API {
				app := fiberv2.New()
				adapter := neomafiberv2.NewAdapter(app)
				api := neoma.NewAPI(config(), adapter)
				return wrap(api, "http",
					func(ctx core.Context) { neomafiberv2.Unwrap(ctx) },
					"/:group",
				)
			},
		},
		{
			name: "fiber-v3",
			newAPI: func() core.API {
				app := fiberv3.New()
				adapter := neomafiberv3.NewAdapter(app)
				api := neoma.NewAPI(config(), adapter)
				return wrap(api, "HTTP/1.1",
					func(ctx core.Context) { neomafiberv3.Unwrap(ctx) },
					"/:group",
				)
			},
		},
		{
			name: "stdlib",
			newAPI: func() core.API {
				mux := http.NewServeMux()
				adapter := neomastdlib.NewAdapter(mux)
				api := neoma.NewAPI(config(), adapter)
				return wrap(api, "HTTP/1.1",
					func(ctx core.Context) { neomastdlib.Unwrap(ctx) },
					"/{group}",
				)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			testAdapter(t, tc.newAPI())
		})
	}
}

func TestUnwrapNeomaflow(t *testing.T) {
	op := &core.Operation{Method: http.MethodGet, Path: "/test"}
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	ctx := neomastdlib.NewContext(op, r, w)
	req, rw := neomastdlib.Unwrap(ctx)
	assert.Equal(t, r, req)
	assert.Equal(t, w, rw)
}

func TestNewContextChi(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	op := &core.Operation{Method: http.MethodGet, Path: "/test"}
	ctx := neomachi.NewContext(op, r, w)
	assert.Equal(t, http.MethodGet, ctx.Method())
	assert.Equal(t, op, ctx.Operation())
}

func TestNewContextGin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	op := &core.Operation{Method: http.MethodGet, Path: "/test"}
	ctx := neomagin.NewContext(op, c)
	assert.Equal(t, http.MethodGet, ctx.Method())
	assert.Equal(t, op, ctx.Operation())
}

func TestNewAdapterWithGroup(t *testing.T) {
	t.Run("echo-v4", func(t *testing.T) {
		e := echov4.New()
		g := e.Group("/api")
		adapter := neomaechov4.NewAdapterWithGroup(e, g)
		assert.NotNil(t, adapter)
	})
	t.Run("echo-v5", func(t *testing.T) {
		e := echov5.New()
		g := e.Group("/api")
		adapter := neomaechov5.NewAdapterWithGroup(e, g)
		assert.NotNil(t, adapter)
	})
	t.Run("gin", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		g := r.Group("/api")
		adapter := neomagin.NewAdapterWithGroup(r, g)
		assert.NotNil(t, adapter)
	})
	t.Run("fiber-v2", func(t *testing.T) {
		app := fiberv2.New()
		g := app.Group("/api")
		adapter := neomafiberv2.NewAdapterWithGroup(app, g)
		assert.NotNil(t, adapter)
	})
	t.Run("fiber-v3", func(t *testing.T) {
		app := fiberv3.New()
		g := app.Group("/api")
		adapter := neomafiberv3.NewAdapterWithGroup(app, g)
		assert.NotNil(t, adapter)
	})
}

func TestConvenienceNew(t *testing.T) {
	cfg := neoma.DefaultConfig("Test", "1.0.0")

	t.Run("chi", func(t *testing.T) {
		api := neomachi.New(chi.NewMux(), cfg)
		assert.NotNil(t, api)
	})
	t.Run("echo-v4", func(t *testing.T) {
		api := neomaechov4.New(echov4.New(), cfg)
		assert.NotNil(t, api)
	})
	t.Run("echo-v5", func(t *testing.T) {
		api := neomaechov5.New(echov5.New(), cfg)
		assert.NotNil(t, api)
	})
	t.Run("gin", func(t *testing.T) {
		api := neomagin.New(gin.New(), cfg)
		assert.NotNil(t, api)
	})
	t.Run("fiber-v2", func(t *testing.T) {
		api := neomafiberv2.New(fiberv2.New(), cfg)
		assert.NotNil(t, api)
	})
	t.Run("fiber-v3", func(t *testing.T) {
		api := neomafiberv3.New(fiberv3.New(), cfg)
		assert.NotNil(t, api)
	})
	t.Run("stdlib", func(t *testing.T) {
		api := neomastdlib.New(http.NewServeMux(), cfg)
		assert.NotNil(t, api)
	})
}
