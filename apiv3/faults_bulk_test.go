package apiv3

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestBulkFaultChangeSendsIDs(t *testing.T) {
	c, got := captureWrite(t, http.StatusOK, bulkResult)

	if _, err := c.Faults.Resolve(context.Background(), "Xk9mZp", SelectFaults("1", "2")); err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if want := "/v3/projects/Xk9mZp/faults/resolve"; got.path != want {
		t.Errorf("path = %q, want %q", got.path, want)
	}
	ids, ok := got.body["fault_ids"].([]any)
	if !ok || len(ids) != 2 || ids[0] != "1" {
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
	c, got := captureWrite(t, http.StatusOK, bulkResult)

	if _, err := c.Faults.Ignore(context.Background(), "Xk9mZp", SelectFaultsMatching("is:unresolved")); err != nil {
		t.Fatalf("Ignore: %v", err)
	}

	if got.body["q"] != "is:unresolved" {
		t.Errorf("q = %v", got.body["q"])
	}
	if _, sent := got.body["fault_ids"]; sent {
		t.Errorf("fault_ids sent for a query selection: %v", got.body)
	}
}

// A bulk change on a fault from another project succeeds with a count of zero,
// so the count has to reach the caller or the no-op reads as a success.
func TestBulkFaultChangeReturnsTheCount(t *testing.T) {
	c, _ := captureWrite(t, http.StatusOK,
		`{"data":{"count":0,"dry_run":false,"fault_ids":[],"fault_ids_truncated":false}}`)

	result, err := c.Faults.Resolve(context.Background(), "Xk9mZp", SelectFaults("99"))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if result.Count != 0 || result.DryRun {
		t.Errorf("result = %+v, want count 0, not a dry run", result)
	}
}

// An empty selection is refused before it is sent: the endpoint would reject it
// anyway without all=true, and the client can say why.
func TestBulkFaultChangeRefusesEmptySelection(t *testing.T) {
	c, got := captureWrite(t, http.StatusOK, bulkResult)
	call := func(f func(context.Context, string, FaultSelection) (*FaultBulkResult, error), sel FaultSelection) func() error {
		return func() error { _, err := f(context.Background(), "Xk9mZp", sel); return err }
	}

	for _, tc := range []struct {
		name string
		call func() error
	}{
		{"resolve", call(c.Faults.Resolve, FaultSelection{})},
		{"unresolve", call(c.Faults.Unresolve, FaultSelection{})},
		{"ignore", call(c.Faults.Ignore, FaultSelection{})},
		{"unignore", call(c.Faults.Unignore, FaultSelection{})},
		{"blank query", func() error {
			_, err := c.Faults.Resolve(context.Background(), "Xk9mZp", SelectFaultsMatching("  "))
			return err
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

// The whole project is a real intent, but it has to be stated: the endpoint
// refuses an unfiltered change without all=true. It must not become a wildcard
// query either — the search runs against notices, so "*" would miss a fault whose
// notices are not searchable.
func TestSelectAllFaultsSendsAll(t *testing.T) {
	c, got := captureWrite(t, http.StatusOK, bulkResult)

	if _, err := c.Faults.Unignore(context.Background(), "Xk9mZp", SelectAllFaults()); err != nil {
		t.Fatalf("Unignore: %v", err)
	}
	if got.body["all"] != true {
		t.Errorf("all = %v, want true", got.body["all"])
	}
	for _, absent := range []string{"q", "fault_ids"} {
		if _, sent := got.body[absent]; sent {
			t.Errorf("%s sent for the whole project: %v", absent, got.body)
		}
	}
}

// A dry run changes nothing server-side, so it must reach the wire, including
// alongside named ids.
func TestDryRunIsSent(t *testing.T) {
	c, got := captureWrite(t, http.StatusOK, bulkResult)

	if _, err := c.Faults.Resolve(context.Background(), "Xk9mZp", SelectFaults("1").DryRun()); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.body["dry_run"] != true {
		t.Errorf("dry_run = %v, want true", got.body["dry_run"])
	}
}

// The listing filters send fractional seconds, so the bulk filters must too, or
// the same cutoff selects different faults in each.
func TestBulkTimeFiltersKeepFractionalSeconds(t *testing.T) {
	c, got := captureWrite(t, http.StatusOK, bulkResult)

	sel := SelectAllFaults().OccurredBefore(time.Unix(1785300000, 500_000_000))
	if _, err := c.Faults.Resolve(context.Background(), "Xk9mZp", sel); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.body["occurred_before"] != 1785300000.5 {
		t.Errorf("occurred_before = %v, want 1785300000.5", got.body["occurred_before"])
	}
}

// The path fault is the source and is destroyed; the body names the keeper. Sent
// the wrong way round, a merge deletes the fault the caller meant to keep.
func TestMergeMergesThePathFaultIntoTheBodyTarget(t *testing.T) {
	c, got := captureWrite(t, http.StatusAccepted, `{"data":{
		"batch_id":"WksB67FpRY3bZQ","source_id":"201","target_id":"202"}}`)

	merge, err := c.Faults.Merge(context.Background(), "Xk9mZp", "201", "202")
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	if want := "/v3/projects/Xk9mZp/faults/201/merge"; got.path != want {
		t.Errorf("path = %q, want the source fault %q", got.path, want)
	}
	if got.body["target_fault_id"] != "202" {
		t.Errorf("target_fault_id = %v, want the fault being kept", got.body["target_fault_id"])
	}
	if merge.BatchId != "WksB67FpRY3bZQ" || merge.SourceId != "201" || merge.TargetId != "202" {
		t.Errorf("merge = %+v", merge)
	}
}

func TestMergeRefusesAFaultIntoItself(t *testing.T) {
	c, got := captureWrite(t, http.StatusAccepted, "")

	if _, err := c.Faults.Merge(context.Background(), "Xk9mZp", "1", "1"); !errors.Is(err, ErrMergeIntoSelf) {
		t.Fatalf("err = %v, want ErrMergeIntoSelf", err)
	}
	if got.method != "" {
		t.Errorf("request reached the server: %s", got.method)
	}
}

// Time filters bound a bulk change without naming ids. They apply only when
// fault_ids is omitted, so they compose with a query and not with a list.
func TestBulkFaultChangeSendsTimeFilters(t *testing.T) {
	c, got := captureWrite(t, http.StatusOK, bulkResult)
	cutoff := time.Unix(1785300000, 0)

	sel := SelectFaultsMatching("is:unresolved").OccurredBefore(cutoff)
	if _, err := c.Faults.Ignore(context.Background(), "Xk9mZp", sel); err != nil {
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
	c, got := captureWrite(t, http.StatusOK, bulkResult)

	sel := FaultSelection{}.CreatedAfter(time.Unix(1785300000, 0))
	if _, err := c.Faults.Resolve(context.Background(), "Xk9mZp", sel); err != nil {
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
	c, got := captureWrite(t, http.StatusOK, bulkResult)

	sel := SelectFaults("1").OccurredBefore(time.Unix(1785300000, 0))
	if _, err := c.Faults.Resolve(context.Background(), "Xk9mZp", sel); !errors.Is(err, ErrFilteredIDs) {
		t.Fatalf("err = %v, want ErrFilteredIDs", err)
	}
	if got.method != "" {
		t.Errorf("request reached the server: %s", got.method)
	}
}

// bulkResult is a typical bulk-change response: the endpoints answer 200 with a
// count of what changed, not 204.
const bulkResult = `{"data":{"count":1,"dry_run":false,"fault_ids":["1"],"fault_ids_truncated":false}}`

// A zero time is no filter, so it neither slips past the every-fault guard nor
// counts as a filter alongside ids. Converted naively it's a real bound in 1754
// that matches every fault.
func TestBulkZeroTimeIsNoFilter(t *testing.T) {
	c, got := captureWrite(t, http.StatusOK, bulkResult)

	if _, err := c.Faults.Resolve(context.Background(), "Xk9mZp",
		FaultSelection{}.CreatedAfter(time.Time{})); !errors.Is(err, ErrEveryFault) {
		t.Errorf("zero CreatedAfter alone: err = %v, want ErrEveryFault", err)
	}
	if got.method != "" {
		t.Errorf("a request was sent: %s %s", got.method, got.path)
	}

	if _, err := c.Faults.Resolve(context.Background(), "Xk9mZp",
		SelectFaults("1").OccurredBefore(time.Time{})); err != nil {
		t.Errorf("ids with a zero OccurredBefore: err = %v, want the ids sent unfiltered", err)
	}
	if _, sent := got.body["occurred_before"]; sent {
		t.Errorf("occurred_before = %v, want it absent", got.body["occurred_before"])
	}
}
