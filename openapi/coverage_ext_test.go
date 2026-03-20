package openapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/errors"
	"github.com/MeGaNeKoS/neoma/neoma"
	"github.com/MeGaNeKoS/neoma/neomatest"
	"github.com/MeGaNeKoS/neoma/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)


func TestSpecRoutes30Version(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.OpenAPIVersion = core.OpenAPIVersion30

	_, api := neomatest.New(t, config)

	neoma.Get[struct{}, struct{}](api, "/test", func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/openapi.json")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), core.OpenAPIVersion30)

	resp = api.Do(http.MethodGet, "/openapi.yaml")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), core.OpenAPIVersion30)
}

func TestSpecRoutes31Version(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.OpenAPIVersion = core.OpenAPIVersion31

	_, api := neomatest.New(t, config)

	neoma.Get[struct{}, struct{}](api, "/test", func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/openapi.json")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), core.OpenAPIVersion31)

	resp = api.Do(http.MethodGet, "/openapi.yaml")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), core.OpenAPIVersion31)
}

func TestSpecRoutes32Version(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.OpenAPIVersion = core.OpenAPIVersion32

	_, api := neomatest.New(t, config)

	neoma.Get[struct{}, struct{}](api, "/test", func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/openapi.json")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), core.OpenAPIVersion32)

	resp = api.Do(http.MethodGet, "/openapi.yaml")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), core.OpenAPIVersion32)
}


func TestSchemaRoute(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	_, api := neomatest.New(t, config)

	type Output struct {
		Body struct {
			Name string `json:"name"`
		}
	}

	neoma.Get[struct{}, Output](api, "/items", func(_ context.Context, _ *struct{}) (*Output, error) {
		return nil, nil
	})

	specResp := api.Do(http.MethodGet, "/openapi.json")
	require.Equal(t, http.StatusOK, specResp.Code)

	var spec map[string]any
	require.NoError(t, json.Unmarshal(specResp.Body.Bytes(), &spec))

	components, ok := spec["components"].(map[string]any)
	if !ok {
		t.Skip("no components in spec")
	}
	schemas, ok := components["schemas"].(map[string]any)
	if !ok {
		t.Skip("no schemas in spec")
	}

	for name := range schemas {
		resp := api.Do(http.MethodGet, "/schemas/" + name + ".json")
		assert.Equal(t, http.StatusOK, resp.Code, "schema %s should be accessible", name)
		assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
	}
}


func TestDocsRouteStoplightDefault(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.Docs = core.DocsConfig{
		Path:    "/docs",
		Enabled: true,
	}
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/docs")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "text/html", resp.Header().Get("Content-Type"))
	assert.Contains(t, resp.Body.String(), "elements-api")
	assert.NotEmpty(t, resp.Header().Get("Content-Security-Policy"))
}


func TestDocsRouteScalar(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.Docs = core.DocsConfig{
		Path:     "/docs",
		Provider: openapi.ScalarProvider{},
		Enabled:  true,
	}
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/docs")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "api-reference")
	assert.Contains(t, resp.Header().Get("Content-Security-Policy"), "script-src")
}


func TestDocsRouteScalarLocalJS(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.Docs = core.DocsConfig{
		Path:     "/docs",
		Provider: openapi.ScalarProvider{LocalJSPath: "/static/scalar.js"},
		Enabled:  true,
	}
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/docs")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "/static/scalar.js")
	assert.Contains(t, resp.Header().Get("Content-Security-Policy"), "'self'")
}


func TestDocsRouteSwaggerUI(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.Docs = core.DocsConfig{
		Path:     "/docs",
		Provider: openapi.SwaggerUIProvider{},
		Enabled:  true,
	}
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/docs")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "swagger-ui")
	assert.NotEmpty(t, resp.Header().Get("Content-Security-Policy"))
}


type customDocsProvider struct{}

func (p customDocsProvider) Render(specURL string, title string) string {
	return "<html><body>Custom Docs: " + title + " " + specURL + "</body></html>"
}

func TestDocsRouteCustomProvider(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.Docs = core.DocsConfig{
		Path:     "/docs",
		Provider: customDocsProvider{},
		Enabled:  true,
	}
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/docs")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "Custom Docs")
	csp := resp.Header().Get("Content-Security-Policy")
	assert.Contains(t, csp, "sandbox")
}


func TestDocsRouteEmptyPath(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.Docs = core.DocsConfig{
		Path:    "",
		Enabled: true,
	}
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/docs")
	assert.NotEqual(t, http.StatusOK, resp.Code)
}


func TestDocsRouteWithServerPrefix(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.Servers = []*core.Server{
		{URL: "/api/v1"},
	}
	config.Docs = core.DocsConfig{
		Path:     "/docs",
		Provider: openapi.ScalarProvider{},
		Enabled:  true,
	}
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/docs")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "/api/v1/openapi")
}


