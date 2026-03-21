package core

import (
	"io"
	"reflect"
)

// DocsProvider renders an HTML documentation page for the API. Implementations
// receive the OpenAPI spec URL and a human-readable title.
type DocsProvider interface {
	Render(specURL string, title string) string
}

// DocsConfig holds the settings for the API documentation endpoint, including
// the URL path, the rendering provider, and optional middleware.
type DocsConfig struct {
	Path        string
	Provider    DocsProvider
	Middlewares Middlewares
	Enabled     bool
}

// InternalSpecConfig holds the settings for the internal OpenAPI spec
// endpoint, which serves the raw specification for internal tooling.
type InternalSpecConfig struct {
	Path        string
	DocsPath    string
	Middlewares Middlewares
	Enabled     bool
}

// Supported OpenAPI specification versions.
const (
	OpenAPIVersion30 = "3.0.3"
	OpenAPIVersion31 = "3.1.0"
	OpenAPIVersion32 = "3.2.0"
)

// ErrorDocEntry describes a single error scenario with its cause and
// recommended fix, for use in human-readable error documentation pages.
type ErrorDocEntry struct {
	Cause string
	Fix   string
}

// ErrorDoc defines the documentation page for a specific HTTP error status
// code. When HTML is set, it is rendered directly; otherwise, the default
// template is used with Title, Description, and Entries.
type ErrorDoc struct {
	Title       string
	Description string
	Entries     []ErrorDocEntry
	HTML        string // if set, renders this directly instead of the default template
}

// Config holds the top-level configuration for a neoma API instance,
// including OpenAPI metadata, content negotiation formats, schema generation
// options, transformers, middleware, and error handling.
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

// HiddenOperationsProvider is implemented by adapters that support operations
// excluded from the public OpenAPI spec but still routable (for example,
// internal health checks or spec endpoints).
type HiddenOperationsProvider interface {
	HiddenOperations() []*Operation
	AddHiddenOperation(op *Operation)
}

// API is the primary interface for interacting with a running neoma API. It
// provides access to the underlying adapter, OpenAPI spec, content
// negotiation, marshaling, middleware registration, and error handling.
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
