package middleware

import (
	"io"
	"maps"
	"strings"

	"github.com/MeGaNeKoS/neoma/core"
)

// OperationDocumenter is implemented by types that can register an operation
// into the OpenAPI documentation.
type OperationDocumenter interface {
	DocumentOperation(op *core.Operation)
}

// PrefixModifier returns an operation modifier that prepends each of the given
// path prefixes to the operation, duplicating the operation when multiple
// prefixes are provided.
func PrefixModifier(prefixes []string) func(o *core.Operation, next func(*core.Operation)) {
	return func(o *core.Operation, next func(*core.Operation)) {
		for _, prefix := range prefixes {
			modified := *o
			if len(prefixes) > 1 && prefix != "" {
				// If there are multiple prefixes, update the ID and tags so you
				// can differentiate between them in clients and the UI.
				friendlyPrefix := strings.ReplaceAll(strings.Trim(prefix, "/"), "/", "-")
				modified.OperationID = friendlyPrefix + "-" + modified.OperationID
				modified.Tags = append(append([]string{}, modified.Tags...), friendlyPrefix)
			}
			modified.Path = prefix + modified.Path
			next(&modified)
		}
	}
}

type groupAdapter struct {
	core.Adapter
	group *Group
}

func (a *groupAdapter) Handle(op *core.Operation, handler func(core.Context)) {
	a.group.modifyOperation(op, func(op *core.Operation) {
		a.Adapter.Handle(op, handler)
	})
}

// Group wraps a core.API to provide shared path prefixes, middleware,
// transformers, and operation modifiers for a set of routes.
type Group struct {
	core.API
	prefixes     []string
	adapter      core.Adapter
	modifiers    []func(o *core.Operation, next func(*core.Operation))
	middlewares  core.Middlewares
	transformers []core.Transformer
}

// NewGroup creates a new route group from the given API with optional path
// prefixes applied to all registered operations.
func NewGroup(api core.API, prefixes ...string) *Group {
	group := &Group{API: api, prefixes: prefixes}
	group.adapter = &groupAdapter{Adapter: api.Adapter(), group: group}
	if len(prefixes) > 0 {
		group.UseModifier(PrefixModifier(prefixes))
	}
	return group
}

// Adapter returns the group's adapter, which wraps the underlying API adapter
// to apply the group's modifiers during operation handling.
func (g *Group) Adapter() core.Adapter {
	return g.adapter
}

// DocumentOperation applies the group's modifiers to the operation and adds it
// to the OpenAPI documentation.
func (g *Group) DocumentOperation(op *core.Operation) {
	g.modifyOperation(op, func(op *core.Operation) {
		if documenter, ok := g.API.(OperationDocumenter); ok {
			documenter.DocumentOperation(op)
		} else {
			if op.Hidden {
				return
			}
			g.OpenAPI().AddOperation(op)
		}
	})
}

// UseModifier adds an operation modifier that can inspect and transform
// operations as they are registered, calling next to continue the chain.
func (g *Group) UseModifier(modifier func(o *core.Operation, next func(*core.Operation))) {
	g.modifiers = append(g.modifiers, modifier)
}

// UseSimpleModifier adds an operation modifier that transforms the operation
// without needing to call a next function.
func (g *Group) UseSimpleModifier(modifier func(o *core.Operation)) {
	g.modifiers = append(g.modifiers, func(o *core.Operation, next func(*core.Operation)) {
		modifier(o)
		next(o)
	})
}

func (g *Group) modifyOperation(op *core.Operation, next func(*core.Operation)) {
	chain := func(op *core.Operation) {
		// If this came from convenience functions, we may need to regenerate
		// the operation ID and summary as they are based on things like the
		// path which may have changed.
		if op.Metadata != nil {
			// Copy so we don't modify the original map.
			meta := make(map[string]any, len(op.Metadata))
			maps.Copy(meta, op.Metadata)
			op.Metadata = meta

			if genID, ok := op.Metadata["_convenience_id"].(string); ok && genID == op.OperationID {
				op.OperationID = g.GenerateOperationID(op.Method, op.Path)
				op.Metadata["_convenience_id"] = op.OperationID
			}
			if genSummary, ok := op.Metadata["_convenience_summary"].(string); ok && genSummary == op.Summary {
				op.Summary = generateGroupSummary(op.Method, op.Path)
				op.Metadata["_convenience_summary"] = op.Summary
			}
		}
		next(op)
	}
	for i := len(g.modifiers) - 1; i >= 0; i-- {
		func(i int, n func(*core.Operation)) {
			chain = func(op *core.Operation) { g.modifiers[i](op, n) }
		}(i, chain)
	}
	chain(op)
}

