# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.0] - 2026-03-21

### Breaking Changes

- **`Operation.ErrorExamples` type changed** from `map[int]any` to `map[int]map[string]*Example`.
  The previous type only allowed a single unnamed example per status code, which did not align
  with OpenAPI's named examples support. Auto-detected error examples (via error discovery) were
  not affected and already supported multiple examples per status code.

  Migration:

  ```go
  // Before (v1.1.x)
  op.ErrorExamples = map[int]any{
      400: myErrorValue,
  }

  // After (v1.2.0)
  op.ErrorExamples = map[int]map[string]*core.Example{
      400: {
          "Validation Error": {
              Summary: "Field validation failed",
              Value:   myErrorValue,
          },
      },
  }

  // Multiple examples per status code (new capability)
  op.ErrorExamples = map[int]map[string]*core.Example{
      400: {
          "Validation Error": {
              Summary: "Field validation failed",
              Value:   map[string]any{"status": 400, "detail": "name is required"},
          },
          "Bad Payload": {
              Summary: "Malformed request body",
              Value:   map[string]any{"status": 400, "detail": "invalid JSON"},
          },
      },
  }
  ```

  When only one named example is provided for a status code, the spec output uses the singular
  `example` field for compatibility. When multiple are provided, it uses the `examples` map with
  names and summaries per the OpenAPI specification.

### Added

- Multipart form-data runtime handling: requests with `multipart/form-data` Content-Type are
  now routed to `ProcessMultipartForm` instead of failing with "unknown content type".
- Required field validation for multipart forms (`required:"true"` tag).
- Schema tag support for multipart non-file fields (`example`, `doc`, `enum`, `default`,
  `minimum`, `maximum`, `format`, etc.) via `schema.FromField`.
- `ExcludeHiddenSchemas` config option (enabled by default) filters hidden operation schemas
  from the public OpenAPI spec and `/schemas/{name}.json` endpoint. Hidden schemas remain
  accessible through `/internal/openapi.json`.
- Body read timeout and max body bytes defaults now apply to multipart operations.

### Fixed

- Multipart semantic errors (`failed to open uploaded file`, `invalid form field value`,
  `required field missing`) now return 422 instead of 400, per RFC 9110.

## [1.1.4] - 2026-03-21

### Fixed

- Multipart required field validation at runtime (422 for missing `required:"true"` fields).
- Multipart OpenAPI schema now includes `required` array for required fields.
- Non-file multipart fields now use `schema.FromField` for full tag support.
- `ensureBodyReadTimeout` and `ensureMaxBodyBytes` now called for multipart operations.
- Inconsistent 400/422 status code on slice file open error.

## [1.1.3] - 2026-03-21

### Added

- `ExcludeHiddenSchemas` config option filters hidden schemas from public spec and schema endpoint.
- Enabled by default in `DefaultConfig`.

## [1.1.2] - 2026-03-21

### Fixed

- Multipart form-data requests now handled at runtime (previously returned "unknown content type").
- Content-Type matching uses `mime.ParseMediaType` for spec-compliant detection.
- Default values applied after multipart form processing.

## [1.1.1] - 2026-03-21

### Fixed

- Multipart semantic errors return 422 instead of 400.

## [1.1.0]

### Added

- Initial multipart form-data OpenAPI spec generation.
- Internal/external spec separation with hidden operations support.

## [1.0.0]

- Initial release.
