package core_test

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddOperationAllMethods(t *testing.T) {
	oapi := &core.OpenAPI{}

	methods := []string{"GET", "PUT", "POST", "DELETE", "OPTIONS", "HEAD", "PATCH", "TRACE"}
	for _, m := range methods {
		oapi.AddOperation(&core.Operation{Method: m, Path: "/test"})
	}

	pi := oapi.Paths["/test"]
	require.NotNil(t, pi)
	assert.NotNil(t, pi.Get)
	assert.NotNil(t, pi.Put)
	assert.NotNil(t, pi.Post)
	assert.NotNil(t, pi.Delete)
	assert.NotNil(t, pi.Options)
	assert.NotNil(t, pi.Head)
	assert.NotNil(t, pi.Patch)
	assert.NotNil(t, pi.Trace)
}

func TestAddOperationHook(t *testing.T) {
	hookCalled := false
	oapi := &core.OpenAPI{
		OnAddOperation: []func(*core.OpenAPI, *core.Operation){
			func(o *core.OpenAPI, op *core.Operation) {
				hookCalled = true
			},
		},
	}
	oapi.AddOperation(&core.Operation{Method: "GET", Path: "/hook"})
	assert.True(t, hookCalled)
}

func TestAddOperationNilPaths(t *testing.T) {
	oapi := &core.OpenAPI{}
	oapi.AddOperation(&core.Operation{Method: "GET", Path: "/new"})
	assert.NotNil(t, oapi.Paths)
	assert.NotNil(t, oapi.Paths["/new"])
}

func TestBaseTypeMap(t *testing.T) {
	typ := reflect.TypeFor[map[string]*int]()
	assert.Equal(t, reflect.TypeFor[int](), core.BaseType(typ))
}

func TestBaseTypePlain(t *testing.T) {
	typ := reflect.TypeFor[string]()
	assert.Equal(t, reflect.TypeFor[string](), core.BaseType(typ))
}

func TestBaseTypeSlice(t *testing.T) {
	typ := reflect.TypeFor[[][]int]()
	assert.Equal(t, reflect.TypeFor[int](), core.BaseType(typ))
}

func TestComponentsMarshalJSON(t *testing.T) {
	c := &core.Components{}
	b, err := c.MarshalJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, b)
}

func TestContactMarshalJSON(t *testing.T) {
	c := &core.Contact{Name: "Test", Email: "test@example.com", URL: "https://example.com"}
	b, err := c.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(b), `"name"`)
}

func TestDerefPointer(t *testing.T) {
	typ := reflect.TypeFor[**int]()
	assert.Equal(t, reflect.TypeFor[int](), core.Deref(typ))
}

func TestDerefSimple(t *testing.T) {
	typ := reflect.TypeFor[int]()
	assert.Equal(t, typ, core.Deref(typ))
}

func TestDiscriminatorMarshalJSON(t *testing.T) {
	d := &core.Discriminator{PropertyName: "type", Mapping: map[string]string{"cat": "#/Cat"}}
	b, err := d.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(b), `"type"`)
}

func TestEncodingMarshalJSON(t *testing.T) {
	e := &core.Encoding{ContentType: "application/json"}
	b, err := e.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(b), `"application/json"`)
}

func TestErrorDetailError(t *testing.T) {
	e := &core.ErrorDetail{Message: "bad"}
	assert.Equal(t, "bad", e.Error())

	e2 := &core.ErrorDetail{Message: "bad", Location: "body.x", Value: 42}
	assert.Contains(t, e2.Error(), "body.x")
	assert.Contains(t, e2.Error(), "42")
}

func TestErrorDetailErrorDetail(t *testing.T) {
	e := &core.ErrorDetail{Message: "msg"}
	assert.Equal(t, e, e.ErrorDetail())
}

func TestErrorWithHeadersSetsHeaders(t *testing.T) {
	err := core.ErrorWithHeaders(
		assert.AnError,
		http.Header{"X-Limit": {"1"}, "X-Debug": {"info"}},
	)
	var he core.Headerer
	require.ErrorAs(t, err, &he)
	require.NotNil(t, he)
	h := he.GetHeaders()
	require.NotNil(t, h)
	assert.Equal(t, "1", h.Get("X-Limit"))
	assert.Equal(t, "info", h.Get("X-Debug"))
}

