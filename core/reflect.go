package core

import (
	"fmt"
	"net/http"
	"reflect"
	"time"
)

func Deref(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

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
