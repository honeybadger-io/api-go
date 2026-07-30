package apiv3

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFaultsListSendsSearchQuery(t *testing.T) {
	var gotQuery, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("q")
		gotPath = r.URL.Path
		writeJSON(w, 0, `{"data":[],"pagination":{"page":1,"per_page":25,"total_count":0,"total_pages":0}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	_, err := c.Faults.List(context.Background(), "Xk9mZp", FaultListOptions{
		Query: "environment:production is:resolved",
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if want := "environment:production is:resolved"; gotQuery != want {
		t.Errorf("q = %q, want %q", gotQuery, want)
	}
	if want := "/v3/accounts/me/projects/Xk9mZp/faults"; gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
}

func TestFaultsListDecodesFaults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 0, `{"data":[
		  {"id":"f1","project_id":"Xk9mZp","klass":"RuntimeError","message":"boom","notices_count":42}
		],"pagination":{"page":1,"per_page":25,"total_count":1,"total_pages":1},
		"meta":{"request_id":"req_faults"}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	resp, err := c.Faults.List(context.Background(), "Xk9mZp", FaultListOptions{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("Data = %v, want 1 fault", resp.Data)
	}
	if resp.RequestID != "req_faults" {
		t.Errorf("RequestID = %q", resp.RequestID)
	}
}

func TestFaultsGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if want := "/v3/accounts/me/projects/Xk9mZp/faults/f1"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		writeJSON(w, 0, `{"data":{"id":"f1","project_id":"Xk9mZp","klass":"RuntimeError"}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	f, err := c.Faults.Get(context.Background(), "Xk9mZp", "f1", FaultGetOptions{})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if f.Id != "f1" {
		t.Errorf("fault id = %q, want f1", f.Id)
	}
}

// A fault's nullable fields must survive as three-state, since "assignee was
// cleared" and "assignee was not returned" are different facts.
func TestFaultNullableFieldsAreThreeState(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 0, `{"data":{"id":"f1","action":null}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	f, err := c.Faults.Get(context.Background(), "Xk9mZp", "f1", FaultGetOptions{})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if !f.Action.IsSpecified() {
		t.Error("action: IsSpecified() = false, want true for an explicit null")
	}
	if !f.Action.IsNull() {
		t.Error("action: IsNull() = false, want true")
	}
	// A nullable field the server omitted entirely reads as unspecified — the
	// third state, distinct from the explicit null above.
	if f.Assignee.IsSpecified() {
		t.Error("assignee was omitted but reads as specified")
	}

	// Environment is optional but NOT declared nullable in the spec, so it
	// generates as a plain pointer and cannot distinguish null from absent.
	// Documented by TestOptionalNonNullableFieldsConflateNullAndAbsent.
	if f.Environment != nil {
		t.Errorf("environment = %v, want nil", *f.Environment)
	}
}

// Notices are the first cursor-paginated endpoint, so this exercises
// CursorPagination against real generated code rather than a fake.
//
// Note the fixture ids are real UUIDs: unlike every other v3 resource, which
// uses short opaque public ids, a notice is addressed by its token UUID
// (`format: uuid` in the spec), so the generated type is uuid.UUID and rejects
// anything else.
func TestFaultsListNoticesUsesCursorPagination(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if want := "/v3/accounts/me/projects/Xk9mZp/faults/f1/notices"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		if got := r.URL.Query().Get("limit"); got != "10" {
			t.Errorf("limit = %q, want 10", got)
		}
		writeJSON(w, 0, `{"data":[{"id":"11111111-1111-4111-8111-111111111111"}],
		  "pagination":{"has_newer":false,"has_older":true,"limit":10,
		    "newest_cursor":"cur_new","oldest_cursor":"cur_old"}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	resp, err := c.Faults.ListNotices(context.Background(), "Xk9mZp", "f1", NoticeListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("ListNotices: %v", err)
	}

	if resp.Pagination != nil {
		t.Error("Pagination should be nil for a cursor endpoint")
	}
	if resp.Cursor == nil {
		t.Fatal("Cursor is nil for a cursor endpoint")
	}
	if !resp.Cursor.HasOlder {
		t.Error("HasOlder = false, want true")
	}
	oldest, err := resp.Cursor.OldestCursor.Get()
	if err != nil || oldest != "cur_old" {
		t.Errorf("OldestCursor = %q (err %v), want cur_old", oldest, err)
	}
}

func TestFaultsListAllNoticesFollowsCursor(t *testing.T) {
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		before := r.URL.Query().Get("before")
		seen = append(seen, before)
		switch before {
		case "":
			writeJSON(w, 0, `{"data":[{"id":"11111111-1111-4111-8111-111111111111"}],
			  "pagination":{"has_older":true,"limit":1,"oldest_cursor":"cur_1"}}`)
		case "cur_1":
			writeJSON(w, 0, `{"data":[{"id":"22222222-2222-4222-8222-222222222222"}],
			  "pagination":{"has_older":false,"limit":1,"oldest_cursor":null}}`)
		default:
			t.Errorf("unexpected before=%q", before)
		}
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	all, err := c.Faults.ListAllNotices(context.Background(), "Xk9mZp", "f1", NoticeListOptions{Limit: 1})
	if err != nil {
		t.Fatalf("ListAllNotices: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("got %d notices, want 2", len(all))
	}
	if want := []string{"", "cur_1"}; !equal(seen, want) {
		t.Errorf("cursors %v, want %v", seen, want)
	}
}

// ListAllNotices walks backwards from newest, so a caller's After is
// meaningless and must not leak into the requests.
func TestListAllNoticesIgnoresAfter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("after"); got != "" {
			t.Errorf("after = %q, want it dropped", got)
		}
		writeJSON(w, 0, `{"data":[{"id":"11111111-1111-4111-8111-111111111111"}],"pagination":{"has_older":false,"limit":1}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	if _, err := c.Faults.ListAllNotices(context.Background(), "Xk9mZp", "f1",
		NoticeListOptions{After: "cur_newer"}); err != nil {
		t.Fatalf("ListAllNotices: %v", err)
	}
}

func TestFaultsListInsufficientScope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusForbidden, `{"error":{"code":"insufficient_scope","message":"Insufficient scope",
		  "details":{"required_scope":"faults:read","token_scopes":["projects:read"]}}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	_, err := c.Faults.List(context.Background(), "Xk9mZp", FaultListOptions{})
	if !errors.Is(err, ErrInsufficientScope) {
		t.Fatalf("err = %v, want ErrInsufficientScope", err)
	}

	var apiErr *Error
	if !asError(err, &apiErr) {
		t.Fatal("not an *apiv3.Error")
	}
	if apiErr.RequiredScope() != "faults:read" {
		t.Errorf("RequiredScope() = %q", apiErr.RequiredScope())
	}
}
