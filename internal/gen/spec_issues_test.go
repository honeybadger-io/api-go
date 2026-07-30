package gen

import (
	"encoding/json"
	"testing"
)

// KNOWN SPEC ISSUE 1: Error.details is declared as an array of {field, message}
// for validation errors, but the InsufficientScope response's example uses an
// object: {required_scope, token_scopes}. Both cannot be true.
//
// openapi/overlay.yaml retypes the field as json.RawMessage so generated code
// accepts either shape and apiv3 decides which it received. These tests assert
// that workaround holds — if the spec settles on one shape, drop the overlay
// action and these tests with it.
func TestSpecIssueErrorDetailsAcceptsBothShapes(t *testing.T) {
	objectForm := `{"error":{"code":"insufficient_scope","message":"Insufficient scope",` +
		`"details":{"required_scope":"faults:write","token_scopes":["faults:read","insights:read"]}},` +
		`"meta":{"request_id":"abc123"}}`

	var e Error
	if err := json.Unmarshal([]byte(objectForm), &e); err != nil {
		t.Fatalf("object form should decode via the RawMessage overlay: %v", err)
	}
	if e.Error.Details == nil {
		t.Fatal("details is nil for the object form")
	}
	var asObject map[string]any
	if err := json.Unmarshal(*e.Error.Details, &asObject); err != nil {
		t.Fatalf("details did not hold an object: %v", err)
	}
	if asObject["required_scope"] != "faults:write" {
		t.Errorf("required_scope = %v", asObject["required_scope"])
	}

	arrayForm := `{"error":{"code":"validation_error","message":"Name can't be blank",` +
		`"details":[{"field":"name","message":"can't be blank"}]}}`

	var e2 Error
	if err := json.Unmarshal([]byte(arrayForm), &e2); err != nil {
		t.Fatalf("array form should decode: %v", err)
	}
	var asArray []struct {
		Field   string `json:"field"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(*e2.Error.Details, &asArray); err != nil {
		t.Fatalf("details did not hold an array: %v", err)
	}
	if len(asArray) != 1 || asArray[0].Field != "name" {
		t.Errorf("array form = %+v", asArray)
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

// The overlay mechanism itself is load-bearing: it is how Go-specific concerns
// stay out of the producer spec. This asserts it is still wired up, since a
// silently-ignored overlay would reintroduce the decode failure above.
func TestOverlayIsApplied(t *testing.T) {
	var e Error
	if err := json.Unmarshal([]byte(`{"error":{"code":"not_found","message":"m","details":{"any":"object"}}}`), &e); err != nil {
		t.Fatal("overlay is not applied: details rejected an object, " +
			"so it is still typed as an array — check openapi/codegen.yaml's output-options.overlay")
	}
}
