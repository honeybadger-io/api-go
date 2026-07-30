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
	_, err := c.Faults.List(context.Background(), "Xk9mZp", Search("environment:production is:resolved"))
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
	resp, err := c.Faults.List(context.Background(), "Xk9mZp")
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
	f, err := c.Faults.Get(context.Background(), "Xk9mZp", "f1")
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
	f, err := c.Faults.Get(context.Background(), "Xk9mZp", "f1")
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

// Notices are the first time-ordered endpoint, so this exercises
// TimeSeriesPagination against real generated code rather than a fake.
//
// Note the fixture ids are real UUIDs: unlike every other v3 resource, which
// uses short opaque public ids, a notice is addressed by its token UUID
// (`format: uuid` in the spec), so the generated type is uuid.UUID and rejects
// anything else.
func TestFaultsListNoticesUsesTimeSeriesPagination(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if want := "/v3/accounts/me/projects/Xk9mZp/faults/f1/notices"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		if got := r.URL.Query().Get("limit"); got != "10" {
			t.Errorf("limit = %q, want 10", got)
		}
		writeJSON(w, 0, `{"data":[{"id":"11111111-1111-4111-8111-111111111111"}],
		  "pagination":{"has_newer":false,"has_older":true,"limit":10,
		    "newest_cursor":"cur_new","oldest_cursor":"cur_old"},
		  "links":{"self":"http://`+r.Host+`/v3/self","older":"http://`+r.Host+`/v3/older"}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	resp, err := c.Faults.ListNotices(context.Background(), "Xk9mZp", "f1", Limit(10))
	if err != nil {
		t.Fatalf("ListNotices: %v", err)
	}

	if resp.Pagination != nil {
		t.Error("Pagination should be nil for a time-ordered endpoint")
	}
	if resp.TimeSeries == nil {
		t.Fatal("TimeSeries is nil for a time-ordered endpoint")
	}
	if !resp.TimeSeries.HasOlder {
		t.Error("HasOlder = false, want true")
	}
	if resp.TimeSeriesLinks == nil {
		t.Fatal("TimeSeriesLinks is nil; the walk needs links to follow")
	}
	oldest, err := resp.TimeSeries.OldestCursor.Get()
	if err != nil || oldest != "cur_old" {
		t.Errorf("OldestCursor = %q (err %v), want cur_old", oldest, err)
	}
}

func TestFaultsListAllNoticesFollowsOlderLinks(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/v3/accounts/me/projects/Xk9mZp/faults/f1/notices":
			writeJSON(w, 0, `{"data":[{"id":"11111111-1111-4111-8111-111111111111"}],
			  "pagination":{"has_older":true,"limit":1,"oldest_cursor":"cur_1"},
			  "links":{"self":"http://`+r.Host+`/v3/self","older":"http://`+r.Host+`/v3/notices/older"}}`)
		case "/v3/notices/older":
			writeJSON(w, 0, `{"data":[{"id":"22222222-2222-4222-8222-222222222222"}],
			  "pagination":{"has_older":false,"limit":1,"oldest_cursor":null},
			  "links":{"self":"http://`+r.Host+`/v3/notices/older"}}`)
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
		}
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	all, err := c.Faults.ListAllNotices(context.Background(), "Xk9mZp", "f1", Limit(1))
	if err != nil {
		t.Fatalf("ListAllNotices: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("got %d notices, want 2", len(all))
	}
	if want := []string{"/v3/accounts/me/projects/Xk9mZp/faults/f1/notices", "/v3/notices/older"}; !equal(paths, want) {
		t.Errorf("paths %v, want %v", paths, want)
	}
}

// A followed link carries the credential, so a link pointing at another host
// must be refused rather than followed.
func TestListAllNoticesRefusesOffHostLink(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 0, `{"data":[{"id":"11111111-1111-4111-8111-111111111111"}],
		  "pagination":{"has_older":true,"limit":1},
		  "links":{"self":"http://`+r.Host+`/v3/self","older":"https://evil.example.com/v3/steal"}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_secret")
	_, err := c.Faults.ListAllNotices(context.Background(), "Xk9mZp", "f1")
	if !errors.Is(err, ErrUntrustedLink) {
		t.Fatalf("err = %v, want ErrUntrustedLink", err)
	}
}

// ListAllNotices walks backwards from newest, so no cursor option can be
// passed to it — After is not a ListAllOption, and the compiler enforces that.
// This asserts the walk never sends one.
func TestListAllNoticesSendsNoAfterCursor(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("after"); got != "" {
			t.Errorf("after = %q, want it dropped", got)
		}
		writeJSON(w, 0, `{"data":[{"id":"11111111-1111-4111-8111-111111111111"}],
		  "pagination":{"has_older":false,"limit":1}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	if _, err := c.Faults.ListAllNotices(context.Background(), "Xk9mZp", "f1"); err != nil {
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
	_, err := c.Faults.List(context.Background(), "Xk9mZp")
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
