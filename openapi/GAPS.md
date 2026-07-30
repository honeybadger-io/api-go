# v2 capabilities not expressible in v3

Tracked here because the list is the input to finishing the migration, and because
each item is a decision someone has to make on the API side rather than a client
workaround. Discovered while porting `api-go`'s `apiv3` package and the MCP
server onto the v3 bundle; every entry was verified against the vendored spec, not
inferred.

Bundle at time of writing: `scoped-api-tokens-v3` @ `01a704e5b`, 112 operations.

The client's behaviour where a gap exists is uniform: **refuse the request and say
why**. Accepting a call and silently dropping what it cannot send would report
success for something that did not happen, which is the one outcome worse than an
error.

## 1. Endpoints with no v3 equivalent

| v2 endpoint | Used by | Notes |
| --- | --- | --- |
| `GET /projects/{id}/faults/summary` | MCP `get_fault_counts` | `listFaultOccurrences` exists but is per-fault and takes no `q`; this is project-wide counts *with* the search filter |
| `GET /projects/{id}/occurrences` | MCP `get_project_occurrence_counts` | Also had an all-projects variant, which has no possible v3 shape since every path is account+project scoped |
| `GET /projects/{id}/reports/{type}` | MCP `get_project_report` | No operationId matches `report` |
| `GET /projects/{id}/integrations` | MCP `get_project_integrations` | Probably `listChannels` — the shapes line up and v2's own comment says "integrations (channels)" — but nobody has confirmed Channels is the replacement rather than an adjacent concept |

These four tools still run on the v2 client, which is why they keep numeric project
ids while everything else moved to opaque strings.

## 2. Write bodies narrower than v2

A field absent from the schema does not exist in the generated request type, so it
cannot be sent at all.

| Operation | Declared | v2 also accepted |
| --- | --- | --- |
| `updateFault` | bare `type: object` | resolved, ignored, assignee, resolve-on-deploy |
| `assignFault` | bare `type: object` | the assignee |
| `createAlarm` / `updateAlarm` | `name` | query, evaluation period, trigger config, lookback lag, streams, description |
| `createDashboard` / `updateDashboard` | `name` | title, default_ts, widgets |
| `createCheckIn` / `updateCheckIn` | name, schedule_type, report_period, grace_period | slug, cron schedule, timezone |
| `updateProject` | `name` | resolve_errors_on_deploy, disable_public_links, user_url, source_url, purge_days, user_search_field |

`createAlarm` is the sharpest case. An alarm without a query or trigger never
fires, so creating one leaves something broken in the account that looks real —
the MCP tool refuses rather than half-doing it. A project with only a name is
still a project, so that one proceeds and rejects only the unsettable fields.

`resolve_on_deploy` deserves its own line: it appears **nowhere** in the bundle, so
it is not a narrow schema but a missing capability.

## 3. List parameters with no v3 equivalent

| Operation | Missing vs v2 |
| --- | --- |
| `listFaults` | `order` (recent/frequent), `created_after`, `occurred_after`, `occurred_before`. v3 takes only `page`, `per_page`, `q` |
| `listNotices` | `created_after`, `created_before` — notices page by cursor instead |
| `listFaultAffectedUsers` | `q` |

Ordering is the one with no workaround: `q` can express environment and status
filters, but there is no documented way to ask for the most frequent faults.

`limit` maps cleanly onto `per_page` and needs nothing.

## 4. A type that cannot represent its values

`listCheckInEvents`' `created_before` is `type: number`, which generates a
`float32`. A float32 has a 24-bit mantissa, so at current epoch values its
precision is coarser than two minutes — paging by it would skip or repeat events.

`apiv3` does not expose the parameter and walks by following links instead. Wants
`int64`, or an opaque cursor like the other time-ordered collections.

## 5. The same field under two names

`Dashboard` is read as `title` and written as `name` — `Dashboard.title` in the
response schema, `name` in `createDashboard`/`updateDashboard`. A client has to
translate between them for one resource, and a caller reading a dashboard cannot
use the field name it just received to write one back.

The MCP tool keeps accepting `title`, since that is what v2 used, and maps it to
`name` on the way out.

## 6. Schema inconsistencies worth a second look

- **`data` is optional on single-resource responses**, so `{}` decodes as a valid
  project with an empty id. `apiv3` treats a missing `data` member as an error, but
  the schema should require it.
- **Required fields are not enforced by decoding.** The spec marks `id`,
  `account_id`, `name`, and `active` required on `Project`, but generated
  non-pointer fields silently become zero values when absent, so `{"data":{}}`
  would otherwise return a project with an empty id.
- **787 anonymous inline structs** in the generated models, from inline object
  schemas. A consumer cannot name or construct those types. Extracting them into
  named component schemas would shrink hand-written wrapper code and improve the
  published docs. `Error.Error` and `Error.Meta` are the cases that bite most.

## Fixed since this list started

Kept for context, since several were fixed within hours of being reported.

- The bundle was invalid OpenAPI — `_index.yaml` files inlined as `Index`
  components. Fixed, and the source repo gained a validator plus a rake task.
- The error code enum was missing `insufficient_scope`, `credential_in_query`, and
  `project_restricted`. Now 20 codes.
- `Error.details` was typed as an array while `insufficient_scope` sent an object.
  Now a `oneOf`.
- Basic auth is gone; `security` is `bearer_auth` only, with a dedicated
  `unsupported_auth_scheme` code.
- `CursorPagination` and `TimePagination` collapsed into one `TimeSeriesPagination`
  with typed links, so one walker covers every time-ordered collection.
- `GET /v3/token` added — introspection requiring no scope, which is what makes
  scope-aware tool filtering and `ambiguous_account` recovery possible.
- Insights query and Streams gained v3 endpoints.
- `me` accepted as an `account_id`, with `ambiguous_account` for credentials
  covering several accounts.
