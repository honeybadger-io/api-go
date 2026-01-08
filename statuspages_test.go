package honeybadgerapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStatusPagesList(t *testing.T) {
	mockStatusPages := `{
		"results": [
			{
				"id": "abc123",
				"name": "Public Status",
				"account_id": "100",
				"url": "https://status.example.com",
				"created_at": "2024-01-01T00:00:00Z",
				"sites": ["site1", "site2"],
				"check_ins": ["check1"]
			},
			{
				"id": "def456",
				"name": "Internal Status",
				"account_id": "100",
				"url": "https://internal-status.example.com",
				"created_at": "2024-01-02T00:00:00Z",
				"sites": [],
				"check_ins": []
			}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/accounts/100/status_pages" {
			t.Errorf("expected path /v2/accounts/100/status_pages, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockStatusPages))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	statusPages, err := client.StatusPages.List(context.Background(), "100")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(statusPages) != 2 {
		t.Errorf("expected 2 status pages, got %d", len(statusPages))
	}

	if statusPages[0].Name != "Public Status" {
		t.Errorf("expected first status page name 'Public Status', got %s", statusPages[0].Name)
	}

	if len(statusPages[0].Sites) != 2 {
		t.Errorf("expected 2 sites, got %d", len(statusPages[0].Sites))
	}
}

func TestStatusPagesGet(t *testing.T) {
	mockStatusPage := `{
		"id": "abc123",
		"name": "Public Status",
		"account_id": "100",
		"url": "https://status.example.com",
		"created_at": "2024-01-01T00:00:00Z",
		"sites": ["site1"],
		"check_ins": ["check1"]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/accounts/100/status_pages/abc123" {
			t.Errorf("expected path /v2/accounts/100/status_pages/abc123, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockStatusPage))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	statusPage, err := client.StatusPages.Get(context.Background(), "100", "abc123")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if statusPage.Name != "Public Status" {
		t.Errorf("expected status page name 'Public Status', got %s", statusPage.Name)
	}
}

func TestStatusPagesCreate(t *testing.T) {
	mockStatusPage := `{
		"id": "new123",
		"name": "New Status Page",
		"account_id": "100",
		"url": "https://new-status.example.com",
		"created_at": "2024-01-01T00:00:00Z",
		"sites": [],
		"check_ins": []
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/accounts/100/status_pages" {
			t.Errorf("expected path /v2/accounts/100/status_pages, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(mockStatusPage))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	params := StatusPageParams{Name: "New Status Page"}
	statusPage, err := client.StatusPages.Create(context.Background(), "100", params)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if statusPage.Name != "New Status Page" {
		t.Errorf("expected status page name 'New Status Page', got %s", statusPage.Name)
	}
}

func TestStatusPagesUpdate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/accounts/100/status_pages/abc123" {
			t.Errorf("expected path /v2/accounts/100/status_pages/abc123, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	params := StatusPageParams{Name: "Updated Status Page"}
	err := client.StatusPages.Update(context.Background(), "100", "abc123", params)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
}

func TestStatusPagesDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/accounts/100/status_pages/abc123" {
			t.Errorf("expected path /v2/accounts/100/status_pages/abc123, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	err := client.StatusPages.Delete(context.Background(), "100", "abc123")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}
