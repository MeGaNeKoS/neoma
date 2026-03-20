package errors_test

import (
	stderrors "errors"
	"net/http"
	"testing"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/errors"
	"github.com/MeGaNeKoS/neoma/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorBadRequest(t *testing.T) {
	se := errors.ErrorBadRequest("bad")
	assert.Equal(t, http.StatusBadRequest, se.StatusCode())
}

func TestErrorConflict(t *testing.T) {
	se := errors.ErrorConflict("conflict")
	assert.Equal(t, http.StatusConflict, se.StatusCode())
}

func TestErrorForbidden(t *testing.T) {
	se := errors.ErrorForbidden("forbidden")
	assert.Equal(t, http.StatusForbidden, se.StatusCode())
}

func TestErrorInternalServerError(t *testing.T) {
	se := errors.ErrorInternalServerError("oops")
	assert.Equal(t, http.StatusInternalServerError, se.StatusCode())
}

func TestErrorN(t *testing.T) {
	se := errors.ErrorN(http.StatusTeapot, "I'm a teapot")
	assert.Equal(t, http.StatusTeapot, se.StatusCode())
}

func TestErrorNotFound(t *testing.T) {
	se := errors.ErrorNotFound("not found")
	assert.Equal(t, http.StatusNotFound, se.StatusCode())
}

func TestErrorTooManyRequests(t *testing.T) {
	se := errors.ErrorTooManyRequests("slow down")
	assert.Equal(t, http.StatusTooManyRequests, se.StatusCode())
}

func TestErrorUnauthorized(t *testing.T) {
	se := errors.ErrorUnauthorized("unauth")
	assert.Equal(t, http.StatusUnauthorized, se.StatusCode())
}

func TestErrorUnprocessableEntity(t *testing.T) {
	se := errors.ErrorUnprocessableEntity("invalid")
	assert.Equal(t, http.StatusUnprocessableEntity, se.StatusCode())
}

func TestNoopHandlerEmptyMessage(t *testing.T) {
	f := errors.NewNoopHandler()
	se := f.NewError(http.StatusNotFound, "")
	assert.Equal(t, "Not Found", se.Error())
}

func TestNoopHandlerNewError(t *testing.T) {
	f := errors.NewNoopHandler()
	se := f.NewError(http.StatusBadRequest, "bad")
	assert.Equal(t, http.StatusBadRequest, se.StatusCode())
	assert.Equal(t, "bad", se.Error())
}

func TestNoopHandlerNewErrorWithContext(t *testing.T) {
	f := errors.NewNoopHandler()
	se := f.NewErrorWithContext(nil, http.StatusInternalServerError, "oops")
	assert.Equal(t, http.StatusInternalServerError, se.StatusCode())
}

func TestNoopHandlerNilSchema(t *testing.T) {
	f := errors.NewNoopHandler()
	r := schema.NewMapRegistry(schema.DefaultSchemaNamer)
	s := f.ErrorSchema(r)
	assert.Nil(t, s)
}

func TestNoopContentType(t *testing.T) {
	f := errors.NewNoopHandler()
	assert.Equal(t, "application/json", f.ErrorContentType("application/json"))
}

func TestProblemDetailAdd(t *testing.T) {
	em := &errors.ProblemDetail{Status: 400, Title: "Bad Request", Detail: "test"}

	em.Add(stderrors.New("plain error"))
	assert.Len(t, em.Errors, 1)
	assert.Equal(t, "plain error", em.Errors[0].Message)

	em.Add(&core.ErrorDetail{Message: "field error", Location: "body.x", Value: 42})
	assert.Len(t, em.Errors, 2)
	assert.Equal(t, "field error", em.Errors[1].Message)
}

func TestProblemDetailContentType(t *testing.T) {
	em := &errors.ProblemDetail{}
	assert.Equal(t, "application/problem+json", em.ContentType("application/json"))
	assert.Equal(t, "application/problem+cbor", em.ContentType("application/cbor"))
	assert.Equal(t, "text/html", em.ContentType("text/html"))
}

func TestRFC9457ContentType(t *testing.T) {
	f := errors.NewRFC9457Handler()
	assert.Equal(t, "application/problem+json", f.ErrorContentType("application/json"))
	assert.Equal(t, "application/problem+cbor", f.ErrorContentType("application/cbor"))
	assert.Equal(t, "text/plain", f.ErrorContentType("text/plain"))
}

func TestRFC9457HandlerErrorSchema(t *testing.T) {
	f := errors.NewRFC9457Handler()
	r := schema.NewMapRegistry(schema.DefaultSchemaNamer)
	s := f.ErrorSchema(r)
	require.NotNil(t, s)
	assert.NotEmpty(t, s.Ref)
}

func TestRFC9457HandlerNewError(t *testing.T) {
	f := errors.NewRFC9457Handler()
	se := f.NewError(http.StatusBadRequest, "invalid input")
	assert.Equal(t, http.StatusBadRequest, se.StatusCode())
	assert.Equal(t, "invalid input", se.Error())

	var em *errors.ProblemDetail
	require.ErrorAs(t, se, &em)
	assert.Equal(t, "Bad Request", em.Title)
	assert.Equal(t, http.StatusBadRequest, em.Status)
	assert.Equal(t, "invalid input", em.Detail)
}

func TestRFC9457HandlerNewErrorWithContext(t *testing.T) {
	f := errors.NewRFC9457Handler()
	se := f.NewErrorWithContext(nil, http.StatusForbidden, "forbidden")
	assert.Equal(t, http.StatusForbidden, se.StatusCode())
	assert.Equal(t, "forbidden", se.Error())
}

func TestRFC9457HandlerNilErrorsSkipped(t *testing.T) {
	f := errors.NewRFC9457Handler()
	se := f.NewError(http.StatusBadRequest, "bad", nil, nil)
	var em *errors.ProblemDetail
	require.ErrorAs(t, se, &em)
	assert.Empty(t, em.Errors)
}

func TestRFC9457HandlerWithErrors(t *testing.T) {
	f := errors.NewRFC9457Handler()
	detail := &core.ErrorDetail{Message: "field too short", Location: "body.name", Value: "ab"}
	se := f.NewError(http.StatusUnprocessableEntity, "validation failed", detail)
	var em *errors.ProblemDetail
	require.ErrorAs(t, se, &em)
	assert.Len(t, em.Errors, 1)
	assert.Equal(t, "field too short", em.Errors[0].Message)
	assert.Equal(t, "body.name", em.Errors[0].Location)
	assert.Equal(t, "ab", em.Errors[0].Value)
}

func TestRFC9457HandlerWithPlainError(t *testing.T) {
	f := errors.NewRFC9457Handler()
	se := f.NewError(http.StatusBadRequest, "bad", stderrors.New("something went wrong"))
	var em *errors.ProblemDetail
	require.ErrorAs(t, se, &em)
	assert.Len(t, em.Errors, 1)
	assert.Equal(t, "something went wrong", em.Errors[0].Message)
}

func TestStatus304NotModified(t *testing.T) {
	se := errors.Status304NotModified()
	assert.Equal(t, http.StatusNotModified, se.StatusCode())
}
