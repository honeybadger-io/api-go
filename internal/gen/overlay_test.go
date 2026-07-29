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
