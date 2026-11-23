package apiv3

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const rateLimitedWarning = `{"code":"error_rate_limited","message":"Errors over the limit are dropped.","start_at":"2026-11-18T14:12:00Z","end_at":"2026-11-18T15:12:00Z","limit_per_minute":1}`

// A fault listing covering a period when errors were dropped says so in
// meta.warnings. Losing it would let a caller read the gap as the app going
// quiet.
func TestFaultListCarriesWarnings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 0, `{"data":[],"pagination":{"page":1,"per_page":25},"meta":{"request_id":"req_1","warnings":[`+rateLimitedWarning+`]}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbp_x")
	resp, err := c.Faults.List(context.Background(), "Xk9mZp")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(resp.Warnings) != 1 {
		t.Fatalf("Warnings = %+v, want one", resp.Warnings)
	}
	got := resp.Warnings[0]
	if got.Code != "error_rate_limited" || got.Message != "Errors over the limit are dropped." {
		t.Errorf("warning = %+v", got)
	}
	if want := time.Date(2026, 11, 18, 14, 12, 0, 0, time.UTC); !got.StartAt.Equal(want) {
		t.Errorf("StartAt = %v, want %v", got.StartAt, want)
	}
	if want := time.Date(2026, 11, 18, 15, 12, 0, 0, time.UTC); !got.EndAt.Equal(want) {
		t.Errorf("EndAt = %v, want %v", got.EndAt, want)
	}
	if got.LimitPerMinute == nil || *got.LimitPerMinute != 1 {
		t.Errorf("LimitPerMinute = %v, want 1", got.LimitPerMinute)
	}
	if resp.RequestID != "req_1" {
		t.Errorf("RequestID = %q, want req_1", resp.RequestID)
	}
}

// Time-ordered listings decode through a different envelope, so they need the
// same check.
func TestNoticeListCarriesWarnings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 0, `{"data":[],"pagination":{"has_older":false,"has_newer":false},"links":{},"meta":{"warnings":[`+rateLimitedWarning+`]}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbp_x")
	resp, err := c.Faults.ListNotices(context.Background(), "Xk9mZp", "1")
	if err != nil {
		t.Fatalf("ListNotices: %v", err)
	}
	if len(resp.Warnings) != 1 || resp.Warnings[0].Code != "error_rate_limited" {
		t.Errorf("Warnings = %+v, want the error_rate_limited warning", resp.Warnings)
	}
}

// The API leaves warnings out when there are none, and so does the serialized
// page, so MCP output gains no empty key.
func TestListWithoutWarningsOmitsThem(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 0, `{"data":[],"pagination":{"page":1,"per_page":25},"meta":{"request_id":"req_1"}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbp_x")
	resp, err := c.Faults.List(context.Background(), "Xk9mZp")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if resp.Warnings != nil {
		t.Errorf("Warnings = %+v, want nil", resp.Warnings)
	}
	out, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if strings.Contains(string(out), "warnings") {
		t.Errorf("serialized page = %s, want no warnings key", out)
	}
}

// A warning serializes with the API's own keys, since the MCP server passes it
// straight through.
func TestWarningSerializesWithAPIKeys(t *testing.T) {
	var w ResponseWarning
	if err := json.Unmarshal([]byte(rateLimitedWarning), &w); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	out, err := json.Marshal(ListResponse[Fault]{Data: []Fault{}, Warnings: []ResponseWarning{w}})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	for _, key := range []string{`"warnings":[`, `"code":"error_rate_limited"`, `"start_at":"2026-11-18T14:12:00Z"`, `"end_at":"2026-11-18T15:12:00Z"`, `"limit_per_minute":1`} {
		if !strings.Contains(string(out), key) {
			t.Errorf("serialized page = %s, want %s", out, key)
		}
	}
}
