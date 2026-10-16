package apiv2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestUptimeList(t *testing.T) {
	mockSites := `{
		"results": [
			{
				"id": "abc123",
				"active": true,
				"frequency": 5,
				"last_checked_at": "2024-01-10T00:00:00Z",
				"match": "200",
				"match_type": "success",
				"name": "My Site",
				"state": "up",
				"url": "https://example.com"
			}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/sites" {
			t.Errorf("expected path /v2/projects/123/sites, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockSites))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	sites, err := client.Uptime.List(context.Background(), 123)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(sites) != 1 {
		t.Errorf("expected 1 site, got %d", len(sites))
	}

	if sites[0].ID != "abc123" {
		t.Errorf("expected site ID abc123, got %s", sites[0].ID)
	}

	if sites[0].Name != "My Site" {
		t.Errorf("expected site name 'My Site', got %s", sites[0].Name)
	}
}

func TestUptimeGet(t *testing.T) {
	mockSite := `{
		"id": "abc123",
		"active": true,
		"frequency": 5,
		"last_checked_at": "2024-01-10T00:00:00Z",
		"match": "200",
		"match_type": "success",
		"name": "My Site",
		"state": "up",
		"url": "https://example.com"
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/sites/abc123" {
			t.Errorf("expected path /v2/projects/123/sites/abc123, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockSite))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	site, err := client.Uptime.Get(context.Background(), 123, "abc123")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if site.ID != "abc123" {
		t.Errorf("expected site ID abc123, got %s", site.ID)
	}

	if site.Name != "My Site" {
		t.Errorf("expected site name 'My Site', got %s", site.Name)
	}
}

func TestUptimeCreate(t *testing.T) {
	mockSite := `{
		"id": "abc123",
		"active": true,
		"frequency": 5,
		"name": "New Site",
		"state": "up",
		"url": "https://example.com",
		"match_type": "success"
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/sites" {
			t.Errorf("expected path /v2/projects/123/sites, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(mockSite))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	params := SiteParams{
		Name: "New Site",
		URL:  "https://example.com",
	}

	site, err := client.Uptime.Create(context.Background(), 123, params)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if site.Name != "New Site" {
		t.Errorf("expected site name 'New Site', got %s", site.Name)
	}
}

func TestUptimeDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/sites/abc123" {
			t.Errorf("expected path /v2/projects/123/sites/abc123, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	err := client.Uptime.Delete(context.Background(), 123, "abc123")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}

func TestUptimeListOutages_WithTimestampOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/sites/abc123/outages" {
			t.Errorf("expected path /v2/projects/123/sites/abc123/outages, got %s", r.URL.Path)
		}

		query := r.URL.Query()
		if query.Get("created_after") != "1704067200" {
			t.Errorf("expected created_after=1704067200 (Unix timestamp), got %s", query.Get("created_after"))
		}
		if query.Get("created_before") != "1704153600" {
			t.Errorf("expected created_before=1704153600 (Unix timestamp), got %s", query.Get("created_before"))
		}
		if query.Get("limit") != "10" {
			t.Errorf("expected limit=10, got %s", query.Get("limit"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	createdAfter, _ := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
	createdBefore, _ := time.Parse(time.RFC3339, "2024-01-02T00:00:00Z")

	_, err := client.Uptime.ListOutages(context.Background(), 123, "abc123", OutageListOptions{
		CreatedAfter:  createdAfter,
		CreatedBefore: createdBefore,
		Limit:         10,
	})
	if err != nil {
		t.Fatalf("ListOutages() error = %v", err)
	}
}

func TestUptimeListUptimeChecks_WithTimestampOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/sites/abc123/uptime_checks" {
			t.Errorf("expected path /v2/projects/123/sites/abc123/uptime_checks, got %s", r.URL.Path)
		}

		query := r.URL.Query()
		if query.Get("created_after") != "1704067200" {
			t.Errorf("expected created_after=1704067200 (Unix timestamp), got %s", query.Get("created_after"))
		}
		if query.Get("created_before") != "1704153600" {
			t.Errorf("expected created_before=1704153600 (Unix timestamp), got %s", query.Get("created_before"))
		}
		if query.Get("limit") != "10" {
			t.Errorf("expected limit=10, got %s", query.Get("limit"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	createdAfter, _ := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
	createdBefore, _ := time.Parse(time.RFC3339, "2024-01-02T00:00:00Z")

	_, err := client.Uptime.ListUptimeChecks(context.Background(), 123, "abc123", UptimeCheckListOptions{
		CreatedAfter:  createdAfter,
		CreatedBefore: createdBefore,
		Limit:         10,
	})
	if err != nil {
		t.Fatalf("ListUptimeChecks() error = %v", err)
	}
}