func TestErrorWithHeadersNewError(t *testing.T) {
	err := core.ErrorWithHeaders(
		assert.AnError,
		http.Header{"X-Test": {"val"}},
	)
	assert.Contains(t, err.Error(), assert.AnError.Error())

	var he core.Headerer
	require.ErrorAs(t, err, &he)
	require.NotNil(t, he)
	h := he.GetHeaders()
	require.NotNil(t, h)
	assert.Equal(t, "val", h.Get("X-Test"))
}

func TestExampleMarshalJSON(t *testing.T) {
	e := &core.Example{Summary: "test", Value: "hello"}
	b, err := e.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(b), `"test"`)
}

func TestExternalDocsMarshalJSON(t *testing.T) {
	ed := &core.ExternalDocs{URL: "https://docs.example.com"}
	b, err := ed.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(b), `"https://docs.example.com"`)
}

func TestInfoMarshalJSON(t *testing.T) {
	i := &core.Info{Title: "API", Version: "1.0"}
	b, err := i.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(b), `"API"`)
}

func TestIsEmptyValue(t *testing.T) {
	assert.True(t, core.IsEmptyValue(reflect.ValueOf("")))
	assert.False(t, core.IsEmptyValue(reflect.ValueOf("x")))
	assert.True(t, core.IsEmptyValue(reflect.ValueOf(0)))
	assert.False(t, core.IsEmptyValue(reflect.ValueOf(1)))
	assert.True(t, core.IsEmptyValue(reflect.ValueOf(false)))
	assert.False(t, core.IsEmptyValue(reflect.ValueOf(true)))
	assert.True(t, core.IsEmptyValue(reflect.ValueOf([]int(nil))))
	assert.False(t, core.IsEmptyValue(reflect.ValueOf([]int{1})))
}

func TestIsEmptyValueAllTypes(t *testing.T) {
	assert.True(t, core.IsEmptyValue(reflect.ValueOf(uint(0))))
	assert.False(t, core.IsEmptyValue(reflect.ValueOf(uint(1))))
	assert.True(t, core.IsEmptyValue(reflect.ValueOf(float64(0))))
	assert.False(t, core.IsEmptyValue(reflect.ValueOf(1.5)))
	var p *int
	assert.True(t, core.IsEmptyValue(reflect.ValueOf(&p).Elem()))
	v := 5
	assert.False(t, core.IsEmptyValue(reflect.ValueOf(&v)))
	assert.True(t, core.IsEmptyValue(reflect.ValueOf([0]int{})))
	assert.False(t, core.IsEmptyValue(reflect.ValueOf([1]int{1})))
	assert.True(t, core.IsEmptyValue(reflect.ValueOf(map[string]int{})))
	assert.False(t, core.IsEmptyValue(reflect.ValueOf(map[string]int{"a": 1})))
	assert.False(t, core.IsEmptyValue(reflect.ValueOf(struct{}{})))
}

func TestIsNilValue(t *testing.T) {
	assert.True(t, core.IsNilValue(nil))
	var p *int
	assert.True(t, core.IsNilValue(p))
	v := 42
	assert.False(t, core.IsNilValue(&v))
	assert.False(t, core.IsNilValue(42))
}

func TestIsNilValueAllTypes(t *testing.T) {
	var s []int
	assert.True(t, core.IsNilValue(s))
	assert.False(t, core.IsNilValue([]int{}))
	var m map[string]int
	assert.True(t, core.IsNilValue(m))
	var ch chan int
	assert.True(t, core.IsNilValue(ch))
	var fn func()
	assert.True(t, core.IsNilValue(fn))
	assert.False(t, core.IsNilValue(0))
	assert.False(t, core.IsNilValue("hello"))
}

func TestLicenseMarshalJSON(t *testing.T) {
	l := &core.License{Name: "MIT", URL: "https://mit.edu"}
	b, err := l.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(b), `"MIT"`)
}

func TestLinkMarshalJSON(t *testing.T) {
	l := &core.Link{OperationID: "getUser"}
	b, err := l.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(b), `"getUser"`)
}

