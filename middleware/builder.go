package middleware

import "github.com/MeGaNeKoS/neoma/core"

// Builder creates operation-specific middleware by inspecting the operation
// definition at registration time.
type Builder interface {
	Build(op *core.Operation) core.MiddlewareFunc
}

// BuilderFunc is an adapter that allows ordinary functions to be used as a
// Builder.
type BuilderFunc func(op *core.Operation) core.MiddlewareFunc

// Build calls the wrapped function to produce a middleware for the given
// operation.
func (f BuilderFunc) Build(op *core.Operation) core.MiddlewareFunc {
	return f(op)
}

// Build is a convenience function that calls builder.Build for the given
// operation.
func Build(builder Builder, op *core.Operation) core.MiddlewareFunc {
	return builder.Build(op)
}

// NewBuilderModifier returns an operation modifier that prepends middleware
// produced by the given Builder to each operation's middleware chain.
func NewBuilderModifier(builder Builder) func(op *core.Operation, next func(*core.Operation)) {
	return func(op *core.Operation, next func(*core.Operation)) {
		mw := builder.Build(op)
		if mw != nil {
			op.Middlewares = append(core.Middlewares{mw}, op.Middlewares...)
		}
		next(op)
	}
}
