package binding

import (
	"net/http"
	"reflect"
	"strconv"
	"time"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/schema"
)

var (
	bodyCallbackType     = reflect.TypeFor[func(core.Context, core.API)]()
	contentTyperType     = reflect.TypeFor[core.ContentTyper]()
	resolverType         = reflect.TypeFor[core.Resolver]()
	resolverWithPathType = reflect.TypeFor[core.ResolverWithPath]()
)

// InputMeta holds metadata about a request input type, including parameter
// locations, body schema, resolvers, and default values.
type InputMeta struct {
	Params             *FindResult[*ParamFieldInfo]
	InputBodyIndex     []int
	HasInputBody       bool
	RawBodyIndex       []int
	InSchema           *core.Schema
	Resolvers          *FindResult[bool]
	Defaults           *FindResult[any]
	MultipartFields    []MultipartFieldInfo
	MultipartBodyIndex []int
}

// OutputMeta holds metadata about a response output type, including header
// locations, status code index, and body field index.
type OutputMeta struct {
	Headers     *FindResult[*HeaderInfo]
	StatusIndex int
	BodyIndex   int
	BodyFunc    bool
}

// AnalyzeInput inspects the input type I and populates the operation's OpenAPI
// metadata (parameters, request body, schemas) while returning an InputMeta
// that the runtime uses to bind request data.
func AnalyzeInput[I any](op *core.Operation, registry core.Registry, fieldsOptionalByDefault bool) *InputMeta {
	inputType := reflect.TypeFor[I]()
	initResponses(op)
	meta := &InputMeta{}

	meta.Params = findParams[I](registry, op, fieldsOptionalByDefault)

	if f, ok := inputType.FieldByName("Body"); ok {
		bodyType := core.Deref(f.Type)
		if IsMultipartFormFilesType(bodyType) && bodyType.NumField() > 0 {
			bodyType = bodyType.Field(0).Type
		}
		multipartFields := AnalyzeMultipartFields(bodyType)
		if len(multipartFields) > 0 && HasMultipartFields(bodyType) {
			meta.HasInputBody = true
			meta.InputBodyIndex = f.Index
			meta.MultipartFields = multipartFields
			meta.MultipartBodyIndex = f.Index
			SetupMultipartRequestBody(op, multipartFields)
		} else {
			meta.HasInputBody = true
			meta.InputBodyIndex = f.Index
			initRequestBody(op)
			setRequestBodyFromBody(op, registry, f, inputType)
			ensureBodyReadTimeout(op)
			ensureMaxBodyBytes(op)
		}
	}

	if op.RequestBody != nil {
		for _, mediaType := range op.RequestBody.Content {
			if mediaType.Schema != nil {
				// Ensure all schema validation errors are set up properly as some
				// parts of the schema may have been user-supplied.
				mediaType.Schema.PrecomputeMessages()
			}
		}
	}

	if op.RequestBody != nil && op.RequestBody.Content != nil {
		// Try to get schema from any available content type.
		// Prefer application/json for backwards compatibility, then try others.
		if op.RequestBody.Content["application/json"] != nil && op.RequestBody.Content["application/json"].Schema != nil {
			meta.HasInputBody = true
			meta.InSchema = op.RequestBody.Content["application/json"].Schema
		} else {
			for _, mediaType := range op.RequestBody.Content {
				if mediaType.Schema != nil && mediaType.Schema.Type != "string" && mediaType.Schema.Format != "binary" {
					meta.HasInputBody = true
					meta.InSchema = mediaType.Schema
					break
				}
			}
		}
	}

	meta.Resolvers = findResolvers[I]()
	meta.Defaults = findDefaults[I](registry)

	return meta
}

// AnalyzeOutput inspects the output type O and populates the operation's OpenAPI
// response metadata (status codes, headers, body schema) while returning an
// OutputMeta that the runtime uses to write responses.
func AnalyzeOutput[O any](op *core.Operation, registry core.Registry) *OutputMeta {
	outputType := reflect.TypeFor[O]()
	initResponses(op)
	meta := &OutputMeta{
		StatusIndex: -1,
		BodyIndex:   -1,
	}

	if f, ok := outputType.FieldByName("Status"); ok {
		meta.StatusIndex = f.Index[0]
		if f.Type.Kind() != reflect.Int {
			panic("status field must be an int")
		}
	}

	if f, ok := outputType.FieldByName("Body"); ok {
		meta.BodyIndex = f.Index[0]
		analyzeOutputBody(meta, f, op, registry, outputType)
	}

	if op.DefaultStatus == 0 {
		switch {
		case meta.BodyIndex != -1:
			op.DefaultStatus = http.StatusOK
		case op.Method == http.MethodHead:
			op.DefaultStatus = http.StatusOK
		default:
			op.DefaultStatus = http.StatusNoContent
		}
	}
	defaultStatusStr := strconv.Itoa(op.DefaultStatus)
	if op.Responses[defaultStatusStr] == nil {
		op.Responses[defaultStatusStr] = &core.Response{
			Description: http.StatusText(op.DefaultStatus),
		}
	}

	meta.Headers = findHeaders[O]()
	analyzeOutputHeaders(meta, op, outputType, registry, defaultStatusStr)

	return meta
}

