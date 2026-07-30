# v2 capabilities not expressible in v3

> **Most of this list is closed.** The bundle at `fb16b9143` closed
> `resolve_on_deploy` and gave the fault bulk operations a `q` filter. The one
> before it, `7295ca79c`, added the missing
> write fields, three of the four missing endpoints, the fault ordering and time
> filters, and extracted the error body into named schemas. What remains is at the
> top; the closed items are kept below because the history explains why the client
> is shaped as it is.

## Still open

| Gap | Detail |
| --- | --- |
| `get_project_report` has no v3 endpoint | No path matches `reports/`. The MCP tool stays on v2, needing a legacy numeric id nothing can now discover |
| Integrations unconfirmed | `listChannels` looks like the replacement — same shape, and v2's own comment says "integrations (channels)" — but nobody has confirmed it |
| Alarm updates are name and description only | `AlarmCreateInput` carries the query, trigger and evaluation settings; `AlarmUpdateInput` carries neither, so an alarm's behaviour cannot be changed after creation |
| A check-in update requires the name | The update body is the same schema as create, with `name` required, so changing only a grace period means resending the name. A caller who does not know it must read first. Verified against a real server: the update **merges** — sending only name and grace_period left `report_period` and `slug` intact — so it is the required `name` that is the burden, not field loss |
| A project update requires the name | Same shape, same consequence |
| Widget and trigger types are anonymous | `DashboardInput.widgets` and `AlarmCreateInput.trigger_config` are inline objects, so a generated Go caller cannot construct them. `apiv3` passes widgets through as raw JSON and hand-rolls the trigger. Naming those schemas would remove both workarounds — and would also remove the overlay below |
| `Dashboard.widgets.items` breaks code generation | It `$ref`s `#/components/schemas/DashboardInput/properties/widgets/items`, a JSON Pointer into another schema's properties. Legal OpenAPI, and it says something true, but `oapi-codegen` refuses it: `unexpected reference depth: 7`. `openapi/overlay.yaml` rewrites the node to a plain object so generation can proceed. Extracting a named `DashboardWidget` component would fix this and the row above it at once |
| Notices have no timestamp filters | Cursor-only (`limit`, `before`, `after`), so v2's `created_after`/`created_before` have no equivalent |
| Channels carry less than v2 integrations did | v2's integration reported `options` and `filters`; the v3 `Channel` has neither, so per-integration configuration is no longer readable. The MCP tool says so in its description |
| Fault update semantics are unstated | `updateFault` says only "Updates a fault's attributes". `FaultInput` has no required fields, which implies a merge, and every other replace-semantics body requires `name` — but nothing says so. It matters because the MCP sends `resolve_on_deploy` on its own: under replace semantics that call would also clear the fault's tags and unassign it. Proceeding on the merge reading, since the alternative makes the field unusable, but this wants one sentence in the spec |
| Writes replace rather than merge — but only some of them | Verified against a real server, the three differ: a **check-in** update merges (omitted fields survive); a **dashboard** update replaces (`config["widgets"] = data["widgets"].map` overwrites wholesale, and omitting widgets would clear them, which is why `apiv3` refuses it); a **project** update's omitted settings cannot be observed at all because the read model does not return them. All three take the create schema and require `name`, so the schema says "replace" uniformly while the behaviour does not. Documenting per-resource semantics — or adding a patch schema — would remove the guesswork |
| Dashboard widget ids are regenerated when omitted | `widget_hash["id"] ||= SecureRandom.uuid`, so a caller that reads a dashboard, edits a widget and sends it back without its `id` silently gets new widget identities. Verified: a round-trip changed `3a86735f…` to `3dcc065f…` |

## Closed

Tracked here because the list is the input to finishing the migration, and because
each item is a decision someone has to make on the API side rather than a client
workaround. Discovered while porting `api-go`'s `apiv3` package and the MCP
server onto the v3 bundle; every entry was verified against the vendored spec, not
inferred.

Bundle at time of writing: `scoped-api-tokens-v3` @ `fb16b9143`, 115 operations.

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
| `updateFault` | resolved, ignored, tags, assignee_id, resolve_on_deploy | — closed |
| `assignFault` | bare `type: object` | the assignee |
| `createAlarm` / `updateAlarm` | `name` | query, evaluation period, trigger config, lookback lag, streams, description |
| `createDashboard` / `updateDashboard` | `name` | title, default_ts, widgets |
| `createCheckIn` / `updateCheckIn` | name, schedule_type, report_period, grace_period | slug, cron schedule, timezone |
| `updateProject` | `name` | resolve_errors_on_deploy, disable_public_links, user_url, source_url, purge_days, user_search_field |

`createAlarm` is the sharpest case. An alarm without a query or trigger never
fires, so creating one leaves something broken in the account that looks real —
the MCP tool refuses rather than half-doing it. A project with only a name is
still a project, so that one proceeds and rejects only the unsettable fields.

`resolve_on_deploy` was the sharpest of these — absent from the bundle entirely,
so a missing capability rather than a narrow schema. `fb16b9143` added it to both
`FaultInput` and `Fault`, and the MCP advertises it again. It is stored as a
pending resolution rather than a column, and resolving or ignoring a fault clears
it, so `update_fault` refuses to accept it together with `resolved` or `ignored`
instead of letting whichever request ran last decide.

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

