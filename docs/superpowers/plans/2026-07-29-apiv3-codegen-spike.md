# apiv3 Codegen Spike Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Establish a reproducible, drift-gated code generation pipeline from the v3 OpenAPI bundle into `internal/gen`, and record — as executable tests — exactly how the generated types behave, so stage 3 can be designed against verified facts instead of assumptions.

**Architecture:** A vendored copy of the v3 bundle plus an OpenAPI Overlay lives under `openapi/`. `make generate` runs a pinned `oapi-codegen` over them and writes a single generated file to `internal/gen`. Hand-written characterization tests in that package assert the decode behavior stage 3 depends on: envelope shape, opaque identifiers, null-vs-absent, and the `allOf` error schema. CI regenerates and fails if the tree is dirty.

**Tech Stack:** Go 1.24+, `github.com/oapi-codegen/oapi-codegen/v2` v2.8.0 (generator, run via `go run`, not a module dependency), `github.com/oapi-codegen/runtime` v1.6.0 (generated-code dependency), standard `testing`.

## Global Constraints

- Generator pinned to `github.com/oapi-codegen/oapi-codegen/v2@v2.8.0` — exact version, everywhere it appears.
- `go` directive in `go.mod` moves from `1.23` to `1.24`. Forced: `github.com/oapi-codegen/runtime` v1.6.0 declares `go 1.24.0`. This is a consumer-visible floor bump and must appear in release notes.
- CI matrix `go-version` moves from `"1.23"` to `"1.24"` in lockstep (`.github/workflows/test.yml:17`).
- The vendored bundle is a **copy**, never a symlink. Source of truth is `openapi/v3/bundled.yaml` in the honeybadger repo, which is actively changing.
- Generated code goes only to `internal/gen`. Nothing outside that directory may be generated, and no generated file may be hand-edited.
- Do not create `apiv3/` in this plan. Stage 1 establishes the pipeline only; the public facade is stage 3 and is blocked on prerequisites 1 and 2 of the design doc.
- Existing package name is `honeybadgerapi` (root). The generated package is `gen`.
- Tests use the standard library only — no testify. Match the existing style in `client_test.go`.

---

## File Structure

| File | Responsibility |
| --- | --- |
| `openapi/bundled.yaml` | Vendored copy of the v3 spec. Input to generation. Never edited by hand. |
| `openapi/overlay.yaml` | OpenAPI Overlay carrying Go-specific concerns, keeping them out of the producer spec. |
| `openapi/codegen.yaml` | `oapi-codegen` configuration: package name, output path, what to emit. |
| `Makefile` | `generate`, `test`, `verify-generated` targets. api-go has no Makefile today. |
| `internal/gen/gen.go` | Generated models and low-level client. Committed, never hand-edited. |
| `internal/gen/doc.go` | Hand-written package doc stating the file is generated and how to regenerate. |
| `internal/gen/characterization_test.go` | Tests pinning decode behavior stage 3 relies on. |
| `internal/gen/overlay_test.go` | Test proving the overlay actually took effect. |
| `.github/workflows/test.yml` | Gains a drift gate; Go version bumped. |
| `docs/superpowers/decisions/2026-07-29-codegen-spike-findings.md` | The decision record. Written last, from test results. |

---

### Task 1: Vendored spec, pinned generator, and `make generate`

**Files:**
- Create: `openapi/bundled.yaml` (copied)
- Create: `openapi/codegen.yaml`
- Create: `Makefile`
- Create: `internal/gen/doc.go`
- Create: `internal/gen/gen.go` (generated output, committed)
- Modify: `go.mod`

**Interfaces:**
- Consumes: nothing.
- Produces: package `github.com/honeybadger-io/api-go/internal/gen`, containing generated types including `Project`, `Fault`, `Account`, `Stream`, `Error`, and a generated `ClientWithResponses`. Later tasks import this package. Also produces `make generate` and `make verify-generated`.

- [ ] **Step 1: Copy the bundle in**

```bash
mkdir -p openapi internal/gen
cp /Users/kevinwebster/Projects/honeybadger/honeybadger/openapi/v3/bundled.yaml openapi/bundled.yaml
head -3 openapi/bundled.yaml
```

