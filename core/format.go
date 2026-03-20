package core

import "io"

type Format struct {
	Marshal   func(writer io.Writer, v any) error
	Unmarshal func(data []byte, v any) error
}
