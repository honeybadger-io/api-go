# Transitioning api-go and the MCP server to Honeybadger API v3

Date: 2026-07-29
Status: Largely built. `apiv3` is complete against the spec as written, and every
MCP tool with a v3 endpoint has been migrated. Sections marked "what was built
differs" record where implementation diverged from the plan and why.

Outstanding, all server-side: the gaps listed in [../../../openapi/GAPS.md](../../../openapi/GAPS.md).
Four MCP tools still run on v2 because v3 has no endpoint for them.

Not yet done on the client side: verification against a real server. Every test
is a hand-written fixture, so this proves the client matches the *document*, not
that the document matches the Rails app.
Scope: `github.com/honeybadger-io/api-go` and `honeybadger-mcp-server`. The
server-side v3 API is designed elsewhere; this document treats
`openapi/v3/bundled.yaml` in the honeybadger repo as its authority.

> The bundle is being actively edited — it grew from 5985 to 6152 lines during
> the writing of this document, and `me`/`ambiguous_account` landed mid-review.
> This document therefore cites **operationIds and schema names**, not line
> numbers, for anything in the spec. Go source citations use file:line.

## Context

`api-go` is a hand-written Go client for the Honeybadger v2 Data API. Every
request goes through `client.newRequest()`, which hardcodes the version:

```go
url := fmt.Sprintf("%s/v2%s", c.baseURL, path)   // client.go:94
```

`honeybadger-mcp-server` pins `api-go v0.8.0` and registers 34 MCP tools, of
which 32 are API-backed (`get_reference` and `search_tools` are not). The CLI is
a second consumer. The module is pre-1.0 (`go 1.23`).

v3 is specified in OpenAPI 3.1: 6152 lines, 111 operations, 16 tags. It differs
from v2 in seven ways that matter to a client:

| Concern | v2 | v3 |
| --- | --- | --- |
| Base | `api.honeybadger.io/v2` | `app.honeybadger.io/v3` |
| Auth | personal token (Basic or Bearer) | Bearer only: OAuth or scoped tokens; personal tokens rejected |
| Envelope | `{results: [...]}` | `{data, pagination, links, meta}` |
| Identifiers | integers (`12345`) | opaque strings (`Xk9mZp`) |
| Path scoping | `/projects/{id}/faults` | `/accounts/{account_id}/projects/{project_id}/faults`, with `me` accepted |
| Pagination | `page`/`per_page` | offset *and* time-ordered (`limit` plus `links.older`) |
| Errors | untyped body | `{error: {code, message, details}, meta: {request_id}}`, 20 codes |

Auth has the widest operational blast radius: personal auth tokens stop working,
so every consumer needs a new credential, not just new code. See decision 9.

**v3 does not yet cover the whole MCP tool surface.** Four API-backed tools have
no v3 operation — `get_fault_counts`, `get_project_occurrence_counts`,
`get_project_report`, `get_project_integrations`. The only related v3 operation is
`getProjectStats`, and whether it subsumes any of them is unestablished; likewise
whether Channels replaces integrations. An earlier draft of this document claimed
full coverage on the basis of the tag list. That was wrong — tags are not routes.
See prerequisite 1.

## Decisions

### 1. `apiv3` is added alongside the existing root package

```
github.com/honeybadger-io/api-go                     # v2 services (unchanged)
github.com/honeybadger-io/api-go/apiv3               # v3: generated models + hand services
github.com/honeybadger-io/api-go/internal/gen        # generated low-level ops
```

The root package is untouched, so v0.8.0 consumers keep compiling. `apiv3` is
purely additive. `Nullable[T]`, `Value`, and `Null` stay in root, imported by
`apiv3`; no cycle, because root never imports `apiv3`.

Naming: `apiv3`, not `v3`. A `v3/` subdirectory makes the import path
`github.com/honeybadger-io/api-go/v3` — which is also what module major version 3
of this module would be called. Go resolves such an import by trying successively
shorter module prefixes, so it can work today, but it is ambiguous to readers and
forecloses ever publishing a real major version 3. `apiv3` follows Google's
generated-client convention (`cloud.google.com/go/aiplatform/apiv1`) and avoids
the question.

