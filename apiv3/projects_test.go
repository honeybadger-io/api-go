package apiv3

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// asError is errors.As with a concrete target, wrapped for readability in tests.
func asError(err error, target **Error) bool {
	return errors.As(err, target)
}

func TestProjectsListDecodesEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("per_page"); got != "50" {
			t.Errorf("per_page = %q, want 50", got)
		}
		if got := r.URL.Query().Get("page"); got != "2" {
			t.Errorf("page = %q, want 2", got)
		}
		writeJSON(w, 0, `{
		  "data":[{"id":"Xk9mZp","account_id":"Ab3kL9","name":"My Rails App","active":true}],
		  "pagination":{"page":2,"per_page":50,"total_count":51,"total_pages":2},
		  "links":{"next":"https://example.test/next"},
		  "meta":{"request_id":"req_list"}
		}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	resp, err := c.Projects.List(context.Background(), Page(2, 50))
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(resp.Data) != 1 {
		t.Fatalf("Data = %v, want 1 project", resp.Data)
	}
	if resp.Data[0].Id != "Xk9mZp" || resp.Data[0].Name != "My Rails App" {
		t.Errorf("project = %+v", resp.Data[0])
	}
	if resp.Pagination == nil || resp.Pagination.TotalCount != 51 {
		t.Errorf("Pagination = %+v", resp.Pagination)
	}
	if resp.TimeSeries != nil {
		t.Error("TimeSeries should be nil for an offset-paginated endpoint")
	}
	if resp.RequestID != "req_list" {
		t.Errorf("RequestID = %q, want req_list", resp.RequestID)
	}
	if resp.Links["next"] != "https://example.test/next" {
		t.Errorf("Links = %v", resp.Links)
	}
}

// Zero-valued options must not send page=0 or per_page=0, which the API would
// reject or interpret oddly.
func TestProjectsListOmitsZeroValuedParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("query = %q, want empty", r.URL.RawQuery)
		}
		writeJSON(w, 0, `{"data":[]}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	if _, err := c.Projects.List(context.Background()); err != nil {
		t.Fatalf("List: %v", err)
	}
}

func TestProjectsListAllWalksPages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("page") {
		case "", "1":
			writeJSON(w, 0, `{"data":[{"id":"p1","account_id":"a","name":"One","active":true}],
			  "pagination":{"page":1,"per_page":1,"total_count":2,"total_pages":2}}`)
		case "2":
			writeJSON(w, 0, `{"data":[{"id":"p2","account_id":"a","name":"Two","active":true}],
			  "pagination":{"page":2,"per_page":1,"total_count":2,"total_pages":2}}`)
		default:
			t.Errorf("unexpected page %q", r.URL.Query().Get("page"))
		}
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	all, err := c.Projects.ListAll(context.Background())
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(all) != 2 || all[0].Id != "p1" || all[1].Id != "p2" {
		t.Errorf("got %+v, want both projects in order", all)
	}
}

func TestProjectsGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if want := "/v3/accounts/me/projects/Xk9mZp"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		writeJSON(w, 0, `{"data":{"id":"Xk9mZp","account_id":"Ab3kL9","name":"My Rails App","active":true},
		  "meta":{"request_id":"req_get"}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	p, err := c.Projects.Get(context.Background(), "Xk9mZp")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if p.Id != "Xk9mZp" || p.AccountId != "Ab3kL9" {
		t.Errorf("project = %+v", p)
	}
}

func TestProjectsGetNotFoundIsTyped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, `{"error":{"code":"not_found","message":"Resource not found"}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	_, err := c.Projects.Get(context.Background(), "nope")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

// A 403 from a scoped token must surface the scope it needs, which is the whole
// value of the spec's InsufficientScope details.
func TestProjectsGetInsufficientScopeNamesScope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", `Bearer error="insufficient_scope"`)
		writeJSON(w, http.StatusForbidden, `{"error":{"code":"insufficient_scope","message":"Insufficient scope",
		  "details":{"required_scope":"projects:read","token_scopes":["faults:read"]}}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	_, err := c.Projects.Get(context.Background(), "Xk9mZp")
	if !errors.Is(err, ErrInsufficientScope) {
		t.Fatalf("err = %v, want ErrInsufficientScope", err)
	}

	var apiErr *Error
	if !asError(err, &apiErr) {
		t.Fatal("not an *apiv3.Error")
	}
	if apiErr.RequiredScope() != "projects:read" {
		t.Errorf("RequiredScope() = %q, want projects:read", apiErr.RequiredScope())
	}
}

// A 429 must carry the reset time so callers can report when to retry rather
// than blocking.
func TestProjectsListRateLimitedCarriesReset(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Limit", "360")
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", "1784000000")
		writeJSON(w, http.StatusTooManyRequests, `{"error":{"code":"rate_limit_exceeded","message":"Rate limit exceeded"}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	_, err := c.Projects.List(context.Background())
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("err = %v, want ErrRateLimited", err)
	}

	var apiErr *Error
	if !asError(err, &apiErr) {
		t.Fatal("not an *apiv3.Error")
	}
	if apiErr.RateLimit == nil {
		t.Fatal("RateLimit is nil; the 429 must carry the reset")
	}
	if apiErr.RateLimit.Reset.Unix() != 1784000000 {
		t.Errorf("Reset = %v", apiErr.RateLimit.Reset)
	}
}

// A 200 whose body is not the documented envelope must produce a typed error
// that still carries the status and the raw body. Asserting only "some error"
// would pass on the generated parser's bare json.SyntaxError, which discards
// both.
func TestProjectsGetMalformedBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 0, `<html>not json</html>`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	_, err := c.Projects.Get(context.Background(), "Xk9mZp")
	if err == nil {
		t.Fatal("want an error for a non-JSON 200")
	}

	var apiErr *Error
	if !asError(err, &apiErr) {
		t.Fatalf("err = %T (%v), want *apiv3.Error", err, err)
	}
	if apiErr.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200 preserved", apiErr.StatusCode)
	}
	if !strings.Contains(string(apiErr.Body), "not json") {
		t.Errorf("Body = %q, want the raw body preserved", apiErr.Body)
	}
	if !strings.Contains(apiErr.Error(), "did not match the documented envelope") {
		t.Errorf("Error() = %q", apiErr.Error())
	}
}

// The spec marks data optional, so {} decodes cleanly. Returning a zero-valued
// Project would hand back an empty id as if it were real.
func TestProjectsGetRejectsMissingDataMember(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 0, `{"meta":{"request_id":"req_x"}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	_, err := c.Projects.Get(context.Background(), "Xk9mZp")
	if err == nil {
		t.Fatal("want an error when the response has no data member")
	}
	if !strings.Contains(err.Error(), "no data member") {
		t.Errorf("err = %v, want it to name the missing data member", err)
	}
}

// An error body that is not the documented envelope must still reach parseError
// through a real call, not just in a unit test of parseError itself.
func TestProxyErrorBodyStillTyped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("<html>502 Bad Gateway</html>"))
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	_, err := c.Projects.List(context.Background())

	var apiErr *Error
	if !asError(err, &apiErr) {
		t.Fatalf("err = %T (%v), want *apiv3.Error", err, err)
	}
	if apiErr.StatusCode != http.StatusBadGateway {
		t.Errorf("StatusCode = %d, want 502", apiErr.StatusCode)
	}
	if !strings.Contains(apiErr.Error(), "502") {
		t.Errorf("Error() = %q, want it to mention the status", apiErr.Error())
	}
}

// writeJSON mirrors what the real API sends. The generated decoders only parse a
// body when Content-Type names JSON, and Go's ResponseWriter sniffs a JSON body
// as text/plain, so tests must set it explicitly.
func writeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	if status != 0 {
		w.WriteHeader(status)
	}
	_, _ = w.Write([]byte(body))
}
