package core

import (
	"errors"
	"mime/multipart"
)

// Empty is used as a response or request body type when no body is expected.
type Empty struct{}

// ExampleProvider is implemented by types that can return an example value
// for use in OpenAPI documentation.
type ExampleProvider interface {
	Example() any
}

// StreamResponse allows handlers to write directly to the response body using
// a streaming callback instead of returning a serializable value.
type StreamResponse struct {
	Body func(ctx Context, api API)
}

// FormFile represents a single file uploaded via a multipart form request.
type FormFile struct {
	multipart.File
	*multipart.FileHeader
}

var (
	// ErrUnknownContentType is returned when the request Content-Type header
	// specifies a media type that has no registered Format.
	ErrUnknownContentType = errors.New("unknown content type")

	// ErrUnknownAcceptContentType is returned when the request Accept header
	// specifies a media type that has no registered Format.
	ErrUnknownAcceptContentType = errors.New("unknown accept content type")
)
