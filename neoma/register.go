package neoma

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/MeGaNeKoS/neoma/binding"
	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/middleware"
	"github.com/MeGaNeKoS/neoma/openapi"
	"github.com/MeGaNeKoS/neoma/schema"
)

const styleDeepObject = "deepObject"

type validateDeps struct {
	pb  *core.PathBuffer
	res *core.ValidateResult
}

var validatePool = sync.Pool{
	New: func() any {
		return &validateDeps{
			pb:  core.NewPathBuffer(make([]byte, 0, 128), 0),
			res: &core.ValidateResult{},
		}
	},
}

var bufPool = sync.Pool{
	New: func() any {
		return &bytes.Buffer{}
	},
}

func Register[I, O any](api core.API, op core.Operation, handler func(context.Context, *I) (*O, error)) {
	oapi := api.OpenAPI()
	registry := oapi.Components.Schemas

	if op.Method == "" {
		panic("method must be specified in operation")
	}
	if op.Path == "" {
		// Allow empty paths for groups that prepend a prefix.
		if _, ok := api.(*middleware.Group); !ok {
			panic("path must be specified in operation")
		}
	}

	inputType := reflect.TypeFor[I]()
	if inputType.Kind() != reflect.Struct {
		panic("input must be a struct")
	}

	outputType := reflect.TypeFor[O]()
	if outputType.Kind() != reflect.Struct {
		panic("output must be a struct")
	}

	fieldsOptional := false
	if cp, ok := api.(interface{ Config() core.Config }); ok {
		fieldsOptional = cp.Config().FieldsOptionalByDefault
	}

	inputMeta := binding.AnalyzeInput[I](&op, registry, fieldsOptional)

	outMeta := binding.AnalyzeOutput[O](&op, registry)

	expandContentTypesFromAPI(api, &op)

	var discovered []core.DiscoveredError
	if !op.SkipDiscoveredErrors {
		if op.OperationID != "" {
			discovered = getDiscoveredErrors(op.OperationID)
		}
		if len(discovered) == 0 {
			if fn := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()); fn != nil {
				name := fn.Name()
				if idx := strings.LastIndex(name, "."); idx >= 0 {
					name = name[idx+1:]
				}
				if idx := strings.IndexAny(name, "[-"); idx >= 0 {
					name = name[:idx]
				}
				discovered = getDiscoveredErrors(name)
			}
		}
	}

	if len(op.Errors) == 0 && len(discovered) > 0 {
		seen := map[int]bool{}
		for _, d := range discovered {
			if !seen[d.Status] {
				op.Errors = append(op.Errors, d.Status)
				seen[d.Status] = true
			}
		}
	}

	if len(op.Errors) > 0 {
		if len(inputMeta.Params.Paths) > 0 || inputMeta.HasInputBody {
			if !containsInt(op.Errors, http.StatusUnprocessableEntity) {
				op.Errors = append(op.Errors, http.StatusUnprocessableEntity)
			}
		}
		if !containsInt(op.Errors, http.StatusInternalServerError) {
			op.Errors = append(op.Errors, http.StatusInternalServerError)
		}
	}

	openapi.DefineErrors(&op, registry, api.ErrorHandler(), discovered)

	if documenter, ok := api.(middleware.OperationDocumenter); ok {
		documenter.DocumentOperation(&op)
	} else if op.Hidden {
		// Store hidden operations for the internal spec if the API
		// supports it. This handles cases where the API is wrapped
		// (e.g., by test utilities) and DocumentOperation is not
		// directly reachable.
		trackHiddenOp(api, &op)
	} else {
		oapi.AddOperation(&op)
	}

	knownParams := make(map[string]struct{})
	var deepPrefixes []string
	for i := range inputMeta.Params.Paths {
		p := inputMeta.Params.Paths[i].Value
		if p == nil || p.Loc != "query" {
			continue
		}
		if p.Style == styleDeepObject {
			deepPrefixes = append(deepPrefixes, p.Name+"[")
			continue
		}
		knownParams[p.Name] = struct{}{}
	}

	rejectUnknownQuery := op.RejectUnknownQueryParameters
	if !rejectUnknownQuery {
		if cp, ok := api.(interface{ Config() core.Config }); ok {
			rejectUnknownQuery = cp.Config().RejectUnknownQueryParameters
		}
	}

	a := api.Adapter()
	a.Handle(&op, api.Middlewares().Handler(op.Middlewares.Handler(func(ctx core.Context) {
		var input I

		deps, _ := validatePool.Get().(*validateDeps)
		defer func() {
			// Only return to pool if they haven't grown too large.
			if cap(deps.pb.Bytes()) <= 2048 && cap(deps.res.Errors) <= 128 {
				deps.pb.Reset()
				deps.res.Reset()
				validatePool.Put(deps)
			}
		}()
		pb := deps.pb
		res := deps.res

		errStatus := http.StatusUnprocessableEntity

		var cookies map[string]*http.Cookie

		v := reflect.ValueOf(&input).Elem()

		if !op.SkipValidateParams && rejectUnknownQuery {
			u := ctx.URL()
			q := u.Query()
		outer:
			for key := range q {
				if _, ok := knownParams[key]; ok {
					continue
				}
				for _, prefix := range deepPrefixes {
					if strings.HasPrefix(key, prefix) {
						continue outer
					}
				}
				pb.Reset()
				pb.Push("query")
				pb.Push(key)
				res.Add(pb, "", "unknown query parameter")
			}
			if res.HasErrors() {
				writeErrFromContext(api, ctx, &binding.ContextError{
					Code: http.StatusUnprocessableEntity,
					Msg:  "validation failed",
					Errs: res.Errors,
				}, *res)
				return
			}
		}

		reqURL := ctx.URL()
		queryValues := reqURL.Query()

		inputMeta.Params.Every(v, func(f reflect.Value, p *binding.ParamFieldInfo) {
			// For pointer fields (#393), check whether a value is present
			// before allocating. If absent, leave the pointer nil.
			if p.IsPointer {
				pb.Reset()
				pb.Push(p.Loc)
				pb.Push(p.Name)

				if p.Loc == "cookie" {
					if cookies == nil {
						cookies = map[string]*http.Cookie{}
						for _, c := range readCookies(ctx) {
							cookies[c.Name] = c
						}
					}
				}

				value := binding.GetParamValue(*p, ctx, cookies)
				if value == "" {
					if !op.SkipValidateParams && p.Required {
						res.Add(pb, "", "required "+p.Loc+" parameter is missing")
					}
					return
				}

				elem := reflect.New(p.Type)
				receiver :=elem.Elem()
				pv, err := binding.ParseInto(ctx, receiver, value, nil, *p, queryValues)
				if err != nil {
					res.Add(pb, value, err.Error())
					return
				}
				f.Set(elem)

				if !op.SkipValidateParams {
					schema.Validate(registry, p.Schema, pb, core.ModeWriteToServer, pv, res)
				}
				return
			}

			f = reflect.Indirect(f)
			if f.Kind() == reflect.Invalid {
				return
			}

			pb.Reset()
			pb.Push(p.Loc)
			pb.Push(p.Name)

			if p.Loc == "cookie" {
				if cookies == nil {
					cookies = map[string]*http.Cookie{}
					for _, c := range readCookies(ctx) {
						cookies[c.Name] = c
					}
				}
			}

			receiver :=f
			if f.Addr().Type().Implements(reflect.TypeFor[binding.ParamWrapper]()) {
				if pw, ok := f.Addr().Interface().(binding.ParamWrapper); ok {
					receiver = pw.Receiver()
				}
			}

			var pv any
			var isSet bool
			if p.Loc == "query" && p.Style == styleDeepObject {
				u := ctx.URL()
				value := binding.ParseDeepObjectQuery(u.Query(), p.Name)
				isSet = len(value) > 0
				if len(value) == 0 {
					if !op.SkipValidateParams && p.Required {
						res.Add(pb, "", "required "+p.Loc+" parameter is missing")
					}
					return
				}
				pv = binding.SetDeepObjectValue(pb, res, receiver, value)
			} else {
				value := binding.GetParamValue(*p, ctx, cookies)
				isSet = value != ""
				if value == "" {
					if !op.SkipValidateParams && p.Required {
						res.Add(pb, "", "required "+p.Loc+" parameter is missing")
					}
					return
				}
				var err error
				pv, err = binding.ParseInto(ctx, receiver, value, nil, *p, queryValues)
				if err != nil {
					res.Add(pb, value, err.Error())
					return
				}
			}

			if f.Addr().Type().Implements(reflect.TypeFor[binding.ParamReactor]()) {
				if pr, ok := f.Addr().Interface().(binding.ParamReactor); ok {
					pr.OnParamSet(isSet, pv)
				}
			}

			if !op.SkipValidateParams {
				schema.Validate(registry, p.Schema, pb, core.ModeWriteToServer, pv, res)
			}
		})

		if inputMeta.HasInputBody {
			if op.BodyReadTimeout > 0 {
				_ = ctx.SetReadDeadline(time.Now().Add(op.BodyReadTimeout))
			} else if op.BodyReadTimeout < 0 {
				_ = ctx.SetReadDeadline(time.Time{})
			}

			buf, _ := bufPool.Get().(*bytes.Buffer)
			bufCloser := func() {
				if buf.Cap() <= 1024*1024 {
					buf.Reset()
					bufPool.Put(buf)
				}
			}
			if cErr := binding.ReadBody(buf, ctx, op.MaxBodyBytes); cErr != nil {
				bufCloser()
				writeErrFromContext(api, ctx, cErr, *res)
				return
			}
			body := buf.Bytes()

			contentType := ctx.Header("Content-Type")
			if contentType == "" {
				if op.RequestBody != nil && op.RequestBody.Content != nil {
					if _, ok := op.RequestBody.Content["application/json"]; ok {
						contentType = "application/json"
					} else {
						for ct := range op.RequestBody.Content {
							contentType = ct
							break
						}
					}
				}
			}

			unmarshaler := func(data []byte, target any) error {
				return api.Unmarshal(contentType, data, target)
			}
			validator := func(data any, vres *core.ValidateResult) {
				pb.Reset()
				pb.Push("body")
				schema.Validate(registry, inputMeta.InSchema, pb, core.ModeWriteToServer, data, vres)
			}

			processErrStatus, cErr := binding.ProcessRegularMsgBody(binding.BodyProcessingConfig{
				Body:           body,
				Op:             op,
				Value:          v,
				HasInputBody:   inputMeta.HasInputBody,
				InputBodyIndex: inputMeta.InputBodyIndex,
				Unmarshaler:    unmarshaler,
				Validator:      validator,
				Defaults:       inputMeta.Defaults,
				Result:         res,
			})
			if processErrStatus > 0 {
				errStatus = processErrStatus
			}
			if cErr != nil {
				bufCloser()
				writeErrFromContext(api, ctx, cErr, *res)
				return
			}
			bufCloser()
		}

		inputMeta.Resolvers.EveryPB(pb, v, func(item reflect.Value, _ bool) {
			item = reflect.Indirect(item)
			if item.Kind() == reflect.Invalid {
				return
			}
			if item.CanAddr() {
				item = item.Addr()
			} else {
				ptr := reflect.New(item.Type())
				ptr.Elem().Set(item)
				item = ptr
			}
			var errs []error
			switch resolver := item.Interface().(type) {
			case core.Resolver:
				errs = resolver.Resolve(ctx)
			case core.ResolverWithPath:
				errs = resolver.Resolve(ctx, pb)
			default:
				panic("matched resolver cannot be run, please file a bug")
			}
			if len(errs) > 0 {
				res.Errors = append(res.Errors, errs...)
			}
		})

		if res.HasErrors() {
			for i := len(res.Errors) - 1; i >= 0; i-- {
				var s core.Error
			if errors.As(res.Errors[i], &s) {
					errStatus = s.StatusCode()
					break
				}
			}
			_ = WriteErr(api, ctx, errStatus, "validation failed", res.Errors...)
			return
		}

		output, err := handler(ctx.Context(), &input)
		if err != nil {
			var he core.Headerer
			if errors.As(err, &he) {
				for k, values := range he.GetHeaders() {
					for i, hv := range values {
						if i == 0 {
							ctx.SetHeader(k, hv)
						} else {
							ctx.AppendHeader(k, hv)
						}
					}
				}
			}

			status := http.StatusInternalServerError
			var se core.Error
			if errors.As(err, &se) {
				binding.WriteResponseWithPanic(api, ctx, se.StatusCode(), "", se)
				return
			}
			se = api.ErrorHandler().NewErrorWithContext(ctx, status, "unexpected error occurred", err)
			binding.WriteResponseWithPanic(api, ctx, se.StatusCode(), "", se)
			return
		}

		if output == nil {
			ctx.SetStatus(op.DefaultStatus)
			return
		}

		ct := ""
		vo := reflect.ValueOf(output).Elem()
		outMeta.Headers.Every(vo, func(f reflect.Value, info *binding.HeaderInfo) {
			f = reflect.Indirect(f)
			if f.Kind() == reflect.Invalid {
				return
			}
			if f.Kind() == reflect.Slice {
				for i := 0; i < f.Len(); i++ {
					binding.WriteHeader(ctx.AppendHeader, info, f.Index(i))
				}
			} else {
				if f.Kind() == reflect.String && info.Name == "Content-Type" {
					ct = f.String()
				}
				binding.WriteHeader(ctx.SetHeader, info, f)
			}
		})

		status := op.DefaultStatus
		if outMeta.StatusIndex != -1 {
			status = int(vo.Field(outMeta.StatusIndex).Int())
		}

		if outMeta.BodyIndex != -1 {
			body := vo.Field(outMeta.BodyIndex).Interface()

			if outMeta.BodyFunc {
				if fn, ok := body.(func(core.Context, core.API)); ok { fn(ctx, api) }
				return
			}

			if b, ok := body.([]byte); ok {
				ctx.SetStatus(status)
				_, _ = ctx.BodyWriter().Write(b)
				return
			}

			binding.WriteResponseWithPanic(api, ctx, status, ct, body)
		} else {
			ctx.SetStatus(status)
		}
	})))
}

