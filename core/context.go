package core

import (
	"context"
	"crypto/tls"
	"io"
	"mime/multipart"
	"net/url"
	"time"
)

type ProtoVersion struct {
	Proto      string
	ProtoMajor int
	ProtoMinor int
}

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

func WithContext(ctx Context, override context.Context) Context {
	return subContext{neomaContext: ctx, override: override}
}

func WithValue(ctx Context, key, value any) Context {
	return WithContext(ctx, context.WithValue(ctx.Context(), key, value))
}

func UnwrapContext(ctx Context) Context {
	for {
		if c, ok := ctx.(interface{ Unwrap() Context }); ok {
			ctx = c.Unwrap()
			continue
		}
		return ctx
	}
}
