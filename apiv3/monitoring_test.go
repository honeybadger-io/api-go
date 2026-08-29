package apiv3

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckInsList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if want := "/v3/projects/Xk9mZp/check_ins"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		writeJSON(w, 0, `{"data":[{"id":"c1","name":"Nightly","slug":"nightly"}],
		  "pagination":{"page":1,"per_page":25,"total_count":1,"total_pages":1}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	resp, err := c.CheckIns.List(context.Background(), "Xk9mZp")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("Data = %v, want 1 check-in", resp.Data)
	}
}

// created_before is not exposed, so no request may carry it: paging by a
// float32 epoch would silently skip or repeat events.
func TestCheckInEventsNeverSendCreatedBefore(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("created_before"); got != "" {
			t.Errorf("created_before = %q, want it never sent", got)
		}
		if got := r.URL.Query().Get("limit"); got != "5" {
			t.Errorf("limit = %q, want 5", got)
		}
		writeJSON(w, 0, `{"data":[{"id":"e1"}],"pagination":{"has_older":false,"limit":5}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	if _, err := c.CheckIns.ListAllEvents(context.Background(), "Xk9mZp", "c1", Limit(5)); err != nil {
		t.Fatalf("ListAllEvents: %v", err)
	}
}

func TestCheckInEventsWalkFollowsLinks(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.URL.Path == "/v3/older-events" {
			writeJSON(w, 0, `{"data":[{"id":"e2"}],"pagination":{"has_older":false,"limit":1},
			  "links":{"self":"http://`+r.Host+`/v3/older-events"}}`)
			return
		}
		writeJSON(w, 0, `{"data":[{"id":"e1"}],"pagination":{"has_older":true,"limit":1},
		  "links":{"self":"http://`+r.Host+`/v3/self","older":"http://`+r.Host+`/v3/older-events"}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	all, err := c.CheckIns.ListAllEvents(context.Background(), "Xk9mZp", "c1")
	if err != nil {
		t.Fatalf("ListAllEvents: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("got %d events, want 2", len(all))
	}
	if len(paths) != 2 || paths[1] != "/v3/older-events" {
		t.Errorf("paths = %v", paths)
	}
}

// Alarms are unpaginated, so one call is the whole collection and there is no
// ListAll to walk.
func TestAlarmsListIsUnpaginated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("query = %q, want none for an unpaginated endpoint", r.URL.RawQuery)
		}
		writeJSON(w, 0, `{"data":[{"id":"a1","name":"Error spike"},{"id":"a2","name":"Latency"}]}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	resp, err := c.Alarms.List(context.Background(), "Xk9mZp")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("Data = %v, want 2 alarms", resp.Data)
	}
	if resp.Pagination != nil {
		t.Errorf("Pagination = %+v, want nil", resp.Pagination)
	}
}

// Alarm history rows come straight from the query service, so they stay untyped.
func TestAlarmsListHistoryPassesRowsThrough(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if want := "/v3/projects/Xk9mZp/alarms/a1/history"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		writeJSON(w, 0, `{"data":[{"state":"triggered","at":"2026-07-29T00:00:00Z","value":91.5}],
		  "pagination":{"page":1,"total_pages":1}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	rows, err := c.Alarms.ListHistory(context.Background(), "Xk9mZp", "a1")
	if err != nil {
		t.Fatalf("ListHistory: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %v, want 1", rows)
	}
	if rows[0]["state"] != "triggered" || rows[0]["value"] != 91.5 {
		t.Errorf("row = %v", rows[0])
	}
}

// Alarm history takes page but no per_page, since its paging object is the query
// service's rather than v3's.
func TestAlarmHistorySendsPageOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("page"); got != "3" {
			t.Errorf("page = %q, want 3", got)
		}
		if _, present := r.URL.Query()["per_page"]; present {
			t.Error("per_page was sent; this endpoint does not accept it")
		}
		writeJSON(w, 0, `{"data":[],"pagination":{"page":3,"total_pages":3}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	if _, err := c.Alarms.ListHistory(context.Background(), "Xk9mZp", "a1", Page(3, 50)); err != nil {
		t.Fatalf("ListHistory: %v", err)
	}
}

func TestDashboardsGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if want := "/v3/projects/Xk9mZp/dashboards/d1"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		writeJSON(w, 0, `{"data":{"id":"d1","name":"Ops"}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	d, err := c.Dashboards.Get(context.Background(), "Xk9mZp", "d1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if d.Id != "d1" {
		t.Errorf("dashboard id = %q, want d1", d.Id)
	}
}

func TestFaultsAffectedUsers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if want := "/v3/projects/Xk9mZp/faults/1/affected_users"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		writeJSON(w, 0, `{"data":[{"user":"a@example.com","count":3}]}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	data, err := c.Faults.AffectedUsers(context.Background(), "Xk9mZp", 1)
	if err != nil {
		t.Fatalf("AffectedUsers: %v", err)
	}
	if len(data) != 1 {
		t.Fatalf("data = %+v, want one affected user", data)
	}
	if data[0].Count != 3 || data[0].User != "a@example.com" {
		t.Errorf("data[0] = %+v", data[0])
	}
}
