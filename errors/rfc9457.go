package errors

import (
	"errors"
	"net/http"
	"reflect"
	"strconv"

	"github.com/MeGaNeKoS/neoma/core"
)

type ProblemDetail struct {
	Type     string              `json:"type,omitempty" format:"uri" default:"about:blank" example:"https://example.com/errors/example" doc:"A URI reference to human-readable documentation for the error."`
	Title    string              `json:"title,omitempty" example:"Bad Request" doc:"A short, human-readable summary of the problem type."`
	Status   int                 `json:"status,omitempty" example:"400" doc:"HTTP status code"`
	Detail   string              `json:"detail,omitempty" example:"Property foo is required but is missing." doc:"A human-readable explanation specific to this occurrence of the problem."`
	Instance string              `json:"instance,omitempty" format:"uri" example:"https://example.com/error-log/abc123" doc:"A URI reference that identifies the specific occurrence of the problem."`
	Errors   []*core.ErrorDetail `json:"errors,omitempty" doc:"Optional list of individual error details"`
}

func (e *ProblemDetail) Error() string {
	return e.Detail
}

func (e *ProblemDetail) StatusCode() int {
	return e.Status
}

func (e *ProblemDetail) GetType() string {
	return e.Type
}

func (e *ProblemDetail) ContentType(ct string) string {
	if ct == "application/json" {
		return "application/problem+json"
	}
	if ct == "application/cbor" {
		return "application/problem+cbor"
	}
	return ct
}

func (e *ProblemDetail) Add(err error) {
	var converted core.ErrorDetailer
	if errors.As(err, &converted) {
		e.Errors = append(e.Errors, converted.ErrorDetail())
		return
	}
	e.Errors = append(e.Errors, &core.ErrorDetail{Message: err.Error()})
}

type RFC9457Handler struct {
	TypeBaseURI  string
	InstanceFunc func(ctx core.Context) string
}

func (f *RFC9457Handler) GetTypeBaseURI() string {
	return f.TypeBaseURI
}

func (f *RFC9457Handler) typeURI(status int) string {
	if f.TypeBaseURI == "" {
		return "about:blank"
	}
	return f.TypeBaseURI + "/" + strconv.Itoa(status)
}

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

func (f *RFC9457Handler) NewErrorWithContext(ctx core.Context, status int, msg string, errs ...error) core.Error {
	se := f.NewError(status, msg, errs...)
	var em *ProblemDetail
	if errors.As(se, &em) && f.InstanceFunc != nil && ctx != nil {
		em.Instance = f.InstanceFunc(ctx)
	}
	return se
}

func (f *RFC9457Handler) ErrorSchema(registry core.Registry) *core.Schema {
	return registry.Schema(reflect.TypeFor[ProblemDetail](), true, "")
}

func (f *RFC9457Handler) ErrorContentType(ct string) string {
	if ct == "application/json" {
		return "application/problem+json"
	}
	if ct == "application/cbor" {
		return "application/problem+cbor"
	}
	return ct
}

func NewRFC9457Handler() *RFC9457Handler {
	return &RFC9457Handler{}
}

func NewRFC9457HandlerWithConfig(typeBaseURI string, instanceFunc func(core.Context) string) *RFC9457Handler {
	return &RFC9457Handler{
		TypeBaseURI:  typeBaseURI,
		InstanceFunc: instanceFunc,
	}
}

var defaultHandler = &RFC9457Handler{}

func ErrorBadRequest(msg string, errs ...error) core.Error {
	return defaultHandler.NewError(http.StatusBadRequest, msg, errs...)
}

func ErrorUnauthorized(msg string, errs ...error) core.Error {
	return defaultHandler.NewError(http.StatusUnauthorized, msg, errs...)
}

func ErrorForbidden(msg string, errs ...error) core.Error {
	return defaultHandler.NewError(http.StatusForbidden, msg, errs...)
}

func ErrorNotFound(msg string, errs ...error) core.Error {
	return defaultHandler.NewError(http.StatusNotFound, msg, errs...)
}

func ErrorConflict(msg string, errs ...error) core.Error {
	return defaultHandler.NewError(http.StatusConflict, msg, errs...)
}

func ErrorUnprocessableEntity(msg string, errs ...error) core.Error {
	return defaultHandler.NewError(http.StatusUnprocessableEntity, msg, errs...)
}

func ErrorTooManyRequests(msg string, errs ...error) core.Error {
	return defaultHandler.NewError(http.StatusTooManyRequests, msg, errs...)
}

func ErrorInternalServerError(msg string, errs ...error) core.Error {
	return defaultHandler.NewError(http.StatusInternalServerError, msg, errs...)
}

func ErrorN(status int, msg string, errs ...error) core.Error {
	return defaultHandler.NewError(status, msg, errs...)
}

func Status304NotModified() core.Error {
	return defaultHandler.NewError(http.StatusNotModified, "")
}

