package openapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/yaml"
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

func marshalPublicSpec(oapi *core.OpenAPI, config core.Config) ([]byte, error) {
	specJSON, err := marshalSpec(oapi, config)
	if err != nil {
		return nil, err
	}
	if !config.ExcludeHiddenSchemas {
		return specJSON, nil
	}
	return filterSpecSchemas(specJSON, publicSchemaNames(oapi))
}

func marshalPublicSpecYAML(oapi *core.OpenAPI, config core.Config) ([]byte, error) {
	if !config.ExcludeHiddenSchemas {
		return marshalSpecYAML(oapi, config)
	}
	specJSON, err := marshalPublicSpec(oapi, config)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(nil)
	if err := yaml.Convert(buf, bytes.NewReader(specJSON)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
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
			specJSON, _ = marshalPublicSpec(oapi, config)
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
			specYAML, _ = marshalPublicSpecYAML(oapi, config)
		}
		_, _ = ctx.BodyWriter().Write(specYAML)
	}))
}

func registerSchemaRoute(adapter core.Adapter, oapi *core.OpenAPI, config core.Config) {
	mw := config.SpecMiddlewares

	var public map[string]bool

	adapter.Handle(&core.Operation{
		Method: http.MethodGet,
		Path:   config.SchemasPath + "/{schema}",
	}, mw.Handler(func(ctx core.Context) {
		name := strings.TrimSuffix(ctx.Param("schema"), ".json")

		if config.ExcludeHiddenSchemas {
			if public == nil {
				public = publicSchemaNames(oapi)
			}
			if !public[name] {
				ctx.SetStatus(http.StatusNotFound)
				return
			}
		}

		ctx.SetHeader("Content-Type", "application/json")
		b, _ := json.Marshal(oapi.Components.Schemas.Map()[name])
		b = rxSchema.ReplaceAll(b, []byte(config.SchemasPath+`/$1.json`))
		_, _ = ctx.BodyWriter().Write(b)
	}))
}

// publicSchemaNames returns the set of schema names reachable from public
// operations in oapi.Paths, resolved transitively through schema references.
func publicSchemaNames(oapi *core.OpenAPI) map[string]bool {
	if oapi.Paths == nil || oapi.Components == nil || oapi.Components.Schemas == nil {
		return nil
	}

	public := map[string]bool{}
	pathsJSON, err := json.Marshal(oapi.Paths)
	if err != nil {
		return nil
	}
	for _, match := range rxSchema.FindAllSubmatch(pathsJSON, -1) {
		public[string(match[1])] = true
	}

	// Resolve transitive refs through schema definitions.
	schemas := oapi.Components.Schemas.Map()
	changed := true
	for changed {
		changed = false
		for name := range public {
			s := schemas[name]
			if s == nil {
				continue
			}
			schemaJSON, err := json.Marshal(s)
			if err != nil {
				continue
			}
			for _, match := range rxSchema.FindAllSubmatch(schemaJSON, -1) {
				ref := string(match[1])
				if !public[ref] {
					public[ref] = true
					changed = true
				}
			}
		}
	}
	return public
}

func filterSpecSchemas(specJSON []byte, public map[string]bool) ([]byte, error) {
	if len(public) == 0 {
		return specJSON, nil
	}

	var doc map[string]any
	if err := json.Unmarshal(specJSON, &doc); err != nil {
		return specJSON, err
	}

	components, _ := doc["components"].(map[string]any)
	if components == nil {
		return specJSON, nil
	}
	schemas, _ := components["schemas"].(map[string]any)
	if schemas == nil {
		return specJSON, nil
	}

	for name := range schemas {
		if !public[name] {
			delete(schemas, name)
		}
	}

	return json.Marshal(doc)
}
