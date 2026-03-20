package openapi_test

import (
	"encoding/json"
	"testing"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildSpec(version string) *core.OpenAPI {
	oapi := &core.OpenAPI{
		OpenAPI: version,
		Info: &core.Info{
			Title:   "Test",
			Version: "1.0.0",
		},
		Paths: map[string]*core.PathItem{
			"/test": {
				Get: &core.Operation{
					OperationID: "get-test",
					Responses: map[string]*core.Response{
						"200": {
							Description: "OK",
							Content: map[string]*core.MediaType{
								"application/json": {
									Schema: &core.Schema{
										Type: "object",
										Properties: map[string]*core.Schema{
											"name": {
												Type:     "string",
												Nullable: true,
											},
											"age": {
												Type:    "integer",
												Minimum: ptrFloat(0),
											},
											"tags": {
												Type:     "array",
												Nullable: true,
												Items:    &core.Schema{Type: "string"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return oapi
}

func ptrFloat(f float64) *float64 { return &f }

func TestDowngrade31To30Version(t *testing.T) {
	spec := buildSpec(core.OpenAPIVersion31)

	downgraded, err := openapi.Downgrade(spec)
	require.NoError(t, err)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(downgraded, &raw))

	assert.Equal(t, core.OpenAPIVersion30, raw["openapi"])
}

func TestDowngrade32To30Version(t *testing.T) {
	spec := buildSpec(core.OpenAPIVersion32)

	downgraded, err := openapi.Downgrade(spec)
	require.NoError(t, err)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(downgraded, &raw))

	assert.Equal(t, core.OpenAPIVersion30, raw["openapi"])
}

func TestDowngradeNullableTypes(t *testing.T) {
	spec := buildSpec(core.OpenAPIVersion32)

	downgraded, err := openapi.Downgrade(spec)
	require.NoError(t, err)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(downgraded, &raw))

	paths := raw["paths"].(map[string]any)
	testPath := paths["/test"].(map[string]any)
	get := testPath["get"].(map[string]any)
	responses := get["responses"].(map[string]any)
	resp200 := responses["200"].(map[string]any)
	content := resp200["content"].(map[string]any)
	appJSON := content["application/json"].(map[string]any)
	schemaMap := appJSON["schema"].(map[string]any)
	props := schemaMap["properties"].(map[string]any)

	// In 3.1: "type": ["string", "null"]
	// In 3.0: "type": "string", "nullable": true
	nameSchema := props["name"].(map[string]any)
	assert.Equal(t, "string", nameSchema["type"],
		"nullable string should downgrade to type:string")
	assert.Equal(t, true, nameSchema["nullable"],
		"nullable string should have nullable:true in 3.0")
}

func TestDowngradePreservesNonNullable(t *testing.T) {
	spec := buildSpec(core.OpenAPIVersion32)

	downgraded, err := openapi.Downgrade(spec)
	require.NoError(t, err)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(downgraded, &raw))

	paths := raw["paths"].(map[string]any)
	testPath := paths["/test"].(map[string]any)
	get := testPath["get"].(map[string]any)
	responses := get["responses"].(map[string]any)
	resp200 := responses["200"].(map[string]any)
	content := resp200["content"].(map[string]any)
	appJSON := content["application/json"].(map[string]any)
	schemaMap := appJSON["schema"].(map[string]any)
	props := schemaMap["properties"].(map[string]any)

	ageSchema := props["age"].(map[string]any)
	assert.Equal(t, "integer", ageSchema["type"])
	assert.Nil(t, ageSchema["nullable"],
		"non-nullable field should not have nullable in 3.0")
}

func TestOriginalSpecUnchangedAfterDowngrade(t *testing.T) {
	spec := buildSpec(core.OpenAPIVersion32)

	_, err := openapi.Downgrade(spec)
	require.NoError(t, err)

	original, err := json.Marshal(spec)
	require.NoError(t, err)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(original, &raw))
	assert.Equal(t, core.OpenAPIVersion32, raw["openapi"])
}

func TestYAMLOutput(t *testing.T) {
	spec := buildSpec(core.OpenAPIVersion32)

	yamlBytes, err := openapi.YAML(spec)
	require.NoError(t, err)
	assert.Contains(t, string(yamlBytes), "openapi: "+core.OpenAPIVersion32)
}

func TestDowngradeYAMLOutput(t *testing.T) {
	spec := buildSpec(core.OpenAPIVersion32)

	yamlBytes, err := openapi.DowngradeYAML(spec)
	require.NoError(t, err)
	assert.Contains(t, string(yamlBytes), "openapi: "+core.OpenAPIVersion30)
}
