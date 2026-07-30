package apiv3

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestBulkFaultChangeSendsIDs(t *testing.T) {
	c, got := captureWrite(t, http.StatusOK, "")

	if err := c.Faults.Resolve(context.Background(), "Xk9mZp", SelectFaults("f1", "f2")); err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if want := "/v3/accounts/me/projects/Xk9mZp/faults/resolve"; got.path != want {
		t.Errorf("path = %q, want %q", got.path, want)
	}
	ids, ok := got.body["fault_ids"].([]any)
	if !ok || len(ids) != 2 || ids[0] != "f1" {
		t.Errorf("fault_ids = %v", got.body["fault_ids"])
	}
	if _, sent := got.body["q"]; sent {
		t.Errorf("q sent alongside fault_ids: %v", got.body)
	}
}

// The endpoint ignores q when fault_ids is present, so a selection carrying both
// must not reach the wire as both — it would read as a filter that silently did
// nothing.
func TestBulkFaultChangeByQueryOmitsIDs(t *testing.T) {
	c, got := captureWrite(t, http.StatusOK, "")

	if err := c.Faults.Ignore(context.Background(), "Xk9mZp", SelectFaultsMatching("is:unresolved")); err != nil {
		t.Fatalf("Ignore: %v", err)
	}

	if got.body["q"] != "is:unresolved" {
		t.Errorf("q = %v", got.body["q"])
	}
	if _, sent := got.body["fault_ids"]; sent {
		t.Errorf("fault_ids sent for a query selection: %v", got.body)
	}
}

// An empty selection is the whole project. The API accepts it — the body is
// optional now — so refusing it is the client's job.
func TestBulkFaultChangeRefusesEmptySelection(t *testing.T) {
	c, got := captureWrite(t, http.StatusOK, "")

	for _, tc := range []struct {
		name string
		call func() error
	}{
		{"resolve", func() error { return c.Faults.Resolve(context.Background(), "Xk9mZp", FaultSelection{}) }},
		{"unresolve", func() error { return c.Faults.Unresolve(context.Background(), "Xk9mZp", FaultSelection{}) }},
		{"ignore", func() error { return c.Faults.Ignore(context.Background(), "Xk9mZp", FaultSelection{}) }},
		{"unignore", func() error { return c.Faults.Unignore(context.Background(), "Xk9mZp", FaultSelection{}) }},
		{"blank query", func() error {
			return c.Faults.Resolve(context.Background(), "Xk9mZp", SelectFaultsMatching("  "))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got.method = ""
			if err := tc.call(); !errors.Is(err, ErrEveryFault) {
				t.Fatalf("err = %v, want ErrEveryFault", err)
			}
			if got.method != "" {
				t.Errorf("request reached the server: %s", got.method)
			}
		})
	}
}

// The path fault is the source and is destroyed; the body names the keeper. Sent
// the wrong way round, a merge deletes the fault the caller meant to keep.
// The whole project is a real intent, but it has to be stated. This must send an
// empty body rather than a wildcard query: the search runs against notices, so
// "*" would miss a fault whose notices are not searchable.
func TestSelectAllFaultsSendsNoFilter(t *testing.T) {
	c, got := captureWrite(t, http.StatusOK, "")

	if err := c.Faults.Unignore(context.Background(), "Xk9mZp", SelectAllFaults()); err != nil {
		t.Fatalf("Unignore: %v", err)
	}
	if len(got.body) != 0 {
		t.Errorf("body = %v, want no filter", got.body)
	}
}

func TestMergeMergesThePathFaultIntoTheBodyTarget(t *testing.T) {
	c, got := captureWrite(t, http.StatusAccepted, `{"data":{
		"batch_id":"WksB67FpRY3bZQ","source_id":"2z4WtH9gARww","target_id":"OmDSjxQSys5k"}}`)

	merge, err := c.Faults.Merge(context.Background(), "Xk9mZp", "2z4WtH9gARww", "OmDSjxQSys5k")
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	if want := "/v3/accounts/me/projects/Xk9mZp/faults/2z4WtH9gARww/merge"; got.path != want {
		t.Errorf("path = %q, want the source fault %q", got.path, want)
	}
	if got.body["target_fault_id"] != "OmDSjxQSys5k" {
		t.Errorf("target_fault_id = %v, want the fault being kept", got.body["target_fault_id"])
	}
	if merge.BatchID != "WksB67FpRY3bZQ" || merge.SourceID != "2z4WtH9gARww" || merge.TargetID != "OmDSjxQSys5k" {
		t.Errorf("merge = %+v", merge)
	}
}

func TestMergeRefusesAFaultIntoItself(t *testing.T) {
	c, got := captureWrite(t, http.StatusAccepted, "")

	if _, err := c.Faults.Merge(context.Background(), "Xk9mZp", "f1", "f1"); !errors.Is(err, ErrMergeIntoSelf) {
		t.Fatalf("err = %v, want ErrMergeIntoSelf", err)
	}
	if got.method != "" {
		t.Errorf("request reached the server: %s", got.method)
	}
}

// Time filters bound a bulk change without naming ids. They apply only when
// fault_ids is omitted, so they compose with a query and not with a list.
func TestBulkFaultChangeSendsTimeFilters(t *testing.T) {
	c, got := captureWrite(t, http.StatusOK, "")
	cutoff := time.Unix(1785300000, 0)

	sel := SelectFaultsMatching("is:unresolved").OccurredBefore(cutoff)
	if err := c.Faults.Ignore(context.Background(), "Xk9mZp", sel); err != nil {
		t.Fatalf("Ignore: %v", err)
	}
	if got.body["q"] != "is:unresolved" {
		t.Errorf("q = %v", got.body["q"])
	}
	if got.body["occurred_before"] != float64(1785300000) {
		t.Errorf("occurred_before = %v", got.body["occurred_before"])
	}
}

// A time filter is itself a bound, so it is a complete selection on its own.
func TestTimeFilterAloneIsABoundedSelection(t *testing.T) {
	c, got := captureWrite(t, http.StatusOK, "")

	sel := FaultSelection{}.CreatedAfter(time.Unix(1785300000, 0))
	if err := c.Faults.Resolve(context.Background(), "Xk9mZp", sel); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.body["created_after"] != float64(1785300000) {
		t.Errorf("created_after = %v", got.body["created_after"])
	}
	if _, sent := got.body["fault_ids"]; sent {
		t.Errorf("fault_ids sent: %v", got.body)
	}
}

// Naming ids and also filtering is contradictory: the endpoint ignores the
// filters when ids are present, so the request would not mean what it reads as.
func TestIDsWithTimeFiltersIsRefused(t *testing.T) {
	c, got := captureWrite(t, http.StatusOK, "")

	sel := SelectFaults("f1").OccurredBefore(time.Unix(1785300000, 0))
	if err := c.Faults.Resolve(context.Background(), "Xk9mZp", sel); !errors.Is(err, ErrFilteredIDs) {
		t.Fatalf("err = %v, want ErrFilteredIDs", err)
	}
	if got.method != "" {
		t.Errorf("request reached the server: %s", got.method)
	}
}