func analyzeOutputBody(meta *OutputMeta, f reflect.StructField, op *core.Operation, registry core.Registry, outputType reflect.Type) {
	if f.Type.Kind() == reflect.Func {
		meta.BodyFunc = true

		if f.Type != bodyCallbackType {
			panic("body field must be a function with signature func(core.Context, core.API)")
		}
	}
	status := op.DefaultStatus
	if status == 0 {
		status = http.StatusOK
	}
	statusStr := strconv.Itoa(status)
	if op.Responses[statusStr] == nil {
		op.Responses[statusStr] = &core.Response{}
	}
	if op.Responses[statusStr].Description == "" {
		op.Responses[statusStr].Description = http.StatusText(status)
	}
	if op.Responses[statusStr].Headers == nil {
		op.Responses[statusStr].Headers = map[string]*core.Param{}
	}
	if !meta.BodyFunc {
		hint := getHint(outputType, f.Name, op.OperationID+"Response")
		if nameHint := f.Tag.Get("nameHint"); nameHint != "" {
			hint = nameHint
		}
		outSchema := schema.FromField(registry, f, hint)
		if op.Responses[statusStr].Content == nil {
			op.Responses[statusStr].Content = map[string]*core.MediaType{}
		}
		contentType := "application/json"
		if reflect.PointerTo(f.Type).Implements(contentTyperType) {
			if ctf, ok := reflect.New(f.Type).Interface().(core.ContentTyper); ok {
				contentType = ctf.ContentType(contentType)
			}
		}
		if len(op.Responses[statusStr].Content) == 0 {
			op.Responses[statusStr].Content[contentType] = &core.MediaType{}
		}
		if op.Responses[statusStr].Content[contentType] != nil && op.Responses[statusStr].Content[contentType].Schema == nil {
			op.Responses[statusStr].Content[contentType].Schema = outSchema
		}

		setResponseExample(op.Responses[statusStr].Content[contentType], f)
	}
}

func analyzeOutputHeaders(meta *OutputMeta, op *core.Operation, outputType reflect.Type, registry core.Registry, defaultStatusStr string) {
	for _, entry := range meta.Headers.Paths {
		v := entry.Value

		hidden := false
		currentType := outputType
		for _, idx := range entry.Path {
			currentType = core.BaseType(currentType)

			field := currentType.Field(idx)
			if boolTag(field, "hidden", false) {
				hidden = true
				break
			}

			currentType = field.Type
		}
		if hidden {
			continue
		}

		resp := op.Responses[defaultStatusStr]
		if resp == nil {
			continue
		}
		if resp.Headers == nil {
			resp.Headers = map[string]*core.Param{}
		}
		f := v.Field
		if f.Type.Kind() == reflect.Slice {
			f.Type = core.Deref(f.Type.Elem())
		}
		if reflect.PointerTo(f.Type).Implements(fmtStringerType) {
			// Special case: this field will be written as a string by calling
			// .String() on the value.
			f.Type = stringType
		}
		resp.Headers[v.Name] = &core.Param{
			Schema: schema.FromField(registry, f, getHint(outputType, f.Name, op.OperationID+defaultStatusStr+v.Name)),
		}
	}
}

func ensureBodyReadTimeout(op *core.Operation) {
	if op.BodyReadTimeout == 0 {
		op.BodyReadTimeout = 5 * time.Second
	}
}

func ensureMaxBodyBytes(op *core.Operation) {
	if op.MaxBodyBytes == 0 {
		op.MaxBodyBytes = 1024 * 1024
	}
}

func findResolvers[I any]() *FindResult[bool] {
	t := reflect.TypeFor[I]()
	return findInType(t, func(t reflect.Type, path []int) bool {
		tp := reflect.PointerTo(t)
		return tp.Implements(resolverType) || tp.Implements(resolverWithPathType)
	}, nil, true)
}

func initRequestBody(op *core.Operation, rbOpts ...func(*core.RequestBody)) {
	if op.RequestBody == nil {
		op.RequestBody = &core.RequestBody{}
	}
	if op.RequestBody.Content == nil {
		op.RequestBody.Content = map[string]*core.MediaType{}
	}
	for _, opt := range rbOpts {
		opt(op.RequestBody)
	}
}

func initResponses(op *core.Operation) {
	if op.Responses == nil {
		op.Responses = map[string]*core.Response{}
	}
}

func setRequestBodyFromBody(op *core.Operation, registry core.Registry, fBody reflect.StructField, inputType reflect.Type) {
	if fBody.Tag.Get("required") == "true" || (fBody.Type.Kind() != reflect.Pointer && fBody.Type.Kind() != reflect.Interface) {
		setRequestBodyRequired(op.RequestBody)
	}
	contentType := "application/json"
	if c := fBody.Tag.Get("contentType"); c != "" {
		contentType = c
	}
	if op.RequestBody.Content[contentType] == nil {
		op.RequestBody.Content[contentType] = &core.MediaType{}
	}
	mt := op.RequestBody.Content[contentType]
	if mt.Schema == nil {
		hint := getHint(inputType, fBody.Name, op.OperationID+"Request")
		if nameHint := fBody.Tag.Get("nameHint"); nameHint != "" {
			hint = nameHint
		}
		mt.Schema = schema.FromField(registry, fBody, hint)
	}
	if mt.Example == nil {
		setResponseExample(mt, fBody)
	}
}

func setRequestBodyRequired(rb *core.RequestBody) {
	rb.Required = true
}

func setResponseExample(mt *core.MediaType, f reflect.StructField) {
	if mt == nil || mt.Example != nil {
		return
	}
	bodyType := core.Deref(f.Type)
	if reflect.PointerTo(bodyType).Implements(exampleProviderType) {
		inst := reflect.New(bodyType)
		result := inst.MethodByName("Example").Call(nil)
		if len(result) > 0 && !result[0].IsNil() {
			mt.Example = result[0].Interface()
		}
	} else {
		if ex := buildExampleFromType(bodyType); ex != nil {
			mt.Example = ex
		}
	}
}
