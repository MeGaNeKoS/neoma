# Contributing to Neoma

## Architecture First

AI-generated code is welcome. Architecture decisions must be made by humans.

- You decide which package a feature belongs in
- You decide what interfaces to use or extend
- You decide how it fits the dependency graph
- AI helps write the implementation

If you are unsure about the right architecture, open an issue first.

## Code Style

### Comments

No unnecessary comments. Do not explain "what" the code does. Only add "why" comments when behavior is non-obvious.

### Function Ordering

Within a file, follow this order:

1. Types (exported, then unexported)
2. Exported functions (alphabetical)
3. Unexported functions (alphabetical)

Methods are grouped with their type. This makes it easy to find any function: exported API is at the top, internals at the bottom, and alphabetical order means you never have to scan the whole file.

### Rules

- No mutable global state unless absolutely necessary. If unavoidable, document the reason in a decision doc under `docs/decisions/`
- Use `errors.As` instead of type assertions on error types
- No import aliases to paper over architectural issues; fix the root cause
- No vendoring code inline; put borrowed code in dedicated packages with attribution (see Dependencies below)
- Remove dead code. Do not comment it out
- Use `_` for unused parameters required by interface signatures

## Testing

- Every exported function must have a test
- Use `require` for error assertions, `assert` for value assertions
- Use testify idioms:
  - `assert.NotEmpty(t, x)` not `assert.True(t, len(x) > 0)`
  - `assert.NoError(t, err)` not `assert.Nil(t, err)`

## Dependencies

Minimize external dependencies.

Borrowed code goes in dedicated packages (e.g., `yaml/`, `validation/uuid.go`) with source URL and version noted in the file or package doc.

## Linting

All code must pass `golangci-lint run ./...` with the project's `.golangci.yml`.

Notable enabled linters:

- **errorlint**: use `errors.As`/`errors.Is`, not type assertions or direct comparisons on error types

## Architecture Decisions

Document non-obvious choices in `docs/decisions/`. Each decision doc should include:

- The decision
- Why it was made
- How it works
- Trade-offs

## Local Development Setup

Prerequisites: Go 1.25+, [golangci-lint](https://golangci-lint.run/usage/install/)

```sh
git clone https://github.com/MeGaNeKoS/neoma.git
cd neoma
```

The repo uses Go workspaces (`go.work`) to link the root module with adapter sub-modules. This is already committed, so `go build` and `go test` work out of the box.

```sh
# Build everything
go build ./...

# Run all tests
go test ./...

# Run linter
golangci-lint run ./...

# Run the CRUD example
cd examples/crud && go run .

# Run the SSE example
cd examples/sse && go run .
```

The adapter sub-modules (e.g., `adapters/neomachi/v5/`) have their own `go.mod` with `replace` directives pointing to the local root. This is for development only; the `replace` directives are removed before publishing releases.

## How to Contribute

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `go test ./...`
5. Run linter: `golangci-lint run ./...`
6. Open a pull request

## License

By contributing, you agree your contributions will be licensed under MPL-2.0.
