package core

type Transformer func(ctx Context, status string, v any) (any, error)
