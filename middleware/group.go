package middleware

import (
	"io"
	"maps"
	"strings"

	"github.com/MeGaNeKoS/neoma/core"
)

type OperationDocumenter interface {
	DocumentOperation(op *core.Operation)
}

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

type Group struct {
	core.API
	prefixes     []string
	adapter      core.Adapter
	modifiers    []func(o *core.Operation, next func(*core.Operation))
	middlewares  core.Middlewares
	transformers []core.Transformer
}

func NewGroup(api core.API, prefixes ...string) *Group {
	group := &Group{API: api, prefixes: prefixes}
	group.adapter = &groupAdapter{Adapter: api.Adapter(), group: group}
	if len(prefixes) > 0 {
		group.UseModifier(PrefixModifier(prefixes))
	}
	return group
}

func (g *Group) Adapter() core.Adapter {
	return g.adapter
}

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

func (g *Group) UseModifier(modifier func(o *core.Operation, next func(*core.Operation))) {
	g.modifiers = append(g.modifiers, modifier)
}

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

func (g *Group) UseMiddleware(middlewares ...core.MiddlewareFunc) {
	g.middlewares = append(g.middlewares, middlewares...)
}

func (g *Group) Middlewares() core.Middlewares {
	m := append(core.Middlewares{}, g.API.Middlewares()...)
	return append(m, g.middlewares...)
}

func (g *Group) UseTransformer(transformers ...core.Transformer) {
	g.transformers = append(g.transformers, transformers...)
}

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

func (g *Group) Group(prefix string) *Group {
	return NewGroup(g, prefix)
}

func (g *Group) Config() core.Config {
	type configProvider interface{ Config() core.Config }
	if cp, ok := g.API.(configProvider); ok {
		return cp.Config()
	}
	return core.Config{}
}

func (g *Group) Negotiate(accept string) (string, error) {
	return g.API.Negotiate(accept)
}

func (g *Group) Marshal(w io.Writer, contentType string, v any) error {
	return g.API.Marshal(w, contentType, v)
}

func (g *Group) Unmarshal(contentType string, data []byte, v any) error {
	return g.API.Unmarshal(contentType, data, v)
}
