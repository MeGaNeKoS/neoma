package middleware

import "github.com/MeGaNeKoS/neoma/core"

func TestMiddleware(mw core.MiddlewareFunc, handler func(core.Context)) func(core.Context) {
	return func(ctx core.Context) {
		mw(ctx, handler)
	}
}

func TestChain(middlewares core.Middlewares, handler func(core.Context)) func(core.Context) {
	return middlewares.Handler(handler)
}

func TestBuilder(builder Builder, op *core.Operation, handler func(core.Context)) func(core.Context) {
	mw := builder.Build(op)
	if mw == nil {
		return handler
	}
	return TestMiddleware(mw, handler)
}
