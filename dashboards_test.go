package honeybadgerapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDashboardsList(t *testing.T) {
	mockResponse := `{
		"results": [
			{
				"id": "abc123",
				"title": "Project Overview",
				"widgets": [{"id": "w1", "type": "errors"}],
				"is_default": true,
				"shared": true,
				"created_at": "2024-01-01T00:00:00Z",
				"updated_at": "2024-01-02T00:00:00Z",
				"project_id": 123
			},
			{
				"id": "def456",
				"title": "Performance",
				"widgets": [],
				"is_default": false,
				"shared": true,
				"created_at": "2024-01-03T00:00:00Z",
				"updated_at": "2024-01-04T00:00:00Z",
				"project_id": 123
			}
		],
		"links": {
			"self": "https://api.honeybadger.io/v2/projects/123/dashboards",
			"next": "",
			"prev": ""
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/dashboards" {
			t.Errorf("expected path /v2/projects/123/dashboards, got %s", r.URL.Path)
		}
		username, password, ok := r.BasicAuth()
		if !ok {
			t.Error("expected Basic Auth to be set")
		}
		if username != "test-token" {
			t.Errorf("expected Basic Auth username test-token, got %s", username)
		}
		if password != "" {
			t.Errorf("expected Basic Auth password to be empty, got %s", password)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	response, err := client.Dashboards.List(context.Background(), 123)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(response.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(response.Results))
	}

	if response.Results[0].ID != "abc123" {
		t.Errorf("expected first dashboard ID abc123, got %s", response.Results[0].ID)
	}

	if response.Results[0].Title != "Project Overview" {
		t.Errorf("expected first dashboard title 'Project Overview', got %s", response.Results[0].Title)
	}

	if !response.Results[0].IsDefault {
		t.Error("expected first dashboard to be default")
	}

	if len(response.Results[0].Widgets) != 1 {
		t.Errorf("expected 1 widget, got %d", len(response.Results[0].Widgets))
	}
}

func TestDashboardsGet(t *testing.T) {
	mockResponse := `{
		"id": "abc123",
		"title": "Project Overview",
		"widgets": [
			{"id": "w1", "type": "errors", "config": {"limit": 10}},
			{"id": "w2", "type": "insights_vis", "config": {"query": "stats count()"}}
		],
		"is_default": true,
		"shared": true,
		"created_at": "2024-01-01T00:00:00Z",
		"updated_at": "2024-01-02T00:00:00Z",
		"project_id": 123
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/dashboards/abc123" {
			t.Errorf("expected path /v2/projects/123/dashboards/abc123, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	dashboard, err := client.Dashboards.Get(context.Background(), 123, "abc123")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if dashboard.ID != "abc123" {
		t.Errorf("expected ID abc123, got %s", dashboard.ID)
	}

	if dashboard.Title != "Project Overview" {
		t.Errorf("expected title 'Project Overview', got %s", dashboard.Title)
	}

	if len(dashboard.Widgets) != 2 {
		t.Errorf("expected 2 widgets, got %d", len(dashboard.Widgets))
	}

	if dashboard.ProjectID != 123 {
		t.Errorf("expected project_id 123, got %d", dashboard.ProjectID)
	}
}

func TestDashboardsCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/dashboards" {
			t.Errorf("expected path /v2/projects/123/dashboards, got %s", r.URL.Path)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		dashboard, ok := body["dashboard"].(map[string]interface{})
		if !ok {
			t.Fatal("expected dashboard key in request body")
		}
		if dashboard["title"] != "My Dashboard" {
			t.Errorf("expected title 'My Dashboard', got %v", dashboard["title"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
			"id": "new123",
			"title": "My Dashboard",
			"widgets": [{"id": "w1", "type": "insights_vis"}],
			"is_default": false,
			"shared": true,
			"created_at": "2024-01-01T00:00:00Z",
			"updated_at": "2024-01-01T00:00:00Z",
			"project_id": 123
		}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	dashboard, err := client.Dashboards.Create(context.Background(), 123, DashboardRequest{
		Title: "My Dashboard",
		Widgets: []map[string]interface{}{
			{"type": "insights_vis", "config": map[string]interface{}{"query": "stats count()"}},
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if dashboard.ID != "new123" {
		t.Errorf("expected ID new123, got %s", dashboard.ID)
	}

	if dashboard.Title != "My Dashboard" {
		t.Errorf("expected title 'My Dashboard', got %s", dashboard.Title)
	}
}

func TestDashboardsUpdate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/dashboards/abc123" {
			t.Errorf("expected path /v2/projects/123/dashboards/abc123, got %s", r.URL.Path)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		dashboard, ok := body["dashboard"].(map[string]interface{})
		if !ok {
			t.Fatal("expected dashboard key in request body")
		}
		if dashboard["title"] != "Updated Title" {
			t.Errorf("expected title 'Updated Title', got %v", dashboard["title"])
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	result, err := client.Dashboards.Update(context.Background(), 123, "abc123", DashboardRequest{
		Title:   "Updated Title",
		Widgets: []map[string]interface{}{},
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if !result.Success {
		t.Error("expected success to be true")
	}
}

func TestDashboardsDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/dashboards/abc123" {
			t.Errorf("expected path /v2/projects/123/dashboards/abc123, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	result, err := client.Dashboards.Delete(context.Background(), 123, "abc123")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if !result.Success {
		t.Error("expected success to be true")
	}
}

func TestDashboardsCreate_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"errors": "Access denied"}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("invalid-token")

	_, err := client.Dashboards.Create(context.Background(), 123, DashboardRequest{
		Title:   "Test",
		Widgets: []map[string]interface{}{},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.StatusCode != 403 {
		t.Errorf("expected status code 403, got %d", apiErr.StatusCode)
	}
}

func TestDashboardsCreate_ValidationError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"errors": "/widgets/0: missing required property 'type'"}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	_, err := client.Dashboards.Create(context.Background(), 123, DashboardRequest{
		Title:   "Test",
		Widgets: []map[string]interface{}{{"invalid": true}},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.StatusCode != 422 {
		t.Errorf("expected status code 422, got %d", apiErr.StatusCode)
	}
}