Expected: first line is `# GENERATED — do not edit. Run `rake openapi:bundle`.`

- [ ] **Step 2: Write the generator config**

Create `openapi/codegen.yaml`:

```yaml
# Config for oapi-codegen. Run via `make generate`.
package: gen
output: internal/gen/gen.go
generate:
  models: true
  client: true
output-options:
  # Keep every schema, including ones no operation references yet. The bundle
  # is still growing; pruning would make regeneration output churn.
  skip-prune: true
```

- [ ] **Step 3: Write the Makefile**

Create `Makefile`:

```makefile
# Generator is pinned. Do not float this version — regenerating with a
# different generator produces diff noise that hides real spec changes.
OAPI_CODEGEN := github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.0

.PHONY: generate verify-generated test

generate:
	go run $(OAPI_CODEGEN) -config openapi/codegen.yaml openapi/bundled.yaml

# Fails when the committed generated code does not match the committed spec.
verify-generated: generate
	git diff --exit-code --stat internal/gen/gen.go

test:
	go test ./...
```

- [ ] **Step 4: Bump the Go directive**

Edit `go.mod` so the `go` line reads:

```
go 1.24
```

Reason: `github.com/oapi-codegen/runtime` v1.6.0 declares `go 1.24.0`. Without this, `go build` fails on the generated code's imports.

- [ ] **Step 5: Generate**

Run: `make generate`
Expected: exits 0, creates `internal/gen/gen.go` of roughly 1MB. The file begins with `// Package gen provides primitives to interact with the openapi HTTP API.` and `// Code generated by github.com/oapi-codegen/oapi-codegen/v2 version v2.8.0 DO NOT EDIT.`

- [ ] **Step 6: Resolve the new dependencies**

Run: `go mod tidy`
Expected: `go.sum` gains `github.com/oapi-codegen/runtime v1.6.0`, plus `github.com/apapsch/go-jsonmerge/v2` and `github.com/google/uuid` as its transitive deps.

- [ ] **Step 7: Write the package doc**

Create `internal/gen/doc.go`:

```go
// Package gen contains code generated from the vendored Honeybadger v3
// OpenAPI bundle at openapi/bundled.yaml.
//
// Do not edit gen.go. Regenerate with `make generate` after updating the
// vendored bundle. This package is internal on purpose: generator output is
// not a stable API, and stage 3's public apiv3 package is what consumers see.
package gen
```

- [ ] **Step 8: Verify it builds and the existing suite still passes**

Run: `go build ./... && go test ./...`
Expected: both exit 0. The root package's existing ~100 tests pass unchanged — nothing in this task touches v2 code.

- [ ] **Step 9: Commit**

Commit **before** testing the drift gate. `git diff` ignores untracked files, so a gate tested against an untracked `internal/gen/gen.go` cannot fail and proves nothing.

```bash
git add openapi/bundled.yaml openapi/codegen.yaml Makefile internal/gen/gen.go internal/gen/doc.go go.mod go.sum
git commit -m "feat: generate v3 API models from vendored OpenAPI bundle

Pins oapi-codegen v2.8.0 and generates into internal/gen. The 3.1 bundle
parses natively, so no 3.0 downconversion is needed.

Bumps the go directive to 1.24, forced by oapi-codegen/runtime v1.6.0."
```

- [ ] **Step 10: Verify the drift gate actually fails when it should**

Now that both files are tracked, confirm the clean case:

Run: `make verify-generated`
Expected: exits 0, no diff.

Then prove it can fail. Edit a schema description, since descriptions become doc comments in generated output:

```bash
sed -i '' 's/description: Project name/description: Project name CHANGED/' openapi/bundled.yaml
make verify-generated; echo "EXIT:$?"
```

Expected: non-zero exit, with a diff shown in `internal/gen/gen.go`.

Restore and regenerate:

```bash
git checkout openapi/bundled.yaml && make generate && git diff --exit-code --stat internal/gen/gen.go
```

Expected: final command exits 0 — tree back to the committed state.

---

### Task 2: Characterization tests for decode behavior

Stage 3's design depends on four claims about the generated types. This task turns each into a test, so a later generator or spec change that breaks one fails loudly.

