package openapi_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/errors"
	"github.com/MeGaNeKoS/neoma/neoma"
	"github.com/MeGaNeKoS/neoma/neomatest"
	"github.com/MeGaNeKoS/neoma/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)


func makeFilterSpec() *core.OpenAPI {
	return &core.OpenAPI{
		OpenAPI: core.OpenAPIVersion31,
		Info:    &core.Info{Title: "Test", Version: "1.0"},
		Paths: map[string]*core.PathItem{
			"/users": {
				Get:  &core.Operation{OperationID: "list-users", Tags: []string{"users"}},
				Post: &core.Operation{OperationID: "create-user", Tags: []string{"users", "admin"}},
			},
			"/items": {
				Get: &core.Operation{OperationID: "list-items", Tags: []string{"items"}},
			},
			"/admin": {
				Delete: &core.Operation{OperationID: "delete-all", Tags: []string{"admin"}},
			},
		},
		Tags: []*core.Tag{{Name: "users"}, {Name: "items"}, {Name: "admin"}},
	}
}

func TestFilterByTag(t *testing.T) {
	spec := makeFilterSpec()
	filtered := openapi.FilterByTag(spec, "users")

	assert.Contains(t, filtered.Paths, "/users")
	assert.NotContains(t, filtered.Paths, "/items")
	assert.NotContains(t, filtered.Paths, "/admin")

	pi := filtered.Paths["/users"]
	require.NotNil(t, pi)
	assert.NotNil(t, pi.Get)
	assert.NotNil(t, pi.Post)
}

func TestFilterByTagNoMatch(t *testing.T) {
	spec := makeFilterSpec()
	filtered := openapi.FilterByTag(spec, "nonexistent")
	assert.Empty(t, filtered.Paths)
}

func TestFilterExcludeTag(t *testing.T) {
	spec := makeFilterSpec()
	filtered := openapi.FilterExcludeTag(spec, "admin")

	if pi, ok := filtered.Paths["/users"]; ok {
		assert.NotNil(t, pi.Get)
		assert.Nil(t, pi.Post) // Post has admin tag so excluded
	}
	assert.Contains(t, filtered.Paths, "/items")
	assert.NotContains(t, filtered.Paths, "/admin")
}

func TestFilterExcludeTagNoMatch(t *testing.T) {
	spec := makeFilterSpec()
	filtered := openapi.FilterExcludeTag(spec, "nonexistent")
	assert.Len(t, filtered.Paths, 3)
}


func TestDefineErrorsWithDiscoveredErrors(t *testing.T) {
	registry := newRegistry()
	factory := errors.NewRFC9457Handler()

	op := &core.Operation{
		Method:      "POST",
		Path:        "/test",
		OperationID: "test",
		Errors:      []int{http.StatusBadRequest},
		Responses:   map[string]*core.Response{},
	}

	discovered := []core.DiscoveredError{
		{Status: http.StatusBadRequest, Title: "Invalid Email", Detail: "Email format is wrong"},
	}

	openapi.DefineErrors(op, registry, factory, discovered)

	resp400 := op.Responses["400"]
	require.NotNil(t, resp400)
	ct := factory.ErrorContentType("application/json")
	mt := resp400.Content[ct]
	require.NotNil(t, mt)
	assert.NotNil(t, mt.Example)
}

func TestDefineErrorsWithMultipleDiscoveredErrors(t *testing.T) {
	registry := newRegistry()
	factory := errors.NewRFC9457Handler()

	op := &core.Operation{
		Method:      "POST",
		Path:        "/test",
		OperationID: "test",
		Errors:      []int{http.StatusBadRequest},
		Responses:   map[string]*core.Response{},
	}

	discovered := []core.DiscoveredError{
		{Status: http.StatusBadRequest, Title: "Invalid Email", Detail: "Email format is wrong"},
		{Status: http.StatusBadRequest, Title: "Missing Name", Detail: "Name is required"},
	}

	openapi.DefineErrors(op, registry, factory, discovered)

	resp400 := op.Responses["400"]
	require.NotNil(t, resp400)
	ct := factory.ErrorContentType("application/json")
	mt := resp400.Content[ct]
	require.NotNil(t, mt)
	assert.NotNil(t, mt.Examples)
	assert.Len(t, mt.Examples, 2)
}


func TestRegisterSpecRoutes(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/test",
		OperationID: "get-test",
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/openapi.json")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "application/openapi+json", resp.Header().Get("Content-Type"))

	resp = api.Do(http.MethodGet, "/openapi.yaml")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "application/openapi+yaml", resp.Header().Get("Content-Type"))
}

func TestRegisterDocsRoute(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.Docs = core.DocsConfig{
		Path:    "/docs",
		Enabled: true,
	}
	_, api := neomatest.New(t, config)

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/test",
		OperationID: "get-test",
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/docs")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "text/html", resp.Header().Get("Content-Type"))
	assert.Contains(t, resp.Body.String(), "<!doctype html>")
}

