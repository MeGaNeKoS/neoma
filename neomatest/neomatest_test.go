package neomatest_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/neoma"
	"github.com/MeGaNeKoS/neoma/neomatest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCreatesAPI(t *testing.T) {
	handler, api := neomatest.New(t)
	require.NotNil(t, handler)
	require.NotNil(t, api)
}

func TestNewWithCustomConfig(t *testing.T) {
	config := neoma.DefaultConfig("Custom API", "2.0.0")
	_, api := neomatest.New(t, config)
	require.NotNil(t, api)
	assert.Equal(t, "Custom API", api.OpenAPI().Info.Title)
}

func TestNewPanicsOnNilOpenAPI(t *testing.T) {
	assert.Panics(t, func() {
		neomatest.New(t, core.Config{})
	})
}

func TestGetHelper(t *testing.T) {
	_, api := neomatest.New(t)

	type Output struct {
		Body struct {
			Msg string `json:"msg"`
		}
	}

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/hello",
		OperationID: "get-hello",
	}, func(_ context.Context, _ *struct{}) (*Output, error) {
		o := &Output{}
		o.Body.Msg = "hi"
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/hello")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), `"msg"`)
	assert.Contains(t, resp.Body.String(), `"hi"`)
}

func TestGetWithHeaders(t *testing.T) {
	_, api := neomatest.New(t)

	type Input struct {
		Auth string `header:"Authorization"`
	}
	type Output struct {
		Body struct {
			Token string `json:"token"`
		}
	}

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/auth",
		OperationID: "get-auth",
	}, func(_ context.Context, in *Input) (*Output, error) {
		o := &Output{}
		o.Body.Token = in.Auth
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/auth", "Authorization: Bearer xyz")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "Bearer xyz")
}

func TestPostHelper(t *testing.T) {
	_, api := neomatest.New(t)

	type Input struct {
		Body struct {
			Name string `json:"name"`
		}
	}
	type Output struct {
		Body struct {
			Echo string `json:"echo"`
		}
	}

	neoma.Register(api, core.Operation{
		Method:      http.MethodPost,
		Path:        "/echo",
		OperationID: "post-echo",
	}, func(_ context.Context, in *Input) (*Output, error) {
		o := &Output{}
		o.Body.Echo = in.Body.Name
		return o, nil
	})

	resp := api.Do(http.MethodPost, "/echo", map[string]string{"name": "world"})
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "world")
}

func TestPostWithReader(t *testing.T) {
	_, api := neomatest.New(t)

	type Input struct {
		Body struct {
			Val int `json:"val"`
		}
	}

	var got int
	neoma.Register(api, core.Operation{
		Method:      http.MethodPost,
		Path:        "/val",
		OperationID: "post-val",
	}, func(_ context.Context, in *Input) (*struct{}, error) {
		got = in.Body.Val
		return nil, nil
	})

	resp := api.Do(http.MethodPost, "/val", "Content-Type: application/json", strings.NewReader(`{"val":42}`))
	assert.Equal(t, http.StatusNoContent, resp.Code)
	assert.Equal(t, 42, got)
}

func TestPutHelper(t *testing.T) {
	_, api := neomatest.New(t)

	type Input struct {
		ID   string `path:"id"`
		Body struct {
			Name string `json:"name"`
		}
	}

	var gotID, gotName string
	neoma.Register(api, core.Operation{
		Method:      http.MethodPut,
		Path:        "/items/{id}",
		OperationID: "put-item",
	}, func(_ context.Context, in *Input) (*struct{}, error) {
		gotID = in.ID
		gotName = in.Body.Name
		return nil, nil
	})

	resp := api.Do(http.MethodPut, "/items/abc", map[string]string{"name": "updated"})
	assert.Equal(t, http.StatusNoContent, resp.Code)
	assert.Equal(t, "abc", gotID)
	assert.Equal(t, "updated", gotName)
}

