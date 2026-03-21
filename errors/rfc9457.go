// Package errors provides error handling implementations for the neoma
// framework, including RFC 9457 Problem Details support for standardized
// HTTP API error responses.
package errors

import (
	"errors"
	"net/http"
	"reflect"
	"strconv"

	"github.com/MeGaNeKoS/neoma/core"
)

// ProblemDetail represents an RFC 9457 Problem Details object for HTTP APIs.
// It carries a type URI, human-readable title and detail, the HTTP status code,
// and an optional list of granular error details.
type ProblemDetail struct {
	Type     string              `json:"type,omitempty" format:"uri" default:"about:blank" example:"https://example.com/errors/example" doc:"A URI reference to human-readable documentation for the error."`
	Title    string              `json:"title,omitempty" example:"Bad Request" doc:"A short, human-readable summary of the problem type."`
	Status   int                 `json:"status,omitempty" example:"400" doc:"HTTP status code"`
	Detail   string              `json:"detail,omitempty" example:"Property foo is required but is missing." doc:"A human-readable explanation specific to this occurrence of the problem."`
	Instance string              `json:"instance,omitempty" format:"uri" example:"https://example.com/error-log/abc123" doc:"A URI reference that identifies the specific occurrence of the problem."`
	Errors   []*core.ErrorDetail `json:"errors,omitempty" doc:"Optional list of individual error details"`
}

// Error returns the problem detail message, satisfying the error interface.
func (e *ProblemDetail) Error() string {
	return e.Detail
}

// StatusCode returns the HTTP status code for this problem.
func (e *ProblemDetail) StatusCode() int {
	return e.Status
}

// GetType returns the problem type URI.
func (e *ProblemDetail) GetType() string {
	return e.Type
}

// ContentType returns the appropriate RFC 9457 content type for the given
// base content type (e.g. "application/problem+json" for "application/json").
func (e *ProblemDetail) ContentType(ct string) string {
	if ct == "application/json" {
		return "application/problem+json"
	}
	if ct == "application/cbor" {
		return "application/problem+cbor"
	}
	return ct
}

// Add appends an error to the problem's error details list. If the error
// implements core.ErrorDetailer, its structured detail is used directly.
func (e *ProblemDetail) Add(err error) {
	var converted core.ErrorDetailer
	if errors.As(err, &converted) {
		e.Errors = append(e.Errors, converted.ErrorDetail())
		return
	}
	e.Errors = append(e.Errors, &core.ErrorDetail{Message: err.Error()})
}

// RFC9457Handler creates RFC 9457 Problem Details errors. It implements the
// core.ErrorHandler interface and supports configurable type URIs and
// per-request instance URIs.
type RFC9457Handler struct {
	TypeBaseURI  string
	InstanceFunc func(ctx core.Context) string
}

// GetTypeBaseURI returns the base URI used to construct problem type URIs.
func (f *RFC9457Handler) GetTypeBaseURI() string {
	return f.TypeBaseURI
}

func (f *RFC9457Handler) typeURI(status int) string {
	if f.TypeBaseURI == "" {
		return "about:blank"
	}
	return f.TypeBaseURI + "/" + strconv.Itoa(status)
}

// NewError creates a new ProblemDetail with the given status, message, and
// optional underlying errors.
func (f *RFC9457Handler) NewError(status int, msg string, errs ...error) core.Error {
	details := make([]*core.ErrorDetail, 0, len(errs))
	for _, err := range errs {
		if err == nil {
			continue
		}
		var converted core.ErrorDetailer
		if errors.As(err, &converted) {
			details = append(details, converted.ErrorDetail())
		} else {
			details = append(details, &core.ErrorDetail{Message: err.Error()})
		}
	}
	return &ProblemDetail{
		Type:   f.typeURI(status),
		Status: status,
		Title:  http.StatusText(status),
		Detail: msg,
		Errors: details,
	}
}