**Deferred: namespacing v2 as `apiv2` (the former stage 0).** Symmetric
`apiv2`/`apiv3` packages with root reduced to primitives is a tidier end state,
and pre-1.0 is the cheapest moment to break imports. But it breaks every consumer
before v3 delivers anything, it is not the pure move it first appears (transport
extraction rides along), and the real problems in this transition are elsewhere.
Revisit at an actual major-version boundary.

**Rejected: a single client with a runtime version switch.** Not idiomatic Go,
and not implementable honestly — v3 changes both identifier types and the
response envelope, so one set of types cannot serve both versions without `any`.
A `WithVersion()` setter would silently change what `Projects.Get` returns.

**Rejected: an immediate hard cut to v3-only.** Leaves no rollback path and
forces MCP, CLI, and external consumers to move on the same day.

### 2. The version lever lives in the MCP server, behind two factories

**What was built differs from what this section proposed**, and the reason is
worth recording. The plan was MCP-owned DTOs behind port interfaces, so a v2 and
a v3 adapter could satisfy one signature. In practice, changing `ClientFactory`'s
type broke all 34 call sites at once, which would have meant a broken build for
the length of a 5,000-line migration with no way to test progress.

What replaced it: a second factory, `V3ClientFactory`, alongside the existing
one. Each tool file moved independently — insights and streams first, then
projects, faults, alarms, check-ins, dashboards — with the build and tests green
at every step. Handlers take the concrete client rather than an interface.

The DTO layer was not needed because handlers already map API types into their
own response structs before serialising, so response shape changes stayed local
to that mapping.

The v2 factory remains for the four tools with no v3 endpoint, and goes away when
they have one.

### 2b. Original proposal, for the record: MCP-owned DTOs

`internal/hbmcp` gains narrow port interfaces per resource group, with two
implementations — one over the root package, one over `apiv3`.

**This is not just interface extraction.** Go interfaces require identical
signatures, so a port method cannot return `hbapi.Project` in one implementation
and `apiv3.Project` in the other. Handlers today take `*hbapi.Client` directly
(`server.go:13`) and build concrete `hbapi` request types throughout
(`projects.go:247`, `faults.go:283`). The ports must therefore be defined in
terms of **MCP-owned DTOs**, with each adapter converting into them. Skipping
that makes the v3 adapter a handler rewrite rather than a drop-in — the whole
point of the stage.

### 3. Generated models are public; only the low-level operations are private

`oapi-codegen` emits models and operations from the vendored spec. Models are
re-exported as `apiv3`'s public types; the low-level operation layer stays in
`internal/gen`. Hand-written services add ergonomics on top: pagination loops,
`*Options` query structs, unwrapping `{data, ...}` into
`ListResponse[T]{Data, Pagination, Links, Meta}`.

An earlier draft kept *all* generated code private behind hand-written public
types. Across 111 operations and ~45 schemas, that is a second complete model
layer — every type redefined and converted field by field. Not a facade; a second
SDK. Public generated models with hand-written services is the narrower boundary
that still gets v2's ergonomics.

Two rules survive from that draft:

1. **The spec is vendored and drift is a test failure.** `make generate`
   regenerates from a checked-in bundle; CI regenerates and fails on a dirty
   tree. Spec updates arrive as deliberate PRs — important given how fast the
   bundle is currently moving.
2. **Go-specific concerns stay out of the producer spec.** Use `oapi-codegen`'s
   OpenAPI Overlay support, checked into api-go, rather than `x-go-type`
   annotations in the honeybadger repo or a post-generation patch script. The
   earlier draft framed this as a choice between annotations and patch scripts;
   overlays are the third option and the right one. Confirm in stage 1.

### 4. Identifiers are plain `string` in `apiv3`

Not a named `type ID string`, and not per-resource types (`ProjectID`,
`FaultID`). Opaque identifiers never do arithmetic, and per-resource types would
need per-property overlay entries.

Trade-off accepted: nothing prevents passing a fault ID where a project ID
belongs. If that bug appears in practice, named types are a mechanical change.

