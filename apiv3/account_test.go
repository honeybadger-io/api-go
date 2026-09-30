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

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
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
