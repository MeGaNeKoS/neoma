package openapi_test

import (
	"net/http"
	"testing"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/errors"
	"github.com/MeGaNeKoS/neoma/openapi"
	"github.com/MeGaNeKoS/neoma/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRegistry() core.Registry {
	return schema.NewMapRegistry(schema.DefaultSchemaNamer)
}


func TestDefineErrorsRFC9457(t *testing.T) {
	registry := newRegistry()
	factory := errors.NewRFC9457Handler()

	op := &core.Operation{
		Method:      "GET",
		Path:        "/test",
		OperationID: "test",
		Errors:      []int{http.StatusBadRequest, http.StatusInternalServerError},
		Responses:   map[string]*core.Response{},
	}

	openapi.DefineErrors(op, registry, factory)

	assert.NotNil(t, op.Responses["400"], "should have 400 response")
	assert.NotNil(t, op.Responses["500"], "should have 500 response")

	resp400 := op.Responses["400"]
	assert.NotNil(t, resp400.Content)
	assert.Equal(t, "Bad Request", resp400.Description)

	resp500 := op.Responses["500"]
	assert.NotNil(t, resp500.Content)
	assert.Equal(t, "Internal Server Error", resp500.Description)

	// RFC 9457 uses application/problem+json
	assert.NotNil(t, resp400.Content["application/problem+json"])
	assert.NotNil(t, resp400.Content["application/problem+json"].Schema)
}


func TestDefineErrorsNoop(t *testing.T) {
	registry := newRegistry()
	factory := errors.NewNoopHandler()

	op := &core.Operation{
		Method:      "GET",
		Path:        "/test",
		OperationID: "test",
		Errors:      []int{http.StatusBadRequest, http.StatusInternalServerError},
		Responses:   map[string]*core.Response{},
	}

	openapi.DefineErrors(op, registry, factory)

	assert.Nil(t, op.Responses["400"], "should not have 400 response with NoopHandler")
	assert.Nil(t, op.Responses["500"], "should not have 500 response with NoopHandler")
}


func TestDefineErrorsWithErrorHeaders(t *testing.T) {
	registry := newRegistry()
	factory := errors.NewRFC9457Handler()

	op := &core.Operation{
		Method:      "GET",
		Path:        "/test",
		OperationID: "test",
		Errors:      []int{http.StatusTooManyRequests},
		Responses:   map[string]*core.Response{},
		ErrorHeaders: map[string]*core.Param{
			"Retry-After": {
				Schema: &core.Schema{Type: core.TypeInteger},
			},
			"X-RateLimit-Remaining": {
				Schema: &core.Schema{Type: core.TypeInteger},
			},
		},
	}

	openapi.DefineErrors(op, registry, factory)

	resp429 := op.Responses["429"]
	require.NotNil(t, resp429)
	assert.NotNil(t, resp429.Headers)
	assert.NotNil(t, resp429.Headers["Retry-After"], "should have Retry-After header")
	assert.NotNil(t, resp429.Headers["X-RateLimit-Remaining"], "should have X-RateLimit-Remaining header")
}


func TestDefineErrorsWithErrorResponses(t *testing.T) {
	registry := newRegistry()
	factory := errors.NewRFC9457Handler()

	customSchema := &core.Schema{
		Type: core.TypeObject,
		Properties: map[string]*core.Schema{
			"code": {Type: core.TypeString},
		},
	}

	op := &core.Operation{
		Method:      "POST",
		Path:        "/test",
		OperationID: "test",
		Errors:      []int{http.StatusBadRequest, http.StatusConflict},
		Responses:   map[string]*core.Response{},
		ErrorResponses: map[int]*core.ErrorResponseConfig{
			http.StatusConflict: {
				Description: "Duplicate resource",
				Schema:      customSchema,
				Headers: map[string]*core.Param{
					"X-Conflict-ID": {
						Schema: &core.Schema{Type: core.TypeString},
					},
				},
			},
		},
	}

	openapi.DefineErrors(op, registry, factory)

	resp400 := op.Responses["400"]
	require.NotNil(t, resp400)
	assert.Equal(t, "Bad Request", resp400.Description)

	resp409 := op.Responses["409"]
	require.NotNil(t, resp409)
	assert.Equal(t, "Duplicate resource", resp409.Description)

	assert.NotNil(t, resp409.Headers)
	assert.NotNil(t, resp409.Headers["X-Conflict-ID"])

	ct := factory.ErrorContentType("application/json")
	assert.NotNil(t, resp409.Content[ct])
	assert.Equal(t, customSchema, resp409.Content[ct].Schema)
}


