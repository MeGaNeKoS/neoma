package middleware

import "github.com/MeGaNeKoS/neoma/core"

// TestMiddleware wraps a single middleware and handler into a callable function
// for use in tests.
func TestMiddleware(mw core.MiddlewareFunc, handler func(core.Context)) func(core.Context) {
	return func(ctx core.Context) {
		mw(ctx, handler)
	}
}

// TestChain composes a middleware chain with a handler into a single callable
// function for use in tests.
func TestChain(middlewares core.Middlewares, handler func(core.Context)) func(core.Context) {
	return middlewares.Handler(handler)
}

// TestBuilder builds middleware from a Builder for the given operation and
// composes it with a handler into a callable function for use in tests.
func TestBuilder(builder Builder, op *core.Operation, handler func(core.Context)) func(core.Context) {
	mw := builder.Build(op)
	if mw == nil {
		return handler
	}
	return TestMiddleware(mw, handler)
}
