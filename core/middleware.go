package core

// MiddlewareFunc is a function that wraps an operation handler, allowing
// pre- and post-processing of requests. Call next to continue the chain.
type MiddlewareFunc func(ctx Context, next func(Context))

// Middlewares is an ordered list of middleware functions.
type Middlewares []MiddlewareFunc

// Handler composes the middleware chain around the given endpoint and returns
// a single handler function that executes them in order.
func (m Middlewares) Handler(endpoint func(Context)) func(Context) {
	return m.chain(endpoint)
}

func (m Middlewares) chain(endpoint func(Context)) func(Context) {
	if len(m) == 0 {
		return endpoint
	}

	w := wrap(m[len(m)-1], endpoint)
	for i := len(m) - 2; i >= 0; i-- {
		w = wrap(m[i], w)
	}
	return w
}

func wrap(fn MiddlewareFunc, next func(Context)) func(Context) {
	return func(ctx Context) {
		fn(ctx, next)
	}
}
