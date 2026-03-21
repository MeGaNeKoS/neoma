package core

import (
	"fmt"
	"net/http"
	"reflect"
	"time"
)

// Deref returns the underlying non-pointer type by repeatedly dereferencing
// pointer types.
func Deref(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

// SetReadDeadline sets a read deadline on the underlying connection of the
// given ResponseWriter, unwrapping middleware wrappers as needed. It returns
// [http.ErrNotSupported] if the writer does not support deadlines.
func SetReadDeadline(w http.ResponseWriter, deadline time.Time) error {
	for {
		switch t := w.(type) {
		case interface{ SetReadDeadline(time.Time) error }:
			return t.SetReadDeadline(deadline)
		case interface{ Unwrap() http.ResponseWriter }:
			w = t.Unwrap()
		default:
			return fmt.Errorf("%w", http.ErrNotSupported)
		}
	}
}

// BaseType returns the innermost element type after dereferencing pointers
// and unwrapping slices, arrays, and maps.
func BaseType(t reflect.Type) reflect.Type {
	t = Deref(t)
	for {
		switch t.Kind() {
		case reflect.Slice, reflect.Array, reflect.Map:
			t = Deref(t.Elem())
		default:
			return t
		}
	}
}
