package openapi

import (
	"testing"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)


func TestInjectHiddenSchemaPropsNilComponents(t *testing.T) {
	doc := map[string]any{
		"openapi": "3.1.0",
		"paths":   map[string]any{},
	}
	props := []hiddenSchemaProp{
		{schemaName: "Foo", propName: "secret", propSchema: &core.Schema{Type: "string"}},
	}
	injectHiddenSchemaProps(doc, props)
}

func TestInjectHiddenSchemaPropsNilSchemas(t *testing.T) {
	doc := map[string]any{
		"openapi":    "3.1.0",
		"components": map[string]any{},
	}
	props := []hiddenSchemaProp{
		{schemaName: "Foo", propName: "secret", propSchema: &core.Schema{Type: "string"}},
	}
	injectHiddenSchemaProps(doc, props)
}

func TestInjectHiddenSchemaPropsNilTargetSchema(t *testing.T) {
	doc := map[string]any{
		"openapi": "3.1.0",
		"components": map[string]any{
			"schemas": map[string]any{
				"Bar": map[string]any{
					"type": "object",
				},
			},
		},
	}
	props := []hiddenSchemaProp{
		{schemaName: "Foo", propName: "secret", propSchema: &core.Schema{Type: "string"}},
	}
	injectHiddenSchemaProps(doc, props)
	schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
	bar := schemas["Bar"].(map[string]any)
	assert.Nil(t, bar["properties"], "Bar should not have a properties key added")
}

func TestInjectHiddenSchemaPropsNilProperties(t *testing.T) {
	doc := map[string]any{
		"openapi": "3.1.0",
		"components": map[string]any{
			"schemas": map[string]any{
				"Foo": map[string]any{
					"type": "object",
				},
			},
		},
	}
	props := []hiddenSchemaProp{
		{schemaName: "Foo", propName: "secret", propSchema: &core.Schema{Type: "string"}},
	}
	injectHiddenSchemaProps(doc, props)
	schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
	foo := schemas["Foo"].(map[string]any)
	fooProps, ok := foo["properties"].(map[string]any)
	assert.True(t, ok, "properties map should have been created")
	assert.NotNil(t, fooProps["secret"], "hidden property should have been injected")
}

func TestInjectHiddenSchemaPropsEmpty(t *testing.T) {
	doc := map[string]any{
		"openapi": "3.1.0",
	}
	injectHiddenSchemaProps(doc, nil)
}


func TestGetErrorDocFromUserDocs(t *testing.T) {
	userDocs := map[int]core.ErrorDoc{
		400: {Title: "Custom Bad Request", Description: "Custom description"},
	}
	doc := getErrorDoc(400, userDocs)
	assert.Equal(t, "Custom Bad Request", doc.Title)
}

func TestGetErrorDocFromDefaults(t *testing.T) {
	doc := getErrorDoc(400, nil)
	assert.Equal(t, "Bad Request", doc.Title)
}

func TestGetErrorDocValidButNotInDefaults(t *testing.T) {
	doc := getErrorDoc(307, nil)
	assert.Equal(t, "Temporary Redirect", doc.Title)
	assert.Contains(t, doc.Description, "No detailed documentation")
}

func TestGetErrorDocUnknownCode(t *testing.T) {
	doc := getErrorDoc(999, nil)
	assert.Empty(t, doc.Title)
}


func TestStatusBadge(t *testing.T) {
	assert.Equal(t, "badge-error", statusBadge(500))
	assert.Equal(t, "badge-error", statusBadge(503))
	assert.Equal(t, "badge-warn", statusBadge(400))
	assert.Equal(t, "badge-warn", statusBadge(404))
	assert.Equal(t, "badge-info", statusBadge(200))
	assert.Equal(t, "badge-info", statusBadge(301))
}

func TestStatusCategory(t *testing.T) {
	assert.Equal(t, "Server Error", statusCategory(500))
	assert.Equal(t, "Server Error", statusCategory(502))
	assert.Equal(t, "Client Error", statusCategory(400))
	assert.Equal(t, "Client Error", statusCategory(422))
	assert.Equal(t, "Info", statusCategory(200))
	assert.Equal(t, "Info", statusCategory(307))
}

func TestStatusColor(t *testing.T) {
	assert.Equal(t, "#dc2626", statusColor(500))
	assert.Equal(t, "#dc2626", statusColor(503))
	assert.Equal(t, "#d97706", statusColor(400))
	assert.Equal(t, "#d97706", statusColor(404))
	assert.Equal(t, "#6b7280", statusColor(200))
	assert.Equal(t, "#6b7280", statusColor(301))
}


func TestGetAPIPrefixEmptyServers(t *testing.T) {
	oapi := &core.OpenAPI{
		Servers: []*core.Server{},
	}
	assert.Empty(t, getAPIPrefix(oapi))
}

func TestGetAPIPrefixServerWithEmptyURL(t *testing.T) {
	oapi := &core.OpenAPI{
		Servers: []*core.Server{
			{URL: ""},
		},
	}
	assert.Empty(t, getAPIPrefix(oapi))
}

func TestGetAPIPrefixServerNoPath(t *testing.T) {
	oapi := &core.OpenAPI{
		Servers: []*core.Server{
			{URL: "https://example.com"},
		},
	}
	assert.Empty(t, getAPIPrefix(oapi))
}

func TestGetAPIPrefixServerWithPath(t *testing.T) {
	oapi := &core.OpenAPI{
		Servers: []*core.Server{
			{URL: "https://api.example.com/v2"},
		},
	}
	assert.Equal(t, "/v2", getAPIPrefix(oapi))
}

func TestGetAPIPrefixServerRelativePath(t *testing.T) {
	oapi := &core.OpenAPI{
		Servers: []*core.Server{
			{URL: "/api/v1"},
		},
	}
	assert.Equal(t, "/api/v1", getAPIPrefix(oapi))
}

func TestGetAPIPrefixServerRelativeNoSlash(t *testing.T) {
	// A relative URL without leading "/" and without host should not produce
	// a prefix, even if url.Parse gives it a Path.
	oapi := &core.OpenAPI{
		Servers: []*core.Server{
			{URL: "relative/path"},
		},
	}
	assert.Empty(t, getAPIPrefix(oapi))
}


func TestGenerateInternalSpecYAMLDirect(t *testing.T) {
	oapi := &core.OpenAPI{
		OpenAPI: core.OpenAPIVersion31,
		Info:    &core.Info{Title: "Test", Version: "1.0"},
		Paths: map[string]*core.PathItem{
			"/test": {
				Get: &core.Operation{OperationID: "get-test"},
			},
		},
	}

	yamlBytes, err := generateInternalSpecYAML(oapi, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, yamlBytes)
	assert.Contains(t, string(yamlBytes), "get-test")
}
