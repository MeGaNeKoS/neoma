# Multi-Module Structure

## Decision

The repository uses multiple Go modules so that optional adapters with heavy or version-specific dependencies do not raise the Go version or dependency footprint for users who don't need them.

## Why

Echo v5 requires Go 1.25. Fiber pulls in fasthttp. CBOR pulls in fxamacker/cbor. Users who only need the stdlib adapter or Chi should not be forced to depend on any of these.

Go has no concept of optional dependencies (unlike Python's `extras_require`). The standard solution is separate modules within the same repository. Each module has its own `go.mod` declaring its own Go version and dependencies.

## Structure

```
go.mod                                    → core library, go 1.25
adapters/neomachi/v5/go.mod              → requires chi/v5
adapters/neomaecho/v4/go.mod             → requires echo/v4
adapters/neomaecho/v5/go.mod             → requires echo/v5
adapters/neomafiber/v2/go.mod            → requires fiber/v2
adapters/neomafiber/v3/go.mod            → requires fiber/v3
adapters/neomagin/go.mod                 → requires gin
formats/cbor/go.mod                       → requires fxamacker/cbor
```

Adapters that only use stdlib (`neomastdlib`) stay in the root module.

## User Experience

```sh
go get github.com/MeGaNeKoS/neoma                        # core
go get github.com/MeGaNeKoS/neoma/adapters/neomachi/v5   # adds chi
```

Each adapter module depends on the root module for `core` types. Users only pull in what they import.

## Local Development

A `go.work` file at the root ties all modules together for local development. The `replace` directives in each adapter's `go.mod` point to the local root module. These are removed when publishing with proper version tags.
