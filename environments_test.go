package honeybadgerapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnvironmentsList(t *testing.T) {
	mockEnvironments := `{
		"results": [
			{"id": 1, "project_id": 123, "name": "production", "notifications": true, "created_at": "2024-01-01T00:00:00Z", "updated_at": "2024-01-01T00:00:00Z"},
			{"id": 2, "project_id": 123, "name": "staging", "notifications": false, "created_at": "2024-01-01T00:00:00Z", "updated_at": "2024-01-01T00:00:00Z"}
		],
		"links": {"self": "/v2/projects/123/environments"}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/environments" {
			t.Errorf("expected path /v2/projects/123/environments, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockEnvironments))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	environments, err := client.Environments.List(context.Background(), 123)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(environments) != 2 {
		t.Errorf("expected 2 environments, got %d", len(environments))
	}

	if environments[0].Name != "production" {
		t.Errorf("expected first environment name 'production', got %s", environments[0].Name)
	}
}

func TestEnvironmentsCreate(t *testing.T) {
	mockEnvironment := `{"id": 1, "project_id": 123, "name": "development", "notifications": true, "created_at": "2024-01-01T00:00:00Z", "updated_at": "2024-01-01T00:00:00Z"}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/environments" {
			t.Errorf("expected path /v2/projects/123/environments, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(mockEnvironment))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	params := EnvironmentParams{Name: "development"}
	environment, err := client.Environments.Create(context.Background(), 123, params)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if environment.Name != "development" {
		t.Errorf("expected environment name 'development', got %s", environment.Name)
	}
}

func TestEnvironmentsDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/environments/1" {
			t.Errorf("expected path /v2/projects/123/environments/1, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	err := client.Environments.Delete(context.Background(), 123, 1)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}
