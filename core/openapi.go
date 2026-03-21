package core

// Contact represents the contact information for the exposed API, as defined
// in the OpenAPI 3.x specification.
type Contact struct {
	Name       string         `yaml:"name,omitempty"`
	URL        string         `yaml:"url,omitempty"`
	Email      string         `yaml:"email,omitempty"`
	Extensions map[string]any `yaml:",inline"`
}

// MarshalJSON serializes the Contact to JSON with support for extensions.
func (c *Contact) MarshalJSON() ([]byte, error) {
	return MarshalJSON([]JSONFieldInfo{
		{"name", c.Name, OmitEmpty},
		{"url", c.URL, OmitEmpty},
		{"email", c.Email, OmitEmpty},
	}, c.Extensions)
}

// License represents the license information for the exposed API, as defined
// in the OpenAPI 3.x specification.
type License struct {
	Name       string         `yaml:"name"`
	Identifier string         `yaml:"identifier,omitempty"`
	URL        string         `yaml:"url,omitempty"`
	Extensions map[string]any `yaml:",inline"`
}

// MarshalJSON serializes the License to JSON with support for extensions.
func (l *License) MarshalJSON() ([]byte, error) {
	return MarshalJSON([]JSONFieldInfo{
		{"name", l.Name, OmitNever},
		{"identifier", l.Identifier, OmitEmpty},
		{"url", l.URL, OmitEmpty},
	}, l.Extensions)
}

// Info represents the metadata about the API, as defined in the OpenAPI 3.x
// specification. The Title and Version fields are required.
type Info struct {
	Title          string         `yaml:"title"`
	Description    string         `yaml:"description,omitempty"`
	TermsOfService string         `yaml:"termsOfService,omitempty"`
	Contact        *Contact       `yaml:"contact,omitempty"`
	License        *License       `yaml:"license,omitempty"`
	Version        string         `yaml:"version"`
	Extensions     map[string]any `yaml:",inline"`
}

// MarshalJSON serializes the Info to JSON with support for extensions.
func (i *Info) MarshalJSON() ([]byte, error) {
	return MarshalJSON([]JSONFieldInfo{
		{"title", i.Title, OmitNever},
		{"description", i.Description, OmitEmpty},
		{"termsOfService", i.TermsOfService, OmitEmpty},
		{"contact", i.Contact, OmitEmpty},
		{"license", i.License, OmitEmpty},
		{"version", i.Version, OmitNever},
	}, i.Extensions)
}

// ServerVariable represents a server variable for server URL template
// substitution, as defined in the OpenAPI 3.x specification.
type ServerVariable struct {
	Enum        []string       `yaml:"enum,omitempty"`
	Default     string         `yaml:"default"`
	Description string         `yaml:"description,omitempty"`
	Extensions  map[string]any `yaml:",inline"`
}

// MarshalJSON serializes the ServerVariable to JSON with support for extensions.
func (v *ServerVariable) MarshalJSON() ([]byte, error) {
	return MarshalJSON([]JSONFieldInfo{
		{"enum", v.Enum, OmitEmpty},
		{"default", v.Default, OmitNever},
		{"description", v.Description, OmitEmpty},
	}, v.Extensions)
}

// Server represents a server that hosts the API, as defined in the OpenAPI 3.x
// specification. The URL field may contain variable substitutions.
type Server struct {
	URL         string                     `yaml:"url"`
	Description string                     `yaml:"description,omitempty"`
	Variables   map[string]*ServerVariable `yaml:"variables,omitempty"`
	Extensions  map[string]any             `yaml:",inline"`
}

// MarshalJSON serializes the Server to JSON with support for extensions.
func (s *Server) MarshalJSON() ([]byte, error) {
	return MarshalJSON([]JSONFieldInfo{
		{"url", s.URL, OmitNever},
		{"description", s.Description, OmitEmpty},
		{"variables", s.Variables, OmitEmpty},
	}, s.Extensions)
}

// Example represents an example value for a media type, parameter, or schema,
// as defined in the OpenAPI 3.x specification.
type Example struct {
	Ref           string         `yaml:"$ref,omitempty"`
	Summary       string         `yaml:"summary,omitempty"`
	Description   string         `yaml:"description,omitempty"`
	Value         any            `yaml:"value,omitempty"`
	ExternalValue string         `yaml:"externalValue,omitempty"`
	Extensions    map[string]any `yaml:",inline"`
}