### 5. Account defaults to `me`, with a per-call override

v3 project resources are account-scoped, and v3 now accepts `me` as an
`account_id`, resolving it from the token.

**`me` is not always sufficient.** Per the `AccountId` parameter description,
`me` resolves only when the credential covers exactly one account; a credential
covering several returns 422 `ambiguous_account` and requires a concrete id.
An earlier draft of this document assumed all tokens are single-account and
dropped account from the facade entirely. That is wrong for multi-account
credentials.

So:

- `apiv3` sends `me` by default, and every service method takes an optional
  account override (an `*Options` field or a client-level default, not a
  positional parameter). Common case stays `Faults.List(ctx, projectID, opts)`.
- MCP tools gain an **optional** `account_id`. This does not break existing
  callers, and two tools already have one — `list_projects` and `create_project`
  (`projects.go:22`, `projects.go:56`). An earlier draft claimed no tool had an
  `account_id`; that claim came from a grep for numeric params and missed these
  two `WithString` declarations.
- MCP needs a `list_accounts` tool. Without it, a multi-account user who hits
  `ambiguous_account` has no way to discover a valid id, and the error is a dead
  end. v3 has `listAccounts`; nothing exposes it.
- `ambiguous_account` maps to a message telling the model to call
  `list_accounts` and retry with an explicit id.

The account segment is worth keeping in canonical paths: account-level resources
need it (`listAccountMembers`, teams, status pages), logs read the account
without a join, and multi-account credentials demonstrably exist.

**Rejected: unscoped `/projects/{project_id}/...` alias paths.** Doubles the
surface of a 111-operation spec; generated clients would offer two ways to do
everything.

**Rejected: client-side account resolution and caching.** In MCP's http mode the
server is multi-tenant with the token arriving per request, so the cache would
have to be keyed by hashed token, bounded, and TTL'd — and keying it wrong serves
one customer's account id to another. `me` plus an explicit override removes the
risk class instead of managing it.

Independent of all this: keep the server-side check that a project belongs to the
credential's account. That check is what enforces isolation.

### 6. MCP tool schemas change with the transport, and only for opted-in clients

30 tools declare numeric identifiers — 34 `mcp.WithNumber("…id")` declarations
and 32 `req.GetInt("…id")` reads across `alarms.go`, `checkins.go`,
`dashboards.go`, `insights.go`, `faults.go`, `projects.go`, `streams.go`. They
move to `WithString`/`GetString` together with the transport swap.

**A global default flip is not safe**, because MCP clients cache tool schemas
until they reconnect. Live http sessions would keep sending integers to a server
that now expects opaque strings, and the failure surfaces as validation errors on
every call. Two consequences:

- Expose v3 as an **opt-in profile** — a separate endpoint or server profile —
  rather than flipping the default under existing sessions.
- Make the default change a versioned, announced release, not a config toggle,
  and measure reconnections before retiring v2.

**Rejected: flipping schemas to string early, while still on v2.** At cutover the
identifier *values* change (`12345` → `Xk9mZp`), so any saved prompt with a
literal id breaks regardless. The early flip would add `strconv` parsing at 32
sites and a schema that lies about its domain, for no reduction in real breakage.

**Rejected: accepting number-or-string in tool schemas permanently.** Doubles
validation paths and gives the model two ways to be right. Acceptable only as a
temporary dispatch during the opt-in window, if measurement shows stale sessions
are a real problem.

### 7. Typed errors; request IDs are best-effort, not guaranteed

v3 errors are typed, with `code` a **10**-value enum: `unauthorized`,
`access_denied`, `not_found`, `validation_error`, `rate_limit_exceeded`,
`maintenance_mode`, `invalid_id`, `forbidden_attributes`, `service_unavailable`,
`ambiguous_account`.

Today's `WrapError` guesses at the message — it pokes `body["message"]`, falls
back to `body["errors"]`, then stuffs the raw string in (`errors.go:38-48`).
`apiv3` decodes the envelope instead.

