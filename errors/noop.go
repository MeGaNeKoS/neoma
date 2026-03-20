package errors

import (
	"net/http"

	"github.com/MeGaNeKoS/neoma/core"
)

type NoopHandler struct{}

func (f *NoopHandler) NewError(status int, msg string, _ ...error) core.Error {
	if msg == "" {
		msg = http.StatusText(status)
	}
	return &minimalError{status: status, message: msg}
}

func (f *NoopHandler) NewErrorWithContext(_ core.Context, status int, msg string, _ ...error) core.Error {
	return f.NewError(status, msg)
}

func (f *NoopHandler) ErrorSchema(_ core.Registry) *core.Schema {
	return nil
}

func (f *NoopHandler) ErrorContentType(ct string) string {
	return ct
}

type minimalError struct {
	status  int
	message string
}

func (e *minimalError) Error() string    { return e.message }
func (e *minimalError) StatusCode() int   { return e.status }

func NewNoopHandler() *NoopHandler {
	return &NoopHandler{}
}
