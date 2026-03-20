package openapi

import (
	"bytes"
	"encoding/json"
	"reflect"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/yaml"
)


func Downgrade(oapi *core.OpenAPI) ([]byte, error) {
	b, err := oapi.MarshalJSON()
	if err != nil {
		return b, err
	}
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	downgradeSpec(v)
	return json.Marshal(v)
}

func DowngradeYAML(oapi *core.OpenAPI) ([]byte, error) {
	specJSON, err := Downgrade(oapi)
	buf := bytes.NewBuffer([]byte{})
	if err == nil {
		err = yaml.Convert(buf, bytes.NewReader(specJSON))
	}
	return buf.Bytes(), err
}

func YAML(oapi *core.OpenAPI) ([]byte, error) {
	specJSON, err := json.Marshal(oapi)
	buf := bytes.NewBuffer([]byte{})
	if err == nil {
		err = yaml.Convert(buf, bytes.NewReader(specJSON))
	}
	return buf.Bytes(), err
}


func downgradeSpec(input any) {
	if input == nil {
		return
	}
	switch value := input.(type) {
	case map[string]any:
		m := value
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		for _, k := range keys {
			v := m[k]
			if k == "openapi" && (v == core.OpenAPIVersion31 || v == core.OpenAPIVersion32) {
				m[k] = core.OpenAPIVersion30
				continue
			}

			if k == "type" {
				// OpenAPI 3.1 supports type arrays, which need to be converted.
				// This may be lossy, but we want to keep it simple.
				if types, ok := v.([]any); ok {
					for _, t := range types {
						if t == "null" {
							// The "null" type is a nullable field in 3.0.
							m["nullable"] = true
						} else {
							// Last non-null wins.
							m["type"] = t
						}
					}
					continue
				}
			}

			// Exclusive values were bools in 3.0.
			if k == "exclusiveMinimum" && reflect.TypeOf(v).Kind() == reflect.Float64 {
				m["minimum"] = v
				m["exclusiveMinimum"] = true
				continue
			}

			if k == "exclusiveMaximum" && reflect.TypeOf(v).Kind() == reflect.Float64 {
				m["maximum"] = v
				m["exclusiveMaximum"] = true
				continue
			}

			// Provide single example for tools that read it.
			if k == "examples" {
				if examples, ok := v.([]any); ok {
					if len(examples) > 0 {
						m["example"] = examples[0]
					}
					if len(examples) == 1 {
						delete(m, k)
					}
					continue
				}
			}

			// Base64 / binary uploads.
			if k == "application/octet-stream" {
				if ct, ok := v.(map[string]any); ok && len(ct) == 0 {
					m[k] = map[string]any{
						"schema": map[string]any{
							"type":   "string",
							"format": "binary",
						},
					}
				}
			}

			if k == "contentEncoding" && v == "base64" {
				delete(m, k)
				m["format"] = "base64"
				continue
			}

			if k == "contentEncoding" || k == "contentMediaType" {
				m["x-"+k] = m[k]
				delete(m, k)
			}

			downgradeSpec(v)
		}
	case []any:
		for _, item := range value {
			downgradeSpec(item)
		}
	}
}