**Files:**
- Create: `internal/gen/characterization_test.go`

**Interfaces:**
- Consumes: package `gen` from Task 1 — types `Project`, `AccountInvitation`, `Error`, `Stream`.
- Produces: nothing importable. Its output is knowledge, recorded in Task 5.

- [ ] **Step 1: Write the failing test file**

Create `internal/gen/characterization_test.go`:

```go
package gen

import (
	"encoding/json"
	"testing"
)

// v3 identifiers are opaque strings, not integers. Stage 3's facade signatures
// depend on this: a numeric id must not silently decode.
//
// Note Id is a plain string, not *string: the spec marks it required, and
// oapi-codegen emits required properties as values.
func TestProjectIDIsOpaqueString(t *testing.T) {
	var p Project
	if err := json.Unmarshal([]byte(`{"id":"Xk9mZp","name":"My Rails App"}`), &p); err != nil {
		t.Fatalf("decoding opaque id: %v", err)
	}
	if p.Id != "Xk9mZp" {
		t.Errorf("Id = %q, want %q", p.Id, "Xk9mZp")
	}

	// A numeric id is a type error, not a silent coercion.
	var p2 Project
	err := json.Unmarshal([]byte(`{"id":12345}`), &p2)
	if err == nil {
		t.Error("numeric id decoded without error; expected a type error")
	}
}

// The design records that generated nullable fields cannot distinguish an
// explicit null from an absent key. Stage 3 must not assume three-state
// decoding. This test documents the limitation rather than asserting a fix.
func TestNullAndAbsentAreIndistinguishable(t *testing.T) {
	var explicitNull AccountInvitation
	if err := json.Unmarshal([]byte(`{"accepted_at":null}`), &explicitNull); err != nil {
		t.Fatalf("decoding explicit null: %v", err)
	}

	var absent AccountInvitation
	if err := json.Unmarshal([]byte(`{}`), &absent); err != nil {
		t.Fatalf("decoding absent field: %v", err)
	}

	if explicitNull.AcceptedAt != nil {
		t.Errorf("explicit null gave non-nil %v; generator behavior changed", explicitNull.AcceptedAt)
	}
	if absent.AcceptedAt != nil {
		t.Errorf("absent gave non-nil %v", absent.AcceptedAt)
	}
	// Both nil: the two cases are indistinguishable. If this ever fails,
	// the generator gained three-state support and the design can be revisited.
}

// The error envelope carries a machine-readable code. Stage 3's typed
// sentinels depend on code being present and required.
func TestErrorEnvelopeCarriesCode(t *testing.T) {
	body := `{"error":{"code":"not_found","message":"Resource not found"},"meta":{"request_id":"abc123"}}`
	var e Error
	if err := json.Unmarshal([]byte(body), &e); err != nil {
		t.Fatalf("decoding error envelope: %v", err)
	}
	if e.Error.Code != "not_found" {
		t.Errorf("Code = %q, want %q", e.Error.Code, "not_found")
	}
	if e.Error.Message != "Resource not found" {
		t.Errorf("Message = %q, want %q", e.Error.Message, "Resource not found")
	}
}

// Unknown fields are ignored by default. Contract tests that need to catch a
// renamed field must opt into DisallowUnknownFields; this test proves the
// default is permissive, so nobody relies on it failing.
func TestUnknownFieldsIgnoredByDefault(t *testing.T) {
	var s Stream
	if err := json.Unmarshal([]byte(`{"slug":"default","totally_new_field":1}`), &s); err != nil {
		t.Fatalf("unknown field caused an error: %v", err)
	}
	if s.Slug == nil || *s.Slug != "default" {
		t.Error("slug did not decode alongside an unknown field")
	}
}
```

- [ ] **Step 2: Run the tests**

Run: `go test ./internal/gen/ -v -run 'TestProjectIDIsOpaqueString|TestNullAndAbsentAreIndistinguishable|TestErrorEnvelopeCarriesCode|TestUnknownFieldsIgnoredByDefault'`

Expected: all four PASS. Any FAIL is a real finding, not a test bug — record the actual behavior in Task 5 and adjust the assertion to match reality, keeping the comment explaining what was expected.

