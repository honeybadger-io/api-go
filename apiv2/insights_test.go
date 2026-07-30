package apiv2

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInsightsQuery(t *testing.T) {
	mockResponse := `{
		"results": [
			{"ts": "2024-01-01T00:00:00Z", "count": 10, "name": "web"},
			{"ts": "2024-01-01T01:00:00Z", "count": 15, "name": "api"}
		],
		"meta": {
			"query": "stats count() by event_type::str",
			"fields": ["ts", "count", "name"],
			"schema": [
				{"name": "ts", "type": "DateTime"},
				{"name": "count", "type": "UInt64"},
				{"name": "name", "type": "String"}
			],
			"rows": 2,
			"total_rows": 2,
			"start_at": "2024-01-01T00:00:00Z",
			"end_at": "2024-01-01T03:00:00Z"
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/insights/queries" {
			t.Errorf("expected path /v2/projects/123/insights/queries, got %s", r.URL.Path)
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
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	response, err := client.Insights.Query(context.Background(), 123, InsightsQueryRequest{
		Query: "stats count() by event_type::str",
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	if len(response.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(response.Results))
	}

	if response.Meta.Query != "stats count() by event_type::str" {
		t.Errorf("expected query in meta, got %s", response.Meta.Query)
	}

	if len(response.Meta.Fields) != 3 {
		t.Errorf("expected 3 fields, got %d", len(response.Meta.Fields))
	}

	if response.Meta.Rows != 2 {
		t.Errorf("expected 2 rows, got %d", response.Meta.Rows)
	}

	if response.Meta.TotalRows != 2 {
		t.Errorf("expected 2 total rows, got %d", response.Meta.TotalRows)
	}

	if response.Error != nil {
		t.Errorf("expected no inline error, got %v", response.Error)
	}
}

func TestInsightsQuery_WithAllOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"results": [],
			"meta": {
				"query": "fields @ts, message::str",
				"fields": [],
				"schema": [],
				"rows": 0,
				"total_rows": 0,
				"start_at": "2024-01-01T00:00:00Z",
				"end_at": "2024-01-07T00:00:00Z"
			}
		}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	response, err := client.Insights.Query(context.Background(), 123, InsightsQueryRequest{
		Query:    "fields @ts, message::str",
		Ts:       "week",
		Timezone: "America/New_York",
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	if response.Meta.Query != "fields @ts, message::str" {
		t.Errorf("expected query in meta, got %s", response.Meta.Query)
	}
}

func TestInsightsQuery_WithStreamIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}

		got, ok := body["stream_ids"].([]interface{})
		if !ok {
			t.Errorf("expected stream_ids array in request body, got %v", body["stream_ids"])
		} else if len(got) != 2 || got[0] != "Oh3Y3WdMFvde" || got[1] != "MuHadpB4C9G4" {
			t.Errorf("expected stream_ids [Oh3Y3WdMFvde MuHadpB4C9G4], got %v", got)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": [], "meta": {"query": "stats count()", "rows": 0, "total_rows": 0}}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	_, err := client.Insights.Query(context.Background(), 123, InsightsQueryRequest{
		Query:     "stats count()",
		StreamIDs: []string{"Oh3Y3WdMFvde", "MuHadpB4C9G4"},
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
}

func TestInsightsQuery_OmitsEmptyStreamIDs(t *testing.T) {
	cases := map[string][]string{
		"nil slice":   nil,
		"empty slice": {},
	}

	for name, streamIDs := range cases {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("failed to decode request body: %v", err)
				}

				if _, present := body["stream_ids"]; present {
					t.Errorf("expected stream_ids omitted, got %v", body["stream_ids"])
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"results": [], "meta": {"query": "stats count()", "rows": 0, "total_rows": 0}}`))
			}))
			defer server.Close()

			client := NewClient().
				WithBaseURL(server.URL).
				WithAuthToken("test-token")

			_, err := client.Insights.Query(context.Background(), 123, InsightsQueryRequest{
				Query:     "stats count()",
				StreamIDs: streamIDs,
			})
			if err != nil {
				t.Fatalf("Query() error = %v", err)
			}
		})
	}
}

func TestInsightsQuery_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"errors": "Invalid API token"}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("invalid-token")

	_, err := client.Insights.Query(context.Background(), 123, InsightsQueryRequest{
		Query: "stats count()",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.StatusCode != 401 {
		t.Errorf("expected status code 401, got %d", apiErr.StatusCode)
	}
}

func TestInsightsQuery_ProjectNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errors": "Project not found"}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	_, err := client.Insights.Query(context.Background(), 999, InsightsQueryRequest{
		Query: "stats count()",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.StatusCode != 404 {
		t.Errorf("expected status code 404, got %d", apiErr.StatusCode)
	}
}

func TestInsightsQuery_InlineError(t *testing.T) {
	mockResponse := `{
		"results": [],
		"meta": {
			"query": "stats count()",
			"fields": [],
			"schema": [],
			"rows": 0,
			"total_rows": 0,
			"start_at": "2024-01-01T00:00:00Z",
			"end_at": "2024-01-01T03:00:00Z"
		},
		"error": {
			"message": "query timed out"
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	response, err := client.Insights.Query(context.Background(), 123, InsightsQueryRequest{
		Query: "stats count()",
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	if response.Error == nil {
		t.Fatal("expected inline error, got nil")
	}

	if response.Error.Message != "query timed out" {
		t.Errorf("expected error message 'query timed out', got %s", response.Error.Message)
	}
}

func TestInsightsQuery_InvalidQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"errors": "Invalid query syntax"}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	_, err := client.Insights.Query(context.Background(), 123, InsightsQueryRequest{
		Query: "INVALID QUERY",
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
