package patch

import (
	"encoding/json"
	"errors"
	"strings"
)

type OpType string

const (
	OpAdd     OpType = "add"
	OpRemove  OpType = "remove"
	OpReplace OpType = "replace"
	OpMove    OpType = "move"  // RFC 6902 only
	OpCopy    OpType = "copy"  // RFC 6902 only
	OpTest    OpType = "test"  // RFC 6902 only
)

// Operation is a single parsed patch instruction.
type Operation struct {
	Op    OpType
	Path  []string // Target path segments, e.g. ["address", "zip"]
	From  []string // Source path for move/copy (RFC 6902 only)
	Value any      // Value to set/add/test. Nil for remove.
}

// Parse parses a patch document into operations without applying them.
// For RFC 6902: operations map directly from the patch array.
// For RFC 7396: the patch object is walked recursively, producing OpReplace
// for non-null leaves and OpRemove for null values.
func Parse(contentType string, patchData []byte) ([]Operation, error) {
	switch contentType {
	case ContentTypeMergePatch:
		return parseMergePatch(patchData)
	case ContentTypeJSONPatch:
		return parseJSONPatch(patchData)
	default:
		return nil, ErrUnsupportedContentType
	}
}

func parseMergePatch(patchData []byte) ([]Operation, error) {
	var raw any
	if err := json.Unmarshal(patchData, &raw); err != nil {
		return nil, errors.Join(ErrInvalidPatch, err)
	}

	obj, ok := raw.(map[string]any)
	if !ok {
		return []Operation{{Op: OpReplace, Path: nil, Value: raw}}, nil
	}

	var ops []Operation
	walkMergePatch(obj, nil, &ops)
	return ops, nil
}

func walkMergePatch(obj map[string]any, prefix []string, ops *[]Operation) {
	for key, value := range obj {
		path := append(append([]string{}, prefix...), key)

		if value == nil {
			*ops = append(*ops, Operation{Op: OpRemove, Path: path})
			continue
		}
		if nested, ok := value.(map[string]any); ok {
			walkMergePatch(nested, path, ops)
			continue
		}
		*ops = append(*ops, Operation{Op: OpReplace, Path: path, Value: value})
	}
}

type jsonPatchRaw struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	From  string `json:"from"`
	Value any    `json:"value"`
}

func parseJSONPatch(patchData []byte) ([]Operation, error) {
	var raw []jsonPatchRaw
	if err := json.Unmarshal(patchData, &raw); err != nil {
		return nil, errors.Join(ErrInvalidPatch, err)
	}

	ops := make([]Operation, 0, len(raw))
	for _, r := range raw {
		op := Operation{
			Op:    OpType(r.Op),
			Path:  parseJSONPointer(r.Path),
			Value: r.Value,
		}
		if r.From != "" {
			op.From = parseJSONPointer(r.From)
		}
		ops = append(ops, op)
	}
	return ops, nil
}

// parseJSONPointer converts an RFC 6901 JSON Pointer into path segments.
func parseJSONPointer(pointer string) []string {
	if pointer == "" || pointer == "/" {
		return nil
	}
	pointer = strings.TrimPrefix(pointer, "/")
	parts := strings.Split(pointer, "/")
	for i, p := range parts {
		// RFC 6901: decode ~1 before ~0.
		p = strings.ReplaceAll(p, "~1", "/")
		p = strings.ReplaceAll(p, "~0", "~")
		parts[i] = p
	}
	return parts
}