Three generated shapes the tests above already account for, confirmed against real output:
- `Error.Error` is a nested **anonymous** struct, not a named type. Field access compiles; the type simply cannot be named or constructed from another package. `Error.Error.Code` is a named enum type `ErrorErrorCode`, and comparing it to an untyped string literal is valid Go.
- `Stream.Slug` is `*StreamSlug`, a named string type. `*s.Slug != "default"` compiles for the same reason.
- Required properties generate as **values**, optional ones as pointers. `Project.Id`, `Project.AccountId`, `Project.Name`, and `Project.Active` are non-pointer; `Project.Token` and `Project.CreatedAt` are pointers.

- [ ] **Step 3: Fix any compile errors by matching generated reality**

Run: `go vet ./internal/gen/`
Expected: exits 0. If a field name differs from the plan (for example `Id` vs `ID`), use the generated name — `oapi-codegen` produces `Id`, not `ID`.

- [ ] **Step 4: Commit**

```bash
git add internal/gen/characterization_test.go
git commit -m "test: pin generated v3 decode behavior

Records four facts stage 3 depends on: opaque string ids, null and absent
being indistinguishable, the error envelope's required code, and unknown
fields being ignored by default."
```

---

### Task 3: Prove the overlay mechanism works

Design decision 3 keeps Go-specific concerns out of the producer spec by using an OpenAPI Overlay. That is currently an assumption. This task makes it a demonstrated fact — or kills it early.

**Files:**
- Create: `openapi/overlay.yaml`
- Create: `internal/gen/overlay_test.go`
- Modify: `openapi/codegen.yaml`
- Modify: `Makefile`
- Modify: `internal/gen/gen.go` (regenerated)

**Interfaces:**
- Consumes: package `gen` from Task 1.
- Produces: a working overlay wired into `make generate`, and the knowledge of whether overlays can carry type mappings.

- [ ] **Step 1: Write the overlay**

Create `openapi/overlay.yaml`. The target selects the `Project` schema's `name` property and attaches a Go type extension:

```yaml
overlay: 1.0.0
info:
  title: Go-specific mappings for api-go
  version: 0.0.1
actions:
  # Proof of mechanism: force a distinctive Go type onto one field. If this
  # appears in generated output, overlays can carry the mappings stage 3 needs
  # without putting Go concerns in the producer spec.
  - target: $.components.schemas.Project.properties.name
    update:
      x-go-name: ProjectDisplayName
```

- [ ] **Step 2: Wire the overlay into the config**

Modify `openapi/codegen.yaml`, adding to the top level:

```yaml
output-options:
  skip-prune: true
  overlay:
    path: openapi/overlay.yaml
```

Keep `package`, `output`, and `generate` exactly as they were.

- [ ] **Step 3: Write the failing test**

Create `internal/gen/overlay_test.go`:

```go
package gen

import (
	"encoding/json"
	"testing"
)

// The overlay renames Project.name's Go field. If this compiles and passes,
// overlays are a viable place for Go-specific mappings.
//
// The field is a plain string, not *string: name is required in the spec.
func TestOverlayAppliedToGeneratedCode(t *testing.T) {
	var p Project
	if err := json.Unmarshal([]byte(`{"name":"My Rails App"}`), &p); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if p.ProjectDisplayName != "My Rails App" {
		t.Errorf("got %q, want %q", p.ProjectDisplayName, "My Rails App")
	}
}
```

- [ ] **Step 4: Run it and watch it fail**

Run: `go test ./internal/gen/ -run TestOverlayAppliedToGeneratedCode`
Expected: FAIL to compile — `p.ProjectDisplayName undefined`. The overlay has not been applied to the committed generated code yet.

- [ ] **Step 5: Regenerate with the overlay**

Run: `make generate`
Expected: exits 0. Confirm the rename landed:

```bash
grep -n "ProjectDisplayName" internal/gen/gen.go | head -3
```

Expected: at least one hit, inside `type Project struct`.

If the overlay is silently ignored — no hits, exit 0 — that is the finding. Do **not** try moving `overlay:` to the top level of the config: v2.8.0's config parser is strict and rejects unknown top-level keys, so `output-options.overlay` as written in Step 2 is the only valid placement.

