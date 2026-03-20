package core

import (
	"io"
	"reflect"
)

type DocsProvider interface {
	Render(specURL string, title string) string
}

type DocsConfig struct {
	Path        string
	Provider    DocsProvider
	Middlewares Middlewares
	Enabled     bool
}

type InternalSpecConfig struct {
	Path        string
	DocsPath    string
	Middlewares Middlewares
	Enabled     bool
}

const (
	OpenAPIVersion30 = "3.0.3"
	OpenAPIVersion31 = "3.1.0"
	OpenAPIVersion32 = "3.2.0"
)

type ErrorDocEntry struct {
	Cause string
	Fix   string
}

type ErrorDoc struct {
	Title       string
	Description string
	Entries     []ErrorDocEntry
	HTML        string // if set, renders this directly instead of the default template
}

type Config struct {
	*OpenAPI
	OpenAPIPath                        string
	OpenAPIVersion                     string
	Docs                               DocsConfig
	SchemasPath                        string
	SpecMiddlewares                    Middlewares
	Formats                            map[string]Format
	DefaultFormat                      string
	NoFormatFallback                   bool
	RejectUnknownQueryParameters       bool
	Transformers                       []Transformer
	CreateHooks                        []func(Config) Config
	ErrorHandler                       ErrorHandler
	GenerateOperationID                func(method, path string) string
	GenerateSummary                    func(method, path string) string
	SchemaNamer                        func(t reflect.Type, hint string) string
	InternalSpec                       InternalSpecConfig
	ExcludeHiddenSchemas               bool
	AllowAdditionalPropertiesByDefault bool
	FieldsOptionalByDefault            bool
	ErrorDocs                          map[int]ErrorDoc
}

type HiddenOperationsProvider interface {
	HiddenOperations() []*Operation
	AddHiddenOperation(op *Operation)
}

type API interface {
	Adapter() Adapter
	OpenAPI() *OpenAPI
	Negotiate(accept string) (string, error)
	Transform(ctx Context, status string, v any) (any, error)
	Marshal(w io.Writer, contentType string, v any) error
	Unmarshal(contentType string, data []byte, v any) error
	UseMiddleware(middlewares ...MiddlewareFunc)
	Middlewares() Middlewares
	ErrorHandler() ErrorHandler
	GenerateOperationID(method, path string) string
	UseGlobalMiddleware(middlewares ...MiddlewareFunc)
}