func expandContentTypesFromAPI(a core.API, op *core.Operation) {
	type configProvider interface{ Config() core.Config }
	if cp, ok := a.(configProvider); ok {
		expandContentTypes(op, cp.Config().Formats)
	}
}

func expandContentTypes(op *core.Operation, formats map[string]core.Format) {
	if len(formats) <= 2 {
		return
	}

	// Collect full content type keys (skip shorthands like "json", "cbor").
	var formatKeys []string
	for k := range formats {
		if strings.Contains(k, "/") {
			formatKeys = append(formatKeys, k)
		}
	}

	if op.RequestBody != nil {
		for _, ct := range formatKeys {
			if _, exists := op.RequestBody.Content[ct]; exists {
				continue
			}
			if jsonMT := op.RequestBody.Content["application/json"]; jsonMT != nil {
				op.RequestBody.Content[ct] = &core.MediaType{
					Schema:  jsonMT.Schema,
					Example: jsonMT.Example,
				}
			}
		}
	}

	for _, resp := range op.Responses {
		if resp.Content == nil {
			continue
		}
		jsonMT := resp.Content["application/json"]
		if jsonMT == nil {
			continue
		}
		for _, ct := range formatKeys {
			if _, exists := resp.Content[ct]; exists {
				continue
			}
			// Skip error content types (application/problem+json etc.)
			if strings.Contains(ct, "problem") {
				continue
			}
			resp.Content[ct] = &core.MediaType{
				Schema:  jsonMT.Schema,
				Example: jsonMT.Example,
			}
		}
	}
}

func containsInt(s []int, v int) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func trackHiddenOp(a core.API, op *core.Operation) {
	if hop, ok := a.(core.HiddenOperationsProvider); ok {
		hop.AddHiddenOperation(op)
		return
	}
	// Try to find the underlying API through known wrapper patterns.
	// The middleware.Group and testAPI types embed core.API, which holds
	// the concrete *api value that implements HiddenOperationsProvider.
	// Use reflection to check for an embedded core.API field.
	v := reflect.ValueOf(a)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() == reflect.Struct {
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			if !f.CanInterface() {
				continue
			}
			if inner, ok := f.Interface().(core.HiddenOperationsProvider); ok {
				inner.AddHiddenOperation(op)
				return
			}
		}
	}
}

func readCookies(ctx core.Context) []*http.Cookie {
	var cookies []*http.Cookie
	ctx.EachHeader(func(name, value string) {
		if strings.EqualFold(name, "Cookie") {
			header := http.Header{}
			header.Add("Cookie", value)
			r := &http.Request{Header: header}
			cookies = append(cookies, r.Cookies()...)
		}
	})
	return cookies
}