Instead, isolate which half failed by putting the extension directly in the vendored bundle:

```bash
cp openapi/bundled.yaml /tmp/bundled.backup.yaml
# Add x-go-name next to Project.properties.name, then:
make generate && grep -c "ProjectDisplayName" internal/gen/gen.go
cp /tmp/bundled.backup.yaml openapi/bundled.yaml && make generate
```

`x-go-name` is honored on properties by v2.8.0 (`pkg/codegen/utils.go`, `Property.GoFieldName`). So:
- Extension works inline but not via overlay → the overlay mechanism is the problem. Record that stage 3 needs producer-spec annotations or a patch step.
- Extension fails inline too → the target path or extension name is wrong, not the mechanism. Fix and retry before concluding anything.

- [ ] **Step 6: Run the test again**

Run: `go test ./internal/gen/ -run TestOverlayAppliedToGeneratedCode -v`
Expected: PASS.

- [ ] **Step 7: Run the whole suite**

Run: `go test ./...`
Expected: exits 0. Task 2's tests still pass — the overlay touched one field's Go name, not its JSON tag, so decoding is unaffected. If `TestProjectIDIsOpaqueString` broke, the overlay did more than intended; narrow the target.

- [ ] **Step 8: Commit**

```bash
git add openapi/overlay.yaml openapi/codegen.yaml internal/gen/gen.go internal/gen/overlay_test.go
git commit -m "feat: apply Go-specific mappings via OpenAPI overlay

Proves overlays can carry Go type concerns, keeping them out of the
producer spec in the honeybadger repo."
```

---

### Task 4: Evaluate `nullable-type` for three-state decoding

The design concluded that null-vs-absent cannot be distinguished on decode. That holds for plain generation, but v2.8.0 has an `output-options: nullable-type` flag that emits `nullable.Nullable[T]` from `github.com/oapi-codegen/nullable` — a value type with an explicit present/null distinction. This task establishes whether it works, because it directly answers open question 3 in the design doc.

**Files:**
- Create: `internal/gen/nullable_behavior_test.go`
- Modify: `openapi/codegen.yaml` (temporarily, then reverted or kept per outcome)

**Interfaces:**
- Consumes: package `gen` from Task 1.
- Produces: a decision on whether `apiv3` response types can distinguish null from absent, recorded in Task 5.

- [ ] **Step 1: Enable the option**

In `openapi/codegen.yaml`, add to `output-options`:

```yaml
  nullable-type: true
```

- [ ] **Step 2: Regenerate and inspect what changed**

Run: `make generate && grep -c "nullable.Nullable\[" internal/gen/gen.go`
Expected: a non-zero count. Confirm the import landed:

Run: `grep -n "oapi-codegen/nullable" internal/gen/gen.go`
Expected: one hit in the import block.

If the count is 0, the option had no effect on this spec — record that and revert. The spec uses 3.1 `type: [T, "null"]` unions rather than 3.0's `nullable: true`, and the generator may only honour the latter. That is the single most important thing this task can discover.

- [ ] **Step 3: Resolve the new dependency**

Run: `go mod tidy && go build ./...`
Expected: both exit 0. `go.sum` gains `github.com/oapi-codegen/nullable`.

- [ ] **Step 4: Write the test**

Create `internal/gen/nullable_behavior_test.go`. Adjust the type and field to whatever `nullable.Nullable` actually landed on — find one with:
`grep -n "nullable.Nullable\[" internal/gen/gen.go | head -5`

```go
package gen

import (
	"encoding/json"
	"testing"
)

// Does nullable-type generation distinguish explicit null from an absent key?
// This is the question the design doc's open question 3 asks.
func TestNullableTypeDistinguishesNullFromAbsent(t *testing.T) {
	// Replace AccountInvitation/AcceptedAt with a field that generated as
	// nullable.Nullable[T], per the grep above.
	var explicitNull AccountInvitation
	if err := json.Unmarshal([]byte(`{"accepted_at":null}`), &explicitNull); err != nil {
		t.Fatalf("decoding explicit null: %v", err)
	}
	var absent AccountInvitation
	if err := json.Unmarshal([]byte(`{}`), &absent); err != nil {
		t.Fatalf("decoding absent: %v", err)
	}

	if !explicitNull.AcceptedAt.IsSpecified() {
		t.Error("explicit null: IsSpecified() = false, want true")
	}
	if !explicitNull.AcceptedAt.IsNull() {
		t.Error("explicit null: IsNull() = false, want true")
	}
	if absent.AcceptedAt.IsSpecified() {
		t.Error("absent: IsSpecified() = true, want false")
	}
}
```

