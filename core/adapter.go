package core

import "net/http"

// Adapter Every adapter package MUST also export these package-level symbols:
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
	Handle(op *Operation, handler func(ctx Context))
	ServeHTTP(http.ResponseWriter, *http.Request)
}
