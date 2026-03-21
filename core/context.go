package core

import (
	"context"
	"crypto/tls"
	"io"
	"mime/multipart"
	"net/url"
	"time"
)

// ProtoVersion holds the HTTP protocol version of a request (for example,
// "HTTP/1.1" or "HTTP/2.0").
type ProtoVersion struct {
	Proto      string
	ProtoMajor int
	ProtoMinor int
}

// Context represents the HTTP request and response state for a single API
// call. It provides read access to the incoming request (method, headers,
// URL, body) and write access to the outgoing response (status, headers,
// body). Implementations are supplied by the adapter.
type Context interface {
	Operation() *Operation
	Context() context.Context
	TLS() *tls.ConnectionState
	Version() ProtoVersion
	Method() string
	Host() string
	RemoteAddr() string
	URL() url.URL
	Param(name string) string
	Query(name string) string
	Header(name string) string
	EachHeader(cb func(name, value string))
	BodyReader() io.Reader
	GetMultipartForm() (*multipart.Form, error)
	SetReadDeadline(time.Time) error
	SetStatus(code int)
	Status() int
	SetHeader(name, value string)
	AppendHeader(name, value string)

	GetResponseHeader(name string) string
	DeleteResponseHeader(name string)

	BodyWriter() io.Writer

	MatchedPattern() string
}

type neomaContext Context

type subContext struct {
	neomaContext
	override context.Context
}

func (c subContext) Context() context.Context {
	return c.override
}

func (c subContext) Unwrap() Context {
	return c.neomaContext
}

// WithContext returns a shallow copy of ctx that uses override as its
// underlying [context.Context]. This is useful for attaching deadlines,
// cancellation signals, or request-scoped values.
func WithContext(ctx Context, override context.Context) Context {
	return subContext{neomaContext: ctx, override: override}
}

// WithValue returns a shallow copy of ctx with the given key-value pair
// attached to its underlying [context.Context].
func WithValue(ctx Context, key, value any) Context {
	return WithContext(ctx, context.WithValue(ctx.Context(), key, value))
}

// UnwrapContext repeatedly unwraps a wrapped Context until it reaches the
// innermost adapter-provided Context. This is useful for accessing the
// original adapter context from middleware that may have wrapped it.
func UnwrapContext(ctx Context) Context {
	for {
		if c, ok := ctx.(interface{ Unwrap() Context }); ok {
			ctx = c.Unwrap()
			continue
		}
		return ctx
	}
}
