package binding

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRegistry() core.Registry {
	return schema.NewMapRegistry(schema.DefaultSchemaNamer)
}

type mockContext struct {
	op      *core.Operation
	params  map[string]string
	queries map[string]string
	headers map[string]string
	body    io.Reader
	status  int
	respHdr map[string]string
	writer  bytes.Buffer
	urlVal  url.URL
}

func (m *mockContext) Operation() *core.Operation { return m.op }
func (m *mockContext) Context() context.Context   { return context.Background() }
func (m *mockContext) TLS() *tls.ConnectionState  { return nil }
func (m *mockContext) Version() core.ProtoVersion {
	return core.ProtoVersion{Proto: "HTTP/1.1", ProtoMajor: 1, ProtoMinor: 1}
}
func (m *mockContext) Method() string            { return m.op.Method }
func (m *mockContext) Host() string              { return "localhost" }
func (m *mockContext) RemoteAddr() string        { return "127.0.0.1" }
func (m *mockContext) URL() url.URL              { return m.urlVal }
func (m *mockContext) Param(name string) string  { return m.params[name] }
func (m *mockContext) Query(name string) string  { return m.queries[name] }
func (m *mockContext) Header(name string) string { return m.headers[name] }
func (m *mockContext) EachHeader(cb func(string, string)) {
	for k, v := range m.headers {
		cb(k, v)
	}
}
func (m *mockContext) BodyReader() io.Reader                      { return m.body }
func (m *mockContext) GetMultipartForm() (*multipart.Form, error) { return nil, nil }
func (m *mockContext) SetReadDeadline(_ time.Time) error          { return nil }
func (m *mockContext) SetStatus(code int)                         { m.status = code }
func (m *mockContext) Status() int                                { return m.status }
func (m *mockContext) SetHeader(name, value string)               { m.respHdr[name] = value }
func (m *mockContext) AppendHeader(name, value string)            { m.respHdr[name] = value }
func (m *mockContext) GetResponseHeader(name string) string       { return m.respHdr[name] }
func (m *mockContext) DeleteResponseHeader(name string)           { delete(m.respHdr, name) }
func (m *mockContext) BodyWriter() io.Writer                      { return &m.writer }
func (m *mockContext) MatchedPattern() string                     { return m.op.Path }

func newMockCtx(method, path string) *mockContext {
	return &mockContext{
		op:      &core.Operation{Method: method, Path: path},
		params:  map[string]string{},
		queries: map[string]string{},
		headers: map[string]string{},
		respHdr: map[string]string{},
	}
}

type mockAPI struct {
	errHandler core.ErrorHandler
}

func (m *mockAPI) Adapter() core.Adapter                                  { return nil }
func (m *mockAPI) OpenAPI() *core.OpenAPI                                 { return &core.OpenAPI{} }
func (m *mockAPI) Negotiate(_ string) (string, error)                     { return "application/json", nil }
func (m *mockAPI) Transform(_ core.Context, _ string, v any) (any, error) { return v, nil }
func (m *mockAPI) Marshal(w io.Writer, _ string, v any) error             { return json.NewEncoder(w).Encode(v) }
func (m *mockAPI) Unmarshal(_ string, data []byte, v any) error           { return json.Unmarshal(data, v) }
func (m *mockAPI) UseMiddleware(_ ...core.MiddlewareFunc)                 {}
func (m *mockAPI) Middlewares() core.Middlewares                          { return nil }
func (m *mockAPI) ErrorHandler() core.ErrorHandler                        { return m.errHandler }
func (m *mockAPI) GenerateOperationID(method, path string) string         { return method + path }
func (m *mockAPI) UseGlobalMiddleware(_ ...core.MiddlewareFunc)           {}

type testResolver struct {
	Called bool
}

func (r *testResolver) Resolve(_ core.Context) []error {
	r.Called = true
	return nil
}

type testInput struct {
	ID      int    `path:"id"`
	Search  string `query:"search"`
	Token   string `header:"Authorization"`
	MyField string `cookie:"session"`
	Body    struct {
		Name string `json:"name" default:"world" example:"Alice"`
		Age  int    `json:"age" example:"30"`
	}
	testResolver
}

type testOutputBody struct {
	Message string `json:"message" example:"hello"`
}

type testOutput struct {
	Status int
	Body   testOutputBody
	ETag   string    `header:"ETag"`
	Date   time.Time `header:"Last-Modified"`
}

func TestAnalyzeInput(t *testing.T) {
	reg := newRegistry()
	op := &core.Operation{Method: "POST", Path: "/items/{id}", OperationID: "createItem"}
	meta := AnalyzeInput[testInput](op, reg, false)

	require.NotNil(t, meta)
	assert.True(t, meta.HasInputBody)
	assert.NotNil(t, meta.InputBodyIndex)
	assert.NotNil(t, meta.InSchema)
	assert.NotNil(t, meta.Params)
	assert.NotNil(t, meta.Resolvers)
	assert.NotNil(t, meta.Defaults)

	assert.GreaterOrEqual(t, len(meta.Params.Paths), 3, "should find path, query, header, cookie params")
}

func TestAnalyzeOutput(t *testing.T) {
	reg := newRegistry()
	op := &core.Operation{Method: "GET", Path: "/items", OperationID: "listItems"}
	meta := AnalyzeOutput[testOutput](op, reg)

	require.NotNil(t, meta)
	assert.GreaterOrEqual(t, meta.StatusIndex, 0)
	assert.GreaterOrEqual(t, meta.BodyIndex, 0)
	assert.False(t, meta.BodyFunc)
	assert.NotNil(t, meta.Headers)
	assert.Equal(t, http.StatusOK, op.DefaultStatus)
}

func TestAnalyzeOutput_NoBody(t *testing.T) {
	type noBodyOutput struct{}
	reg := newRegistry()
	op := &core.Operation{Method: "DELETE", Path: "/items/{id}", OperationID: "deleteItem"}
	meta := AnalyzeOutput[noBodyOutput](op, reg)

	assert.Equal(t, -1, meta.StatusIndex)
	assert.Equal(t, -1, meta.BodyIndex)
	assert.Equal(t, http.StatusNoContent, op.DefaultStatus)
}

func TestAnalyzeOutput_StatusNotInt_Panics(t *testing.T) {
	type badStatus struct {
		Status string
	}
	reg := newRegistry()
	op := &core.Operation{Method: "GET", Path: "/x", OperationID: "x"}
	assert.Panics(t, func() {
		AnalyzeOutput[badStatus](op, reg)
	})
}

func TestAnalyzeOutput_StreamBody(t *testing.T) {
	type streamOutput struct {
		Body func(core.Context, core.API)
	}
	reg := newRegistry()
	op := &core.Operation{Method: "GET", Path: "/stream", OperationID: "stream"}
	meta := AnalyzeOutput[streamOutput](op, reg)

	assert.True(t, meta.BodyFunc)
	assert.GreaterOrEqual(t, meta.BodyIndex, 0)
}

func TestAnalyzeOutput_Head(t *testing.T) {
	type emptyOut struct{}
	reg := newRegistry()
	op := &core.Operation{Method: http.MethodHead, Path: "/ping", OperationID: "headPing"}
	meta := AnalyzeOutput[emptyOut](op, reg)

	assert.Equal(t, -1, meta.BodyIndex)
	assert.Equal(t, http.StatusOK, op.DefaultStatus)
}

func TestReadBody_Success(t *testing.T) {
	ctx := newMockCtx("POST", "/")
	ctx.body = strings.NewReader(`{"name":"test"}`)

	var buf bytes.Buffer
	err := ReadBody(&buf, ctx, 1024)
	assert.Nil(t, err)
	assert.JSONEq(t, `{"name":"test"}`, buf.String())
}

func TestReadBody_LimitExceeded(t *testing.T) {
	ctx := newMockCtx("POST", "/")
	ctx.body = strings.NewReader("abcdefghij")

	var buf bytes.Buffer
	err := ReadBody(&buf, ctx, 10)
	require.NotNil(t, err)
	assert.Equal(t, http.StatusRequestEntityTooLarge, err.Code)
}

func TestReadBody_NilReader(t *testing.T) {
	ctx := newMockCtx("POST", "/")
	ctx.body = nil

	var buf bytes.Buffer
	err := ReadBody(&buf, ctx, 1024)
	assert.Nil(t, err)
	assert.Empty(t, buf.String())
}

func TestReadBody_NoLimit(t *testing.T) {
	ctx := newMockCtx("POST", "/")
	ctx.body = strings.NewReader("hello world")

	var buf bytes.Buffer
	err := ReadBody(&buf, ctx, 0)
	assert.Nil(t, err)
	assert.Equal(t, "hello world", buf.String())
}

func TestProcessRegularMsgBody_Valid(t *testing.T) {
	type Body struct {
		Name string `json:"name"`
	}
	type Input struct {
		Body Body
	}
	v := reflect.New(reflect.TypeFor[Input]()).Elem()
	cfg := BodyProcessingConfig{
		Body:           []byte(`{"name":"test"}`),
		Op:             core.Operation{},
		Value:          v,
		HasInputBody:   true,
		InputBodyIndex: []int{0},
		Unmarshaler:    json.Unmarshal,
		Validator:      func(data any, res *core.ValidateResult) {},
		Defaults:       &FindResult[any]{},
		Result:         &core.ValidateResult{},
	}
	errStatus, cErr := ProcessRegularMsgBody(cfg)
	assert.Nil(t, cErr)
	assert.Equal(t, -1, errStatus)
	assert.Equal(t, "test", v.Field(0).FieldByName("Name").String())
}

func TestProcessRegularMsgBody_InvalidJSON(t *testing.T) {
	type Body struct {
		Name string `json:"name"`
	}
	type Input struct {
		Body Body
	}
	v := reflect.New(reflect.TypeFor[Input]()).Elem()
	res := &core.ValidateResult{}
	cfg := BodyProcessingConfig{
		Body:           []byte(`{invalid`),
		Op:             core.Operation{},
		Value:          v,
		HasInputBody:   true,
		InputBodyIndex: []int{0},
		Unmarshaler:    json.Unmarshal,
		Validator:      func(data any, res *core.ValidateResult) {},
		Defaults:       &FindResult[any]{},
		Result:         res,
	}
	errStatus, cErr := ProcessRegularMsgBody(cfg)
	assert.Nil(t, cErr)
	assert.Equal(t, http.StatusBadRequest, errStatus)
	assert.NotEmpty(t, res.Errors)
}

func TestProcessRegularMsgBody_EmptyRequired(t *testing.T) {
	cfg := BodyProcessingConfig{
		Body: nil,
		Op:   core.Operation{RequestBody: &core.RequestBody{Required: true}},
	}
	_, cErr := ProcessRegularMsgBody(cfg)
	require.NotNil(t, cErr)
	assert.Equal(t, http.StatusBadRequest, cErr.Code)
}

func TestProcessRegularMsgBody_EmptyNotRequired(t *testing.T) {
	cfg := BodyProcessingConfig{
		Body: nil,
		Op:   core.Operation{},
	}
	_, cErr := ProcessRegularMsgBody(cfg)
	assert.Nil(t, cErr)
}

func TestProcessRegularMsgBody_NoInputBody(t *testing.T) {
	cfg := BodyProcessingConfig{
		Body:         []byte(`{"x":1}`),
		Op:           core.Operation{},
		HasInputBody: false,
		Unmarshaler:  json.Unmarshal,
		Validator:    func(data any, res *core.ValidateResult) {},
		Result:       &core.ValidateResult{},
	}
	_, cErr := ProcessRegularMsgBody(cfg)
	assert.Nil(t, cErr)
}

func TestParseDeepObjectQuery(t *testing.T) {
	q := url.Values{
		"filter[status]": {"active"},
		"filter[type]":   {"admin"},
		"other":          {"ignored"},
	}
	result := ParseDeepObjectQuery(q, "filter")
	assert.Equal(t, "active", result["status"])
	assert.Equal(t, "admin", result["type"])
	assert.NotContains(t, result, "other")
}

func TestParseDeepObjectQuery_NoMatch(t *testing.T) {
	q := url.Values{
		"filter[status]": {"active"},
	}
	result := ParseDeepObjectQuery(q, "xyz")
	assert.Empty(t, result)
}

func TestSetDeepObjectValue_Struct(t *testing.T) {
	type Filter struct {
		Status string `json:"status"`
		Count  int    `json:"count"`
	}
	pb := core.NewPathBuffer(nil, 0)
	res := &core.ValidateResult{}
	f := reflect.New(reflect.TypeFor[Filter]()).Elem()
	data := map[string]string{"status": "active", "count": "5"}

	result := SetDeepObjectValue(pb, res, f, data)
	assert.Equal(t, "active", result["status"])
	assert.Equal(t, 5, result["count"])
	assert.Empty(t, res.Errors)
}

func TestSetDeepObjectValue_StructDefault(t *testing.T) {
	type Filter struct {
		Status string `json:"status" default:"pending"`
	}
	pb := core.NewPathBuffer(nil, 0)
	res := &core.ValidateResult{}
	f := reflect.New(reflect.TypeFor[Filter]()).Elem()
	data := map[string]string{}

	result := SetDeepObjectValue(pb, res, f, data)
	assert.Equal(t, "pending", result["status"])
}

func TestSetDeepObjectValue_Map(t *testing.T) {
	pb := core.NewPathBuffer(nil, 0)
	res := &core.ValidateResult{}
	f := reflect.New(reflect.TypeFor[map[string]int]()).Elem()
	data := map[string]string{"a": "1", "b": "2"}

	result := SetDeepObjectValue(pb, res, f, data)
	assert.Equal(t, 1, result["a"])
	assert.Equal(t, 2, result["b"])
	assert.Empty(t, res.Errors)
}

func TestSetDeepObjectValue_MapInvalidValue(t *testing.T) {
	pb := core.NewPathBuffer(nil, 0)
	res := &core.ValidateResult{}
	f := reflect.New(reflect.TypeFor[map[string]int]()).Elem()
	data := map[string]string{"a": "notanint"}

	SetDeepObjectValue(pb, res, f, data)
	assert.NotEmpty(t, res.Errors)
}

