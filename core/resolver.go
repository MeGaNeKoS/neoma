package core

// Resolver is implemented by input types that need custom logic to extract or
// validate values from the request. Returned errors are added to the
// validation error response.
type Resolver interface {
	Resolve(ctx Context) []error
}

// ResolverWithPath is like [Resolver] but receives a [PathBuffer] so that
// returned errors can include the field's location path for structured
// error reporting.
type ResolverWithPath interface {
	Resolve(ctx Context, prefix *PathBuffer) []error
}
