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

func TestSchemaLinkTransformerLinkHeader(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Output struct {
		Body struct {
			Greeting string `json:"greeting"`
		}
	}

	neoma.Get[struct{}, Output](api, "/greeting", func(_ context.Context, _ *struct{}) (*Output, error) {
		o := &Output{}
		o.Body.Greeting = "Hello!"
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/greeting")
	assert.Equal(t, http.StatusOK, resp.Code)

	link := resp.Header().Get("Link")
	assert.Contains(t, link, `rel="describedBy"`)
	assert.Contains(t, link, "/schemas/")
	assert.Contains(t, link, ".json")
}

func TestSchemaLinkTransformerSchemaField(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Output struct {
		Body struct {
			Msg string `json:"msg"`
		}
	}

	neoma.Get[struct{}, Output](api, "/msg", func(_ context.Context, _ *struct{}) (*Output, error) {
		o := &Output{}
		o.Body.Msg = "hi"
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/msg")
	assert.Equal(t, http.StatusOK, resp.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	schema, ok := body["$schema"].(string)
	require.True(t, ok, "response should contain $schema field")
	assert.Contains(t, schema, "/schemas/")
	assert.Contains(t, schema, ".json")
}

func TestSchemaLinkTransformerNilResponse(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Output struct {
		Body struct {
			Data string `json:"data"`
		}
	}

	neoma.Get[struct{}, Output](api, "/nil", func(_ context.Context, _ *struct{}) (*Output, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/nil")
	assert.Equal(t, http.StatusOK, resp.Code)
	// No Link header when output is nil (no body to describe)
	assert.Empty(t, resp.Header().Get("Link"))
}

func TestSchemaLinkTransformerOpenAPISchemaProperty(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Output struct {
		Body struct {
			Value int `json:"value"`
		}
	}

	neoma.Get[struct{}, Output](api, "/value", func(_ context.Context, _ *struct{}) (*Output, error) {
		o := &Output{}
		o.Body.Value = 42
		return o, nil
	})

	oapi := api.OpenAPI()
	for _, schema := range oapi.Components.Schemas.Map() {
		if schema.Type == core.TypeObject && schema.Properties != nil {
			if prop, ok := schema.Properties["$schema"]; ok {
				assert.Equal(t, core.TypeString, prop.Type)
				assert.Equal(t, "uri", prop.Format)
				assert.True(t, prop.ReadOnly)
				return
			}
		}
	}
	t.Fatal("expected $schema property in at least one schema")
}

func TestSchemaLinkTransformerWithAPIPrefix(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.Servers = []*core.Server{
		{URL: "https://api.example.com/v2"},
	}
	_, api := neomatest.New(t, config)

	type Output struct {
		Body struct {
			ID int `json:"id"`
		}
	}

	neoma.Get[struct{}, Output](api, "/item", func(_ context.Context, _ *struct{}) (*Output, error) {
		o := &Output{}
		o.Body.ID = 1
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/item")
	assert.Equal(t, http.StatusOK, resp.Code)

	link := resp.Header().Get("Link")
	assert.Contains(t, link, "/v2/schemas/")
}

func TestSchemaLinkTransformerSliceBodyNoLink(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Output struct {
		Body []struct {
			ID int `json:"id"`
		}
	}

	neoma.Get[struct{}, Output](api, "/items", func(_ context.Context, _ *struct{}) (*Output, error) {
		o := &Output{}
		o.Body = append(o.Body, struct {
			ID int `json:"id"`
		}{ID: 1})
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/items")
	assert.Equal(t, http.StatusOK, resp.Code)

	// Slice responses don't get the Link header since they're not struct types
	link := resp.Header().Get("Link")
	assert.Empty(t, link)
}

func TestSchemaLinkTransformerDisabledWhenNoCreateHooks(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.CreateHooks = nil
	_, api := neomatest.New(t, config)

	type Output struct {
		Body struct {
			OK bool `json:"ok"`
		}
	}

	neoma.Get[struct{}, Output](api, "/no-link", func(_ context.Context, _ *struct{}) (*Output, error) {
		o := &Output{}
		o.Body.OK = true
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/no-link")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Empty(t, resp.Header().Get("Link"))
}

func TestSchemaLinkTransformerDirectInstantiation(t *testing.T) {
	tr := openapi.NewSchemaLinkTransformer("/schemas")
	assert.NotNil(t, tr)
}
