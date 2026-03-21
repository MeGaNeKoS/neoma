// Package neomatest provides testing utilities for neoma APIs, including an
// in-memory adapter for sending requests and inspecting responses without
// starting a real HTTP server.
package neomatest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"reflect"
	"strings"
	"testing/iotest"

	"github.com/MeGaNeKoS/neoma/adapters/neomastdlib"
	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/neoma"
)

// TB is a subset of testing.TB used for logging during test requests.
type TB interface {
	Helper()
	Log(args ...any)
	Logf(format string, args ...any)
}

// TestAPI extends core.API with convenience methods for making test requests.
type TestAPI interface {
	core.API
	DoCtx(ctx context.Context, method, path string, args ...any) *httptest.ResponseRecorder
	Do(method, path string, args ...any) *httptest.ResponseRecorder
}

type testAPI struct {
	core.API
	tb TB
}

func (a *testAPI) Do(method, path string, args ...any) *httptest.ResponseRecorder {
	return a.DoCtx(context.Background(), method, path, args...)
}

func (a *testAPI) DoCtx(ctx context.Context, method, path string, args ...any) *httptest.ResponseRecorder {
	a.tb.Helper()
	var b io.Reader
	isJSON := false
	for _, arg := range args {
		kind := reflect.Indirect(reflect.ValueOf(arg)).Kind()
		if reader, ok := arg.(io.Reader); ok {
			b = reader
			break
		} else if _, ok := arg.(string); ok {
		} else if kind == reflect.Struct || kind == reflect.Map || kind == reflect.Slice {
			encoded, err := json.Marshal(arg)
			if err != nil {
				panic(err)
			}
			b = bytes.NewReader(encoded)
			isJSON = true
		} else {
			panic("unsupported argument type, expected string header or io.Reader/slice/map/struct body")
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, path, b)
	if err != nil {
		panic(err)
	}
	req.RequestURI = path
	req.RemoteAddr = "127.0.0.1:12345"
	if isJSON {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, arg := range args {
		if s, ok := arg.(string); ok {
			parts := strings.Split(s, ":")
			req.Header.Set(parts[0], strings.TrimSpace(strings.Join(parts[1:], ":")))

			if strings.ToLower(parts[0]) == "host" {
				req.Host = strings.TrimSpace(parts[1])
			}
		}
	}
	resp := httptest.NewRecorder()

	b2, _ := DumpRequest(req)
	a.tb.Log("Making request:\n" + strings.TrimSpace(string(b2)))

	a.Adapter().ServeHTTP(resp, req)

	b2, _ = DumpResponse(resp.Result())
	a.tb.Log("Got response:\n" + strings.TrimSpace(string(b2)))

	return resp
}

// DumpRequest serializes an HTTP request, including its body, into a byte
// slice suitable for logging.
func DumpRequest(req *http.Request) ([]byte, error) {
	var buf bytes.Buffer
	b, err := httputil.DumpRequest(req, false)

	if err == nil {
		buf.Write(b)
		req.Body, err = dumpBody(req.Body, &buf)
	}

	return buf.Bytes(), err
}

// DumpResponse serializes an HTTP response, including its body, into a byte
// slice suitable for logging.
func DumpResponse(resp *http.Response) ([]byte, error) {
	var buf bytes.Buffer
	b, err := httputil.DumpResponse(resp, false)

	if err == nil {
		buf.Write(b)
		resp.Body, err = dumpBody(resp.Body, &buf)
	}

	return buf.Bytes(), err
}

// New creates an in-memory HTTP handler and TestAPI backed by the standard
// library adapter, ready for use in tests.
func New(tb TB, configs ...core.Config) (http.Handler, TestAPI) {
	for _, config := range configs {
		if config.OpenAPI == nil {
			panic("custom core.Config structs must specify a value for OpenAPI")
		}
	}
	if len(configs) == 0 {
		configs = append(configs, neoma.DefaultConfig("Test API", "1.0.0"))
	}
	mux := http.NewServeMux()
	adapter := neomastdlib.NewAdapter(mux)
	api := neoma.NewAPI(configs[0], adapter)
	return mux, Wrap(tb, api)
}

// NewAdapter returns a new standard library adapter backed by an in-memory
// ServeMux, useful for tests that need a standalone adapter.
func NewAdapter() core.Adapter {
	return neomastdlib.NewAdapter(http.NewServeMux())
}

// NewContext creates a new core.Context from the given operation, request, and
// response writer, backed by the standard library adapter.
func NewContext(op *core.Operation, r *http.Request, w http.ResponseWriter) core.Context {
	return neomastdlib.NewContext(op, r, w)
}

// PrintRequest writes a formatted dump of the HTTP request to stdout.
func PrintRequest(req *http.Request) {
	b, _ := DumpRequest(req)
	b = bytes.ReplaceAll(b, []byte("\r"), []byte(""))
	fmt.Println(string(b))
}

// PrintResponse writes a formatted dump of the HTTP response to stdout.
func PrintResponse(resp *http.Response) {
	b, _ := DumpResponse(resp)
	b = bytes.ReplaceAll(b, []byte("\r"), []byte(""))
	fmt.Println(string(b))
}

// Wrap wraps an existing core.API with test request/response helpers.
func Wrap(tb TB, api core.API) TestAPI {
	return &testAPI{api, tb}
}

func dumpBody(body io.ReadCloser, buf *bytes.Buffer) (io.ReadCloser, error) {
	if body == nil {
		return nil, nil //nolint:nilnil // legitimate nil body
	}

	b, err := io.ReadAll(body)
	if err != nil {
		return io.NopCloser(iotest.ErrReader(err)), err
	}
	_ = body.Close()
	if strings.Contains(buf.String(), "json") {
		if err := json.Indent(buf, b, "", "  "); err != nil {
			// Indent failed, so just write the buffer.
			buf.Write(b)
		}
	} else {
		buf.Write(b)
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}
