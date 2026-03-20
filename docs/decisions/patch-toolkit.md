# Patch Toolkit

## Decision

Provide a `patch` package that implements RFC 6902 (JSON Patch) and RFC 7396 (JSON Merge Patch) as a composable toolkit, not as automatic PATCH endpoint generation.

## Why

Original Huma's `autopatch` package auto-generates PATCH endpoints by scanning for GET+PUT pairs and creating handlers that internally call GET, apply the patch, then PUT. This approach is fragile:

- Internal HTTP round-trips are inefficient and hard to debug.
- The handler bypasses application logic (validation, authorization, side effects).
- It couples patching to the HTTP layer instead of the data layer.
- `application/json` was silently accepted as merge-patch, which is incorrect per RFC 5789.

Auto-generation is also not an RFC requirement. The RFCs define patch formats, not how frameworks should wire endpoints.

## How

The `patch` package provides three levels of usage:

**Parse into operations** (for custom storage like SQL or document stores):
```go
ops, err := patch.Parse(contentType, patchBody)
```
Returns `[]patch.Operation` with `Op`, `Path`, `From`, `Value` fields. Users iterate and apply to their own storage.

**Apply to a Go struct** (handles JSON round-trip internally):
```go
err := patch.ApplyTo(contentType, &thing, patchBody)
```

**Apply to raw JSON** (for users already working with bytes):
```go
patched, err := patch.Apply(contentType, original, patchBody)
```

The package also provides:
- `patch.AcceptPatch` constant for the `Accept-Patch` response header (RFC 5789 Section 3.1).
- `patch.ErrUnsupportedContentType` and `patch.ErrInvalidPatch` sentinel errors mapping to 415 and 422.
- `patch.Equal(a, b)` for detecting no-op patches (304 Not Modified).
- `patch.MakeOptionalSchema(s)` for generating merge-patch request body schemas.

## RFC Compliance

- Only `application/merge-patch+json` and `application/json-patch+json` are accepted. Plain `application/json` is rejected.
- RFC 7396 (not 7386, which is obsoleted) is the reference for merge patch.
- RFC 6901 JSON Pointer escaping (`~0`, `~1`) is correctly decoded in `Parse`.
- All 15 test cases from RFC 7396 Appendix A are covered.