// MarshalJSON serializes the Example to JSON with support for extensions.
func (e *Example) MarshalJSON() ([]byte, error) {
	return MarshalJSON([]JSONFieldInfo{
		{"$ref", e.Ref, OmitEmpty},
		{"summary", e.Summary, OmitEmpty},
		{"description", e.Description, OmitEmpty},
		{"value", e.Value, OmitNil},
		{"externalValue", e.ExternalValue, OmitEmpty},
	}, e.Extensions)
}

// Encoding represents a single encoding definition applied to a single schema
// property within a media type, as defined in the OpenAPI 3.x specification.
type Encoding struct {
	ContentType   string             `yaml:"contentType,omitempty"`
	Headers       map[string]*Header `yaml:"headers,omitempty"`
	Style         string             `yaml:"style,omitempty"`
	Explode       *bool              `yaml:"explode,omitempty"`
	AllowReserved bool               `yaml:"allowReserved,omitempty"`
	Extensions    map[string]any     `yaml:",inline"`
}

// MarshalJSON serializes the Encoding to JSON with support for extensions.
func (e *Encoding) MarshalJSON() ([]byte, error) {
	return MarshalJSON([]JSONFieldInfo{
		{"contentType", e.ContentType, OmitEmpty},
		{"headers", e.Headers, OmitEmpty},
		{"style", e.Style, OmitEmpty},
		{"explode", e.Explode, OmitEmpty},
		{"allowReserved", e.AllowReserved, OmitEmpty},
	}, e.Extensions)
}

// MediaType represents a media type with a schema and optional examples, as
// defined in the OpenAPI 3.x specification.
type MediaType struct {
	Schema     *Schema              `yaml:"schema,omitempty"`
	Example    any                  `yaml:"example,omitempty"`
	Examples   map[string]*Example  `yaml:"examples,omitempty"`
	Encoding   map[string]*Encoding `yaml:"encoding,omitempty"`
	Extensions map[string]any       `yaml:",inline"`
}

// MarshalJSON serializes the MediaType to JSON with support for extensions.
func (m *MediaType) MarshalJSON() ([]byte, error) {
	return MarshalJSON([]JSONFieldInfo{
		{"schema", m.Schema, OmitEmpty},
		{"example", m.Example, OmitNil},
		{"examples", m.Examples, OmitEmpty},
		{"encoding", m.Encoding, OmitEmpty},
	}, m.Extensions)
}

// Param represents a parameter passed to an operation via path, query, header,
// or cookie, as defined in the OpenAPI 3.x specification.
type Param struct {
	Ref             string              `yaml:"$ref,omitempty"`
	Name            string              `yaml:"name,omitempty"`
	In              string              `yaml:"in,omitempty"`
	Description     string              `yaml:"description,omitempty"`
	Required        bool                `yaml:"required,omitempty"`
	Deprecated      bool                `yaml:"deprecated,omitempty"`
	AllowEmptyValue bool                `yaml:"allowEmptyValue,omitempty"`
	Style           string              `yaml:"style,omitempty"`
	Explode         *bool               `yaml:"explode,omitempty"`
	AllowReserved   bool                `yaml:"allowReserved,omitempty"`
	Schema          *Schema             `yaml:"schema,omitempty"`
	Example         any                 `yaml:"example,omitempty"`
	Examples        map[string]*Example `yaml:"examples,omitempty"`
	Extensions      map[string]any      `yaml:",inline"`
}

// MarshalJSON serializes the Param to JSON with support for extensions.
func (p *Param) MarshalJSON() ([]byte, error) {
	return MarshalJSON([]JSONFieldInfo{
		{"$ref", p.Ref, OmitEmpty},
		{"name", p.Name, OmitEmpty},
		{"in", p.In, OmitEmpty},
		{"description", p.Description, OmitEmpty},
		{"required", p.Required, OmitEmpty},
		{"deprecated", p.Deprecated, OmitEmpty},
		{"allowEmptyValue", p.AllowEmptyValue, OmitEmpty},
		{"style", p.Style, OmitEmpty},
		{"explode", p.Explode, OmitEmpty},
		{"allowReserved", p.AllowReserved, OmitEmpty},
		{"schema", p.Schema, OmitEmpty},
		{"example", p.Example, OmitNil},
		{"examples", p.Examples, OmitEmpty},
	}, p.Extensions)
}

// Header is an alias for Param, representing an HTTP header parameter as
// defined in the OpenAPI 3.x specification.
type Header = Param