- **`apiv3.Error`** carries `Code`, `Message`, `Details []FieldError`,
  `RequestID`, `StatusCode`. Package-level sentinels (`ErrNotFound`,
  `ErrRateLimited`, `ErrAmbiguousAccount`, …) with an `Is()` method let MCP write
  `errors.Is(err, apiv3.ErrNotFound)` instead of matching strings.
- **`request_id` is not on every response.** 204s have no body, `meta` is
  optional on most operations, and Insights and Streams declare empty `meta`
  objects. So the hook is best-effort and must be **context-aware** —
  `func(ctx context.Context, resp *http.Response, id string)` — so MCP can
  correlate the calls that do carry one and log the ones that don't. A bare
  `func(id string)` cannot correlate anything.
- **Rate limiting**: 360/hour/user, `X-RateLimit-*`. These appear only in the
  spec's prose, not as declared response `headers`, so nothing generated will
  surface them — see prerequisite 5. `RateLimitError` carries `Reset` and
  `RetryAfter()`; retry is opt-in (`WithRetryOn429(n)`) and stays off in MCP,
  where a multi-minute block reads as a hang.

The root package's `Number` type, which absorbs string-or-int (`types.go:12`),
stays for v2 notice payloads. `apiv3` does not use it; v3 is strict.

### 8. The Insights query endpoint is passed through, not modelled

**Request is unchanged from v2.** `{query, ts, timezone, stream_ids[]}` matches
`InsightsQueryRequest` (`insights.go:14-21`) field for field. Only the path and
account scope move.

**Response `data` is `additionalProperties: true`** — the query service's output,
passed through, varying by query. `apiv3` decodes it to `map[string]any`. v2's
typed `InsightsQueryMeta` (`insights.go:23-31`) is not reintroduced as a
hand-written type: the spec explicitly declines to guarantee it, so a struct
would be an unverifiable claim.

**Query errors move to 422 and *do* carry a code.** v2 returned HTTP 200 with an
inline `error` field. v3 returns 422 whose schema is `allOf: [Error, {error:
additionalProperties}]`. `allOf` intersects rather than replaces, so `Error`'s
required `code` and `message` still apply — the query service's detail is
*additional*, not a substitute. An earlier draft claimed this response had no
`code` and needed a bespoke error type; that was a misreading of `allOf`. It
decodes as a normal `apiv3.Error` with extra fields preserved. If the intent was
actually a free-form body, the spec should stop using `allOf` — worth confirming.

**One intentional behavior break.** v3 rejects a `stream_id` not belonging to the
project; v2 silently dropped it. Parity tests must encode that as an expected
difference.

Stream identifiers are 12-character lowercase hex strings assigned by the
Insights backend — explicitly not public IDs, and already `[]string` in v2. (An
earlier draft called them ULIDs; wrong, though harmless, since both are opaque
strings.) `Stream.slug` is an enum (`default`/`internal`).

### 9. `apiv3` accepts Bearer tokens only

v3 rejects personal auth tokens. Only OAuth access tokens and the new granular
scoped tokens are accepted, both as `Authorization: Bearer <token>`. Scoped
tokens are opaque — no client-readable structure.

`apiv3` exposes one auth method, `WithBearerToken`, and **not** `WithAuthToken`.
The rejected path is absent from the v3 surface by construction. The root package
keeps `WithAuthToken` because v2 still accepts personal tokens.

**MCP http mode already satisfies this.** `newClientFactory` uses
`WithBearerToken` with the request's token (`server.go:96-98`); those tokens are
`hbo_`-prefixed RS256 JWTs, audience-bound per RFC 8707, scopes from the `scope`
claim (`claims.go:29-50`).

**MCP stdio mode is the migration problem.** It builds
`WithAuthToken(cfg.AuthToken)` (`server.go:102-104`) from a single public
interface: the `--auth-token` flag bound to `HONEYBADGER_PERSONAL_AUTH_TOKEN`
(`main.go:87`, `main.go:137`). v2 does not accept scoped tokens and v3 does not
accept personal ones, so the credential is **not interchangeable**:

- Keep `--auth-token` / `HONEYBADGER_PERSONAL_AUTH_TOKEN` meaning the v2
  credential. Do not repurpose it.