- **No OAuth client can create a project, and every one of them can delete
  projects.** `createProject` is the only operation in the catalog requiring a
  `:create` scope — every other create maps to `:write` (`createAlarm` →
  `alarms:write`, `createCheckIn` → `checkins:write`).

  This is not about old grants. Doorkeeper declares `default_scopes :read, :write`
  and **no `optional_scopes`**, so a granular scope cannot be requested or issued
  at all; every `oauth_applications` row is `read write`. Every token therefore
  goes through `Api::LegacyScopes` expansion, and `V1["write"]` does not include
  `projects:create` — so `createProject` is unreachable for all OAuth clients,
  permanently, while `projects:write` keeps `updateProject` and `deleteProject`
  available.

  Verified end to end: with a token issued minutes earlier, the MCP hides
  `create_project` and still offers `delete_project`. The scope filtering is
  correct; the catalog and the OAuth configuration disagree.

  **The fix is to add `projects:create` to `V1["write"]`**, and it is permitted by
  that list's own DO-NOT-EDIT rule rather than an exception to it. v2 gates every
  write on one check — `!token.can_write?` in `ApiController#enforce_oauth_scope!`,
  where `can_write?` is `scopes.exists?("write")` — so any v2 `write` grant could
  `POST /v2/projects`. There was no separate create permission. Omitting
  `projects:create` therefore narrows existing grants rather than holding a line
  against widening them, which is the `uptime:*` case verbatim: "omitting it would
  have narrowed every existing grant … adding it preserves the surface rather than
  extending it."

  Adding it to `write` only; `read` should not gain it. Offering the granular
  catalog through Doorkeeper `optional_scopes` is the larger, separate fix — it
  would let a client ask for less, but it does not repair the grants already
  issued. Until this lands, the destructive half of project management is the only
  half OAuth can reach.

- **`update_alarm` cannot succeed with only a name.** `observer_params` derives
  `lookback_duration` from `params[:evaluation_period]`, so an update that does
  not resend the evaluation period fails with `lookback_duration: can't be
  blank`. `AlarmUpdateInput` declares only name and description, so a client
  following the spec always gets a 422. Either the schema should carry the
  observer fields, or the server should default them from the existing observer.

- **A failed alarm update still commits the name.** `AlarmService#update` runs
  `alarm.update(...)` unconditionally, after `update_observer` has already
  returned errors, so a 422 leaves the alarm renamed while the query and trigger
  are unchanged. Verified: a name-only update answered 422 and the name changed
  anyway. A partial write reported as a failure is worse than either outcome on
  its own — the caller has no way to know what landed.

- **`Alarm.query` declares an object and renders a string.** The presenter reads
  `observer_payload&.dig("query")`, which is BadgerQL text. `AlarmCreateInput`
  already types it as a string, so the read and write halves disagree with each
  other. Overlaid; the create succeeded and only the decode failed.

- **Alarm durations have an undocumented format.** `evaluation_period` and
  `lookback_lag` are declared as bare strings with no pattern or example, and the
  API rejects `5 minutes` with `format is invalid`. It wants a compact duration —
  `5m`, `10m`, `1d`. `lookback_lag` is also refused when blank, though nothing
  marks it required. A create following the spec alone cannot succeed.

- **`listFaultAffectedUsers` declares an object and renders an array.** The
  response schema says `data: {type: object}`; the controller renders
  `data: users`, an array of `{user, count}` pairs. Decoding as an object fails
  outright against a real server. `apiv3` models the array.

- **`data` is optional on single-resource responses**, so `{}` decodes as a valid
  project with an empty id. `apiv3` treats a missing `data` member as an error, but
  the schema should require it.
- **Required fields are not enforced by decoding.** The spec marks `id`,
  `account_id`, `name`, and `active` required on `Project`, but generated
  non-pointer fields silently become zero values when absent, so `{"data":{}}`
  would otherwise return a project with an empty id.
- **The bulk endpoints' spec and the app disagree about ids plus query.** The spec
  says `q` is "applied when fault_ids is omitted", implying ids win. The Rails
  `bulk_collection` builds the query-filtered collection first and then applies
  `fault_ids` to it, which intersects them. Either behaviour is defensible; a
  destructive operation should not have two readings. `apiv3` makes the case
  unrepresentable — `FaultSelection`'s fields are unexported and its constructors
  set one or the other — so no api-go caller depends on the answer, but the spec
  and the app should still be reconciled.

- **The fault bulk endpoints promise time filters they do not declare.** All four
  say "omit to act on everything the query and time filters match", but they
  declare only `account_id` and `project_id`, and their body carries only
  `fault_ids` and `q`. So there is no way for a client to send a time filter, and
  no way to tell whether the sentence describes a missing parameter or is left
  over from `listFaults`. `apiv3` documents `q` alone. Either add
  `occurred_after`/`occurred_before` to the body or drop the clause — as written
  a caller can reasonably expect a bounded change and get a project-wide one.

- **787 anonymous inline structs** in the generated models, from inline object
  schemas. A consumer cannot name or construct those types. Extracting them into
  named component schemas would shrink hand-written wrapper code and improve the
  published docs. `Error.Error` and `Error.Meta` are the cases that bite most.

## Fixed since this list started

Closed by the bundle at `1ebb7621d`:

- `CheckIn.report_period` and `grace_period` are typed as the interval strings the
  app sends, with examples. The overlay that corrected them locally is removed —
  the spec is now the authority again.
- `Project` reports `resolve_errors_on_deploy`, `user_url`, `source_url`,
  `user_search_field` and `customer_throttle`, and `V3::ProjectPresenter` renders
  all five. The settings are no longer write-only, so a caller can confirm what it
  wrote and read current values before a partial update.


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
