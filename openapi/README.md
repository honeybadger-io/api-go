# Vendored OpenAPI spec

`bundled.yaml` is a **copy** of the Honeybadger v3 OpenAPI bundle. It is the
input to `make generate`, which writes `internal/gen/gen.go`.

## Provenance

| | |
| --- | --- |
| Source repo | `honeybadger` (the Rails app) |
| Path | `openapi/v3/bundled.yaml` |
| Branch | `feat/terraform-3-v3-contract` (honeybadger#6730, draft, stacked on #6729) |
| Commit | `6d07eea82e` (local to the honeybadger checkout; not pushed yet) |
| Vendored | 2026-10-06 |
| sha256 | `bcce7c7f52d4efb0124232da46dac2f2b6ced1bd50bc7a68f561ec829fd3c09c` |

Record the branch, commit, **and checksum** on every refresh. The bundle is
**gitignored in the source repo** (`.gitignore:86`) — it is a build artifact of
`rake openapi:bundle`, not a tracked file — so there is no upstream blob to diff
against.

The checksum is machine-checked: `openapi/bundled.yaml.sha256` holds it and
`make verify-spec` compares the two, so a bundle that quietly changed underneath
the provenance table fails the build instead of being described by a stale table.
Use `make update-spec-checksum` when vendoring deliberately, in the same commit
as the table update. `make verify` runs it alongside the codegen drift gate.

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
make update-spec-checksum
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

Open gaps, deliberate decisions and the history are in [GAPS.md](GAPS.md). None
of them affect generation: the bundle validates, every write body generates a
plain struct that `apiv3` aliases directly, and the only overlay left is the
deliberate `Error.details` mapping.

`apiv3` treats `code` as an open string rather than the generated enum. The enum
grew from 10 to 20 values during v3's development, so a closed set would silently
drop codes the client hasn't caught up with.

### Fixed since first vendoring

- **`Error.details` is a `oneOf`** — an array of field errors for
  `validation_error`, or an object naming the missing permission for
  `insufficient_scope`. It was previously typed as an array while the
  insufficient_scope example sent an object, so a real 403 could not decode.
  `overlay.yaml` still retypes it to `json.RawMessage`, now by choice rather than
  necessity: the generator renders a `oneOf` as a union wrapper with As*/From*
  accessors, and `apiv3` already discriminates on the error code.

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
