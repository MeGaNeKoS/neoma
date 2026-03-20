package core

import (
	"errors"
	"mime/multipart"
)

type Empty struct{}

type ExampleProvider interface {
	Example() any
}

type StreamResponse struct {
	Body func(ctx Context, api API)
}

type FormFile struct {
	multipart.File
	*multipart.FileHeader
}

var (
	ErrUnknownContentType       = errors.New("unknown content type")
	ErrUnknownAcceptContentType = errors.New("unknown accept content type")
)
