package neoma_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MeGaNeKoS/neoma/adapters/neomastdlib"
	"github.com/MeGaNeKoS/neoma/neoma"
)


func BenchmarkSimpleGet(b *testing.B) {
	mux := http.NewServeMux()
	adapter := neomastdlib.NewAdapter(mux)
	config := neoma.DefaultConfig("Bench API", "1.0.0")
	api := neoma.NewAPI(config, adapter)

	type Output struct {
		Body struct {
			Message string `json:"message"`
		}
	}

	neoma.Get[struct{}, Output](api, "/bench", func(_ context.Context, _ *struct{}) (*Output, error) {
		o := &Output{}
		o.Body.Message = "ok"
		return o, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/bench", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		adapter.ServeHTTP(w, req)
	}
}


func BenchmarkPostWithBody(b *testing.B) {
	mux := http.NewServeMux()
	adapter := neomastdlib.NewAdapter(mux)
	config := neoma.DefaultConfig("Bench API", "1.0.0")
	api := neoma.NewAPI(config, adapter)

	type Input struct {
		Body struct {
			Name  string `json:"name" minLength:"1" maxLength:"100"`
			Email string `json:"email"`
		}
	}
	type Output struct {
		Body struct {
			ID int `json:"id"`
		}
	}

	neoma.Post[Input, Output](api, "/users", func(_ context.Context, _ *Input) (*Output, error) {
		o := &Output{}
		o.Body.ID = 1
		return o, nil
	})

	bodyStr := `{"name":"Alice","email":"alice@example.com"}`

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(bodyStr))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		adapter.ServeHTTP(w, req)
	}
}


func BenchmarkParamParsing(b *testing.B) {
	mux := http.NewServeMux()
	adapter := neomastdlib.NewAdapter(mux)
	config := neoma.DefaultConfig("Bench API", "1.0.0")
	api := neoma.NewAPI(config, adapter)

	type Input struct {
		ID     string `path:"id"`
		Search string `query:"search"`
		Limit  int    `query:"limit"`
	}
	type Output struct {
		Body struct {
			OK bool `json:"ok"`
		}
	}

	neoma.Get[Input, Output](api, "/items/{id}", func(_ context.Context, _ *Input) (*Output, error) {
		o := &Output{}
		o.Body.OK = true
		return o, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/items/abc123?search=foo&limit=25", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		adapter.ServeHTTP(w, req)
	}
}