func TestGetAPIPrefixRelative(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.Servers = []*core.Server{
		{URL: "/api"},
	}
	config.Docs = core.DocsConfig{
		Path:     "/docs",
		Provider: openapi.ScalarProvider{},
		Enabled:  true,
	}
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/docs")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "/api/openapi")
}

func TestGetAPIPrefixAbsolute(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.Servers = []*core.Server{
		{URL: "https://example.com/api/v2"},
	}
	config.Docs = core.DocsConfig{
		Path:     "/docs",
		Provider: openapi.ScalarProvider{},
		Enabled:  true,
	}
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/docs")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "/api/v2/openapi")
}

func TestGetAPIPrefixEmptyServer(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.Servers = []*core.Server{
		{URL: ""},
	}
	config.Docs = core.DocsConfig{
		Path:     "/docs",
		Provider: openapi.ScalarProvider{},
		Enabled:  true,
	}
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/docs")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "/openapi")
}

func TestGetAPIPrefixNoPathServer(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.Servers = []*core.Server{
		{URL: "https://example.com"},
	}
	config.Docs = core.DocsConfig{
		Path:     "/docs",
		Provider: openapi.ScalarProvider{},
		Enabled:  true,
	}
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/docs")
	assert.Equal(t, http.StatusOK, resp.Code)
}


func TestDocsRouteWithMiddleware(t *testing.T) {
	authMW := func(ctx core.Context, next func(core.Context)) {
		if ctx.Header("X-Docs-Auth") != "valid" {
			ctx.SetStatus(http.StatusUnauthorized)
			return
		}
		next(ctx)
	}

	config := neoma.DefaultConfig("Test", "1.0.0")
	config.Docs = core.DocsConfig{
		Path:        "/docs",
		Provider:    openapi.StoplightProvider{},
		Enabled:     true,
		Middlewares: core.Middlewares{authMW},
	}
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/docs")
	assert.Equal(t, http.StatusUnauthorized, resp.Code)

	resp = api.Do(http.MethodGet, "/docs", "X-Docs-Auth: valid")
	assert.Equal(t, http.StatusOK, resp.Code)
}


func TestErrorDocRoutesHTML(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.ErrorHandler = errors.NewRFC9457HandlerWithConfig("/errors", nil)
	_, api := neomatest.New(t, config)

	neoma.Register(api, core.Operation{
		Method:      http.MethodPost,
		Path:        "/items",
		OperationID: "create-item",
		Errors:      []int{400, 422, 500},
	}, func(_ context.Context, _ *struct{ Body struct{ Name string `json:"name"` } }) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/errors/400")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Header().Get("Content-Type"), "text/html")
	body := resp.Body.String()
	assert.Contains(t, body, "Bad Request")
	assert.Contains(t, body, "400")
	assert.Contains(t, body, "Causes and Fixes")
	assert.Contains(t, body, "POST")

	resp = api.Do(http.MethodGet, "/errors/422")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "Unprocessable Entity")

	resp = api.Do(http.MethodGet, "/errors/500")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "Internal Server Error")
}

func TestErrorDocRoutesJSON(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.ErrorHandler = errors.NewRFC9457HandlerWithConfig("/errors", nil)
	_, api := neomatest.New(t, config)

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/test",
		OperationID: "get-test",
		Errors:      []int{404},
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/errors/404", "Accept: application/json")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Header().Get("Content-Type"), "application/json")

	var doc map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &doc))
	assert.Equal(t, "Not Found", doc["title"])
	assert.InDelta(t, 404.0, doc["status"], 0.001)
	assert.NotEmpty(t, doc["description"])
	assert.NotNil(t, doc["entries"])
	assert.NotNil(t, doc["endpoints"])
	assert.NotNil(t, doc["example"])
}

func TestErrorDocRoutesUnknownCode(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.ErrorHandler = errors.NewRFC9457HandlerWithConfig("/errors", nil)
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/errors/abc")
	assert.Equal(t, http.StatusNotFound, resp.Code)

	resp = api.Do(http.MethodGet, "/errors/999")
	assert.Equal(t, http.StatusNotFound, resp.Code)
}

func TestErrorDocRoutesKnownButUndocumented(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.ErrorHandler = errors.NewRFC9457HandlerWithConfig("/errors", nil)
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/errors/401")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "Unauthorized")
}