- Add an explicitly named v3 credential alongside it.
- Fail fast at startup when the selected adapter's credential is absent, with a
  message naming the missing flag. Otherwise every tool call returns 401.
- Do not make v3 the stdio default until credential migration is measurable.
  "Both tokens stay provisioned" is a hope about other people's installs, not a
  property this server can guarantee.

**Scope gating turned out to be possible after all.** This section originally
concluded it was not, on the grounds that opaque scoped tokens cannot be
inspected. `GET /v3/token` landed afterwards and changed that: introspection
requires no scope, so any credential can be asked what it holds.

What was built instead of the asymmetry described here:

- The MCP server classifies the credential by prefix and verifies what it can.
  OAuth tokens keep full JWT verification; opaque tokens are forwarded for the
  API to judge.
- Introspection results are cached per credential digest with a short TTL and an
  LRU bound, so scopes cost one call per credential per window rather than one
  per request. That matters because the server is hosted: the cache is derivable
  and disposable, never state a request depends on.
- The advertised tool catalog is filtered per request from those scopes, using a
  tool-to-operation map plus `apiv3.OperationScopes`, which is generated from the
  spec's per-operation `security` declarations.
- Where scopes are unknown — introspection unavailable — nothing is filtered and
  the API refuses what it must. Absent knowledge is not absent permission.

In stdio there is still no per-request credential, so `--read-only` decides
there.

**403 handling is required regardless.** A token may permit faults but not
check-ins, and scopes can be revoked mid-session, so filtering can never be
sufficient. `apiv3` surfaces `access_denied` as a typed error; how useful the
message is depends on prerequisite 6.

## Prerequisites (honeybadger repo)

Two earlier items — the invalid bundle (`_index.yaml` inlined as `Index`
components) and missing Insights/Streams operations — are **done**. Adding
`redocly lint` to that repo's CI is still worthwhile: the invalid bundle was
caught by reading structure, not by a validator.

### 1. Route-and-feature parity matrix — BLOCKS stage 3

Four API-backed MCP tools have no v3 operation: `get_fault_counts`,
`get_project_occurrence_counts`, `get_project_report`,
`get_project_integrations`. `getProjectStats` exists and is untyped; whether it
covers any of them is unestablished, as is whether Channels replaces
integrations.

Wanted: an explicit table of every v2 route MCP or the CLI calls, mapped to its
v3 operation or marked intentionally dropped. Prose assurance is what produced
the false coverage claim in the first place. Until this exists, stage 4 cannot
ship the current catalog without silently dropping tools.

### 2. Complete the write schemas — BLOCKS stage 3

Several v3 write operations exist by name but do not specify what MCP sends:

- `updateFault` — request body is a bare `type: object` with no properties. MCP
  sends resolved, ignored, assignee, resolve-on-deploy.
- `createAlarm` / `updateAlarm` — body specifies only `name`. MCP sends query,
  evaluation period, trigger configuration, lookback lag, streams
  (`alarms.go:237`).
- Dashboard writes — body specifies `name`; MCP sends `title`, `default_ts`,
  `widgets` (`dashboards.go:182`). The `title` vs `name` divergence needs
  resolving, not guessing.
- Check-in writes — omit slug, cron schedule, timezone that MCP sends
  (`checkins.go:202`).

Generated request structs drop unspecified fields, so this is not cosmetic: the
adapters would silently send incomplete writes.

### 3. Confirm `me` semantics — needed before the facade is written

`me` plus `ambiguous_account` has landed. Confirm the intended reality: are user
and account tokens always single-account, with `ambiguous_account` reachable only
by OAuth credentials? The answer decides whether the account override is a rare
escape hatch or a routine parameter, and whether MCP's `list_accounts` tool is
core or a fallback.

### 4. Update the spec's documented auth

The spec still documents Basic as `base64(YOUR_API_TOKEN:)` pointing at user
settings, and `basic_auth` remains in top-level `security:`. Narrow to
`bearer_auth`, name the two accepted token types, state that personal tokens are
rejected.

