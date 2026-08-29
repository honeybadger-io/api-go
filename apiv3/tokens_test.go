package apiv3

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTokensGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if want := "/v3/token"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		writeJSON(w, 0, `{"data":{
		  "kind":"user","name":"CI token",
		  "scopes":["faults:read","insights:read"],
		  "account_id":"Ab3kL9","project_ids":["Xk9mZp","Nm8pQx"],
		  "expires_at":null,"last_used_at":"2026-07-29T12:00:00Z"
		},"meta":{"request_id":"req_tok"}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	info, err := c.Tokens.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if info.Kind != TokenKindUser {
		t.Errorf("Kind = %q, want user", info.Kind)
	}
	if info.Name != "CI token" {
		t.Errorf("Name = %q", info.Name)
	}
	if info.AccountID != "Ab3kL9" {
		t.Errorf("AccountID = %q, want Ab3kL9", info.AccountID)
	}
	if len(info.ProjectIDs) != 2 {
		t.Errorf("ProjectIDs = %v, want 2", info.ProjectIDs)
	}
	if !info.HasScope("faults:read") {
		t.Error("HasScope(faults:read) = false")
	}
	if info.HasScope("faults:write") {
		t.Error("HasScope(faults:write) = true, want false")
	}
	// A null expiry must read as absent, not as the zero time.
	if info.ExpiresAt != nil {
		t.Errorf("ExpiresAt = %v, want nil for null", *info.ExpiresAt)
	}
	if info.LastUsedAt == nil || *info.LastUsedAt != "2026-07-29T12:00:00Z" {
		t.Errorf("LastUsedAt = %v", info.LastUsedAt)
	}
}

// Token introspection reveals the account a credential belongs to, which a
// caller can use to resolve an ambiguous_account error by scoping their
// credential on the server side. v3 does not accept account IDs in paths.
func TestTokensGetRevealsAccountID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v3/projects":
			writeJSON(w, http.StatusUnprocessableEntity,
				`{"error":{"code":"ambiguous_account","message":"\"me\" is ambiguous"}}`)
		case "/v3/token":
			writeJSON(w, 0, `{"data":{"kind":"oauth","scopes":["projects:read"],"account_id":"Ab3kL9"}}`)
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
		}
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbo_x")

	_, err := c.Projects.List(context.Background())
	var apiErr *Error
	if !asError(err, &apiErr) || apiErr.Code != CodeAmbiguousAccount {
		t.Fatalf("err = %v, want ambiguous_account", err)
	}

	info, err := c.Tokens.Get(context.Background())
	if err != nil {
		t.Fatalf("Tokens.Get: %v", err)
	}
	if info.AccountID != "Ab3kL9" {
		t.Errorf("AccountID = %q, want Ab3kL9", info.AccountID)
	}
}