func TestMarshalJSONOmitEmpty(t *testing.T) {
	fields := []core.JSONFieldInfo{
		{Name: "present", Value: "hello", Omit: core.OmitEmpty},
		{Name: "empty", Value: "", Omit: core.OmitEmpty},
		{Name: "always", Value: "", Omit: core.OmitNever},
	}
	b, err := core.MarshalJSON(fields, nil)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))

	assert.Contains(t, m, "present")
	assert.NotContains(t, m, "empty")
	assert.Contains(t, m, "always")
}

func TestMarshalJSONOmitNil(t *testing.T) {
	fields := []core.JSONFieldInfo{
		{Name: "nilval", Value: nil, Omit: core.OmitNil},
		{Name: "nonnil", Value: "ok", Omit: core.OmitNil},
	}
	b, err := core.MarshalJSON(fields, nil)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))

	assert.NotContains(t, m, "nilval")
	assert.Contains(t, m, "nonnil")
}

func TestMarshalJSONWithExtensions(t *testing.T) {
	fields := []core.JSONFieldInfo{
		{Name: "a", Value: 1, Omit: core.OmitNever},
	}
	ext := map[string]any{"x-custom": true}
	b, err := core.MarshalJSON(fields, ext)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))

	assert.Equal(t, true, m["x-custom"])
	assert.InDelta(t, float64(1), m["a"], 0.01)
}

func TestMediaTypeMarshalJSON(t *testing.T) {
	mt := &core.MediaType{Schema: &core.Schema{Type: "string"}}
	b, err := mt.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(b), `"schema"`)
}

func TestMiddlewareChainEmpty(t *testing.T) {
	var m core.Middlewares
	called := false
	handler := m.Handler(func(ctx core.Context) { called = true })
	handler(nil)
	assert.True(t, called)
}

func TestMiddlewareChainMultiple(t *testing.T) {
	var order []int
	m := core.Middlewares{
		func(ctx core.Context, next func(core.Context)) {
			order = append(order, 1)
			next(ctx)
		},
		func(ctx core.Context, next func(core.Context)) {
			order = append(order, 2)
			next(ctx)
		},
	}
	handler := m.Handler(func(ctx core.Context) {
		order = append(order, 3)
	})
	handler(nil)
	assert.Equal(t, []int{1, 2, 3}, order)
}

func TestOAuthFlowMarshalJSON(t *testing.T) {
	f := &core.OAuthFlow{TokenURL: "https://oauth.example.com/token", Scopes: map[string]string{"read": "read"}}
	b, err := f.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(b), `"tokenUrl"`)
}

func TestOAuthFlowsMarshalJSON(t *testing.T) {
	f := &core.OAuthFlows{
		Password: &core.OAuthFlow{TokenURL: "https://oauth.example.com/token", Scopes: map[string]string{}},
	}
	b, err := f.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(b), `"password"`)
}

func TestOpenAPIMarshalJSON(t *testing.T) {
	o := &core.OpenAPI{OpenAPI: core.OpenAPIVersion31, Info: &core.Info{Title: "Test", Version: "1.0"}}
	b, err := o.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(b), core.OpenAPIVersion31)
}

func TestOperationMarshalJSON(t *testing.T) {
	op := &core.Operation{OperationID: "listItems", Summary: "List items"}
	b, err := op.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(b), `"listItems"`)
}

func TestParamMarshalJSON(t *testing.T) {
	p := &core.Param{Name: "id", In: "path", Required: true}
	b, err := p.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(b), `"id"`)
}

func TestPathBufferBytes(t *testing.T) {
	pb := core.NewPathBuffer([]byte{}, 0)
	pb.Push("abc")
	assert.Equal(t, []byte("abc"), pb.Bytes())
}

func TestPathBufferLen(t *testing.T) {
	pb := core.NewPathBuffer([]byte{}, 0)
	assert.Equal(t, 0, pb.Len())

	pb.Push("x")
	assert.Equal(t, 1, pb.Len())
}

func TestPathBufferPushIndex(t *testing.T) {
	pb := core.NewPathBuffer([]byte{}, 0)
	pb.Push("items")
	pb.PushIndex(3)
	assert.Equal(t, "items[3]", pb.String())

	pb.Push("name")
	assert.Equal(t, "items[3].name", pb.String())

	pb.Pop()
	assert.Equal(t, "items[3]", pb.String())

	pb.Pop()
	assert.Equal(t, "items", pb.String())
}

