# Vendored OpenAPI spec

`bundled.yaml` is a **copy** of the Honeybadger v3 OpenAPI bundle. It is the
input to `make generate`, which writes `internal/gen/gen.go`.

## Provenance

| | |
| --- | --- |
| Source repo | `honeybadger` (the Rails app) |
| Path | `openapi/v3/bundled.yaml` |
| Branch | `scoped-api-tokens-v3` |
| Commit | `fb16b9143` (rebased branch) |
| Vendored | 2026-07-29 |
| sha256 | `8103c20eff1ec28ab6debeb11d99a9f0cce44eae7b083e9f43606bc042d2766e` |

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

The full list, with what the client does about each, is in
[GAPS.md](GAPS.md). The summary below covers only what affects generation.

### Write bodies that are narrower than v2

These request schemas declare fewer fields than v2 accepted. A field absent from
the schema does not appear in the generated request type at all, so `apiv3`
cannot send it — the method follows the spec rather than inventing fields, and
`apiv3/writes.go` documents each gap at the method.

| Operation | Declared | v2 also accepted |
| --- | --- | --- |
| `assignFault` | bare `type: object` | the assignee |
| `createAlarm` / `updateAlarm` | `name` | query, evaluation period, trigger config, lookback lag, streams |
| `createDashboard` / `updateDashboard` | `name` | title, default_ts, widgets |
| `createCheckIn` / `updateCheckIn` | name, schedule_type, report_period, grace_period | slug, cron schedule, timezone |
| `updateProject` | `name` | resolve_errors_on_deploy, disable_public_links, user_url, source_url, purge_days, user_search_field |

`Faults.Update` and `Faults.Assign` are absent from `apiv3` entirely: with a bare
`type: object` there is nothing to build a typed method from, and guessing field
names would produce a method that compiles and silently does nothing. The other
fault state changes do not need them — v3 replaced v2's mutable PUT with discrete
endpoints (`resolve`, `unresolve`, `ignore`, `unignore`, `pause_recording`,
`resume_recording`, `merge`), all of which are fully specified.

### `created_before` cannot express a timestamp

`ListCheckInEvents`' `created_before` is `type: number`, which generates a
`float32`. A float32 has a 24-bit mantissa, so at current epoch values its
precision is coarser than two minutes — paging by it would skip or repeat events.
`apiv3` does not expose the parameter, and walks by following links instead.
Wants `int64`, or an opaque cursor like the other time-ordered collections.

### Otherwise clean The bundle now validates — the source repo gained
`lib/openapi/validator.rb` and a rake task, so a malformed bundle fails there
rather than here.

`apiv3` still treats `code` as an open string rather than the generated enum. The
enum has grown from 10 to 20 values during v3's development, so a closed set would
silently drop codes the client has not caught up with.

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