func TestSetDeepObjectValue_MapNonStringKey(t *testing.T) {
	pb := core.NewPathBuffer(nil, 0)
	res := &core.ValidateResult{}
	f := reflect.New(reflect.TypeFor[map[int]string]()).Elem()
	data := map[string]string{"a": "val"}

	result := SetDeepObjectValue(pb, res, f, data)
	assert.Empty(t, result)
}

func TestSetDeepObjectValue_StructInvalidField(t *testing.T) {
	type Filter struct {
		Count int `json:"count"`
	}
	pb := core.NewPathBuffer(nil, 0)
	res := &core.ValidateResult{}
	f := reflect.New(reflect.TypeFor[Filter]()).Elem()
	data := map[string]string{"count": "abc"}

	SetDeepObjectValue(pb, res, f, data)
	assert.NotEmpty(t, res.Errors)
}

func TestFindDefaults(t *testing.T) {
	type Input struct {
		Name string `json:"name" default:"world"`
		Age  int    `json:"age" default:"25"`
	}
	reg := newRegistry()
	result := findDefaults[Input](reg)

	assert.NotEmpty(t, result.Paths, "should find fields with default tags")
}

func TestFindDefaults_NoDefaults(t *testing.T) {
	type Input struct {
		Name string `json:"name"`
	}
	reg := newRegistry()
	result := findDefaults[Input](reg)

	assert.Empty(t, result.Paths)
}

func TestFindDefaults_PointerStructPanics(t *testing.T) {
	type Inner struct {
		X int
	}
	type Input struct {
		Ptr *Inner `default:"something"`
	}
	reg := newRegistry()
	assert.Panics(t, func() {
		findDefaults[Input](reg)
	})
}

func TestBuildExampleFromType_Struct(t *testing.T) {
	type Item struct {
		Name  string `json:"name" example:"widget"`
		Count int    `json:"count" example:"42"`
	}
	result := buildExampleFromType(reflect.TypeFor[Item]())
	require.NotNil(t, result)

	item, ok := result.(*Item)
	require.True(t, ok)
	assert.Equal(t, "widget", item.Name)
	assert.Equal(t, 42, item.Count)
}

func TestBuildExampleFromType_NonStruct(t *testing.T) {
	result := buildExampleFromType(reflect.TypeFor[string]())
	assert.Nil(t, result)
}

func TestBuildExampleFromType_NoExamples(t *testing.T) {
	type Plain struct {
		X int
	}
	result := buildExampleFromType(reflect.TypeFor[Plain]())
	assert.Nil(t, result)
}

func TestBuildExampleValue_Struct(t *testing.T) {
	type Item struct {
		Name string `json:"name" example:"hello"`
	}
	v := reflect.New(reflect.TypeFor[Item]()).Elem()
	ok := buildExampleValue(v)
	assert.True(t, ok)
	assert.Equal(t, "hello", v.FieldByName("Name").String())
}

func TestBuildExampleValue_Slice(t *testing.T) {
	type Item struct {
		Name string `json:"name" example:"elem"`
	}
	v := reflect.New(reflect.TypeFor[[]Item]()).Elem()
	ok := buildExampleValue(v)
	assert.True(t, ok)
	assert.Equal(t, 1, v.Len())
	assert.Equal(t, "elem", v.Index(0).FieldByName("Name").String())
}

func TestBuildExampleValue_Map(t *testing.T) {
	type Item struct {
		Name string `json:"name" example:"val"`
	}
	v := reflect.New(reflect.TypeFor[map[string]Item]()).Elem()
	ok := buildExampleValue(v)
	assert.True(t, ok)
	assert.Equal(t, 1, v.Len())
}

func TestBuildExampleValue_Invalid(t *testing.T) {
	ok := buildExampleValue(reflect.Value{})
	assert.False(t, ok)
}

func TestSetExampleFromTag_String(t *testing.T) {
	v := reflect.New(reflect.TypeFor[string]()).Elem()
	ok := setExampleFromTag(v, reflect.TypeFor[string](), "hello")
	assert.True(t, ok)
	assert.Equal(t, "hello", v.String())
}

func TestSetExampleFromTag_Int(t *testing.T) {
	v := reflect.New(reflect.TypeFor[int]()).Elem()
	ok := setExampleFromTag(v, reflect.TypeFor[int](), "42")
	assert.True(t, ok)
	assert.Equal(t, int64(42), v.Int())
}

func TestSetExampleFromTag_Float(t *testing.T) {
	v := reflect.New(reflect.TypeFor[float64]()).Elem()
	ok := setExampleFromTag(v, reflect.TypeFor[float64](), "3.14")
	assert.True(t, ok)
	assert.InDelta(t, 3.14, v.Float(), 0.001)
}

func TestSetExampleFromTag_Bool(t *testing.T) {
	v := reflect.New(reflect.TypeFor[bool]()).Elem()
	ok := setExampleFromTag(v, reflect.TypeFor[bool](), "true")
	assert.True(t, ok)
	assert.True(t, v.Bool())
}

func TestSetExampleFromTag_Time(t *testing.T) {
	v := reflect.New(reflect.TypeFor[time.Time]()).Elem()
	ok := setExampleFromTag(v, timeType, "2024-01-15T10:30:00Z")
	assert.True(t, ok)
	assert.Equal(t, 2024, v.Interface().(time.Time).Year())
}

func TestSetExampleFromTag_TimeInvalid(t *testing.T) {
	v := reflect.New(reflect.TypeFor[time.Time]()).Elem()
	ok := setExampleFromTag(v, timeType, "not-a-time")
	assert.False(t, ok)
}

func TestSetExampleFromTag_StringSlice(t *testing.T) {
	v := reflect.New(reflect.TypeFor[[]string]()).Elem()
	ok := setExampleFromTag(v, reflect.TypeFor[[]string](), "a, b, c")
	assert.True(t, ok)
	assert.Equal(t, []string{"a", "b", "c"}, v.Interface())
}

func TestSetExampleFromTag_IntSlice(t *testing.T) {
	v := reflect.New(reflect.TypeFor[[]int]()).Elem()
	ok := setExampleFromTag(v, reflect.TypeFor[[]int](), "1,2,3")
	assert.False(t, ok)
}

func TestSetExampleFromTag_InvalidScalar(t *testing.T) {
	v := reflect.New(reflect.TypeFor[int]()).Elem()
	ok := setExampleFromTag(v, reflect.TypeFor[int](), "notanumber")
	assert.False(t, ok)
}

func TestBuildExampleFromType_PointerField(t *testing.T) {
	type Inner struct {
		Val string `json:"val" example:"inner_val"`
	}
	type Outer struct {
		Ptr *Inner
	}
	result := buildExampleFromType(reflect.TypeFor[Outer]())
	require.NotNil(t, result)
	outer := result.(*Outer)
	require.NotNil(t, outer.Ptr)
	assert.Equal(t, "inner_val", outer.Ptr.Val)
}

func TestBuildExampleFromType_PointerToPointer(t *testing.T) {
	type Item struct {
		Name string `json:"name" example:"hello"`
	}
	result := buildExampleFromType(reflect.PointerTo(reflect.TypeFor[Item]()))
	require.NotNil(t, result)
}

func TestBuildExampleValue_PointerSlice(t *testing.T) {
	type Item struct {
		Name string `json:"name" example:"elem"`
	}
	v := reflect.New(reflect.TypeFor[[]*Item]()).Elem()
	ok := buildExampleValue(v)
	assert.True(t, ok)
	assert.Equal(t, 1, v.Len())
}

func TestBuildExampleValue_PointerMap(t *testing.T) {
	type Item struct {
		Name string `json:"name" example:"val"`
	}
	v := reflect.New(reflect.TypeFor[map[string]*Item]()).Elem()
	ok := buildExampleValue(v)
	assert.True(t, ok)
	assert.Equal(t, 1, v.Len())
}

func TestSetExampleFromTag_TimeDateOnly(t *testing.T) {
	v := reflect.New(reflect.TypeFor[time.Time]()).Elem()
	ok := setExampleFromTag(v, timeType, "2024-01-15")
	assert.True(t, ok)
	assert.Equal(t, 2024, v.Interface().(time.Time).Year())
	assert.Equal(t, time.January, v.Interface().(time.Time).Month())
	assert.Equal(t, 15, v.Interface().(time.Time).Day())
}

func TestFindInType_Struct(t *testing.T) {
	type Inner struct {
		X string `custom:"yes"`
	}
	type Outer struct {
		Inner
		Y int
	}
	result := findInType(reflect.TypeFor[Outer](), nil, func(sf reflect.StructField, path []int) string {
		if v := sf.Tag.Get("custom"); v != "" {
			return v
		}
		return ""
	}, true)

	assert.NotEmpty(t, result.Paths)
	assert.Equal(t, "yes", result.Paths[0].Value)
}

func TestFindInType_OnType(t *testing.T) {
	result := findInType(reflect.TypeFor[testInput](), func(t reflect.Type, path []int) bool {
		tp := reflect.PointerTo(t)
		return tp.Implements(resolverType)
	}, nil, true)
	assert.NotEmpty(t, result.Paths)
}

func TestEvery(t *testing.T) {
	type S struct {
		Name string `default:"x"`
	}
	fr := &FindResult[string]{
		Paths: []FindResultPath[string]{
			{Path: []int{0}, Value: "default_val"},
		},
	}

	v := reflect.New(reflect.TypeFor[S]()).Elem()
	visited := false
	fr.Every(v, func(item reflect.Value, val string) {
		visited = true
		assert.Equal(t, "default_val", val)
	})
	assert.True(t, visited)
}

func TestEvery_NilPointer(t *testing.T) {
	type S struct {
		Ptr *string
	}
	fr := &FindResult[string]{
		Paths: []FindResultPath[string]{
			{Path: []int{0, 0}, Value: "val"},
		},
	}
	v := reflect.New(reflect.TypeFor[S]()).Elem()
	visited := false
	fr.Every(v, func(item reflect.Value, val string) {
		visited = true
	})
	assert.False(t, visited)
}

func TestEveryPB(t *testing.T) {
	type S struct {
		Name string `query:"q"`
	}
	fr := &FindResult[string]{
		Paths: []FindResultPath[string]{
			{Path: []int{0}, Value: "val"},
		},
	}
	v := reflect.New(reflect.TypeFor[S]()).Elem()
	pb := core.NewPathBuffer(nil, 0)
	visited := false
	fr.EveryPB(pb, v, func(item reflect.Value, val string) {
		visited = true
		assert.Equal(t, "val", val)
	})
	assert.True(t, visited)
}

func TestEvery_Slice(t *testing.T) {
	type Item struct {
		Name string
	}
	type S struct {
		Items []Item
	}
	fr := &FindResult[string]{
		Paths: []FindResultPath[string]{
			{Path: []int{0, 0}, Value: "found"},
		},
	}
	v := reflect.New(reflect.TypeFor[S]()).Elem()
	items := reflect.MakeSlice(reflect.TypeFor[[]Item](), 2, 2)
	items.Index(0).FieldByName("Name").SetString("a")
	items.Index(1).FieldByName("Name").SetString("b")
	v.Field(0).Set(items)

	count := 0
	fr.Every(v, func(item reflect.Value, val string) {
		count++
	})
	assert.Equal(t, 2, count)
}

func TestWriteHeader_String(t *testing.T) {
	var name, value string
	write := func(n, v string) { name = n; value = v }
	info := &HeaderInfo{Name: "X-Custom"}
	f := reflect.ValueOf("hello")
	WriteHeader(write, info, f)
	assert.Equal(t, "X-Custom", name)
	assert.Equal(t, "hello", value)
}

func TestWriteHeader_EmptyString(t *testing.T) {
	called := false
	write := func(n, v string) { called = true }
	info := &HeaderInfo{Name: "X-Custom"}
	f := reflect.ValueOf("")
	WriteHeader(write, info, f)
	assert.False(t, called)
}

func TestWriteHeader_Int(t *testing.T) {
	var value string
	write := func(n, v string) { value = v }
	info := &HeaderInfo{Name: "X-Count"}
	f := reflect.ValueOf(42)
	WriteHeader(write, info, f)
	assert.Equal(t, "42", value)
}

func TestWriteHeader_Uint(t *testing.T) {
	var value string
	write := func(n, v string) { value = v }
	info := &HeaderInfo{Name: "X-Count"}
	f := reflect.ValueOf(uint(100))
	WriteHeader(write, info, f)
	assert.Equal(t, "100", value)
}

func TestWriteHeader_Float(t *testing.T) {
	var value string
	write := func(n, v string) { value = v }
	info := &HeaderInfo{Name: "X-Rate"}
	f := reflect.ValueOf(3.14)
	WriteHeader(write, info, f)
	assert.Equal(t, "3.14", value)
}

func TestWriteHeader_Bool(t *testing.T) {
	var value string
	write := func(n, v string) { value = v }
	info := &HeaderInfo{Name: "X-Flag"}
	f := reflect.ValueOf(true)
	WriteHeader(write, info, f)
	assert.Equal(t, "true", value)
}

func TestWriteHeader_Time(t *testing.T) {
	var value string
	write := func(n, v string) { value = v }
	info := &HeaderInfo{Name: "Last-Modified", TimeFormat: http.TimeFormat}
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	v := reflect.New(reflect.TypeFor[time.Time]()).Elem()
	v.Set(reflect.ValueOf(now))
	WriteHeader(write, info, v)
	assert.Contains(t, value, "2024")
}

func TestWriteHeader_Stringer(t *testing.T) {
	var value string
	write := func(n, v string) { value = v }
	info := &HeaderInfo{Name: "X-URL"}
	u := url.URL{Scheme: "https", Host: "example.com"}
	v := reflect.New(reflect.TypeFor[url.URL]()).Elem()
	v.Set(reflect.ValueOf(u))
	WriteHeader(write, info, v)
	assert.Equal(t, "https://example.com", value)
}

