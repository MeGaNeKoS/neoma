package openapi

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"reflect"
	"strings"
	"sync"

	"github.com/MeGaNeKoS/neoma/core"
)

type schemaField struct {
	Schema string `json:"$schema"`
}

type schemaTypeInfo struct {
	t      reflect.Type
	fields []int
	ref    string
	header string
}

// SchemaLinkTransformer adds a Link header with rel="describedBy" and a
// $schema field to JSON response bodies, pointing to the JSON Schema that
// describes the response structure (RFC 8288).
type SchemaLinkTransformer struct {
	schemasPath string
	types       map[reflect.Type]schemaTypeInfo
}

var bufPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

// NewSchemaLinkTransformer creates a transformer that adds a Link header with
// rel="describedBy" and a $schema field to response bodies.
func NewSchemaLinkTransformer(schemasPath string) *SchemaLinkTransformer {
	return &SchemaLinkTransformer{
		schemasPath: schemasPath,
		types:       make(map[reflect.Type]schemaTypeInfo),
	}
}

func (t *SchemaLinkTransformer) addSchemaField(oapi *core.OpenAPI, content *core.MediaType) bool {
	if content == nil || content.Schema == nil || content.Schema.Ref == "" || !strings.HasPrefix(content.Schema.Ref, "#/") {
		return true
	}

	schema := oapi.Components.Schemas.SchemaFromRef(content.Schema.Ref)
	if schema == nil || schema.Type != core.TypeObject || (schema.Properties != nil && schema.Properties["$schema"] != nil) {
		return true
	}

	// Create an example so it's easier for users to find the schema URL when
	// they are reading the documentation.
	server := "https://example.com"
	for _, s := range oapi.Servers {
		if s.URL != "" {
			server = s.URL
			break
		}
	}

	if schema.Properties == nil {
		schema.Properties = map[string]*core.Schema{}
	}

	schema.Properties["$schema"] = &core.Schema{
		Type:        core.TypeString,
		Format:      "uri",
		Description: "A URL to the JSON Schema for this object.",
		ReadOnly:    true,
		Examples:    []any{server + t.schemasPath + "/" + path.Base(content.Schema.Ref) + ".json"},
	}
	return false
}

// OnAddOperation is triggered whenever a new operation is added to the API,
// enabling this transformer to precompute and cache information about the
// response and schema.
func (t *SchemaLinkTransformer) OnAddOperation(oapi *core.OpenAPI, op *core.Operation) {
	if op.RequestBody != nil && op.RequestBody.Content != nil {
		for _, content := range op.RequestBody.Content {
			t.addSchemaField(oapi, content)
		}
	}

	schemasPath := t.schemasPath
	if prefix := getAPIPrefix(oapi); prefix != "" {
		schemasPath = path.Join(prefix, schemasPath)
	}

	registry := oapi.Components.Schemas
	for _, resp := range op.Responses {
		for _, content := range resp.Content {
			if t.addSchemaField(oapi, content) {
				continue
			}

			typ := core.Deref(registry.TypeFromRef(content.Schema.Ref))

			extra := schemaField{
				Schema: schemasPath + "/" + path.Base(content.Schema.Ref) + ".json",
			}

			fieldIndexes := []int{}
			fields := []reflect.StructField{
				reflect.TypeOf(extra).Field(0),
			}
			for i := 0; i < typ.NumField(); i++ {
				f := typ.Field(i)
				if f.IsExported() {
					fields = append(fields, f)
					fieldIndexes = append(fieldIndexes, i)
				}
			}

			func() {
				defer func() {
					if r := recover(); r != nil {
						fmt.Fprintln(os.Stderr, "Warning: unable to create schema link for type", typ, ":", r)
					}
				}()
				newType := reflect.StructOf(fields)
				t.types[typ] = schemaTypeInfo{
					t:      newType,
					fields: fieldIndexes,
					ref:    extra.Schema,
					header: "<" + extra.Schema + ">; rel=\"describedBy\"",
				}
			}()
		}
	}
}

// Transform adds the Link header and $schema field to the response.
func (t *SchemaLinkTransformer) Transform(ctx core.Context, status string, v any) (any, error) {
	if v == nil {
		return v, nil
	}

	vv := reflect.ValueOf(v)
	if vv.Kind() == reflect.Pointer && vv.IsNil() {
		return v, nil
	}

	typ := core.Deref(reflect.TypeOf(v))
	if typ.Kind() != reflect.Struct {
		return v, nil
	}

	info, ok := t.types[typ]
	if !ok || info.t == nil {
		return v, nil
	}

	host := ctx.Host()
	ctx.AppendHeader("Link", info.header)

	tmp := reflect.New(info.t).Elem()

	// Set the $schema field.
	buf := bufPool.Get().(*bytes.Buffer)
	if len(host) >= 9 && (host[:9] == "localhost" || host[:9] == "127.0.0.1") {
		buf.WriteString("http://")
	} else {
		buf.WriteString("https://")
	}
	buf.WriteString(host)
	buf.WriteString(info.ref)
	tmp.Field(0).SetString(buf.String())
	buf.Reset()
	bufPool.Put(buf)

	// Copy over all exported fields.
	vv = reflect.Indirect(vv)
	for i, j := range info.fields {
		tmp.Field(i + 1).Set(vv.Field(j))
	}

	return tmp.Addr().Interface(), nil
}
