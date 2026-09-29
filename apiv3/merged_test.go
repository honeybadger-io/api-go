package apiv3

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// mergedServer answers requests for fault 1 as the API does once it has been
// merged into fault 2: a read gets a 301 to the survivor, a write a 409 naming
// it. It records every request it receives.
func mergedServer(t *testing.T) (*httptest.Server, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Method+" "+r.URL.Path)
		mu.Unlock()
		if r.URL.Path == "/v3/projects/Xk9mZp/faults/2" {
			writeJSON(w, http.StatusOK, `{"data":{"id":"2","project_id":"Xk9mZp","klass":"Survivor"}}`)
			return
		}
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusConflict, `{"error":{"code":"fault_merged","message":"Fault was merged",
			  "details":{"merged_into":"2","location":"/v3/projects/Xk9mZp/faults/2"}}}`)
			return
		}
		w.Header().Set("Location", "http://"+r.Host+"/v3/projects/Xk9mZp/faults/2")
		w.WriteHeader(http.StatusMovedPermanently)
	}))
	t.Cleanup(srv.Close)
	return srv, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), seen...)
	}
}

// A write to a merged fault fails with the survivor named in the details, and
// is never retried against the survivor.
func TestWriteToMergedFaultNamesTheSurvivor(t *testing.T) {
	srv, seen := mergedServer(t)
	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")

	err := c.Faults.Delete(context.Background(), "Xk9mZp", "1")
	if !errors.Is(err, ErrFaultMerged) {
		t.Fatalf("err = %v, want ErrFaultMerged", err)
	}
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %T, want *Error", err)
	}
	if id, ok := apiErr.MergedInto(); !ok || id != "2" {
		t.Errorf("MergedInto() = %q, %v; want 2, true", id, ok)
	}
	if got := seen(); len(got) != 1 || got[0] != "DELETE /v3/projects/Xk9mZp/faults/1" {
		t.Errorf("requests = %v, want only the DELETE", got)
	}
}

// A read of a merged fault must not quietly return a different fault.
func TestGetMergedFaultReportsTheSurvivor(t *testing.T) {
	srv, seen := mergedServer(t)
	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")

	f, err := c.Faults.Get(context.Background(), "Xk9mZp", "1")
	if !errors.Is(err, ErrFaultMerged) {
		t.Fatalf("Get = %+v, %v; want ErrFaultMerged", f, err)
	}
	var apiErr *Error
	if errors.As(err, &apiErr) {
		if id, ok := apiErr.MergedInto(); !ok || id != "2" {
			t.Errorf("MergedInto() = %q, %v; want 2, true", id, ok)
		}
	}
	if len(seen()) != 1 {
		t.Errorf("requests = %v, want the redirect not followed", seen())
	}
}

// A caller's own http.Client follows redirects by default; the library must not
// inherit that, and must not change the caller's client either.
func TestCustomHTTPClientDoesNotFollowRedirects(t *testing.T) {
	srv, seen := mergedServer(t)
	own := &http.Client{}
	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x").WithHTTPClient(own)

	if _, err := c.Faults.Get(context.Background(), "Xk9mZp", "1"); !errors.Is(err, ErrFaultMerged) {
		t.Fatalf("err = %v, want ErrFaultMerged", err)
	}
	if len(seen()) != 1 {
		t.Errorf("requests = %v, want the redirect not followed", seen())
	}
	if own.CheckRedirect != nil {
		t.Error("the caller's http.Client was modified")
	}
}

// A response cut off mid-body keeps its cause, so callers can tell a transport
// failure from an API error.
func TestTruncatedBodyUnwrapsToItsCause(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":`))
	}))
	t.Cleanup(srv.Close)
	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")

	_, err := c.Projects.Get(context.Background(), "Xk9mZp")
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("err = %v, want it to unwrap to io.ErrUnexpectedEOF", err)
	}
}

// Only a 301 to a fault is a merge. Any other redirect, such as http to https,
// stays a plain redirect error.
func TestRedirectElsewhereIsNotAMerge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "https://api.example.com/v3/projects")
		w.WriteHeader(http.StatusMovedPermanently)
	}))
	t.Cleanup(srv.Close)
	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")

	_, err := c.Projects.List(context.Background())
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusMovedPermanently {
		t.Fatalf("err = %v, want the 301 reported", err)
	}
	if errors.Is(err, ErrFaultMerged) {
		t.Errorf("a redirect to %s was reported as a merged fault", apiErr.Location)
	}
}
