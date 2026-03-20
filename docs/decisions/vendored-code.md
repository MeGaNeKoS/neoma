# Vendored Code

## Decision

Small pieces of third-party code are vendored into dedicated packages rather than imported as dependencies.

## Why

Neoma is a library, not an end application. Every dependency we add becomes a transitive dependency for every user. This matters for two reasons:

1. **Binary size and module graph.** Importing `github.com/google/uuid` for one validation function pulls in the entire module. Importing a YAML library for JSON-to-YAML conversion pulls in a full parser we never use. Users pay for these in compile time and binary size.

2. **Supply chain security.** Each dependency is an attack surface. A compromised upstream module affects every application that imports our library. Fewer dependencies means a smaller attack surface for our users.

## What Is Vendored

| Package | Source | Version | What we use |
|---------|--------|---------|-------------|
| `yaml/` | [itchyny/json2yaml](https://github.com/itchyny/json2yaml) | v0.1.5 | JSON-to-YAML converter for OpenAPI YAML output |
| `validation/uuid.go` | [google/uuid](https://github.com/google/uuid) | v1.6.0 | UUID format validation for JSON Schema |

## Rules

- Vendored code lives in its own package with source URL and version in a comment at the top.
- Lint is suppressed for vendored code via `.golangci.yml` path exclusions.
- When updating, check the upstream repository for the latest tag and copy the relevant code.
