package apiv3

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// A configured client must be immutable. MCP's http transport takes a token per
// request, so if With* mutated a shared client, one tenant's request could be
// sent with another tenant's credential.
func TestWithBearerTokenDoesNotMutateReceiver(t *testing.T) {
	base := NewClient().WithBearerToken("token_a")
	derived := base.WithBearerToken("token_b")

	if base.bearerToken != "token_a" {
		t.Errorf("base token = %q, want it unchanged", base.bearerToken)
	}
	if derived.bearerToken != "token_b" {
		t.Errorf("derived token = %q", derived.bearerToken)
	}
	if base == derived {
		t.Error("With* returned the same client; it must return a copy")
	}
}

// Services must follow the clone, not keep pointing at the client they came
// from — otherwise a derived client's calls would use the original's credential.
func TestServicesFollowTheClone(t *testing.T) {
	base := NewClient().WithBearerToken("token_a")
	derived := base.WithBearerToken("token_b")

	if derived.Projects.client != derived {
		t.Error("Projects still points at the original client")
	}
	if derived.Faults.client != derived {
		t.Error("Faults still points at the original client")
	}
	if base.Projects.client != base {
		t.Error("the original's Projects was repointed at the clone")
	}
}

// The credential each request carries must match the client that issued it, even
// when clients derived from a shared base are used concurrently.
func TestConcurrentClientsDoNotCrossCredentials(t *testing.T) {
	var mu sync.Mutex
	seen := map[string]string{} // tenant (project id in path) -> token used

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The project id in the path identifies which tenant made the request.
		parts := strings.Split(r.URL.Path, "/")
		mu.Lock()
		seen[parts[len(parts)-1]] = r.Header.Get("Authorization")
		mu.Unlock()
		writeJSON(w, 0, `{"data":{"id":"p","account_id":"a","name":"n","active":true}}`)
	}))
	defer srv.Close()

	base := NewClient().WithBaseURL(srv.URL)

	var wg sync.WaitGroup
	for _, tenant := range []string{"a", "b", "c", "d", "e"} {
		wg.Add(1)
		go func(tenant string) {
			defer wg.Done()
			c := base.WithBearerToken("token_" + tenant)
			if _, err := c.Projects.Get(context.Background(), tenant); err != nil {
				t.Errorf("tenant %s: %v", tenant, err)
			}
		}(tenant)
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	for tenant, auth := range seen {
		if want := "Bearer token_" + tenant; auth != want {
			t.Errorf("tenant %s was sent %q, want %q", tenant, auth, want)
		}
	}
	if len(seen) != 5 {
		t.Errorf("saw %d tenants, want 5", len(seen))
	}
}

// WithHTTPClient(nil) must not produce a client that panics on first use.
func TestWithNilHTTPClientIsIgnored(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 0, `{"data":[]}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x").WithHTTPClient(nil)
	if c.httpClient == nil {
		t.Fatal("httpClient is nil; a nil argument must be ignored")
	}
	if _, err := c.Projects.List(context.Background()); err != nil {
		t.Fatalf("List after WithHTTPClient(nil): %v", err)
	}
}

// An error's rate-limit snapshot must come from the response that failed, not
// from whatever succeeded most recently.
func TestErrorRateLimitComesFromItsOwnResponse(t *testing.T) {
	var call int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call++
		if call == 1 {
			w.Header().Set("X-RateLimit-Limit", "360")
			w.Header().Set("X-RateLimit-Remaining", "359")
			w.Header().Set("X-RateLimit-Reset", "1784000000")
			writeJSON(w, 0, `{"data":[]}`)
			return
		}
		// Second response carries no rate-limit headers.
		writeJSON(w, http.StatusNotFound, `{"error":{"code":"not_found","message":"Resource not found"}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	if _, err := c.Projects.List(context.Background()); err != nil {
		t.Fatalf("first call: %v", err)
	}
	_, err := c.Projects.Get(context.Background(), "nope")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}

	var apiErr *Error
	if !asError(err, &apiErr) {
		t.Fatal("not an *apiv3.Error")
	}
	if apiErr.RateLimit != nil {
		t.Errorf("RateLimit = %+v, want nil: it was inherited from an earlier response",
			apiErr.RateLimit)
	}

	// The observational snapshot still reflects the earlier success.
	if rl := c.LastRateLimit(); rl == nil || rl.Remaining != 359 {
		t.Errorf("LastRateLimit() = %+v, want the earlier snapshot", rl)
	}
}
