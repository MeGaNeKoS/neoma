package core

import "time"

// ErrorResponseConfig holds configuration for a specific error response status
// code, allowing customization of the description, headers, and schema.
type ErrorResponseConfig struct {
	Description string
	Headers     map[string]*Param
	Schema      *Schema
}

// Operation represents a single API operation on a path, combining the
// OpenAPI 3.x operation object fields (tags, summary, parameters, responses,
// etc.) with framework-specific settings (method, path, body limits,
// validation flags, middleware).
type Operation struct {
	Method                       string                       `yaml:"-"`
	Path                         string                       `yaml:"-"`
	DefaultStatus                int                          `yaml:"-"`
	MaxBodyBytes                 int64                        `yaml:"-"`
	BodyReadTimeout              time.Duration                `yaml:"-"`
	Errors                       []int                        `yaml:"-"`
	ErrorHeaders                 map[string]*Param            `yaml:"-"`
	ErrorResponses               map[int]*ErrorResponseConfig `yaml:"-"`
	SkipValidateParams           bool                         `yaml:"-"`
	SkipValidateBody             bool                         `yaml:"-"`
	SkipDiscoveredErrors         bool                         `yaml:"-"`
	ErrorExamples                map[int]any                  `yaml:"-"`
	RejectUnknownQueryParameters bool                         `yaml:"-"`
	Hidden                       bool                         `yaml:"-"`
	HiddenParameters             []*Param                     `yaml:"-"`
	Metadata                     map[string]any               `yaml:"-"`
	Middlewares                  Middlewares                  `yaml:"-"`

	Tags          []string                          `yaml:"tags,omitempty"`
	Summary       string                            `yaml:"summary,omitempty"`
	Description   string                            `yaml:"description,omitempty"`
	ExternalDocs  *ExternalDocs                     `yaml:"externalDocs,omitempty"`
	OperationID   string                            `yaml:"operationId,omitempty"`
	Parameters    []*Param                          `yaml:"parameters,omitempty"`
	RequestBody   *RequestBody                      `yaml:"requestBody,omitempty"`
	Responses     map[string]*Response              `yaml:"responses,omitempty"`
	Callbacks     map[string]map[string]*PathItem   `yaml:"callbacks,omitempty"`
	Deprecated    bool                              `yaml:"deprecated,omitempty"`
	Security      []map[string][]string             `yaml:"security,omitempty"`
	Servers       []*Server                         `yaml:"servers,omitempty"`
	Extensions    map[string]any                    `yaml:",inline"`
}

// MarshalJSON serializes the Operation to JSON, including only the OpenAPI
// spec fields and any extensions. Framework-specific fields are excluded.
func (o *Operation) MarshalJSON() ([]byte, error) {
	return MarshalJSON([]JSONFieldInfo{
		{"tags", o.Tags, OmitEmpty},
		{"summary", o.Summary, OmitEmpty},
		{"description", o.Description, OmitEmpty},
		{"externalDocs", o.ExternalDocs, OmitEmpty},
		{"operationId", o.OperationID, OmitEmpty},
		{"parameters", o.Parameters, OmitEmpty},
		{"requestBody", o.RequestBody, OmitEmpty},
		{"responses", o.Responses, OmitEmpty},
		{"callbacks", o.Callbacks, OmitEmpty},
		{"deprecated", o.Deprecated, OmitEmpty},
		{"security", o.Security, OmitNil},
		{"servers", o.Servers, OmitEmpty},
	}, o.Extensions)
}
