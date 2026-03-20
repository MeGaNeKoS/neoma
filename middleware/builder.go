package middleware

import "github.com/MeGaNeKoS/neoma/core"

type Builder interface {
	Build(op *core.Operation) core.MiddlewareFunc
}

type BuilderFunc func(op *core.Operation) core.MiddlewareFunc

func (f BuilderFunc) Build(op *core.Operation) core.MiddlewareFunc {
	return f(op)
}

func Build(builder Builder, op *core.Operation) core.MiddlewareFunc {
	return builder.Build(op)
}

func NewBuilderModifier(builder Builder) func(op *core.Operation, next func(*core.Operation)) {
	return func(op *core.Operation, next func(*core.Operation)) {
		mw := builder.Build(op)
		if mw != nil {
			op.Middlewares = append(core.Middlewares{mw}, op.Middlewares...)
		}
		next(op)
	}
}