func TestWriteResponse_Success(t *testing.T) {
	api := &mockAPI{}
	ctx := newMockCtx("GET", "/")
	body := map[string]string{"msg": "ok"}

	err := WriteResponse(api, ctx, http.StatusOK, "application/json", body)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, ctx.status)
}

func TestFindHeaders(t *testing.T) {
	result := findHeaders[testOutput]()
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Paths)

	names := make(map[string]bool)
	for _, p := range result.Paths {
		names[p.Value.Name] = true
	}
	assert.True(t, names["ETag"])
	assert.True(t, names["Last-Modified"])
}

func TestFindHeaders_NoHeaders(t *testing.T) {
	type NoHeaders struct {
		Status int
		Body   string
	}
	result := findHeaders[NoHeaders]()
	assert.Empty(t, result.Paths)
}

func TestGetParamValue_Path(t *testing.T) {
	ctx := newMockCtx("GET", "/items/{id}")
	ctx.params["id"] = "42"
	p := ParamFieldInfo{Name: "id", Loc: "path"}
	v := GetParamValue(p, ctx, nil)
	assert.Equal(t, "42", v)
}

func TestGetParamValue_Query(t *testing.T) {
	ctx := newMockCtx("GET", "/items")
	ctx.queries["search"] = "hello"
	p := ParamFieldInfo{Name: "search", Loc: "query"}
	v := GetParamValue(p, ctx, nil)
	assert.Equal(t, "hello", v)
}

func TestGetParamValue_Header(t *testing.T) {
	ctx := newMockCtx("GET", "/")
	ctx.headers["Authorization"] = "Bearer token"
	p := ParamFieldInfo{Name: "Authorization", Loc: "header"}
	v := GetParamValue(p, ctx, nil)
	assert.Equal(t, "Bearer token", v)
}

func TestGetParamValue_Cookie(t *testing.T) {
	cookies := map[string]*http.Cookie{
		"session": {Name: "session", Value: "abc123"},
	}
	ctx := newMockCtx("GET", "/")
	p := ParamFieldInfo{Name: "session", Loc: "cookie"}
	v := GetParamValue(p, ctx, cookies)
	assert.Equal(t, "abc123", v)
}

func TestGetParamValue_Default(t *testing.T) {
	ctx := newMockCtx("GET", "/")
	p := ParamFieldInfo{Name: "missing", Loc: "query", Default: "fallback"}
	v := GetParamValue(p, ctx, nil)
	assert.Equal(t, "fallback", v)
}

func TestFindParams(t *testing.T) {
	reg := newRegistry()
	op := &core.Operation{Method: "POST", Path: "/items/{id}", OperationID: "createItem"}
	result := findParams[testInput](reg, op, false)

	require.NotNil(t, result)
	names := make(map[string]string)
	for _, p := range result.Paths {
		names[p.Value.Name] = p.Value.Loc
	}
	assert.Equal(t, "path", names["id"])
	assert.Equal(t, "query", names["search"])
	assert.Equal(t, "header", names["Authorization"])
	assert.Equal(t, "cookie", names["session"])
}

func TestBoolTag(t *testing.T) {
	type S struct {
		A string `flag:"true"`
		B string `flag:"false"`
		C string
	}
	st := reflect.TypeFor[S]()
	fa, _ := st.FieldByName("A")
	fb, _ := st.FieldByName("B")
	fc, _ := st.FieldByName("C")

	assert.True(t, boolTag(fa, "flag", false))
	assert.False(t, boolTag(fb, "flag", true))
	assert.True(t, boolTag(fc, "flag", true))
	assert.False(t, boolTag(fc, "flag", false))
}

func TestBoolTag_InvalidPanics(t *testing.T) {
	type S struct {
		X string `flag:"maybe"`
	}
	st := reflect.TypeFor[S]()
	fx, _ := st.FieldByName("X")
	assert.Panics(t, func() {
		boolTag(fx, "flag", false)
	})
}

func TestParseParamLocation_Path(t *testing.T) {
	f := reflect.StructField{
		Name: "ID",
		Type: reflect.TypeFor[int](),
		Tag:  `path:"id"`,
	}
	pl, ok := parseParamLocation(f, false)
	require.True(t, ok)
	assert.Equal(t, "path", pl.PFI.Loc)
	assert.Equal(t, "id", pl.PFI.Name)
	assert.True(t, pl.PFI.Required)
}

func TestParseParamLocation_Query(t *testing.T) {
	f := reflect.StructField{
		Name: "Search",
		Type: reflect.TypeFor[string](),
		Tag:  `query:"q"`,
	}
	pl, ok := parseParamLocation(f, false)
	require.True(t, ok)
	assert.Equal(t, "query", pl.PFI.Loc)
	assert.Equal(t, "q", pl.PFI.Name)
}

func TestParseParamLocation_QueryExplode(t *testing.T) {
	f := reflect.StructField{
		Name: "Tags",
		Type: reflect.TypeFor[[]string](),
		Tag:  `query:"tags,explode"`,
	}
	pl, ok := parseParamLocation(f, false)
	require.True(t, ok)
	assert.True(t, pl.PFI.Explode)
}

func TestParseParamLocation_QueryDeepObject(t *testing.T) {
	f := reflect.StructField{
		Name: "Filter",
		Type: reflect.TypeFor[map[string]string](),
		Tag:  `query:"filter,deepObject"`,
	}
	pl, ok := parseParamLocation(f, false)
	require.True(t, ok)
	assert.Equal(t, "deepObject", pl.PFI.Style)
}

func TestParseParamLocation_Header(t *testing.T) {
	f := reflect.StructField{
		Name: "Auth",
		Type: reflect.TypeFor[string](),
		Tag:  `header:"Authorization"`,
	}
	pl, ok := parseParamLocation(f, false)
	require.True(t, ok)
	assert.Equal(t, "header", pl.PFI.Loc)
	assert.Equal(t, "Authorization", pl.PFI.Name)
}

func TestParseParamLocation_Cookie(t *testing.T) {
	f := reflect.StructField{
		Name: "Session",
		Type: reflect.TypeFor[string](),
		Tag:  `cookie:"session"`,
	}
	pl, ok := parseParamLocation(f, false)
	require.True(t, ok)
	assert.Equal(t, "cookie", pl.PFI.Loc)
	assert.Equal(t, "session", pl.PFI.Name)
}

func TestParseParamLocation_Form(t *testing.T) {
	f := reflect.StructField{
		Name: "Field",
		Type: reflect.TypeFor[string](),
		Tag:  `form:"field"`,
	}
	pl, ok := parseParamLocation(f, false)
	require.True(t, ok)
	assert.Equal(t, "form", pl.PFI.Loc)
	assert.True(t, pl.PFI.Required)
}

func TestParseParamLocation_FormOptionalByDefault(t *testing.T) {
	f := reflect.StructField{
		Name: "Field",
		Type: reflect.TypeFor[string](),
		Tag:  `form:"field"`,
	}
	pl, ok := parseParamLocation(f, true)
	require.True(t, ok)
	assert.False(t, pl.PFI.Required)
}

func TestParseParamLocation_NoTag(t *testing.T) {
	f := reflect.StructField{
		Name: "Plain",
		Type: reflect.TypeFor[string](),
	}
	_, ok := parseParamLocation(f, false)
	assert.False(t, ok)
}

func TestParseParamLocation_Default(t *testing.T) {
	f := reflect.StructField{
		Name: "Search",
		Type: reflect.TypeFor[string](),
		Tag:  `query:"q" default:"all"`,
	}
	pl, ok := parseParamLocation(f, false)
	require.True(t, ok)
	assert.Equal(t, "all", pl.PFI.Default)
}

func TestParseParamLocation_PointerQuery(t *testing.T) {
	f := reflect.StructField{
		Name: "Limit",
		Type: reflect.PointerTo(reflect.TypeFor[int]()),
		Tag:  `query:"limit"`,
	}
	pl, ok := parseParamLocation(f, false)
	require.True(t, ok)
	assert.True(t, pl.PFI.IsPointer)
	assert.Equal(t, reflect.TypeFor[int](), pl.PFI.Type)
}

func TestParseParamLocation_PointerPathPanics(t *testing.T) {
	f := reflect.StructField{
		Name: "ID",
		Type: reflect.PointerTo(reflect.TypeFor[int]()),
		Tag:  `path:"id"`,
	}
	assert.Panics(t, func() {
		parseParamLocation(f, false)
	})
}

func TestParseParamLocation_CookieHttpCookie(t *testing.T) {
	f := reflect.StructField{
		Name: "Session",
		Type: reflect.TypeFor[http.Cookie](),
		Tag:  `cookie:"session"`,
	}
	pl, ok := parseParamLocation(f, false)
	require.True(t, ok)
	assert.Equal(t, stringType, pl.PFI.Type)
}

func TestParseInto_String(t *testing.T) {
	ctx := newMockCtx("GET", "/")
	f := reflect.New(reflect.TypeFor[string]()).Elem()
	p := ParamFieldInfo{Type: reflect.TypeFor[string]()}
	v, err := ParseInto(ctx, f, "hello", nil, p)
	require.NoError(t, err)
	assert.Equal(t, "hello", v)
	assert.Equal(t, "hello", f.String())
}

func TestParseInto_Int(t *testing.T) {
	ctx := newMockCtx("GET", "/")
	f := reflect.New(reflect.TypeFor[int]()).Elem()
	p := ParamFieldInfo{Type: reflect.TypeFor[int]()}
	v, err := ParseInto(ctx, f, "42", nil, p)
	require.NoError(t, err)
	assert.Equal(t, int64(42), v)
}

func TestParseInto_IntInvalid(t *testing.T) {
	ctx := newMockCtx("GET", "/")
	f := reflect.New(reflect.TypeFor[int]()).Elem()
	p := ParamFieldInfo{Type: reflect.TypeFor[int]()}
	_, err := ParseInto(ctx, f, "abc", nil, p)
	assert.Error(t, err)
}

func TestParseInto_Int8(t *testing.T) {
	ctx := newMockCtx("GET", "/")
	f := reflect.New(reflect.TypeFor[int8]()).Elem()
	p := ParamFieldInfo{Type: reflect.TypeFor[int8]()}
	v, err := ParseInto(ctx, f, "100", nil, p)
	require.NoError(t, err)
	assert.Equal(t, int64(100), v)
}

func TestParseInto_Uint(t *testing.T) {
	ctx := newMockCtx("GET", "/")
	f := reflect.New(reflect.TypeFor[uint]()).Elem()
	p := ParamFieldInfo{Type: reflect.TypeFor[uint]()}
	v, err := ParseInto(ctx, f, "99", nil, p)
	require.NoError(t, err)
	assert.Equal(t, uint64(99), v)
}

func TestParseInto_UintInvalid(t *testing.T) {
	ctx := newMockCtx("GET", "/")
	f := reflect.New(reflect.TypeFor[uint]()).Elem()
	p := ParamFieldInfo{Type: reflect.TypeFor[uint]()}
	_, err := ParseInto(ctx, f, "-1", nil, p)
	assert.Error(t, err)
}

func TestParseInto_Float64(t *testing.T) {
	ctx := newMockCtx("GET", "/")
	f := reflect.New(reflect.TypeFor[float64]()).Elem()
	p := ParamFieldInfo{Type: reflect.TypeFor[float64]()}
	v, err := ParseInto(ctx, f, "3.14", nil, p)
	require.NoError(t, err)
	assert.InDelta(t, 3.14, v, 0.001)
}

func TestParseInto_Float32(t *testing.T) {
	ctx := newMockCtx("GET", "/")
	f := reflect.New(reflect.TypeFor[float32]()).Elem()
	p := ParamFieldInfo{Type: reflect.TypeFor[float32]()}
	v, err := ParseInto(ctx, f, "1.5", nil, p)
	require.NoError(t, err)
	assert.InDelta(t, 1.5, v, 0.001)
}

func TestParseInto_FloatInvalid(t *testing.T) {
	ctx := newMockCtx("GET", "/")
	f := reflect.New(reflect.TypeFor[float64]()).Elem()
	p := ParamFieldInfo{Type: reflect.TypeFor[float64]()}
	_, err := ParseInto(ctx, f, "notafloat", nil, p)
	assert.Error(t, err)
}

func TestParseInto_Bool(t *testing.T) {
	ctx := newMockCtx("GET", "/")
	f := reflect.New(reflect.TypeFor[bool]()).Elem()
	p := ParamFieldInfo{Type: reflect.TypeFor[bool]()}
	v, err := ParseInto(ctx, f, "true", nil, p)
	require.NoError(t, err)
	assert.Equal(t, true, v)
}

func TestParseInto_BoolInvalid(t *testing.T) {
	ctx := newMockCtx("GET", "/")
	f := reflect.New(reflect.TypeFor[bool]()).Elem()
	p := ParamFieldInfo{Type: reflect.TypeFor[bool]()}
	_, err := ParseInto(ctx, f, "maybe", nil, p)
	assert.Error(t, err)
}

func TestParseInto_Time(t *testing.T) {
	ctx := newMockCtx("GET", "/")
	f := reflect.New(reflect.TypeFor[time.Time]()).Elem()
	p := ParamFieldInfo{Type: timeType, TimeFormat: time.RFC3339}
	v, err := ParseInto(ctx, f, "2024-01-15T10:30:00Z", nil, p)
	require.NoError(t, err)
	assert.NotNil(t, v)
	assert.Equal(t, 2024, f.Interface().(time.Time).Year())
}

func TestParseInto_TimeInvalid(t *testing.T) {
	ctx := newMockCtx("GET", "/")
	f := reflect.New(reflect.TypeFor[time.Time]()).Elem()
	p := ParamFieldInfo{Type: timeType, TimeFormat: time.RFC3339}
	_, err := ParseInto(ctx, f, "nope", nil, p)
	assert.Error(t, err)
}

