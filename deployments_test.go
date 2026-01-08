package honeybadgerapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeploymentsList(t *testing.T) {
	mockDeployments := `{
		"results": [
			{
				"id": 1,
				"created_at": "2013-04-30T13:12:51Z",
				"environment": "production",
				"local_username": "deploy",
				"project_id": 123,
				"repository": "some/repo",
				"revision": "2013-04-29-take-2-16-g6cf7eae"
			},
			{
				"id": 2,
				"created_at": "2013-04-29T10:00:00Z",
				"environment": "staging",
				"local_username": "developer",
				"project_id": 123,
				"repository": "some/repo",
				"revision": "2013-04-28-abc123"
			}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/deploys" {
			t.Errorf("expected path /v2/projects/123/deploys, got %s", r.URL.Path)
		}
		// Check Basic Auth
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
		_, _ = w.Write([]byte(mockDeployments))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	deployments, err := client.Deployments.List(context.Background(), 123, DeploymentListOptions{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(deployments) != 2 {
		t.Errorf("expected 2 deployments, got %d", len(deployments))
	}

	if deployments[0].ID != 1 {
		t.Errorf("expected first deployment ID 1, got %d", deployments[0].ID)
	}

	if deployments[0].Environment != "production" {
		t.Errorf("expected first deployment environment 'production', got %s", deployments[0].Environment)
	}

	if deployments[0].Revision != "2013-04-29-take-2-16-g6cf7eae" {
		t.Errorf("expected first deployment revision '2013-04-29-take-2-16-g6cf7eae', got %s", deployments[0].Revision)
	}
}

func TestDeploymentsListWithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}

		// Verify query parameters
		query := r.URL.Query()
		if query.Get("environment") != "production" {
			t.Errorf("expected environment=production, got %s", query.Get("environment"))
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

	_, err := client.Deployments.List(context.Background(), 123, DeploymentListOptions{
		Environment: "production",
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
}

func TestDeploymentsGet(t *testing.T) {
	mockDeployment := `{
		"id": 1,
		"created_at": "2013-04-30T13:12:51Z",
		"environment": "production",
		"local_username": "deploy",
		"project_id": 123,
		"repository": "some/repo",
		"revision": "2013-04-29-take-2-16-g6cf7eae"
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/deploys/1" {
			t.Errorf("expected path /v2/projects/123/deploys/1, got %s", r.URL.Path)
		}
		// Check Basic Auth
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
		_, _ = w.Write([]byte(mockDeployment))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	deployment, err := client.Deployments.Get(context.Background(), 123, 1)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if deployment.ID != 1 {
		t.Errorf("expected deployment ID 1, got %d", deployment.ID)
	}

	if deployment.Environment != "production" {
		t.Errorf("expected deployment environment 'production', got %s", deployment.Environment)
	}
}

func TestDeploymentsDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/deploys/1" {
			t.Errorf("expected path /v2/projects/123/deploys/1, got %s", r.URL.Path)
		}
		// Check Basic Auth
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

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	err := client.Deployments.Delete(context.Background(), 123, 1)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}