func TestErrorDocRoutesAllKnownCodes(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.ErrorHandler = errors.NewRFC9457HandlerWithConfig("/errors", nil)
	_, api := neomatest.New(t, config)

	codes := []int{400, 401, 403, 404, 409, 412, 422, 429, 500, 502, 503}

	for _, code := range codes {
		codeStr := func() string { b, _ := json.Marshal(code); return string(b) }()
		resp := api.Do(http.MethodGet, "/errors/" + strings.TrimSpace(codeStr))
		assert.Equal(t, http.StatusOK, resp.Code, "error doc for %d should exist", code)
	}
}

func TestErrorDocRoutesHTMLWithApiTitle(t *testing.T) {
	config := neoma.DefaultConfig("My Cool API", "1.0.0")
	config.ErrorHandler = errors.NewRFC9457HandlerWithConfig("/errors", nil)
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/errors/400")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "My Cool API")
}

func TestErrorDocRoutesStatusHelpers(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.ErrorHandler = errors.NewRFC9457HandlerWithConfig("/errors", nil)
	_, api := neomatest.New(t, config)

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/test",
		OperationID: "test-colors",
		Errors:      []int{400, 500},
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/errors/400")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "Client Error")

	resp = api.Do(http.MethodGet, "/errors/500")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "Server Error")
}

func TestErrorDocRoutesHTMLGenericCode(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.ErrorHandler = errors.NewRFC9457HandlerWithConfig("/errors", nil)
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/errors/418")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "No detailed documentation")
}

func TestErrorDocRoutesNoEndpoints(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.ErrorHandler = errors.NewRFC9457HandlerWithConfig("/errors", nil)
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/errors/403", "Accept: application/json")
	assert.Equal(t, http.StatusOK, resp.Code)

	var doc map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &doc))
	assert.Nil(t, doc["endpoints"])
}


func TestInternalSpecWithHiddenParams(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.InternalSpec = core.InternalSpecConfig{
		Path:    "/internal/openapi",
		Enabled: true,
	}
	_, api := neomatest.New(t, config)

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/with-hidden-param",
		OperationID: "with-hidden-param",
		HiddenParameters: []*core.Param{
			{
				Name: "X-Internal-Debug",
				In:   "header",
				Schema: &core.Schema{
					Type: "string",
				},
			},
		},
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/openapi.json")
	require.Equal(t, http.StatusOK, resp.Code)
	assert.NotContains(t, resp.Body.String(), "X-Internal-Debug")

	resp = api.Do(http.MethodGet, "/internal/openapi.json")
	require.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "X-Internal-Debug")
}


func TestInternalSpec30Formats(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.InternalSpec = core.InternalSpecConfig{
		Path:    "/internal/openapi",
		Enabled: true,
	}
	_, api := neomatest.New(t, config)

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/test",
		OperationID: "test",
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/internal/openapi.json")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.NotEmpty(t, resp.Body.String())

	resp = api.Do(http.MethodGet, "/internal/openapi.yaml")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.NotEmpty(t, resp.Body.String())
}


func TestInternalDocsRoute(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.InternalSpec = core.InternalSpecConfig{
		Path:     "/internal/openapi",
		DocsPath: "/internal/docs",
		Enabled:  true,
	}
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/internal/docs")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Header().Get("Content-Type"), "text/html")
	assert.Contains(t, resp.Body.String(), "/internal/openapi")
	assert.Contains(t, resp.Body.String(), "(Internal)")
}

