package openapi

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/MeGaNeKoS/neoma/core"
)


type endpointInfo struct {
	Method string
	Path   string
}

var defaultErrorDocs = map[int]core.ErrorDoc{
	400: {
		Title:       "Bad Request",
		Description: "The server cannot process the request due to something that is perceived to be a client error.",
		Entries: []core.ErrorDocEntry{
			{Cause: "Malformed JSON in the request body", Fix: "Validate your JSON payload with a linter before sending"},
			{Cause: "Missing required query parameters", Fix: "Check the API documentation for required parameters"},
			{Cause: "Invalid parameter format (e.g., string where number expected)", Fix: "Ensure parameter types match the schema"},
		},
	},
	401: {
		Title:       "Unauthorized",
		Description: "The request requires authentication. No valid credentials were provided.",
		Entries: []core.ErrorDocEntry{
			{Cause: "Missing Authorization header", Fix: "Include a valid Authorization: Bearer &lt;token&gt; header"},
			{Cause: "Expired or invalid authentication token", Fix: "Refresh your authentication token"},
			{Cause: "Using the wrong authentication scheme", Fix: "Check the API documentation for the required authentication method"},
		},
	},
	403: {
		Title:       "Forbidden",
		Description: "The server understood the request but refuses to authorize it. Re-authenticating will not help.",
		Entries: []core.ErrorDocEntry{
			{Cause: "Your account lacks the required permissions", Fix: "Contact an administrator to request access"},
			{Cause: "The resource is restricted to specific roles", Fix: "Verify you are using credentials with the correct scope/role"},
		},
	},
	404: {
		Title:       "Not Found",
		Description: "The requested resource does not exist on the server.",
		Entries: []core.ErrorDocEntry{
			{Cause: "The resource ID does not exist", Fix: "Verify the resource ID exists (e.g., list resources first)"},
			{Cause: "The URL path is misspelled or incorrect", Fix: "Check the URL path against the API documentation"},
			{Cause: "The resource was previously deleted", Fix: "If using path parameters, ensure they are URL-encoded correctly"},
		},
	},
	409: {
		Title:       "Conflict",
		Description: "The request conflicts with the current state of the target resource.",
		Entries: []core.ErrorDocEntry{
			{Cause: "Trying to create a resource that already exists", Fix: "Fetch the current resource state before modifying"},
			{Cause: "Concurrent modification detected (optimistic locking)", Fix: "Use If-Match headers for conditional updates"},
			{Cause: "State transition not allowed", Fix: "Check business rules for allowed state transitions"},
		},
	},
	412: {
		Title:       "Precondition Failed",
		Description: "One or more conditions in the request headers evaluated to false on the server.",
		Entries: []core.ErrorDocEntry{
			{Cause: "If-Match ETag does not match the current resource version", Fix: "Re-fetch the resource to get the latest ETag"},
			{Cause: "Another client modified the resource since you last fetched it", Fix: "Retry the request with the updated precondition headers"},
		},
	},
	422: {
		Title:       "Unprocessable Entity",
		Description: "The request body was syntactically correct but failed semantic validation.",
		Entries: []core.ErrorDocEntry{
			{Cause: "Required fields are missing from the request body", Fix: "Check the errors array in the response for specific field issues"},
			{Cause: "Field values are outside allowed ranges", Fix: "Refer to the API schema for field constraints (required, enum, min, max, pattern)"},
			{Cause: "Field values don't match the expected pattern or enum", Fix: "Each error includes location (e.g., body.email) and message"},
		},
	},
	429: {
		Title:       "Too Many Requests",
		Description: "You have sent too many requests in a given amount of time.",
		Entries: []core.ErrorDocEntry{
			{Cause: "Rate limit exceeded for your API key or IP address", Fix: "Check the Retry-After header for when to retry"},
			{Cause: "Burst limit exceeded (too many concurrent requests)", Fix: "Implement exponential backoff in your client"},
		},
	},
	500: {
		Title:       "Internal Server Error",
		Description: "The server encountered an unexpected condition that prevented it from fulfilling the request.",
		Entries: []core.ErrorDocEntry{
			{Cause: "An unhandled exception occurred in the server", Fix: "This is a server-side issue; your request is likely correct"},
			{Cause: "A downstream service or database is unavailable", Fix: "Retry after a short delay (the issue may be transient)"},
			{Cause: "A configuration error on the server side", Fix: "If the error persists, report it with the instance URI from the response"},
		},
	},
	502: {
		Title:       "Bad Gateway",
		Description: "The server received an invalid response from an upstream server while trying to fulfill the request.",
		Entries: []core.ErrorDocEntry{
			{Cause: "An upstream service returned an unexpected response", Fix: "Retry after a short delay"},
			{Cause: "Network connectivity issues between services", Fix: "If persistent, the issue is on the server side"},
		},
	},
	503: {
		Title:       "Service Unavailable",
		Description: "The server is temporarily unable to handle the request, usually due to maintenance or overload.",
		Entries: []core.ErrorDocEntry{
			{Cause: "The server is undergoing maintenance", Fix: "Check the Retry-After header if present"},
			{Cause: "The server is overloaded with requests", Fix: "Retry with exponential backoff"},
			{Cause: "A dependent service is down", Fix: "Check the service status page if available"},
		},
	},
}


