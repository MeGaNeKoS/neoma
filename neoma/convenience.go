package neoma

import (
	"context"
	"net/http"

	"github.com/MeGaNeKoS/neoma/core"
)

func Delete[I, O any](api core.API, path string, handler func(context.Context, *I) (*O, error), operationHandlers ...func(o *core.Operation)) {
	convenience(api, http.MethodDelete, path, handler, operationHandlers...)
}

func Get[I, O any](api core.API, path string, handler func(context.Context, *I) (*O, error), operationHandlers ...func(o *core.Operation)) {
	convenience(api, http.MethodGet, path, handler, operationHandlers...)
}

func Head[I, O any](api core.API, path string, handler func(context.Context, *I) (*O, error), operationHandlers ...func(o *core.Operation)) {
	convenience(api, http.MethodHead, path, handler, operationHandlers...)
}

func OperationTags(tags ...string) func(o *core.Operation) {
	return func(o *core.Operation) {
		o.Tags = tags
	}
}

func Patch[I, O any](api core.API, path string, handler func(context.Context, *I) (*O, error), operationHandlers ...func(o *core.Operation)) {
	convenience(api, http.MethodPatch, path, handler, operationHandlers...)
}

func Post[I, O any](api core.API, path string, handler func(context.Context, *I) (*O, error), operationHandlers ...func(o *core.Operation)) {
	convenience(api, http.MethodPost, path, handler, operationHandlers...)
}

func Put[I, O any](api core.API, path string, handler func(context.Context, *I) (*O, error), operationHandlers ...func(o *core.Operation)) {
	convenience(api, http.MethodPut, path, handler, operationHandlers...)
}

func convenience[I, O any](api core.API, method, path string, handler func(context.Context, *I) (*O, error), operationHandlers ...func(o *core.Operation)) {
	var o *O
	opID := generateConvenienceOperationID(api, method, path, o)
	opSummary := generateConvenienceSummary(method, path, o)
	operation := core.Operation{
		OperationID: opID,
		Summary:     opSummary,
		Method:      method,
		Path:        path,
		Metadata:    map[string]any{},
	}
	for _, oh := range operationHandlers {
		oh(&operation)
	}
	// If not modified by operationHandlers, store hints that these were
	// auto-generated so groups can regenerate them when the path changes.
	if operation.OperationID == opID {
		operation.Metadata["_convenience_id"] = opID
		operation.Metadata["_convenience_id_out"] = o
	}
	if operation.Summary == opSummary {
		operation.Metadata["_convenience_summary"] = opSummary
		operation.Metadata["_convenience_summary_out"] = o
	}
	Register(api, operation, handler)
}
