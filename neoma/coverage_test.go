package neoma_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/errors"
	"github.com/MeGaNeKoS/neoma/neoma"
	"github.com/MeGaNeKoS/neoma/neomatest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)


func TestConvenienceGet(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Output struct {
		Body struct {
			Msg string `json:"msg"`
		}
	}

	neoma.Get[struct{}, Output](api, "/hello", func(_ context.Context, _ *struct{}) (*Output, error) {
		o := &Output{}
		o.Body.Msg = "hi"
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/hello")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "hi")
}

func TestConveniencePost(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Input struct {
		Body struct {
			Name string `json:"name"`
		}
	}

	neoma.Post[Input, struct{}](api, "/items", func(_ context.Context, in *Input) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodPost, "/items", map[string]string{"name": "test"})
	assert.Equal(t, http.StatusNoContent, resp.Code)
}

func TestConveniencePut(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	neoma.Put[struct{}, struct{}](api, "/items/{id}", func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodPut, "/items/123")
	assert.Equal(t, http.StatusNoContent, resp.Code)
}

func TestConveniencePatch(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	neoma.Patch[struct{}, struct{}](api, "/items/{id}", func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodPatch, "/items/123")
	assert.Equal(t, http.StatusNoContent, resp.Code)
}

func TestConvenienceDelete(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	neoma.Delete[struct{}, struct{}](api, "/items/{id}", func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodDelete, "/items/123")
	assert.Equal(t, http.StatusNoContent, resp.Code)
}

func TestConvenienceHead(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	neoma.Head[struct{}, struct{}](api, "/items", func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodHead, "/items")
	assert.Less(t, resp.Code, 300)
}

func TestOperationTags(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	neoma.Get[struct{}, struct{}](api, "/tagged", func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	}, neoma.OperationTags("users", "admin"))

	oapi := api.OpenAPI()
	pi := oapi.Paths["/tagged"]
	require.NotNil(t, pi)
	require.NotNil(t, pi.Get)
	assert.Equal(t, []string{"users", "admin"}, pi.Get.Tags)
}

func TestConvenienceWithOperationHandler(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	neoma.Get[struct{}, struct{}](api, "/custom", func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	}, func(o *core.Operation) {
		o.OperationID = "custom-id"
		o.Summary = "Custom summary"
	})

	oapi := api.OpenAPI()
	pi := oapi.Paths["/custom"]
	require.NotNil(t, pi)
	assert.Equal(t, "custom-id", pi.Get.OperationID)
	assert.Equal(t, "Custom summary", pi.Get.Summary)
}


func TestRegisterReturnsError(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Input struct {
		Body struct {
			Name string `json:"name"`
		}
	}

	neoma.Register(api, core.Operation{
		Method:      http.MethodPost,
		Path:        "/test",
		OperationID: "test-error",
	}, func(_ context.Context, in *Input) (*struct{}, error) {
		return nil, errors.ErrorUnprocessableEntity("validation failed")
	})

	resp := api.Do(http.MethodPost, "/test", map[string]string{"name": "test"})
	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
}

func TestRegisterStatusError(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/notfound",
		OperationID: "test-notfound",
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, errors.ErrorNotFound("item not found")
	})

	resp := api.Do(http.MethodGet, "/notfound")
	assert.Equal(t, http.StatusNotFound, resp.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Contains(t, body["detail"], "item not found")
}


func TestStreamResponseCoverage(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Output struct {
		Body func(ctx core.Context, api core.API)
	}

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/stream",
		OperationID: "stream",
	}, func(_ context.Context, _ *struct{}) (*Output, error) {
		return &Output{
			Body: func(ctx core.Context, _ core.API) {
				ctx.SetHeader("Content-Type", "text/plain")
				_, _ = ctx.BodyWriter().Write([]byte("streamed!"))
			},
		}, nil
	})

	resp := api.Do(http.MethodGet, "/stream")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "streamed!")
}


func TestResponseHeaders(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Output struct {
		Status int
		ETag   string `header:"ETag"`
		Body   struct {
			Msg string `json:"msg"`
		}
	}

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/with-headers",
		OperationID: "with-headers",
	}, func(_ context.Context, _ *struct{}) (*Output, error) {
		o := &Output{Status: 200, ETag: "abc123"}
		o.Body.Msg = "ok"
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/with-headers")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "abc123", resp.Header().Get("ETag"))
}


func TestWriteErr(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/write-err",
		OperationID: "write-err",
	}, func(ctx context.Context, _ *struct{}) (*struct{}, error) {
		return nil, errors.ErrorBadRequest("bad input")
	})

	resp := api.Do(http.MethodGet, "/write-err")
	assert.Equal(t, http.StatusBadRequest, resp.Code)
}


func TestRegisterWithDiscoveredErrors(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/discovered",
		OperationID: "discovered",
		Errors:      []int{400, 404},
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	oapi := api.OpenAPI()
	pi := oapi.Paths["/discovered"]
	require.NotNil(t, pi)
	require.NotNil(t, pi.Get)
	assert.Contains(t, pi.Get.Responses, "400")
	assert.Contains(t, pi.Get.Responses, "404")
}
