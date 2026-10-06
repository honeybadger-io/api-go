package apiv3

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/oapi-codegen/nullable"
)

func TestEnvironmentsRequests(t *testing.T) {
	env := `{"data":{"id":"e1","project_id":"Xk9mZp","name":"staging","notifications":true}}`
	ctx := context.Background()

	c, got := captureWrite(t, http.StatusCreated, env)
	if _, err := c.Environments.Create(ctx, "Xk9mZp", EnvironmentCreateParams{Name: "staging"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.method != http.MethodPost || got.path != "/v3/projects/Xk9mZp/environments" || got.body["name"] != "staging" {
		t.Errorf("Create sent %s %s %v", got.method, got.path, got.body)
	}

	c, got = captureWrite(t, http.StatusOK, env)
	if _, err := c.Environments.Update(ctx, "Xk9mZp", "e1",
		EnvironmentUpdateParams{Notifications: nullable.NewNullableWithValue(false)}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.method != http.MethodPatch || got.path != "/v3/projects/Xk9mZp/environments/e1" || got.body["notifications"] != false {
		t.Errorf("Update sent %s %s %v", got.method, got.path, got.body)
	}
	if _, present := got.body["name"]; present {
		t.Error("Update sent name though it was unset")
	}

	c, got = captureWrite(t, http.StatusNoContent, "")
	if err := c.Environments.Delete(ctx, "Xk9mZp", "e1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got.method != http.MethodDelete || got.path != "/v3/projects/Xk9mZp/environments/e1" {
		t.Errorf("Delete sent %s %s", got.method, got.path)
	}
}

func TestTeamsRequests(t *testing.T) {
	team := `{"data":{"id":"t1","account_id":"Ab3kL9","name":"Ops","member_count":0,"project_ids":[],"created_at":"2026-10-06T00:00:00Z","links":{"web":"https://app/x"}}}`
	ctx := context.Background()

	c, got := captureWrite(t, http.StatusCreated, team)
	if _, err := c.Teams.Create(ctx, TeamCreateParams{Name: "Ops",
		ProjectIds: nullable.NewNullableWithValue([]string{"Xk9mZp"})}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.method != http.MethodPost || got.path != "/v3/teams" || got.body["name"] != "Ops" {
		t.Errorf("Create sent %s %s %v", got.method, got.path, got.body)
	}

	c, got = captureWrite(t, http.StatusOK, `{"data":[],"pagination":{"page":1,"per_page":25,"total_count":0}}`)
	if _, err := c.Teams.List(ctx, Named("Ops")); err != nil {
		t.Fatalf("List: %v", err)
	}
	if got.path != "/v3/teams" || got.query.Get("name") != "Ops" {
		t.Errorf("List sent %s?%s", got.path, got.query.Encode())
	}

	c, got = captureWrite(t, http.StatusNoContent, "")
	if err := c.Teams.Delete(ctx, "t1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got.method != http.MethodDelete || got.path != "/v3/teams/t1" {
		t.Errorf("Delete sent %s %s", got.method, got.path)
	}
}

func TestSitesRequests(t *testing.T) {
	id := uuid.MustParse("9f8b6d2e-4c1a-4b7f-9e35-2a6c8d0f1b47")
	site := `{"data":{"id":"9f8b6d2e-4c1a-4b7f-9e35-2a6c8d0f1b47","project_id":"Xk9mZp","name":"Home","url":"https://example.com","active":true,"frequency":5,"locations":["us-east"],"match_type":"success","state":"up","created_at":"2026-10-06T00:00:00Z","links":{"web":"https://app/x"}}}`
	ctx := context.Background()

	c, got := captureWrite(t, http.StatusCreated, site)
	created, err := c.Sites.Create(ctx, "Xk9mZp", SiteCreateParams{Url: "https://example.com",
		Locations: nullable.NewNullableWithValue([]SiteLocation{"us-east"})})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.method != http.MethodPost || got.path != "/v3/projects/Xk9mZp/sites" || got.body["url"] != "https://example.com" {
		t.Errorf("Create sent %s %s %v", got.method, got.path, got.body)
	}
	if created.Id != id {
		t.Errorf("created id = %v, want %v", created.Id, id)
	}

	c, got = captureWrite(t, http.StatusOK, site)
	if _, err := c.Sites.Update(ctx, "Xk9mZp", id, SiteUpdateParams{Match: nullable.NewNullNullable[string]()}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.method != http.MethodPatch || got.path != "/v3/projects/Xk9mZp/sites/"+id.String() {
		t.Errorf("Update sent %s %s", got.method, got.path)
	}
	if v, present := got.body["match"]; !present || v != nil {
		t.Errorf("match = %v (present %v), want an explicit null", v, present)
	}

	c, got = captureWrite(t, http.StatusNoContent, "")
	if err := c.Sites.Delete(ctx, "Xk9mZp", id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got.method != http.MethodDelete || got.path != "/v3/projects/Xk9mZp/sites/"+id.String() {
		t.Errorf("Delete sent %s %s", got.method, got.path)
	}
}
