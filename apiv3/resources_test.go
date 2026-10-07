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

// Create and update share the nested input types, so one site list serves both.
func TestStatusPagesRequests(t *testing.T) {
	siteID := uuid.MustParse("9f8b6d2e-4c1a-4b7f-9e35-2a6c8d0f1b47")
	page := `{"data":{"id":"sp1","account_id":"Ab3kL9","name":"Status","url":"https://status.example.com","sites":[],"check_ins":[],"features":{},"hide_branding":false,"incidents_enabled":true,"message_enabled":false,"password_protected":false,"search_engine_indexing_disabled":false,"created_at":"2026-10-06T00:00:00Z","links":{"web":"https://app/x"}}}`
	sites := []StatusPageSiteInput{{SiteId: siteID, DisplayName: nullable.NewNullableWithValue("Home")}}
	ctx := context.Background()

	c, got := captureWrite(t, http.StatusCreated, page)
	if _, err := c.StatusPages.Create(ctx, StatusPageCreateParams{Name: "Status",
		Sites: nullable.NewNullableWithValue(sites)}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.method != http.MethodPost || got.path != "/v3/status_pages" || got.body["name"] != "Status" {
		t.Errorf("Create sent %s %s %v", got.method, got.path, got.body)
	}
	entries, ok := got.body["sites"].([]any)
	if !ok || len(entries) != 1 || entries[0].(map[string]any)["site_id"] != siteID.String() {
		t.Errorf("sites = %v", got.body["sites"])
	}

	c, got = captureWrite(t, http.StatusOK, page)
	if _, err := c.StatusPages.Update(ctx, "sp1", StatusPageUpdateParams{
		Sites:    nullable.NewNullableWithValue(sites),
		Features: nullable.NewNullNullable[StatusPageFeatures](),
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.method != http.MethodPatch || got.path != "/v3/status_pages/sp1" {
		t.Errorf("Update sent %s %s", got.method, got.path)
	}
	if v, present := got.body["features"]; !present || v != nil {
		t.Errorf("features = %v (present %v), want an explicit null", v, present)
	}

	c, got = captureWrite(t, http.StatusNoContent, "")
	if err := c.StatusPages.Delete(ctx, "sp1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got.method != http.MethodDelete || got.path != "/v3/status_pages/sp1" {
		t.Errorf("Delete sent %s %s", got.method, got.path)
	}
}

// Deploys page by cursor and filter by environment and who deployed.
func TestDeploysRequests(t *testing.T) {
	ctx := context.Background()
	c, got := captureWrite(t, http.StatusOK, `{"data":[],"time_series":{"has_older":false}}`)
	if _, err := c.Deploys.List(ctx, "Xk9mZp", Limit(5), Before("cur1"),
		InEnvironment("production"), DeployedBy("ci")); err != nil {
		t.Fatalf("List: %v", err)
	}
	if got.path != "/v3/projects/Xk9mZp/deploys" {
		t.Errorf("path = %s", got.path)
	}
	for key, want := range map[string]string{"limit": "5", "before": "cur1", "environment": "production", "local_username": "ci"} {
		if got.query.Get(key) != want {
			t.Errorf("%s = %q, want %q (query %s)", key, got.query.Get(key), want, got.query.Encode())
		}
	}

	c, got = captureWrite(t, http.StatusOK, `{"data":{"id":"d1","project_id":"Xk9mZp","environment":"production","created_at":"2026-10-06T00:00:00Z"}}`)
	if _, err := c.Deploys.Get(ctx, "Xk9mZp", "d1"); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.path != "/v3/projects/Xk9mZp/deploys/d1" {
		t.Errorf("path = %s", got.path)
	}
}

func TestSiteOutagesAndChecksRequests(t *testing.T) {
	id := uuid.MustParse("9f8b6d2e-4c1a-4b7f-9e35-2a6c8d0f1b47")
	ctx := context.Background()
	empty := `{"data":[],"time_series":{"has_older":false}}`

	c, got := captureWrite(t, http.StatusOK, empty)
	if _, err := c.Sites.ListOutages(ctx, "Xk9mZp", id, Limit(10)); err != nil {
		t.Fatalf("ListOutages: %v", err)
	}
	if got.path != "/v3/projects/Xk9mZp/sites/"+id.String()+"/outages" || got.query.Get("limit") != "10" {
		t.Errorf("ListOutages sent %s?%s", got.path, got.query.Encode())
	}

	c, got = captureWrite(t, http.StatusOK, empty)
	if _, err := c.Sites.ListUptimeChecks(ctx, "Xk9mZp", id); err != nil {
		t.Fatalf("ListUptimeChecks: %v", err)
	}
	if got.path != "/v3/projects/Xk9mZp/sites/"+id.String()+"/checks" {
		t.Errorf("ListUptimeChecks sent %s", got.path)
	}
}
