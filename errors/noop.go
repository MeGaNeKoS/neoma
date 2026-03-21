package errors

import (
	"net/http"

	"github.com/MeGaNeKoS/neoma/core"
)

// NoopHandler is a minimal error handler that produces plain errors without
// RFC 9457 Problem Details structure. It is useful for testing or when
// structured error responses are not needed.
type NoopHandler struct{}

// NewError creates a minimal error with the given status and message,
// ignoring any additional errors.
func (f *NoopHandler) NewError(status int, msg string, _ ...error) core.Error {
	if msg == "" {
		msg = http.StatusText(status)
	}
	return &minimalError{status: status, message: msg}
}

// NewErrorWithContext creates a minimal error, ignoring the request context.
func (f *NoopHandler) NewErrorWithContext(_ core.Context, status int, msg string, _ ...error) core.Error {
	return f.NewError(status, msg)
}

// ErrorSchema returns nil because NoopHandler does not define an error schema.
func (f *NoopHandler) ErrorSchema(_ core.Registry) *core.Schema {
	return nil
}

// ErrorContentType returns the content type unchanged.
func (f *NoopHandler) ErrorContentType(ct string) string {
	return ct
}

type minimalError struct {
	status  int
	message string
}

func (e *minimalError) Error() string    { return e.message }
func (e *minimalError) StatusCode() int   { return e.status }

// NewNoopHandler returns a new NoopHandler.
func NewNoopHandler() *NoopHandler {
	return &NoopHandler{}
}
