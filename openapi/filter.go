package openapi

import (
	"slices"

	"github.com/MeGaNeKoS/neoma/core"
)


func FilterByTag(oapi *core.OpenAPI, tag string) *core.OpenAPI {
	out := cloneOpenAPI(oapi)
	for path, pi := range oapi.Paths {
		cp := clonePathItem(pi)
		if filterOps(cp, func(op *core.Operation) bool {
			return slices.Contains(op.Tags, tag)
		}) {
			out.Paths[path] = cp
		}
	}
	return out
}

func FilterExcludeTag(oapi *core.OpenAPI, tag string) *core.OpenAPI {
	out := cloneOpenAPI(oapi)
	for path, pi := range oapi.Paths {
		cp := clonePathItem(pi)
		if filterOps(cp, func(op *core.Operation) bool {
			return !slices.Contains(op.Tags, tag)
		}) {
			out.Paths[path] = cp
		}
	}
	return out
}


func cloneOpenAPI(oapi *core.OpenAPI) *core.OpenAPI {
	out := *oapi
	out.Paths = make(map[string]*core.PathItem, len(oapi.Paths))
	// Shallow copy everything else (tags, components, etc.)
	if oapi.Tags != nil {
		out.Tags = make([]*core.Tag, len(oapi.Tags))
		copy(out.Tags, oapi.Tags)
	}
	return &out
}

func clonePathItem(pi *core.PathItem) *core.PathItem {
	cp := *pi
	return &cp
}

func filterOps(pi *core.PathItem, pred func(*core.Operation) bool) bool {
	kept := false
	for _, pair := range []struct {
		op **core.Operation
	}{
		{&pi.Get},
		{&pi.Put},
		{&pi.Post},
		{&pi.Delete},
		{&pi.Options},
		{&pi.Head},
		{&pi.Patch},
		{&pi.Trace},
	} {
		if *pair.op == nil {
			continue
		}
		if pred(*pair.op) {
			kept = true
		} else {
			*pair.op = nil
		}
	}
	return kept
}
