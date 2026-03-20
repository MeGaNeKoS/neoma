package neoma_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/neoma"
	"github.com/MeGaNeKoS/neoma/neomatest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type (
	PaginationParams struct {
		Page    int    `query:"page" minimum:"1" default:"1" doc:"Page number"`
		PerPage int    `query:"per_page" minimum:"1" maximum:"100" default:"20" doc:"Items per page"`
		Sort    string `query:"sort" enum:"asc,desc" default:"asc" doc:"Sort order"`
	}

	OASItemInput struct {
		ID int `path:"id" doc:"Item ID"`
	}

	OASCreateItemInput struct {
		Body struct {
			Name        string   `json:"name" minLength:"1" maxLength:"255" doc:"Item name"`
			Description string   `json:"description,omitempty" doc:"Item description"`
			Tags        []string `json:"tags,omitempty" maxItems:"10" doc:"Tags for the item"`
			Price       float64  `json:"price" minimum:"0" doc:"Price in USD"`
			Active      bool     `json:"active" default:"true" doc:"Whether item is active"`
		}
	}

	OASItemOutput struct {
		Body struct {
			ID          int       `json:"id" doc:"Item ID"`
			Name        string    `json:"name" doc:"Item name"`
			Description string    `json:"description,omitempty" doc:"Item description"`
			Tags        []string  `json:"tags,omitempty" doc:"Tags"`
			Price       float64   `json:"price" doc:"Price in USD"`
			Active      bool      `json:"active" doc:"Whether item is active"`
			CreatedAt   time.Time `json:"created_at" doc:"Creation timestamp"`
		}
	}

	OASListItemsInput struct {
		PaginationParams
	}

	OASListItemsOutput struct {
		Body []struct {
			ID   int    `json:"id" doc:"Item ID"`
			Name string `json:"name" doc:"Item name"`
		}
	}
)

func TestOASStructure(t *testing.T) {
	_, api := neomatest.New(t)
	spec := api.OpenAPI()

	assert.Equal(t, core.OpenAPIVersion32, spec.OpenAPI, "must be OpenAPI 3.2.0")
	require.NotNil(t, spec.Info, "info is required by OAS")
	assert.NotEmpty(t, spec.Info.Title, "info.title is required by OAS")
	assert.NotEmpty(t, spec.Info.Version, "info.version is required by OAS")
	assert.NotNil(t, spec.Components, "components should exist for schemas")
}

func TestOASOperationFields(t *testing.T) {
	_, api := neomatest.New(t)

	neoma.Get(api, "/items", func(ctx context.Context, input *OASListItemsInput) (*OASListItemsOutput, error) {
		return nil, nil
	})
	neoma.Post(api, "/items", func(ctx context.Context, input *OASCreateItemInput) (*OASItemOutput, error) {
		return nil, nil
	})
	neoma.Get(api, "/items/{id}", func(ctx context.Context, input *OASItemInput) (*OASItemOutput, error) {
		return nil, nil
	})

	spec := api.OpenAPI()

	assert.Contains(t, spec.Paths, "/items")
	assert.Contains(t, spec.Paths, "/items/{id}")

	require.NotNil(t, spec.Paths["/items"])
	getItems := spec.Paths["/items"].Get
	require.NotNil(t, getItems)
	assert.NotEmpty(t, getItems.OperationID, "operationId is recommended by OAS")

	paramNames := make(map[string]bool)
	for _, p := range getItems.Parameters {
		paramNames[p.Name] = true
	}
	assert.True(t, paramNames["page"], "should document 'page' query param")
	assert.True(t, paramNames["per_page"], "should document 'per_page' query param")
	assert.True(t, paramNames["sort"], "should document 'sort' query param")

	require.NotNil(t, spec.Paths["/items"])
	postItems := spec.Paths["/items"].Post
	require.NotNil(t, postItems)
	require.NotNil(t, postItems.RequestBody, "POST must have requestBody")
	assert.Contains(t, postItems.RequestBody.Content, "application/json",
		"POST body should be documented as application/json")

	require.NotNil(t, spec.Paths["/items/{id}"])
	getItem := spec.Paths["/items/{id}"].Get
	require.NotNil(t, getItem)
	for _, p := range getItem.Parameters {
		if p.In == "path" {
			assert.True(t, p.Required,
				"path parameter %q must be required (OAS 3.1 rule)", p.Name)
		}
	}
}

