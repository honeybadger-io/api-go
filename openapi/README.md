# Vendored OpenAPI spec

`bundled.yaml` is a **copy** of the Honeybadger v3 OpenAPI bundle. It is the
input to `make generate`, which writes `internal/gen/gen.go`.

## Provenance

| | |
| --- | --- |
| Source repo | `honeybadger` (the Rails app) |
| Path | `openapi/v3/bundled.yaml` |
| Branch | `scoped-api-tokens-v3` |
| Commit | `db59097aa562394d51f8d25b2e49630de5c5acff` |
| Vendored | 2026-07-29 |
| sha256 | `3d94288f5194efdf60b6d7173d9b5ba98da085b710553e50f93ddd63f11b07be` |

Record the branch, commit, **and checksum** on every refresh. The bundle is
**gitignored in the source repo** (`.gitignore:86`) — it is a build artifact of
`rake openapi:bundle`, not a tracked file — so there is no upstream blob to diff
against.

The commit alone is not enough. The artifact is regenerated whenever anyone runs
the rake task, so the same commit can produce different bundles: during this
vendoring the source file changed twice within a few minutes, once mid-copy. The
checksum is what makes "is my copy still what I think it is" answerable.

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

One problem remains in the current bundle, pinned by a test in `internal/gen/` so
it surfaces if the spec changes.

**`Error.details` is typed as an array but used as an object.** The schema
declares `details: {type: array, items: {field, message}}` for validation errors,
while the `insufficient_scope` example under the `Forbidden` response sends
`details: {required_scope: "faults:write", token_scopes: [...]}`. Both cannot
hold. `overlay.yaml` retypes the field as `json.RawMessage` so generated code
accepts either, and `apiv3` decides which it received.

`apiv3` also treats `code` as an open string rather than the generated enum. The
enum has grown from 10 to 20 values during v3's development, so a closed set would
silently drop codes the client has not caught up with.

### Fixed since first vendoring

- **Basic auth removed** — `security` is `bearer_auth` only, and a Basic attempt
  now answers `unsupported_auth_scheme`.
- **The error enum gained the missing codes** — `insufficient_scope`,
  `credential_in_query`, and `project_restricted` are declared, along with
  `requires_user_token`, `account_inactive`, `account_parked`,
  `feature_unavailable`, `delete_failed`, and `limit_reached`.
- **The two time-series pagination schemes collapsed into one.**
  `CursorPagination` and `TimePagination` became `TimeSeriesPagination` with a
  typed `TimeSeriesLinks`, so one walker covers every time-ordered collection.
- **`GET /v3/token` added** — credential introspection requiring no scope,
  returning kind, scopes, account_id, and project_ids.
