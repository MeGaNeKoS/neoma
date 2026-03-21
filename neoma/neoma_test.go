package neoma_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/errors"
	"github.com/MeGaNeKoS/neoma/neoma"
	"github.com/MeGaNeKoS/neoma/neomatest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)


func newTestAPI(t *testing.T, configs ...core.Config) neomatest.TestAPI {
	t.Helper()
	_, api := neomatest.New(t, configs...)
	return api
}


func TestBasicGet(t *testing.T) {
	api := newTestAPI(t)

	type Output struct {
		Body struct {
			Message string `json:"message"`
		}
	}

	neoma.Get[struct{}, Output](api, "/hello", func(_ context.Context, _ *struct{}) (*Output, error) {
		o := &Output{}
		o.Body.Message = "hello world"
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/hello")
	assert.Equal(t, http.StatusOK, resp.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, "hello world", body["message"])
}


func TestPostWithBody(t *testing.T) {
	api := newTestAPI(t)

	type Input struct {
		Body struct {
			Name string `json:"name" required:"true"`
		}
	}
	type Output struct {
		Body struct {
			Greeting string `json:"greeting"`
		}
	}

	neoma.Post[Input, Output](api, "/greet", func(_ context.Context, in *Input) (*Output, error) {
		o := &Output{}
		o.Body.Greeting = "Hello, " + in.Body.Name + "!"
		return o, nil
	})

	resp := api.Do(http.MethodPost, "/greet", map[string]any{"name": "Alice"})
	assert.Equal(t, http.StatusOK, resp.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, "Hello, Alice!", body["greeting"])
}


func TestPathParameters(t *testing.T) {
	api := newTestAPI(t)

	type Input struct {
		ID string `path:"id"`
	}
	type Output struct {
		Body struct {
			ID string `json:"id"`
		}
	}

	neoma.Get[Input, Output](api, "/items/{id}", func(_ context.Context, in *Input) (*Output, error) {
		o := &Output{}
		o.Body.ID = in.ID
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/items/abc123")
	assert.Equal(t, http.StatusOK, resp.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, "abc123", body["id"])
}


func TestQueryParameters(t *testing.T) {
	api := newTestAPI(t)

	type Input struct {
		Search string `query:"search"`
		Limit  int    `query:"limit"`
	}
	type Output struct {
		Body struct {
			Search string `json:"search"`
			Limit  int    `json:"limit"`
		}
	}

	neoma.Get[Input, Output](api, "/search", func(_ context.Context, in *Input) (*Output, error) {
		o := &Output{}
		o.Body.Search = in.Search
		o.Body.Limit = in.Limit
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/search?search=foo&limit=10")
	assert.Equal(t, http.StatusOK, resp.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, "foo", body["search"])
	assert.InDelta(t, 10.0, body["limit"], 0.001)
}


func TestHeaderParameters(t *testing.T) {
	api := newTestAPI(t)

	type Input struct {
		Token string `header:"X-Token"`
	}
	type Output struct {
		Body struct {
			Token string `json:"token"`
		}
	}

	neoma.Get[Input, Output](api, "/auth", func(_ context.Context, in *Input) (*Output, error) {
		o := &Output{}
		o.Body.Token = in.Token
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/auth", "X-Token: secret123")
	assert.Equal(t, http.StatusOK, resp.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, "secret123", body["token"])
}


func TestValidation422(t *testing.T) {
	api := newTestAPI(t)

	type Input struct {
		Body struct {
			Name string `json:"name" minLength:"3"`
		}
	}
	type Output struct {
		Body struct {
			Name string `json:"name"`
		}
	}

	neoma.Post[Input, Output](api, "/validate", func(_ context.Context, in *Input) (*Output, error) {
		o := &Output{}
		o.Body.Name = in.Body.Name
		return o, nil
	})

	resp := api.Do(http.MethodPost, "/validate", map[string]any{"name": "ab"})
	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.InDelta(t, 422.0, body["status"], 0.001)
}


func TestNoopHandlerNoErrorSchemas(t *testing.T) {
	config := neoma.DefaultConfig("Test API", "1.0.0")
	config.ErrorHandler = errors.NewNoopHandler()
	api := newTestAPI(t, config)

	type Input struct {
		Body struct {
			Name string `json:"name"`
		}
	}

	neoma.Post[Input, struct{}](api, "/noop-err", func(_ context.Context, _ *Input) (*struct{}, error) {
		return nil, nil
	})

	oapi := api.OpenAPI()
	if oapi.Paths != nil {
		pi := oapi.Paths["/noop-err"]
		if pi != nil && pi.Post != nil {
			for code, resp := range pi.Post.Responses {
				if code != "204" { // only the success response should exist
					if resp.Content != nil {
						for _, mt := range resp.Content {
							assert.Nil(t, mt.Schema, "NoopHandler should produce no error schemas, but found one at status %s", code)
						}
					}
				}
			}
		}
	}
}


func TestErrorResponseHeaders(t *testing.T) {
	config := neoma.DefaultConfig("Test API", "1.0.0")
	api := newTestAPI(t, config)

	neoma.Register[struct{}, struct{}](api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/with-err-headers",
		OperationID: "with-err-headers",
		Errors:      []int{http.StatusTooManyRequests, http.StatusInternalServerError},
		ErrorHeaders: map[string]*core.Param{
			"Retry-After": {
				Schema: &core.Schema{Type: "integer"},
			},
		},
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	oapi := api.OpenAPI()
	pi := oapi.Paths["/with-err-headers"]
	require.NotNil(t, pi)
	assert.NotNil(t, pi.Get)

	for code, resp := range pi.Get.Responses {
		if code == "204" {
			continue
		}
		assert.NotNil(t, resp.Headers, "error response %s should have headers", code)
		assert.NotNil(t, resp.Headers["Retry-After"], "error response %s should have Retry-After header", code)
	}
}


func TestDefaultValues(t *testing.T) {
	api := newTestAPI(t)

	type Input struct {
		Body struct {
			Color string `json:"color,omitempty" default:"blue"`
		}
	}
	type Output struct {
		Body struct {
			Color string `json:"color"`
		}
	}

	neoma.Post[Input, Output](api, "/defaults", func(_ context.Context, in *Input) (*Output, error) {
		o := &Output{}
		o.Body.Color = in.Body.Color
		return o, nil
	})

	resp := api.Do(http.MethodPost, "/defaults", strings.NewReader(`{}`), "Content-Type: application/json")
	assert.Equal(t, http.StatusOK, resp.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, "blue", body["color"])
}


func TestStreamResponse(t *testing.T) {
	api := newTestAPI(t)

	type Output struct {
		Body func(ctx core.Context, api core.API)
	}

	neoma.Get[struct{}, Output](api, "/stream", func(_ context.Context, _ *struct{}) (*Output, error) {
		return &Output{
			Body: func(ctx core.Context, _ core.API) {
				ctx.SetHeader("Content-Type", "text/plain")
				ctx.SetStatus(http.StatusOK)
				_, _ = ctx.BodyWriter().Write([]byte("streamed data"))
			},
		}, nil
	})

	resp := api.Do(http.MethodGet, "/stream")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "streamed data")
}


func TestMiddlewareExecutionOrder(t *testing.T) {
	api := newTestAPI(t)

	var order []string

	api.UseMiddleware(func(ctx core.Context, next func(core.Context)) {
		order = append(order, "mw1-before")
		next(ctx)
		order = append(order, "mw1-after")
	})

	api.UseMiddleware(func(ctx core.Context, next func(core.Context)) {
		order = append(order, "mw2-before")
		next(ctx)
		order = append(order, "mw2-after")
	})

	type Output struct {
		Body struct {
			OK bool `json:"ok"`
		}
	}

	neoma.Get[struct{}, Output](api, "/mw-order", func(_ context.Context, _ *struct{}) (*Output, error) {
		order = append(order, "handler")
		o := &Output{}
		o.Body.OK = true
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/mw-order")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, []string{
		"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after",
	}, order)
}


func TestResponseHeaderControl(t *testing.T) {
	api := newTestAPI(t)

	// Middleware sets a header before the handler, then uses
	// GetResponseHeader / DeleteResponseHeader to verify control works.
	var capturedHeader string
	var headerDeleted bool

	api.UseMiddleware(func(ctx core.Context, next func(core.Context)) {
		ctx.SetHeader("X-Test-Before", "original")

		capturedHeader = ctx.GetResponseHeader("X-Test-Before")

		ctx.DeleteResponseHeader("X-Test-Before")
		headerDeleted = ctx.GetResponseHeader("X-Test-Before") == ""
		ctx.SetHeader("X-Test-Before", "replaced")

		next(ctx)
	})

	type Output struct {
		Body struct {
			OK bool `json:"ok"`
		}
	}

	neoma.Get[struct{}, Output](api, "/header-ctrl", func(_ context.Context, _ *struct{}) (*Output, error) {
		return &Output{}, nil
	})

	resp := api.Do(http.MethodGet, "/header-ctrl")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "original", capturedHeader, "GetResponseHeader should return the previously set header")
	assert.True(t, headerDeleted, "DeleteResponseHeader should remove the header")
	assert.Equal(t, "replaced", resp.Header().Get("X-Test-Before"), "final header should be the replaced value")
}


func TestContentNegotiation(t *testing.T) {
	api := newTestAPI(t)

	type Output struct {
		Body struct {
			Data string `json:"data"`
		}
	}

	neoma.Get[struct{}, Output](api, "/negotiate", func(_ context.Context, _ *struct{}) (*Output, error) {
		o := &Output{}
		o.Body.Data = "content"
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/negotiate", "Accept: application/json")
	assert.Equal(t, http.StatusOK, resp.Code)
	ct := resp.Header().Get("Content-Type")
	assert.Contains(t, ct, "json")

	var body map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, "content", body["data"])
}


func TestNoResponseBody204(t *testing.T) {
	api := newTestAPI(t)

	neoma.Delete[struct{}, struct{}](api, "/empty", func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodDelete, "/empty")
	assert.Equal(t, http.StatusNoContent, resp.Code)
}

func TestHandlerReturnsStatusError(t *testing.T) {
	api := newTestAPI(t)

	type Output struct {
		Body struct {
			Data string `json:"data"`
		}
	}

	neoma.Get[struct{}, Output](api, "/not-found", func(_ context.Context, _ *struct{}) (*Output, error) {
		return nil, errors.ErrorNotFound("resource not found")
	})

	resp := api.Do(http.MethodGet, "/not-found")
	assert.Equal(t, http.StatusNotFound, resp.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.InDelta(t, 404.0, body["status"], 0.001)
}

func TestHandlerReturnsGenericError(t *testing.T) {
	api := newTestAPI(t)

	type Output struct {
		Body struct {
			Data string `json:"data"`
		}
	}

	neoma.Get[struct{}, Output](api, "/internal", func(_ context.Context, _ *struct{}) (*Output, error) {
		return nil, assert.AnError
	})

	resp := api.Do(http.MethodGet, "/internal")
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestPutAndPatch(t *testing.T) {
	api := newTestAPI(t)

	type Input struct {
		Body struct {
			Value string `json:"value"`
		}
	}
	type Output struct {
		Body struct {
			Value string `json:"value"`
		}
	}

	neoma.Put[Input, Output](api, "/put-item", func(_ context.Context, in *Input) (*Output, error) {
		o := &Output{}
		o.Body.Value = in.Body.Value
		return o, nil
	})
	neoma.Patch[Input, Output](api, "/patch-item", func(_ context.Context, in *Input) (*Output, error) {
		o := &Output{}
		o.Body.Value = in.Body.Value
		return o, nil
	})

	resp := api.Do(http.MethodPut, "/put-item", map[string]any{"value": "updated"})
	assert.Equal(t, http.StatusOK, resp.Code)

	resp = api.Do(http.MethodPatch, "/patch-item", map[string]any{"value": "patched"})
	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestOperationTagsHelper(t *testing.T) {
	api := newTestAPI(t)

	type Output struct {
		Body struct {
			OK bool `json:"ok"`
		}
	}

	neoma.Get[struct{}, Output](api, "/tagged", func(_ context.Context, _ *struct{}) (*Output, error) {
		return &Output{}, nil
	}, neoma.OperationTags("mytag"))

	oapi := api.OpenAPI()
	pi := oapi.Paths["/tagged"]
	require.NotNil(t, pi)
	assert.NotNil(t, pi.Get)
	assert.Contains(t, pi.Get.Tags, "mytag")
}

func TestMultipartFileUpload(t *testing.T) {
	api := newTestAPI(t)

	type UploadInput struct {
		Body struct {
			File core.FormFile `form:"file"`
			Name string        `form:"name"`
		}
	}
	type UploadOutput struct {
		Body struct {
			Filename string `json:"filename"`
			Size     int64  `json:"size"`
			Name     string `json:"name"`
			Content  string `json:"content"`
		}
	}

	neoma.Post[UploadInput, UploadOutput](api, "/upload", func(_ context.Context, in *UploadInput) (*UploadOutput, error) {
		o := &UploadOutput{}
		o.Body.Filename = in.Body.File.Filename
		o.Body.Size = in.Body.File.Size
		o.Body.Name = in.Body.Name
		data, err := io.ReadAll(in.Body.File)
		if err != nil {
			return nil, err
		}
		o.Body.Content = string(data)
		return o, nil
	})

	// Build multipart request
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "test.txt")
	require.NoError(t, err)
	_, err = part.Write([]byte("hello multipart"))
	require.NoError(t, err)
	require.NoError(t, writer.WriteField("name", "my-upload"))
	require.NoError(t, writer.Close())

	resp := api.Do(http.MethodPost, "/upload",
		"Content-Type: "+writer.FormDataContentType(),
		bytes.NewReader(buf.Bytes()),
	)

	assert.Equal(t, http.StatusOK, resp.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, "test.txt", body["filename"])
	assert.Equal(t, "my-upload", body["name"])
	assert.Equal(t, "hello multipart", body["content"])
}

func TestMultipartMultipleFiles(t *testing.T) {
	api := newTestAPI(t)

	type UploadInput struct {
		Body struct {
			Photos []core.FormFile `form:"photos"`
		}
	}
	type UploadOutput struct {
		Body struct {
			Count     int      `json:"count"`
			Filenames []string `json:"filenames"`
		}
	}

	neoma.Post[UploadInput, UploadOutput](api, "/photos", func(_ context.Context, in *UploadInput) (*UploadOutput, error) {
		o := &UploadOutput{}
		o.Body.Count = len(in.Body.Photos)
		for _, f := range in.Body.Photos {
			o.Body.Filenames = append(o.Body.Filenames, f.Filename)
		}
		return o, nil
	})

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	for _, name := range []string{"a.jpg", "b.png"} {
		part, err := writer.CreateFormFile("photos", name)
		require.NoError(t, err)
		_, err = part.Write([]byte("data-" + name))
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())

	resp := api.Do(http.MethodPost, "/photos",
		"Content-Type: "+writer.FormDataContentType(),
		bytes.NewReader(buf.Bytes()),
	)

	assert.Equal(t, http.StatusOK, resp.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.InDelta(t, 2, body["count"], 0)
	filenames := body["filenames"].([]any)
	assert.Equal(t, "a.jpg", filenames[0])
	assert.Equal(t, "b.png", filenames[1])
}

func TestMultipartWithDefaults(t *testing.T) {
	api := newTestAPI(t)

	type UploadInput struct {
		Body struct {
			File core.FormFile `form:"file"`
			Tag  string        `form:"tag" default:"untagged"`
		}
	}
	type UploadOutput struct {
		Body struct {
			Filename string `json:"filename"`
			Tag      string `json:"tag"`
		}
	}

	neoma.Post[UploadInput, UploadOutput](api, "/upload-default", func(_ context.Context, in *UploadInput) (*UploadOutput, error) {
		o := &UploadOutput{}
		o.Body.Filename = in.Body.File.Filename
		o.Body.Tag = in.Body.Tag
		return o, nil
	})

	// Send multipart request without the "tag" field to test default
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "doc.pdf")
	require.NoError(t, err)
	_, err = part.Write([]byte("pdf content"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	resp := api.Do(http.MethodPost, "/upload-default",
		"Content-Type: "+writer.FormDataContentType(),
		bytes.NewReader(buf.Bytes()),
	)

	assert.Equal(t, http.StatusOK, resp.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, "doc.pdf", body["filename"])
	assert.Equal(t, "untagged", body["tag"])
}