func TestPathBufferPushPop(t *testing.T) {
	pb := core.NewPathBuffer([]byte{}, 0)
	pb.Push("foo")
	assert.Equal(t, "foo", pb.String())

	pb.Push("bar")
	assert.Equal(t, "foo.bar", pb.String())

	pb.Pop()
	assert.Equal(t, "foo", pb.String())

	pb.Pop()
	assert.Empty(t, pb.String())
}

func TestPathBufferReset(t *testing.T) {
	pb := core.NewPathBuffer([]byte{}, 0)
	pb.Push("a")
	pb.Push("b")
	assert.Positive(t, pb.Len())

	pb.Reset()
	assert.Equal(t, 0, pb.Len())
	assert.Empty(t, pb.String())
}

func TestPathBufferWith(t *testing.T) {
	pb := core.NewPathBuffer([]byte{}, 0)
	pb.Push("root")

	got := pb.With("child")
	assert.Equal(t, "root.child", got)
	assert.Equal(t, "root", pb.String())
}

func TestPathBufferWithIndex(t *testing.T) {
	pb := core.NewPathBuffer([]byte{}, 0)
	pb.Push("arr")

	got := pb.WithIndex(7)
	assert.Equal(t, "arr[7]", got)
	assert.Equal(t, "arr", pb.String())
}

func TestPathItemMarshalJSON(t *testing.T) {
	pi := &core.PathItem{Summary: "test"}
	b, err := pi.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(b), `"test"`)
}

func TestPrecomputeMessages(t *testing.T) {
	lo := 1.0
	hi := 10.0
	mult := 2.0
	minLen := 1
	maxLen := 50
	minItems := 0
	maxItems := 100
	minProps := 1
	maxProps := 5

	s := &core.Schema{
		Type:              "object",
		Minimum:           &lo,
		Maximum:           &hi,
		ExclusiveMinimum:  &lo,
		ExclusiveMaximum:  &hi,
		MultipleOf:        &mult,
		MinLength:         &minLen,
		MaxLength:         &maxLen,
		MinItems:          &minItems,
		MaxItems:          &maxItems,
		MinProperties:     &minProps,
		MaxProperties:     &maxProps,
		Pattern:           "^[a-z]+$",
		Enum:              []any{"a", "b"},
		Required:          []string{"name"},
		DependentRequired: map[string][]string{"a": {"b"}},
		Properties: map[string]*core.Schema{
			"name": {Type: "string"},
		},
		Items: &core.Schema{Type: "string"},
		OneOf: []*core.Schema{{Type: "string"}},
		AnyOf: []*core.Schema{{Type: "integer"}},
		AllOf: []*core.Schema{{Type: "number"}},
		Not:   &core.Schema{Type: "boolean"},
	}
	s.PrecomputeMessages()

	assert.NotEmpty(t, s.MsgMinimum)
	assert.NotEmpty(t, s.MsgMaximum)
	assert.NotEmpty(t, s.MsgExclusiveMinimum)
	assert.NotEmpty(t, s.MsgExclusiveMaximum)
	assert.NotEmpty(t, s.MsgMultipleOf)
	assert.NotEmpty(t, s.MsgMinLength)
	assert.NotEmpty(t, s.MsgMaxLength)
	assert.NotEmpty(t, s.MsgPattern)
	assert.NotEmpty(t, s.MsgMinItems)
	assert.NotEmpty(t, s.MsgMaxItems)
	assert.NotEmpty(t, s.MsgMinProperties)
	assert.NotEmpty(t, s.MsgMaxProperties)
	assert.NotEmpty(t, s.MsgEnum)
	assert.NotEmpty(t, s.MsgRequired)
	assert.NotEmpty(t, s.MsgDependentRequired)
	assert.NotNil(t, s.PatternRe)
	assert.NotEmpty(t, s.PropertyNames)
	assert.NotEmpty(t, s.RequiredMap)
}

func TestPrecomputeMessagesPatternDescription(t *testing.T) {
	s := &core.Schema{
		Pattern:            "^[a-z]+$",
		PatternDescription: "lowercase letters only",
	}
	s.PrecomputeMessages()
	assert.Contains(t, s.MsgPattern, "lowercase letters only")
}

