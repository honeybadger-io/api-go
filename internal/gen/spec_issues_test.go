package gen

import (
	"encoding/json"
	"strings"
	"testing"
)

// KNOWN SPEC ISSUE 1: Error.details is declared as an array of {field, message}
// for validation errors, but the InsufficientScope response's example uses an
// object: {required_scope, token_scopes}. Both cannot be true, and the generated
// type follows the schema, so a real 403 body fails to decode.
//
// This test asserts the CURRENT BROKEN behavior on purpose. When the spec is
// fixed it will fail, which is the signal that apiv3 can stop working around it.
// See openapi/README.md, "Known spec issues".
func TestSpecIssueErrorDetailsIsArrayButUsedAsObject(t *testing.T) {
	// Verbatim from the spec's own InsufficientScope example.
	body := `{"error":{"code":"insufficient_scope","message":"Insufficient scope",` +
		`"details":{"required_scope":"faults:write","token_scopes":["faults:read","insights:read"]}},` +
		`"meta":{"request_id":"abc123"}}`

	var e Error
	err := json.Unmarshal([]byte(body), &e)
	if err == nil {
		t.Fatal("the spec's InsufficientScope example now decodes; " +
			"the details type conflict looks fixed — drop apiv3's workaround and delete this test")
	}
	if !strings.Contains(err.Error(), "cannot unmarshal object into Go struct field") {
		t.Errorf("unexpected failure mode: %v", err)
	}

	// The array form, used by validation errors, does decode.
	arrayForm := `{"error":{"code":"validation_error","message":"Name can't be blank",` +
		`"details":[{"field":"name","message":"can't be blank"}]}}`
	var e2 Error
	if err := json.Unmarshal([]byte(arrayForm), &e2); err != nil {
		t.Fatalf("array form should decode: %v", err)
	}
	if e2.Error.Details == nil || len(*e2.Error.Details) != 1 {
		t.Fatal("array form decoded but details is empty")
	}
}

// KNOWN SPEC ISSUE 2: insufficient_scope, credential_in_query, and
// project_restricted appear in response descriptions and examples but are absent
// from the code enum, so no generated constant exists for them.
//
// apiv3 must therefore treat code as an open string, not a closed enum.
func TestSpecIssueNewAuthCodesMissingFromEnum(t *testing.T) {
	generated := map[ErrorErrorCode]bool{
		ErrorErrorCodeAccessDenied:        true,
		ErrorErrorCodeAmbiguousAccount:    true,
		ErrorErrorCodeForbiddenAttributes: true,
		ErrorErrorCodeInvalidId:           true,
		ErrorErrorCodeMaintenanceMode:     true,
		ErrorErrorCodeNotFound:            true,
		ErrorErrorCodeRateLimitExceeded:   true,
		ErrorErrorCodeServiceUnavailable:  true,
		ErrorErrorCodeUnauthorized:        true,
		ErrorErrorCodeValidationError:     true,
	}
	if len(generated) != 10 {
		t.Fatalf("expected 10 generated codes, got %d", len(generated))
	}

	for _, missing := range []string{"insufficient_scope", "credential_in_query", "project_restricted"} {
		if generated[ErrorErrorCode(missing)] {
			t.Errorf("%q is now in the enum; the spec gained it — update apiv3's sentinels", missing)
		}
	}

	// An unknown code still decodes, because the generated type is a string.
	// This is what lets apiv3 handle codes the enum has not caught up with.
	var e Error
	if err := json.Unmarshal([]byte(`{"error":{"code":"insufficient_scope","message":"Insufficient scope"}}`), &e); err != nil {
		t.Fatalf("unknown code should still decode as a string: %v", err)
	}
	if e.Error.Code != "insufficient_scope" {
		t.Errorf("Code = %q, want %q", e.Error.Code, "insufficient_scope")
	}
}