- [ ] **Step 5: Run it**

Run: `go test ./internal/gen/ -run TestNullableTypeDistinguishesNullFromAbsent -v`
Expected: PASS, proving three-state decode is available.

If the method names differ, check the actual API:
`go doc github.com/oapi-codegen/nullable.Nullable`
and use what exists. A compile error here is a naming problem, not a finding.

- [ ] **Step 6: Decide and record**

Two outcomes, both legitimate:
- **Works** → keep `nullable-type: true`. Task 2's `TestNullAndAbsentAreIndistinguishable` will now fail, because behavior genuinely changed. Update that test to assert the new behavior and note in its comment that plain generation conflates the two. Run `go test ./...` and confirm green.
- **No effect on this spec** → revert `nullable-type: true` from the config, delete `internal/gen/nullable_behavior_test.go`, regenerate, and confirm `go test ./...` is green with Task 2 unchanged.

- [ ] **Step 7: Commit**

For the "works" case:

```bash
git add openapi/codegen.yaml internal/gen/gen.go internal/gen/nullable_behavior_test.go internal/gen/characterization_test.go go.mod go.sum
git commit -m "feat: generate nullable fields as nullable.Nullable[T]

Three-state decode (absent, null, value) is available on response types
after all, which the design doc had concluded was impossible."
```

For the "no effect" case:

```bash
git add openapi/codegen.yaml internal/gen/gen.go
git commit -m "test: record that nullable-type does not apply to 3.1 null unions

The option only honours 3.0-style nullable: true, so response types
cannot distinguish null from absent. Config reverted."
```

---

### Task 5: CI drift gate and the decision record

**Files:**
- Modify: `.github/workflows/test.yml`
- Create: `docs/superpowers/decisions/2026-07-29-codegen-spike-findings.md`

**Interfaces:**
- Consumes: `make verify-generated` from Task 1; test results from Tasks 2, 3, and 4.
- Produces: the stage 1 deliverable named in the design doc — a decision record plus a working, gated `make generate`.

- [ ] **Step 1: Bump the CI Go version and add the drift gate**

In `.github/workflows/test.yml`, change the matrix line (currently `go-version: ["1.23"]`, line 17) to:

```yaml
        go-version: ["1.24"]
```

Then add this step after the existing `Install dependencies` step:

```yaml
      - name: Verify generated code matches the vendored spec
        if: matrix.os == 'ubuntu-latest'
        run: make verify-generated
```

The `if` keeps it to one platform: generator output is identical across platforms, and running it three times only triples the chance of a network flake in `go run`.

- [ ] **Step 2: Verify the workflow is valid YAML**

Run: `go run github.com/mikefarah/yq/v4@v4.44.3 '.jobs.test.strategy.matrix' .github/workflows/test.yml`
Expected: prints the matrix with `go-version: ["1.24"]`. Any parse error means the indentation is wrong — the added step must sit at the same level as the surrounding `- uses:` / `- name:` entries.

- [ ] **Step 3: Confirm the gate passes locally exactly as CI will run it**

Run: `make verify-generated && go test ./...`
Expected: both exit 0.

- [ ] **Step 4: Write the decision record**

Create `docs/superpowers/decisions/2026-07-29-codegen-spike-findings.md`. Fill every bracketed value from what actually happened in Tasks 1–4; do not copy expectations forward as results.

