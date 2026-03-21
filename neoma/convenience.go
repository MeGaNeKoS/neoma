package neoma

import (
	"context"
	"net/http"

	"github.com/MeGaNeKoS/neoma/core"
)

// Delete registers a handler for the HTTP DELETE method on the given API.
func Delete[I, O any](api core.API, path string, handler func(context.Context, *I) (*O, error), operationHandlers ...func(o *core.Operation)) {
	convenience(api, http.MethodDelete, path, handler, operationHandlers...)
}

// Get registers a handler for the HTTP GET method on the given API.
func Get[I, O any](api core.API, path string, handler func(context.Context, *I) (*O, error), operationHandlers ...func(o *core.Operation)) {
	convenience(api, http.MethodGet, path, handler, operationHandlers...)
}

// Head registers a handler for the HTTP HEAD method on the given API.
func Head[I, O any](api core.API, path string, handler func(context.Context, *I) (*O, error), operationHandlers ...func(o *core.Operation)) {
	convenience(api, http.MethodHead, path, handler, operationHandlers...)
}

// OperationTags returns an operation handler that sets the given tags on the
// operation, for use with the convenience registration functions.
func OperationTags(tags ...string) func(o *core.Operation) {
	return func(o *core.Operation) {
		o.Tags = tags
	}
}

// Patch registers a handler for the HTTP PATCH method on the given API.
func Patch[I, O any](api core.API, path string, handler func(context.Context, *I) (*O, error), operationHandlers ...func(o *core.Operation)) {
	convenience(api, http.MethodPatch, path, handler, operationHandlers...)
}

// Post registers a handler for the HTTP POST method on the given API.
func Post[I, O any](api core.API, path string, handler func(context.Context, *I) (*O, error), operationHandlers ...func(o *core.Operation)) {
	convenience(api, http.MethodPost, path, handler, operationHandlers...)
}

// Put registers a handler for the HTTP PUT method on the given API.
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
