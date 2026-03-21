package core

import "io"

// Format defines a content type's marshaling and unmarshaling functions for
// encoding response bodies and decoding request bodies.
type Format struct {
	Marshal   func(writer io.Writer, v any) error
	Unmarshal func(data []byte, v any) error
}
