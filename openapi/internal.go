package openapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"path"
	"strings"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/yaml"
)


type hiddenParamEntry struct {
	path   string
	method string
	params []*core.Param
}


// GenerateInternalSpec returns a copy of the OpenAPI spec with hidden operations
// merged in, intended for internal-only consumption.
func GenerateInternalSpec(oapi *core.OpenAPI, hiddenOps []*core.Operation) (*core.OpenAPI, error) {
	out := *oapi
	out.Paths = make(map[string]*core.PathItem, len(oapi.Paths)+len(hiddenOps))
	for k, v := range oapi.Paths {
		cp := *v
		out.Paths[k] = &cp
	}
	for _, op := range hiddenOps {
		pi, ok := out.Paths[op.Path]
		if !ok {
			pi = &core.PathItem{}
			out.Paths[op.Path] = pi
		}
		switch strings.ToUpper(op.Method) {
		case "GET":
			pi.Get = op
		case "PUT":
			pi.Put = op
		case "POST":
			pi.Post = op
		case "DELETE":
			pi.Delete = op
		case "OPTIONS":
			pi.Options = op
		case "HEAD":
			pi.Head = op
		case "PATCH":
			pi.Patch = op
		case "TRACE":
			pi.Trace = op
		}
	}
	return &out, nil
}

// GenerateInternalSpecJSON returns the internal OpenAPI spec as JSON, including
// hidden operations, hidden parameters, and hidden schema properties.
func GenerateInternalSpecJSON(oapi *core.OpenAPI, hiddenOps []*core.Operation) ([]byte, error) {
	data, err := json.Marshal(oapi)
	if err != nil {
		return nil, err
	}

	opsWithHiddenParams := collectHiddenParams(oapi)
	hiddenSchemaProps := collectHiddenSchemaProps(oapi)

	if len(hiddenOps) == 0 && len(opsWithHiddenParams) == 0 && len(hiddenSchemaProps) == 0 {
		return data, nil
	}

	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if doc == nil {
		doc = map[string]any{}
	}

	paths, _ := doc["paths"].(map[string]any)
	if paths == nil {
		paths = map[string]any{}
		doc["paths"] = paths
	}

	for _, op := range hiddenOps {
		opJSON, err := json.Marshal(op)
		if err != nil {
			continue
		}
		var opMap any
		if err := json.Unmarshal(opJSON, &opMap); err != nil {
			continue
		}

		pi, _ := paths[op.Path].(map[string]any)
		if pi == nil {
			pi = map[string]any{}
			paths[op.Path] = pi
		}
		pi[strings.ToLower(op.Method)] = opMap
	}

	for _, hp := range opsWithHiddenParams {
		pi, _ := paths[hp.path].(map[string]any)
		if pi == nil {
			continue
		}
		opMap, _ := pi[strings.ToLower(hp.method)].(map[string]any)
		if opMap == nil {
			continue
		}
		params, _ := opMap["parameters"].([]any)
		for _, p := range hp.params {
			pJSON, err := json.Marshal(p)
			if err != nil {
				continue
			}
			var pMap any
			if err := json.Unmarshal(pJSON, &pMap); err != nil {
				continue
			}
			params = append(params, pMap)
		}
		opMap["parameters"] = params
	}

	// Inject hidden schema properties (MarshalJSON excludes them from public spec).
	injectHiddenSchemaProps(doc, hiddenSchemaProps)

	return json.Marshal(doc)
}

// RegisterInternalSpecRoutes registers HTTP routes that serve the internal
// OpenAPI spec (JSON and YAML) and an internal documentation page.
func RegisterInternalSpecRoutes(adapter core.Adapter, api core.API, config core.Config) {
	isc := config.InternalSpec
	if !isc.Enabled || isc.Path == "" {
		return
	}

	oapi := config.OpenAPI
	mw := isc.Middlewares

	var specJSON []byte
	adapter.Handle(&core.Operation{
		Method: http.MethodGet,
		Path:   isc.Path + ".json",
	}, mw.Handler(func(ctx core.Context) {
		ctx.SetHeader("Content-Type", "application/openapi+json")
		if specJSON == nil {
			specJSON, _ = GenerateInternalSpecJSON(oapi, getHiddenOps(api))
		}
		_, _ = ctx.BodyWriter().Write(specJSON)
	}))

	var specYAML []byte
	adapter.Handle(&core.Operation{
		Method: http.MethodGet,
		Path:   isc.Path + ".yaml",
	}, mw.Handler(func(ctx core.Context) {
		ctx.SetHeader("Content-Type", "application/openapi+yaml")
		if specYAML == nil {
			specYAML, _ = generateInternalSpecYAML(oapi, getHiddenOps(api))
		}
		_, _ = ctx.BodyWriter().Write(specYAML)
	}))

	registerInternalDocsRoute(adapter, oapi, config)
}