// RequestBody represents the request body for an operation, as defined in the
// OpenAPI 3.x specification.
type RequestBody struct {
	Ref         string                `yaml:"$ref,omitempty"`
	Description string                `yaml:"description,omitempty"`
	Content     map[string]*MediaType `yaml:"content"`
	Required    bool                  `yaml:"required,omitempty"`
	Extensions  map[string]any        `yaml:",inline"`
}

// MarshalJSON serializes the RequestBody to JSON with support for extensions.
func (r *RequestBody) MarshalJSON() ([]byte, error) {
	return MarshalJSON([]JSONFieldInfo{
		{"$ref", r.Ref, OmitEmpty},
		{"description", r.Description, OmitEmpty},
		{"content", r.Content, OmitNever},
		{"required", r.Required, OmitEmpty},
	}, r.Extensions)
}

// Link represents a possible design-time link for a response, as defined in
// the OpenAPI 3.x specification.
type Link struct {
	Ref          string         `yaml:"$ref,omitempty"`
	OperationRef string         `yaml:"operationRef,omitempty"`
	OperationID  string         `yaml:"operationId,omitempty"`
	Parameters   map[string]any `yaml:"parameters,omitempty"`
	RequestBody  any            `yaml:"requestBody,omitempty"`
	Description  string         `yaml:"description,omitempty"`
	Server       *Server        `yaml:"server,omitempty"`
	Extensions   map[string]any `yaml:",inline"`
}

// MarshalJSON serializes the Link to JSON with support for extensions.
func (l *Link) MarshalJSON() ([]byte, error) {
	return MarshalJSON([]JSONFieldInfo{
		{"$ref", l.Ref, OmitEmpty},
		{"operationRef", l.OperationRef, OmitEmpty},
		{"operationId", l.OperationID, OmitEmpty},
		{"parameters", l.Parameters, OmitEmpty},
		{"requestBody", l.RequestBody, OmitNil},
		{"description", l.Description, OmitEmpty},
		{"server", l.Server, OmitEmpty},
	}, l.Extensions)
}

// Response represents a single response from an API operation, including
// headers, content, and links, as defined in the OpenAPI 3.x specification.
type Response struct {
	Ref         string                `yaml:"$ref,omitempty"`
	Description string                `yaml:"description,omitempty"`
	Headers     map[string]*Param     `yaml:"headers,omitempty"`
	Content     map[string]*MediaType `yaml:"content,omitempty"`
	Links       map[string]*Link      `yaml:"links,omitempty"`
	Extensions  map[string]any        `yaml:",inline"`
}

// MarshalJSON serializes the Response to JSON with support for extensions.
func (r *Response) MarshalJSON() ([]byte, error) {
	return MarshalJSON([]JSONFieldInfo{
		{"$ref", r.Ref, OmitEmpty},
		{"description", r.Description, OmitEmpty},
		{"headers", r.Headers, OmitEmpty},
		{"content", r.Content, OmitEmpty},
		{"links", r.Links, OmitEmpty},
	}, r.Extensions)
}

// PathItem represents the operations available on a single API path, as
// defined in the OpenAPI 3.x specification.
type PathItem struct {
	Ref         string         `yaml:"$ref,omitempty"`
	Summary     string         `yaml:"summary,omitempty"`
	Description string         `yaml:"description,omitempty"`
	Get         *Operation     `yaml:"get,omitempty"`
	Put         *Operation     `yaml:"put,omitempty"`
	Post        *Operation     `yaml:"post,omitempty"`
	Delete      *Operation     `yaml:"delete,omitempty"`
	Options     *Operation     `yaml:"options,omitempty"`
	Head        *Operation     `yaml:"head,omitempty"`
	Patch       *Operation     `yaml:"patch,omitempty"`
	Trace       *Operation     `yaml:"trace,omitempty"`
	Servers     []*Server      `yaml:"servers,omitempty"`
	Parameters  []*Param       `yaml:"parameters,omitempty"`
	Extensions  map[string]any `yaml:",inline"`
}

// MarshalJSON serializes the PathItem to JSON with support for extensions.
func (p *PathItem) MarshalJSON() ([]byte, error) {
	return MarshalJSON([]JSONFieldInfo{
		{"$ref", p.Ref, OmitEmpty},
		{"summary", p.Summary, OmitEmpty},
		{"description", p.Description, OmitEmpty},
		{"get", p.Get, OmitEmpty},
		{"put", p.Put, OmitEmpty},
		{"post", p.Post, OmitEmpty},
		{"delete", p.Delete, OmitEmpty},
		{"options", p.Options, OmitEmpty},
		{"head", p.Head, OmitEmpty},
		{"patch", p.Patch, OmitEmpty},
		{"trace", p.Trace, OmitEmpty},
		{"servers", p.Servers, OmitEmpty},
		{"parameters", p.Parameters, OmitEmpty},
	}, p.Extensions)
}