func TestDocsProviderScalar(t *testing.T) {
	p := openapi.ScalarProvider{}
	html := p.Render("/openapi", "My API")
	assert.Contains(t, html, "My API")
	assert.Contains(t, html, "/openapi")
}

func TestDocsProviderScalarLocalJS(t *testing.T) {
	p := openapi.ScalarProvider{LocalJSPath: "/static/scalar.js"}
	html := p.Render("/openapi", "")
	assert.Contains(t, html, "/static/scalar.js")
	assert.Contains(t, html, "API Reference")
}

func TestDocsProviderStoplight(t *testing.T) {
	p := openapi.StoplightProvider{}
	html := p.Render("/openapi", "My API")
	assert.Contains(t, html, "My API")
}

func TestDocsProviderSwaggerUI(t *testing.T) {
	p := openapi.SwaggerUIProvider{}
	html := p.Render("/openapi", "")
	assert.Contains(t, html, "SwaggerUI")
}


func TestRegisterErrorDocRoutesNoBaseURI(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))
	resp := api.Do(http.MethodGet, "/errors/400")
	assert.NotEqual(t, http.StatusOK, resp.Code)
}


func TestDowngradeExclusiveMinimum(t *testing.T) {
	spec := &core.OpenAPI{
		OpenAPI: core.OpenAPIVersion31,
		Info:    &core.Info{Title: "Test", Version: "1.0"},
		Paths: map[string]*core.PathItem{
			"/test": {
				Get: &core.Operation{
					OperationID: "test",
					Responses: map[string]*core.Response{
						"200": {
							Description: "OK",
							Content: map[string]*core.MediaType{
								"application/json": {
									Schema: &core.Schema{
										Type:             "integer",
										ExclusiveMinimum: ptrFloat(0),
										ExclusiveMaximum: ptrFloat(100),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	b, err := openapi.Downgrade(spec)
	require.NoError(t, err)
	assert.Contains(t, string(b), `"exclusiveMinimum":true`)
	assert.Contains(t, string(b), `"exclusiveMaximum":true`)
}

func TestDowngradeExamples(t *testing.T) {
	spec := &core.OpenAPI{
		OpenAPI: core.OpenAPIVersion31,
		Info:    &core.Info{Title: "Test", Version: "1.0"},
		Paths: map[string]*core.PathItem{
			"/test": {
				Get: &core.Operation{
					OperationID: "test",
					Responses: map[string]*core.Response{
						"200": {
							Description: "OK",
							Content: map[string]*core.MediaType{
								"application/json": {
									Schema: &core.Schema{
										Type:     "string",
										Examples: []any{"hello", "world"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	b, err := openapi.Downgrade(spec)
	require.NoError(t, err)
	assert.Contains(t, string(b), `"example":"hello"`)
}

func TestDowngradeContentEncoding(t *testing.T) {
	spec := &core.OpenAPI{
		OpenAPI: core.OpenAPIVersion31,
		Info:    &core.Info{Title: "Test", Version: "1.0"},
		Paths: map[string]*core.PathItem{
			"/test": {
				Get: &core.Operation{
					OperationID: "test",
					Responses: map[string]*core.Response{
						"200": {
							Description: "OK",
							Content: map[string]*core.MediaType{
								"application/json": {
									Schema: &core.Schema{
										Type:            "string",
										ContentEncoding: "base64",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	b, err := openapi.Downgrade(spec)
	require.NoError(t, err)
	assert.Contains(t, string(b), `"format":"base64"`)
}


func TestGenerateInternalSpec(t *testing.T) {
	oapi := &core.OpenAPI{
		OpenAPI: core.OpenAPIVersion31,
		Info:    &core.Info{Title: "Test", Version: "1.0"},
		Paths: map[string]*core.PathItem{
			"/visible": {
				Get: &core.Operation{Method: "GET", Path: "/visible", OperationID: "get-visible"},
			},
		},
	}

	hiddenOps := []*core.Operation{
		{Method: "POST", Path: "/visible", OperationID: "post-visible", Hidden: true},
		{Method: "GET", Path: "/hidden", OperationID: "get-hidden", Hidden: true},
	}

	result, err := openapi.GenerateInternalSpec(oapi, hiddenOps)
	require.NoError(t, err)

	assert.Contains(t, result.Paths, "/visible")
	assert.Contains(t, result.Paths, "/hidden")
	require.NotNil(t, result.Paths["/visible"])
	assert.NotNil(t, result.Paths["/visible"].Get)
	assert.NotNil(t, result.Paths["/visible"].Post)

	require.NotNil(t, oapi.Paths["/visible"])
	assert.Nil(t, oapi.Paths["/visible"].Post)
	assert.NotContains(t, oapi.Paths, "/hidden")
}
