package apiv3

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClientDefaults(t *testing.T) {
	c := NewClient()
	if c.baseURL != DefaultBaseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, DefaultBaseURL)
	}
	if c.httpClient.Timeout != 30*time.Second {
		t.Errorf("timeout = %v, want 30s", c.httpClient.Timeout)
	}
}

// WithBaseURL takes a host without the version segment, matching the v2 client
// and the MCP server's HONEYBADGER_API_URL. apiv3 appends /v3 itself.
func TestWithBaseURLAppendsVersion(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"https://app.honeybadger.io", "https://app.honeybadger.io/v3"},
		{"https://app.honeybadger.io/", "https://app.honeybadger.io/v3"},
		{"http://localhost:3000", "http://localhost:3000/v3"},
	}
	for _, tt := range tests {
		got := NewClient().WithBaseURL(tt.in).serverURL()
		if got != tt.want {
			t.Errorf("WithBaseURL(%q).serverURL() = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// A caller who already has a versioned URL should not get /v3/v3.
func TestWithBaseURLDoesNotDoubleVersion(t *testing.T) {
	got := NewClient().WithBaseURL("https://app.honeybadger.io/v3").serverURL()
	if got != "https://app.honeybadger.io/v3" {
		t.Errorf("serverURL() = %q, want no duplicated version segment", got)
	}
}

func TestBearerTokenIsSent(t *testing.T) {
	var gotAuth, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		writeJSON(w, 0, `{"data":[],"meta":{"request_id":"req_1"}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_secret")
	if _, err := c.gen().ListAccountsWithResponse(context.Background(), nil); err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if want := "Bearer hbt_secret"; gotAuth != want {
		t.Errorf("Authorization = %q, want %q", gotAuth, want)
	}
	if want := "/v3/accounts"; gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
}

// v3 rejects personal auth tokens and the spec's only scheme is Bearer, so
// there is deliberately no Basic-auth option. This test documents that choice:
// no request may ever carry a Basic Authorization header.
func TestNoBasicAuthIsSent(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		writeJSON(w, 0, `{"data":[]}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_secret")
	if _, err := c.gen().ListAccountsWithResponse(context.Background(), nil); err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if _, _, ok := parseBasic(gotAuth); ok {
		t.Errorf("Authorization was Basic (%q); apiv3 must only ever send Bearer", gotAuth)
	}
}

func TestRateLimitIsCaptured(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Limit", "360")
		w.Header().Set("X-RateLimit-Remaining", "359")
		w.Header().Set("X-RateLimit-Reset", "1784000000")
		writeJSON(w, 0, `{"data":[]}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	if _, err := c.gen().ListAccountsWithResponse(context.Background(), nil); err != nil {
		t.Fatalf("request failed: %v", err)
	}

	rl := c.LastRateLimit()
	if rl == nil {
		t.Fatal("LastRateLimit() = nil, want a snapshot")
	}
	if rl.Limit != 360 || rl.Remaining != 359 {
		t.Errorf("got limit=%d remaining=%d, want 360/359", rl.Limit, rl.Remaining)
	}
	if rl.Reset.Unix() != 1784000000 {
		t.Errorf("Reset = %v, want unix 1784000000", rl.Reset)
	}
}

// The request-id hook is context-aware: request_id lives in the response body,
// not a header, and is absent on 204s and on operations with an empty meta.
func TestRequestIDHookReceivesContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 0, `{"data":[],"meta":{"request_id":"req_abc"}}`)
	}))
	defer srv.Close()

	type ctxKey struct{}
	var gotID string
	var gotMarker any
	var gotStatus int

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x").
		WithRequestIDHook(func(ctx context.Context, status int, id string) {
			gotMarker = ctx.Value(ctxKey{})
			gotStatus = status
			gotID = id
		})

	ctx := context.WithValue(context.Background(), ctxKey{}, "marker")
	if _, err := c.gen().ListAccountsWithResponse(ctx, nil); err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if gotID != "req_abc" {
		t.Errorf("id = %q, want %q", gotID, "req_abc")
	}
	if gotStatus != http.StatusOK {
		t.Errorf("status = %d, want 200", gotStatus)
	}
	if gotMarker != "marker" {
		t.Errorf("hook lost the caller's context: marker = %v", gotMarker)
	}
}

func TestRequestIDHookNotCalledWhenAbsent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	called := false
	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x").
		WithRequestIDHook(func(ctx context.Context, status int, id string) { called = true })

	if _, err := c.gen().ListAccountsWithResponse(context.Background(), nil); err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if called {
		t.Error("hook fired for a 204 with no body; it should only fire when a request_id exists")
	}
}
