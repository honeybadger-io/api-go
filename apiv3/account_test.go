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

// Usage reads the credential's account, so the path names no account, and a
// null limit is how the API says a quota is unlimited.
func TestAccountUsageReportsQuotas(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		writeJSON(w, 0, `{"data":{"id":"Ab3kL9","quota":{"errors_limit":150000,"errors_consumed":4200,"uptime_limit":null,"uptime_consumed":3,"check_ins_limit":20,"check_ins_consumed":7,"insights_daily_limit":5368709120,"insights_daily_consumed":4294967296,"insights_limited":false}}}`)
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
	if !q.UptimeLimit.IsNull() {
		t.Errorf("UptimeLimit = %v, want null for unlimited", q.UptimeLimit)
	}
	if limit, err := q.CheckInsLimit.Get(); err != nil || limit != 20 {
		t.Errorf("CheckInsLimit = %v (%v), want 20", limit, err)
	}
	// Insights counts bytes, which pass 32 bits on a busy day.
	if q.InsightsDailyLimit != 5368709120 || q.InsightsDailyConsumed != 4294967296 || q.InsightsLimited {
		t.Errorf("insights = %d/%d limited %v", q.InsightsDailyConsumed, q.InsightsDailyLimit, q.InsightsLimited)
	}
}
