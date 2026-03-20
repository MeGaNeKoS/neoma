# Neoma

A modern, fast, and flexible framework for building HTTP REST APIs in Go backed by [OpenAPI 3.2](https://spec.openapis.org/oas/v3.2.0.html) and [JSON Schema](https://json-schema.org/).

[![Go Reference](https://pkg.go.dev/badge/github.com/MeGaNeKoS/neoma.svg)](https://pkg.go.dev/github.com/MeGaNeKoS/neoma)
[![License: MPL-2.0](https://img.shields.io/badge/License-MPL%202.0-brightgreen.svg)](https://opensource.org/licenses/MPL-2.0)
[![Go Version](https://img.shields.io/github/go-mod/go-version/MeGaNeKoS/neoma)](https://go.dev/)

- [What is Neoma?](#what-is-neoma)
- [Install](#install)
- [Example](#example)
- [Documentation](#documentation)

## What is Neoma?

Neoma provides a declarative, type-safe layer on top of your router of choice. Define your endpoints with annotated Go structs, and Neoma handles the rest: request parsing, validation, content negotiation, error formatting, OpenAPI generation, and interactive documentation. The goals of this project are to provide:

- Incremental adoption for teams with existing services
  - Bring your own router (Chi, Echo, Gin, Fiber, or Go 1.22+ stdlib), middleware, and logging
  - Extensible OpenAPI and JSON Schema layer to document existing routes
- A modern REST API backend framework for Go developers
  - Described by [OpenAPI 3.2](https://spec.openapis.org/oas/v3.2.0.html) (also supports 3.1 and 3.0) and [JSON Schema](https://json-schema.org/)
- Guard rails to prevent common mistakes
- Documentation that cannot get out of date
- High quality generated developer tooling

Features include:

- Declarative interface on top of your router of choice:
  - Operation and model documentation
  - Request params (path, query, header, or cookie)
  - Request body
  - Responses (including errors)
  - Response headers
- Pluggable error model: [RFC 9457](https://datatracker.ietf.org/doc/html/rfc9457) Problem Details by default, or bring your own `ErrorHandler`
  - Auto-generated error documentation pages with RFC links, causes, and fixes
  - Per-endpoint error examples discovered at build time via `go generate`
  - `Link` header with `rel="type"` pointing to resolvable error documentation
- Per-operation request size limits with sane defaults
- [Content negotiation](https://developer.mozilla.org/en-US/docs/Web/HTTP/Content_negotiation) between server and client
  - Support for JSON ([RFC 8259](https://tools.ietf.org/html/rfc8259)) and optionally CBOR ([RFC 7049](https://tools.ietf.org/html/rfc7049)) content types via the `Accept` header
- Conditional requests support, e.g. `If-Match` or `If-Unmodified-Since` header utilities
- Optional automatic generation of `PATCH` operations that support:
  - [RFC 7386](https://www.rfc-editor.org/rfc/rfc7386) JSON Merge Patch
  - [RFC 6902](https://www.rfc-editor.org/rfc/rfc6902) JSON Patch
  - [Shorthand](https://github.com/danielgtaylor/shorthand) patches
- Annotated Go types for input and output models
  - Generates JSON Schema from Go types
  - Static typing for path, query, header params, bodies, response headers, etc.
  - Automatic input model validation with detailed error locations (e.g. `body.items[3].tags`)
- Dual OpenAPI specs: public and internal
  - Mark fields, parameters, or entire operations as `hidden:"true"` to exclude from the public spec
  - Internal spec includes everything for your team, served at a separate endpoint with optional auth middleware
- Pipeline architecture: request processing is split into named, replaceable stages
  - Insert, replace, or remove stages without modifying framework internals
  - Zero reflection at request time; all struct analysis happens once at registration
- Interactive documentation generation using [Scalar](https://scalar.com/) (default), [Stoplight Elements](https://stoplight.io/open-source/elements), or [Swagger UI](https://swagger.io/tools/swagger-ui/)
  - Docs endpoints support middleware for authentication
- Optional CLI built-in, configured via arguments or environment variables
  - Set via e.g. `-p 8000`, `--port=8000`, or `SERVICE_PORT=8000`
  - Startup actions and graceful shutdown built-in
- Generates OpenAPI for access to a rich ecosystem of tools
  - SDKs with [OpenAPI Generator](https://github.com/OpenAPITools/openapi-generator) or [oapi-codegen](https://github.com/deepmap/oapi-codegen)
  - Mocks with [Prism](https://stoplight.io/open-source/prism)
  - CLI with [Restish](https://rest.sh/)
  - And [plenty](https://openapi.tools/) [more](https://apis.guru/awesome-openapi3/category.html)
- Generates JSON Schema for each resource, served at individual schema endpoints
- Hexagonal architecture: clean separation between core interfaces, domain logic, and adapter implementations

## Install

Install via `go get`. Note that Go 1.25 or newer is required.

```sh
# After: go mod init ...
go get -u github.com/MeGaNeKoS/neoma
```

## Example

Here is a complete basic hello world example in Neoma, that shows how to initialize an app complete with CLI, declare a resource operation, and define its handler function.

```go
package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/MeGaNeKoS/neoma/adapters/neomachi/v5"
	"github.com/MeGaNeKoS/neoma/formats/cbor"
	"github.com/MeGaNeKoS/neoma/neoma"
	"github.com/MeGaNeKoS/neoma/neomacli"
	"github.com/go-chi/chi/v5"
)

type Options struct {
	Port int `help:"Port to listen on" short:"p" default:"8888"`
}

type GreetingOutput struct {
	Body struct {
		Message string `json:"message" example:"Hello, world!" doc:"Greeting message"`
	}
}

func main() {
	cli := neomacli.New(func(hooks neomacli.Hooks, options *Options) {
		router := chi.NewMux()
		adapter := neomachi.NewAdapter(router)
		config := neoma.DefaultConfig("My API", "1.0.0")
		config.Formats["application/cbor"] = cbor.DefaultCBORFormat
		config.Formats["cbor"] = cbor.DefaultCBORFormat
		api := neoma.NewAPI(config, adapter)

		neoma.Get(api, "/greeting/{name}", func(ctx context.Context, input *struct {
			Name string `path:"name" maxLength:"30" example:"world" doc:"Name to greet"`
		}) (*GreetingOutput, error) {
			resp := &GreetingOutput{}
			resp.Body.Message = fmt.Sprintf("Hello, %s!", input.Name)
			return resp, nil
		})

		hooks.OnStart(func() {
			http.ListenAndServe(fmt.Sprintf(":%d", options.Port), router)
		})
	})

	cli.Run()
}
```

> [!TIP]
> Replace `chi.NewMux()` with `http.NewServeMux()` and `neomachi` with `neomastdlib` to use the standard library router from Go 1.22+. Everything else stays the same.

You can test it with `go run greet.go` (optionally pass `--port` to change the default) and make a sample request using [Restish](https://rest.sh/) (or `curl`):

```sh
$ curl localhost:8888/greeting/world
{"message":"Hello, world!"}
```

Even though the example is tiny you can also see some generated documentation at http://localhost:8888/public/docs. The generated OpenAPI spec is available at http://localhost:8888/openapi.json.

## Adapters

Neoma supports multiple routers through adapter packages. Bring your own router, middleware, and logging.

| Adapter       | Import                   | Router                                       | Install                              |
|---------------|--------------------------|----------------------------------------------|--------------------------------------|
| `neomachi`    | `adapters/neomachi/v5`   | [chi/v5](https://github.com/go-chi/chi)      | `go get github.com/go-chi/chi/v5`    |
| `neomaecho`   | `adapters/neomaecho/v4`  | [echo/v4](https://github.com/labstack/echo)  | `go get github.com/labstack/echo/v4` |
| `neomaecho`   | `adapters/neomaecho/v5`  | [echo/v5](https://github.com/labstack/echo)  | `go get github.com/labstack/echo/v5` |
| `neomagin`    | `adapters/neomagin/v1`   | [gin](https://github.com/gin-gonic/gin)      | `go get github.com/gin-gonic/gin`    |
| `neomafiber`  | `adapters/neomafiber/v2` | [fiber/v2](https://github.com/gofiber/fiber) | `go get github.com/gofiber/fiber/v2` |
| `neomafiber`  | `adapters/neomafiber/v3` | [fiber/v3](https://github.com/gofiber/fiber) | `go get github.com/gofiber/fiber/v3` |
| `neomastdlib` | `adapters/neomastdlib`   | `net/http` ServeMux (Go 1.22+)               | (included in Go standard library)    |

Each adapter follows the same pattern:

```go
import "github.com/MeGaNeKoS/neoma/adapters/neomachi/v5"

adapter := neomachi.NewAdapter(router)
api := neoma.NewAPI(config, adapter)
```

## Error Handling

Neoma's error model is fully pluggable through the `ErrorHandler` interface. Two built-in handlers are provided.

**RFC 9457 Problem Details (default):**

```go
config.ErrorHandler = errors.NewRFC9457Handler()

// With custom type URIs and instance tracking:
config.ErrorHandler = errors.NewRFC9457HandlerWithConfig(
    "/errors",
    func(ctx core.Context) string { return ctx.URL().Path },
)
```

```json
{
    "type": "/errors/422",
    "title": "Unprocessable Entity",
    "status": 422,
    "detail": "Validation failed",
    "instance": "/items",
    "errors": [{"message": "expected string length <= 10", "location": "body.name", "value": "too-long"}]
}
```

When using type URIs, Neoma auto-serves error documentation pages with causes, fixes, endpoint listings, and RFC references at each URI (e.g. `/errors/422`).

**No-Op (full custom control):**

```go
config.ErrorHandler = errors.NewNoopHandler()
```

Suppresses all error schemas from the OpenAPI spec. You handle error serialization yourself.

**Convenience functions:**

```go
errors.Error400BadRequest("invalid input")
errors.Error404NotFound("item not found")
errors.Error422UnprocessableEntity("validation failed")
errors.Error500InternalServerError("unexpected error")
```

## OpenAPI

Neoma generates [OpenAPI 3.2](https://spec.openapis.org/oas/v3.2.0.html) specs natively from your Go types. Set `config.OpenAPIVersion` to use 3.1 or 3.0 instead.

| Endpoint          | Content                                                        |
|-------------------|----------------------------------------------------------------|
| `/openapi.json`   | OpenAPI spec (JSON)                                            |
| `/openapi.yaml`   | OpenAPI spec (YAML)                                            |
| `/public/docs`    | Interactive docs UI ([Scalar](https://scalar.com/) by default) |
| `/schemas/{name}` | Individual JSON Schema                                         |

### Public vs. Internal Spec

Mark operations, fields, or parameters as hidden to exclude them from the public spec while keeping a complete internal spec for your team:

```go
config.InternalSpec = core.InternalSpecConfig{
    Enabled:  true,
    Path:     "/internal/openapi",
    DocsPath: "/internal/docs",
}
```

```go
type Input struct {
    Name  string `query:"name"`
    Debug bool   `query:"debug" hidden:"true"`
}
```

The `Debug` parameter appears only in the internal spec. The public spec shows `Name` only.

### Docs UI Providers

```go
config.Docs.Provider = openapi.ScalarProvider{}      // default
config.Docs.Provider = openapi.StoplightProvider{}    // Stoplight Elements
config.Docs.Provider = openapi.SwaggerUIProvider{}    // Swagger UI
```

Docs endpoints support middleware for authentication:

```go
config.Docs.Middlewares = core.Middlewares{authMiddleware}
```

## Auto Discovery

The `neoma-discover` tool scans your handlers for error constructors at build time and generates per-endpoint error examples in the OpenAPI spec. No manual error listing required.

```go
//go:generate go run github.com/MeGaNeKoS/neoma/cmd/neoma-discover -output neoma_errors_gen.go ./...
```

```sh
go generate ./...
```

The generated file auto-registers via `init()` when the handlers package is imported. No manual wiring needed. Each operation's error responses will include concrete, endpoint-specific examples.

## Configuration

Key configuration options (all set on the `Config` struct returned by `neoma.DefaultConfig`):

| Field                                | Default              | Description                                                |
|--------------------------------------|----------------------|------------------------------------------------------------|
| `OpenAPIVersion`                     | `"3.2.0"`            | OpenAPI spec version (`"3.2.0"`, `"3.1.0"`, or `"3.0.3"`)  |
| `OpenAPIPath`                        | `"/openapi"`         | URL prefix for spec endpoints                              |
| `Docs.Path`                          | `"/public/docs"`     | URL path for the docs UI                                   |
| `Docs.Provider`                      | `ScalarProvider{}`   | Docs renderer (Scalar, Stoplight, Swagger UI)              |
| `SchemasPath`                        | `"/schemas"`         | URL path for individual JSON Schema endpoints              |
| `DefaultFormat`                      | `"application/json"` | Fallback content type                                      |
| `ErrorHandler`                       | RFC 9457             | Error response handler                                     |
| `AllowAdditionalPropertiesByDefault` | `true`               | Allow extra JSON fields by default                         |
| `FieldsOptionalByDefault`            | `true`               | Struct fields are optional unless tagged `required:"true"` |
| `RejectUnknownQueryParameters`       | `false`              | Return 422 on unknown query parameters                     |
| `InternalSpec.Enabled`               | `false`              | Enable the internal OpenAPI spec                           |

## Documentation

Full documentation is available in the [wiki](https://github.com/MeGaNeKoS/neoma/wiki):

- [Getting Started](https://github.com/MeGaNeKoS/neoma/wiki/Getting-Started): Installation, hello world, first API
- [Architecture](https://github.com/MeGaNeKoS/neoma/wiki/Architecture): Hexagonal design, package layout, dependency graph
- [Configuration](https://github.com/MeGaNeKoS/neoma/wiki/Configuration): All config fields and options
- [Operations](https://github.com/MeGaNeKoS/neoma/wiki/Operations): Input/output structs, validation tags, registration
- [Error Handling](https://github.com/MeGaNeKoS/neoma/wiki/Error-Handling): Pluggable errors, RFC 9457, convenience functions
- [OpenAPI Specification](https://github.com/MeGaNeKoS/neoma/wiki/OpenAPI-Specification): Spec generation, schemas, examples
- [Public and Internal Specs](https://github.com/MeGaNeKoS/neoma/wiki/Public-and-Internal-Specs): Dual spec, hidden fields
- [Adapters](https://github.com/MeGaNeKoS/neoma/wiki/Adapters): Router adapters, writing your own
- [Middleware](https://github.com/MeGaNeKoS/neoma/wiki/Middleware): Groups, builders, testing
- [Pipeline](https://github.com/MeGaNeKoS/neoma/wiki/Pipeline): Request processing stages
- [Auto Discovery](https://github.com/MeGaNeKoS/neoma/wiki/Auto-Discovery): Error discovery tool
- [Testing](https://github.com/MeGaNeKoS/neoma/wiki/Testing): neomatest package
- [CLI Integration](https://github.com/MeGaNeKoS/neoma/wiki/CLI-Integration): neomacli, Cobra, flags
- [Server-Sent Events](https://github.com/MeGaNeKoS/neoma/wiki/Server-Sent-Events): SSE support

Go package documentation: [pkg.go.dev/github.com/MeGaNeKoS/neoma](https://pkg.go.dev/github.com/MeGaNeKoS/neoma)

## Credits

Neoma draws inspiration from [Huma](https://github.com/danielgtaylor/huma) by Daniel Taylor.

## License

[MPL-2.0](LICENSE)
