# Dual OpenAPI Spec (Public and Internal)

## Decision

Neoma serves two separate OpenAPI specs: a public spec that excludes hidden items, and an internal spec that includes everything.

## Why

Different audiences need different views of the same API. The public spec is for external consumers (SDK generation, partner documentation). The internal spec is for the team (debugging, internal tooling, admin endpoints).

Rather than maintaining two separate API definitions or using post-processing filters, Neoma uses a single codebase with `hidden:"true"` tags. The framework generates both specs from the same source, ensuring they never diverge.

## What Can Be Hidden

The `hidden:"true"` tag works on every part of the API:

**Body fields** (excluded from the public schema):
```go
type Input struct {
    Name    string `json:"name"`
    TraceID string `json:"trace_id" hidden:"true"`
}
```

**Query, path, header, and cookie parameters** (moved to `HiddenParameters`, excluded from public spec):
```go
type Input struct {
    Search string `query:"search"`
    Debug  bool   `query:"debug" hidden:"true"`
    Token  string `header:"X-Internal-Token" hidden:"true"`
}
```

**Response headers** (skipped in public spec output):
```go
type Output struct {
    Body      MyBody
    RequestID string `header:"X-Request-ID" hidden:"true"`
}
```

**Entire operations** (stored separately, not added to public paths):
```go
neoma.Register(api, core.Operation{
    Method: http.MethodGet,
    Path:   "/admin/metrics",
    Hidden: true,
}, handler)
```

## How It Works

1. During registration, hidden params are separated into `op.HiddenParameters` instead of `op.Parameters`. Hidden operations go to `HiddenOperationsProvider` instead of the public paths.
2. `Schema.MarshalJSON` checks the `includeHiddenFields` global. When false (default), hidden schema properties are filtered out. When true (internal spec generation), everything is included.
3. The internal spec generator (`openapi/internal.go`) toggles `includeHiddenFields`, marshals the spec, then injects hidden operations and hidden parameters back into the JSON.

## Configuration

```go
config.InternalSpec = core.InternalSpecConfig{
    Enabled:     true,
    Path:        "/internal/openapi",
    DocsPath:    "/internal/docs",
    Middlewares: core.Middlewares{adminAuthMiddleware},
}
```

## Served Endpoints

| Endpoint | Content |
|----------|---------|
| `/openapi.json` | Public spec (hidden items excluded) |
| `/openapi.yaml` | Public spec (YAML) |
| `/public/docs` | Public docs UI |
| `/internal/openapi.json` | Internal spec (everything included) |
| `/internal/openapi.yaml` | Internal spec (YAML) |
| `/internal/docs` | Internal docs UI |

Both public and internal spec endpoints support middleware for authentication.
