package honeybadgerapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAlarmsList(t *testing.T) {
	mockResponse := `{
		"results": [
			{
				"id": "abc123",
				"name": "High Error Rate",
				"description": "Triggers when error count exceeds threshold",
				"state": "ok",
				"query": "filter event_type::str == \"notice\" | stats count()",
				"stream_ids": ["default"],
				"evaluation_period": "PT5M",
				"lookback_lag": "PT1M",
				"trigger_config": {"type": "alert_result_count", "config": {"operator": "gt", "value": 10}},
				"created_at": "2024-01-01T00:00:00Z",
				"updated_at": "2024-01-02T00:00:00Z",
				"url": "https://app.honeybadger.io/projects/123/insights/alarms/abc123",
				"project_id": 123
			}
		],
		"links": {
			"self": "https://api.honeybadger.io/v2/projects/123/alarms",
			"next": "",
			"prev": ""
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/alarms" {
			t.Errorf("expected path /v2/projects/123/alarms, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	response, err := client.Alarms.List(context.Background(), 123)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(response.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(response.Results))
	}

	if response.Results[0].ID != "abc123" {
		t.Errorf("expected alarm ID abc123, got %s", response.Results[0].ID)
	}

	if response.Results[0].Name != "High Error Rate" {
		t.Errorf("expected alarm name 'High Error Rate', got %s", response.Results[0].Name)
	}

	if response.Results[0].State != "ok" {
		t.Errorf("expected state 'ok', got %s", response.Results[0].State)
	}
}

func TestAlarmsGet(t *testing.T) {
	mockResponse := `{
		"id": "abc123",
		"name": "High Error Rate",
		"description": "Triggers when error count exceeds threshold",
		"state": "alarm",
		"query": "filter event_type::str == \"notice\" | stats count()",
		"stream_ids": ["default"],
		"evaluation_period": "PT5M",
		"lookback_lag": "PT1M",
		"trigger_config": {"type": "alert_result_count", "config": {"operator": "gt", "value": 10}},
		"last_checked_at": "2024-01-02T12:00:00Z",
		"next_check_at": "2024-01-02T12:05:00Z",
		"created_at": "2024-01-01T00:00:00Z",
		"updated_at": "2024-01-02T00:00:00Z",
		"url": "https://app.honeybadger.io/projects/123/insights/alarms/abc123",
		"project_id": 123
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/alarms/abc123" {
			t.Errorf("expected path /v2/projects/123/alarms/abc123, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	alarm, err := client.Alarms.Get(context.Background(), 123, "abc123")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if alarm.ID != "abc123" {
		t.Errorf("expected ID abc123, got %s", alarm.ID)
	}

	if alarm.State != "alarm" {
		t.Errorf("expected state 'alarm', got %s", alarm.State)
	}

	if alarm.LastCheckedAt == nil {
		t.Error("expected LastCheckedAt to be set")
	}
}

func TestAlarmsCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/alarms" {
			t.Errorf("expected path /v2/projects/123/alarms, got %s", r.URL.Path)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		alarm, ok := body["alarm"].(map[string]interface{})
		if !ok {
			t.Fatal("expected alarm key in request body")
		}
		if alarm["name"] != "New Alarm" {
			t.Errorf("expected name 'New Alarm', got %v", alarm["name"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
			"id": "new123",
			"name": "New Alarm",
			"description": "",
			"state": "initial",
			"query": "stats count()",
			"stream_ids": ["default"],
			"evaluation_period": "PT5M",
			"trigger_config": {"type": "alert_result_count", "config": {"operator": "gt", "value": 5}},
			"created_at": "2024-01-01T00:00:00Z",
			"updated_at": "2024-01-01T00:00:00Z",
			"url": "https://app.honeybadger.io/projects/123/insights/alarms/new123",
			"project_id": 123
		}`))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	alarm, err := client.Alarms.Create(context.Background(), 123, AlarmRequest{
		Name:             "New Alarm",
		Query:            "stats count()",
		EvaluationPeriod: "PT5M",
		TriggerConfig: map[string]interface{}{
			"type":   "alert_result_count",
			"config": map[string]interface{}{"operator": "gt", "value": 5},
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if alarm.ID != "new123" {
		t.Errorf("expected ID new123, got %s", alarm.ID)
	}

	if alarm.Name != "New Alarm" {
		t.Errorf("expected name 'New Alarm', got %s", alarm.Name)
	}
}

func TestAlarmsUpdate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/alarms/abc123" {
			t.Errorf("expected path /v2/projects/123/alarms/abc123, got %s", r.URL.Path)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		alarm, ok := body["alarm"].(map[string]interface{})
		if !ok {
			t.Fatal("expected alarm key in request body")
		}
		if alarm["name"] != "Updated Alarm" {
			t.Errorf("expected name 'Updated Alarm', got %v", alarm["name"])
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	result, err := client.Alarms.Update(context.Background(), 123, "abc123", AlarmRequest{
		Name:             "Updated Alarm",
		Query:            "stats count()",
		EvaluationPeriod: "PT10M",
		TriggerConfig: map[string]interface{}{
			"type":   "alert_result_count",
			"config": map[string]interface{}{"operator": "gt", "value": 20},
		},
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if !result.Success {
		t.Error("expected success to be true")
	}
}

func TestAlarmsDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/alarms/abc123" {
			t.Errorf("expected path /v2/projects/123/alarms/abc123, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	result, err := client.Alarms.Delete(context.Background(), 123, "abc123")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if !result.Success {
		t.Error("expected success to be true")
	}
}

func TestAlarmsHistory(t *testing.T) {
	mockResponse := `{
		"triggers": [
			{
				"id": "trigger1",
				"state": "alarm",
				"result": {"count": 15},
				"created_at": "2024-01-02T12:00:00Z"
			},
			{
				"id": "trigger2",
				"state": "ok",
				"result": {"count": 3},
				"created_at": "2024-01-02T11:55:00Z"
			}
		],
		"links": {
			"self": "https://api.honeybadger.io/v2/projects/123/alarms/abc123/history",
			"next": "",
			"prev": ""
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/alarms/abc123/history" {
			t.Errorf("expected path /v2/projects/123/alarms/abc123/history, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	response, err := client.Alarms.History(context.Background(), 123, "abc123", 0)
	if err != nil {
		t.Fatalf("History() error = %v", err)
	}

	if len(response.Triggers) != 2 {
		t.Errorf("expected 2 triggers, got %d", len(response.Triggers))
	}

	if response.Triggers[0].ID != "trigger1" {
		t.Errorf("expected first trigger ID trigger1, got %s", response.Triggers[0].ID)
	}

	if response.Triggers[0].State != "alarm" {
		t.Errorf("expected first trigger state 'alarm', got %s", response.Triggers[0].State)
	}
}

func TestAlarmsHistory_WithPage(t *testing.T) {
	mockResponse := `{
		"triggers": [],
		"links": {"self": "", "next": "", "prev": ""}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/alarms/abc123/history" {
			t.Errorf("expected path /v2/projects/123/alarms/abc123/history, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("page") != "3" {
			t.Errorf("expected page=3 query parameter, got %q", r.URL.Query().Get("page"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	_, err := client.Alarms.History(context.Background(), 123, "abc123", 3)
	if err != nil {
		t.Fatalf("History() error = %v", err)
	}
}