func TestRequestBodyMarshalJSON(t *testing.T) {
	rb := &core.RequestBody{Description: "body", Required: true}
	b, err := rb.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(b), `"body"`)
}

func TestResponseMarshalJSON(t *testing.T) {
	r := &core.Response{Description: "OK"}
	b, err := r.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(b), `"OK"`)
}

func TestSchemaMarshalJSONBasic(t *testing.T) {
	s := &core.Schema{Type: "string", Description: "a test"}
	b, err := s.MarshalJSON()
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	assert.Equal(t, "string", m["type"])
	assert.Equal(t, "a test", m["description"])
}

func TestSchemaMarshalJSONBinaryFormat(t *testing.T) {
	s := &core.Schema{Type: "string", Format: "binary"}
	b, err := s.MarshalJSON()
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	assert.Equal(t, "application/octet-stream", m["contentMediaType"])
}

func TestSchemaMarshalJSONExtensions(t *testing.T) {
	s := &core.Schema{
		Type:       "string",
		Extensions: map[string]any{"x-custom": "hello"},
	}
	b, err := s.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(b), `"x-custom"`)
}

func TestSchemaMarshalJSONHiddenFields(t *testing.T) {
	s := &core.Schema{
		Type: "object",
		Properties: map[string]*core.Schema{
			"visible": {Type: "string"},
			"hidden":  {Type: "string", Hidden: true},
		},
	}

	b, err := s.MarshalJSON()
	require.NoError(t, err)
	assert.NotContains(t, string(b), `"hidden"`)
	assert.Contains(t, string(b), `"visible"`)
}

func TestSchemaMarshalJSONNullable(t *testing.T) {
	s := &core.Schema{Type: "string", Nullable: true}
	b, err := s.MarshalJSON()
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	typ, ok := m["type"].([]any)
	require.True(t, ok)
	assert.Contains(t, typ, "string")
	assert.Contains(t, typ, "null")
}

func TestSecuritySchemeMarshalJSON(t *testing.T) {
	ss := &core.SecurityScheme{Type: "http", Scheme: "bearer"}
	b, err := ss.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(b), `"bearer"`)
}

func TestServerMarshalJSON(t *testing.T) {
	s := &core.Server{URL: "https://api.example.com"}
	b, err := s.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(b), `"https://api.example.com"`)
}

func TestServerVariableMarshalJSON(t *testing.T) {
	sv := &core.ServerVariable{Default: "prod", Description: "env"}
	b, err := sv.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(b), `"prod"`)
}


func TestSetReadDeadlineDirectly(t *testing.T) {
	dw := &deadlineWriter{ResponseWriter: httptest.NewRecorder()}
	err := core.SetReadDeadline(dw, time.Now())
	require.NoError(t, err)
	assert.True(t, dw.called)
}

func TestSetReadDeadlineNotSupported(t *testing.T) {
	w := httptest.NewRecorder()
	err := core.SetReadDeadline(w, time.Now())
	assert.ErrorIs(t, err, http.ErrNotSupported)
}

func TestSetReadDeadlineUnwrap(t *testing.T) {
	dw := &deadlineWriter{ResponseWriter: httptest.NewRecorder()}
	ww := &wrappedWriter{ResponseWriter: httptest.NewRecorder(), inner: dw}
	err := core.SetReadDeadline(ww, time.Now())
	require.NoError(t, err)
	assert.True(t, dw.called)
}

func TestTagMarshalJSON(t *testing.T) {
	tag := &core.Tag{Name: "users"}
	b, err := tag.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(b), `"users"`)
}

func TestValidateResultAdd(t *testing.T) {
	pb := core.NewPathBuffer([]byte{}, 0)
	pb.Push("body")
	pb.Push("name")

	var res core.ValidateResult
	res.Add(pb, "bad", "expected string")

	require.True(t, res.HasErrors())
	require.Len(t, res.Errors, 1)
	assert.Contains(t, res.Errors[0].Error(), "expected string")
}

func TestValidateResultAddf(t *testing.T) {
	pb := core.NewPathBuffer([]byte{}, 0)
	pb.Push("body")

	var res core.ValidateResult
	res.Addf(pb, 42, "expected value <= %d", 10)

	require.True(t, res.HasErrors())
	assert.Contains(t, res.Errors[0].Error(), "expected value <= 10")
}

