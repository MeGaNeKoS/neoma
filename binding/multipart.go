package binding

import (
	"net/http"
	"reflect"

	"github.com/MeGaNeKoS/neoma/core"
)

var (
	formFileType      = reflect.TypeFor[core.FormFile]()
	formFileSliceType = reflect.TypeFor[[]core.FormFile]()
)

type MultipartFormFiles[T any] struct {
	data T
}

func (m *MultipartFormFiles[T]) Data() T { return m.data }

type MultipartFieldInfo struct {
	Name    string
	Index   []int
	IsFile  bool
	IsSlice bool
	Type    reflect.Type
}

type MultipartProcessingConfig struct {
	Value  reflect.Value
	Fields []MultipartFieldInfo
}

func AnalyzeMultipartFields(t reflect.Type) []MultipartFieldInfo {
	t = core.Deref(t)
	if t.Kind() != reflect.Struct {
		return nil
	}

	var fields []MultipartFieldInfo
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}

		name := f.Tag.Get("form")
		if name == "" {
			continue
		}

		ft := f.Type
		info := MultipartFieldInfo{
			Name:  name,
			Index: f.Index,
			Type:  ft,
		}

		switch ft {
		case formFileType:
			info.IsFile = true
		case formFileSliceType:
			info.IsFile = true
			info.IsSlice = true
		}

		fields = append(fields, info)
	}
	return fields
}

func ProcessMultipartForm(ctx core.Context, cfg MultipartProcessingConfig) *ContextError {
	form, err := ctx.GetMultipartForm()
	if err != nil {
		return &ContextError{
			Code: http.StatusBadRequest,
			Msg:  "failed to parse multipart form",
			Errs: []error{err},
		}
	}
	if form == nil {
		return &ContextError{
			Code: http.StatusBadRequest,
			Msg:  "missing multipart form data",
		}
	}

	for _, fi := range cfg.Fields {
		field := cfg.Value.FieldByIndex(fi.Index)

		if fi.IsFile {
			fileHeaders := form.File[fi.Name]
			if len(fileHeaders) == 0 {
				continue
			}
			if fi.IsSlice {
				files := make([]core.FormFile, 0, len(fileHeaders))
				for _, fh := range fileHeaders {
					opened, openErr := fh.Open()
					if openErr != nil {
						return &ContextError{
							Code: http.StatusBadRequest,
							Msg:  "failed to open uploaded file",
							Errs: []error{openErr},
						}
					}
					files = append(files, core.FormFile{File: opened, FileHeader: fh})
				}
				field.Set(reflect.ValueOf(files))
			} else {
				fh := fileHeaders[0]
				opened, openErr := fh.Open()
				if openErr != nil {
					return &ContextError{
						Code: http.StatusBadRequest,
						Msg:  "failed to open uploaded file",
						Errs: []error{openErr},
					}
				}
				field.Set(reflect.ValueOf(core.FormFile{File: opened, FileHeader: fh}))
			}
			continue
		}

		values := form.Value[fi.Name]
		if len(values) == 0 {
			continue
		}

		target := field
		if field.CanAddr() && field.Addr().Type().Implements(paramWrapperType) {
			if pw, ok := field.Addr().Interface().(ParamWrapper); ok {
				target = pw.Receiver()
			}
		}

		if err := setFieldValue(target, values[0]); err != nil {
			return &ContextError{
				Code: http.StatusBadRequest,
				Msg:  "invalid form field value for " + fi.Name,
				Errs: []error{err},
			}
		}
	}
	return nil
}

func SetupMultipartRequestBody(op *core.Operation, fields []MultipartFieldInfo) {
	initRequestBody(op)

	mt := op.RequestBody.Content["multipart/form-data"]
	if mt == nil {
		mt = &core.MediaType{}
		op.RequestBody.Content["multipart/form-data"] = mt
	}

	if mt.Schema == nil {
		mt.Schema = &core.Schema{
			Type:       core.TypeObject,
			Properties: map[string]*core.Schema{},
		}
	}

	for _, fi := range fields {
		if _, exists := mt.Schema.Properties[fi.Name]; exists {
			continue
		}
		if fi.IsFile {
			if fi.IsSlice {
				mt.Schema.Properties[fi.Name] = &core.Schema{
					Type:  core.TypeArray,
					Items: &core.Schema{Type: core.TypeString, Format: "binary"},
				}
			} else {
				mt.Schema.Properties[fi.Name] = &core.Schema{
					Type:   core.TypeString,
					Format: "binary",
				}
			}
		} else {
			mt.Schema.Properties[fi.Name] = schemaForType(fi.Type)
		}
	}
}

func schemaForType(t reflect.Type) *core.Schema {
	t = core.Deref(t)
	switch t.Kind() {
	case reflect.String:
		return &core.Schema{Type: core.TypeString}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &core.Schema{Type: core.TypeInteger}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &core.Schema{Type: core.TypeInteger}
	case reflect.Float32, reflect.Float64:
		return &core.Schema{Type: core.TypeNumber}
	case reflect.Bool:
		return &core.Schema{Type: core.TypeBoolean}
	default:
		return &core.Schema{Type: core.TypeString}
	}
}

func HasMultipartFields(t reflect.Type) bool {
	t = core.Deref(t)
	if t.Kind() != reflect.Struct {
		return false
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		if f.Tag.Get("form") != "" {
			ft := core.Deref(f.Type)
			if ft == formFileType || ft == formFileSliceType {
				return true
			}
		}
	}
	return false
}

func IsMultipartFormFilesType(t reflect.Type) bool {
	t = core.Deref(t)
	if t.Kind() != reflect.Struct || t.NumField() == 0 {
		return false
	}
	return t.Name() == "MultipartFormFiles" && t.PkgPath() == reflect.TypeFor[MultipartFormFiles[struct{}]]().PkgPath()
}

