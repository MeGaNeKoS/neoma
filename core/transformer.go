package core

// Transformer is a function that can modify or replace a response value before
// it is serialized. The status parameter is the HTTP status code as a string.
type Transformer func(ctx Context, status string, v any) (any, error)