func TestDeleteHelper(t *testing.T) {
	_, api := neomatest.New(t)

	deleted := false
	neoma.Register(api, core.Operation{
		Method:      http.MethodDelete,
		Path:        "/items/{id}",
		OperationID: "delete-item",
	}, func(_ context.Context, _ *struct {
		ID string `path:"id"`
	}) (*struct{}, error) {
		deleted = true
		return nil, nil
	})

	resp := api.Do(http.MethodDelete, "/items/xyz")
	assert.Equal(t, http.StatusNoContent, resp.Code)
	assert.True(t, deleted)
}

func TestPatchHelper(t *testing.T) {
	_, api := neomatest.New(t)

	type Input struct {
		Body struct {
			Name string `json:"name"`
		}
	}

	var got string
	neoma.Register(api, core.Operation{
		Method:      http.MethodPatch,
		Path:        "/items/{id}",
		OperationID: "patch-item",
	}, func(_ context.Context, in *Input) (*struct{}, error) {
		got = in.Body.Name
		return nil, nil
	})

	resp := api.Do(http.MethodPatch, "/items/1", map[string]string{"name": "patched"})
	assert.Equal(t, http.StatusNoContent, resp.Code)
	assert.Equal(t, "patched", got)
}

func TestResponseNotFound(t *testing.T) {
	_, api := neomatest.New(t)

	resp := api.Do(http.MethodGet, "/nonexistent")
	assert.Equal(t, http.StatusNotFound, resp.Code)
}

func TestResponseHeaders(t *testing.T) {
	_, api := neomatest.New(t)

	type Output struct {
		Body struct {
			V string `json:"v"`
		}
	}

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/ct",
		OperationID: "get-ct",
	}, func(_ context.Context, _ *struct{}) (*Output, error) {
		o := &Output{}
		o.Body.V = "ok"
		return o, nil
	})

	resp := api.Do(http.MethodGet, "/ct")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Header().Get("Content-Type"), "json")
}

func TestGetCtx(t *testing.T) {
	_, api := neomatest.New(t)

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/ctx",
		OperationID: "get-ctx",
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	ctx := context.Background()
	resp := api.DoCtx(ctx, http.MethodGet, "/ctx")
	assert.Equal(t, http.StatusNoContent, resp.Code)
}

func TestDoMethod(t *testing.T) {
	_, api := neomatest.New(t)

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/do-test",
		OperationID: "do-test",
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/do-test")
	assert.Equal(t, http.StatusNoContent, resp.Code)
}

func TestWrap(t *testing.T) {
	_, baseAPI := neomatest.New(t)

	neoma.Register(baseAPI, core.Operation{
		Method:      http.MethodGet,
		Path:        "/wrap",
		OperationID: "wrap-test",
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	wrapped := neomatest.Wrap(t, baseAPI)
	resp := wrapped.Do(http.MethodGet, "/wrap")
	assert.Equal(t, http.StatusNoContent, resp.Code)
}

func TestDumpRequest(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/test", strings.NewReader(`{"msg":"hi"}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	b, err := neomatest.DumpRequest(req)
	require.NoError(t, err)
	assert.NotEmpty(t, b)
	assert.Contains(t, string(b), "GET /test")
}

func TestDumpRequestNilBody(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/no-body", nil)
	require.NoError(t, err)

	b, err := neomatest.DumpRequest(req)
	require.NoError(t, err)
	assert.NotEmpty(t, b)
}

func TestDumpResponse(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
	}

	b, err := neomatest.DumpResponse(resp)
	require.NoError(t, err)
	assert.NotEmpty(t, b)
	assert.Contains(t, string(b), "200")
}

func TestPrintRequest(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/print", strings.NewReader(`{"v":1}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	assert.NotPanics(t, func() {
		neomatest.PrintRequest(req)
	})
}

