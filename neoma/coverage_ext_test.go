package neoma_test

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/errors"
	"github.com/MeGaNeKoS/neoma/neoma"
	"github.com/MeGaNeKoS/neoma/neomatest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUseGlobalMiddleware(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	var order []string

	api.UseMiddleware(func(ctx core.Context, next func(core.Context)) {
		order = append(order, "normal")
		next(ctx)
	})

	api.UseGlobalMiddleware(func(ctx core.Context, next func(core.Context)) {
		order = append(order, "global")
		next(ctx)
	})

	type Output struct {
		Body struct {
			OK bool `json:"ok"`
		}
	}

	neoma.Get[struct{}, Output](api, "/global-mw", func(_ context.Context, _ *struct{}) (*Output, error) {
		order = append(order, "handler")
		return &Output{}, nil
	})

	resp := api.Do(http.MethodGet, "/global-mw")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, []string{"global", "normal", "handler"}, order)
}

func TestAPIConfigAccessor(t *testing.T) {
	config := neoma.DefaultConfig("ConfigTest", "2.0.0")
	_, api := neomatest.New(t, config)

	oapi := api.OpenAPI()
	assert.Equal(t, "ConfigTest", oapi.Info.Title)
}

func TestDocumentOperationHidden(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.InternalSpec = core.InternalSpecConfig{
		Path:    "/internal/openapi",
		Enabled: true,
	}
	_, api := neomatest.New(t, config)

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/secret",
		OperationID: "get-secret",
		Hidden:      true,
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/public",
		OperationID: "get-public",
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	oapi := api.OpenAPI()
	assert.Nil(t, oapi.Paths["/secret"])
	assert.NotNil(t, oapi.Paths["/public"])

	resp := api.Do(http.MethodGet, "/internal/openapi.json")
	assert.Equal(t, http.StatusOK, resp.Code)

	var spec map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &spec))
	paths := spec["paths"].(map[string]any)
	assert.Contains(t, paths, "/secret")
	assert.Contains(t, paths, "/public")
}