func TestOASResponseDescriptions(t *testing.T) {
	_, api := neomatest.New(t)

	neoma.Register(api, core.Operation{
		OperationID: "get-item",
		Method:      "GET",
		Path:        "/items/{id}",
		Errors:      []int{404, 422},
	}, func(ctx context.Context, input *OASItemInput) (*OASItemOutput, error) {
		return nil, nil
	})

	spec := api.OpenAPI()
	require.NotNil(t, spec.Paths["/items/{id}"])
	op := spec.Paths["/items/{id}"].Get
	require.NotNil(t, op)

	for code, resp := range op.Responses {
		assert.NotEmpty(t, resp.Description,
			"response %s must have a description (OAS 3.1 requirement)", code)
	}
}

func TestOASErrorHeaders(t *testing.T) {
	_, api := neomatest.New(t)

	neoma.Register(api, core.Operation{
		OperationID: "rate-limited",
		Method:      "GET",
		Path:        "/limited",
		Errors:      []int{429},
		ErrorHeaders: map[string]*core.Param{
			"Retry-After": {
				Description: "Seconds to wait before retrying",
				Schema:      &core.Schema{Type: "integer"},
			},
		},
	}, func(ctx context.Context, input *struct{}) (*struct{ Body struct{ OK bool `json:"ok"` } }, error) {
		return nil, nil
	})

	spec := api.OpenAPI()
	require.NotNil(t, spec.Paths["/limited"])
	op := spec.Paths["/limited"].Get
	require.NotNil(t, op)

	resp429, ok := op.Responses["429"]
	require.True(t, ok, "should have 429 response")
	require.NotNil(t, resp429.Headers, "429 response should have headers")
	assert.Contains(t, resp429.Headers, "Retry-After",
		"Retry-After header should be documented on 429 response")
}

func TestOASJSONMarshalValid(t *testing.T) {
	_, api := neomatest.New(t)

	neoma.Get(api, "/health", func(ctx context.Context, input *struct{}) (*struct {
		Body struct {
			Status string `json:"status"`
		}
	}, error) {
		return nil, nil
	})

	specJSON, err := json.MarshalIndent(api.OpenAPI(), "", "  ")
	require.NoError(t, err, "spec should marshal to JSON without error")

	var raw map[string]any
	err = json.Unmarshal(specJSON, &raw)
	require.NoError(t, err, "spec JSON should be valid JSON")

	assert.Contains(t, raw, "openapi")
	assert.Contains(t, raw, "info")
	assert.Equal(t, core.OpenAPIVersion32, raw["openapi"])

	info := raw["info"].(map[string]any)
	assert.Contains(t, info, "title")
	assert.Contains(t, info, "version")

	t.Logf("Generated spec size: %d bytes", len(specJSON))
}

func TestOASSchemaTypesValid(t *testing.T) {
	_, api := neomatest.New(t)

	neoma.Post(api, "/items", func(ctx context.Context, input *OASCreateItemInput) (*OASItemOutput, error) {
		return nil, nil
	})

	specJSON, err := json.Marshal(api.OpenAPI())
	require.NoError(t, err)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(specJSON, &raw))

	components, ok := raw["components"].(map[string]any)
	if !ok {
		t.Skip("no components")
	}
	schemas, ok := components["schemas"].(map[string]any)
	if !ok {
		t.Skip("no schemas")
	}

	validTypes := map[string]bool{
		"string": true, "number": true, "integer": true,
		"boolean": true, "array": true, "object": true,
	}

	for name, s := range schemas {
		schema, ok := s.(map[string]any)
		if !ok {
			continue
		}
		if typ, ok := schema["type"]; ok {
			switch v := typ.(type) {
			case string:
				assert.True(t, validTypes[v],
					"schema %q has invalid type %q (must be one of: string, number, integer, boolean, array, object)", name, v)
			case []any:
				// OAS 3.1 / JSON Schema: type can be array for nullable ["string", "null"]
				for _, item := range v {
					str, ok := item.(string)
					if ok && str != "null" {
						assert.True(t, validTypes[str],
							"schema %q has invalid type %q in type array", name, str)
					}
				}
			}
		}
	}
}

func TestOASPathParamsMustBeRequired(t *testing.T) {
	_, api := neomatest.New(t)

	neoma.Get(api, "/a/{x}/b/{y}", func(ctx context.Context, input *struct {
		X string `path:"x"`
		Y int    `path:"y"`
	}) (*struct{ Body struct{ OK bool `json:"ok"` } }, error) {
		return nil, nil
	})

	spec := api.OpenAPI()
	for path, pi := range spec.Paths {
		for _, op := range []*core.Operation{pi.Get, pi.Post, pi.Put, pi.Delete, pi.Patch} {
			if op == nil {
				continue
			}
			for _, p := range op.Parameters {
				if p.In == "path" {
					assert.True(t, p.Required,
						"OAS rule: path param %q in %s %s must be required", p.Name, op.Method, path)
				}
			}
		}
	}
}