Note: this is a documentation-correctness fix, not a codegen-safety one. An
earlier draft claimed generated clients would emit a dead Basic-auth option;
`oapi-codegen` supplies auth via request editors instead, so nothing dangerous is
generated either way.

### 5. Declare the rate-limit headers, and distinguish 401 causes

`X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset` appear only in
prose. Declared as response `headers`, they become part of the generated surface
instead of something every client hand-rolls.

Separately: a personal token on v3 is a *permanent* failure needing a new
credential; an expired OAuth token needs a refresh. Both are `unauthorized`
today, so clients must guess. A distinct code or a stable message resolves it.

### 6. Name the required scope in the 403 body — nice-to-have

Scoped tokens are opaque, so the first signal is a refusal. `error.details`
naming the missing scope turns "access denied" into "this token lacks
`checkins:write`".

### 7. Scope declarations, per operation — nice-to-have

Not expressible as written: `bearer_auth` is `type: http, scheme: bearer`, which
has no scope support. An `oauth2` scheme with a named scope list plus
per-operation `security` entries would make required scopes generated and
documented rather than tribal.

### 8. Audit pagination component references — nice-to-have

Both offset and cursor pagination components exist. A list operation referencing
the wrong one silently breaks client pagination loops. Cheap to audit now.

### 9. v2 sunset date, if any

Determines how long the root package's services stay and whether v2 should emit
`Sunset` headers.

## Staging

**Stage 1 — codegen spike** (api-go, throwaway branch).
Pin and test current `oapi-codegen`, which advertises native OpenAPI 3.1
support including `type: [T, "null"]` — start there rather than assuming a
3.1→3.0 downconversion is needed. Confirm three things: 3.1 unions generate
usefully, Overlay files can carry the Go-specific mappings, and the nullable
generation option produces types that behave correctly on decode (see below).
Deliverable: a decision record and a working `make generate`.

> **Nullable, verified.** `*Nullable[T]` does **not** preserve three states on
> decode. `encoding/json` sets a nil pointer field to nil for both explicit
> `null` and an absent key, without calling `UnmarshalJSON`. Confirmed
> empirically: unmarshalling `{"a":null}` into `struct{A *Nullable[string]}`
> yields `A == nil`, identical to the absent case. `nullable_test.go` only
> unmarshals into a bare `Nullable[T]`, never a `*Nullable[T]` struct field, so
> the gap was untested. `Nullable[T]` remains correct and useful for **requests**
> — omitted vs null vs value on marshal, which is what `nullable.go`'s own
> comment claims. Response types needing to distinguish null from absent need a
> different mechanism (a value-typed wrapper with a `Present` flag).

**Stage 2 — ports, DTOs, and the v2 adapter** (MCP).
Define MCP-owned DTOs and per-resource-group port interfaces; implement over the
root package. Tool schemas untouched, numeric identifiers unchanged. Per decision
2, the DTOs are the substance of this stage — interface extraction alone does not
give stage 4 a drop-in.

**Stage 3 — the `apiv3` package** (api-go).
Vendored spec, generated models, hand-written services, `/v3` transport,
rate-limit headers, typed errors, context-aware request-id hook, contract tests.
Ships as a new package nothing consumes yet. Blocked on prerequisites 1 and 2.

**Stage 4 — v3 adapter, opt-in** (MCP).
Second adapter with string identifiers and Bearer credentials; tool schemas move
to `WithString`/`GetString` here, with the transport. Exposed as an opt-in
profile per decision 6 — not a default flip under live sessions. Retiring v2
waits on measured client reconnection and credential migration.

Ordering: stage 1 first, since its answer shapes stage 3 and it blocks on
nothing. Stage 2 is independent of stages 1 and 3 and can run in parallel.

## Testing

**Unit tests in the existing style.** `httptest.NewServer` plus table tests,
matching `client_test.go`. `apiv3` tests assert path, query, request body, and
envelope decoding.

