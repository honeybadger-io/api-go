package apiv3

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func TestNewClientDefaults(t *testing.T) {
	c := NewClient()
	if c.baseURL != DefaultBaseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, DefaultBaseURL)
	}
	// Deadlines are set per attempt instead (see the retry tests): a client-wide
	// timeout would cut off a delete that legitimately takes longer than 30s.
	if c.httpClient.Timeout != 0 {
		t.Errorf("timeout = %v, want none on the http.Client", c.httpClient.Timeout)
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
	if _, err := c.Projects.List(context.Background()); err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if want := "Bearer hbt_secret"; gotAuth != want {
		t.Errorf("Authorization = %q, want %q", gotAuth, want)
	}
	if want := "/v3/projects"; gotPath != want {
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
	if _, err := c.Projects.List(context.Background()); err != nil {
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
	if _, err := c.Projects.List(context.Background()); err != nil {
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
	if _, err := c.Projects.List(ctx); err != nil {
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

// Most operations declare meta as an optional, sometimes empty, object. The hook
// must stay quiet rather than reporting an empty id.
func TestRequestIDHookNotCalledWhenAbsent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 0, `{"data":[],"meta":{}}`)
	}))
	defer srv.Close()

	called := false
	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x").
		WithRequestIDHook(func(ctx context.Context, status int, id string) { called = true })

	if _, err := c.Projects.List(context.Background()); err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if called {
		t.Error("hook fired for a response with no request_id")
	}
}

// A body past the read cap is an error, not a quietly truncated response that
// fails later as malformed JSON.
func TestOversizedBodyIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[`))
		chunk := bytes.Repeat([]byte(" "), 1<<20)
		for written := 0; written <= maxBodyBytes; written += len(chunk) {
			if _, err := w.Write(chunk); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	_, err := c.Projects.List(context.Background())
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("err = %v, want the oversized body reported", err)
	}
}

// Every request names the library, after the caller's product when one is set.
func TestUserAgent(t *testing.T) {
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, r.Header.Get("User-Agent"))
		writeJSON(w, 0, `{"data":{"id":"p1","account_id":"a","name":"One","active":true}}`)
	}))
	defer srv.Close()

	base := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	ctx := context.Background()
	if _, err := base.Projects.Get(ctx, "p1"); err != nil {
		t.Fatal(err)
	}
	if _, err := base.WithUserAgent("terraform-provider-honeybadger/1.2.0").Projects.Get(ctx, "p1"); err != nil {
		t.Fatal(err)
	}
	want := []string{"honeybadger-api-go", "terraform-provider-honeybadger/1.2.0 honeybadger-api-go"}
	if !slices.Equal(got, want) {
		t.Errorf("User-Agent = %q, want %q", got, want)
	}
}
