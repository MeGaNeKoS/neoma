package neoma

import (
	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/negotiate"
	"github.com/MeGaNeKoS/neoma/openapi"
	"github.com/MeGaNeKoS/neoma/schema"
)

func DefaultConfig(title string, version string) core.Config {
	return core.Config{
		OpenAPI: &core.OpenAPI{
			OpenAPI: core.OpenAPIVersion32,
			Info: &core.Info{
				Title:   title,
				Version: version,
			},
			Components: &core.Components{
				Schemas: schema.NewMapRegistry(schema.DefaultSchemaNamer),
			},
		},
		OpenAPIVersion: core.OpenAPIVersion32,
		OpenAPIPath:    "/openapi",
		Docs: core.DocsConfig{
			Path:    "/public/docs",
			Provider: openapi.ScalarProvider{},
			Enabled: true,
		},
		SchemasPath:                       "/schemas",
		Formats:                           negotiate.DefaultFormats(),
		DefaultFormat:                     "application/json",
		AllowAdditionalPropertiesByDefault: true,
		FieldsOptionalByDefault:           true,
		CreateHooks: []func(core.Config) core.Config{
			func(c core.Config) core.Config {
				linkTransformer := openapi.NewSchemaLinkTransformer(c.SchemasPath)
				c.OnAddOperation = append(c.OnAddOperation, linkTransformer.OnAddOperation)
				c.Transformers = append(c.Transformers, linkTransformer.Transform)
				return c
			},
		},
	}
}