func TestPrintResponse(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"text/plain"}},
		Body:       io.NopCloser(strings.NewReader("hello")),
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
	}

	assert.NotPanics(t, func() {
		neomatest.PrintResponse(resp)
	})
}

func TestNewAdapter(t *testing.T) {
	adapter := neomatest.NewAdapter()
	require.NotNil(t, adapter)
}

func TestNewContext(t *testing.T) {
	op := &core.Operation{Method: http.MethodGet, Path: "/test"}
	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	ctx := neomatest.NewContext(op, req, w)
	require.NotNil(t, ctx)
	assert.Equal(t, http.MethodGet, ctx.Method())
	assert.Equal(t, op, ctx.Operation())
}

func TestDoCtxUnsupportedArgTypePanics(t *testing.T) {
	_, api := neomatest.New(t)

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/panic-arg",
		OperationID: "panic-arg",
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	assert.Panics(t, func() {
		api.DoCtx(context.Background(), http.MethodGet, "/panic-arg", 42)
	})
}

func TestDoCtxUnsupportedArgTypeBoolPanics(t *testing.T) {
	_, api := neomatest.New(t)

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/panic-bool",
		OperationID: "panic-bool",
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	assert.Panics(t, func() {
		api.DoCtx(context.Background(), http.MethodGet, "/panic-bool", true)
	})
}

func TestDumpRequestNonJSON(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/upload", strings.NewReader("plain text body"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "text/plain")

	b, err := neomatest.DumpRequest(req)
	require.NoError(t, err)
	assert.NotEmpty(t, b)
	assert.Contains(t, string(b), "plain text body")
}

func TestDumpResponseNonJSON(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"text/plain"}},
		Body:       io.NopCloser(strings.NewReader("hello plain")),
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
	}

	b, err := neomatest.DumpResponse(resp)
	require.NoError(t, err)
	assert.NotEmpty(t, b)
	assert.Contains(t, string(b), "hello plain")
}

func TestDumpResponseNilBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusNoContent,
		Header:     http.Header{},
		Body:       nil,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
	}

	b, err := neomatest.DumpResponse(resp)
	require.NoError(t, err)
	assert.NotEmpty(t, b)
	assert.Contains(t, string(b), "204")
}

func TestDumpRequestNilBodyExplicit(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/nil-body", nil)
	require.NoError(t, err)

	b, err := neomatest.DumpRequest(req)
	require.NoError(t, err)
	assert.NotEmpty(t, b)
	assert.Contains(t, string(b), "GET")
}

func TestDoCtxWithHostHeader(t *testing.T) {
	_, api := neomatest.New(t)

	neoma.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/host-test",
		OperationID: "host-test",
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})

	resp := api.Do(http.MethodGet, "/host-test", "Host: example.com")
	assert.Equal(t, http.StatusNoContent, resp.Code)
}

func TestDumpResponseInvalidJSONBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader("this is not valid json {")),
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
	}

	b, err := neomatest.DumpResponse(resp)
	require.NoError(t, err)
	assert.NotEmpty(t, b)
	assert.Contains(t, string(b), "this is not valid json {")
}

func TestDumpRequestInvalidJSONBody(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/test", strings.NewReader("not json {"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	b, dumpErr := neomatest.DumpRequest(req)
	require.NoError(t, dumpErr)
	assert.NotEmpty(t, b)
	assert.Contains(t, string(b), "not json {")
}

type errReader struct{}

func (e *errReader) Read(_ []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

func (e *errReader) Close() error {
	return nil
}

func TestDumpResponseReadError(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"text/plain"}},
		Body:       &errReader{},
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
	}

	_, err := neomatest.DumpResponse(resp)
	assert.Error(t, err, "should return error when body read fails")
}

func TestDumpRequestReadError(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/test", &errReader{})
	require.NoError(t, err)

	_, dumpErr := neomatest.DumpRequest(req)
	assert.Error(t, dumpErr, "should return error when body read fails")
}
