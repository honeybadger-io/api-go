package honeybadgerapi

import "encoding/json"

// Nullable represents a JSON field value that is either an explicit null or a
// value of type T. Request params use *Nullable[T] with the omitempty tag so
// that a nil pointer omits the field entirely, giving three states: omitted
// (nil pointer), explicit null (Null), and a value (Value).
type Nullable[T any] struct {
	value  T
	isNull bool
}

// Value returns a Nullable holding v. In a request, the field marshals to v.
func Value[T any](v T) *Nullable[T] {
	return &Nullable[T]{value: v}
}

// Null returns a Nullable representing an explicit JSON null. In a request,
// the field marshals to null, which the API interprets as clearing the field.
func Null[T any]() *Nullable[T] {
	return &Nullable[T]{isNull: true}
}

// IsNull reports whether the Nullable represents an explicit JSON null.
// A nil receiver (an omitted field) returns false.
func (n *Nullable[T]) IsNull() bool {
	return n != nil && n.isNull
}

// Get returns the held value and true, or the zero value and false if the
// Nullable is nil (an omitted field) or represents null.
func (n *Nullable[T]) Get() (T, bool) {
	if n == nil || n.isNull {
		var zero T
		return zero, false
	}
	return n.value, true
}

// MarshalJSON implements json.Marshaler.
func (n *Nullable[T]) MarshalJSON() ([]byte, error) {
	if n.isNull {
		return []byte("null"), nil
	}
	return json.Marshal(n.value)
}

// UnmarshalJSON implements json.Unmarshaler.
func (n *Nullable[T]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*n = Nullable[T]{isNull: true}
		return nil
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*n = Nullable[T]{value: v}
	return nil
}
