package apiv3

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultAccountIsMe(t *testing.T) {
	if AccountMe != "me" {
		t.Fatalf("AccountMe = %q, want %q", AccountMe, "me")
	}
	if got := NewClient().accountID(""); got != AccountMe {
		t.Errorf("accountID(\"\") = %q, want %q", got, AccountMe)
	}
}

func TestWithAccountIDOverridesDefault(t *testing.T) {
	c := NewClient().WithAccountID("Ab3kL9")
	if got := c.accountID(""); got != "Ab3kL9" {
		t.Errorf("accountID(\"\") = %q, want the client default", got)
	}
}

// A per-call account beats the client default, which beats "me".
func TestPerCallAccountWins(t *testing.T) {
	c := NewClient().WithAccountID("Ab3kL9")
	if got := c.accountID("Zz9zZ9"); got != "Zz9zZ9" {
		t.Errorf("accountID(\"Zz9zZ9\") = %q, want the per-call value", got)
	}
}

// The default must reach the wire as /v3/..., since that is what
// spares every caller from resolving an account id.
func TestMeReachesTheRequestPath(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		writeJSON(w, 0, `{"data":[],"pagination":{"page":1,"per_page":25,"total_count":0,"total_pages":0}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	if _, err := c.Projects.List(context.Background()); err != nil {
		t.Fatalf("List: %v", err)
	}
	if want := "/v3/projects"; gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
}

// v3 removed account IDs from paths — the account is resolved from the
// credential. InAccount is accepted but does not alter the request path.
func TestExplicitAccountDoesNotAlterPath(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		writeJSON(w, 0, `{"data":[],"pagination":{"page":1,"per_page":25,"total_count":0,"total_pages":0}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	_, err := c.Projects.List(context.Background(), InAccount("Ab3kL9"))
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if want := "/v3/projects"; gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
}

// ambiguous_account is the failure a multi-account credential hits when using
// "me". The error must be recognizable so callers can retry with an explicit id.
func TestAmbiguousAccountIsTyped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnprocessableEntity, `{"error":{"code":"ambiguous_account","message":"This credential covers more than one account, so \"me\" is ambiguous."},"meta":{"request_id":"req_amb"}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
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
