package openapi

import (
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/MeGaNeKoS/neoma/core"
)

const scalarCDN = "https://unpkg.com/@scalar/api-reference@1.44.20/dist/browser/standalone.js"

// ScalarProvider renders API documentation using the Scalar UI. Set LocalJSPath
// to serve the Scalar JavaScript bundle from a local path instead of the CDN.
type ScalarProvider struct {
	LocalJSPath string
}

// Render returns an HTML page that loads the Scalar API reference UI for the given spec URL.
func (p ScalarProvider) Render(specURL string, title string) string {
	if title == "" {
		title = "API Reference"
	}
	jsSrc := scalarCDN
	if p.LocalJSPath != "" {
		jsSrc = p.LocalJSPath
	}
	return `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="referrer" content="no-referrer">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>` + title + `</title>
  </head>
  <body>
    <script id="api-reference" data-url="` + specURL + `.json"></script>
    <script src="` + jsSrc + `"></script>
  </body>
</html>`
}

func (p ScalarProvider) csp() string {
	scriptSrc := scalarCDN
	if p.LocalJSPath != "" {
		scriptSrc = "'self'"
	}
	return strings.Join([]string{
		"default-src 'none'",
		"base-uri 'none'",
		"connect-src 'self' https:",
		"form-action 'none'",
		"frame-ancestors 'none'",
		"script-src 'unsafe-eval' " + scriptSrc,
		"style-src 'unsafe-inline'",
		"img-src 'self' data: blob: https:",
		"font-src 'self' data:",
		"worker-src blob:",
	}, "; ")
}

// StoplightProvider renders API documentation using Stoplight Elements.
type StoplightProvider struct{}

// Render returns an HTML page that loads the Stoplight Elements UI for the given spec URL.
func (p StoplightProvider) Render(specURL string, title string) string {
	if title == "" {
		title = "Elements in HTML"
	}
	return `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="referrer" content="no-referrer">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>` + title + `</title>
    <link rel="stylesheet" href="https://unpkg.com/@stoplight/elements@9.0.15/styles.min.css" crossorigin integrity="sha384-iVQBHadsD+eV0M5+ubRCEVXrXEBj+BqcuwjUwPoVJc0Pb1fmrhYSAhL+BFProHdV">
    <script src="https://unpkg.com/@stoplight/elements@9.0.15/web-components.min.js" crossorigin integrity="sha384-xjOcq9PZ/k+pGtPS/xcsCRXGjKKfTlIa4H1IYEnC+97jNa6sAMWTNrV6hY08W3GL"></script>
  </head>
  <body style="height: 100vh;">
    <elements-api
      apiDescriptionUrl="` + specURL + `.yaml"
      router="hash"
      layout="sidebar"
      tryItCredentialsPolicy="same-origin"
    ></elements-api>
  </body>
</html>`
}

func (p StoplightProvider) csp() string {
	return strings.Join([]string{
		"default-src 'none'",
		"base-uri 'none'",
		"connect-src 'self'",
		"form-action 'none'",
		"frame-ancestors 'none'",
		"script-src https://unpkg.com/@stoplight/elements@9.0.15/web-components.min.js",
		"style-src 'unsafe-inline' https://unpkg.com/@stoplight/elements@9.0.15/styles.min.css",
		"img-src 'self' data: blob:",
		"font-src 'self' data:",
	}, "; ")
}

// SwaggerUIProvider renders API documentation using Swagger UI. Set OAuthClientID
// and OAuthScopes to enable OAuth2 authorization in the UI.
type SwaggerUIProvider struct {
	OAuthClientID string
	OAuthScopes   []string
}

