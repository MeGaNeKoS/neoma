package neoma

import (
	stderrors "errors"
	"io"
	"reflect"
	"regexp"
	"strings"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/errors"
	"github.com/MeGaNeKoS/neoma/negotiate"
	"github.com/MeGaNeKoS/neoma/openapi"
	"github.com/MeGaNeKoS/neoma/schema"

	"github.com/MeGaNeKoS/neoma/binding"
	"github.com/MeGaNeKoS/neoma/casing"
)

type api struct {
	adapter      core.Adapter
	config       core.Config
	negotiator   *negotiate.Negotiator
	transformers []core.Transformer
	middlewares  core.Middlewares
	errorHandler core.ErrorHandler
	genOpID      func(method, path string) string
	hiddenOps    []*core.Operation
}

func (a *api) Adapter() core.Adapter {
	return a.adapter
}

func (a *api) OpenAPI() *core.OpenAPI {
	return a.config.OpenAPI
}

func (a *api) Negotiate(accept string) (string, error) {
	return a.negotiator.Negotiate(accept)
}

func (a *api) Transform(ctx core.Context, status string, v any) (any, error) {
	var err error
	for _, t := range a.transformers {
		v, err = t(ctx, status, v)
		if err != nil {
			return nil, err
		}
	}
	return v, nil
}

func (a *api) Marshal(w io.Writer, ct string, v any) error {
	return a.negotiator.Marshal(w, ct, v)
}

func (a *api) Unmarshal(ct string, data []byte, v any) error {
	return a.negotiator.Unmarshal(ct, data, v)
}

func (a *api) UseMiddleware(middlewares ...core.MiddlewareFunc) {
	a.middlewares = append(a.middlewares, middlewares...)
}

func (a *api) Middlewares() core.Middlewares {
	return a.middlewares
}

func (a *api) ErrorHandler() core.ErrorHandler {
	return a.errorHandler
}

func (a *api) GenerateOperationID(method, path string) string {
	return a.genOpID(method, path)
}

func (a *api) UseGlobalMiddleware(middlewares ...core.MiddlewareFunc) {
	// Global middleware runs before route matching. For now we prepend to the
	// existing middleware stack, which achieves the same effect when the adapter
	// does not natively support global middleware.
	a.middlewares = append(middlewares, a.middlewares...)
}

func (a *api) Config() core.Config {
	return a.config
}

func (a *api) DocumentOperation(op *core.Operation) {
	if op.Hidden {
		a.hiddenOps = append(a.hiddenOps, op)
		return
	}
	a.OpenAPI().AddOperation(op)
}

func (a *api) HiddenOperations() []*core.Operation {
	return a.hiddenOps
}

func (a *api) AddHiddenOperation(op *core.Operation) {
	a.hiddenOps = append(a.hiddenOps, op)
}

var reRemoveIDs = regexp.MustCompile(`\{([^}]+)}`)

func defaultGenerateOperationID(method, path string) string {
	return casing.Kebab(method + "-" + reRemoveIDs.ReplaceAllString(path, "by-$1"))
}

func defaultGenerateSummary(method, path string) string {
	path = reRemoveIDs.ReplaceAllString(path, "by-$1")
	phrase := strings.ReplaceAll(
		casing.Kebab(strings.ToLower(method)+" "+path, strings.ToLower, casing.Initialism),
		"-", " ",
	)
	if len(phrase) == 0 {
		return method
	}
	return strings.ToUpper(phrase[:1]) + phrase[1:]
}

func NewAPI(config core.Config, a core.Adapter) core.API {
	for i := 0; i < len(config.CreateHooks); i++ {
		config = config.CreateHooks[i](config)
	}

	if config.OpenAPI == nil {
		config.OpenAPI = &core.OpenAPI{}
	}
	if config.OpenAPIVersion != "" {
		config.OpenAPI.OpenAPI = config.OpenAPIVersion
	} else if config.OpenAPI.OpenAPI == "" {
		config.OpenAPI.OpenAPI = core.OpenAPIVersion32
	}
	if config.Components == nil {
		config.Components = &core.Components{}
	}

	if config.Components.Schemas == nil {
		namer := config.SchemaNamer
		if namer == nil {
			namer = schema.DefaultSchemaNamer
		}
		config.Components.Schemas = schema.NewMapRegistryWithConfig(
			namer,
			schema.RegistryConfig{
				AllowAdditionalPropertiesByDefault: config.AllowAdditionalPropertiesByDefault,
				FieldsOptionalByDefault:            config.FieldsOptionalByDefault,
			},
		)
	}

	ef := config.ErrorHandler
	if ef == nil {
		ef = errors.NewRFC9457Handler()
	}

	if config.DefaultFormat == "" && !config.NoFormatFallback {
		if f, ok := config.Formats["application/json"]; ok && f.Marshal != nil {
			config.DefaultFormat = "application/json"
		}
	}

	neg := negotiate.NewNegotiator(config.Formats, config.DefaultFormat, config.NoFormatFallback)

	genID := config.GenerateOperationID
	if genID == nil {
		genID = defaultGenerateOperationID
	}

	newAPI := &api{
		adapter:      a,
		config:       config,
		negotiator:   neg,
		transformers: config.Transformers,
		errorHandler: ef,
		genOpID:      genID,
	}

	openapi.RegisterSpecRoutes(a, config.OpenAPI, config)

	openapi.RegisterDocsRoute(a, config.OpenAPI, config)

	if config.InternalSpec.Enabled && config.InternalSpec.Path != "" {
		openapi.RegisterInternalSpecRoutes(a, newAPI, config)
	}

	openapi.RegisterErrorDocRoutes(a, ef, config)

	return newAPI
}

func generateConvenienceOperationID(api core.API, method, path string, response any) string {
	action := method
	t := core.Deref(reflect.TypeOf(response))
	if t.Kind() == reflect.Struct {
		body, hasBody := t.FieldByName("Body")
		if hasBody && method == "GET" && core.Deref(body.Type).Kind() == reflect.Slice {
			action = "list"
		}
	}
	return api.GenerateOperationID(action, path)
}

func generateConvenienceSummary(method, path string, response any) string {
	action := method
	t := core.Deref(reflect.TypeOf(response))
	if t.Kind() == reflect.Struct {
		body, hasBody := t.FieldByName("Body")
		if hasBody && method == "GET" && core.Deref(body.Type).Kind() == reflect.Slice {
			action = "list"
		}
	}
	return defaultGenerateSummary(action, path)
}

func WriteErr(api core.API, ctx core.Context, status int, msg string, errs ...error) error {
	se := api.ErrorHandler().NewErrorWithContext(ctx, status, msg, errs...)
	status = se.StatusCode()

	// Set RFC 9457 Link header if the error has a type URI.
	var tp core.Linker
	if stderrors.As(se, &tp) {
		if typeURI := tp.GetType(); typeURI != "" && typeURI != "about:blank" {
			ctx.SetHeader("Link", `<`+typeURI+`>; rel="type"`)
		}
	}

	return binding.WriteResponse(api, ctx, status, "", se)
}

func writeErrFromContext(api core.API, ctx core.Context, cErr *binding.ContextError, res core.ValidateResult) {
	if cErr.Errs != nil {
		_ = WriteErr(api, ctx, cErr.Code, cErr.Msg, cErr.Errs...)
	} else {
		_ = WriteErr(api, ctx, cErr.Code, cErr.Msg, res.Errors...)
	}
}
