# Codegen spike findings

Date: 2026-07-29
Stage: 1 of the api-go v3 transition
Spec: ../specs/2026-07-29-api-v3-transition-design.md
Plan: ../plans/2026-07-29-apiv3-codegen-spike.md

## Decision

Generate with `github.com/oapi-codegen/oapi-codegen/v2@v2.8.0` directly from the
OpenAPI 3.1 bundle. No downconversion to 3.0. Go-specific mappings ride in an
OpenAPI Overlay checked into this repo. Nullable generation is enabled.

Inputs: a 6152-line bundle with 111 operations. Output: `internal/gen/gen.go`,
29835 lines.

## What was tested

| Question | Answer | Evidence |
| --- | --- | --- |
| Does the 3.1 bundle generate without downconversion? | Yes | `make generate` exits 0 |
| Does generated code compile? | Yes | `go build ./...` |
| Do `type: [T, "null"]` unions preserve null vs absent? | **Yes, with `nullable-type: true`** | `TestNullableTypeDistinguishesNullFromAbsent` |
| Can overlays carry Go-specific mappings? | Yes | `TestOverlayAppliedToGeneratedCode` |
| Are unknown fields rejected? | No, ignored by default | `TestUnknownFieldsIgnoredByDefault` |
| Do required properties avoid pointers? | Yes — required become values, optional become pointers | `Project.Id string` vs `Project.Token *string` |
| Are opaque string ids enforced? | Yes — a numeric id is a decode error | `TestProjectIDIsOpaqueString` |
| Does the error envelope carry a required code? | Yes | `TestErrorEnvelopeCarriesCode` |

## Consequences for stage 3

1. **Go floor rises to 1.24.** `github.com/oapi-codegen/runtime` v1.6.0 declares
   `go 1.24.0`. Consumer-visible; belongs in release notes. Both CI jobs moved
   from 1.23 to 1.24 — the `lint` job pins its own version separately from the
   `test` matrix, and both had to change.

   Note the generator itself needs Go ≥ 1.25 and is fetched by `go run`, which
   relies on `GOTOOLCHAIN=auto` (the default). A CI environment pinning
   `GOTOOLCHAIN=local` would fail to generate.

2. **Three-state decoding is available — the design doc was wrong about this.**
   With `output-options: nullable-type: true`, properties declared
   `type: [T, "null"]` generate as `nullable.Nullable[T]` from
   `github.com/oapi-codegen/nullable`: a value type (never a pointer) exposing
   `IsSpecified()`, `IsNull()`, and `Get()`. 46 fields converted.

   This supersedes the design's conclusion that response types cannot
   distinguish null from absent, and largely answers open question 3.

   The limitation that remains: fields that are **optional but not nullable**
   still generate as plain pointers and still conflate the two cases.
   `Project.Token` is one. So three-state availability is a property of how the
   spec declares each field, not a blanket guarantee — pinned by
   `TestOptionalNonNullableFieldsConflateNullAndAbsent`.

   The repo's own `Nullable[T]` in `nullable.go` is unaffected and stays a
   request-side type for the v2 services.

3. **787 anonymous inline structs** in the generated output, from inline object
   schemas in the bundle. This is the main design question handed to stage 3,
   because a consumer cannot name or easily construct an anonymous struct:

   - Ask the honeybadger repo to extract inline objects into named component
     schemas. Cheap, and it improves the published docs too.
   - Wrap them in hand-written facade types, accepting the conversion cost.

   `Error` is the case that matters most: both `Error.Error` and `Error.Meta`
   are anonymous structs, so design decision 7's `apiv3.Error` type must be
   hand-written and converted from the generated shape regardless of which route
   is chosen for the rest.

4. **Contract tests need `DisallowUnknownFields`** to catch renamed fields, since
   plain `json.Unmarshal` ignores them.

5. **The drift gate works and was proven to fail.** Changing one schema
   description in the vendored bundle makes `make verify-generated` exit
   non-zero. Tested by mutating and restoring, not assumed.

## What this does not settle

- Whether the generated client's request/response wrappers are usable directly
  or need a facade. Stage 3 decides, once prerequisites 1 and 2 are met.
- Whether the write-schema gaps (design prerequisite 2) can be worked around
  client-side. They cannot be, on current evidence: unspecified properties do
  not appear in generated request structs at all.
- Which fields genuinely need null-vs-absent discrimination. Now that the
  mechanism exists, the question is whether the spec declares the right fields
  as nullable — a spec-review task, not a client one.

## Reproducing

```bash
make generate            # regenerate from the vendored bundle
make verify-generated    # fails if generated code and spec disagree
go test ./...            # includes the characterization tests
```