func TestParseInto_URL(t *testing.T) {
	ctx := newMockCtx("GET", "/")
	f := reflect.New(reflect.TypeFor[url.URL]()).Elem()
	p := ParamFieldInfo{Type: urlType}
	v, err := ParseInto(ctx, f, "https://example.com/path", nil, p)
	require.NoError(t, err)
	assert.NotNil(t, v)
	assert.Equal(t, "example.com", f.Interface().(url.URL).Host)
}

func TestParseInto_SlicePreSplit(t *testing.T) {
	ctx := newMockCtx("GET", "/")
	f := reflect.New(reflect.TypeFor[[]string]()).Elem()
	p := ParamFieldInfo{Type: reflect.TypeFor[[]string]()}
	v, err := ParseInto(ctx, f, "", []string{"a", "b", "c"}, p)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, v)
}

func TestParseInto_SliceCommaSeparated(t *testing.T) {
	ctx := newMockCtx("GET", "/")
	f := reflect.New(reflect.TypeFor[[]string]()).Elem()
	p := ParamFieldInfo{Type: reflect.TypeFor[[]string](), Name: "tags"}
	v, err := ParseInto(ctx, f, "a,b,c", nil, p)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, v)
}

func TestParseInto_SliceExplode(t *testing.T) {
	ctx := newMockCtx("GET", "/")
	ctx.urlVal = url.URL{RawQuery: "tags=a&tags=b"}
	f := reflect.New(reflect.TypeFor[[]string]()).Elem()
	p := ParamFieldInfo{Type: reflect.TypeFor[[]string](), Name: "tags", Explode: true}
	v, err := ParseInto(ctx, f, "a", nil, p)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, v)
}

func TestParseInto_SliceExplodeParsedQuery(t *testing.T) {
	ctx := newMockCtx("GET", "/")
	f := reflect.New(reflect.TypeFor[[]string]()).Elem()
	p := ParamFieldInfo{Type: reflect.TypeFor[[]string](), Name: "tags", Explode: true}
	pq := url.Values{"tags": {"x", "y"}}
	v, err := ParseInto(ctx, f, "x", nil, p, pq)
	require.NoError(t, err)
	assert.Equal(t, []string{"x", "y"}, v)
}

func TestParseInto_Unsupported(t *testing.T) {
	ctx := newMockCtx("GET", "/")
	f := reflect.New(reflect.TypeFor[complex128]()).Elem()
	p := ParamFieldInfo{Type: reflect.TypeFor[complex128]()}
	_, err := ParseInto(ctx, f, "1+2i", nil, p)
	assert.Error(t, err)
}

type textUnmarshalerType struct {
	Val string
}

func (t *textUnmarshalerType) UnmarshalText(b []byte) error {
	t.Val = string(b)
	return nil
}

func TestParseInto_TextUnmarshaler(t *testing.T) {
	ctx := newMockCtx("GET", "/")
	f := reflect.New(reflect.TypeFor[textUnmarshalerType]()).Elem()
	p := ParamFieldInfo{Type: reflect.TypeFor[textUnmarshalerType]()}
	v, err := ParseInto(ctx, f, "custom_value", nil, p)
	require.NoError(t, err)
	assert.Equal(t, "custom_value", v)
	assert.Equal(t, "custom_value", f.Interface().(textUnmarshalerType).Val)
}

func TestParseScalar_String(t *testing.T) {
	f := reflect.New(reflect.TypeFor[string]()).Elem()
	err := parseScalar(f, "hello")
	require.NoError(t, err)
	assert.Equal(t, "hello", f.String())
}

func TestParseScalar_Interface(t *testing.T) {
	f := reflect.New(reflect.TypeFor[any]()).Elem()
	err := parseScalar(f, "hello")
	require.NoError(t, err)
	assert.Equal(t, "hello", f.Interface())
}

func TestParseScalar_Int(t *testing.T) {
	for _, kind := range []reflect.Type{
		reflect.TypeFor[int](),
		reflect.TypeFor[int8](),
		reflect.TypeFor[int16](),
		reflect.TypeFor[int32](),
		reflect.TypeFor[int64](),
	} {
		t.Run(kind.String(), func(t *testing.T) {
			fv := reflect.New(kind).Elem()
			require.NoError(t, parseScalar(fv, "10"))
			assert.Equal(t, int64(10), fv.Int())
		})
	}
}

func TestParseScalar_Uint(t *testing.T) {
	for _, kind := range []reflect.Type{
		reflect.TypeFor[uint](),
		reflect.TypeFor[uint8](),
		reflect.TypeFor[uint16](),
		reflect.TypeFor[uint32](),
		reflect.TypeFor[uint64](),
	} {
		t.Run(kind.String(), func(t *testing.T) {
			fv := reflect.New(kind).Elem()
			require.NoError(t, parseScalar(fv, "10"))
			assert.Equal(t, uint64(10), fv.Uint())
		})
	}
}

func TestParseScalar_Float(t *testing.T) {
	for _, kind := range []reflect.Type{
		reflect.TypeFor[float32](),
		reflect.TypeFor[float64](),
	} {
		t.Run(kind.String(), func(t *testing.T) {
			fv := reflect.New(kind).Elem()
			require.NoError(t, parseScalar(fv, "1.5"))
			assert.InDelta(t, 1.5, fv.Float(), 0.001)
		})
	}
}

func TestParseScalar_Bool(t *testing.T) {
	fv := reflect.New(reflect.TypeFor[bool]()).Elem()
	require.NoError(t, parseScalar(fv, "true"))
	assert.True(t, fv.Bool())
}

func TestParseScalar_IntInvalid(t *testing.T) {
	fv := reflect.New(reflect.TypeFor[int]()).Elem()
	assert.Error(t, parseScalar(fv, "abc"))
}

func TestParseScalar_UintInvalid(t *testing.T) {
	fv := reflect.New(reflect.TypeFor[uint]()).Elem()
	assert.Error(t, parseScalar(fv, "abc"))
}

func TestParseScalar_FloatInvalid(t *testing.T) {
	fv := reflect.New(reflect.TypeFor[float64]()).Elem()
	assert.Error(t, parseScalar(fv, "abc"))
}

func TestParseScalar_BoolInvalid(t *testing.T) {
	fv := reflect.New(reflect.TypeFor[bool]()).Elem()
	assert.Error(t, parseScalar(fv, "maybe"))
}

func TestParseScalar_Unsupported(t *testing.T) {
	fv := reflect.New(reflect.TypeFor[complex128]()).Elem()
	err := parseScalar(fv, "1+2i")
	assert.ErrorIs(t, err, ErrUnparsable)
}

func TestSetFieldValue_String(t *testing.T) {
	f := reflect.New(reflect.TypeFor[string]()).Elem()
	require.NoError(t, setFieldValue(f, "hello"))
	assert.Equal(t, "hello", f.String())
}

func TestSetFieldValue_Int(t *testing.T) {
	f := reflect.New(reflect.TypeFor[int]()).Elem()
	require.NoError(t, setFieldValue(f, "42"))
	assert.Equal(t, int64(42), f.Int())
}

func TestSetFieldValue_Unsupported(t *testing.T) {
	f := reflect.New(reflect.TypeFor[complex128]()).Elem()
	err := setFieldValue(f, "1+2i")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported type")
}

func TestParseSliceInto_IntSlice(t *testing.T) {
	f := reflect.New(reflect.TypeFor[[]int]()).Elem()
	v, err := parseSliceInto(f, []string{"1", "2", "3"})
	require.NoError(t, err)
	assert.NotNil(t, v)
	assert.Equal(t, 3, f.Len())
	assert.Equal(t, int64(1), f.Index(0).Int())
}

func TestParseSliceInto_StringSlice(t *testing.T) {
	f := reflect.New(reflect.TypeFor[[]string]()).Elem()
	v, err := parseSliceInto(f, []string{"a", "b"})
	require.NoError(t, err)
	assert.NotNil(t, v)
	assert.Equal(t, 2, f.Len())
}

func TestParseSliceInto_BoolSlice(t *testing.T) {
	f := reflect.New(reflect.TypeFor[[]bool]()).Elem()
	v, err := parseSliceInto(f, []string{"true", "false"})
	require.NoError(t, err)
	assert.NotNil(t, v)
	assert.Equal(t, 2, f.Len())
	assert.True(t, f.Index(0).Bool())
	assert.False(t, f.Index(1).Bool())
}

func TestParseSliceInto_Float64Slice(t *testing.T) {
	f := reflect.New(reflect.TypeFor[[]float64]()).Elem()
	v, err := parseSliceInto(f, []string{"1.1", "2.2"})
	require.NoError(t, err)
	assert.NotNil(t, v)
	assert.Equal(t, 2, f.Len())
}

func TestParseSliceInto_UintSlice(t *testing.T) {
	f := reflect.New(reflect.TypeFor[[]uint]()).Elem()
	v, err := parseSliceInto(f, []string{"1", "2"})
	require.NoError(t, err)
	assert.NotNil(t, v)
	assert.Equal(t, 2, f.Len())
}

func TestParseSliceInto_InvalidInt(t *testing.T) {
	f := reflect.New(reflect.TypeFor[[]int]()).Elem()
	_, err := parseSliceInto(f, []string{"abc"})
	assert.Error(t, err)
}

func TestParseSliceInto_InvalidBool(t *testing.T) {
	f := reflect.New(reflect.TypeFor[[]bool]()).Elem()
	_, err := parseSliceInto(f, []string{"maybe"})
	assert.Error(t, err)
}

func TestParseSliceInto_InvalidFloat(t *testing.T) {
	f := reflect.New(reflect.TypeFor[[]float64]()).Elem()
	_, err := parseSliceInto(f, []string{"abc"})
	assert.Error(t, err)
}

func TestParseSliceInto_InvalidUint(t *testing.T) {
	f := reflect.New(reflect.TypeFor[[]uint]()).Elem()
	_, err := parseSliceInto(f, []string{"-1"})
	assert.Error(t, err)
}

func TestParseSliceInto_UnsupportedElem(t *testing.T) {
	f := reflect.New(reflect.TypeFor[[]complex128]()).Elem()
	_, err := parseSliceInto(f, []string{"1"})
	assert.ErrorIs(t, err, ErrUnparsable)
}

func TestParseSliceInto_StringSubtype(t *testing.T) {
	type MyEnum string
	f := reflect.New(reflect.TypeFor[[]MyEnum]()).Elem()
	v, err := parseSliceInto(f, []string{"a", "b"})
	require.NoError(t, err)
	assert.NotNil(t, v)
	assert.Equal(t, 2, f.Len())
}

func TestParseSliceInto_Int8Slice(t *testing.T) {
	f := reflect.New(reflect.TypeFor[[]int8]()).Elem()
	v, err := parseSliceInto(f, []string{"1", "2"})
	require.NoError(t, err)
	assert.NotNil(t, v)
	assert.Equal(t, 2, f.Len())
}

func TestParseSliceInto_Uint8Slice(t *testing.T) {
	f := reflect.New(reflect.TypeFor[[]uint8]()).Elem()
	v, err := parseSliceInto(f, []string{"1", "2"})
	require.NoError(t, err)
	assert.NotNil(t, v)
	assert.Equal(t, 2, f.Len())
}

func TestParseSliceInto_Float32Slice(t *testing.T) {
	f := reflect.New(reflect.TypeFor[[]float32]()).Elem()
	v, err := parseSliceInto(f, []string{"1.1", "2.2"})
	require.NoError(t, err)
	assert.NotNil(t, v)
	assert.Equal(t, 2, f.Len())
}

func TestContextError_Error(t *testing.T) {
	e := &ContextError{Code: 400, Msg: "bad request"}
	assert.Equal(t, "bad request", e.Error())
}

func TestAnalyzeInput_WithDefaultStatus(t *testing.T) {
	reg := newRegistry()
	op := &core.Operation{Method: "GET", Path: "/items", OperationID: "getItems", DefaultStatus: http.StatusOK}
	meta := AnalyzeOutput[testOutput](op, reg)
	assert.Equal(t, http.StatusOK, op.DefaultStatus)
	assert.NotNil(t, meta)
}

func TestWriteResponse_NoContentType(t *testing.T) {
	api := &mockAPI{}
	ctx := newMockCtx("GET", "/")
	body := map[string]string{"msg": "ok"}

	err := WriteResponse(api, ctx, http.StatusOK, "", body)
	require.NoError(t, err)
	assert.Equal(t, "application/json", ctx.respHdr["Content-Type"])
}

func TestEveryPB_Slice(t *testing.T) {
	type Item struct {
		Name string `json:"name"`
	}
	type S struct {
		Items []Item `json:"items"`
	}
	fr := &FindResult[string]{
		Paths: []FindResultPath[string]{
			{Path: []int{0, 0}, Value: "found"},
		},
	}
	v := reflect.New(reflect.TypeFor[S]()).Elem()
	items := reflect.MakeSlice(reflect.TypeFor[[]Item](), 2, 2)
	items.Index(0).FieldByName("Name").SetString("a")
	items.Index(1).FieldByName("Name").SetString("b")
	v.Field(0).Set(items)

	pb := core.NewPathBuffer(nil, 0)
	count := 0
	fr.EveryPB(pb, v, func(item reflect.Value, val string) {
		count++
	})
	assert.Equal(t, 2, count)
}

func TestEveryPB_Map(t *testing.T) {
	type Item struct {
		Name string `json:"name"`
	}
	type S struct {
		Items map[string]Item `json:"items"`
	}
	fr := &FindResult[string]{
		Paths: []FindResultPath[string]{
			{Path: []int{0, 0}, Value: "found"},
		},
	}
	v := reflect.New(reflect.TypeFor[S]()).Elem()
	m := reflect.MakeMap(reflect.TypeFor[map[string]Item]())
	m.SetMapIndex(reflect.ValueOf("k1"), reflect.ValueOf(Item{Name: "a"}))
	v.Field(0).Set(m)

	pb := core.NewPathBuffer(nil, 0)
	count := 0
	fr.EveryPB(pb, v, func(item reflect.Value, val string) {
		count++
	})
	assert.Equal(t, 1, count)
}