func TestTransformSuccess(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.Transformers = []core.Transformer{
		func(ctx core.Context, status string, v any) (any, error) {
			return v, nil
		},
	}
	_, api := neomatest.New(t, config)

	type Output struct {
		Body struct {
			Msg string `json:"msg"`
		}
	}

	neoma.Get[struct{}, Output](api, "/transform-ok", func(_ context.Context, _ *struct{}) (*Output, error) {
		o := &Output{}
		o.Body.Msg = "ok"
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/transform-ok")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "ok")
}

func TestCustomGenerateOperationID(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.GenerateOperationID = func(method, path string) string {
		return "custom-" + strings.ToLower(method) + "-op"
	}
	_, api := neomatest.New(t, config)

	neoma.Get[struct{}, struct{}](api, "/custom-id", func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	oapi := api.OpenAPI()
	pi := oapi.Paths["/custom-id"]
	require.NotNil(t, pi)
	require.NotNil(t, pi.Get)
	assert.Equal(t, "custom-get-op", pi.Get.OperationID)
}

func TestNewAPIWith30Version(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.OpenAPIVersion = core.OpenAPIVersion30

	_, api := neomatest.New(t, config)

	neoma.Get[struct{}, struct{}](api, "/v30", func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/openapi.json")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), core.OpenAPIVersion30)
}

func TestCreateHooks(t *testing.T) {
	hookCalled := false
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.CreateHooks = []func(core.Config) core.Config{
		func(c core.Config) core.Config {
			hookCalled = true
			return c
		},
	}
	_, _ = neomatest.New(t, config)
	assert.True(t, hookCalled)
}

func TestRejectUnknownQueryParameters(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Input struct {
		Name string `query:"name"`
	}
	type Output struct {
		Body struct {
			Name string `json:"name"`
		}
	}

	neoma.Register(api, core.Operation{
		Method:                       http.MethodGet,
		Path:                         "/search",
		OperationID:                  "search",
		RejectUnknownQueryParameters: true,
	}, func(_ context.Context, in *Input) (*Output, error) {
		o := &Output{}
		o.Body.Name = in.Name
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/search?name=hello")
	assert.Equal(t, http.StatusOK, resp.Code)

	resp = api.Do(http.MethodGet, "/search?name=hello&unknown=bad")
	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
}

func TestRequiredParamMissing(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Input struct {
		ID string `query:"id" required:"true"`
	}
	type Output struct {
		Body struct {
			ID string `json:"id"`
		}
	}

	neoma.Get[Input, Output](api, "/required-param", func(_ context.Context, in *Input) (*Output, error) {
		o := &Output{}
		o.Body.ID = in.ID
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/required-param")
	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
}

func TestCookieParameter(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Input struct {
		SessionID string `cookie:"session_id"`
	}
	type Output struct {
		Body struct {
			SessionID string `json:"session_id"`
		}
	}

	neoma.Get[Input, Output](api, "/cookie-param", func(_ context.Context, in *Input) (*Output, error) {
		o := &Output{}
		o.Body.SessionID = in.SessionID
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/cookie-param", "Cookie: session_id=abc123")
	assert.Equal(t, http.StatusOK, resp.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, "abc123", body["session_id"])
}

func TestBodyReadTimeout(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Input struct {
		Body struct {
			Data string `json:"data"`
		}
	}

	neoma.Register(api, core.Operation{
		Method:          http.MethodPost,
		Path:            "/pos-timeout",
		OperationID:     "pos-timeout",
		BodyReadTimeout: 5000000000, // 5s
	}, func(_ context.Context, in *Input) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodPost, "/pos-timeout", map[string]any{"data": "hello"})
	assert.Equal(t, http.StatusNoContent, resp.Code)

	neoma.Register(api, core.Operation{
		Method:          http.MethodPost,
		Path:            "/neg-timeout",
		OperationID:     "neg-timeout",
		BodyReadTimeout: -1,
	}, func(_ context.Context, in *Input) (*struct{}, error) {
		return nil, nil
	})

	resp = api.Do(http.MethodPost, "/neg-timeout", map[string]any{"data": "hello"})
	assert.Equal(t, http.StatusNoContent, resp.Code)
}

func TestBodyContentTypeInference(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Input struct {
		Body struct {
			Msg string `json:"msg"`
		}
	}
	type Output struct {
		Body struct {
			Msg string `json:"msg"`
		}
	}

	neoma.Post[Input, Output](api, "/infer-ct", func(_ context.Context, in *Input) (*Output, error) {
		o := &Output{}
		o.Body.Msg = in.Body.Msg
		return o, nil
	})

	resp := api.Do(http.MethodPost, "/infer-ct", strings.NewReader(`{"msg":"hello"}`), "Content-Type: application/json")
	assert.Equal(t, http.StatusOK, resp.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, "hello", body["msg"])
}

func TestEmptyBodyOnPost(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Input struct {
		Body struct {
			Name string `json:"name" required:"true"`
		}
	}

	neoma.Post[Input, struct{}](api, "/empty-body", func(_ context.Context, _ *Input) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodPost, "/empty-body", strings.NewReader(""), "Content-Type: application/json")
	assert.GreaterOrEqual(t, resp.Code, 400)
}

type resolverInput struct {
	Token string `header:"Authorization"`
}

func (r *resolverInput) Resolve(_ core.Context) []error {
	if r.Token == "" {
		return []error{stderrors.New("authorization required")}
	}
	if r.Token != "Bearer valid" {
		return []error{&core.ErrorDetail{
			Message:  "invalid token",
			Location: "header.Authorization",
			Value:    r.Token,
		}}
	}
	return nil
}

func TestResolverErrors(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Output struct {
		Body struct {
			OK bool `json:"ok"`
		}
	}

	neoma.Get[resolverInput, Output](api, "/resolver", func(_ context.Context, _ *resolverInput) (*Output, error) {
		o := &Output{}
		o.Body.OK = true
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/resolver")
	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)

	resp = api.Do(http.MethodGet, "/resolver", "Authorization: Bearer invalid")
	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)

	resp = api.Do(http.MethodGet, "/resolver", "Authorization: Bearer valid")
	assert.Equal(t, http.StatusOK, resp.Code)
}

type statusResolverInput struct{}

func (r *statusResolverInput) Resolve(_ core.Context) []error {
	return []error{errors.ErrorForbidden("access denied")}
}

func TestResolverStatusError(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Output struct {
		Body struct {
			OK bool `json:"ok"`
		}
	}

	neoma.Get[statusResolverInput, Output](api, "/resolver-status", func(_ context.Context, _ *statusResolverInput) (*Output, error) {
		o := &Output{}
		o.Body.OK = true
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/resolver-status")
	assert.Equal(t, http.StatusForbidden, resp.Code)
}

func TestHandlerErrorWithHeaders(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	neoma.Get[struct{}, struct{}](api, "/err-headers", func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, core.ErrorWithHeaders(
			errors.ErrorTooManyRequests("slow down"),
			http.Header{"Retry-After": {"120"}},
		)
	})

	resp := api.Do(http.MethodGet, "/err-headers")
	assert.Equal(t, http.StatusTooManyRequests, resp.Code)
	assert.Equal(t, "120", resp.Header().Get("Retry-After"))
}

func TestRawBytesBody(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Output struct {
		Body []byte
	}

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/raw-bytes",
		OperationID: "raw-bytes",
	}, func(_ context.Context, _ *struct{}) (*Output, error) {
		return &Output{Body: []byte("raw data")}, nil
	})

	resp := api.Do(http.MethodGet, "/raw-bytes")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "raw data", resp.Body.String())
}

func TestNilOutputReturnsDefaultStatus(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	neoma.Register(api, core.Operation{
		Method:      http.MethodDelete,
		Path:        "/nil-output",
		OperationID: "nil-output",
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodDelete, "/nil-output")
	assert.Equal(t, http.StatusNoContent, resp.Code)
}

func TestOutputContentTypeHeader(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Output struct {
		ContentType string `header:"Content-Type"`
		Body        struct {
			Msg string `json:"msg"`
		}
	}

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/ct-header",
		OperationID: "ct-header",
	}, func(_ context.Context, _ *struct{}) (*Output, error) {
		o := &Output{ContentType: "application/json; charset=utf-8"}
		o.Body.Msg = "hello"
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/ct-header")
	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestOutputSliceHeader(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Output struct {
		Links []string `header:"Link"`
		Body  struct {
			OK bool `json:"ok"`
		}
	}

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/slice-header",
		OperationID: "slice-header",
	}, func(_ context.Context, _ *struct{}) (*Output, error) {
		o := &Output{}
		o.Links = []string{"<https://example.com>; rel=first", "<https://example.com?p=2>; rel=next"}
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/slice-header")
	assert.Equal(t, http.StatusOK, resp.Code)
	links := resp.Result().Header["Link"]
	assert.Len(t, links, 2)
}

func TestHandlerGenericError(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Output struct {
		Body struct {
			Data string `json:"data"`
		}
	}

	neoma.Get[struct{}, Output](api, "/generic-err", func(_ context.Context, _ *struct{}) (*Output, error) {
		return nil, stderrors.New("something went wrong")
	})

	resp := api.Do(http.MethodGet, "/generic-err")
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestRegisterDiscoveredErrorsAutoPopulate(t *testing.T) {
	neoma.RegisterDiscoveredErrors(map[string][]core.DiscoveredError{
		"handleItem": {
			{Status: 404, Title: "Not Found", Detail: "Item not found"},
			{Status: 409, Title: "Conflict", Detail: "Item already exists"},
		},
	})

	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/auto-errors",
		OperationID: "handleItem",
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	oapi := api.OpenAPI()
	pi := oapi.Paths["/auto-errors"]
	require.NotNil(t, pi)
	require.NotNil(t, pi.Get)
	assert.Contains(t, pi.Get.Responses, "404")
	assert.Contains(t, pi.Get.Responses, "409")
}

func TestConvenienceListSummary(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Output struct {
		Body []struct {
			ID int `json:"id"`
		}
	}

	neoma.Get[struct{}, Output](api, "/items", func(_ context.Context, _ *struct{}) (*Output, error) {
		return nil, nil
	})

	oapi := api.OpenAPI()
	pi := oapi.Paths["/items"]
	require.NotNil(t, pi)
	require.NotNil(t, pi.Get)
	assert.Contains(t, strings.ToLower(pi.Get.Summary), "list")
	assert.Contains(t, strings.ToLower(pi.Get.OperationID), "list")
}

func TestWriteErrWithTypeURI(t *testing.T) {
	factory := errors.NewRFC9457HandlerWithConfig("/errors", nil)
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.ErrorHandler = factory
	_, api := neomatest.New(t, config)

	type Input struct {
		Body struct {
			Email string `json:"email" format:"email" required:"true"`
		}
	}

	neoma.Post[Input, struct{}](api, "/write-err-link", func(_ context.Context, _ *Input) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodPost, "/write-err-link", map[string]any{"email": "not-an-email"})
	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	linkHeader := resp.Header().Get("Link")
	assert.Contains(t, linkHeader, "rel=\"type\"")
}

func TestValidationFailureFallbackToResErrors(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Input struct {
		Body struct {
			Email string `json:"email" format:"email"`
		}
	}

	neoma.Post[Input, struct{}](api, "/validate-fallback", func(_ context.Context, _ *Input) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodPost, "/validate-fallback", map[string]any{"email": "not-an-email"})
	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
}

func TestRegisterPanicNoMethod(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))
	assert.Panics(t, func() {
		neoma.Register(api, core.Operation{
			Path:        "/panic",
			OperationID: "panic",
		}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
			return nil, nil
		})
	})
}

func TestRegisterPanicNoPath(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))
	assert.Panics(t, func() {
		neoma.Register(api, core.Operation{
			Method:      http.MethodGet,
			OperationID: "panic",
		}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
			return nil, nil
		})
	})
}

func TestDeepObjectQueryParam(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Filter struct {
		Name string `json:"name"`
	}
	type Input struct {
		Filter Filter `query:"filter,deepObject"`
	}
	type Output struct {
		Body struct {
			Name string `json:"name"`
		}
	}

	neoma.Register(api, core.Operation{
		Method:                       http.MethodGet,
		Path:                         "/deep-object",
		OperationID:                  "deep-object",
		RejectUnknownQueryParameters: true,
	}, func(_ context.Context, in *Input) (*Output, error) {
		o := &Output{}
		o.Body.Name = in.Filter.Name
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/deep-object?filter[name]=hello")
	assert.Equal(t, http.StatusOK, resp.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, "hello", body["name"])
}

func TestOptionalPointerParam(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Input struct {
		Limit *int `query:"limit"`
	}
	type Output struct {
		Body struct {
			HasLimit bool `json:"has_limit"`
			Limit    int  `json:"limit"`
		}
	}

	neoma.Get[Input, Output](api, "/opt-ptr", func(_ context.Context, in *Input) (*Output, error) {
		o := &Output{}
		if in.Limit != nil {
			o.Body.HasLimit = true
			o.Body.Limit = *in.Limit
		}
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/opt-ptr")
	assert.Equal(t, http.StatusOK, resp.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, false, body["has_limit"])

	resp = api.Do(http.MethodGet, "/opt-ptr?limit=5")
	assert.Equal(t, http.StatusOK, resp.Code)
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, true, body["has_limit"])
	assert.InDelta(t, 5.0, body["limit"], 0.001)
}

func TestRequiredPointerParamMissing(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Input struct {
		Limit *int `query:"limit" required:"true"`
	}
	type Output struct {
		Body struct {
			Limit int `json:"limit"`
		}
	}

	neoma.Get[Input, Output](api, "/req-ptr-param", func(_ context.Context, in *Input) (*Output, error) {
		o := &Output{}
		if in.Limit != nil {
			o.Body.Limit = *in.Limit
		}
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/req-ptr-param")
	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
}

func TestInvalidParamValue(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Input struct {
		Count int `query:"count"`
	}
	type Output struct {
		Body struct {
			Count int `json:"count"`
		}
	}

	neoma.Get[Input, Output](api, "/bad-param", func(_ context.Context, in *Input) (*Output, error) {
		o := &Output{}
		o.Body.Count = in.Count
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/bad-param?count=notanumber")
	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
}

func TestOutputOnlyStatus(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Output struct {
		Status int
	}

	neoma.Register(api, core.Operation{
		Method:      http.MethodPost,
		Path:        "/status-only",
		OperationID: "status-only",
	}, func(_ context.Context, _ *struct{}) (*Output, error) {
		return &Output{Status: http.StatusAccepted}, nil
	})

	resp := api.Do(http.MethodPost, "/status-only")
	assert.Equal(t, http.StatusAccepted, resp.Code)
}

func TestCustomSchemaNamer(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.SchemaNamer = func(t_ reflect.Type, hint string) string {
		return "Custom" + hint
	}
	config.Components.Schemas = nil
	_, api := neomatest.New(t, config)

	assert.NotNil(t, api.OpenAPI())
}

func TestNoFormatFallback(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.NoFormatFallback = true
	config.DefaultFormat = ""
	_, api := neomatest.New(t, config)

	assert.NotNil(t, api)
}

func TestSkipValidateParams(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Input struct {
		Val string `query:"val" required:"true" minLength:"5"`
	}

	neoma.Register(api, core.Operation{
		Method:             http.MethodGet,
		Path:               "/skip-validate",
		OperationID:        "skip-validate",
		SkipValidateParams: true,
	}, func(_ context.Context, in *Input) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/skip-validate")
	assert.Equal(t, http.StatusNoContent, resp.Code)

	resp = api.Do(http.MethodGet, "/skip-validate?val=ab")
	assert.Equal(t, http.StatusNoContent, resp.Code)
}

func TestAutoAdd422And500(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Input struct {
		Body struct {
			Name string `json:"name"`
		}
	}

	neoma.Register(api, core.Operation{
		Method:      http.MethodPost,
		Path:        "/auto-errors",
		OperationID: "auto-422-500",
		Errors:      []int{400},
	}, func(_ context.Context, _ *Input) (*struct{}, error) {
		return nil, nil
	})

	oapi := api.OpenAPI()
	pi := oapi.Paths["/auto-errors"]
	require.NotNil(t, pi)
	require.NotNil(t, pi.Post)
	assert.Contains(t, pi.Post.Responses, "422")
	assert.Contains(t, pi.Post.Responses, "500")
	assert.Contains(t, pi.Post.Responses, "400")
}

func TestMaxBodyBytes(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Input struct {
		Body struct {
			Data string `json:"data"`
		}
	}

	neoma.Register(api, core.Operation{
		Method:       http.MethodPost,
		Path:         "/max-body",
		OperationID:  "max-body",
		MaxBodyBytes: 10,
	}, func(_ context.Context, _ *Input) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodPost, "/max-body", strings.NewReader(`{"data":"this is a very long string that exceeds the limit"}`), "Content-Type: application/json")
	assert.GreaterOrEqual(t, resp.Code, 400)
}

func TestInvalidPointerParamValue(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Input struct {
		Count *int `query:"count"`
	}
	type Output struct {
		Body struct {
			Count int `json:"count"`
		}
	}

	neoma.Get[Input, Output](api, "/bad-ptr-param", func(_ context.Context, in *Input) (*Output, error) {
		o := &Output{}
		if in.Count != nil {
			o.Body.Count = *in.Count
		}
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/bad-ptr-param?count=notanumber")
	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
}

func TestRequiredDeepObjectMissing(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Filter struct {
		Name string `json:"name"`
	}
	type Input struct {
		Filter Filter `query:"filter,deepObject" required:"true"`
	}

	neoma.Get[Input, struct{}](api, "/req-deep-obj", func(_ context.Context, _ *Input) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/req-deep-obj")
	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
}

func handleByFuncName(_ context.Context, _ *struct{}) (*struct{}, error) {
	return nil, nil
}

func TestNewAPIMinimalConfig(t *testing.T) {
	config := core.Config{
		OpenAPI: &core.OpenAPI{
			Info: &core.Info{Title: "Minimal", Version: "0.1.0"},
		},
		Formats:       neoma.DefaultConfig("", "").Formats,
		DefaultFormat: "application/json",
	}
	adapter := neomatest.NewAdapter()
	api := neoma.NewAPI(config, adapter)
	assert.NotNil(t, api.OpenAPI())
	assert.NotNil(t, api.OpenAPI().Components)
}

func TestNewAPIEmptyOpenAPIVersion(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.OpenAPIVersion = ""
	config.OpenAPI.OpenAPI = ""
	_, api := neomatest.New(t, config)
	assert.Equal(t, core.OpenAPIVersion32, api.OpenAPI().OpenAPI)
}

func TestNewAPIDefaultFormatAutoDetect(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.DefaultFormat = ""
	_, api := neomatest.New(t, config)
	assert.NotNil(t, api)
}

func TestNewAPIWithCustomErrorFactory(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.ErrorHandler = errors.NewNoopHandler()
	_, api := neomatest.New(t, config)
	assert.NotNil(t, api.ErrorHandler())
}

func TestNewAPIWithInternalSpecAndErrorDocs(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.InternalSpec = core.InternalSpecConfig{
		Path:    "/internal/openapi",
		Enabled: true,
	}
	config.ErrorHandler = errors.NewRFC9457HandlerWithConfig("/errors", nil)
	_, api := neomatest.New(t, config)
	assert.NotNil(t, api)
}

func TestReadCookiesMultiple(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Input struct {
		Session string `cookie:"session"`
		Theme   string `cookie:"theme"`
	}
	type Output struct {
		Body struct {
			Session string `json:"session"`
			Theme   string `json:"theme"`
		}
	}

	neoma.Get[Input, Output](api, "/multi-cookie", func(_ context.Context, in *Input) (*Output, error) {
		o := &Output{}
		o.Body.Session = in.Session
		o.Body.Theme = in.Theme
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/multi-cookie", "Cookie: session=abc; theme=dark")
	assert.Equal(t, http.StatusOK, resp.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, "abc", body["session"])
	assert.Equal(t, "dark", body["theme"])
}

func TestDefaultGenerateSummaryEdgeCases(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Output struct {
		Body struct {
			OK bool `json:"ok"`
		}
	}

	neoma.Get[struct{}, Output](api, "/users/{user_id}/posts/{post_id}", func(_ context.Context, _ *struct{}) (*Output, error) {
		return &Output{}, nil
	})

	oapi := api.OpenAPI()
	pi := oapi.Paths["/users/{user_id}/posts/{post_id}"]
	require.NotNil(t, pi)
	require.NotNil(t, pi.Get)
	assert.NotEmpty(t, pi.Get.Summary)
	assert.NotEmpty(t, pi.Get.OperationID)
}

func TestDiscoveredErrorsByFuncName(t *testing.T) {
	neoma.RegisterDiscoveredErrors(map[string][]core.DiscoveredError{
		"handleByFuncName": {
			{Status: 503, Title: "Service Unavailable", Detail: "Service down"},
		},
	})

	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	neoma.Register(api, core.Operation{
		Method: http.MethodGet,
		Path:   "/by-func-name",
	}, handleByFuncName)

	oapi := api.OpenAPI()
	pi := oapi.Paths["/by-func-name"]
	require.NotNil(t, pi)
	require.NotNil(t, pi.Get)
	assert.Contains(t, pi.Get.Responses, "503")
}

func TestConfigAccessorViaInterface(t *testing.T) {
	config := neoma.DefaultConfig("ConfigAccess", "3.0.0")
	config.FieldsOptionalByDefault = true
	adapter := neomatest.NewAdapter()
	api := neoma.NewAPI(config, adapter)

	type configProvider interface {
		Config() core.Config
	}
	cp, ok := api.(configProvider)
	require.True(t, ok, "api should implement configProvider interface")
	got := cp.Config()
	assert.True(t, got.FieldsOptionalByDefault)
	assert.Equal(t, "ConfigAccess", got.Info.Title)
}

func TestDocumentOperationHiddenDirectly(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	adapter := neomatest.NewAdapter()
	api := neoma.NewAPI(config, adapter)

	type documenter interface {
		DocumentOperation(op *core.Operation)
	}
	doc, ok := api.(documenter)
	require.True(t, ok, "api should implement OperationDocumenter")

	hiddenOp := &core.Operation{
		Method:      http.MethodGet,
		Path:        "/internal-hidden",
		OperationID: "internal-hidden",
		Hidden:      true,
	}
	doc.DocumentOperation(hiddenOp)

	assert.Nil(t, api.OpenAPI().Paths["/internal-hidden"])

	type hiddenProvider interface {
		HiddenOperations() []*core.Operation
	}
	hp, ok := api.(hiddenProvider)
	require.True(t, ok)
	found := false
	for _, op := range hp.HiddenOperations() {
		if op.OperationID == "internal-hidden" {
			found = true
			break
		}
	}
	assert.True(t, found, "hidden operation should be stored in hiddenOps")

	visibleOp := &core.Operation{
		Method:      http.MethodGet,
		Path:        "/visible-doc",
		OperationID: "visible-doc",
		Hidden:      false,
	}
	doc.DocumentOperation(visibleOp)
	assert.NotNil(t, api.OpenAPI().Paths["/visible-doc"])
}

func TestTransformReturnsError(t *testing.T) {
	transformErr := stderrors.New("transform failed")
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.Transformers = []core.Transformer{
		func(ctx core.Context, status string, v any) (any, error) {
			return nil, transformErr
		},
	}
	adapter := neomatest.NewAdapter()
	api := neoma.NewAPI(config, adapter)

	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	w := &fakeResponseWriter{}
	ctx := neomatest.NewContext(&core.Operation{}, req, w)

	result, err := api.Transform(ctx, "200", "some value")
	assert.Nil(t, result)
	assert.ErrorIs(t, err, transformErr)
}

type fakeResponseWriter struct {
	headers http.Header
	code    int
	body    []byte
}

func (f *fakeResponseWriter) Header() http.Header {
	if f.headers == nil {
		f.headers = http.Header{}
	}
	return f.headers
}

func (f *fakeResponseWriter) Write(b []byte) (int, error) {
	f.body = append(f.body, b...)
	return len(b), nil
}

func (f *fakeResponseWriter) WriteHeader(code int) {
	f.code = code
}

func TestDefaultGenerateSummaryEmptyPath(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Output struct {
		Body struct {
			OK bool `json:"ok"`
		}
	}

	neoma.Get[struct{}, Output](api, "/", func(_ context.Context, _ *struct{}) (*Output, error) {
		return &Output{}, nil
	})

	oapi := api.OpenAPI()
	pi := oapi.Paths["/"]
	require.NotNil(t, pi)
	require.NotNil(t, pi.Get)
	assert.NotEmpty(t, pi.Get.Summary)
}

func TestTrackHiddenOpReflectionPath(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	_, api := neomatest.New(t, config)

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/hidden-reflection",
		OperationID: "hidden-reflection",
		Hidden:      true,
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	assert.Nil(t, api.OpenAPI().Paths["/hidden-reflection"])

	resp := api.Do(http.MethodGet, "/hidden-reflection")
	assert.Equal(t, http.StatusNoContent, resp.Code)
}

func TestFieldsOptionalByDefaultViaConfigProvider(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.FieldsOptionalByDefault = true
	adapter := neomatest.NewAdapter()
	rawAPI := neoma.NewAPI(config, adapter)

	type Input struct {
		Body struct {
			Name string `json:"name"`
		}
	}

	neoma.Post[Input, struct{}](rawAPI, "/opt-fields", func(_ context.Context, _ *Input) (*struct{}, error) {
		return nil, nil
	})

	oapi := rawAPI.OpenAPI()
	pi := oapi.Paths["/opt-fields"]
	require.NotNil(t, pi)
	require.NotNil(t, pi.Post)
}

func TestTransformChainFirstErrors(t *testing.T) {
	errFirst := stderrors.New("first transformer error")
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.Transformers = []core.Transformer{
		func(ctx core.Context, status string, v any) (any, error) {
			return nil, errFirst
		},
		func(ctx core.Context, status string, v any) (any, error) {
			// This should never be called since the first one errors.
			return v, nil
		},
	}
	adapter := neomatest.NewAdapter()
	api := neoma.NewAPI(config, adapter)

	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	w := &fakeResponseWriter{}
	ctx := neomatest.NewContext(&core.Operation{}, req, w)

	result, err := api.Transform(ctx, "200", "value")
	assert.Nil(t, result)
	assert.ErrorIs(t, err, errFirst)
}

func TestTransformChainSuccessAll(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	config.Transformers = []core.Transformer{
		func(ctx core.Context, status string, v any) (any, error) {
			return "modified-by-first", nil
		},
		func(ctx core.Context, status string, v any) (any, error) {
			return v.(string) + "-and-second", nil
		},
	}
	adapter := neomatest.NewAdapter()
	api := neoma.NewAPI(config, adapter)

	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	w := &fakeResponseWriter{}
	ctx := neomatest.NewContext(&core.Operation{}, req, w)

	result, err := api.Transform(ctx, "200", "original")
	require.NoError(t, err)
	assert.Equal(t, "modified-by-first-and-second", result)
}

func TestRegisterHiddenOnRawAPI(t *testing.T) {
	config := neoma.DefaultConfig("Test", "1.0.0")
	adapter := neomatest.NewAdapter()
	rawAPI := neoma.NewAPI(config, adapter)

	neoma.Register(rawAPI, core.Operation{
		Method:      http.MethodGet,
		Path:        "/raw-hidden",
		OperationID: "raw-hidden",
		Hidden:      true,
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	assert.Nil(t, rawAPI.OpenAPI().Paths["/raw-hidden"])

	hp, ok := rawAPI.(core.HiddenOperationsProvider)
	require.True(t, ok)
	found := false
	for _, op := range hp.HiddenOperations() {
		if op.OperationID == "raw-hidden" {
			found = true
			break
		}
	}
	assert.True(t, found, "hidden op should be tracked via raw API")
}

func TestSkipDiscoveredErrors(t *testing.T) {
	neoma.RegisterDiscoveredErrors(map[string][]core.DiscoveredError{
		"skipMe": {
			{Status: 503, Title: "Service Unavailable", Detail: "skipped"},
		},
	})

	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	neoma.Register(api, core.Operation{
		Method:               http.MethodGet,
		Path:                 "/skip-discovered",
		OperationID:          "skipMe",
		SkipDiscoveredErrors: true,
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	oapi := api.OpenAPI()
	pi := oapi.Paths["/skip-discovered"]
	require.NotNil(t, pi)
	require.NotNil(t, pi.Get)
	assert.NotContains(t, pi.Get.Responses, "503")
}

func TestNewAPINilOpenAPI(t *testing.T) {
	config := core.Config{
		Formats:       neoma.DefaultConfig("", "").Formats,
		DefaultFormat: "application/json",
	}
	adapter := neomatest.NewAdapter()
	api := neoma.NewAPI(config, adapter)
	assert.NotNil(t, api.OpenAPI(), "OpenAPI should be auto-created when nil")
}

func TestBodyContentTypeInferenceNoHeader(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Input struct {
		Body struct {
			Msg string `json:"msg"`
		}
	}
	type Output struct {
		Body struct {
			Msg string `json:"msg"`
		}
	}

	neoma.Post[Input, Output](api, "/no-ct-header", func(_ context.Context, in *Input) (*Output, error) {
		o := &Output{}
		o.Body.Msg = in.Body.Msg
		return o, nil
	})

	resp := api.Do(http.MethodPost, "/no-ct-header", strings.NewReader(`{"msg":"hello"}`))
	assert.LessOrEqual(t, resp.Code, 299)
}

func TestOptionalPointerCookieParam(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Input struct {
		Session *string `cookie:"session"`
	}
	type Output struct {
		Body struct {
			HasSession bool   `json:"has_session"`
			Session    string `json:"session"`
		}
	}

	neoma.Get[Input, Output](api, "/opt-cookie", func(_ context.Context, in *Input) (*Output, error) {
		o := &Output{}
		if in.Session != nil {
			o.Body.HasSession = true
			o.Body.Session = *in.Session
		}
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/opt-cookie")
	assert.Equal(t, http.StatusOK, resp.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, false, body["has_session"])

	resp = api.Do(http.MethodGet, "/opt-cookie", "Cookie: session=abc")
	assert.Equal(t, http.StatusOK, resp.Code)
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, true, body["has_session"])
	assert.Equal(t, "abc", body["session"])
}

type resolverWithPathInput struct {
	Name string `query:"name"`
}

func (r *resolverWithPathInput) Resolve(_ core.Context, _ *core.PathBuffer) []error {
	if r.Name == "" {
		return []error{stderrors.New("name is required")}
	}
	return nil
}

func TestResolverWithPath(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Output struct {
		Body struct {
			Name string `json:"name"`
		}
	}

	neoma.Get[resolverWithPathInput, Output](api, "/resolver-with-path",
		func(_ context.Context, in *resolverWithPathInput) (*Output, error) {
			o := &Output{}
			o.Body.Name = in.Name
			return o, nil
		},
	)

	resp := api.Do(http.MethodGet, "/resolver-with-path")
	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)

	resp = api.Do(http.MethodGet, "/resolver-with-path?name=hello")
	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestRequestBodyExampleAutoBuiltFromTags(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type ExampleInput struct {
		Body struct {
			Name string `json:"name" example:"Alice"`
			Age  int    `json:"age" example:"30"`
		}
	}

	type Output struct {
		Body struct {
			OK bool `json:"ok"`
		}
	}

	neoma.Post[ExampleInput, Output](api, "/example-body",
		func(_ context.Context, _ *ExampleInput) (*Output, error) {
			o := &Output{}
			o.Body.OK = true
			return o, nil
		},
	)

	resp := api.Do(http.MethodGet, "/openapi.json")
	require.Equal(t, http.StatusOK, resp.Code)

	var spec map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &spec))

	paths, ok := spec["paths"].(map[string]any)
	require.True(t, ok, "spec should have paths")

	pathItem, ok := paths["/example-body"].(map[string]any)
	require.True(t, ok, "spec should have /example-body path")

	post, ok := pathItem["post"].(map[string]any)
	require.True(t, ok, "path item should have post operation")

	reqBody, ok := post["requestBody"].(map[string]any)
	require.True(t, ok, "post should have requestBody")

	content, ok := reqBody["content"].(map[string]any)
	require.True(t, ok, "requestBody should have content")

	jsonContent, ok := content["application/json"].(map[string]any)
	require.True(t, ok, "content should have application/json")

	example, ok := jsonContent["example"]
	require.True(t, ok, "application/json media type should have an example field")
	require.NotNil(t, example, "example should not be nil")

	exMap, ok := example.(map[string]any)
	require.True(t, ok, "example should be a JSON object")
	assert.Equal(t, "Alice", exMap["name"])
	assert.InDelta(t, 30, exMap["age"], 0.01)
}
