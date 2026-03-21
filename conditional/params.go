// Package conditional implements HTTP conditional request handling using
// If-Match, If-None-Match, If-Modified-Since, and If-Unmodified-Since headers.
package conditional

import (
	"net/http"
	"strings"
	"time"

	"github.com/MeGaNeKoS/neoma/core"
)

func trimETag(value string) string {
	if strings.HasPrefix(value, "W/") && len(value) > 2 {
		value = value[2:]
	}
	return strings.Trim(value, "\"")
}

// Params holds the parsed HTTP conditional request headers for an operation.
type Params struct {
	IfMatch           []string  `header:"If-Match" doc:"Succeeds if the server's resource matches one of the passed values."`
	IfNoneMatch       []string  `header:"If-None-Match" doc:"Succeeds if the server's resource matches none of the passed values. On writes, the special value * may be used to match any existing value."`
	IfModifiedSince   time.Time `header:"If-Modified-Since" doc:"Succeeds if the server's resource date is more recent than the passed date."`
	IfUnmodifiedSince time.Time `header:"If-Unmodified-Since" doc:"Succeeds if the server's resource date is older or the same as the passed date."`

	isWrite bool
}

// Resolve reads the request method from the context to determine whether the
// current operation is a write (POST, PUT, PATCH, DELETE).
func (p *Params) Resolve(ctx core.Context) []error {
	switch ctx.Method() {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		p.isWrite = true
	}
	return nil
}

// HasConditionalParams reports whether any conditional headers were provided
// in the request.
func (p *Params) HasConditionalParams() bool {
	return len(p.IfMatch) > 0 || len(p.IfNoneMatch) > 0 || !p.IfModifiedSince.IsZero() || !p.IfUnmodifiedSince.IsZero()
}

// Check evaluates all conditional headers against the given ETag and
// modification time, returning an HTTP status code and message if a
// precondition fails, or zero if all preconditions pass.
func (p *Params) Check(etag string, modified time.Time) (status int, msg string) {
	failed := false
	var msgs []string

	foundMsg := "found no existing resource"
	if etag != "" {
		foundMsg = "found resource with ETag " + etag
	}

	for _, match := range p.IfNoneMatch {
		trimmed := trimETag(match)
		if trimmed == etag || (trimmed == "*" && etag != "") {
			if p.isWrite {
				msgs = append(msgs, "If-None-Match: "+match+" precondition failed, "+foundMsg)
			}
			failed = true
		}
	}

	if len(p.IfMatch) > 0 {
		found := false
		for _, match := range p.IfMatch {
			if trimETag(match) == etag {
				found = true
				break
			}
		}
		if !found {
			if p.isWrite {
				msgs = append(msgs, "If-Match precondition failed, "+foundMsg)
			}
			failed = true
		}
	}

	if !p.IfModifiedSince.IsZero() && !modified.After(p.IfModifiedSince) {
		if p.isWrite {
			msgs = append(msgs, "If-Modified-Since: "+p.IfModifiedSince.Format(http.TimeFormat)+" precondition failed, resource was modified at "+modified.Format(http.TimeFormat))
		}
		failed = true
	}

	if !p.IfUnmodifiedSince.IsZero() && modified.After(p.IfUnmodifiedSince) {
		if p.isWrite {
			msgs = append(msgs, "If-Unmodified-Since: "+p.IfUnmodifiedSince.Format(http.TimeFormat)+" precondition failed, resource was modified at "+modified.Format(http.TimeFormat))
		}
		failed = true
	}

	if !failed {
		return 0, ""
	}

	if p.isWrite {
		return http.StatusPreconditionFailed, strings.Join(msgs, "; ")
	}

	return http.StatusNotModified, ""
}