func TestFindResolvers(t *testing.T) {
	result := findResolvers[testInput]()
	assert.NotEmpty(t, result.Paths)
}

func TestFindResolvers_NoResolver(t *testing.T) {
	type Plain struct {
		Name string
	}
	result := findResolvers[Plain]()
	assert.Empty(t, result.Paths)
}

func TestAnyFieldHasExample(t *testing.T) {
	type WithEx struct {
		Name string `example:"hi"`
	}
	assert.True(t, anyFieldHasExample(reflect.TypeFor[WithEx]()))

	type WithoutEx struct {
		Name string
	}
	assert.False(t, anyFieldHasExample(reflect.TypeFor[WithoutEx]()))

	assert.False(t, anyFieldHasExample(reflect.TypeFor[string]()))
}

func TestAnyFieldHasExample_Nested(t *testing.T) {
	type Inner struct {
		Val string `example:"val"`
	}
	type Outer struct {
		Inner Inner
	}
	assert.True(t, anyFieldHasExample(reflect.TypeFor[Outer]()))
}

func TestAnyFieldHasExample_Pointer(t *testing.T) {
	type Inner struct {
		Val string `example:"val"`
	}
	assert.True(t, anyFieldHasExample(reflect.PointerTo(reflect.TypeFor[Inner]())))
}

func TestBuildExampleValue_ExampleProvider(t *testing.T) {
	v := reflect.New(reflect.TypeFor[exampleProviderImpl]()).Elem()
	ok := buildExampleValue(v)
	assert.True(t, ok)
}

type exampleProviderImpl struct {
	Val string
}

func (e *exampleProviderImpl) Example() any {
	return &exampleProviderImpl{Val: "provided"}
}

func TestBuildExampleStruct_PointerFieldNoExample(t *testing.T) {
	type Inner struct {
		Name string
	}
	type Outer struct {
		Ptr *Inner
	}
	result := buildExampleFromType(reflect.TypeFor[Outer]())
	assert.Nil(t, result)
}

func TestBuildExampleStruct_PointerWithExampleTag(t *testing.T) {
	type Outer struct {
		Name *string `example:"pointed"`
	}
	result := buildExampleFromType(reflect.TypeFor[Outer]())
	require.NotNil(t, result)
	outer := result.(*Outer)
	require.NotNil(t, outer.Name)
	assert.Equal(t, "pointed", *outer.Name)
}

func TestBuildExampleMap_NonStructValue(t *testing.T) {
	v := reflect.New(reflect.TypeFor[map[string]string]()).Elem()
	ok := buildExampleMap(v, reflect.TypeFor[map[string]string]())
	assert.False(t, ok)
}

func TestBuildExampleSlice_NonStructElem(t *testing.T) {
	v := reflect.New(reflect.TypeFor[[]string]()).Elem()
	ok := buildExampleSlice(v, reflect.TypeFor[[]string]())
	assert.False(t, ok)
}

func TestBuildExampleValue_UnsettableValue(t *testing.T) {
	x := 42
	v := reflect.ValueOf(x)
	ok := buildExampleValue(v)
	assert.False(t, ok)
}

func TestWriteResponseWithPanic_Success(t *testing.T) {
	api := &mockAPI{}
	ctx := newMockCtx("GET", "/")
	body := map[string]string{"msg": "ok"}
	assert.NotPanics(t, func() {
		WriteResponseWithPanic(api, ctx, http.StatusOK, "application/json", body)
	})
}

func TestReadBody_CloserCalled(t *testing.T) {
	rc := io.NopCloser(strings.NewReader("data"))
	ctx := newMockCtx("POST", "/")
	ctx.body = rc

	var buf bytes.Buffer
	err := ReadBody(&buf, ctx, 1024)
	assert.Nil(t, err)
	assert.Equal(t, "data", buf.String())
}

func TestEvery_Map(t *testing.T) {
	type Item struct {
		Name string
	}
	type S struct {
		Items map[string]Item
	}
	fr := &FindResult[string]{
		Paths: []FindResultPath[string]{
			{Path: []int{0, 0}, Value: "found"},
		},
	}
	v := reflect.New(reflect.TypeFor[S]()).Elem()
	m := reflect.MakeMap(reflect.TypeFor[map[string]Item]())
	m.SetMapIndex(reflect.ValueOf("k1"), reflect.ValueOf(Item{Name: "a"}))
	v.Field(0).Set(m)

	count := 0
	fr.Every(v, func(item reflect.Value, val string) {
		count++
	})
	assert.Equal(t, 1, count)
}

func TestEveryPB_IntMapKey(t *testing.T) {
	type Item struct {
		Name string `json:"name"`
	}
	type S struct {
		Items map[int]Item `json:"items"`
	}
	fr := &FindResult[string]{
		Paths: []FindResultPath[string]{
			{Path: []int{0, 0}, Value: "found"},
		},
	}
	v := reflect.New(reflect.TypeFor[S]()).Elem()
	m := reflect.MakeMap(reflect.TypeFor[map[int]Item]())
	m.SetMapIndex(reflect.ValueOf(1), reflect.ValueOf(Item{Name: "a"}))
	v.Field(0).Set(m)

	pb := core.NewPathBuffer(nil, 0)
	count := 0
	fr.EveryPB(pb, v, func(item reflect.Value, val string) {
		count++
	})
	assert.Equal(t, 1, count)
}

func TestFindHeaders_WithTimeFormat(t *testing.T) {
	type Out struct {
		Status int
		Body   string
		Date   time.Time `header:"Date" timeFormat:"2006-01-02"`
	}
	result := findHeaders[Out]()
	require.NotEmpty(t, result.Paths)
	for _, p := range result.Paths {
		if p.Value.Name == "Date" {
			assert.Equal(t, "2006-01-02", p.Value.TimeFormat)
		}
	}
}

func TestWriteHeader_ZeroTime(t *testing.T) {
	var value string
	write := func(n, v string) { value = v }
	info := &HeaderInfo{Name: "Date", TimeFormat: http.TimeFormat}
	v := reflect.New(reflect.TypeFor[time.Time]()).Elem()
	WriteHeader(write, info, v)
	assert.NotEmpty(t, value)
}

func TestProcessRegularMsgBody_SkipValidation(t *testing.T) {
	type Body struct {
		Name string `json:"name"`
	}
	type Input struct {
		Body Body
	}
	v := reflect.New(reflect.TypeFor[Input]()).Elem()
	cfg := BodyProcessingConfig{
		Body:           []byte(`{"name":"test"}`),
		Op:             core.Operation{SkipValidateBody: true},
		Value:          v,
		HasInputBody:   true,
		InputBodyIndex: []int{0},
		Unmarshaler:    json.Unmarshal,
		Validator:      func(data any, res *core.ValidateResult) {},
		Defaults:       &FindResult[any]{},
		Result:         &core.ValidateResult{},
	}
	errStatus, cErr := ProcessRegularMsgBody(cfg)
	assert.Nil(t, cErr)
	assert.Equal(t, -1, errStatus)
	assert.Equal(t, "test", v.Field(0).FieldByName("Name").String())
}

func TestProcessRegularMsgBody_ValidationErrors(t *testing.T) {
	type Body struct {
		Name string `json:"name"`
	}
	type Input struct {
		Body Body
	}
	v := reflect.New(reflect.TypeFor[Input]()).Elem()
	cfg := BodyProcessingConfig{
		Body:           []byte(`{"name":"test"}`),
		Op:             core.Operation{},
		Value:          v,
		HasInputBody:   true,
		InputBodyIndex: []int{0},
		Unmarshaler:    json.Unmarshal,
		Validator: func(data any, res *core.ValidateResult) {
			pb := core.NewPathBuffer(nil, 0)
			pb.Push("body")
			pb.Push("name")
			res.Add(pb, "test", "name too short")
		},
		Defaults: &FindResult[any]{},
		Result:   &core.ValidateResult{},
	}
	errStatus, cErr := ProcessRegularMsgBody(cfg)
	assert.Nil(t, cErr)
	assert.Equal(t, http.StatusUnprocessableEntity, errStatus)
}

func TestParseInto_URLInvalid(t *testing.T) {
	ctx := newMockCtx("GET", "/")
	f := reflect.New(reflect.TypeFor[url.URL]()).Elem()
	p := ParamFieldInfo{Type: urlType}
	_, err := ParseInto(ctx, f, "://invalid", nil, p)
	assert.Error(t, err)
}

func TestJsonName(t *testing.T) {
	type S struct {
		Name    string `json:"custom_name"`
		NoTag   string
		WithOpt string `json:"opt,omitempty"`
	}
	st := reflect.TypeFor[S]()

	f1, _ := st.FieldByName("Name")
	assert.Equal(t, "custom_name", jsonName(f1))

	f2, _ := st.FieldByName("NoTag")
	assert.Equal(t, "notag", jsonName(f2))

	f3, _ := st.FieldByName("WithOpt")
	assert.Equal(t, "opt", jsonName(f3))
}

func TestGetHint(t *testing.T) {
	named := reflect.TypeFor[testOutput]()
	assert.Equal(t, "testOutputBody", getHint(named, "Body", "fallback"))

	anon := reflect.TypeFor[struct{ X int }]()
	assert.Equal(t, "fallback", getHint(anon, "X", "fallback"))
}

func TestFindInType_Slice(t *testing.T) {
	type Item struct {
		Tag string `marker:"yes"`
	}
	result := findInType(reflect.TypeFor[[]Item](), nil, func(sf reflect.StructField, path []int) string {
		if v := sf.Tag.Get("marker"); v != "" {
			return v
		}
		return ""
	}, true)
	assert.NotEmpty(t, result.Paths)
}

func TestFindInType_Map(t *testing.T) {
	type Item struct {
		Tag string `marker:"yes"`
	}
	result := findInType(reflect.TypeFor[map[string]Item](), nil, func(sf reflect.StructField, path []int) string {
		if v := sf.Tag.Get("marker"); v != "" {
			return v
		}
		return ""
	}, true)
	assert.NotEmpty(t, result.Paths)
}

func TestFindInType_Ignore(t *testing.T) {
	type S struct {
		A string `marker:"a"`
		B string `marker:"b"`
	}
	result := findInType(reflect.TypeFor[S](), nil, func(sf reflect.StructField, path []int) string {
		if v := sf.Tag.Get("marker"); v != "" {
			return v
		}
		return ""
	}, true, "A")
	names := make(map[string]bool)
	for _, p := range result.Paths {
		names[p.Value] = true
	}
	assert.False(t, names["a"])
	assert.True(t, names["b"])
}

func TestWriteHeader_Fallback(t *testing.T) {
	var value string
	write := func(n, v string) { value = v }
	info := &HeaderInfo{Name: "X-Custom"}
	type custom struct{ X int }
	cv := reflect.New(reflect.TypeFor[custom]()).Elem()
	cv.FieldByName("X").SetInt(99)
	WriteHeader(write, info, cv)
	assert.Contains(t, value, "99")
}

func TestParseInto_SliceIntCommaSep(t *testing.T) {
	ctx := newMockCtx("GET", "/")
	f := reflect.New(reflect.TypeFor[[]int]()).Elem()
	p := ParamFieldInfo{Type: reflect.TypeFor[[]int](), Name: "ids"}
	v, err := ParseInto(ctx, f, "1,2,3", nil, p)
	require.NoError(t, err)
	assert.NotNil(t, v)
	assert.Equal(t, 3, f.Len())
}

type mockCanceledContext struct {
	mockContext
	ctx context.Context
}

func (m *mockCanceledContext) Context() context.Context { return m.ctx }

type mockError struct {
	status int
	msg    string
}

func (e *mockError) StatusCode() int { return e.status }
func (e *mockError) Error() string   { return e.msg }

type mockContentTyperError struct {
	mockError
}

func (e *mockContentTyperError) ContentType(_ string) string {
	return "application/problem+json"
}

type mockErrorHandler struct {
	useContentTyper bool
}

func (h *mockErrorHandler) NewError(status int, msg string, errs ...error) core.Error {
	return h.NewErrorWithContext(nil, status, msg, errs...)
}
func (h *mockErrorHandler) NewErrorWithContext(_ core.Context, status int, msg string, _ ...error) core.Error {
	if h.useContentTyper {
		return &mockContentTyperError{mockError{status: status, msg: msg}}
	}
	return &mockError{status: status, msg: msg}
}
func (h *mockErrorHandler) ErrorSchema(_ core.Registry) *core.Schema { return nil }
func (h *mockErrorHandler) ErrorContentType(ct string) string        { return ct }

type mockAPIWithNegotiateError struct {
	mockAPI
}

func (m *mockAPIWithNegotiateError) Negotiate(_ string) (string, error) {
	return "", errors.New("unsupported accept type")
}

type mockAPIWithTransformError struct {
	mockAPI
}

func (m *mockAPIWithTransformError) Transform(_ core.Context, _ string, _ any) (any, error) {
	return nil, errors.New("transform failed")
}

type mockAPIWithMarshalError struct {
	mockAPI
}

func (m *mockAPIWithMarshalError) Marshal(_ io.Writer, _ string, _ any) error {
	return errors.New("marshal failed")
}

type contentTyperBody struct {
	Message string `json:"message"`
}

func (c *contentTyperBody) ContentType(_ string) string {
	return "application/vnd.custom+json"
}

func TestWriteResponse_NegotiateError(t *testing.T) {
	api := &mockAPIWithNegotiateError{
		mockAPI: mockAPI{errHandler: &mockErrorHandler{useContentTyper: false}},
	}
	ctx := newMockCtx("GET", "/")
	body := map[string]string{"msg": "ok"}

	err := WriteResponse(api, ctx, http.StatusOK, "", body)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported accept type")
	assert.Equal(t, http.StatusNotAcceptable, ctx.status)
	assert.Equal(t, "application/json", ctx.respHdr["Content-Type"])
}

