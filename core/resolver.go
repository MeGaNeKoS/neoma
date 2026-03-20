package core

type Resolver interface {
	Resolve(ctx Context) []error
}

type ResolverWithPath interface {
	Resolve(ctx Context, prefix *PathBuffer) []error
}
