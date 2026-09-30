package apiv3

import "testing"

// The generated map must actually be populated: an awk script that silently
// matched nothing would leave an empty map, and every scope check would then
// pass by accident.
func TestOperationScopesIsPopulated(t *testing.T) {
	if len(OperationScopes) < 100 {
		t.Fatalf("OperationScopes has %d entries; the generator matched almost nothing",
			len(OperationScopes))
	}
}

func TestOperationScopesKnownEntries(t *testing.T) {
	for op, want := range map[string]string{
		"listFaults":       "faults:read",
		"resolveFaults":    "faults:write",
		"runInsightsQuery": "insights:read",
		"listProjects":     "projects:read",
		"createProject":    "projects:create",
		"listCheckIns":     "checkins:read",
		"deleteCheckIn":    "checkins:write",
	} {
		if got := OperationScopes[op]; got != want {
			t.Errorf("OperationScopes[%q] = %q, want %q", op, got, want)
		}
	}
}

// Introspection must need no scope, or a credential could not discover its own
// limits without already holding something.
func TestGetTokenRequiresNoScope(t *testing.T) {
	if scope, ok := OperationScopes["getToken"]; ok {
		t.Errorf("getToken requires %q; it must stay reachable by any credential", scope)
	}
}

// Every scope should look like "resource:action". A malformed entry means the
// generator picked up the wrong line.
func TestOperationScopesAreWellFormed(t *testing.T) {
	for op, scope := range OperationScopes {
		if scope == "" {
			t.Errorf("%s has an empty scope", op)
			continue
		}
		colons := 0
		for _, c := range scope {
			if c == ':' {
				colons++
			}
		}
		if colons != 1 {
			t.Errorf("%s has scope %q, want one colon", op, scope)
		}
	}
}
