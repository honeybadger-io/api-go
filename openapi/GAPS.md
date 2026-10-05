# v3 gaps and decisions

This started as the list of what v2 could do that v3 could not. Nearly all of it
is closed: the bundle now covers every capability the client and the MCP server
used from v2, and a series of `spec-fixes` rounds in September 2026 reshaped the
write bodies so that `apiv3`'s inputs are plain aliases of the generated types.

What remains is below: a short open list, the decisions made not to change
things, and the history, kept because it explains why the client is shaped as it
is. Every entry was verified against the vendored bundle or a running server, not
inferred.

The client's rule where a gap exists is unchanged: **refuse the request and say
why**. Accepting a call and quietly dropping what it can't send reports success
for something that didn't happen.

## Open

| Gap | Detail |
| --- | --- |
| An invalid alarm trigger is dropped on update (Opticon) | On Opticon `main`, `Observer#update_or_version` returns the observer unchanged when only the new trigger config is invalid, so a v3 `PATCH` with `operator: "zz"` or a negative `value` answers 200, applies the rest of the body and keeps the old trigger. It contradicts the spec's operator enum and `value >= 0`. The `observer-26-field-threshold` branch replaces this with a plain save that 422s. Deferred until that lands |
| New check-ins and sites join every integration | Creating a check-in or site appended it to every integration's `check_in_ids`/`site_ids` (`after_create :add_to_channels`), so an empty or narrowed list lasted only until the next one. The vendored spec replaces this with `all_sites`/`all_check_ins`: an integration follows every site or check-in, including ones added later, or exactly the listed ones, and an empty list with the flag off means none. That comes from honeybadger #6729/#6730, still drafts; close this when they merge |
| Alarm `state` lags its history (Opticon) | `state` stays `initial` after history already records an evaluation. It comes from Opticon's trigger messages, not the v3 presenter |
| Notices have no time filters | `listNotices` pages by cursor only (`limit`, `before`, `after`), so v2's `created_after`/`created_before` have no equivalent |
| Fault update semantics are unstated | `updateFault` says only "Updates a fault's attributes". `FaultInput` has no required fields, which implies a merge, and the MCP relies on that when it sends `resolve_on_deploy` alone. Wants one sentence in the spec |
| Bulk fault ids plus a query | The spec says `q` and the time filters apply "when fault_ids is omitted", implying ids win; the app has intersected them. `FaultSelection` makes the combination unrepresentable, so no `apiv3` caller depends on the answer, but a destructive operation shouldn't have two readings |

## Not testable locally yet

- **Merged faults.** A write to a merged fault answers 409 `fault_merged` with the
  survivor in `details.merged_into`, and a read answers 301; `Error.MergedInto`
  reads either. Covered by unit tests, but the seed data has no merged fault.
- **Credential variations.** Write-only, scoped and OAuth tokens, and HTTP-mode
  confirmation secrets, need credentials the local pass didn't have.

## Decided: not changing

Raised during the September 2026 end-to-end pass and deliberately left as is.

- **Masked secrets reveal a prefix.** Integration secrets come back with the first
  third visible and the full length preserved. It's the web UI's existing policy.
  Sending the exact mask back leaves the secret unchanged; a hand-edited mask is
  stored as a new secret, since the API can't tell it from one.
- **Webhook URLs are returned in full**, including ones that act as credentials
  (Discord and the like).
- **An empty time-series page past the oldest cursor** reports `has_newer: true`
  without a cursor to follow. You only reach it by ignoring `has_older: false`.
- **The project's `token` is its newest key.** `/projects/{id}/keys` is the full
  list.

## Closed

### September 2026 (`spec-fixes`)

Integrations:
- The create body is one flat object (`type`, the shared settings, and `config`
  for the type's settings) instead of a 34-way `oneOf`, and update is the same
  without `type`. Per-type rules sit in `if`/`then` blocks and an
  `x-integration-types` extension that generators ignore. `apiv3` aliases both.
- Settings sent outside `config` are refused, unknown ids in `site_ids`,
  `check_in_ids` and the alarm id lists are refused with 422 (an empty list turns
  those notifications off), environment names are stored as given, the
  `environments` shorthand is gone, `included_environments` and the filters are
  readable, and fixed-list options and `threshold` are validated.
- OAuth types can be created: they start turned off and unconnected, and can't be
  turned on until connected at `links.web`.

Dashboards:
- `PATCH` merges: omitted fields keep their values, and a sent `widgets` list
  replaces the set (keep each widget's `id` to keep its identity). Only `title` is
  required on create.
- The widget input is a named `DashboardWidgetInput` whose `config` is a plain
  object, with per-type config schemas for validation. A GET's widgets can be sent
  straight back.

Naming and types:
- Fault ids are opaque strings, like every other v3 id: the path parameter,
  `Fault.id`, the `fault_id` on notices and comments, bulk `fault_ids`, merge ids
  and `details.merged_into`. The server still accepts numbers on input.
- `CheckInScheduleType`, `AlarmTriggerCondition`, `PauseDuration`, `FaultMerge`,
  `FaultMergeInput` and `CheckInReplaceInput` are named components, so
  `CheckInParams`, `AlarmParams`, `AlarmTrigger` and `FaultMerge` stopped being
  hand-written.
- `AlarmTriggerConfig` requires `type` and `config`, `config` requires `operator`
  (an enum) and `value`, and a trigger is sent whole: an update's trigger replaces
  the stored one.
- Unknown ids in any request body are refused with 422 rather than dropped.
- Plan-gated fields a plan doesn't include are refused with 403
  `feature_unavailable` rather than ignored.
- `replaceCheckIns` is all-or-nothing.
- `listProjects` takes an exact `name`.
- Occurrence reports are typed (`OccurrenceSeries`), and an unknown `period` is a
  400.
- `created_before` is a double, so it can carry a real timestamp. `apiv3` still
  walks by `links.older`, as the spec recommends.

Earlier in the month:
- Writes to merged faults answer 409 with the survivor rather than a 301 a
  redirect-following client would replay.
- Snooze, unsnooze, pause and resume return the fault, with
  `recording_paused_until`.
- `getProjectKey` exists; affected users are capped at 500, documented.
- Alarm history is typed and pages like every other numbered collection; alarm
  updates can change the trigger.

### Before September 2026

- **Endpoints**: fault counts, occurrence reports (per project and account-wide)
  and integrations all have v3 endpoints; project reports were dropped in favour of
  Insights.
- **Write bodies**: alarms, dashboards, check-ins, projects and fault assignment
  declare everything v2 accepted, including `resolve_on_deploy`.
- **List parameters**: faults take `order` and the time filters; affected users
  take `q`.
- **Dashboards** read and write `title` (the spec once wrote it as `name`).
- **OAuth**: the frozen `write` grant includes `projects:create`, so an OAuth
  client can create a project, not only update and delete one.
- **Alarm updates** merge onto the stored observer, so a name-only update works,
  and a failed update no longer commits the name.
- **Responses**: `data` is required on single-resource responses; anonymous
  generated structs went from 787 to 23.
- **The bundle** validates (the source repo has a validator and rake task), uses
  `bearer_auth` only, declares the full error code enum, has one
  `TimeSeriesPagination`, and gained `GET /v3/token`, Insights, Streams, and `me`
  as an account id with `ambiguous_account`.
- **`Error.details`** is a `oneOf`; `overlay.yaml` maps it to `json.RawMessage` by
  choice, since `apiv3` already discriminates on the error code.
