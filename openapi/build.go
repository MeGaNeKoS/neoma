package openapi

import (
	"net/http"
	"strconv"

	"github.com/MeGaNeKoS/neoma/core"
)

// DefineErrors populates the error responses on an operation based on its declared
// error status codes, the error schema from the factory, and any discovered errors.
func DefineErrors(op *core.Operation, registry core.Registry, factory core.ErrorHandler, discovered ...[]core.DiscoveredError) {
	// Get the error schema from the factory. If nil, skip entirely.
	// This supports factories that want no error documentation (solves #880).
	errSchema := factory.ErrorSchema(registry)
	if errSchema == nil {
		return
	}

	errContentType := factory.ErrorContentType("application/json")

	if op.Responses == nil {
		op.Responses = map[string]*core.Response{}
	}

	var discoveredByStatus map[int][]core.DiscoveredError
	if len(discovered) > 0 && len(discovered[0]) > 0 {
		discoveredByStatus = make(map[int][]core.DiscoveredError)
		for _, de := range discovered[0] {
			discoveredByStatus[de.Status] = append(discoveredByStatus[de.Status], de)
		}
	}

	for _, code := range op.Errors {
		statusStr := strconv.Itoa(code)

		if _, exists := op.Responses[statusStr]; exists {
			continue
		}

		resp := buildErrorResponse(code, errSchema, errContentType, op)

		mt := resp.Content[errContentType]
		if mt == nil {
			continue
		}
		if manual, ok := op.ErrorExamples[code]; ok {
			mt.Example = manual
		} else if des, ok := discoveredByStatus[code]; ok && len(des) > 0 {
			applyDiscoveredExamples(mt, des, factory)
		} else {
			if example := factory.NewError(code, http.StatusText(code)); example != nil {
				mt.Example = example
			}
		}

		op.Responses[statusStr] = resp
	}

	// If there are no explicit error codes and at most one success response,
	// set a default error response so consumers know errors are possible.
	if len(op.Responses) <= 1 && len(op.Errors) == 0 {
		resp := &core.Response{
			Description: "Error",
			Content: map[string]*core.MediaType{
				errContentType: {
					Schema: errSchema,
				},
			},
		}
		mergeErrorHeaders(resp, op.ErrorHeaders)
		op.Responses["default"] = resp
	}
}

func applyDiscoveredExamples(mt *core.MediaType, des []core.DiscoveredError, factory core.ErrorHandler) {
	if mt == nil {
		return
	}
	if len(des) == 1 {
		de := des[0]
		mt.Example = factory.NewError(de.Status, de.Detail)
		return
	}

	mt.Examples = make(map[string]*core.Example, len(des))
	for _, de := range des {
		name := de.Title
		if name == "" {
			name = http.StatusText(de.Status)
		}
		if _, exists := mt.Examples[name]; exists {
			name = name + " - " + de.Detail
		}
		mt.Examples[name] = &core.Example{
			Summary: de.Detail,
			Value:   factory.NewError(de.Status, de.Detail),
		}
	}
}

func buildErrorResponse(
	code int,
	defaultSchema *core.Schema,
	errContentType string,
	op *core.Operation,
) *core.Response {
	errSchema := defaultSchema
	description := http.StatusText(code)

	var statusHeaders map[string]*core.Param

	// Apply per-status overrides from ErrorResponses (solves #878).
	if op.ErrorResponses != nil {
		if erc, ok := op.ErrorResponses[code]; ok && erc != nil {
			if erc.Description != "" {
				description = erc.Description
			}
			if erc.Schema != nil {
				errSchema = erc.Schema
			}
			if erc.Headers != nil {
				statusHeaders = erc.Headers
			}
		}
	}

	resp := &core.Response{
		Description: description,
		Content: map[string]*core.MediaType{
			errContentType: {
				Schema: errSchema,
			},
		},
	}

	mergeErrorHeaders(resp, statusHeaders)
	mergeErrorHeaders(resp, op.ErrorHeaders)

	return resp
}

func mergeErrorHeaders(resp *core.Response, headers map[string]*core.Param) {
	if len(headers) == 0 {
		return
	}
	if resp.Headers == nil {
		resp.Headers = make(map[string]*core.Param, len(headers))
	}
	for name, param := range headers {
		if _, exists := resp.Headers[name]; !exists {
			resp.Headers[name] = param
		}
	}
}