// Render returns an HTML page that loads the Swagger UI for the given spec URL.
func (p SwaggerUIProvider) Render(specURL string, title string) string {
	if title == "" {
		title = "SwaggerUI in HTML"
	}
	return `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="referrer" content="no-referrer">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>` + title + `</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.31.1/swagger-ui.css" crossorigin integrity="sha384-KX9Rx9vM1AmUNAn07bPAiZhFD4C8jdNgG6f5MRNvR+EfAxs2PmMFtUUazui7ryZQ">
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5.31.1/swagger-ui-bundle.js" crossorigin integrity="sha384-o9idN8HE6/V6SAewgnr6/5nz7+Npt5J0Cb4tNyXK8pycsVmgl1ZNbRS7tlEGxd+J"></script>
    <script data-url="` + specURL + `.json">
      const url = document.currentScript.dataset.url;
      window.onload = () => {
        const cfg = {
          url: url,
          dom_id: '#swagger-ui',
        };
        window.ui = SwaggerUIBundle(cfg);` + p.oauthInitJS() + `
      };
    </script>
  </body>
</html>`
}

func (p SwaggerUIProvider) oauthInitJS() string {
	if p.OAuthClientID == "" {
		return ""
	}
	scopes := ""
	if len(p.OAuthScopes) > 0 {
		scopes = `"` + strings.Join(p.OAuthScopes, `", "`) + `"`
	}
	return `
        window.ui.initOAuth({
          clientId: "` + p.OAuthClientID + `",
          scopes: [` + scopes + `],
        });`
}

func (p SwaggerUIProvider) csp() string {
	return strings.Join([]string{
		"default-src 'none'",
		"base-uri 'none'",
		"connect-src 'self'",
		"form-action 'none'",
		"frame-ancestors 'none'",
		"script-src 'unsafe-inline' https://unpkg.com/swagger-ui-dist@5.31.1/swagger-ui-bundle.js",
		"style-src https://unpkg.com/swagger-ui-dist@5.31.1/swagger-ui.css",
		"img-src 'self' data: blob:",
		"font-src 'self' data:",
	}, "; ")
}

type cspProvider interface {
	csp() string
}

// RegisterDocsRoute registers an HTTP route that serves an interactive API
// documentation page using the configured documentation provider.
func RegisterDocsRoute(adapter core.Adapter, oapi *core.OpenAPI, config core.Config) {
	docsPath := config.Docs.Path
	if docsPath == "" {
		return
	}

	openAPIPath := config.OpenAPIPath
	if prefix := getAPIPrefix(oapi); prefix != "" {
		openAPIPath = path.Join(prefix, openAPIPath)
	}

	provider := config.Docs.Provider
	if provider == nil {
		provider = StoplightProvider{}
	}

	var title string
	if oapi.Info != nil && oapi.Info.Title != "" {
		title = oapi.Info.Title + " Reference"
	}

	body := []byte(provider.Render(openAPIPath, title))

	var cspHeader string
	if cp, ok := provider.(cspProvider); ok {
		cspHeader = cp.csp()
	} else {
		cspHeader = strings.Join([]string{
			"default-src 'none'",
			"base-uri 'none'",
			"connect-src 'self'",
			"form-action 'none'",
			"frame-ancestors 'none'",
			"sandbox allow-same-origin allow-scripts allow-popups allow-popups-to-escape-sandbox",
			"script-src 'unsafe-inline'",
			"style-src 'unsafe-inline'",
		}, "; ")
	}

	endpoint := func(ctx core.Context) {
		ctx.SetHeader("Content-Security-Policy", cspHeader)
		ctx.SetHeader("Content-Type", "text/html")
		_, _ = ctx.BodyWriter().Write(body)
	}

	handler := config.Docs.Middlewares.Handler(endpoint)

	adapter.Handle(&core.Operation{
		Method: http.MethodGet,
		Path:   docsPath,
	}, handler)
}

func getAPIPrefix(oapi *core.OpenAPI) string {
	for _, server := range oapi.Servers {
		if server.URL == "" {
			continue
		}

		serverURL, err := url.Parse(server.URL)
		if err != nil {
			panic("invalid server URL: " + server.URL + ": " + err.Error())
		}

		if serverURL.Path == "" {
			continue
		}

		if strings.HasPrefix(server.URL, "/") || serverURL.Host != "" {
			return serverURL.Path
		}
	}

	return ""
}
