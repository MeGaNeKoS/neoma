// Package patch implements RFC 6902 (JSON Patch) and RFC 7396 (JSON Merge
// Patch) for use in HTTP PATCH handlers per RFC 5789.
package patch

import (
	"encoding/json"
	"errors"

	"github.com/MeGaNeKoS/neoma/core"
	jsonpatch "github.com/evanphx/json-patch/v5"
)

const (
	ContentTypeMergePatch = "application/merge-patch+json" // RFC 7396
	ContentTypeJSONPatch  = "application/json-patch+json"  // RFC 6902
)

// AcceptPatch is the value for the Accept-Patch response header (RFC 5789 Section 3.1).
const AcceptPatch = ContentTypeMergePatch + ", " + ContentTypeJSONPatch

var (
	// ErrUnsupportedContentType should map to 415 with an Accept-Patch header.
	ErrUnsupportedContentType = errors.New("unsupported patch content type: use application/merge-patch+json (RFC 7396) or application/json-patch+json (RFC 6902)")

	// ErrInvalidPatch should map to 422 Unprocessable Entity.
	ErrInvalidPatch = errors.New("invalid patch document")
)

// Apply applies a patch to a raw JSON document. Both formats are applied
// atomically (RFC 5789 Section 2).
func Apply(contentType string, original, patchData []byte) ([]byte, error) {
	switch contentType {
	case ContentTypeMergePatch:
		result, err := jsonpatch.MergePatch(original, patchData)
		if err != nil {
			return nil, errors.Join(ErrInvalidPatch, err)
		}
		return result, nil
	case ContentTypeJSONPatch:
		p, err := jsonpatch.DecodePatch(patchData)
		if err != nil {
			return nil, errors.Join(ErrInvalidPatch, err)
		}
		result, err := p.Apply(original)
		if err != nil {
			return nil, errors.Join(ErrInvalidPatch, err)
		}
		return result, nil
	default:
		return nil, ErrUnsupportedContentType
	}
}

// ApplyTo applies a patch directly to a Go struct. On error the target is
// left unchanged.
func ApplyTo[T any](contentType string, target *T, patchData []byte) error {
	original, err := json.Marshal(target)
	if err != nil {
		return errors.Join(ErrInvalidPatch, err)
	}

	patched, err := Apply(contentType, original, patchData)
	if err != nil {
		return err
	}

	// Unmarshal into a zero value so deleted fields (RFC 7396 null) are
	// properly zeroed instead of left stale.
	var result T
	if err := json.Unmarshal(patched, &result); err != nil {
		return err
	}
	*target = result
	return nil
}

// Equal reports whether two JSON documents are semantically equal.
func Equal(a, b []byte) bool {
	return jsonpatch.Equal(a, b)
}

// MakeOptionalSchema deep-copies the schema with all Required fields removed.
// Useful for merge-patch request body schemas where every field is optional.
func MakeOptionalSchema(s *core.Schema) *core.Schema {
	if s == nil {
		return nil
	}

	out := &core.Schema{
		Type:                 s.Type,
		Title:                s.Title,
		Description:          s.Description,
		Format:               s.Format,
		ContentEncoding:      s.ContentEncoding,
		Default:              s.Default,
		Examples:             s.Examples,
		AdditionalProperties: s.AdditionalProperties,
		Enum:                 s.Enum,
		Minimum:              s.Minimum,
		ExclusiveMinimum:     s.ExclusiveMinimum,
		Maximum:              s.Maximum,
		ExclusiveMaximum:     s.ExclusiveMaximum,
		MultipleOf:           s.MultipleOf,
		MinLength:            s.MinLength,
		MaxLength:            s.MaxLength,
		Pattern:              s.Pattern,
		PatternDescription:   s.PatternDescription,
		MinItems:             s.MinItems,
		MaxItems:             s.MaxItems,
		UniqueItems:          s.UniqueItems,
		MinProperties:        s.MinProperties,
		MaxProperties:        s.MaxProperties,
		ReadOnly:             s.ReadOnly,
		WriteOnly:            s.WriteOnly,
		Deprecated:           s.Deprecated,
		Extensions:           s.Extensions,
		DependentRequired:    s.DependentRequired,
		Discriminator:        s.Discriminator,
	}

	if s.Items != nil {
		out.Items = MakeOptionalSchema(s.Items)
	}
	if s.Properties != nil {
		out.Properties = make(map[string]*core.Schema, len(s.Properties))
		for k, v := range s.Properties {
			out.Properties[k] = MakeOptionalSchema(v)
		}
	}
	if s.OneOf != nil {
		out.OneOf = make([]*core.Schema, len(s.OneOf))
		for i, v := range s.OneOf {
			out.OneOf[i] = MakeOptionalSchema(v)
		}
	}
	if s.AnyOf != nil {
		out.AnyOf = make([]*core.Schema, len(s.AnyOf))
		for i, v := range s.AnyOf {
			out.AnyOf[i] = MakeOptionalSchema(v)
		}
	}
	if s.AllOf != nil {
		out.AllOf = make([]*core.Schema, len(s.AllOf))
		for i, v := range s.AllOf {
			out.AllOf[i] = MakeOptionalSchema(v)
		}
	}
	if s.Not != nil {
		out.Not = MakeOptionalSchema(s.Not)
	}

	out.Required = nil
	return out
}