func TestWriteResponse_NegotiateErrorWithContentTyper(t *testing.T) {
	api := &mockAPIWithNegotiateError{
		mockAPI: mockAPI{errHandler: &mockErrorHandler{useContentTyper: true}},
	}
	ctx := newMockCtx("GET", "/")
	body := map[string]string{"msg": "ok"}

	err := WriteResponse(api, ctx, http.StatusOK, "", body)
	require.Error(t, err)
	assert.Equal(t, http.StatusNotAcceptable, ctx.status)
	assert.Equal(t, "application/problem+json", ctx.respHdr["Content-Type"])
}

func TestWriteResponse_BodyWithContentTyper(t *testing.T) {
	api := &mockAPI{}
	ctx := newMockCtx("GET", "/")
	body := &contentTyperBody{Message: "hello"}

	err := WriteResponse(api, ctx, http.StatusOK, "", body)
	require.NoError(t, err)
	assert.Equal(t, "application/vnd.custom+json", ctx.respHdr["Content-Type"])
}

func TestWriteResponseWithPanic_ErrorPath(t *testing.T) {
	api := &mockAPIWithTransformError{
		mockAPI: mockAPI{},
	}
	ctx := newMockCtx("GET", "/")
	body := map[string]string{"msg": "ok"}

	assert.Panics(t, func() {
		WriteResponseWithPanic(api, ctx, http.StatusOK, "application/json", body)
	})
}

func TestTransformAndWrite_TransformError(t *testing.T) {
	api := &mockAPIWithTransformError{mockAPI: mockAPI{}}
	ctx := newMockCtx("GET", "/test")

	err := transformAndWrite(api, ctx, http.StatusOK, "application/json", "body")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error transforming response")
	assert.Contains(t, ctx.writer.String(), "error transforming response")
}

func TestTransformAndWrite_MarshalError(t *testing.T) {
	api := &mockAPIWithMarshalError{mockAPI: mockAPI{}}
	ctx := newMockCtx("GET", "/test")

	err := transformAndWrite(api, ctx, http.StatusOK, "application/json", "body")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error marshaling response")
	assert.Contains(t, ctx.writer.String(), "error marshaling response")
}

func TestTransformAndWrite_MarshalErrorCanceledContext(t *testing.T) {
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	api := &mockAPIWithMarshalError{mockAPI: mockAPI{}}
	inner := newMockCtx("GET", "/test")
	ctx := &mockCanceledContext{mockContext: *inner, ctx: cancelCtx}

	err := transformAndWrite(api, ctx, http.StatusOK, "application/json", "body")
	require.NoError(t, err)
	assert.Equal(t, 499, ctx.status)
}

func TestTransformAndWrite_NoContent(t *testing.T) {
	api := &mockAPI{}
	ctx := newMockCtx("GET", "/test")

	err := transformAndWrite(api, ctx, http.StatusNoContent, "application/json", "body")
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, ctx.status)
	assert.Empty(t, ctx.writer.String())
}

func TestTransformAndWrite_NotModified(t *testing.T) {
	api := &mockAPI{}
	ctx := newMockCtx("GET", "/test")

	err := transformAndWrite(api, ctx, http.StatusNotModified, "application/json", "body")
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotModified, ctx.status)
	assert.Empty(t, ctx.writer.String())
}

func TestTransformAndWrite_NonStandardStatus(t *testing.T) {
	api := &mockAPI{}
	ctx := newMockCtx("GET", "/test")

	err := transformAndWrite(api, ctx, 299, "application/json", map[string]string{"x": "y"})
	require.NoError(t, err)
	assert.Equal(t, 299, ctx.status)
}

func TestFindParams_HiddenField(t *testing.T) {
	type inputWithHidden struct {
		Secret string `query:"secret" hidden:"true"`
	}
	reg := newRegistry()
	op := &core.Operation{Method: "GET", Path: "/hidden", OperationID: "getHidden"}
	result := findParams[inputWithHidden](reg, op, false)

	require.NotNil(t, result)
	found := false
	for _, p := range result.Paths {
		if p.Value.Name == "secret" {
			found = true
			break
		}
	}
	assert.True(t, found, "hidden param should still be found in result")

	assert.NotEmpty(t, op.HiddenParameters)
	hiddenNames := make(map[string]bool)
	for _, hp := range op.HiddenParameters {
		hiddenNames[hp.Name] = true
	}
	assert.True(t, hiddenNames["secret"])

	for _, p := range op.Parameters {
		assert.NotEqual(t, "secret", p.Name, "hidden param should not appear in op.Parameters")
	}
}

func TestFindParams_HiddenFormField(t *testing.T) {
	type inputWithHiddenForm struct {
		Field string `form:"field" hidden:"true"`
	}
	reg := newRegistry()
	op := &core.Operation{Method: "POST", Path: "/form", OperationID: "postForm"}
	findParams[inputWithHiddenForm](reg, op, false)

	assert.Empty(t, op.HiddenParameters, "hidden form field should not go into HiddenParameters")
}

func TestFindParams_CookieParam(t *testing.T) {
	type inputWithCookie struct {
		Token string `cookie:"token"`
	}
	reg := newRegistry()
	op := &core.Operation{Method: "GET", Path: "/cookie", OperationID: "getCookie"}
	result := findParams[inputWithCookie](reg, op, false)

	found := false
	for _, p := range result.Paths {
		if p.Value.Name == "token" && p.Value.Loc == "cookie" {
			found = true
		}
	}
	assert.True(t, found, "cookie param should be found")
}

func TestFindParams_CookieWithHttpCookieType(t *testing.T) {
	type inputWithCookieStruct struct {
		Session http.Cookie `cookie:"session"`
	}
	reg := newRegistry()
	op := &core.Operation{Method: "GET", Path: "/cookie", OperationID: "getCookieStruct"}
	result := findParams[inputWithCookieStruct](reg, op, false)

	for _, p := range result.Paths {
		if p.Value.Name == "session" {
			assert.Equal(t, stringType, p.Value.Type, "http.Cookie cookie should parse from string")
		}
	}
}

func TestFindParams_DeepObjectStyle(t *testing.T) {
	type inputWithDeepObject struct {
		Filter map[string]string `query:"filter,deepObject"`
	}
	reg := newRegistry()
	op := &core.Operation{Method: "GET", Path: "/deep", OperationID: "getDeep"}
	result := findParams[inputWithDeepObject](reg, op, false)

	for _, p := range result.Paths {
		if p.Value.Name == "filter" {
			assert.Equal(t, "deepObject", p.Value.Style)
		}
	}
}

func TestFindParams_RequiredOverride(t *testing.T) {
	type inputRequiredQuery struct {
		Search string `query:"q" required:"true"`
	}
	reg := newRegistry()
	op := &core.Operation{Method: "GET", Path: "/search", OperationID: "search"}
	result := findParams[inputRequiredQuery](reg, op, false)

	for _, p := range result.Paths {
		if p.Value.Name == "q" {
			assert.True(t, p.Value.Required)
		}
	}
	for _, param := range op.Parameters {
		if param.Name == "q" {
			assert.True(t, param.Required)
		}
	}
}

func TestFindParams_TimeParam(t *testing.T) {
	type inputWithTime struct {
		Since time.Time `query:"since"`
	}
	reg := newRegistry()
	op := &core.Operation{Method: "GET", Path: "/time", OperationID: "getTime"}
	result := findParams[inputWithTime](reg, op, false)

	for _, p := range result.Paths {
		if p.Value.Name == "since" {
			assert.Equal(t, time.RFC3339Nano, p.Value.TimeFormat, "query time should default to RFC3339Nano")
		}
	}
}

func TestFindParams_TimeParamHeader(t *testing.T) {
	type inputWithHeaderTime struct {
		IfModifiedSince time.Time `header:"If-Modified-Since"`
	}
	reg := newRegistry()
	op := &core.Operation{Method: "GET", Path: "/timeheader", OperationID: "getTimeHeader"}
	result := findParams[inputWithHeaderTime](reg, op, false)

	for _, p := range result.Paths {
		if p.Value.Name == "If-Modified-Since" {
			assert.Equal(t, http.TimeFormat, p.Value.TimeFormat, "header time should default to http.TimeFormat")
		}
	}
}

func TestFindParams_TimeParamCustomFormat(t *testing.T) {
	type inputWithCustomTime struct {
		Date time.Time `query:"date" timeFormat:"2006-01-02"`
	}
	reg := newRegistry()
	op := &core.Operation{Method: "GET", Path: "/date", OperationID: "getDate"}
	result := findParams[inputWithCustomTime](reg, op, false)

	for _, p := range result.Paths {
		if p.Value.Name == "date" {
			assert.Equal(t, "2006-01-02", p.Value.TimeFormat)
		}
	}
}

func TestParseBodyInto_UnmarshalError(t *testing.T) {
	type Body struct {
		Name string `json:"name"`
	}
	type Input struct {
		Body Body
	}
	v := reflect.New(reflect.TypeFor[Input]()).Elem()
	defaults := &FindResult[any]{}

	errDetail := parseBodyInto(v, []int{0}, json.Unmarshal, []byte(`{invalid`), defaults)
	require.NotNil(t, errDetail)
	assert.Equal(t, "body", errDetail.Location)
	assert.Contains(t, errDetail.Message, "invalid")
}

func TestParseBodyInto_WithDefaults(t *testing.T) {
	type Body struct {
		Name     string `json:"name" default:"world"`
		Greeting string `json:"greeting"`
	}
	type Input struct {
		Body Body
	}
	v := reflect.New(reflect.TypeFor[Input]()).Elem()

	defaults := &FindResult[any]{
		Paths: []FindResultPath[any]{
			// Path is relative: field 0 (Body) -> field 0 (Name).
			{Path: []int{0, 0}, Value: "world"},
		},
	}

	errDetail := parseBodyInto(v, []int{0}, json.Unmarshal, []byte(`{"greeting":"hi"}`), defaults)
	assert.Nil(t, errDetail)
	assert.Equal(t, "hi", v.Field(0).FieldByName("Greeting").String())
	assert.Equal(t, "world", v.Field(0).FieldByName("Name").String())
}

func TestReadBody_ErrorFromReader(t *testing.T) {
	ctx := newMockCtx("POST", "/")
	ctx.body = &errorReader{err: errors.New("read failure")}

	var buf bytes.Buffer
	cErr := ReadBody(&buf, ctx, 1024)
	require.NotNil(t, cErr)
	assert.Equal(t, http.StatusInternalServerError, cErr.Code)
	assert.Contains(t, cErr.Msg, "cannot read request body")
	assert.NotEmpty(t, cErr.Errs)
}

type errorReader struct {
	err error
}

func (r *errorReader) Read(_ []byte) (int, error) {
	return 0, r.err
}

type timeoutError struct{}

