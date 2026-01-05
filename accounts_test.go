package honeybadgerapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAccountsList(t *testing.T) {
	mockAccounts := `[
		{"id": 1, "email": "account1@example.com", "name": "Account 1"},
		{"id": 2, "email": "account2@example.com", "name": "Account 2"}
	]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/accounts" {
			t.Errorf("expected path /v2/accounts, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockAccounts))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	accounts, err := client.Accounts.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(accounts) != 2 {
		t.Errorf("expected 2 accounts, got %d", len(accounts))
	}

	if accounts[0].Name != "Account 1" {
		t.Errorf("expected first account name 'Account 1', got %s", accounts[0].Name)
	}
}

func TestAccountsGet(t *testing.T) {
	quotaConsumed := 45.5
	mockAccount := `{
		"id": 1,
		"email": "account@example.com",
		"name": "My Account",
		"quota_consumed": 45.5
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/accounts/1" {
			t.Errorf("expected path /v2/accounts/1, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockAccount))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	account, err := client.Accounts.Get(context.Background(), 1)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if account.Name != "My Account" {
		t.Errorf("expected account name 'My Account', got %s", account.Name)
	}

	if account.QuotaConsumed == nil || *account.QuotaConsumed != quotaConsumed {
		t.Errorf("expected quota consumed 45.5, got %v", account.QuotaConsumed)
	}
}

func TestAccountsListUsers(t *testing.T) {
	mockUsers := `[
		{"id": 1, "role": "Owner", "name": "User 1", "email": "user1@example.com"},
		{"id": 2, "role": "Member", "name": "User 2", "email": "user2@example.com"}
	]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/accounts/1/users" {
			t.Errorf("expected path /v2/accounts/1/users, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockUsers))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	users, err := client.Accounts.ListUsers(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}

	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}

	if users[0].Role != "Owner" {
		t.Errorf("expected first user role 'Owner', got %s", users[0].Role)
	}
}