```markdown
# Codegen spike findings

Date: 2026-07-29
Stage: 1 of the api-go v3 transition
Spec: ../specs/2026-07-29-api-v3-transition-design.md

## Decision

Generate with `github.com/oapi-codegen/oapi-codegen/v2@v2.8.0` directly from the
OpenAPI 3.1 bundle. No downconversion to 3.0.

## What was tested

| Question | Answer | Evidence |
| --- | --- | --- |
| Does the 3.1 bundle generate without downconversion? | Yes | `make generate` exits 0 on a 6152-line 3.1 bundle |
| Does generated code compile? | Yes | `go build ./...` |
| Do `type: [T, "null"]` unions preserve null vs absent? | No | `TestNullAndAbsentAreIndistinguishable` |
| Can overlays carry Go-specific mappings? | [Yes / No — from Task 3] | `TestOverlayAppliedToGeneratedCode` |
| Are unknown fields rejected? | No, ignored by default | `TestUnknownFieldsIgnoredByDefault` |
| Do required properties avoid pointers? | Yes — required become values, optional become pointers | `Project.Id string` vs `Project.Token *string` |
| Does `nullable-type` give three-state decode? | [Yes / No effect — from Task 4] | `TestNullableTypeDistinguishesNullFromAbsent` |

## Consequences for stage 3

1. **Go floor rises to 1.24.** `github.com/oapi-codegen/runtime` v1.6.0 declares
   `go 1.24.0`. Consumer-visible; belongs in release notes.
2. **No three-state decoding.** `Nullable[T]` stays a request-side type. Response
   types that must distinguish null from absent need a different mechanism, and
   open question 3 in the design doc asks whether any actually do.
3. **[Count from Task 1] anonymous inline structs** appear in the generated
   models, from inline object schemas in the bundle. Verify with:
   `grep -c '\*struct {' internal/gen/gen.go`
   These are awkward to expose publicly — a consumer cannot name or easily
   construct them. Two ways out, and this is the main open design question
   handed to stage 3:
   - Ask the honeybadger repo to extract inline objects into named component
     schemas. Cheap, and it improves the published docs too.
   - Wrap them in hand-written facade types, accepting the conversion cost.

   `Error` is the case that matters most: both `Error.Error` and `Error.Meta` are
   anonymous structs, so design decision 7's `apiv3.Error` type must be
   hand-written and converted from the generated shape regardless of which route
   is chosen for the rest.
4. **Contract tests need `DisallowUnknownFields`** to catch renamed fields, since
   plain `json.Unmarshal` ignores them.

## What this does not settle

- Whether the generated client's request/response wrappers are usable directly
  or need a facade. Stage 3 decides, once prerequisites 1 and 2 are met.
- Whether the write-schema gaps (design prerequisite 2) can be worked around
  client-side. They cannot be, on current evidence: unspecified properties do
  not appear in generated request structs at all.
```

- [ ] **Step 5: Verify the inline-struct count you wrote is real**

Run: `grep -c '\*struct {' internal/gen/gen.go`
Expected: a number in the mid-hundreds. Put the actual figure in the record; do not carry over the plan's estimate.

- [ ] **Step 6: Commit**

```bash
git add .github/workflows/test.yml docs/superpowers/decisions/2026-07-29-codegen-spike-findings.md
git commit -m "ci: gate generated code against the vendored spec

Adds make verify-generated to CI and records the spike's findings,
including the go 1.24 floor and the inline-struct problem stage 3 inherits."
```

---

## Verification

Run all of these from the repo root. Every one must pass before the stage is done.

```bash
make generate            # exits 0, no error output
make verify-generated    # exits 0, no diff
go build ./...           # exits 0
go test ./...            # exits 0, includes the ~100 existing root-package tests
go vet ./...             # exits 0
```

Then confirm the deliverables named in the design doc exist:

```bash
test -f Makefile && test -f openapi/bundled.yaml && test -f openapi/overlay.yaml
test -f internal/gen/gen.go
test -f docs/superpowers/decisions/2026-07-29-codegen-spike-findings.md
```

## Out of scope

Named explicitly, because each is easy to drift into:

- No `apiv3/` package. Stage 3, blocked on design prerequisites 1 and 2.
- No changes to the root package's v2 services, types, or auth.
- No MCP server changes. That is stages 2 and 4, in a different repo.
- No renaming of root-package code to `apiv2`. Deferred in the design doc.
- No attempt to fix the write-schema gaps or missing routes. Those are
  server-side prerequisites, not client work.
