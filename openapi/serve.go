package openapi

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/MeGaNeKoS/neoma/core"
)

var rxSchema = regexp.MustCompile(`#/components/schemas/([^"]+)`)

// RegisterSpecRoutes registers HTTP routes that serve the public OpenAPI spec
// in JSON and YAML formats, and optionally individual schema routes.
func RegisterSpecRoutes(adapter core.Adapter, oapi *core.OpenAPI, config core.Config) {
	if config.OpenAPIPath != "" {
		registerOpenAPIRoutes(adapter, oapi, config)
	}
	if config.SchemasPath != "" {
		registerSchemaRoute(adapter, oapi, config)
	}
}

func marshalSpec(oapi *core.OpenAPI, config core.Config) ([]byte, error) {
	switch config.OpenAPIVersion {
	case core.OpenAPIVersion30:
		return Downgrade(oapi)
	default:
		return json.Marshal(oapi)
	}
}

func marshalSpecYAML(oapi *core.OpenAPI, config core.Config) ([]byte, error) {
	switch config.OpenAPIVersion {
	case core.OpenAPIVersion30:
		return DowngradeYAML(oapi)
	default:
		return YAML(oapi)
	}
}

func registerOpenAPIRoutes(adapter core.Adapter, oapi *core.OpenAPI, config core.Config) {
	mw := config.SpecMiddlewares

	var specJSON []byte
	adapter.Handle(&core.Operation{
		Method: http.MethodGet,
		Path:   config.OpenAPIPath + ".json",
	}, mw.Handler(func(ctx core.Context) {
		ctx.SetHeader("Content-Type", "application/openapi+json")
		if specJSON == nil {
			specJSON, _ = marshalSpec(oapi, config)
		}
		_, _ = ctx.BodyWriter().Write(specJSON)
	}))

	var specYAML []byte
	adapter.Handle(&core.Operation{
		Method: http.MethodGet,
		Path:   config.OpenAPIPath + ".yaml",
	}, mw.Handler(func(ctx core.Context) {
		ctx.SetHeader("Content-Type", "application/openapi+yaml")
		if specYAML == nil {
			specYAML, _ = marshalSpecYAML(oapi, config)
		}
		_, _ = ctx.BodyWriter().Write(specYAML)
	}))
}

func registerSchemaRoute(adapter core.Adapter, oapi *core.OpenAPI, config core.Config) {
	mw := config.SpecMiddlewares

	adapter.Handle(&core.Operation{
		Method: http.MethodGet,
		Path:   config.SchemasPath + "/{schema}",
	}, mw.Handler(func(ctx core.Context) {
		name := strings.TrimSuffix(ctx.Param("schema"), ".json")
		ctx.SetHeader("Content-Type", "application/json")
		b, _ := json.Marshal(oapi.Components.Schemas.Map()[name])
		b = rxSchema.ReplaceAll(b, []byte(config.SchemasPath+`/$1.json`))
		_, _ = ctx.BodyWriter().Write(b)
	}))
}