func TestValidateResultHasErrorsEmpty(t *testing.T) {
	var res core.ValidateResult
	assert.False(t, res.HasErrors())
}

func TestValidateResultReset(t *testing.T) {
	pb := core.NewPathBuffer([]byte{}, 0)
	var res core.ValidateResult
	res.Add(pb, nil, "err")
	require.True(t, res.HasErrors())

	res.Reset()
	assert.False(t, res.HasErrors())
}

func TestWithContext(t *testing.T) {
	base := &stubContext{goCtx: context.Background()}
	newCtx := context.WithValue(context.Background(), ctxKey("key"), "val")

	wrapped := core.WithContext(base, newCtx)
	assert.Equal(t, "val", wrapped.Context().Value(ctxKey("key")))
}

func TestWithValue(t *testing.T) {
	base := &stubContext{goCtx: context.Background()}
	wrapped := core.WithValue(base, ctxKey("mykey"), 123)
	assert.Equal(t, 123, wrapped.Context().Value(ctxKey("mykey")))
}

func TestUnwrapContextPeelsLayers(t *testing.T) {
	base := &stubContext{goCtx: context.Background()}
	layer1 := core.WithContext(base, context.WithValue(context.Background(), ctxKey("a"), 1))
	layer2 := core.WithValue(layer1, ctxKey("b"), 2)

	unwrapped := core.UnwrapContext(layer2)
	assert.Equal(t, base, unwrapped)
}

func TestUnwrapContextNoLayers(t *testing.T) {
	base := &stubContext{goCtx: context.Background()}
	unwrapped := core.UnwrapContext(base)
	assert.Equal(t, base, unwrapped)
}

func TestErrWithHeadersUnwrap(t *testing.T) {
	inner := assert.AnError
	wrapped := core.ErrorWithHeaders(inner, http.Header{"X-Test": {"v"}})

	require.ErrorIs(t, wrapped, inner)
}

func TestErrWithHeadersUnwrapReturnsInner(t *testing.T) {
	inner := errors.New("root cause")
	wrapped := core.ErrorWithHeaders(inner, http.Header{"X-Key": {"val"}})

	got := errors.Unwrap(wrapped)
	assert.Equal(t, inner, got)
}

type ctxKey string

type deadlineWriter struct {
	http.ResponseWriter
	called bool
}

func (d *deadlineWriter) SetReadDeadline(_ time.Time) error {
	d.called = true
	return nil
}

type stubContext struct {
	goCtx context.Context
}

func (s *stubContext) Operation() *core.Operation {
	return &core.Operation{Method: "GET", Path: "/test"}
}
func (s *stubContext) Context() context.Context                   { return s.goCtx }
func (s *stubContext) TLS() *tls.ConnectionState                  { return nil }
func (s *stubContext) Version() core.ProtoVersion                 { return core.ProtoVersion{} }
func (s *stubContext) Method() string                             { return "GET" }
func (s *stubContext) Host() string                               { return "" }
func (s *stubContext) RemoteAddr() string                         { return "" }
func (s *stubContext) URL() url.URL                               { return url.URL{} }
func (s *stubContext) Param(string) string                        { return "" }
func (s *stubContext) Query(string) string                        { return "" }
func (s *stubContext) Header(string) string                       { return "" }
func (s *stubContext) EachHeader(func(string, string))            {}
func (s *stubContext) BodyReader() io.Reader                      { return nil }
func (s *stubContext) GetMultipartForm() (*multipart.Form, error) { return nil, nil }
func (s *stubContext) SetReadDeadline(time.Time) error            { return nil }
func (s *stubContext) SetStatus(int)                              {}
func (s *stubContext) Status() int                                { return 0 }
func (s *stubContext) SetHeader(string, string)                   {}
func (s *stubContext) AppendHeader(string, string)                {}
func (s *stubContext) GetResponseHeader(string) string            { return "" }
func (s *stubContext) DeleteResponseHeader(string)                {}
func (s *stubContext) BodyWriter() io.Writer                      { return io.Discard }
func (s *stubContext) MatchedPattern() string                     { return "" }

type wrappedWriter struct {
	http.ResponseWriter
	inner http.ResponseWriter
}

func (w *wrappedWriter) Unwrap() http.ResponseWriter {
	return w.inner
}
