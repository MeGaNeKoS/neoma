package middleware_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/middleware"
	"github.com/MeGaNeKoS/neoma/neoma"
	"github.com/MeGaNeKoS/neoma/neomatest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)


func newAPI(t *testing.T) neomatest.TestAPI {
	t.Helper()
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))
	return api
}


func TestNewGroupPrefix(t *testing.T) {
	api := newAPI(t)
	grp := middleware.NewGroup(api, "/v1")
	require.NotNil(t, grp)

	neoma.Register(grp, core.Operation{
		Method:      http.MethodGet,
		Path:        "/items",
		OperationID: "list-items",
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/v1/items")
	assert.Equal(t, http.StatusNoContent, resp.Code)
}

func TestNewGroupNoPrefix(t *testing.T) {
	api := newAPI(t)
	grp := middleware.NewGroup(api)
	require.NotNil(t, grp)

	neoma.Register(grp, core.Operation{
		Method:      http.MethodGet,
		Path:        "/plain",
		OperationID: "plain",
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/plain")
	assert.Equal(t, http.StatusNoContent, resp.Code)
}


func TestGroupMiddlewareScoped(t *testing.T) {
	api := newAPI(t)
	grp := middleware.NewGroup(api, "/scoped")

	called := false
	grp.UseMiddleware(func(ctx core.Context, next func(core.Context)) {
		called = true
		next(ctx)
	})

	neoma.Register(grp, core.Operation{
		Method:      http.MethodGet,
		Path:        "/endpoint",
		OperationID: "scoped-endpoint",
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/unscoped",
		OperationID: "unscoped",
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/scoped/endpoint")
	assert.Equal(t, http.StatusNoContent, resp.Code)
	assert.True(t, called, "group middleware should have been called")

	called = false
	resp = api.Do(http.MethodGet, "/unscoped")
	assert.Equal(t, http.StatusNoContent, resp.Code)
	assert.False(t, called, "group middleware should NOT be called for unscoped routes")
}


func TestNestedGroups(t *testing.T) {
	api := newAPI(t)
	v1 := middleware.NewGroup(api, "/v1")
	admin := v1.Group("/admin")

	neoma.Register(admin, core.Operation{
		Method:      http.MethodGet,
		Path:        "/users",
		OperationID: "admin-users",
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/v1/admin/users")
	assert.Equal(t, http.StatusNoContent, resp.Code)
}


func TestUseSimpleModifier(t *testing.T) {
	api := newAPI(t)
	grp := middleware.NewGroup(api, "/mod")

	grp.UseSimpleModifier(func(op *core.Operation) {
		op.Tags = append(op.Tags, "modified")
	})

	neoma.Register(grp, core.Operation{
		Method:      http.MethodGet,
		Path:        "/test",
		OperationID: "modified-op",
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	oapi := api.OpenAPI()
	pi := oapi.Paths["/mod/test"]
	require.NotNil(t, pi)
	require.NotNil(t, pi.Get)
	assert.Contains(t, pi.Get.Tags, "modified")
}


func TestMiddlewareBuilder(t *testing.T) {
	buildCalled := false

	builder := middleware.BuilderFunc(func(op *core.Operation) core.MiddlewareFunc {
		buildCalled = true
		return func(ctx core.Context, next func(core.Context)) {
			ctx.SetHeader("X-Builder", "yes")
			next(ctx)
		}
	})

	op := &core.Operation{Method: "GET", Path: "/test"}
	mw := middleware.Build(builder, op)
	assert.True(t, buildCalled)
	require.NotNil(t, mw)
}

func TestNewBuilderModifier(t *testing.T) {
	builder := middleware.BuilderFunc(func(op *core.Operation) core.MiddlewareFunc {
		return func(ctx core.Context, next func(core.Context)) {
			next(ctx)
		}
	})

	modifier := middleware.NewBuilderModifier(builder)
	require.NotNil(t, modifier)

	op := &core.Operation{Method: "GET", Path: "/test"}
	nextCalled := false
	modifier(op, func(o *core.Operation) {
		nextCalled = true
		assert.Len(t, o.Middlewares, 1)
	})
	assert.True(t, nextCalled)
}


func TestTestMiddlewareHelper(t *testing.T) {
	mw := func(ctx core.Context, next func(core.Context)) {
		ctx.SetHeader("X-Test", "1")
		next(ctx)
	}

	handlerCalled := false
	handler := middleware.TestMiddleware(mw, func(ctx core.Context) {
		handlerCalled = true
	})

	api := newAPI(t)
	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/mw-test",
		OperationID: "mw-test",
		Middlewares: core.Middlewares{mw},
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	require.NotNil(t, handler)

	resp := api.Do(http.MethodGet, "/mw-test")
	assert.Equal(t, http.StatusNoContent, resp.Code)

	_ = handlerCalled
}


func TestTestChainHelper(t *testing.T) {
	var order []string

	mw1 := func(ctx core.Context, next func(core.Context)) {
		order = append(order, "mw1")
		next(ctx)
	}
	mw2 := func(ctx core.Context, next func(core.Context)) {
		order = append(order, "mw2")
		next(ctx)
	}

	chain := core.Middlewares{mw1, mw2}
	handler := middleware.TestChain(chain, func(ctx core.Context) {
		order = append(order, "handler")
	})

	require.NotNil(t, handler)

	api := newAPI(t)
	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/chain-test",
		OperationID: "chain-test",
		Middlewares: chain,
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/chain-test")
	assert.Equal(t, http.StatusNoContent, resp.Code)
}


func TestGroupTransformers(t *testing.T) {
	api := newAPI(t)
	grp := middleware.NewGroup(api, "/tx")

	transformerRan := false
	grp.UseTransformer(func(ctx core.Context, status string, v any) (any, error) {
		transformerRan = true
		return v, nil
	})

	type Out struct {
		Body struct {
			Msg string `json:"msg"`
		}
	}

	neoma.Register(grp, core.Operation{
		Method:      http.MethodGet,
		Path:        "/hello",
		OperationID: "tx-hello",
	}, func(_ context.Context, _ *struct{}) (*Out, error) {
		o := &Out{}
		o.Body.Msg = "hi"
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/tx/hello")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.True(t, transformerRan)
}
