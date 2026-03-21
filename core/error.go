package core

import (
	"fmt"
	"net/http"
)

// ErrorDetailer is implemented by errors that can provide structured detail
// information suitable for inclusion in an API error response.
type ErrorDetailer interface {
	ErrorDetail() *ErrorDetail
}

// ErrorDetail represents a single validation or processing error with a
// human-readable message, the location where the error occurred, and the
// offending value. It implements both the error and ErrorDetailer interfaces.
type ErrorDetail struct {
	Message  string `json:"message,omitempty" doc:"Error message text"`
	Location string `json:"location,omitempty" doc:"Where the error occurred, e.g. 'body.items[3].tags'"`
	Value    any    `json:"value,omitempty" doc:"The value at the given location"`
}

// Error returns a human-readable string that includes the message, location,
// and value when available.
func (e *ErrorDetail) Error() string {
	if e.Location == "" && e.Value == nil {
		return e.Message
	}
	return fmt.Sprintf("%s (%s: %v)", e.Message, e.Location, e.Value)
}

// ErrorDetail returns itself, satisfying the ErrorDetailer interface.
func (e *ErrorDetail) ErrorDetail() *ErrorDetail {
	return e
}

// Error represents an HTTP error response with a status code and message.
// Implementations typically follow RFC 9457 (Problem Details for HTTP APIs).
type Error interface {
	StatusCode() int
	Error() string
}

// Headerer is implemented by error or response types that provide additional
// HTTP headers to include in the response.
type Headerer interface {
	GetHeaders() http.Header
}

// ContentTyper is implemented by response types that need to override the
// default Content-Type header. The method receives the negotiated content type
// and returns the actual content type to use.
type ContentTyper interface {
	ContentType(string) string
}

// Linker is implemented by response types that provide a link relation type,
// enabling automatic Link header generation per RFC 8288.
type Linker interface {
	GetType() string
}

// ErrorHandler defines how the framework creates and serializes error
// responses. Implementations control the error body schema, content type, and
// construction logic (e.g., RFC 9457 problem details).
type ErrorHandler interface {
	NewError(status int, msg string, errs ...error) Error
	NewErrorWithContext(ctx Context, status int, msg string, errs ...error) Error
	ErrorSchema(registry Registry) *Schema
	ErrorContentType(ct string) string
}

// DiscoveredError represents an error response that was automatically
// discovered during operation registration, such as validation failures or
// missing required parameters.
type DiscoveredError struct {
	Status int
	Title  string
	Detail string
}

type errWithHeaders struct {
	err     error
	headers http.Header
}

func (e *errWithHeaders) Error() string           { return e.err.Error() }
func (e *errWithHeaders) Unwrap() error           { return e.err }
func (e *errWithHeaders) GetHeaders() http.Header { return e.headers }

// ErrorWithHeaders wraps an error with additional HTTP headers that will be
// sent in the response. The returned error implements the Headerer interface.
func ErrorWithHeaders(err error, headers http.Header) error {
	return &errWithHeaders{err: err, headers: headers}
}
