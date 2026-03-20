# Schema Link Transformer

## Decision

Add a `SchemaLinkTransformer` that automatically appends a `Link` header with `rel="describedBy"` and injects a `$schema` field into JSON response bodies, pointing to the JSON Schema describing the response structure.

## Why

RFC 8288 (Web Linking) defines `rel="describedBy"` for linking a resource to its schema. This enables clients and editors (e.g. VSCode) to discover response structure for validation and completion.

Original Huma implements this via `SchemaLinkTransformer` in `transforms.go`. The rewrite was missing it.

## How

- `openapi/schema_link.go` contains the transformer.
- On operation registration (`OnAddOperation`), it pre-computes Link header values and creates wrapper types with a `$schema` field for each response schema.
- On response (`Transform`), it appends the `Link` header and injects `$schema` into the body.
- Registered automatically via `CreateHooks` in `DefaultConfig`.

## Example Response

```http
HTTP/1.1 200 OK
Link: </schemas/GreetingOutput.json>; rel="describedBy"
Content-Type: application/json

{
  "$schema": "http://localhost:8888/schemas/GreetingOutput.json",
  "message": "Hello!"
}
```
