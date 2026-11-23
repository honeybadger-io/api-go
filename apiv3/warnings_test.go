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

// Fault counts covering a period when errors were dropped carry the warning,
// and serialize flat, the way the MCP server passes them through.
func TestFaultSummaryCarriesWarnings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 0, `{"data":{"total":3,"environments":[{"environment":"production","resolved":false,"ignored":false,"count":2},{"environment":null,"resolved":true,"ignored":false,"count":1}]},"meta":{"request_id":"req_1","warnings":[`+rateLimitedWarning+`]}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbp_x")
	summary, err := c.Faults.Summary(context.Background(), "Xk9mZp")
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if summary.Total != 3 || len(summary.Environments) != 2 || summary.Environments[0].Count != 2 {
		t.Errorf("summary = %+v", summary)
	}
	if !summary.Environments[1].Environment.IsNull() {
		t.Errorf("Environments[1].Environment = %v, want null", summary.Environments[1].Environment)
	}
	if len(summary.Warnings) != 1 || summary.Warnings[0].Code != "error_rate_limited" {
		t.Errorf("Warnings = %+v, want the error_rate_limited warning", summary.Warnings)
	}

	out, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	for _, key := range []string{`"total":3`, `"environments":[`, `"environment":null`, `"warnings":[{"code":"error_rate_limited"`} {
		if !strings.Contains(string(out), key) {
			t.Errorf("serialized summary = %s, want %s", out, key)
		}
	}
}

// Without warnings the serialized summary has no warnings key.
func TestFaultSummaryWithoutWarningsOmitsThem(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 0, `{"data":{"total":0,"environments":[]},"meta":{"request_id":"req_1"}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbp_x")
	summary, err := c.Faults.Summary(context.Background(), "Xk9mZp")
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	out, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(out) != `{"environments":[],"total":0}` {
		t.Errorf("serialized summary = %s, want {\"environments\":[],\"total\":0}", out)
	}
}

// A project's occurrence counts covering a period when errors were dropped
// carry the warning, so a dip isn't read as the app getting healthier. They
// serialize flat, the way the MCP server passes them through.
func TestProjectOccurrencesCarryWarnings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 0, `{"data":{"project_id":"Xk9mZp","buckets":[{"timestamp":1704067200,"count":4}]},"meta":{"request_id":"req_1","period":"day","interval":"hour","warnings":[`+rateLimitedWarning+`]}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbp_x")
	series, err := c.Projects.Occurrences(context.Background(), "Xk9mZp", OccurrenceOptions{Period: PeriodDay})
	if err != nil {
		t.Fatalf("Occurrences: %v", err)
	}
	if series.ProjectId != "Xk9mZp" || len(series.Buckets) != 1 || series.Buckets[0].Count != 4 {
		t.Errorf("series = %+v", series)
	}
	if len(series.Warnings) != 1 || series.Warnings[0].Code != "error_rate_limited" {
		t.Errorf("Warnings = %+v, want the error_rate_limited warning", series.Warnings)
	}

	out, err := json.Marshal(series)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	for _, key := range []string{`"project_id":"Xk9mZp"`, `"buckets":[{`, `"warnings":[{"code":"error_rate_limited"`} {
		if !strings.Contains(string(out), key) {
			t.Errorf("serialized series = %s, want %s", out, key)
		}
	}
}

// Without warnings the serialized series has no warnings key.
func TestProjectOccurrencesWithoutWarningsOmitThem(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 0, `{"data":{"project_id":"Xk9mZp","buckets":[]},"meta":{"request_id":"req_1","period":"hour","interval":"minute"}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbp_x")
	series, err := c.Projects.Occurrences(context.Background(), "Xk9mZp", OccurrenceOptions{})
	if err != nil {
		t.Fatalf("Occurrences: %v", err)
	}
	out, err := json.Marshal(series)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(out) != `{"buckets":[],"project_id":"Xk9mZp"}` {
		t.Errorf("serialized series = %s, want {\"buckets\":[],\"project_id\":\"Xk9mZp\"}", out)
	}
}

const insightsQuotaWarning = `{"code":"insights_quota_exceeded","message":"Insights dropped events.","start_at":"2026-11-18T14:12:00Z","end_at":"2026-11-19T00:00:00Z"}`

// A query covering a period when Insights dropped events says so, so missing
// rows aren't read as nothing happening. Meta stays the raw block.
func TestInsightsQueryCarriesWarnings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 0, `{"data":{"results":[]},"links":{"web":"https://app.honeybadger.io/x"},"meta":{"request_id":"req_q","warnings":[`+insightsQuotaWarning+`]}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbp_x")
	result, err := c.Insights.Query(context.Background(), "Xk9mZp", InsightsQuery{Query: "fields @ts"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(result.Warnings) != 1 || result.Warnings[0].Code != "insights_quota_exceeded" || result.Warnings[0].LimitPerMinute != nil {
		t.Errorf("Warnings = %+v, want the insights_quota_exceeded warning with no limit_per_minute", result.Warnings)
	}
	if _, ok := result.Meta["warnings"]; !ok {
		t.Errorf("Meta = %v, want the raw warnings kept", result.Meta)
	}
	if result.RequestID != "req_q" {
		t.Errorf("RequestID = %q, want req_q", result.RequestID)
	}
}

// Without warnings the serialized result has no top-level warnings key.
func TestInsightsQueryWithoutWarningsOmitsThem(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 0, `{"data":{"results":[]},"links":{"web":"https://app.honeybadger.io/x"},"meta":{"request_id":"req_q"}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbp_x")
	result, err := c.Insights.Query(context.Background(), "Xk9mZp", InsightsQuery{Query: "fields @ts"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if result.Warnings != nil {
		t.Errorf("Warnings = %+v, want nil", result.Warnings)
	}
	out, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if strings.Contains(string(out), "warnings") {
		t.Errorf("serialized result = %s, want no warnings key", out)
	}
}