**Contract tests from the spec.**
- *Drift gate*: CI runs `make generate` and fails on a dirty tree.
- *Decode tests*: the bundle does **not** carry a complete response example per
  operation — most `example:` entries are property-level, and the reusable error
  responses are the main complete ones. So "one generated test per operation" is
  not constructible as an earlier draft assumed. What works: hand-assembled
  fixtures per resource, decoded with `DisallowUnknownFields` so a renamed field
  fails instead of being silently ignored by `json.Unmarshal`.

**Golden responses from a real server.** The Rails app is the truth; the spec is
a claim about it. A `make record` target captures real v3 responses from staging
into `testdata/`.

Redaction is mandatory: `Project.token` is the error-reporting API key, and
member and invitation payloads carry real email addresses. `make record` redacts
before writing, and the PR adding `testdata/` needs a review pass for leaked
secrets.

**Live e2e — the existing rig cannot gate anything yet.** `make docker-local`
builds MCP against a local api-go checkout, and `HONEYBADGER_API_URL` is
configurable, so pointing at a local Rails serving `/v3` works. But the existing
9 tests do not assert success: they start the server with the literal token
`test-token` and check only the MCP envelope shape — content array present, type
is `text`, text is a string (`e2e/e2e_test.go:204-232`, comment: "we just verify
we get a valid response structure"). Handlers return upstream failures as
`CallToolResult` error content, so a 401 or 404 passes green.

Before e2e can gate a cutover it must: assert `isError == false`, decode and
validate v3 payloads, use real opaque identifiers, cover every port method
including writes, and exercise 403, 422 `ambiguous_account`, and 429. Provision
a deliberately under-scoped token as a second fixture so the 403 path is covered
by a test rather than a support ticket.

**Parity tests, temporary.** Per ported resource, one test calling v2 and v3 for
the same object, asserting the fields MCP surfaces agree. Note the hole: v3
exposes only opaque ids with no v2 integer mapping, so these need an explicit
fixture mapping rather than "the same object" resolving itself. Known intentional
divergences to encode as expected differences: invalid `stream_ids` (rejected in
v3, dropped in v2) and Insights query errors (422 in v3, inline on a 200 in v2).

**Not building:** a mock server generated from the spec. It validates the
generator against itself and passes while the real application disagrees.

## Open questions

1. **Generator choice.** Unresolved until stage 1 runs.
2. **Cursor pagination in the facade.** Whether `ListAll` covers offset and
   cursor with one signature or splits into two iterators. Deferred to stage 3.
3. **Response-side nullability.** Which v3 fields genuinely need null-vs-absent
   discrimination on decode? If few or none, the simplest answer is plain
   pointers and no new mechanism.
4. **v2-only services.** `deployments.go`, `comments.go`, `environments.go` have
   v3 equivalents but no MCP tools. Port for parity, or only what MCP and the CLI
   call?
5. **CLI migration.** Out of scope here, but it is a second consumer needing its
   own stage-4 equivalent plus a credential swap.
6. **Scope-to-tool mapping.** http mode filters tools with one coarse `write`
   check (`server.go:20`). Granular scopes could drive per-tool filtering, but
   only for introspectable JWTs, not opaque scoped tokens. Decide during stage 4.

## Risks

| Risk | Mitigation |
| --- | --- |
| Spec route coverage incomplete | Prerequisite 1's matrix; stage 3 blocked until it exists |
| Write schemas incomplete, adapters send lossy payloads | Prerequisite 2; `DisallowUnknownFields` fixtures catch regressions after |
| Bundle changes under the implementation | Vendored copy plus drift gate; cite operationIds not line numbers |
| Stale cached tool schemas break live sessions | Opt-in profile rather than default flip (decision 6) |
| stdio installs break on credential change | Keep `--auth-token` as the v2 credential, name the v3 one separately, fail fast (decision 9) |
| Multi-account credentials hit `ambiguous_account` | Account override in the facade, optional `account_id` on tools, `list_accounts` exposed (decision 5) |
| e2e suite green while every request 401s | Rewrite assertions before using e2e as a gate |
| Secrets leaking into `testdata/` | Redaction in `make record`, review pass on the adding PR |
| MCP callers with hardcoded numeric ids | Unavoidable; announce with the opt-in release |