func (e *timeoutError) Error() string   { return "timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return false }

func TestReadBody_TimeoutError(t *testing.T) {
	ctx := newMockCtx("POST", "/")
	ctx.body = &errorReader{err: &timeoutError{}}

	var buf bytes.Buffer
	cErr := ReadBody(&buf, ctx, 1024)
	require.NotNil(t, cErr)
	assert.Equal(t, http.StatusRequestTimeout, cErr.Code)
	assert.Contains(t, cErr.Msg, "timeout")
}

type exampleBodyForOutput struct {
	Title string `json:"title"`
}

func (e *exampleBodyForOutput) Example() any {
	return &exampleBodyForOutput{Title: "example title"}
}

func TestSetResponseExample_WithExampleProvider(t *testing.T) {
	mt := &core.MediaType{}
	f := reflect.StructField{
		Name: "Body",
		Type: reflect.TypeFor[exampleBodyForOutput](),
	}
	setResponseExample(mt, f)
	require.NotNil(t, mt.Example)
	ex, ok := mt.Example.(*exampleBodyForOutput)
	require.True(t, ok)
	assert.Equal(t, "example title", ex.Title)
}

func TestSetResponseExample_NilMediaType(t *testing.T) {
	f := reflect.StructField{
		Name: "Body",
		Type: reflect.TypeFor[exampleBodyForOutput](),
	}
	assert.NotPanics(t, func() {
		setResponseExample(nil, f)
	})
}

func TestSetResponseExample_ExistingExample(t *testing.T) {
	mt := &core.MediaType{Example: "already set"}
	f := reflect.StructField{
		Name: "Body",
		Type: reflect.TypeFor[exampleBodyForOutput](),
	}
	setResponseExample(mt, f)
	assert.Equal(t, "already set", mt.Example)
}

func TestSetResponseExample_PointerExampleProvider(t *testing.T) {
	mt := &core.MediaType{}
	f := reflect.StructField{
		Name: "Body",
		Type: reflect.PointerTo(reflect.TypeFor[exampleBodyForOutput]()),
	}
	setResponseExample(mt, f)
	require.NotNil(t, mt.Example)
}

func TestSetResponseExample_FallbackToTags(t *testing.T) {
	type taggedBody struct {
		Name string `json:"name" example:"test_name"`
	}
	mt := &core.MediaType{}
	f := reflect.StructField{
		Name: "Body",
		Type: reflect.TypeFor[taggedBody](),
	}
	setResponseExample(mt, f)
	require.NotNil(t, mt.Example)
}

func TestAnalyzeOutputHeaders_HiddenHeader(t *testing.T) {
	type outputWithHiddenHeader struct {
		Status  int
		Body    string
		Public  string `header:"X-Public"`
		Private string `header:"X-Private" hidden:"true"`
	}
	reg := newRegistry()
	op := &core.Operation{Method: "GET", Path: "/hidden-hdr", OperationID: "hiddenHdr"}
	meta := AnalyzeOutput[outputWithHiddenHeader](op, reg)

	require.NotNil(t, meta)
	statusStr := "200"
	require.NotNil(t, op.Responses[statusStr])
	require.NotNil(t, op.Responses[statusStr].Headers)

	_, hasPublic := op.Responses[statusStr].Headers["X-Public"]
	assert.True(t, hasPublic, "non-hidden header should be documented")

	_, hasPrivate := op.Responses[statusStr].Headers["X-Private"]
	assert.False(t, hasPrivate, "hidden header should not be documented")
}

func TestAnalyzeOutputHeaders_StringerHeader(t *testing.T) {
	type outputWithStringer struct {
		Status int
		Body   string
		Link   url.URL `header:"Link"`
	}
	reg := newRegistry()
	op := &core.Operation{Method: "GET", Path: "/stringer", OperationID: "stringer"}
	meta := AnalyzeOutput[outputWithStringer](op, reg)

	require.NotNil(t, meta)
	statusStr := "200"
	require.NotNil(t, op.Responses[statusStr])
	require.NotNil(t, op.Responses[statusStr].Headers)

	hdr, ok := op.Responses[statusStr].Headers["Link"]
	require.True(t, ok)
	// The schema should be a string type since url.URL implements fmt.Stringer.
	require.NotNil(t, hdr.Schema)
	assert.Equal(t, "string", hdr.Schema.Type)
}

func TestAnalyzeOutputHeaders_SliceHeader(t *testing.T) {
	type outputWithSliceHeader struct {
		Status int
		Body   string
		Tags   []string `header:"X-Tags"`
	}
	reg := newRegistry()
	op := &core.Operation{Method: "GET", Path: "/slice-hdr", OperationID: "sliceHdr"}
	_ = AnalyzeOutput[outputWithSliceHeader](op, reg)

	statusStr := "200"
	require.NotNil(t, op.Responses[statusStr])
	require.NotNil(t, op.Responses[statusStr].Headers)

	hdr, ok := op.Responses[statusStr].Headers["X-Tags"]
	require.True(t, ok)
	require.NotNil(t, hdr.Schema)
	assert.Equal(t, "string", hdr.Schema.Type)
}

type contentTyperOutputBody struct {
	Data string `json:"data" example:"sample"`
}

func (c *contentTyperOutputBody) ContentType(_ string) string {
	return "application/vnd.output+json"
}

func TestAnalyzeOutput_BodyWithContentTyper(t *testing.T) {
	type output struct {
		Status int
		Body   contentTyperOutputBody
	}
	reg := newRegistry()
	op := &core.Operation{Method: "GET", Path: "/ct", OperationID: "ct"}
	meta := AnalyzeOutput[output](op, reg)

	require.NotNil(t, meta)
	statusStr := "200"
	require.NotNil(t, op.Responses[statusStr])
	require.NotNil(t, op.Responses[statusStr].Content)

	_, hasCustomCT := op.Responses[statusStr].Content["application/vnd.output+json"]
	assert.True(t, hasCustomCT, "response content should use ContentTyper content type")
}

func TestAnalyzeOutputHeaders_HiddenParentStruct(t *testing.T) {
	type hiddenGroup struct {
		Internal string `header:"X-Internal"`
	}
	type outputWithHiddenParent struct {
		Status int
		Body   string
		Hidden hiddenGroup `hidden:"true"`
	}
	reg := newRegistry()
	op := &core.Operation{Method: "GET", Path: "/hparent", OperationID: "hparent"}
	_ = AnalyzeOutput[outputWithHiddenParent](op, reg)

	statusStr := "200"
	require.NotNil(t, op.Responses[statusStr])
	if op.Responses[statusStr].Headers != nil {
		_, hasInternal := op.Responses[statusStr].Headers["X-Internal"]
		assert.False(t, hasInternal, "header in hidden parent should not be documented")
	}
}

func TestDocumentParam_NoDuplicates(t *testing.T) {
	op := &core.Operation{
		Method: "GET",
		Path:   "/dedup",
		Parameters: []*core.Param{
			{Name: "q", In: "query"},
		},
	}
	pl := &ParamLocation{
		PFI: &ParamFieldInfo{Name: "q", Loc: "query"},
	}
	documentParam(op, pl)
	count := 0
	for _, p := range op.Parameters {
		if p.Name == "q" && p.In == "query" {
			count++
		}
	}
	assert.Equal(t, 1, count, "should not duplicate existing params")
}

type nilExampleProvider struct {
	Val string
}

func (n *nilExampleProvider) Example() any {
	return nil
}

func TestSetResponseExample_ExampleProviderReturnsNil(t *testing.T) {
	mt := &core.MediaType{}
	f := reflect.StructField{
		Name: "Body",
		Type: reflect.TypeFor[nilExampleProvider](),
	}
	setResponseExample(mt, f)
	assert.Nil(t, mt.Example)
}

func TestBuildExampleValue_ExampleProviderReturnsNil(t *testing.T) {
	v := reflect.New(reflect.TypeFor[nilExampleProvider]()).Elem()
	ok := buildExampleValue(v)
	assert.False(t, ok)
}

func TestSetExampleFromTag_TimeNano(t *testing.T) {
	v := reflect.New(reflect.TypeFor[time.Time]()).Elem()
	ok := setExampleFromTag(v, timeType, "2024-06-15T10:30:00.123456789Z")
	assert.True(t, ok)
	ts := v.Interface().(time.Time)
	assert.Equal(t, 2024, ts.Year())
	assert.Equal(t, 123456789, ts.Nanosecond())
}

func TestBuildExampleStruct_UnexportedFieldsSkipped(t *testing.T) {
	type withUnexported struct {
		Public  string `example:"visible"`
		private string //nolint:unused
	}
	result := buildExampleFromType(reflect.TypeFor[withUnexported]())
	require.NotNil(t, result)
	s := result.(*withUnexported)
	assert.Equal(t, "visible", s.Public)
}

func TestWriteResponse_ExplicitContentType(t *testing.T) {
	api := &mockAPI{}
	ctx := newMockCtx("GET", "/")
	body := map[string]string{"msg": "ok"}

	err := WriteResponse(api, ctx, http.StatusOK, "text/plain", body)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, ctx.status)
}

func TestValidateBody_UnknownContentType(t *testing.T) {
	res := &core.ValidateResult{}
	status := validateBody(
		[]byte(`data`),
		func(data []byte, v any) error { return core.ErrUnknownContentType },
		func(data any, res *core.ValidateResult) {},
		res,
	)
	assert.Equal(t, http.StatusUnsupportedMediaType, status)
	assert.NotEmpty(t, res.Errors)
}

func TestProcessRegularMsgBody_UnmarshalFailsAfterValidation(t *testing.T) {
	type Body struct {
		Count int `json:"count"`
	}
	type Input struct {
		Body Body
	}
	v := reflect.New(reflect.TypeFor[Input]()).Elem()
	res := &core.ValidateResult{}

	// The validator sees a generic map and passes. But json.Unmarshal into
	// the concrete struct will fail because "not_a_number" is not an int.
	cfg := BodyProcessingConfig{
		Body:           []byte(`{"count":"not_a_number"}`),
		Op:             core.Operation{},
		Value:          v,
		HasInputBody:   true,
		InputBodyIndex: []int{0},
		Unmarshaler:    json.Unmarshal,
		Validator:      func(data any, res *core.ValidateResult) {},
		Defaults:       &FindResult[any]{},
		Result:         res,
	}
	_, cErr := ProcessRegularMsgBody(cfg)
	assert.Nil(t, cErr)
	assert.NotEmpty(t, res.Errors)
}

func TestContextError_Fields(t *testing.T) {
	e := &ContextError{
		Code: 500,
		Msg:  "internal error",
		Errs: []error{errors.New("underlying")},
	}
	assert.Equal(t, 500, e.Code)
	assert.Equal(t, "internal error", e.Error())
	assert.Len(t, e.Errs, 1)
}

func TestAnalyzeInput_NonJSONContentType(t *testing.T) {
	type inputWithCustomCT struct {
		Body []byte `contentType:"application/octet-stream"`
	}
	reg := newRegistry()
	op := &core.Operation{Method: "POST", Path: "/upload", OperationID: "upload"}
	meta := AnalyzeInput[inputWithCustomCT](op, reg, false)

	require.NotNil(t, meta)
	assert.True(t, meta.HasInputBody)
	require.NotNil(t, op.RequestBody)
	require.NotNil(t, op.RequestBody.Content)
	_, hasCustom := op.RequestBody.Content["application/octet-stream"]
	assert.True(t, hasCustom)
}

func TestAnalyzeInput_BodyWithNameHint(t *testing.T) {
	type inputWithNameHint struct {
		Body struct {
			Name string `json:"name"`
		} `nameHint:"CustomBodyName"`
	}
	reg := newRegistry()
	op := &core.Operation{Method: "POST", Path: "/hint", OperationID: "hint"}
	meta := AnalyzeInput[inputWithNameHint](op, reg, false)

	require.NotNil(t, meta)
	assert.True(t, meta.HasInputBody)
}

func TestAnalyzeInput_OptionalPointerBody(t *testing.T) {
	type innerBody struct {
		Name string `json:"name"`
	}
	type inputOptionalBody struct {
		Body *innerBody
	}
	reg := newRegistry()
	op := &core.Operation{Method: "POST", Path: "/opt", OperationID: "optBody"}
	meta := AnalyzeInput[inputOptionalBody](op, reg, false)

	require.NotNil(t, meta)
	assert.False(t, op.RequestBody.Required)
}

func TestParseBodyInto_PointerDefault(t *testing.T) {
	type Body struct {
		Name *string `json:"name"`
	}
	type Input struct {
		Body Body
	}
	defaultVal := "default_name"
	v := reflect.New(reflect.TypeFor[Input]()).Elem()

	defaults := &FindResult[any]{
		Paths: []FindResultPath[any]{
			{Path: []int{0, 0}, Value: &defaultVal},
		},
	}

	errDetail := parseBodyInto(v, []int{0}, json.Unmarshal, []byte(`{}`), defaults)
	assert.Nil(t, errDetail)
	bodyVal := v.Field(0)
	nameField := bodyVal.FieldByName("Name")
	require.False(t, nameField.IsNil())
	assert.Equal(t, "default_name", nameField.Elem().String())
}

func TestBuildExampleStruct_PointerToNonStruct(t *testing.T) {
	type withPtrToNonStruct struct {
		Name  string `example:"test"`
		Count *int
	}
	result := buildExampleFromType(reflect.TypeFor[withPtrToNonStruct]())
	require.NotNil(t, result)
	s := result.(*withPtrToNonStruct)
	assert.Equal(t, "test", s.Name)
	assert.Nil(t, s.Count)
}

type convertibleExampleType int

func (c *convertibleExampleType) Example() any {
	// Return a plain int, which is ConvertibleTo convertibleExampleType but not
	// directly AssignableTo (since the types differ).
	return convertibleExampleType(42)
}

func TestBuildExampleValue_ConvertibleType(t *testing.T) {
	v := reflect.New(reflect.TypeFor[convertibleExampleType]()).Elem()
	ok := buildExampleValue(v)
	assert.True(t, ok)
	assert.Equal(t, convertibleExampleType(42), v.Interface())
}

func TestBuildExampleMap_NoExamplesInValue(t *testing.T) {
	type noEx struct {
		Name string
	}
	v := reflect.New(reflect.TypeFor[map[string]noEx]()).Elem()
	ok := buildExampleMap(v, reflect.TypeFor[map[string]noEx]())
	assert.False(t, ok)
}

func TestBuildExampleSlice_NoExamplesInElem(t *testing.T) {
	type noEx struct {
		Name string
	}
	v := reflect.New(reflect.TypeFor[[]noEx]()).Elem()
	ok := buildExampleSlice(v, reflect.TypeFor[[]noEx]())
	assert.False(t, ok)
}

func TestAnyFieldHasExample_UnexportedSkipped(t *testing.T) {
	type withPrivate struct {
		private string //nolint:unused
	}
	assert.False(t, anyFieldHasExample(reflect.TypeFor[withPrivate]()))
}

type EmbeddedHeaders struct {
	ETag string `header:"ETag"`
}

func TestFindHeaders_EmbeddedStructDiscovery(t *testing.T) {
	type outputWithEmbedded struct {
		Status int
		Body   string
		EmbeddedHeaders
	}
	result := findHeaders[outputWithEmbedded]()
	// The exported embedded struct's inner fields should be found via walkType
	// recursion into anonymous fields. The onField callback in findHeaders
	// returns nil for the anonymous field itself (sf.Anonymous check), but
	// walkType recurses and finds the non-anonymous ETag field inside.
	found := false
	for _, p := range result.Paths {
		if p.Value.Name == "ETag" {
			found = true
		}
	}
	assert.True(t, found, "headers from exported embedded structs should be discovered")
}

func TestAnalyzeOutput_BodyWithNameHint(t *testing.T) {
	type outputWithNameHint struct {
		Status int
		Body   struct {
			Data string `json:"data" example:"sample"`
		} `nameHint:"CustomOutput"`
	}
	reg := newRegistry()
	op := &core.Operation{Method: "GET", Path: "/nh", OperationID: "nh"}
	meta := AnalyzeOutput[outputWithNameHint](op, reg)

	require.NotNil(t, meta)
	assert.GreaterOrEqual(t, meta.BodyIndex, 0)
}

type mockAPIWithNegotiateAndMarshalError struct {
	mockAPI
}

func (m *mockAPIWithNegotiateAndMarshalError) Negotiate(_ string) (string, error) {
	return "", errors.New("bad accept")
}

func (m *mockAPIWithNegotiateAndMarshalError) Marshal(_ io.Writer, _ string, _ any) error {
	return errors.New("marshal failed too")
}

func TestWriteResponse_NegotiateErrorThenMarshalError(t *testing.T) {
	api := &mockAPIWithNegotiateAndMarshalError{
		mockAPI: mockAPI{errHandler: &mockErrorHandler{}},
	}
	ctx := newMockCtx("GET", "/")

	err := WriteResponse(api, ctx, http.StatusOK, "", "body")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "marshal failed too")
}

