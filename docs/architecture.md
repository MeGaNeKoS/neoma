# Architecture

Neoma follows hexagonal architecture (ports and adapters). Domain logic and interfaces live in the core, with concrete implementations pushed to the edges. Dependencies always point inward.

## Principles

1. **Dependencies point inward.** `core/` has zero internal imports. Everything depends on `core/`, never the reverse.
2. **Interfaces for contracts, structs for implementation.** Swappable components are defined as interfaces in `core/`. Concrete implementations live in their own packages.
3. **One responsibility per package.** If you can't describe what a package does in one sentence, it's doing too much.
4. **Registration is expensive, request handling is cheap.** All reflection, schema generation, and analysis happen once at startup. The per-request path uses only pre-computed metadata.
5. **No package-level mutable state.** Configuration is instance-level on the `API` struct.

## Layer Model

```
+--------------------------------------------------------------+
|                     OUTER RING                               |
|                   (Adapters, Infra)                          |
|                                                              |
|  adapters/neomachi/v5    adapters/neomaecho/v4               |
|  adapters/neomaecho/v5   adapters/neomafiber/v2              |
|  adapters/neomafiber/v3  adapters/neomagin/v1                |
|  adapters/neomastdlib                                        |
|                                                              |
|  openapi/          neomacli/        neomatest/               |
+--------------------------------------------------------------+
|                     MIDDLE RING                              |
|                   (Use Cases, Domain Logic)                  |
|                                                              |
|  neoma/             Request registration + wiring            |
|  binding/           Parse, validate, serialize               |
|  schema/            Generate and validate schemas            |
|  negotiate/         Content negotiation and format registry  |
|  middleware/        Group routing and middleware             |
|  errors/            Error model implementations              |
|  sse/               Server-Sent Events                       |
+--------------------------------------------------------------+
|                     INNER RING                               |
|                   (Ports, Contracts)                         |
|                                                              |
|  core/context.go     Context interface                       |
|  core/adapter.go     Adapter interface + contract            |
|  core/config.go      API interface, Config, DocsProvider     |
|  core/error.go       Error, ErrorHandler, Headerer, Linker   |
|  core/schema.go      Schema struct, Registry interface       |
|  core/operation.go   Operation struct                        |
|  core/validate.go    ValidateResult, ValidateMode            |
|  core/resolver.go    Resolver interfaces                     |
|  core/middleware.go   MiddlewareFunc, Middlewares            |
|                                                              |
|  validation/         Vendored validators (UUID)              |
|  yaml/               JSON-to-YAML converter (vendored)       |
|  casing/             String case conversion                  |
|  conditional/        HTTP conditional request handling       |
|                                                              |
|  Dependencies: standard library only (or vendored).          |
|  No internal imports. This is the foundation.                |
+--------------------------------------------------------------+
```

## Package Dependency Graph

```
                    core/  (leaf: zero internal deps)
                   / | | \  \
                  /  | |  \  \
            errors/ schema/ negotiate/ middleware/
                  \  |  /      |
                   \ | /       |
                  openapi/  binding/
                      \      /
                      neoma/  (imports everything, wires it together)
                        |
                   adapters/* (outer ring)
```

No circular dependencies. `binding` imports `schema` directly. No runtime wiring hacks.

## Package Responsibilities

| Package              | One sentence                                           | Layer  |
|----------------------|--------------------------------------------------------|--------|
| `core`               | Shared interfaces and types that everything depends on | Inner  |
| `validation`         | Vendored validators (UUID)                             | Inner  |
| `errors`             | Error model implementations (RFC 9457, Noop)           | Middle |
| `schema`             | JSON Schema generation from Go types and validation    | Middle |
| `binding`            | Struct reflection, request parsing, response writing   | Middle |
| `negotiate`          | Content negotiation, format registry, Q-value parsing  | Middle |
| `middleware`         | Group scoping, operation-aware builders                | Middle |
| `openapi`            | Spec serving, docs UI, internal spec, error doc pages  | Outer  |
| `neoma`              | Public facade: `NewAPI`, `Register`, `Get`/`Post`/etc. | Facade |
| `neomacli`           | CLI wrapper using Cobra                                | Outer  |
| `neomatest`          | In-memory test API with request helpers                | Outer  |
| `sse`                | Server-Sent Events                                     | Outer  |
| `adapters/*`         | Router-specific Context and Adapter implementations    | Outer  |
| `formats/cbor`       | CBOR format (explicit registration, not side-effect)   | Outer  |
| `cmd/neoma-discover` | Build-time error discovery via AST scanning            | Tool   |

## Adapter Contract

Every adapter package MUST export:

```go
func NewAdapter(router) core.Adapter
func New(router, core.Config) core.API
func Unwrap(core.Context) <router-type>
var  MultipartMaxMemory int64
```

All adapters use `core.UnwrapContext()` for context unwrapping and `core.SetReadDeadline()` for deadline handling.

## Interface Contracts

### core.ErrorHandler

```go
NewError(status, msg, errs...) Error
NewErrorWithContext(ctx, status, msg, errs...) Error
ErrorSchema(registry) *Schema    // nil = no schema in OAS
ErrorContentType(ct) string      // e.g., "application/problem+json"
```

Error structs can optionally implement `Headerer`, `ContentTyper`, and `Linker` for additional response behavior.

### core.API

The `neoma.api` struct implements `core.API`. Groups and test wrappers delegate to it. All configuration is instance-level.

## Request Processing Flow

```
Register[I, O](api, op, handler)
  |
  +-- STARTUP (once, expensive)
  |   +-- Reflect on I: params, body, resolvers, defaults
  |   +-- Reflect on O: headers, status, body
  |   +-- Build schemas, populate OpenAPI
  |   +-- Look up discovered errors, define error responses
  |   +-- Create handler closure with pre-computed metadata
  |
  +-- PER REQUEST (closure, cheap)
      +-- Parse params (cached field offsets, pre-parsed query)
      +-- Read + unmarshal body
      +-- Validate against pre-built schema
      +-- Apply defaults
      +-- Run resolvers
      +-- Call handler
      +-- Write response headers + body
      +-- Return pooled resources
```

## Design Decisions

Non-obvious architectural choices are documented in `docs/decisions/`:

| Decision                                              | Summary                                                              |
|-------------------------------------------------------|----------------------------------------------------------------------|
| [OpenAPI Versioning](decisions/openapi-versioning.md) | Internal spec is always 3.2; downgrade to 3.0 at serve time          |
| [Dual Spec](decisions/dual-spec.md)                   | Public spec excludes hidden items; internal spec includes everything |
| [Error Discovery](decisions/error-discovery.md)       | AST error scanning and remap warnings                                |
| [Vendored Code](decisions/vendored-code.md)           | Small third-party snippets vendored to minimize dependencies         |
| [SSE](decisions/sse.md)                               | Server-Sent Events via streaming response callback                   |
| [Multi-Module](decisions/multi-module.md)             | Separate Go modules per adapter to isolate dependencies              |

## Adding a New Package

1. Which layer does it belong to?
2. What does it depend on? (same or inner layers only)
3. Does it need a new interface in `core/`?
4. Does it create a circular dependency?
5. Can you describe it in one sentence?

## Adding a New Adapter

1. Create `adapters/neoma<name>/v<N>/neoma<name>.go`
2. Implement `core.Context` on your context struct
3. Implement `core.Adapter` on your adapter struct
4. Export: `NewAdapter`, `New`, `Unwrap`, `MultipartMaxMemory`
5. Use `core.UnwrapContext()` and `core.SetReadDeadline()`
6. Add to the shared test in `adapters/adapters_test.go`