// NewErrorWithContext creates a new ProblemDetail like NewError, but also
// sets the instance URI using the configured InstanceFunc and request context.
func (f *RFC9457Handler) NewErrorWithContext(ctx core.Context, status int, msg string, errs ...error) core.Error {
	se := f.NewError(status, msg, errs...)
	var em *ProblemDetail
	if errors.As(se, &em) && f.InstanceFunc != nil && ctx != nil {
		em.Instance = f.InstanceFunc(ctx)
	}
	return se
}

// ErrorSchema returns the JSON Schema for ProblemDetail using the given registry.
func (f *RFC9457Handler) ErrorSchema(registry core.Registry) *core.Schema {
	return registry.Schema(reflect.TypeFor[ProblemDetail](), true, "")
}

// ErrorContentType returns the RFC 9457 content type for error responses.
func (f *RFC9457Handler) ErrorContentType(ct string) string {
	if ct == "application/json" {
		return "application/problem+json"
	}
	if ct == "application/cbor" {
		return "application/problem+cbor"
	}
	return ct
}

// NewRFC9457Handler returns a new RFC9457Handler with default settings.
func NewRFC9457Handler() *RFC9457Handler {
	return &RFC9457Handler{}
}

// NewRFC9457HandlerWithConfig returns a new RFC9457Handler with a custom type
// base URI and an optional function to generate per-request instance URIs.
func NewRFC9457HandlerWithConfig(typeBaseURI string, instanceFunc func(core.Context) string) *RFC9457Handler {
	return &RFC9457Handler{
		TypeBaseURI:  typeBaseURI,
		InstanceFunc: instanceFunc,
	}
}

var defaultHandler = &RFC9457Handler{}

// ErrorBadRequest creates a 400 Bad Request problem detail error.
func ErrorBadRequest(msg string, errs ...error) core.Error {
	return defaultHandler.NewError(http.StatusBadRequest, msg, errs...)
}

// ErrorUnauthorized creates a 401 Unauthorized problem detail error.
func ErrorUnauthorized(msg string, errs ...error) core.Error {
	return defaultHandler.NewError(http.StatusUnauthorized, msg, errs...)
}

// ErrorForbidden creates a 403 Forbidden problem detail error.
func ErrorForbidden(msg string, errs ...error) core.Error {
	return defaultHandler.NewError(http.StatusForbidden, msg, errs...)
}

// ErrorNotFound creates a 404 Not Found problem detail error.
func ErrorNotFound(msg string, errs ...error) core.Error {
	return defaultHandler.NewError(http.StatusNotFound, msg, errs...)
}

// ErrorConflict creates a 409 Conflict problem detail error.
func ErrorConflict(msg string, errs ...error) core.Error {
	return defaultHandler.NewError(http.StatusConflict, msg, errs...)
}

// ErrorUnprocessableEntity creates a 422 Unprocessable Entity problem detail error.
func ErrorUnprocessableEntity(msg string, errs ...error) core.Error {
	return defaultHandler.NewError(http.StatusUnprocessableEntity, msg, errs...)
}

// ErrorTooManyRequests creates a 429 Too Many Requests problem detail error.
func ErrorTooManyRequests(msg string, errs ...error) core.Error {
	return defaultHandler.NewError(http.StatusTooManyRequests, msg, errs...)
}

// ErrorInternalServerError creates a 500 Internal Server Error problem detail error.
func ErrorInternalServerError(msg string, errs ...error) core.Error {
	return defaultHandler.NewError(http.StatusInternalServerError, msg, errs...)
}

// ErrorN creates a problem detail error with an arbitrary HTTP status code.
func ErrorN(status int, msg string, errs ...error) core.Error {
	return defaultHandler.NewError(status, msg, errs...)
}

// Status304NotModified creates a 304 Not Modified error, typically used for
// conditional request handling.
func Status304NotModified() core.Error {
	return defaultHandler.NewError(http.StatusNotModified, "")
}

