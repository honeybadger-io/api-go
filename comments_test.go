package honeybadgerapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCommentsList(t *testing.T) {
	mockComments := `{
		"results": [
			{
				"id": 1,
				"fault_id": 100,
				"event": "comment",
				"source": "user",
				"created_at": "2024-01-01T00:00:00Z",
				"author": "Test User",
				"body": "This is a comment"
			},
			{
				"id": 2,
				"fault_id": 100,
				"event": "comment",
				"source": "system",
				"created_at": "2024-01-02T00:00:00Z",
				"author": "",
				"body": "Automated comment"
			}
		],
		"links": {"self": "/v2/projects/123/faults/100/comments"}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/faults/100/comments" {
			t.Errorf("expected path /v2/projects/123/faults/100/comments, got %s", r.URL.Path)
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
		_, _ = w.Write([]byte(mockComments))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	comments, err := client.Comments.List(context.Background(), 123, 100)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(comments) != 2 {
		t.Errorf("expected 2 comments, got %d", len(comments))
	}

	if comments[0].ID != 1 {
		t.Errorf("expected first comment ID 1, got %d", comments[0].ID)
	}

	if comments[0].Body != "This is a comment" {
		t.Errorf("expected first comment body 'This is a comment', got %s", comments[0].Body)
	}
}

func TestCommentsGet(t *testing.T) {
	mockComment := `{
		"id": 1,
		"fault_id": 100,
		"event": "comment",
		"source": "user",
		"created_at": "2024-01-01T00:00:00Z",
		"author": "Test User",
		"body": "This is a comment"
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/faults/100/comments/1" {
			t.Errorf("expected path /v2/projects/123/faults/100/comments/1, got %s", r.URL.Path)
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
		_, _ = w.Write([]byte(mockComment))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	comment, err := client.Comments.Get(context.Background(), 123, 100, 1)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if comment.ID != 1 {
		t.Errorf("expected comment ID 1, got %d", comment.ID)
	}

	if comment.Body != "This is a comment" {
		t.Errorf("expected comment body 'This is a comment', got %s", comment.Body)
	}
}

func TestCommentsCreate(t *testing.T) {
	mockComment := `{
		"id": 1,
		"fault_id": 100,
		"event": "comment",
		"source": "user",
		"created_at": "2024-01-01T00:00:00Z",
		"author": "Test User",
		"body": "New comment"
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/faults/100/comments" {
			t.Errorf("expected path /v2/projects/123/faults/100/comments, got %s", r.URL.Path)
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
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(mockComment))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	comment, err := client.Comments.Create(context.Background(), 123, 100, "New comment")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if comment.ID != 1 {
		t.Errorf("expected comment ID 1, got %d", comment.ID)
	}

	if comment.Body != "New comment" {
		t.Errorf("expected comment body 'New comment', got %s", comment.Body)
	}
}

func TestCommentsUpdate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/faults/100/comments/1" {
			t.Errorf("expected path /v2/projects/123/faults/100/comments/1, got %s", r.URL.Path)
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

	err := client.Comments.Update(context.Background(), 123, 100, 1, "Updated comment")
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
}

func TestCommentsDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/faults/100/comments/1" {
			t.Errorf("expected path /v2/projects/123/faults/100/comments/1, got %s", r.URL.Path)
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

	err := client.Comments.Delete(context.Background(), 123, 100, 1)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}
