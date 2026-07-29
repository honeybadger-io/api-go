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

// Fields that are optional but NOT nullable in the spec still generate as
// plain pointers, and those conflate an explicit null with an absent key.
//
// Only properties declared `type: [T, "null"]` get nullable.Nullable[T] — see
// TestNullableTypeDistinguishesNullFromAbsent. Project.token is optional with a
// plain `type: string`, so it lands here. Stage 3 must not assume every
// optional field carries three states; it depends on how the spec declares it.
func TestOptionalNonNullableFieldsConflateNullAndAbsent(t *testing.T) {
	var explicitNull Project
	if err := json.Unmarshal([]byte(`{"token":null}`), &explicitNull); err != nil {
		t.Fatalf("decoding explicit null: %v", err)
	}

	var absent Project
	if err := json.Unmarshal([]byte(`{}`), &absent); err != nil {
		t.Fatalf("decoding absent field: %v", err)
	}

	if explicitNull.Token != nil {
		t.Errorf("explicit null gave non-nil %v; generator behavior changed", *explicitNull.Token)
	}
	if absent.Token != nil {
		t.Errorf("absent gave non-nil %v", *absent.Token)
	}
	// Both nil: indistinguishable for this class of field.
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