// ExternalDocs represents a reference to external documentation, as defined in
// the OpenAPI 3.x specification.
type ExternalDocs struct {
	Description string         `yaml:"description,omitempty"`
	URL         string         `yaml:"url"`
	Extensions  map[string]any `yaml:",inline"`
}

// MarshalJSON serializes the ExternalDocs to JSON with support for extensions.
func (e *ExternalDocs) MarshalJSON() ([]byte, error) {
	return MarshalJSON([]JSONFieldInfo{
		{"description", e.Description, OmitEmpty},
		{"url", e.URL, OmitNever},
	}, e.Extensions)
}

// Tag represents a tag for API documentation control, allowing operations to
// be grouped by resources or other qualifiers, as defined in the OpenAPI 3.x
// specification.
type Tag struct {
	Name         string         `yaml:"name"`
	Description  string         `yaml:"description,omitempty"`
	Tags         []*Tag         `yaml:"tags,omitempty"`
	ExternalDocs *ExternalDocs  `yaml:"externalDocs,omitempty"`
	Extensions   map[string]any `yaml:",inline"`
}

// MarshalJSON serializes the Tag to JSON with support for extensions.
func (t *Tag) MarshalJSON() ([]byte, error) {
	return MarshalJSON([]JSONFieldInfo{
		{"name", t.Name, OmitNever},
		{"description", t.Description, OmitEmpty},
		{"tags", t.Tags, OmitEmpty},
		{"externalDocs", t.ExternalDocs, OmitEmpty},
	}, t.Extensions)
}

// OAuthFlow represents the configuration for an OAuth 2.0 flow, as defined in
// the OpenAPI 3.x specification.
type OAuthFlow struct {
	AuthorizationURL string            `yaml:"authorizationUrl,omitempty"`
	TokenURL         string            `yaml:"tokenUrl"`
	RefreshURL       string            `yaml:"refreshUrl,omitempty"`
	Scopes           map[string]string `yaml:"scopes"`
	Extensions       map[string]any    `yaml:",inline"`
}

// MarshalJSON serializes the OAuthFlow to JSON with support for extensions.
func (o *OAuthFlow) MarshalJSON() ([]byte, error) {
	return MarshalJSON([]JSONFieldInfo{
		{"authorizationUrl", o.AuthorizationURL, OmitEmpty},
		{"tokenUrl", o.TokenURL, OmitNever},
		{"refreshUrl", o.RefreshURL, OmitEmpty},
		{"scopes", o.Scopes, OmitNever},
	}, o.Extensions)
}

// OAuthFlows represents the available OAuth 2.0 flows (implicit, password,
// client credentials, authorization code), as defined in the OpenAPI 3.x
// specification.
type OAuthFlows struct {
	Implicit          *OAuthFlow     `yaml:"implicit,omitempty"`
	Password          *OAuthFlow     `yaml:"password,omitempty"`
	ClientCredentials *OAuthFlow     `yaml:"clientCredentials,omitempty"`
	AuthorizationCode *OAuthFlow     `yaml:"authorizationCode,omitempty"`
	Extensions        map[string]any `yaml:",inline"`
}

// MarshalJSON serializes the OAuthFlows to JSON with support for extensions.
func (o *OAuthFlows) MarshalJSON() ([]byte, error) {
	return MarshalJSON([]JSONFieldInfo{
		{"implicit", o.Implicit, OmitEmpty},
		{"password", o.Password, OmitEmpty},
		{"clientCredentials", o.ClientCredentials, OmitEmpty},
		{"authorizationCode", o.AuthorizationCode, OmitEmpty},
	}, o.Extensions)
}

// SecurityScheme represents a security scheme that can be used by API
// operations (e.g., HTTP authentication, API key, OAuth 2.0, OpenID Connect),
// as defined in the OpenAPI 3.x specification.
type SecurityScheme struct {
	Type             string         `yaml:"type"`
	Description      string         `yaml:"description,omitempty"`
	Name             string         `yaml:"name,omitempty"`
	In               string         `yaml:"in,omitempty"`
	Scheme           string         `yaml:"scheme,omitempty"`
	BearerFormat     string         `yaml:"bearerFormat,omitempty"`
	Flows            *OAuthFlows    `yaml:"flows,omitempty"`
	OpenIDConnectURL string         `yaml:"openIdConnectUrl,omitempty"`
	Extensions       map[string]any `yaml:",inline"`
}

