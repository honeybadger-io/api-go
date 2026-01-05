package honeybadgerapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckInsList(t *testing.T) {
	reportPeriod := "1 day"
	gracePeriod := "5 minutes"
	mockCheckIns := `[
		{
			"id": 1,
			"name": "Daily Backup",
			"slug": "daily-backup",
			"schedule_type": "simple",
			"report_period": "1 day",
			"grace_period": "5 minutes",
			"project_id": 123,
			"created_at": "2024-01-01T00:00:00Z",
			"last_check_in_at": "2024-01-10T00:00:00Z"
		},
		{
			"id": 2,
			"name": "Hourly Sync",
			"slug": "hourly-sync",
			"schedule_type": "cron",
			"cron_schedule": "0 * * * *",
			"cron_timezone": "UTC",
			"project_id": 123,
			"created_at": "2024-01-01T00:00:00Z",
			"last_check_in_at": null
		}
	]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/check_ins" {
			t.Errorf("expected path /v2/projects/123/check_ins, got %s", r.URL.Path)
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
		_, _ = w.Write([]byte(mockCheckIns))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	checkIns, err := client.CheckIns.List(context.Background(), 123)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(checkIns) != 2 {
		t.Errorf("expected 2 check-ins, got %d", len(checkIns))
	}

	if checkIns[0].ID != 1 {
		t.Errorf("expected first check-in ID 1, got %d", checkIns[0].ID)
	}

	if checkIns[0].Name != "Daily Backup" {
		t.Errorf("expected first check-in name 'Daily Backup', got %s", checkIns[0].Name)
	}

	if checkIns[0].ReportPeriod == nil || *checkIns[0].ReportPeriod != reportPeriod {
		t.Errorf("expected first check-in report period '1 day', got %v", checkIns[0].ReportPeriod)
	}

	if checkIns[0].GracePeriod == nil || *checkIns[0].GracePeriod != gracePeriod {
		t.Errorf("expected first check-in grace period '5 minutes', got %v", checkIns[0].GracePeriod)
	}
}

func TestCheckInsGet(t *testing.T) {
	mockCheckIn := `{
		"id": 1,
		"name": "Daily Backup",
		"slug": "daily-backup",
		"schedule_type": "simple",
		"report_period": "1 day",
		"project_id": 123,
		"created_at": "2024-01-01T00:00:00Z",
		"last_check_in_at": "2024-01-10T00:00:00Z"
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/check_ins/1" {
			t.Errorf("expected path /v2/projects/123/check_ins/1, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockCheckIn))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	checkIn, err := client.CheckIns.Get(context.Background(), 123, 1)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if checkIn.ID != 1 {
		t.Errorf("expected check-in ID 1, got %d", checkIn.ID)
	}

	if checkIn.Name != "Daily Backup" {
		t.Errorf("expected check-in name 'Daily Backup', got %s", checkIn.Name)
	}
}

func TestCheckInsCreate(t *testing.T) {
	mockCheckIn := `{
		"id": 1,
		"name": "New Check-In",
		"slug": "new-check-in",
		"schedule_type": "simple",
		"report_period": "1 hour",
		"project_id": 123,
		"created_at": "2024-01-01T00:00:00Z",
		"last_check_in_at": null
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/check_ins" {
			t.Errorf("expected path /v2/projects/123/check_ins, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(mockCheckIn))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	reportPeriod := "1 hour"
	params := CheckInParams{
		Name:         "New Check-In",
		ScheduleType: "simple",
		ReportPeriod: &reportPeriod,
	}

	checkIn, err := client.CheckIns.Create(context.Background(), 123, params)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if checkIn.ID != 1 {
		t.Errorf("expected check-in ID 1, got %d", checkIn.ID)
	}

	if checkIn.Name != "New Check-In" {
		t.Errorf("expected check-in name 'New Check-In', got %s", checkIn.Name)
	}
}

func TestCheckInsUpdate(t *testing.T) {
	mockCheckIn := `{
		"id": 1,
		"name": "Updated Check-In",
		"slug": "updated-check-in",
		"schedule_type": "simple",
		"report_period": "2 hours",
		"project_id": 123,
		"created_at": "2024-01-01T00:00:00Z",
		"last_check_in_at": null
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/check_ins/1" {
			t.Errorf("expected path /v2/projects/123/check_ins/1, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockCheckIn))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	reportPeriod := "2 hours"
	params := CheckInParams{
		Name:         "Updated Check-In",
		ReportPeriod: &reportPeriod,
	}

	checkIn, err := client.CheckIns.Update(context.Background(), 123, 1, params)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if checkIn.Name != "Updated Check-In" {
		t.Errorf("expected check-in name 'Updated Check-In', got %s", checkIn.Name)
	}
}

func TestCheckInsDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/projects/123/check_ins/1" {
			t.Errorf("expected path /v2/projects/123/check_ins/1, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	err := client.CheckIns.Delete(context.Background(), 123, 1)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}