func TestEveryPB_PathParam(t *testing.T) {
	type S struct {
		ID string `path:"id"`
	}
	fr := &FindResult[string]{
		Paths: []FindResultPath[string]{
			{Path: []int{0}, Value: "val"},
		},
	}
	v := reflect.New(reflect.TypeFor[S]()).Elem()
	v.Field(0).SetString("123")
	pb := core.NewPathBuffer(nil, 0)
	var captured string
	fr.EveryPB(pb, v, func(item reflect.Value, val string) {
		captured = pb.String()
	})
	assert.Contains(t, captured, "path")
	assert.Contains(t, captured, "id")
}

func TestEveryPB_QueryParam(t *testing.T) {
	type S struct {
		Search string `query:"q"`
	}
	fr := &FindResult[string]{
		Paths: []FindResultPath[string]{
			{Path: []int{0}, Value: "val"},
		},
	}
	v := reflect.New(reflect.TypeFor[S]()).Elem()
	v.Field(0).SetString("test")
	pb := core.NewPathBuffer(nil, 0)
	var captured string
	fr.EveryPB(pb, v, func(item reflect.Value, val string) {
		captured = pb.String()
	})
	assert.Contains(t, captured, "query")
	assert.Contains(t, captured, "q")
}

func TestEveryPB_HeaderParam(t *testing.T) {
	type S struct {
		Auth string `header:"Authorization"`
	}
	fr := &FindResult[string]{
		Paths: []FindResultPath[string]{
			{Path: []int{0}, Value: "val"},
		},
	}
	v := reflect.New(reflect.TypeFor[S]()).Elem()
	v.Field(0).SetString("Bearer token")
	pb := core.NewPathBuffer(nil, 0)
	var captured string
	fr.EveryPB(pb, v, func(item reflect.Value, val string) {
		captured = pb.String()
	})
	assert.Contains(t, captured, "header")
	assert.Contains(t, captured, "Authorization")
}

func TestParseInto_SliceUnparsable(t *testing.T) {
	ctx := newMockCtx("GET", "/")
	f := reflect.New(reflect.TypeFor[[]complex128]()).Elem()
	p := ParamFieldInfo{Type: reflect.TypeFor[[]complex128](), Name: "vals"}
	_, err := ParseInto(ctx, f, "1,2", nil, p)
	assert.Error(t, err)
}

type badTextUnmarshaler struct{}

func (b *badTextUnmarshaler) UnmarshalText(_ []byte) error {
	return errors.New("unmarshal text failed")
}

func TestParseInto_TextUnmarshalerError(t *testing.T) {
	ctx := newMockCtx("GET", "/")
	f := reflect.New(reflect.TypeFor[badTextUnmarshaler]()).Elem()
	p := ParamFieldInfo{Type: reflect.TypeFor[badTextUnmarshaler]()}
	_, err := ParseInto(ctx, f, "value", nil, p)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid value")
}

type multipartMockContext struct {
	mockContext
	form *multipart.Form
}

func (m *multipartMockContext) GetMultipartForm() (*multipart.Form, error) {
	return m.form, nil
}

func newMultipartFileHeader(name string, content []byte) *multipart.FileHeader {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", name)
	if err != nil {
		panic(err)
	}
	_, _ = part.Write(content)
	_ = writer.Close()

	reader := multipart.NewReader(body, writer.Boundary())
	form, err2 := reader.ReadForm(10 << 20)
	if err2 != nil {
		panic(err2)
	}
	files := form.File["file"]
	if len(files) == 0 {
		panic("no file in form")
	}
	return files[0]
}

func TestAnalyzeMultipartFields(t *testing.T) {
	type formInput struct {
		Name   string        `form:"name"`
		Age    int           `form:"age"`
		Avatar core.FormFile `form:"avatar"`
	}

	fields := AnalyzeMultipartFields(reflect.TypeFor[formInput]())

	require.Len(t, fields, 3)
	assert.Equal(t, "name", fields[0].Name)
	assert.False(t, fields[0].IsFile)
	assert.Equal(t, "age", fields[1].Name)
	assert.False(t, fields[1].IsFile)
	assert.Equal(t, "avatar", fields[2].Name)
	assert.True(t, fields[2].IsFile)
	assert.False(t, fields[2].IsSlice)
}

func TestAnalyzeMultipartFields_MultipleFiles(t *testing.T) {
	type formInput struct {
		Photos []core.FormFile `form:"photos"`
	}

	fields := AnalyzeMultipartFields(reflect.TypeFor[formInput]())

	require.Len(t, fields, 1)
	assert.Equal(t, "photos", fields[0].Name)
	assert.True(t, fields[0].IsFile)
	assert.True(t, fields[0].IsSlice)
}

func TestProcessMultipartForm_SimpleFields(t *testing.T) {
	type formInput struct {
		Name string `form:"name"`
		Age  int    `form:"age"`
	}

	form := &multipart.Form{
		Value: map[string][]string{
			"name": {"Alice"},
			"age":  {"30"},
		},
		File: map[string][]*multipart.FileHeader{},
	}

	ctx := &multipartMockContext{
		mockContext: *newMockCtx("POST", "/upload"),
		form:        form,
	}

	fields := AnalyzeMultipartFields(reflect.TypeFor[formInput]())
	v := reflect.New(reflect.TypeFor[formInput]()).Elem()

	ctxErr := ProcessMultipartForm(ctx, MultipartProcessingConfig{
		Value:  v,
		Fields: fields,
	})

	assert.Nil(t, ctxErr)
	result := v.Interface().(formInput)
	assert.Equal(t, "Alice", result.Name)
	assert.Equal(t, 30, result.Age)
}

func TestProcessMultipartForm_SingleFile(t *testing.T) {
	type formInput struct {
		Avatar core.FormFile `form:"avatar"`
	}

	fh := newMultipartFileHeader("avatar.png", []byte("fake-png-data"))

	form := &multipart.Form{
		Value: map[string][]string{},
		File: map[string][]*multipart.FileHeader{
			"avatar": {fh},
		},
	}

	ctx := &multipartMockContext{
		mockContext: *newMockCtx("POST", "/upload"),
		form:        form,
	}

	fields := AnalyzeMultipartFields(reflect.TypeFor[formInput]())
	v := reflect.New(reflect.TypeFor[formInput]()).Elem()

	ctxErr := ProcessMultipartForm(ctx, MultipartProcessingConfig{
		Value:  v,
		Fields: fields,
	})

	assert.Nil(t, ctxErr)
	result := v.Interface().(formInput)
	assert.NotNil(t, result.Avatar.File)
	assert.Equal(t, "avatar.png", result.Avatar.Filename)

	buf := &bytes.Buffer{}
	_, _ = io.Copy(buf, result.Avatar.File)
	assert.Equal(t, "fake-png-data", buf.String())
}

func TestProcessMultipartForm_MultipleFiles(t *testing.T) {
	type formInput struct {
		Photos []core.FormFile `form:"photos"`
	}

	fh1 := newMultipartFileHeader("a.jpg", []byte("jpg-a"))
	fh2 := newMultipartFileHeader("b.jpg", []byte("jpg-b"))

	form := &multipart.Form{
		Value: map[string][]string{},
		File: map[string][]*multipart.FileHeader{
			"photos": {fh1, fh2},
		},
	}

	ctx := &multipartMockContext{
		mockContext: *newMockCtx("POST", "/upload"),
		form:        form,
	}

	fields := AnalyzeMultipartFields(reflect.TypeFor[formInput]())
	v := reflect.New(reflect.TypeFor[formInput]()).Elem()

	ctxErr := ProcessMultipartForm(ctx, MultipartProcessingConfig{
		Value:  v,
		Fields: fields,
	})

	assert.Nil(t, ctxErr)
	result := v.Interface().(formInput)
	require.Len(t, result.Photos, 2)
	assert.Equal(t, "a.jpg", result.Photos[0].Filename)
	assert.Equal(t, "b.jpg", result.Photos[1].Filename)
}

func TestProcessMultipartForm_NilForm(t *testing.T) {
	ctx := &multipartMockContext{
		mockContext: *newMockCtx("POST", "/upload"),
		form:        nil,
	}

	ctxErr := ProcessMultipartForm(ctx, MultipartProcessingConfig{
		Value:  reflect.New(reflect.TypeFor[struct{}]()).Elem(),
		Fields: nil,
	})

	require.NotNil(t, ctxErr)
	assert.Equal(t, http.StatusBadRequest, ctxErr.Code)
}

func TestProcessMultipartForm_InvalidFieldValue(t *testing.T) {
	type formInput struct {
		Age int `form:"age"`
	}

	form := &multipart.Form{
		Value: map[string][]string{
			"age": {"not-a-number"},
		},
		File: map[string][]*multipart.FileHeader{},
	}

	ctx := &multipartMockContext{
		mockContext: *newMockCtx("POST", "/upload"),
		form:        form,
	}

	fields := AnalyzeMultipartFields(reflect.TypeFor[formInput]())
	v := reflect.New(reflect.TypeFor[formInput]()).Elem()

	ctxErr := ProcessMultipartForm(ctx, MultipartProcessingConfig{
		Value:  v,
		Fields: fields,
	})

	require.NotNil(t, ctxErr)
	assert.Equal(t, http.StatusUnprocessableEntity, ctxErr.Code)
	assert.Contains(t, ctxErr.Msg, "age")
}

func TestSetupMultipartRequestBody(t *testing.T) {
	type formInput struct {
		Name   string          `form:"name"`
		Count  int             `form:"count"`
		Avatar core.FormFile   `form:"avatar"`
		Photos []core.FormFile `form:"photos"`
	}

	op := &core.Operation{Method: "POST", Path: "/upload", OperationID: "upload"}
	fields := AnalyzeMultipartFields(reflect.TypeFor[formInput]())
	SetupMultipartRequestBody(op, fields)

	require.NotNil(t, op.RequestBody)
	require.NotNil(t, op.RequestBody.Content["multipart/form-data"])
	s := op.RequestBody.Content["multipart/form-data"].Schema
	require.NotNil(t, s)
	assert.Equal(t, core.TypeObject, s.Type)
	require.NotNil(t, s.Properties["name"])
	assert.Equal(t, core.TypeString, s.Properties["name"].Type)
	require.NotNil(t, s.Properties["count"])
	assert.Equal(t, core.TypeInteger, s.Properties["count"].Type)
	require.NotNil(t, s.Properties["avatar"])
	assert.Equal(t, core.TypeString, s.Properties["avatar"].Type)
	assert.Equal(t, "binary", s.Properties["avatar"].Format)
	require.NotNil(t, s.Properties["photos"])
	assert.Equal(t, core.TypeArray, s.Properties["photos"].Type)
	require.NotNil(t, s.Properties["photos"].Items)
	assert.Equal(t, "binary", s.Properties["photos"].Items.Format)
}

func TestAnalyzeInput_MultipartDetection(t *testing.T) {
	type formBody struct {
		Name   string        `form:"name"`
		Avatar core.FormFile `form:"avatar"`
	}
	type input struct {
		Body formBody
	}

	reg := newRegistry()
	op := &core.Operation{Method: "POST", Path: "/upload", OperationID: "upload"}
	meta := AnalyzeInput[input](op, reg, false)

	require.NotNil(t, meta)
	assert.True(t, meta.HasInputBody)
	assert.NotNil(t, meta.MultipartFields)
	assert.Len(t, meta.MultipartFields, 2)
	require.NotNil(t, op.RequestBody)
	assert.NotNil(t, op.RequestBody.Content["multipart/form-data"])
}

func TestHasMultipartFields(t *testing.T) {
	type withFile struct {
		Avatar core.FormFile `form:"avatar"`
	}
	type withoutFile struct {
		Name string `form:"name"`
	}
	type noForm struct {
		Name string `json:"name"`
	}

	assert.True(t, HasMultipartFields(reflect.TypeFor[withFile]()))
	assert.False(t, HasMultipartFields(reflect.TypeFor[withoutFile]()))
	assert.False(t, HasMultipartFields(reflect.TypeFor[noForm]()))
}

func TestMultipartFormFiles_Data(t *testing.T) {
	type inner struct {
		Name string
	}
	m := &MultipartFormFiles[inner]{data: inner{Name: "test"}}
	assert.Equal(t, "test", m.Data().Name)
}

func TestProcessMultipartForm_MissingFileSkipped(t *testing.T) {
	type formInput struct {
		Avatar core.FormFile `form:"avatar"`
	}

	form := &multipart.Form{
		Value: map[string][]string{},
		File:  map[string][]*multipart.FileHeader{},
	}

	ctx := &multipartMockContext{
		mockContext: *newMockCtx("POST", "/upload"),
		form:        form,
	}

	fields := AnalyzeMultipartFields(reflect.TypeFor[formInput]())
	v := reflect.New(reflect.TypeFor[formInput]()).Elem()

	ctxErr := ProcessMultipartForm(ctx, MultipartProcessingConfig{
		Value:  v,
		Fields: fields,
	})

	assert.Nil(t, ctxErr)
	result := v.Interface().(formInput)
	assert.Nil(t, result.Avatar.File)
}

func TestProcessMultipartForm_MissingValueSkipped(t *testing.T) {
	type formInput struct {
		Name string `form:"name"`
	}

	form := &multipart.Form{
		Value: map[string][]string{},
		File:  map[string][]*multipart.FileHeader{},
	}

	ctx := &multipartMockContext{
		mockContext: *newMockCtx("POST", "/upload"),
		form:        form,
	}

	fields := AnalyzeMultipartFields(reflect.TypeFor[formInput]())
	v := reflect.New(reflect.TypeFor[formInput]()).Elem()

	ctxErr := ProcessMultipartForm(ctx, MultipartProcessingConfig{
		Value:  v,
		Fields: fields,
	})

	assert.Nil(t, ctxErr)
	result := v.Interface().(formInput)
	assert.Empty(t, result.Name)
}