func TestInternalDocsRouteDefaultPath(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.InternalSpec = core.InternalSpecConfig{
		Path:    "/internal/openapi",
		Enabled: true,
	}
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/internal/docs")
	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestInternalDocsRouteCustomProvider(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.Docs = core.DocsConfig{
		Path:     "/docs",
		Provider: customDocsProvider{},
		Enabled:  true,
	}
	config.InternalSpec = core.InternalSpecConfig{
		Path:     "/internal/openapi",
		DocsPath: "/internal/docs",
		Enabled:  true,
	}
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/internal/docs")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "Custom Docs")
	assert.Contains(t, resp.Header().Get("Content-Security-Policy"), "sandbox")
}


func TestInternalSpecWithMiddleware(t *testing.T) {
	authMW := func(ctx core.Context, next func(core.Context)) {
		if ctx.Header("Authorization") != "Bearer internal" {
			ctx.SetStatus(http.StatusForbidden)
			return
		}
		next(ctx)
	}

	config := neoma.DefaultConfig("Test", "1.0.0")
	config.InternalSpec = core.InternalSpecConfig{
		Path:        "/internal/openapi",
		Enabled:     true,
		Middlewares: core.Middlewares{authMW},
	}
	_, api := neomatest.New(t, config)

	neoma.Get[struct{}, struct{}](api, "/test", func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/internal/openapi.json")
	assert.Equal(t, http.StatusForbidden, resp.Code)

	resp = api.Do(http.MethodGet, "/internal/openapi.json", "Authorization: Bearer internal")
	assert.Equal(t, http.StatusOK, resp.Code)
}


func TestGenerateInternalSpecAllMethods(t *testing.T) {
	oapi := &core.OpenAPI{
		OpenAPI: core.OpenAPIVersion31,
		Info:    &core.Info{Title: "Test", Version: "1.0"},
		Paths:   map[string]*core.PathItem{},
	}

	hiddenOps := []*core.Operation{
		{Method: "GET", Path: "/test", OperationID: "get-test"},
		{Method: "PUT", Path: "/test", OperationID: "put-test"},
		{Method: "POST", Path: "/test", OperationID: "post-test"},
		{Method: "DELETE", Path: "/test", OperationID: "delete-test"},
		{Method: "OPTIONS", Path: "/test", OperationID: "options-test"},
		{Method: "HEAD", Path: "/test", OperationID: "head-test"},
		{Method: "PATCH", Path: "/test", OperationID: "patch-test"},
		{Method: "TRACE", Path: "/test", OperationID: "trace-test"},
	}

	result, err := openapi.GenerateInternalSpec(oapi, hiddenOps)
	require.NoError(t, err)

	pi := result.Paths["/test"]
	require.NotNil(t, pi)
	assert.NotNil(t, pi.Get)
	assert.NotNil(t, pi.Put)
	assert.NotNil(t, pi.Post)
	assert.NotNil(t, pi.Delete)
	assert.NotNil(t, pi.Options)
	assert.NotNil(t, pi.Head)
	assert.NotNil(t, pi.Patch)
	assert.NotNil(t, pi.Trace)
}


func TestGenerateInternalSpecJSONNoHidden(t *testing.T) {
	oapi := &core.OpenAPI{
		OpenAPI: core.OpenAPIVersion31,
		Info:    &core.Info{Title: "Test", Version: "1.0"},
		Paths: map[string]*core.PathItem{
			"/test": {
				Get: &core.Operation{OperationID: "get-test"},
			},
		},
	}

	data, err := openapi.GenerateInternalSpecJSON(oapi, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}


func TestDowngradeOctetStream(t *testing.T) {
	spec := &core.OpenAPI{
		OpenAPI: core.OpenAPIVersion31,
		Info:    &core.Info{Title: "Test", Version: "1.0"},
		Paths: map[string]*core.PathItem{
			"/upload": {
				Post: &core.Operation{
					OperationID: "upload",
					RequestBody: &core.RequestBody{
						Content: map[string]*core.MediaType{
							"application/octet-stream": {},
						},
					},
					Responses: map[string]*core.Response{
						"200": {Description: "OK"},
					},
				},
			},
		},
	}

	b, err := openapi.Downgrade(spec)
	require.NoError(t, err)
	assert.Contains(t, string(b), `"format":"binary"`)
}


func TestDowngradeContentEncodingNonBase64(t *testing.T) {
	spec := &core.OpenAPI{
		OpenAPI: core.OpenAPIVersion31,
		Info:    &core.Info{Title: "Test", Version: "1.0"},
		Paths: map[string]*core.PathItem{
			"/test": {
				Get: &core.Operation{
					OperationID: "test-ce",
					Responses: map[string]*core.Response{
						"200": {
							Description: "OK",
							Content: map[string]*core.MediaType{
								"application/json": {
									Schema: &core.Schema{
										Type:            "string",
										ContentEncoding: "gzip",
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
	assert.Contains(t, string(b), "x-contentEncoding")
}


func TestDowngradeTypeArrayWithNull(t *testing.T) {
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
										Nullable: true,
										Type:     "string",
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
	assert.Contains(t, string(b), `"nullable":true`)
}


func TestDowngradeSingleExample(t *testing.T) {
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
										Examples: []any{"only-one"},
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
	assert.Contains(t, string(b), `"example":"only-one"`)
	assert.NotContains(t, string(b), `"examples"`)
}


func TestFilterByTagWithAllMethods(t *testing.T) {
	spec := &core.OpenAPI{
		OpenAPI: core.OpenAPIVersion31,
		Info:    &core.Info{Title: "Test", Version: "1.0"},
		Paths: map[string]*core.PathItem{
			"/test": {
				Get:     &core.Operation{OperationID: "get", Tags: []string{"a"}},
				Put:     &core.Operation{OperationID: "put", Tags: []string{"b"}},
				Post:    &core.Operation{OperationID: "post", Tags: []string{"a"}},
				Delete:  &core.Operation{OperationID: "delete", Tags: []string{"b"}},
				Options: &core.Operation{OperationID: "options", Tags: []string{"a"}},
				Head:    &core.Operation{OperationID: "head", Tags: []string{"b"}},
				Patch:   &core.Operation{OperationID: "patch", Tags: []string{"a"}},
				Trace:   &core.Operation{OperationID: "trace", Tags: []string{"b"}},
			},
		},
	}

	filtered := openapi.FilterByTag(spec, "a")
	pi := filtered.Paths["/test"]
	require.NotNil(t, pi)
	assert.NotNil(t, pi.Get)
	assert.Nil(t, pi.Put)
	assert.NotNil(t, pi.Post)
	assert.Nil(t, pi.Delete)
	assert.NotNil(t, pi.Options)
	assert.Nil(t, pi.Head)
	assert.NotNil(t, pi.Patch)
	assert.Nil(t, pi.Trace)
}


func TestFilterWithNilTags(t *testing.T) {
	spec := &core.OpenAPI{
		OpenAPI: core.OpenAPIVersion31,
		Info:    &core.Info{Title: "Test", Version: "1.0"},
		Paths: map[string]*core.PathItem{
			"/test": {
				Get: &core.Operation{OperationID: "get"},
			},
		},
	}

	filtered := openapi.FilterByTag(spec, "sometag")
	assert.Empty(t, filtered.Paths)

	filtered = openapi.FilterExcludeTag(spec, "sometag")
	assert.Len(t, filtered.Paths, 1)
}


func TestDefineErrorsDiscoveredTitleCollision(t *testing.T) {
	registry := newRegistry()
	factory := errors.NewRFC9457Handler()

	op := &core.Operation{
		Method:      "POST",
		Path:        "/test",
		OperationID: "test",
		Errors:      []int{400},
		Responses:   map[string]*core.Response{},
	}

	discovered := []core.DiscoveredError{
		{Status: 400, Title: "Bad Request", Detail: "Email is invalid"},
		{Status: 400, Title: "Bad Request", Detail: "Name is too short"},
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


func TestDefineErrorsDiscoveredEmptyTitle(t *testing.T) {
	registry := newRegistry()
	factory := errors.NewRFC9457Handler()

	op := &core.Operation{
		Method:      "POST",
		Path:        "/test",
		OperationID: "test",
		Errors:      []int{400},
		Responses:   map[string]*core.Response{},
	}

	discovered := []core.DiscoveredError{
		{Status: 400, Title: "", Detail: "Something wrong"},
		{Status: 400, Title: "", Detail: "Another thing wrong"},
	}

	openapi.DefineErrors(op, registry, factory, discovered)

	resp400 := op.Responses["400"]
	require.NotNil(t, resp400)
	ct := factory.ErrorContentType("application/json")
	mt := resp400.Content[ct]
	require.NotNil(t, mt)
	assert.NotNil(t, mt.Examples)
}


func TestFindEndpointsNilSpec(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.ErrorHandler = errors.NewRFC9457HandlerWithConfig("/errors", nil)
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/errors/400", "Accept: application/json")
	assert.Equal(t, http.StatusOK, resp.Code)

	var doc map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &doc))
	assert.Nil(t, doc["endpoints"])
}


func TestErrorDocRoutesNoBaseURI(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/errors/400")
	assert.NotEqual(t, http.StatusOK, resp.Code)
}


func TestDowngradeExclusiveMax(t *testing.T) {
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
										Type:             "number",
										ExclusiveMaximum: ptrFloat(10),
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
	assert.Contains(t, string(b), `"exclusiveMaximum":true`)
	assert.Contains(t, string(b), `"maximum":10`)
}


func TestSpecMiddleware(t *testing.T) {
	authMW := func(ctx core.Context, next func(core.Context)) {
		if ctx.Header("X-Spec-Auth") != "valid" {
			ctx.SetStatus(http.StatusForbidden)
			return
		}
		next(ctx)
	}

	config := neoma.DefaultConfig("Test", "1.0.0")
	config.SpecMiddlewares = core.Middlewares{authMW}
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/openapi.json")
	assert.Equal(t, http.StatusForbidden, resp.Code)

	resp = api.Do(http.MethodGet, "/openapi.json", "X-Spec-Auth: valid")
	assert.Equal(t, http.StatusOK, resp.Code)
}


func TestInternalDocsWithServerPrefix(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.Servers = []*core.Server{
		{URL: "/api/v1"},
	}
	config.InternalSpec = core.InternalSpecConfig{
		Path:     "/internal/openapi",
		DocsPath: "/internal/docs",
		Enabled:  true,
	}
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/internal/docs")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "/api/v1/internal/openapi")
}


func TestInternalSpecNoHiddenProvider(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.InternalSpec = core.InternalSpecConfig{
		Path:    "/internal/openapi",
		Enabled: true,
	}
	_, api := neomatest.New(t, config)

	neoma.Get[struct{}, struct{}](api, "/visible", func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/internal/openapi.json")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "/visible")
}


func TestStoplightProviderCSP(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.Docs = core.DocsConfig{
		Path:     "/docs",
		Provider: openapi.StoplightProvider{},
		Enabled:  true,
	}
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/docs")
	assert.Equal(t, http.StatusOK, resp.Code)
	csp := resp.Header().Get("Content-Security-Policy")
	assert.Contains(t, csp, "stoplight")
}


func TestSwaggerUIProviderCSP(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.Docs = core.DocsConfig{
		Path:     "/docs",
		Provider: openapi.SwaggerUIProvider{},
		Enabled:  true,
	}
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/docs")
	assert.Equal(t, http.StatusOK, resp.Code)
	csp := resp.Header().Get("Content-Security-Policy")
	assert.Contains(t, csp, "swagger-ui")
}


func TestStoplightProviderEmptyTitle(t *testing.T) {
	p := openapi.StoplightProvider{}
	html := p.Render("/openapi", "")
	assert.Contains(t, html, "Elements in HTML")
}


func TestSwaggerUIProviderWithTitle(t *testing.T) {
	p := openapi.SwaggerUIProvider{}
	html := p.Render("/openapi", "My API")
	assert.Contains(t, html, "My API")
}


func TestScalarProviderEmptyTitle(t *testing.T) {
	p := openapi.ScalarProvider{}
	html := p.Render("/openapi", "")
	assert.Contains(t, html, "API Reference")
}


func TestErrorDocMultipleEndpoints(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.ErrorHandler = errors.NewRFC9457HandlerWithConfig("/errors", nil)
	_, api := neomatest.New(t, config)

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/items",
		OperationID: "list-items",
		Errors:      []int{400},
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	neoma.Register(api, core.Operation{
		Method:      http.MethodPost,
		Path:        "/items",
		OperationID: "create-item",
		Errors:      []int{400, 422},
	}, func(_ context.Context, _ *struct{ Body struct{ Name string `json:"name"` } }) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/errors/400", "Accept: application/json")
	assert.Equal(t, http.StatusOK, resp.Code)

	var doc map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &doc))
	endpoints, ok := doc["endpoints"].([]any)
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(endpoints), 2)
}


func TestErrorDocHTMLWithExample(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.ErrorHandler = errors.NewRFC9457HandlerWithConfig("/errors", nil)
	_, api := neomatest.New(t, config)

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/items/{id}",
		OperationID: "get-item",
		Errors:      []int{404},
	}, func(_ context.Context, _ *struct{ ID string `path:"id"` }) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/errors/404")
	assert.Equal(t, http.StatusOK, resp.Code)
	body := resp.Body.String()
	assert.Contains(t, body, "Example Response")
	assert.Contains(t, body, "Debug with cURL")
	assert.Contains(t, body, "RFC")
}


func TestErrorDoc5xxBadgeAndCategory(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.ErrorHandler = errors.NewRFC9457HandlerWithConfig("/errors", nil)
	_, api := neomatest.New(t, config)

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/test",
		OperationID: "test-5xx",
		Errors:      []int{502},
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/errors/502")
	assert.Equal(t, http.StatusOK, resp.Code)
	body := resp.Body.String()
	assert.Contains(t, body, "badge-error", "5xx should use badge-error CSS class")
	assert.Contains(t, body, "Server Error", "5xx should show Server Error category")
	assert.Contains(t, body, "#dc2626", "5xx should use the error color")
}


func TestErrorDoc2xxBadgeAndCategory(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.ErrorHandler = errors.NewRFC9457HandlerWithConfig("/errors", nil)
	config.ErrorDocs = map[int]core.ErrorDoc{
		200: {
			Title:       "OK",
			Description: "The request succeeded.",
		},
	}
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/errors/200")
	assert.Equal(t, http.StatusOK, resp.Code)
	body := resp.Body.String()
	assert.Contains(t, body, "badge-info", "2xx should use badge-info CSS class")
	assert.Contains(t, body, "Info", "2xx should show Info category")
	assert.Contains(t, body, "#6b7280", "2xx should use the info color")
}


func TestErrorDocUnknownCodeWithValidStatus(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.ErrorHandler = errors.NewRFC9457HandlerWithConfig("/errors", nil)
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/errors/307")
	assert.Equal(t, http.StatusOK, resp.Code)
	body := resp.Body.String()
	assert.Contains(t, body, "Temporary Redirect")
	assert.Contains(t, body, "No detailed documentation")
}


func TestGetAPIPrefixFullURLWithPath(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.Servers = []*core.Server{
		{URL: "https://api.example.com/v2"},
	}
	config.Docs = core.DocsConfig{
		Path:     "/docs",
		Provider: openapi.ScalarProvider{},
		Enabled:  true,
	}
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/docs")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "/v2/openapi")
}


func TestGetAPIPrefixRelativeURL(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.Servers = []*core.Server{
		{URL: "relative/path"},
	}
	config.Docs = core.DocsConfig{
		Path:     "/docs",
		Provider: openapi.ScalarProvider{},
		Enabled:  true,
	}
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/docs")
	assert.Equal(t, http.StatusOK, resp.Code)
	// A relative URL without a leading slash and without a host should not
	// produce a prefix. The spec URL should be just /openapi (no prefix).
	body := resp.Body.String()
	assert.NotContains(t, body, "relative/path/openapi")
}


func TestErrorDocCustomHTML(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.ErrorHandler = errors.NewRFC9457HandlerWithConfig("/errors", nil)
	config.ErrorDocs = map[int]core.ErrorDoc{
		400: {
			Title:       "Bad Request",
			Description: "Custom override",
			HTML:        "<html><body>Custom 400 page</body></html>",
		},
	}
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/errors/400")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "Custom 400 page")
}


func TestErrorDoc5xxJSON(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.ErrorHandler = errors.NewRFC9457HandlerWithConfig("/errors", nil)
	_, api := neomatest.New(t, config)

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/test",
		OperationID: "test-503",
		Errors:      []int{503},
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/errors/503", "Accept: application/json")
	assert.Equal(t, http.StatusOK, resp.Code)

	var doc map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &doc))
	assert.Equal(t, "Service Unavailable", doc["title"])
	assert.InDelta(t, 503.0, doc["status"], 0.001)
}


func TestDefineErrorsWithManualExamples(t *testing.T) {
	registry := newRegistry()
	factory := errors.NewRFC9457Handler()

	op := &core.Operation{
		Method:      "POST",
		Path:        "/test",
		OperationID: "test",
		Errors:      []int{400},
		Responses:   map[string]*core.Response{},
		ErrorExamples: map[int]any{
			400: map[string]string{"custom": "example"},
		},
	}

	openapi.DefineErrors(op, registry, factory)

	resp400 := op.Responses["400"]
	require.NotNil(t, resp400)
	ct := factory.ErrorContentType("application/json")
	mt := resp400.Content[ct]
	require.NotNil(t, mt)
	assert.NotNil(t, mt.Example)
}


func TestDefineErrorsNilResponses(t *testing.T) {
	registry := newRegistry()
	factory := errors.NewRFC9457Handler()

	op := &core.Operation{
		Method:      "GET",
		Path:        "/test",
		OperationID: "test",
		Errors:      []int{404},
		Responses:   nil, // nil Responses map
	}

	openapi.DefineErrors(op, registry, factory)

	require.NotNil(t, op.Responses)
	assert.NotNil(t, op.Responses["404"])
}


func TestErrorDocHTMLNoTitle(t *testing.T) {
	config := neoma.DefaultConfig("", "1.0.0")
	config.ErrorHandler = errors.NewRFC9457HandlerWithConfig("/errors", nil)
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/errors/400")
	assert.Equal(t, http.StatusOK, resp.Code)
	body := resp.Body.String()
	assert.Contains(t, body, "400")
	assert.Contains(t, body, "Bad Request")
}


func TestErrorDocHTMLDefaultDocsPath(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.ErrorHandler = errors.NewRFC9457HandlerWithConfig("/errors", nil)
	config.Docs = core.DocsConfig{} // No docs path set
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/errors/400")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "/public/docs")
}


func TestInternalSpecYAMLWithHiddenOps(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.InternalSpec = core.InternalSpecConfig{
		Path:    "/internal/openapi",
		Enabled: true,
	}
	_, api := neomatest.New(t, config)

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/visible",
		OperationID: "get-visible",
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/hidden",
		OperationID: "get-hidden",
		Hidden:      true,
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/internal/openapi.yaml")
	assert.Equal(t, http.StatusOK, resp.Code)
	body := resp.Body.String()
	assert.Contains(t, body, "/visible")
	assert.Contains(t, body, "/hidden")
}


func TestInternalSpecJSONWithHiddenSchemaProps(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.InternalSpec = core.InternalSpecConfig{
		Path:    "/internal/openapi",
		Enabled: true,
	}
	_, api := neomatest.New(t, config)

	type Output struct {
		Body struct {
			Name   string `json:"name"`
			Secret string `json:"secret" hidden:"true"`
		}
	}

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/items",
		OperationID: "get-items",
	}, func(_ context.Context, _ *struct{}) (*Output, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/openapi.json")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.NotContains(t, resp.Body.String(), `"secret"`)

	resp = api.Do(http.MethodGet, "/internal/openapi.json")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), `"secret"`)
}


func TestErrorDocSamePathMultipleMethods(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.ErrorHandler = errors.NewRFC9457HandlerWithConfig("/errors", nil)
	_, api := neomatest.New(t, config)

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/things",
		OperationID: "list-things",
		Errors:      []int{400},
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	neoma.Register(api, core.Operation{
		Method:      http.MethodDelete,
		Path:        "/things",
		OperationID: "delete-things",
		Errors:      []int{400},
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/errors/400", "Accept: application/json")
	assert.Equal(t, http.StatusOK, resp.Code)

	var doc map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &doc))
	endpoints, ok := doc["endpoints"].([]any)
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(endpoints), 2)

	if len(endpoints) >= 2 {
		first := endpoints[0].(string)
		second := endpoints[1].(string)
		assert.Less(t, first, second, "endpoints should be sorted")
	}
}


func TestGenerateInternalSpecMergingPaths(t *testing.T) {
	oapi := &core.OpenAPI{
		OpenAPI: core.OpenAPIVersion31,
		Info:    &core.Info{Title: "Test", Version: "1.0"},
		Paths: map[string]*core.PathItem{
			"/items": {
				Get: &core.Operation{OperationID: "list-items"},
			},
		},
	}

	hiddenOps := []*core.Operation{
		{Method: "POST", Path: "/items", OperationID: "create-item", Hidden: true},
		{Method: "GET", Path: "/new-path", OperationID: "new-hidden", Hidden: true},
	}

	result, err := openapi.GenerateInternalSpec(oapi, hiddenOps)
	require.NoError(t, err)

	pi := result.Paths["/items"]
	require.NotNil(t, pi)
	assert.NotNil(t, pi.Get)
	assert.NotNil(t, pi.Post)

	assert.NotNil(t, result.Paths["/new-path"])

	require.NotNil(t, oapi.Paths["/items"])
	assert.Nil(t, oapi.Paths["/items"].Post)
	assert.NotContains(t, oapi.Paths, "/new-path")
}


func TestGenerateInternalSpecJSONHiddenOpsAndParams(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.InternalSpec = core.InternalSpecConfig{
		Path:    "/internal/openapi",
		Enabled: true,
	}
	_, api := neomatest.New(t, config)

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/visible",
		OperationID: "get-visible",
		HiddenParameters: []*core.Param{
			{
				Name: "X-Debug",
				In:   "header",
				Schema: &core.Schema{
					Type: "string",
				},
			},
		},
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/secret",
		OperationID: "get-secret",
		Hidden:      true,
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/internal/openapi.json")
	assert.Equal(t, http.StatusOK, resp.Code)
	body := resp.Body.String()
	assert.Contains(t, body, "/secret")
	assert.Contains(t, body, "X-Debug")

	resp = api.Do(http.MethodGet, "/internal/openapi.yaml")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "/secret")
}


func TestTagNestingOAS32(t *testing.T) {
	oapi := &core.OpenAPI{
		OpenAPI: core.OpenAPIVersion32,
		Info:    &core.Info{Title: "Test", Version: "1.0"},
		Tags: []*core.Tag{
			{
				Name: "Users",
				Tags: []*core.Tag{
					{Name: "Admin"},
					{Name: "Public"},
				},
			},
		},
	}

	b, err := json.Marshal(oapi)
	require.NoError(t, err)

	var spec map[string]any
	require.NoError(t, json.Unmarshal(b, &spec))

	tags, ok := spec["tags"].([]any)
	require.True(t, ok, "spec should have a tags array")
	require.Len(t, tags, 1)

	usersTag, ok := tags[0].(map[string]any)
	require.True(t, ok, "first tag should be an object")
	assert.Equal(t, "Users", usersTag["name"])

	nestedTags, ok := usersTag["tags"].([]any)
	require.True(t, ok, "Users tag should have nested tags")
	require.Len(t, nestedTags, 2)

	adminTag, ok := nestedTags[0].(map[string]any)
	require.True(t, ok, "nested tag should be an object")
	assert.Equal(t, "Admin", adminTag["name"])

	publicTag, ok := nestedTags[1].(map[string]any)
	require.True(t, ok, "nested tag should be an object")
	assert.Equal(t, "Public", publicTag["name"])
}


func TestSwaggerUIOAuthConfig(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.Docs = core.DocsConfig{
		Path: "/docs",
		Provider: openapi.SwaggerUIProvider{
			OAuthClientID: "my-client",
			OAuthScopes:   []string{"read", "write"},
		},
		Enabled: true,
	}
	_, api := neomatest.New(t, config)

	resp := api.Do(http.MethodGet, "/docs")
	require.Equal(t, http.StatusOK, resp.Code)

	body := resp.Body.String()
	assert.Contains(t, body, "initOAuth")
	assert.Contains(t, body, "my-client")
	assert.Contains(t, body, "read")
	assert.Contains(t, body, "write")
}
