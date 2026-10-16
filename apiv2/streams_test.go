package apiv2

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStreamsList(t *testing.T) {
	mockStreams := `{
		"results": [
			{"id": "abc123def456", "name": "Default", "slug": "default", "internal": false, "project_id": 123, "created_at": "2024-01-01T00:00:00Z"},
			{"id": "789ghi012jkl", "name": "Internal", "slug": "internal", "internal": true, "project_id": 123, "created_at": "2024-01-01T00:00:00Z"}
		],
		"links": {"self": "/v2/projects/123/streams"}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/streams" {
			t.Errorf("expected path /v2/projects/123/streams, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockStreams))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	streams, err := client.Streams.List(context.Background(), 123)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(streams) != 2 {
		t.Fatalf("expected 2 streams, got %d", len(streams))
	}

	if streams[0].ID != "abc123def456" {
		t.Errorf("expected first stream ID 'abc123def456', got %s", streams[0].ID)
	}

	if streams[0].Name != "Default" {
		t.Errorf("expected first stream name 'Default', got %s", streams[0].Name)
	}

	if !streams[1].Internal {
		t.Error("expected second stream to be internal")
	}

	if streams[0].ProjectID == nil || *streams[0].ProjectID != 123 {
		t.Errorf("expected first stream project_id 123, got %v", streams[0].ProjectID)
	}
}

// TestStreamUnmarshalNullProjectID verifies that account-level streams, whose
// project_id is null, decode to a nil ProjectID rather than erroring.
func TestStreamUnmarshalNullProjectID(t *testing.T) {
	var s Stream
	if err := json.Unmarshal([]byte(`{"id": "acct123", "name": "Account", "slug": "account", "internal": false, "project_id": null, "created_at": "2024-01-01T00:00:00Z"}`), &s); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if s.ProjectID != nil {
		t.Errorf("expected nil project_id for account-level stream, got %v", *s.ProjectID)
	}
}

func TestStreamsListError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errors": "Not found"}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	_, err := client.Streams.List(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
