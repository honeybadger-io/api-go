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

// The code enum is now complete: insufficient_scope, credential_in_query, and
// project_restricted were once documented in responses without being declared,
// and are declared now. It has grown from 10 values to 20 during v3's
// development.
//
// apiv3 still treats Code as an open string rather than using these constants as
// a closed set — this test is why. The enum keeps moving, and a client that
// rejects unknown codes turns a real API error into an unrecognised one.
func TestSpecIssueErrorCodeEnumKeepsGrowing(t *testing.T) {
	// Sampled rather than exhaustive: the point is that codes keep being added,
	// so pinning the full list would make this a change-detector.
	for _, code := range []ErrorErrorCode{
		ErrorErrorCodeInsufficientScope,
		ErrorErrorCodeCredentialInQuery,
		ErrorErrorCodeProjectRestricted,
		ErrorErrorCodeUnsupportedAuthScheme,
		ErrorErrorCodeRequiresUserToken,
	} {
		if code == "" {
			t.Errorf("generated constant for %q is empty", code)
		}
	}

	// An unknown code still decodes, because the generated type is a string.
	// This is what lets apiv3 handle codes the enum has not caught up with.
	var e Error
	if err := json.Unmarshal([]byte(`{"error":{"code":"a_code_from_the_future","message":"m"}}`), &e); err != nil {
		t.Fatalf("unknown code should still decode as a string: %v", err)
	}
	if e.Error.Code != "a_code_from_the_future" {
		t.Errorf("Code = %q, want the raw value preserved", e.Error.Code)
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
