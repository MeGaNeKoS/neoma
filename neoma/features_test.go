package neoma_test

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/neoma"
	"github.com/MeGaNeKoS/neoma/schema"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)


func TestUUIDFieldFormat(t *testing.T) {
	// UUID fields with a `format:"uuid"` tag should produce a schema with
	// type=string, format=uuid. The google/uuid.UUID type implements
	// encoding.TextUnmarshaler, so it naturally becomes type=string.
	// The format tag adds the uuid format.
	r := schema.NewMapRegistry(schema.DefaultSchemaNamer)

	type Resource struct {
		ID uuid.UUID `json:"id" format:"uuid"`
	}

	s := r.Schema(reflect.TypeFor[Resource](), false, "Resource")
	require.NotNil(t, s)
	assert.Equal(t, core.TypeObject, s.Type)

	idProp := s.Properties["id"]
	require.NotNil(t, idProp)
	assert.Equal(t, core.TypeString, idProp.Type)
	assert.Equal(t, "uuid", idProp.Format)
}

func TestUUIDFieldValidation(t *testing.T) {
	r := schema.NewMapRegistry(schema.DefaultSchemaNamer)
	s := &core.Schema{
		Type:   core.TypeString,
		Format: "uuid",
	}
	s.PrecomputeMessages()

	pb := core.NewPathBuffer([]byte(""), 0)
	res := &core.ValidateResult{}

	schema.Validate(r, s, pb, core.ModeWriteToServer, "550e8400-e29b-41d4-a716-446655440000", res)
	assert.Empty(t, res.Errors, "valid UUID should pass validation")

	res.Reset()
	pb.Reset()
	schema.Validate(r, s, pb, core.ModeWriteToServer, "not-a-uuid", res)
	assert.NotEmpty(t, res.Errors, "invalid UUID should fail validation")
}

func TestUUIDFieldEndToEnd(t *testing.T) {
	api := newTestAPI(t)

	type Input struct {
		Body struct {
			ID uuid.UUID `json:"id" format:"uuid"`
		}
	}
	type Output struct {
		Body struct {
			ID string `json:"id"`
		}
	}

	neoma.Post[Input, Output](api, "/resources", func(_ context.Context, in *Input) (*Output, error) {
		o := &Output{}
		o.Body.ID = in.Body.ID.String()
		return o, nil
	})

	resp := api.Do(http.MethodPost, "/resources", map[string]any{
		"id": "550e8400-e29b-41d4-a716-446655440000",
	})
	assert.Equal(t, http.StatusOK, resp.Code)

	resp = api.Do(http.MethodPost, "/resources", map[string]any{
		"id": "not-a-uuid",
	})
	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
}


func TestExcludeHiddenSchemas(t *testing.T) {
	// When ExcludeHiddenSchemas is set in config, hidden operations should
	// not contribute their request/response schemas to the OpenAPI spec.
	config := neoma.DefaultConfig("Test API", "1.0.0")
	config.ExcludeHiddenSchemas = true
	api := newTestAPI(t, config)

	type SecretOutput struct {
		Body struct {
			Internal string `json:"internal"`
		}
	}

	neoma.Get[struct{}, SecretOutput](api, "/internal", func(_ context.Context, _ *struct{}) (*SecretOutput, error) {
		o := &SecretOutput{}
		o.Body.Internal = "secret"
		return o, nil
	}, func(op *core.Operation) {
		op.Hidden = true
	})

	type PublicOutput struct {
		Body struct {
			Public string `json:"public"`
		}
	}
	neoma.Get[struct{}, PublicOutput](api, "/public", func(_ context.Context, _ *struct{}) (*PublicOutput, error) {
		o := &PublicOutput{}
		o.Body.Public = "visible"
		return o, nil
	})

	oapi := api.OpenAPI()
	assert.Nil(t, oapi.Paths["/internal"], "hidden operation should not appear in paths")
	assert.NotNil(t, oapi.Paths["/public"], "public operation should appear in paths")

	resp := api.Do(http.MethodGet, "/internal")
	assert.Equal(t, http.StatusOK, resp.Code)

	// Public spec should not contain hidden schemas.
	specResp := api.Do(http.MethodGet, "/openapi.json")
	assert.Equal(t, http.StatusOK, specResp.Code)

	var spec map[string]any
	require.NoError(t, json.Unmarshal(specResp.Body.Bytes(), &spec))
	components, _ := spec["components"].(map[string]any)
	schemas, _ := components["schemas"].(map[string]any)

	assert.NotNil(t, schemas["PublicOutputBody"], "public schema should be in spec")
	assert.Nil(t, schemas["SecretOutputBody"], "hidden schema should not be in spec")

	// Individual schema endpoint should 404 for hidden schemas.
	schemaResp := api.Do(http.MethodGet, "/schemas/SecretOutputBody.json")
	assert.Equal(t, http.StatusNotFound, schemaResp.Code)

	publicSchemaResp := api.Do(http.MethodGet, "/schemas/PublicOutputBody.json")
	assert.Equal(t, http.StatusOK, publicSchemaResp.Code)
}