func collectHiddenParams(oapi *core.OpenAPI) []hiddenParamEntry {
	var result []hiddenParamEntry
	for p, pi := range oapi.Paths {
		for method, op := range map[string]*core.Operation{
			"get": pi.Get, "put": pi.Put, "post": pi.Post,
			"delete": pi.Delete, "patch": pi.Patch, "head": pi.Head,
			"options": pi.Options, "trace": pi.Trace,
		} {
			if op != nil && len(op.HiddenParameters) > 0 {
				result = append(result, hiddenParamEntry{
					path:   p,
					method: method,
					params: op.HiddenParameters,
				})
			}
		}
	}
	return result
}

type hiddenSchemaProp struct {
	schemaName string
	propName   string
	propSchema *core.Schema
}

func collectHiddenSchemaProps(oapi *core.OpenAPI) []hiddenSchemaProp {
	if oapi.Components == nil || oapi.Components.Schemas == nil {
		return nil
	}
	var result []hiddenSchemaProp
	for name, s := range oapi.Components.Schemas.Map() {
		for propName, prop := range s.Properties {
			if prop.Hidden {
				result = append(result, hiddenSchemaProp{
					schemaName: name,
					propName:   propName,
					propSchema: prop,
				})
			}
		}
	}
	return result
}

func injectHiddenSchemaProps(doc map[string]any, props []hiddenSchemaProp) {
	if len(props) == 0 {
		return
	}
	components, _ := doc["components"].(map[string]any)
	if components == nil {
		return
	}
	schemas, _ := components["schemas"].(map[string]any)
	if schemas == nil {
		return
	}
	for _, hp := range props {
		schema, _ := schemas[hp.schemaName].(map[string]any)
		if schema == nil {
			continue
		}
		schemaProps, _ := schema["properties"].(map[string]any)
		if schemaProps == nil {
			schemaProps = map[string]any{}
			schema["properties"] = schemaProps
		}
		propJSON, err := json.Marshal(hp.propSchema)
		if err != nil {
			continue
		}
		var propMap any
		if err := json.Unmarshal(propJSON, &propMap); err != nil {
			continue
		}
		schemaProps[hp.propName] = propMap
	}
}

func generateInternalSpecYAML(oapi *core.OpenAPI, hiddenOps []*core.Operation) ([]byte, error) {
	specJSON, err := GenerateInternalSpecJSON(oapi, hiddenOps)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(nil)
	if err := yaml.Convert(buf, bytes.NewReader(specJSON)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func getHiddenOps(api core.API) []*core.Operation {
	if hop, ok := api.(core.HiddenOperationsProvider); ok {
		return hop.HiddenOperations()
	}
	return nil
}

func registerInternalDocsRoute(adapter core.Adapter, oapi *core.OpenAPI, config core.Config) {
	isc := config.InternalSpec
	docsPath := isc.DocsPath
	if docsPath == "" {
		docsPath = "/internal/docs"
	}

	provider := config.Docs.Provider
	if provider == nil {
		provider = StoplightProvider{}
	}

	var title string
	if oapi.Info != nil && oapi.Info.Title != "" {
		title = oapi.Info.Title + " (Internal) Reference"
	}

	// Point docs UI to the internal spec, not the public one.
	internalSpecPath := isc.Path
	if prefix := getAPIPrefix(oapi); prefix != "" {
		internalSpecPath = path.Join(prefix, internalSpecPath)
	}

	body := []byte(provider.Render(internalSpecPath, title))

	var cspHeader string
	if cp, ok := provider.(cspProvider); ok {
		cspHeader = cp.csp()
	} else {
		cspHeader = strings.Join([]string{
			"default-src 'none'",
			"base-uri 'none'",
			"connect-src 'self'",
			"form-action 'none'",
			"frame-ancestors 'none'",
			"sandbox allow-same-origin allow-scripts allow-popups allow-popups-to-escape-sandbox",
			"script-src 'unsafe-inline'",
			"style-src 'unsafe-inline'",
		}, "; ")
	}

	endpoint := func(ctx core.Context) {
		ctx.SetHeader("Content-Security-Policy", cspHeader)
		ctx.SetHeader("Content-Type", "text/html")
		_, _ = ctx.BodyWriter().Write(body)
	}

	handler := isc.Middlewares.Handler(endpoint)
	adapter.Handle(&core.Operation{
		Method: http.MethodGet,
		Path:   docsPath,
	}, handler)
}
