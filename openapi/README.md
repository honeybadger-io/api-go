# Vendored OpenAPI spec

`bundled.yaml` is a **copy** of the Honeybadger v3 OpenAPI bundle. It is the
input to `make generate`, which writes `internal/gen/gen.go`.

## Provenance

| | |
| --- | --- |
| Source repo | `honeybadger` (the Rails app) |
| Path | `openapi/v3/bundled.yaml` |
| Branch | `scoped-api-tokens-v3` |
| Commit | `4fc37ce2852f9f4d1a9ee607bfe58f8f2af1211d` |
| Vendored | 2026-07-29 |

Record the branch and commit on every refresh. The bundle is **gitignored in the
source repo** (`.gitignore:86`) — it is a build artifact of `rake openapi:bundle`,
not a tracked file — so there is no upstream blob to diff against. Without a
recorded commit there is no way to tell which version of the spec generated the
code in `internal/gen`.

## Refreshing

```bash
# In the honeybadger repo, on the branch you want:
rake openapi:bundle

# Here:
cp <honeybadger>/openapi/v3/bundled.yaml openapi/bundled.yaml
make generate
go test ./...
```

Then update the provenance table above in the same commit as the regenerated
code, so the three always move together.

Longer term, consider vendoring the split sources under `openapi/v3/**` and
running the bundle step here instead. That would make the input tracked and
diffable, at the cost of reimplementing the bundler.

## overlay.yaml

Go-specific concerns — field renames, type mappings — live in `overlay.yaml`, an
[OpenAPI Overlay](https://spec.openapis.org/overlay/v1.0.0.html) applied during
generation. They belong here rather than in the producer spec, so the Rails repo
never carries Go-shaped annotations.

## Known spec issues

Two problems in the current bundle that `apiv3` has to work around. Both are
pinned by tests in `internal/gen/` so they surface if the spec changes.

1. **`Error.details` is typed as an array but used as an object.** The schema
   declares `details: {type: array, items: {field, message}}` for validation
   errors, while the `InsufficientScope` response's example is
   `details: {required_scope: "faults:write", token_scopes: [...]}`. Decoding a
   real 403 into the generated type fails outright.

2. **Three error codes are missing from the `code` enum.**
   `insufficient_scope`, `credential_in_query`, and `project_restricted` appear
   in response descriptions and examples, but `Error.error.code`'s enum still
   lists only ten values, so the generated `ErrorErrorCode` constants cannot
   express them.

Consequence for `apiv3`: treat `code` as an open string rather than a closed
enum, and decode `details` leniently. Both are the resilient choice regardless
of whether the spec is fixed.