// RegisterErrorDocRoutes registers HTTP routes that serve human-readable error
// documentation pages (HTML and JSON) for each HTTP status code used by the API.
func RegisterErrorDocRoutes(adapter core.Adapter, factory core.ErrorHandler, config core.Config) {
	var baseURI string
	if f, ok := factory.(interface{ GetTypeBaseURI() string }); ok {
		baseURI = f.GetTypeBaseURI()
	}
	if baseURI == "" {
		return
	}

	oapi := config.OpenAPI

	adapter.Handle(&core.Operation{
		Method: http.MethodGet,
		Path:   baseURI + "/{code}",
	}, func(ctx core.Context) {
		codeStr := ctx.Param("code")
		code, err := strconv.Atoi(codeStr)
		if err != nil {
			ctx.SetStatus(http.StatusNotFound)
			_, _ = ctx.BodyWriter().Write([]byte("Unknown error code"))
			return
		}

		doc := getErrorDoc(code, config.ErrorDocs)
		if doc.Title == "" {
			ctx.SetStatus(http.StatusNotFound)
			_, _ = ctx.BodyWriter().Write([]byte("Unknown HTTP status code"))
			return
		}

		endpoints := findEndpointsForError(oapi, code)

		var exampleJSON string
		if ex := factory.NewError(code, http.StatusText(code)); ex != nil {
			if b, err := json.MarshalIndent(ex, "  ", "  "); err == nil {
				exampleJSON = string(b)
			}
		}

		accept := ctx.Header("Accept")
		if strings.Contains(accept, "application/json") {
			writeJSONErrorDoc(ctx, code, doc, endpoints, exampleJSON)
			return
		}

		if doc.HTML != "" {
			ctx.SetHeader("Content-Type", "text/html; charset=utf-8")
			_, _ = ctx.BodyWriter().Write([]byte(doc.HTML))
			return
		}

		writeHTMLErrorDoc(ctx, code, doc, endpoints, exampleJSON, baseURI, config)
	})
}


func findEndpointsForError(oapi *core.OpenAPI, code int) []endpointInfo {
	if oapi == nil || oapi.Paths == nil {
		return nil
	}
	codeStr := strconv.Itoa(code)
	var result []endpointInfo
	for path, pi := range oapi.Paths {
		for method, op := range map[string]*core.Operation{
			"GET": pi.Get, "POST": pi.Post, "PUT": pi.Put,
			"DELETE": pi.Delete, "PATCH": pi.Patch, "HEAD": pi.Head,
		} {
			if op == nil {
				continue
			}
			if _, ok := op.Responses[codeStr]; ok {
				result = append(result, endpointInfo{Method: method, Path: path})
			}
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Path != result[j].Path {
			return result[i].Path < result[j].Path
		}
		return result[i].Method < result[j].Method
	})
	return result
}

func getErrorDoc(code int, userDocs map[int]core.ErrorDoc) core.ErrorDoc {
	if doc, ok := userDocs[code]; ok {
		return doc
	}
	if doc, ok := defaultErrorDocs[code]; ok {
		return doc
	}

	title := http.StatusText(code)
	if title == "" {
		return core.ErrorDoc{}
	}
	return core.ErrorDoc{
		Title:       title,
		Description: "No detailed documentation available for this status code.",
	}
}

func statusBadge(code int) string {
	switch {
	case code >= 500:
		return "badge-error"
	case code >= 400:
		return "badge-warn"
	default:
		return "badge-info"
	}
}

func statusCategory(code int) string {
	switch {
	case code >= 500:
		return "Server Error"
	case code >= 400:
		return "Client Error"
	default:
		return "Info"
	}
}

func statusColor(code int) string {
	switch {
	case code >= 500:
		return "#dc2626"
	case code >= 400:
		return "#d97706"
	default:
		return "#6b7280"
	}
}

