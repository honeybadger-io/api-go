package honeybadgerapi

import (
	"encoding/json"
	"testing"
)

func TestNullableMarshal(t *testing.T) {
	type params struct {
		ID   *Nullable[int]    `json:"id,omitempty"`
		Name *Nullable[string] `json:"name,omitempty"`
	}

	tests := []struct {
		name     string
		params   params
		expected string
	}{
		{"omitted", params{}, `{}`},
		{"value", params{ID: Value(42)}, `{"id":42}`},
		{"null", params{ID: Null[int]()}, `{"id":null}`},
		{"string value", params{Name: Value("test")}, `{"name":"test"}`},
		{"zero value is not omitted", params{ID: Value(0)}, `{"id":0}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.params)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if string(data) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(data))
			}
		})
	}
}

func TestNullableUnmarshal(t *testing.T) {
	var n Nullable[int]

	if err := json.Unmarshal([]byte("null"), &n); err != nil {
		t.Fatalf("Unmarshal(null) error = %v", err)
	}
	if !n.IsNull() {
		t.Error("expected IsNull() to be true after unmarshaling null")
	}
	if _, ok := n.Get(); ok {
		t.Error("expected Get() ok to be false for null")
	}

	if err := json.Unmarshal([]byte("42"), &n); err != nil {
		t.Fatalf("Unmarshal(42) error = %v", err)
	}
	if n.IsNull() {
		t.Error("expected IsNull() to be false after unmarshaling a value")
	}
	if v, ok := n.Get(); !ok || v != 42 {
		t.Errorf("expected Get() = (42, true), got (%d, %t)", v, ok)
	}

	if err := json.Unmarshal([]byte(`"nope"`), &n); err == nil {
		t.Error("expected type error unmarshaling string into Nullable[int]")
	}
}

func TestNullableAccessors(t *testing.T) {
	v := Value(7)
	if v.IsNull() {
		t.Error("Value(7).IsNull() should be false")
	}
	if got, ok := v.Get(); !ok || got != 7 {
		t.Errorf("Value(7).Get() = (%d, %t), want (7, true)", got, ok)
	}

	n := Null[int]()
	if !n.IsNull() {
		t.Error("Null[int]().IsNull() should be true")
	}
	if got, ok := n.Get(); ok || got != 0 {
		t.Errorf("Null[int]().Get() = (%d, %t), want (0, false)", got, ok)
	}
}
