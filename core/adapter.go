package core

import "net/http"

// Adapter defines the interface between neoma and an HTTP router. An adapter
// registers operation handlers and serves HTTP requests. Every adapter package
// must also export these package-level symbols:
//
//	func NewAdapter(router) Adapter
//	func New(router, Config) API
//	func Unwrap(Context) <router-specific type>
//	var MultipartMaxMemory int64
//
// Optional, when the router supports native route groups:
//
//	func NewAdapterWithGroup(router, group) Adapter
type Adapter interface {
	// Handle registers a handler for the given operation.
	Handle(op *Operation, handler func(ctx Context))

	// ServeHTTP dispatches incoming HTTP requests to registered handlers.
	ServeHTTP(http.ResponseWriter, *http.Request)
}