func generateGroupSummary(method, path string) string {
	clean := strings.NewReplacer("{", "", "}", "").Replace(path)
	parts := strings.Split(strings.Trim(clean, "/"), "/")
	if len(parts) == 0 {
		return method
	}
	last := parts[len(parts)-1]
	last = strings.ReplaceAll(last, "-", " ")
	return strings.ToLower(method) + " " + last
}

// UseMiddleware appends middleware functions to the group's middleware chain.
func (g *Group) UseMiddleware(middlewares ...core.MiddlewareFunc) {
	g.middlewares = append(g.middlewares, middlewares...)
}

// Middlewares returns the combined middleware chain of the parent API followed
// by this group's own middleware.
func (g *Group) Middlewares() core.Middlewares {
	m := append(core.Middlewares{}, g.API.Middlewares()...)
	return append(m, g.middlewares...)
}

// UseTransformer appends response transformers to the group's transformer
// chain.
func (g *Group) UseTransformer(transformers ...core.Transformer) {
	g.transformers = append(g.transformers, transformers...)
}

// Transform runs the parent API's transformers followed by the group's own
// transformers on the response value.
func (g *Group) Transform(ctx core.Context, status string, v any) (any, error) {
	v, err := g.API.Transform(ctx, status, v)
	if err != nil {
		return v, err
	}
	for _, transformer := range g.transformers {
		v, err = transformer(ctx, status, v)
		if err != nil {
			return v, err
		}
	}
	return v, nil
}

// Group creates a sub-group with the given path prefix.
func (g *Group) Group(prefix string) *Group {
	return NewGroup(g, prefix)
}

// UseDefaultTag sets a default tag for operations in this group. If an
// operation already specifies tags, they are left unchanged.
func (g *Group) UseDefaultTag(tag string) {
	if tag == "" {
		return
	}
	g.UseSimpleModifier(func(o *core.Operation) {
		if len(o.Tags) == 0 {
			o.Tags = []string{tag}
		}
	})
}

// UseDefaultSecurity sets a default security requirement for operations in
// this group. If an operation already specifies security, it is left unchanged.
func (g *Group) UseDefaultSecurity(scheme string) {
	g.UseSimpleModifier(func(o *core.Operation) {
		if len(o.Security) == 0 {
			o.Security = []map[string][]string{{scheme: {}}}
		}
	})
}

// WithSecurity registers a security scheme in the OpenAPI spec, applies the
// security requirement to all operations in this group, and adds the
// middleware to the group's chain. This is a one-call replacement for
// manually registering the scheme, calling UseDefaultSecurity, and
// UseMiddleware separately.
func (g *Group) WithSecurity(name string, scheme *core.SecurityScheme, fn core.MiddlewareFunc) {
	oapi := g.OpenAPI()
	if oapi.Components == nil {
		oapi.Components = &core.Components{}
	}
	if oapi.Components.SecuritySchemes == nil {
		oapi.Components.SecuritySchemes = map[string]*core.SecurityScheme{}
	}
	if _, exists := oapi.Components.SecuritySchemes[name]; !exists {
		oapi.Components.SecuritySchemes[name] = scheme
	}
	g.UseDefaultSecurity(name)
	g.UseMiddleware(fn)
}

// Config returns the configuration from the underlying API, if available.
func (g *Group) Config() core.Config {
	type configProvider interface{ Config() core.Config }
	if cp, ok := g.API.(configProvider); ok {
		return cp.Config()
	}
	return core.Config{}
}

// Negotiate delegates content negotiation to the underlying API.
func (g *Group) Negotiate(accept string) (string, error) {
	return g.API.Negotiate(accept)
}

// Marshal delegates response marshalling to the underlying API.
func (g *Group) Marshal(w io.Writer, contentType string, v any) error {
	return g.API.Marshal(w, contentType, v)
}

// Unmarshal delegates request unmarshalling to the underlying API.
func (g *Group) Unmarshal(contentType string, data []byte, v any) error {
	return g.API.Unmarshal(contentType, data, v)
}
