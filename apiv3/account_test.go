package apiv3

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// v3 resolves the account from the credential, so no account id reaches the
// request path.
func TestAccountIsResolvedFromTheCredential(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		writeJSON(w, 0, `{"data":[],"pagination":{"page":1,"per_page":25}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbu_x")
	if _, err := c.Projects.List(context.Background()); err != nil {
		t.Fatalf("List: %v", err)
	}
	if want := "/v3/projects"; gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
}

// ambiguous_account is what a credential covering more than one account gets.
// There is no per-request account to retry with, so the error must be
// recognizable enough to tell the caller to use a single-account credential.
func TestAmbiguousAccountIsTyped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, `{"error":{"code":"ambiguous_account","message":"This credential covers more than one account."},"meta":{"request_id":"req_amb"}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbu_x")
	_, err := c.Projects.List(context.Background())
	if err == nil {
		t.Fatal("want an error for a 422 ambiguous_account")
	}

	var apiErr *Error
	if !asError(err, &apiErr) {
		t.Fatalf("err = %T, want *apiv3.Error", err)
	}
	if apiErr.Code != CodeAmbiguousAccount {
		t.Errorf("Code = %q, want %q", apiErr.Code, CodeAmbiguousAccount)
	}
	if apiErr.RequestID != "req_amb" {
		t.Errorf("RequestID = %q, want req_amb", apiErr.RequestID)
	}
}

// Usage reads the credential's account, so the path names no account. On a
// plan that limits uptime monitors and check-ins separately, each limit is
// present and the shared monitors pool is absent.
func TestAccountUsageReportsQuotas(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		writeJSON(w, 0, `{"data":{"id":"Ab3kL9","quota":{"errors_limit":150000,"errors_consumed":4200,"uptime_limit":5,"uptime_consumed":3,"check_ins_limit":20,"check_ins_consumed":7,"insights_daily_limit":5368709120,"insights_daily_consumed":4294967296,"insights_limited":false}}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hba_x")
	usage, err := c.Account.Usage(context.Background())
	if err != nil {
		t.Fatalf("Usage: %v", err)
	}
	if want := "/v3/account/usage"; gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
	if usage.Id != "Ab3kL9" {
		t.Errorf("Id = %q, want Ab3kL9", usage.Id)
	}
	q := usage.Quota
	if q.ErrorsLimit != 150000 || q.ErrorsConsumed != 4200 || q.UptimeConsumed != 3 || q.CheckInsConsumed != 7 {
		t.Errorf("quota = %+v", q)
	}
	if q.UptimeLimit == nil || *q.UptimeLimit != 5 {
		t.Errorf("UptimeLimit = %v, want 5", q.UptimeLimit)
	}
	if q.CheckInsLimit == nil || *q.CheckInsLimit != 20 {
		t.Errorf("CheckInsLimit = %v, want 20", q.CheckInsLimit)
	}
	if q.MonitorsLimit.IsSpecified() || q.MonitorsConsumed != nil {
		t.Errorf("monitors = %v/%v, want both absent on a per-type plan", q.MonitorsConsumed, q.MonitorsLimit)
	}
	// Insights counts bytes, which pass 32 bits on a busy day.
	if limit, err := q.InsightsDailyLimit.Get(); err != nil || limit != 5368709120 {
		t.Errorf("InsightsDailyLimit = %v (%v), want 5368709120", limit, err)
	}
	if q.InsightsDailyConsumed != 4294967296 || q.InsightsLimited {
		t.Errorf("insights = %d consumed, limited %v", q.InsightsDailyConsumed, q.InsightsLimited)
	}
}

// On a plan where uptime monitors and check-ins share one pool, the separate
// limits are absent and monitors_limit is null when the pool is uncapped. A
// null Insights limit is uncapped too, unlike 0, which blocks all ingestion.
func TestAccountUsageReportsAPooledPlan(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 0, `{"data":{"id":"Ab3kL9","quota":{"errors_limit":150000,"errors_consumed":4200,"uptime_consumed":3,"check_ins_consumed":7,"monitors_limit":null,"monitors_consumed":10,"insights_daily_limit":null,"insights_daily_consumed":0,"insights_limited":false}}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hba_x")
	usage, err := c.Account.Usage(context.Background())
	if err != nil {
		t.Fatalf("Usage: %v", err)
	}
	q := usage.Quota
	if q.UptimeLimit != nil || q.CheckInsLimit != nil {
		t.Errorf("UptimeLimit = %v, CheckInsLimit = %v, want both absent on a pooled plan", q.UptimeLimit, q.CheckInsLimit)
	}
	if !q.MonitorsLimit.IsNull() {
		t.Errorf("MonitorsLimit = %v, want null for an uncapped pool", q.MonitorsLimit)
	}
	if q.MonitorsConsumed == nil || *q.MonitorsConsumed != 10 {
		t.Errorf("MonitorsConsumed = %v, want 10", q.MonitorsConsumed)
	}
	if !q.InsightsDailyLimit.IsNull() {
		t.Errorf("InsightsDailyLimit = %v, want null for uncapped", q.InsightsDailyLimit)
	}
}

// 0 is a real Insights limit that blocks all ingestion, so it must not read as
// null.
func TestAccountUsageKeepsAZeroInsightsLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 0, `{"data":{"id":"Ab3kL9","quota":{"errors_limit":0,"errors_consumed":0,"uptime_consumed":0,"check_ins_consumed":0,"monitors_limit":25,"monitors_consumed":0,"insights_daily_limit":0,"insights_daily_consumed":0,"insights_limited":true}}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hba_x")
	usage, err := c.Account.Usage(context.Background())
	if err != nil {
		t.Fatalf("Usage: %v", err)
	}
	q := usage.Quota
	if limit, err := q.InsightsDailyLimit.Get(); err != nil || limit != 0 {
		t.Errorf("InsightsDailyLimit = %v (%v), want 0", limit, err)
	}
	if limit, err := q.MonitorsLimit.Get(); err != nil || limit != 25 {
		t.Errorf("MonitorsLimit = %v (%v), want 25", limit, err)
	}
}