// MarshalJSON serializes the SecurityScheme to JSON with support for extensions.
func (s *SecurityScheme) MarshalJSON() ([]byte, error) {
	return MarshalJSON([]JSONFieldInfo{
		{"type", s.Type, OmitNever},
		{"description", s.Description, OmitEmpty},
		{"name", s.Name, OmitEmpty},
		{"in", s.In, OmitEmpty},
		{"scheme", s.Scheme, OmitEmpty},
		{"bearerFormat", s.BearerFormat, OmitEmpty},
		{"flows", s.Flows, OmitEmpty},
		{"openIdConnectUrl", s.OpenIDConnectURL, OmitEmpty},
	}, s.Extensions)
}

// Components holds a set of reusable objects for the API specification,
// as defined in the OpenAPI 3.x specification.
type Components struct {
	Schemas         Registry                      `yaml:"schemas,omitempty"`
	Responses       map[string]*Response          `yaml:"responses,omitempty"`
	Parameters      map[string]*Param             `yaml:"parameters,omitempty"`
	Examples        map[string]*Example           `yaml:"examples,omitempty"`
	RequestBodies   map[string]*RequestBody       `yaml:"requestBodies,omitempty"`
	Headers         map[string]*Header            `yaml:"headers,omitempty"`
	SecuritySchemes map[string]*SecurityScheme    `yaml:"securitySchemes,omitempty"`
	Links           map[string]*Link              `yaml:"links,omitempty"`
	Callbacks       map[string]map[string]*PathItem `yaml:"callbacks,omitempty"`
	Extensions      map[string]any                `yaml:",inline"`
}

// MarshalJSON serializes the Components to JSON with support for extensions.
func (c *Components) MarshalJSON() ([]byte, error) {
	return MarshalJSON([]JSONFieldInfo{
		{"schemas", c.Schemas, OmitEmpty},
		{"responses", c.Responses, OmitEmpty},
		{"parameters", c.Parameters, OmitEmpty},
		{"examples", c.Examples, OmitEmpty},
		{"requestBodies", c.RequestBodies, OmitEmpty},
		{"headers", c.Headers, OmitEmpty},
		{"securitySchemes", c.SecuritySchemes, OmitEmpty},
		{"links", c.Links, OmitEmpty},
		{"callbacks", c.Callbacks, OmitEmpty},
	}, c.Extensions)
}

// OpenAPI represents the root document object of an OpenAPI 3.x specification.
// It contains all API metadata, paths, components, and security definitions.
type OpenAPI struct {
	OpenAPI string `yaml:"openapi"`

	Info         *Info                    `yaml:"info"`
	Servers      []*Server                `yaml:"servers,omitempty"`
	Paths        map[string]*PathItem     `yaml:"paths,omitempty"`
	Components   *Components              `yaml:"components,omitempty"`
	Security     []map[string][]string    `yaml:"security,omitempty"`
	Tags         []*Tag                   `yaml:"tags,omitempty"`
	ExternalDocs *ExternalDocs            `yaml:"externalDocs,omitempty"`
	Extensions   map[string]any           `yaml:",inline"`

	// OnAddOperation is called when a new operation is added to the spec.
	// This is useful for transformers that need to modify the spec.
	OnAddOperation []func(oapi *OpenAPI, op *Operation) `yaml:"-"`
}

// MarshalJSON serializes the OpenAPI document to JSON with support for extensions.
func (o *OpenAPI) MarshalJSON() ([]byte, error) {
	return MarshalJSON([]JSONFieldInfo{
		{"openapi", o.OpenAPI, OmitNever},
		{"info", o.Info, OmitEmpty},
		{"servers", o.Servers, OmitEmpty},
		{"paths", o.Paths, OmitEmpty},
		{"components", o.Components, OmitEmpty},
		{"security", o.Security, OmitNil},
		{"tags", o.Tags, OmitEmpty},
		{"externalDocs", o.ExternalDocs, OmitEmpty},
	}, o.Extensions)
}

// AddOperation registers an operation under its path and method in the spec,
// then invokes any OnAddOperation hooks.
func (o *OpenAPI) AddOperation(op *Operation) {
	if o.Paths == nil {
		o.Paths = map[string]*PathItem{}
	}

	pi := o.Paths[op.Path]
	if pi == nil {
		pi = &PathItem{}
		o.Paths[op.Path] = pi
	}

	switch op.Method {
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

	for _, hook := range o.OnAddOperation {
		hook(o, op)
	}
}
