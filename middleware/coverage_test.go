package middleware_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/middleware"
	"github.com/MeGaNeKoS/neoma/neoma"
	"github.com/MeGaNeKoS/neoma/neomatest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)


func TestPrefixModifierSingle(t *testing.T) {
	modifier := middleware.PrefixModifier([]string{"/api"})
	require.NotNil(t, modifier)

	op := &core.Operation{Method: "GET", Path: "/users", OperationID: "list-users"}
	var results []*core.Operation
	modifier(op, func(o *core.Operation) {
		results = append(results, o)
	})

	require.Len(t, results, 1)
	assert.Equal(t, "/api/users", results[0].Path)
	assert.Equal(t, "list-users", results[0].OperationID) // Not modified for single prefix
}

func TestPrefixModifierMultiple(t *testing.T) {
	modifier := middleware.PrefixModifier([]string{"/v1", "/v2"})
	require.NotNil(t, modifier)

	op := &core.Operation{Method: "GET", Path: "/users", OperationID: "list-users", Tags: []string{"users"}}
	var results []*core.Operation
	modifier(op, func(o *core.Operation) {
		cp := *o
		results = append(results, &cp)
	})

	require.Len(t, results, 2)
	assert.Equal(t, "/v1/users", results[0].Path)
	assert.Contains(t, results[0].OperationID, "v1-")
	assert.Equal(t, "/v2/users", results[1].Path)
	assert.Contains(t, results[1].OperationID, "v2-")
}

func TestPrefixModifierEmpty(t *testing.T) {
	modifier := middleware.PrefixModifier([]string{""})
	op := &core.Operation{Method: "GET", Path: "/users", OperationID: "list-users"}
	var called bool
	modifier(op, func(o *core.Operation) {
		called = true
		assert.Equal(t, "/users", o.Path) // No prefix added
	})
	assert.True(t, called)
}


