# OpenAPI Versioning Strategy

## Decision

The internal OpenAPI spec is always built using the latest version (3.2). When serving, the version is set from `config.OpenAPIVersion`. For 3.0, a post-marshal downgrade transforms the JSON structure. For 3.1 and 3.2, the spec is served as-is.

## Why

Go's `json.Marshaler` interface is `MarshalJSON() ([]byte, error)`. It accepts no parameters. Every OpenAPI type (`Schema`, `Param`, `Response`, etc.) implements this interface for JSON output.

Making marshaling version-aware would require one of:

1. A mutable global variable (like `includeHiddenFields`) to pass the target version into MarshalJSON. This adds more non-goroutine-safe global state.
2. A custom marshal function signature instead of `json.Marshaler`, losing stdlib and third-party compatibility.
3. Version branching inside every type's MarshalJSON method, polluting the entire codebase for one legacy version.

None of these are worth it when 3.0 is the only version that needs structural transformation. 3.1 and 3.2 are structurally identical.

## How It Works

1. `Schema.MarshalJSON` always outputs 3.1+ format (e.g., nullable as `"type": ["string", "null"]`, examples as array).
2. `config.OpenAPIVersion` controls what version string goes on the spec and whether to downgrade.
3. For 3.0: `openapi/downgrade.go` marshals to JSON, walks the untyped `map[string]any`, and converts 3.1+ features to 3.0 equivalents (type arrays to nullable, exclusive min/max numbers to bools, examples array to single example, contentEncoding to x-contentEncoding).
4. For 3.1/3.2: direct marshal, no transformation.

## Trade-offs

- The 3.0 downgrade is lossy (e.g., multiple examples become one).
- The 3.0 path does double marshal/unmarshal (marshal to JSON, unmarshal to map, walk, re-marshal).
- The walk operates on untyped `map[string]any`, which is less type-safe than struct manipulation.

These trade-offs are acceptable because 3.0 is increasingly legacy, and keeping one clean marshal path for 3.1/3.2 is worth the cost.
