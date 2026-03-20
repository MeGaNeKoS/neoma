package openapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/neoma"
	"github.com/MeGaNeKoS/neoma/neomatest"
	"github.com/MeGaNeKoS/neoma/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)


func newTestAPI(t *testing.T, middlewares ...core.MiddlewareFunc) neomatest.TestAPI {
	t.Helper()
	config := neoma.DefaultConfig("Test API", "1.0.0")
	config.InternalSpec = core.InternalSpecConfig{
		Path:        "/internal/openapi",
		Enabled:     true,
		Middlewares: middlewares,
	}
	_, api := neomatest.New(t, config)
	return api
}

type PublicInput struct {
	ID string `path:"id"`
}

type PublicOutput struct {
	Body struct {
		Name   string `json:"name"`
		Secret string `json:"secret" hidden:"true"`
	}
}

type HiddenInput struct{}
type HiddenOutput struct {
	Body struct {
		Debug string `json:"debug"`
	}
}


func TestInternalSpecIncludesHiddenOperations(t *testing.T) {
	api := newTestAPI(t)

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/public",
		OperationID: "get-public",
		Errors:      []int{http.StatusNotFound},
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

	resp := api.Do(http.MethodGet, "/openapi.json")
	require.Equal(t, http.StatusOK, resp.Code)

	var publicSpec map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &publicSpec))

	paths, _ := publicSpec["paths"].(map[string]any)
	assert.Contains(t, paths, "/public", "public spec should contain /public")
	assert.NotContains(t, paths, "/hidden", "public spec should NOT contain /hidden")

	resp = api.Do(http.MethodGet, "/internal/openapi.json")
	require.Equal(t, http.StatusOK, resp.Code)

	var internalSpec map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &internalSpec))

	iPaths, _ := internalSpec["paths"].(map[string]any)
	assert.Contains(t, iPaths, "/public", "internal spec should contain /public")
	assert.Contains(t, iPaths, "/hidden", "internal spec should contain /hidden")
}


func TestInternalSpecIncludesHiddenFields(t *testing.T) {
	api := newTestAPI(t)

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/items/{id}",
		OperationID: "get-item",
		Errors:      []int{http.StatusNotFound},
	}, func(_ context.Context, input *PublicInput) (*PublicOutput, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/openapi.json")
	require.Equal(t, http.StatusOK, resp.Code)

	publicJSON := resp.Body.String()
	assert.NotContains(t, publicJSON, `"secret"`, "public spec should NOT contain hidden field 'secret'")
	assert.Contains(t, publicJSON, `"name"`, "public spec should contain field 'name'")

	resp = api.Do(http.MethodGet, "/internal/openapi.json")
	require.Equal(t, http.StatusOK, resp.Code)

	internalJSON := resp.Body.String()
	assert.Contains(t, internalJSON, `"secret"`, "internal spec should contain hidden field 'secret'")
	assert.Contains(t, internalJSON, `"name"`, "internal spec should contain field 'name'")
}


func TestInternalSpecMiddleware(t *testing.T) {
	blocked := false
	authMiddleware := func(ctx core.Context, next func(core.Context)) {
		if ctx.Header("Authorization") != "Bearer internal-token" {
			blocked = true
			ctx.SetStatus(http.StatusForbidden)
			return
		}
		next(ctx)
	}

	api := newTestAPI(t, authMiddleware)

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/test",
		OperationID: "get-test",
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/internal/openapi.json")
	assert.Equal(t, http.StatusForbidden, resp.Code)
	assert.True(t, blocked, "middleware should have blocked the request")

	blocked = false
	resp = api.Do(http.MethodGet, "/internal/openapi.json", "Authorization: Bearer internal-token")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.False(t, blocked, "middleware should have allowed the request")
}


func TestPublicSpecExcludesHidden(t *testing.T) {
	api := newTestAPI(t)

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/items/{id}",
		OperationID: "get-item",
		Errors:      []int{http.StatusNotFound},
	}, func(_ context.Context, input *PublicInput) (*PublicOutput, error) {
		return nil, nil
	})

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/debug/stats",
		OperationID: "get-debug-stats",
		Hidden:      true,
	}, func(_ context.Context, _ *HiddenInput) (*HiddenOutput, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/openapi.json")
	require.Equal(t, http.StatusOK, resp.Code)

	var publicSpec map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &publicSpec))

	paths, _ := publicSpec["paths"].(map[string]any)
	assert.NotContains(t, paths, "/debug/stats", "public spec should NOT contain hidden op")
	assert.Contains(t, paths, "/items/{id}", "public spec should contain public op")

	publicJSON := resp.Body.String()
	assert.NotContains(t, publicJSON, `"secret"`, "public spec should NOT contain hidden field")
}


func TestInternalSpecFormats(t *testing.T) {
	api := newTestAPI(t)

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/test",
		OperationID: "get-test",
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	tests := []struct {
		path        string
		contentType string
	}{
		{"/internal/openapi.json", "application/openapi+json"},
		{"/internal/openapi.yaml", "application/openapi+yaml"},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			resp := api.Do(http.MethodGet, tc.path)
			assert.Equal(t, http.StatusOK, resp.Code, "endpoint %s should return 200", tc.path)
			assert.Equal(t, tc.contentType, resp.Header().Get("Content-Type"),
				"endpoint %s should set correct Content-Type", tc.path)
			assert.NotEmpty(t, resp.Body.String(), "endpoint %s should return a non-empty body", tc.path)
		})
	}
}


func TestInternalSpecDisabled(t *testing.T) {
	config := neoma.DefaultConfig("Test API", "1.0.0")
	_, api := neomatest.New(t, config)

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/test",
		OperationID: "get-test",
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/internal/openapi.json")
	assert.NotEqual(t, http.StatusOK, resp.Code, "disabled internal spec should not serve")
}


func TestGenerateInternalSpecJSON(t *testing.T) {
	oapi := &core.OpenAPI{
		OpenAPI: core.OpenAPIVersion31,
		Info:    &core.Info{Title: "Test", Version: "1.0.0"},
		Paths: map[string]*core.PathItem{
			"/visible": {
				Get: &core.Operation{
					Method:      "GET",
					Path:        "/visible",
					OperationID: "get-visible",
				},
			},
		},
	}

	hiddenOps := []*core.Operation{
		{
			Method:      "GET",
			Path:        "/hidden",
			OperationID: "get-hidden",
			Hidden:      true,
		},
	}

	data, err := openapi.GenerateInternalSpecJSON(oapi, hiddenOps)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	var spec map[string]any
	require.NoError(t, json.Unmarshal(data, &spec))

	paths, _ := spec["paths"].(map[string]any)
	assert.Contains(t, paths, "/visible", "internal spec should contain visible path")
	assert.Contains(t, paths, "/hidden", "internal spec should contain hidden path")
}


func TestInternalSpecEmptyPath(t *testing.T) {
	config := neoma.DefaultConfig("Test API", "1.0.0")
	config.InternalSpec = core.InternalSpecConfig{
		Path:    "",
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

	resp := api.Do(http.MethodGet, "/internal/openapi.json")
	assert.NotEqual(t, http.StatusOK, resp.Code)
}