func TestNestedGroupMiddleware(t *testing.T) {
	api := newAPI(t)

	outerCalled := false
	innerCalled := false

	v1 := middleware.NewGroup(api, "/v1")
	v1.UseMiddleware(func(ctx core.Context, next func(core.Context)) {
		outerCalled = true
		next(ctx)
	})

	admin := v1.Group("/admin")
	admin.UseMiddleware(func(ctx core.Context, next func(core.Context)) {
		innerCalled = true
		next(ctx)
	})

	neoma.Register(admin, core.Operation{
		Method:      http.MethodGet,
		Path:        "/settings",
		OperationID: "admin-settings",
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/v1/admin/settings")
	assert.Equal(t, http.StatusNoContent, resp.Code)
	assert.True(t, outerCalled, "outer group middleware should have been called")
	assert.True(t, innerCalled, "inner group middleware should have been called")
}


func TestGroupTransformChain(t *testing.T) {
	api := newAPI(t)
	grp := middleware.NewGroup(api, "/tx")

	var transformCalled bool
	grp.UseTransformer(func(ctx core.Context, status string, v any) (any, error) {
		transformCalled = true
		return v, nil
	})

	type Out struct {
		Body struct {
			Msg string `json:"msg"`
		}
	}

	neoma.Register(grp, core.Operation{
		Method:      http.MethodGet,
		Path:        "/chain",
		OperationID: "tx-chain",
	}, func(_ context.Context, _ *struct{}) (*Out, error) {
		o := &Out{}
		o.Body.Msg = "hi"
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/tx/chain")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.True(t, transformCalled)
}


func TestGroupDocumentOperation(t *testing.T) {
	api := newAPI(t)
	grp := middleware.NewGroup(api, "/v1")

	neoma.Register(grp, core.Operation{
		Method:      http.MethodGet,
		Path:        "/documented",
		OperationID: "documented-op",
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	oapi := api.OpenAPI()
	assert.Contains(t, oapi.Paths, "/v1/documented")
}

func TestGroupDocumentOperationHidden(t *testing.T) {
	api := newAPI(t)
	grp := middleware.NewGroup(api, "/v1")

	neoma.Register(grp, core.Operation{
		Method:      http.MethodGet,
		Path:        "/hidden",
		OperationID: "hidden-op",
		Hidden:      true,
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	oapi := api.OpenAPI()
	assert.NotContains(t, oapi.Paths, "/v1/hidden")
}


func TestGroupNegotiate(t *testing.T) {
	api := newAPI(t)
	grp := middleware.NewGroup(api, "/v1")

	ct, err := grp.Negotiate("application/json")
	require.NoError(t, err)
	assert.Equal(t, "application/json", ct)
}


func TestTestBuilderNilMiddleware(t *testing.T) {
	builder := middleware.BuilderFunc(func(op *core.Operation) core.MiddlewareFunc {
		return nil // Builder returns nil
	})

	called := false
	handler := middleware.TestBuilder(builder, &core.Operation{}, func(ctx core.Context) {
		called = true
	})

	require.NotNil(t, handler)
	handler(nil)
	assert.True(t, called) // Should call handler directly since mw is nil
}


func TestConvenienceFunctionsWithGroup(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))
	grp := middleware.NewGroup(api, "/v1")

	neoma.Get[struct{}, struct{}](grp, "/items", func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/v1/items")
	assert.Equal(t, http.StatusNoContent, resp.Code)

	oapi := api.OpenAPI()
	assert.Contains(t, oapi.Paths, "/v1/items")
}


func TestGroupTransformError(t *testing.T) {
	api := newAPI(t)
	grp := middleware.NewGroup(api, "/txerr")

	errTransform := assert.AnError
	grp.UseTransformer(func(ctx core.Context, status string, v any) (any, error) {
		return nil, errTransform
	})

	type Out struct {
		Body struct {
			Msg string `json:"msg"`
		}
	}

	neoma.Register(grp, core.Operation{
		Method:      http.MethodGet,
		Path:        "/fail",
		OperationID: "txerr-fail",
	}, func(_ context.Context, _ *struct{}) (*Out, error) {
		o := &Out{}
		o.Body.Msg = "should fail"
		return o, nil
	})

	assert.Panics(t, func() {
		api.Do(http.MethodGet, "/txerr/fail")
	})
}


func TestGroupTransformParentError(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	grp := middleware.NewGroup(api, "/txperr")

	grpTransformCalled := false
	grp.UseTransformer(func(ctx core.Context, status string, v any) (any, error) {
		grpTransformCalled = true
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
		OperationID: "txperr-hello",
	}, func(_ context.Context, _ *struct{}) (*Out, error) {
		o := &Out{}
		o.Body.Msg = "hi"
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/txperr/hello")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.True(t, grpTransformCalled)
}


func TestGroupUnmarshal(t *testing.T) {
	api := newAPI(t)
	grp := middleware.NewGroup(api, "/v1")

	type Payload struct {
		Name string `json:"name"`
	}

	data := []byte(`{"name":"test"}`)
	var p Payload
	err := grp.Unmarshal("application/json", data, &p)
	require.NoError(t, err)
	assert.Equal(t, "test", p.Name)
}


func TestGroupMarshal(t *testing.T) {
	api := newAPI(t)
	grp := middleware.NewGroup(api, "/v1")

	var buf strings.Builder
	err := grp.Marshal(&buf, "application/json", map[string]string{"key": "val"})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), `"key"`)
}


func TestConvenienceFunctionsWithGroupSingleSegment(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))
	grp := middleware.NewGroup(api, "/v1")

	neoma.Get[struct{}, struct{}](grp, "/x", func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/v1/x")
	assert.Equal(t, http.StatusNoContent, resp.Code)

	oapi := api.OpenAPI()
	pi := oapi.Paths["/v1/x"]
	require.NotNil(t, pi)
	require.NotNil(t, pi.Get)
	assert.NotEmpty(t, pi.Get.Summary)
}


func TestTestMiddlewareDirectInvocation(t *testing.T) {
	var order []string

	mw := func(ctx core.Context, next func(core.Context)) {
		order = append(order, "middleware")
		next(ctx)
	}

	handler := middleware.TestMiddleware(mw, func(ctx core.Context) {
		order = append(order, "handler")
	})

	require.NotNil(t, handler)

	handler(nil)

	require.Len(t, order, 2)
	assert.Equal(t, "middleware", order[0])
	assert.Equal(t, "handler", order[1])
}


func TestTestBuilderWithMiddleware(t *testing.T) {
	var mwCalled, handlerCalled bool

	builder := middleware.BuilderFunc(func(op *core.Operation) core.MiddlewareFunc {
		return func(ctx core.Context, next func(core.Context)) {
			mwCalled = true
			next(ctx)
		}
	})

	handler := middleware.TestBuilder(builder, &core.Operation{}, func(ctx core.Context) {
		handlerCalled = true
	})

	require.NotNil(t, handler)
	handler(nil)
	assert.True(t, mwCalled)
	assert.True(t, handlerCalled)
}