func writeHTMLErrorDoc(ctx core.Context, code int, doc core.ErrorDoc, endpoints []endpointInfo, exampleJSON string, baseURI string, config core.Config) {
	codeStr := strconv.Itoa(code)
	apiTitle := ""
	if config.OpenAPI != nil && config.Info != nil {
		apiTitle = config.Info.Title + " &middot; "
	}

	var buf strings.Builder
	buf.WriteString(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>` + apiTitle + `Error ` + codeStr + ` &ndash; ` + doc.Title + `</title>
  <style>
    * { box-sizing: border-box; }
    body { font-family: system-ui, -apple-system, sans-serif; max-width: 720px; margin: 2rem auto; padding: 0 1.5rem; color: #1a1a1a; line-height: 1.6; }
    .status { font-size: 4rem; font-weight: 800; color: ` + statusColor(code) + `; margin: 0; }
    h1 { color: #1a1a1a; margin: 0.25rem 0 1rem; font-size: 1.75rem; }
    h2 { margin-top: 2rem; border-bottom: 1px solid #e5e5e5; padding-bottom: 0.3rem; font-size: 1.1rem; color: #555; text-transform: uppercase; letter-spacing: 0.05em; }
    p { color: #444; }
    code { background: #f5f5f5; padding: 2px 6px; border-radius: 4px; font-size: 0.9em; }
    pre { background: #1e1e2e; color: #cdd6f4; padding: 1rem; border-radius: 8px; overflow-x: auto; font-size: 0.85rem; line-height: 1.5; }
    pre code { background: none; padding: 0; color: inherit; }
    ul { padding-left: 1.25rem; }
    li { margin-bottom: 0.4rem; }
    .endpoint { display: inline-block; background: #e8f4fd; color: #0969da; padding: 2px 8px; border-radius: 4px; font-family: monospace; font-size: 0.85rem; margin: 2px; }
    .method { font-weight: 700; }
    .causes li, .fixes li { color: #444; }
    .card { background: #f8f9fa; border: 1px solid #e5e5e5; border-radius: 8px; padding: 1rem 1.25rem; margin: 1rem 0; }
    a { color: #0969da; text-decoration: none; }
    a:hover { text-decoration: underline; }
    .nav { margin-top: 2.5rem; padding-top: 1rem; border-top: 1px solid #e5e5e5; font-size: 0.9rem; }
    .badge { display: inline-block; padding: 2px 8px; border-radius: 4px; font-size: 0.75rem; font-weight: 600; text-transform: uppercase; }
    .badge-error { background: #fee2e2; color: #991b1b; }
    .badge-warn { background: #fef3c7; color: #92400e; }
    .badge-info { background: #dbeafe; color: #1e40af; }
  </style>
</head>
<body>
  <p class="status">` + codeStr + `</p>
  <h1>` + doc.Title + ` <span class="badge ` + statusBadge(code) + `">` + statusCategory(code) + `</span></h1>
  <p>` + doc.Description + `</p>`)

	if len(doc.Entries) > 0 {
		buf.WriteString(`
  <h2>Causes and Fixes</h2>
  <table>
    <tr><th>Cause</th><th>Fix</th></tr>`)
		for _, entry := range doc.Entries {
			buf.WriteString(`
    <tr><td>` + entry.Cause + `</td><td>` + entry.Fix + `</td></tr>`)
		}
		buf.WriteString(`
  </table>`)
	}

	if len(endpoints) > 0 {
		buf.WriteString(`
  <h2>Endpoints Returning This Error</h2>
  <div>`)
		for _, ep := range endpoints {
			buf.WriteString(`
    <span class="endpoint"><span class="method">` + ep.Method + `</span> ` + ep.Path + `</span>`)
		}
		buf.WriteString(`
  </div>`)
	}

	if exampleJSON != "" {
		buf.WriteString(`
  <h2>Example Response</h2>
  <div class="card">
    <div style="margin-bottom: 0.5rem">
      <code>Content-Type: application/problem+json</code><br>
      <code>Link: &lt;` + baseURI + `/` + codeStr + `&gt;; rel="type"</code>
    </div>
  </div>
  <pre><code>` + exampleJSON + `</code></pre>`)
	}

	if len(endpoints) > 0 {
		ep := endpoints[0]
		buf.WriteString(`
  <h2>Debug with cURL</h2>
  <pre><code>curl -v -X ` + ep.Method + ` http://localhost:8080` + ep.Path + ` \
  -H "Content-Type: application/json"</code></pre>`)
	}

	buf.WriteString(`
  <h2>Reference</h2>
  <ul>
    <li>Error format: <a href="https://datatracker.ietf.org/doc/html/rfc9457">RFC 9457 &ndash; Problem Details for HTTP APIs</a></li>
    <li>Status code: <a href="https://datatracker.ietf.org/doc/html/rfc9110#status.` + codeStr + `">RFC 9110 &ndash; HTTP Semantics &sect;` + codeStr + `</a></li>
  </ul>`)

	docsPath := config.Docs.Path
	if docsPath == "" {
		docsPath = "/public/docs"
	}
	buf.WriteString(`
  <div class="nav">
    <a href="` + docsPath + `">&larr; API Documentation</a>
  </div>
</body>
</html>`)

	ctx.SetHeader("Content-Type", "text/html; charset=utf-8")
	_, _ = ctx.BodyWriter().Write([]byte(buf.String()))
}

func writeJSONErrorDoc(ctx core.Context, code int, doc core.ErrorDoc, endpoints []endpointInfo, exampleJSON string) {
	ctx.SetHeader("Content-Type", "application/json")
	result := map[string]any{
		"status":      code,
		"title":       doc.Title,
		"description": doc.Description,
		"entries":     doc.Entries,
	}
	if len(endpoints) > 0 {
		eps := make([]string, len(endpoints))
		for i, ep := range endpoints {
			eps[i] = ep.Method + " " + ep.Path
		}
		result["endpoints"] = eps
	}
	if exampleJSON != "" {
		result["example"] = exampleJSON
	}
	b, _ := json.MarshalIndent(result, "", "  ")
	_, _ = ctx.BodyWriter().Write(b)
}
