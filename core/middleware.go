package core

type MiddlewareFunc func(ctx Context, next func(Context))

type Middlewares []MiddlewareFunc

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