type CustomAppError struct {
	Status  int    `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *CustomAppError) Error() string {
	return e.Message
}

func (e *CustomAppError) StatusCode() int {
	return e.Status
}

func TestHandlerReturnsCustomError(t *testing.T) {
	// When a handler returns a custom type implementing Error, the
	// framework should extract its status code and route the error
	// through the ErrorHandler envelope.
	api := newTestAPI(t)

	type Output struct {
		Body struct {
			Data string `json:"data"`
		}
	}

	neoma.Get[struct{}, Output](api, "/custom-err", func(_ context.Context, _ *struct{}) (*Output, error) {
		return nil, &CustomAppError{
			Status:  http.StatusConflict,
			Code:    "DUPLICATE",
			Message: "resource already exists",
		}
	})

	resp := api.Do(http.MethodGet, "/custom-err")
	assert.Equal(t, http.StatusConflict, resp.Code)

	// The default RFC 9457 handler wraps the error in ProblemDetail.
	var body map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.InDelta(t, 409, body["status"], 0)
	assert.Equal(t, "resource already exists", body["detail"])
}

func TestHandlerReturnsErrorWithHeaders(t *testing.T) {
	// When a handler returns an error that implements both Error and
	// HeadersError, both the status and headers should be set.
	api := newTestAPI(t)

	type Output struct {
		Body struct {
			OK bool `json:"ok"`
		}
	}

	neoma.Get[struct{}, Output](api, "/rate-limited", func(_ context.Context, _ *struct{}) (*Output, error) {
		err := &CustomAppError{
			Status:  http.StatusTooManyRequests,
			Code:    "RATE_LIMITED",
			Message: "too many requests",
		}
		return nil, core.ErrorWithHeaders(err, http.Header{
			"Retry-After": {"60"},
		})
	})

	resp := api.Do(http.MethodGet, "/rate-limited")
	assert.Equal(t, http.StatusTooManyRequests, resp.Code)
	assert.Equal(t, "60", resp.Header().Get("Retry-After"))
}


func TestReadOnlyNotRequiredOnWrite(t *testing.T) {
	// Fields marked readOnly should not be required when the client sends
	// data to the server (ModeWriteToServer). This allows servers to
	// compute fields like "id" or "createdAt" without the client needing
	// to send them.
	r := schema.NewMapRegistry(schema.DefaultSchemaNamer)

	type Resource struct {
		ID   string `json:"id" readOnly:"true"`
		Name string `json:"name"`
	}

	s := r.Schema(reflect.TypeFor[Resource](), false, "Resource")

	require.NotNil(t, s)
	assert.Contains(t, s.Required, "id")
	assert.Contains(t, s.Required, "name")

	// But during ModeWriteToServer validation, a missing readOnly field
	// should not cause an error.
	pb := core.NewPathBuffer([]byte(""), 0)
	res := &core.ValidateResult{}

	input := map[string]any{
		"name": "Alice",
	}
	schema.Validate(r, s, pb, core.ModeWriteToServer, input, res)
	assert.Empty(t, res.Errors, "missing readOnly field should not fail validation on write")
}

func TestReadOnlyFieldAcceptedOnWrite(t *testing.T) {
	r := schema.NewMapRegistry(schema.DefaultSchemaNamer)

	type Resource struct {
		ID   string `json:"id" readOnly:"true"`
		Name string `json:"name"`
	}

	s := r.Schema(reflect.TypeFor[Resource](), false, "Resource")
	pb := core.NewPathBuffer([]byte(""), 0)
	res := &core.ValidateResult{}

	input := map[string]any{
		"id":   "123",
		"name": "Alice",
	}
	schema.Validate(r, s, pb, core.ModeWriteToServer, input, res)
	assert.Empty(t, res.Errors, "sending readOnly field on write should still be accepted (permissive)")
}

func TestReadOnlyRequiredOnRead(t *testing.T) {
	// When reading from server (response validation), readOnly fields
	// should be required if they are listed in required.
	r := schema.NewMapRegistry(schema.DefaultSchemaNamer)

	type Resource struct {
		ID   string `json:"id" readOnly:"true"`
		Name string `json:"name"`
	}

	s := r.Schema(reflect.TypeFor[Resource](), false, "Resource")
	pb := core.NewPathBuffer([]byte(""), 0)
	res := &core.ValidateResult{}

	input := map[string]any{
		"name": "Alice",
	}
	schema.Validate(r, s, pb, core.ModeReadFromServer, input, res)
	assert.NotEmpty(t, res.Errors, "missing readOnly field should fail validation on read from server")
}

func TestReadOnlyEndToEnd(t *testing.T) {
	api := newTestAPI(t)

	type Input struct {
		Body struct {
			ID   string `json:"id" readOnly:"true"`
			Name string `json:"name"`
		}
	}
	type Output struct {
		Body struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
	}

	neoma.Post[Input, Output](api, "/readonly-test", func(_ context.Context, in *Input) (*Output, error) {
		o := &Output{}
		o.Body.ID = "generated-id"
		o.Body.Name = in.Body.Name
		return o, nil
	})

	resp := api.Do(http.MethodPost, "/readonly-test", map[string]any{
		"name": "Bob",
	})
	assert.Equal(t, http.StatusOK, resp.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, "generated-id", body["id"])
	assert.Equal(t, "Bob", body["name"])
}
