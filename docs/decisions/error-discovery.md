# Error Discovery

## Decision

The `neoma-discover` tool detects error types by analyzing Go types (`go/types`), not by matching function names or string patterns.

## Why

Robustness. The tool needs to find which errors each handler can return, regardless of how the user structures their code. String matching (e.g., looking for `ErrorNotFound` by name) would:

- Break if users wrap our functions in their own helpers
- Miss custom error constructors that return the same type
- Require maintaining a hardcoded list of known function names

Type-based detection works by shape: the tool finds `config.ErrorHandler`, reads its `NewError` return type, then scans all code for anything that produces that type. This catches struct literals, function calls, variables, and nested call chains without knowing any names in advance.

## How It Works

1. Find the `ErrorHandler` assignment in the user's module
2. Read the `NewError()` method on the handler type to determine the concrete error type
3. Find the `StatusCode()` method on that type to determine which struct field holds the HTTP status
4. Scan all handler functions for references to that error type (struct literals, function calls, variables)
5. Follow call chains within the module (handler calls service, service calls repository)
6. Extract status codes from source (integer literals in struct fields or function arguments)
7. Generate an `init()` file mapping handler names to their discovered error statuses

## Trade-offs

- Requires `golang.org/x/tools/go/packages` as a build-time dependency (only for the CLI tool, not the library)
- Cannot detect errors created at runtime via dynamic values (e.g., status code from a database)

## Error Remap Warnings

The tool detects when a handler catches an error from a callee that already returns our error type, and remaps it to a different status. For example:

```go
// service.GetItem can return errors.ErrorUnavailable (503)
item, err := service.GetItem(input.ID)
if err != nil {
    return nil, errors.ErrorNotFound("item not found") // remaps 503 to 404
}
```

Output:
```
neoma-discover: warning: errors from service.GetItem being remapped to 404 Not Found in GetItem
```

This only fires when both the callee and the handler produce our error type (the one derived from `config.ErrorHandler`). Normal error mapping (generic Go error to our error type) does not trigger the warning.
