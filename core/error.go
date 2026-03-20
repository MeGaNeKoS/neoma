package core

import (
	"fmt"
	"net/http"
)

type ErrorDetailer interface {
	ErrorDetail() *ErrorDetail
}

type ErrorDetail struct {
	Message  string `json:"message,omitempty" doc:"Error message text"`
	Location string `json:"location,omitempty" doc:"Where the error occurred, e.g. 'body.items[3].tags'"`
	Value    any    `json:"value,omitempty" doc:"The value at the given location"`
}

func (e *ErrorDetail) Error() string {
	if e.Location == "" && e.Value == nil {
		return e.Message
	}
	return fmt.Sprintf("%s (%s: %v)", e.Message, e.Location, e.Value)
}

func (e *ErrorDetail) ErrorDetail() *ErrorDetail {
	return e
}

type Error interface {
	StatusCode() int
	Error() string
}

type Headerer interface {
	GetHeaders() http.Header
}

type ContentTyper interface {
	ContentType(string) string
}

type Linker interface {
	GetType() string
}

type ErrorHandler interface {
	NewError(status int, msg string, errs ...error) Error
	NewErrorWithContext(ctx Context, status int, msg string, errs ...error) Error
	ErrorSchema(registry Registry) *Schema
	ErrorContentType(ct string) string
}

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

func ErrorWithHeaders(err error, headers http.Header) error {
	return &errWithHeaders{err: err, headers: headers}
}
