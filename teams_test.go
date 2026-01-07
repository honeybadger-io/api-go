package honeybadgerapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTeamsList(t *testing.T) {
	mockTeams := `{
		"results": [
			{"id": 1, "name": "Engineering", "account_id": 100},
			{"id": 2, "name": "Product", "account_id": 100}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/teams" {
			t.Errorf("expected path /v2/teams, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("account_id") != "100" {
			t.Errorf("expected account_id=100, got %s", r.URL.Query().Get("account_id"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockTeams))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	teams, err := client.Teams.List(context.Background(), "100")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(teams) != 2 {
		t.Errorf("expected 2 teams, got %d", len(teams))
	}

	if teams[0].Name != "Engineering" {
		t.Errorf("expected first team name 'Engineering', got %s", teams[0].Name)
	}
}

func TestTeamsCreate(t *testing.T) {
	mockTeam := `{"id": 1, "name": "New Team", "account_id": 100}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/teams" {
			t.Errorf("expected path /v2/teams, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(mockTeam))
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	team, err := client.Teams.Create(context.Background(), "100", "New Team")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if team.Name != "New Team" {
		t.Errorf("expected team name 'New Team', got %s", team.Name)
	}
}

func TestTeamsDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE method, got %s", r.Method)
		}
		if r.URL.Path != "/v2/teams/1" {
			t.Errorf("expected path /v2/teams/1, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient().
		WithBaseURL(server.URL).
		WithAuthToken("test-token")

	err := client.Teams.Delete(context.Background(), 1)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}
