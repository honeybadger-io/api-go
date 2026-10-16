package gen

import (
	"encoding/json"
	"testing"
)

// Does nullable-type generation distinguish explicit null from an absent key?
// This is the question the design doc's open question 3 asks.
//
// AccountInvitation.AcceptedAt generates as nullable.Nullable[time.Time], a
// value type (never *Nullable — the package explicitly warns against that).
func TestNullableTypeDistinguishesNullFromAbsent(t *testing.T) {
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

// A present value is the third state, and must be readable.
func TestNullableTypeReadsPresentValue(t *testing.T) {
	var withValue AccountInvitation
	if err := json.Unmarshal([]byte(`{"accepted_at":"2026-07-29T12:00:00Z"}`), &withValue); err != nil {
		t.Fatalf("decoding value: %v", err)
	}
	if !withValue.AcceptedAt.IsSpecified() {
		t.Fatal("present value: IsSpecified() = false, want true")
	}
	if withValue.AcceptedAt.IsNull() {
		t.Fatal("present value: IsNull() = true, want false")
	}
	got, err := withValue.AcceptedAt.Get()
	if err != nil {
		t.Fatalf("Get() returned an error for a present value: %v", err)
	}
	if got.Year() != 2026 || got.Month() != 7 || got.Day() != 29 {
		t.Errorf("Get() = %v, want 2026-07-29", got)
	}
}