func TestDefineErrorsDefaultResponse(t *testing.T) {
	registry := newRegistry()
	factory := errors.NewRFC9457Handler()

	op := &core.Operation{
		Method:      "GET",
		Path:        "/test",
		OperationID: "test",
		Errors:      []int{}, // no explicit errors
		Responses: map[string]*core.Response{
			"200": {Description: "OK"},
		},
	}

	openapi.DefineErrors(op, registry, factory)

	assert.NotNil(t, op.Responses["default"], "should have a default error response")
	assert.Equal(t, "Error", op.Responses["default"].Description)

	ct := factory.ErrorContentType("application/json")
	assert.NotNil(t, op.Responses["default"].Content[ct])
	assert.NotNil(t, op.Responses["default"].Content[ct].Schema)
}


func TestDefineErrorsSkipsExisting(t *testing.T) {
	registry := newRegistry()
	factory := errors.NewRFC9457Handler()

	existingResp := &core.Response{
		Description: "My custom 400",
	}

	op := &core.Operation{
		Method:      "GET",
		Path:        "/test",
		OperationID: "test",
		Errors:      []int{http.StatusBadRequest},
		Responses: map[string]*core.Response{
			"400": existingResp,
		},
	}

	openapi.DefineErrors(op, registry, factory)

	require.NotNil(t, op.Responses["400"])
	assert.Equal(t, existingResp, op.Responses["400"])
	assert.Equal(t, "My custom 400", op.Responses["400"].Description)
}


func TestDefineErrorsDefaultWithErrorHeaders(t *testing.T) {
	registry := newRegistry()
	factory := errors.NewRFC9457Handler()

	op := &core.Operation{
		Method:      "GET",
		Path:        "/test",
		OperationID: "test",
		Errors:      []int{},
		Responses:   map[string]*core.Response{},
		ErrorHeaders: map[string]*core.Param{
			"X-Request-ID": {
				Schema: &core.Schema{Type: core.TypeString},
			},
		},
	}

	openapi.DefineErrors(op, registry, factory)

	assert.NotNil(t, op.Responses["default"])
	assert.NotNil(t, op.Responses["default"].Headers)
	assert.NotNil(t, op.Responses["default"].Headers["X-Request-ID"])
}


func TestDefineErrorsNoDefaultWhenMultipleResponses(t *testing.T) {
	registry := newRegistry()
	factory := errors.NewRFC9457Handler()

	op := &core.Operation{
		Method:      "GET",
		Path:        "/test",
		OperationID: "test",
		Errors:      []int{},
		Responses: map[string]*core.Response{
			"200": {Description: "OK"},
			"301": {Description: "Redirect"},
		},
	}

	openapi.DefineErrors(op, registry, factory)

	assert.Nil(t, op.Responses["default"], "should not create default error with multiple existing responses")
}

func TestDefineErrorsNoopHandler(t *testing.T) {
	registry := newRegistry()
	factory := errors.NewNoopHandler()

	op := &core.Operation{
		Method:      "GET",
		Path:        "/test",
		OperationID: "test",
		Errors:      []int{http.StatusBadRequest},
		Responses:   map[string]*core.Response{},
	}

	openapi.DefineErrors(op, registry, factory)

	assert.Nil(t, op.Responses["400"])
}
