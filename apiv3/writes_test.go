package apiv3

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// captureWrite records what a write actually put on the wire, which is the only
// thing worth asserting about a request body assembled from a partial schema.
type captured struct {
	method string
	path   string
	body   map[string]any
}

func captureWrite(t *testing.T, status int, response string) (*Client, *captured) {
	t.Helper()
	got := &captured{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.method, got.path = r.Method, r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &got.body)
		}
		if response == "" {
			w.WriteHeader(status)
			return
		}
		writeJSON(w, status, response)
	}))
	t.Cleanup(srv.Close)
	return NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x"), got
}

func TestFaultsResolveSendsFaultIDs(t *testing.T) {
	c, got := captureWrite(t, http.StatusNoContent, "")

	if err := c.Faults.Resolve(context.Background(), "Xk9mZp", []string{"f1", "f2"}); err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if got.method != http.MethodPost {
		t.Errorf("method = %q, want POST", got.method)
	}
	if want := "/v3/accounts/me/projects/Xk9mZp/faults/resolve"; got.path != want {
		t.Errorf("path = %q, want %q", got.path, want)
	}
	ids, ok := got.body["fault_ids"].([]any)
	if !ok || len(ids) != 2 || ids[0] != "f1" {
		t.Errorf("fault_ids = %v", got.body["fault_ids"])
	}
}

// A 204 carries no body, so it must not be treated as a malformed envelope.
func TestDeleteAcceptsNoContent(t *testing.T) {
	c, got := captureWrite(t, http.StatusNoContent, "")

	if err := c.Faults.Delete(context.Background(), "Xk9mZp", "f1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got.method != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", got.method)
	}
}

func TestPauseAndResumeRecordingSendNoBody(t *testing.T) {
	for _, tc := range []struct {
		name string
		call func(*Client) error
		path string
	}{
		{"pause", func(c *Client) error {
			return c.Faults.PauseRecording(context.Background(), "Xk9mZp", "f1")
		}, "/v3/accounts/me/projects/Xk9mZp/faults/f1/pause_recording"},
		{"resume", func(c *Client) error {
			return c.Faults.ResumeRecording(context.Background(), "Xk9mZp", "f1")
		}, "/v3/accounts/me/projects/Xk9mZp/faults/f1/resume_recording"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, got := captureWrite(t, http.StatusNoContent, "")
			if err := tc.call(c); err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			if got.path != tc.path {
				t.Errorf("path = %q, want %q", got.path, tc.path)
			}
			if got.body != nil {
				t.Errorf("body = %v, want none", got.body)
			}
		})
	}
}

// An account token can hold faults:write and still be refused, because a comment
// attributes text to a person.
func TestAddCommentRequiresUserToken(t *testing.T) {
	c, _ := captureWrite(t, http.StatusForbidden,
		`{"error":{"code":"requires_user_token","message":"This endpoint records the person who acted"}}`)

	err := c.Faults.AddComment(context.Background(), "Xk9mZp", "f1", "looking into it")
	if !errors.Is(err, ErrRequiresUserToken) {
		t.Fatalf("err = %v, want ErrRequiresUserToken", err)
	}
}

func TestProjectsCreateSendsRequiredName(t *testing.T) {
	c, got := captureWrite(t, http.StatusCreated,
		`{"data":{"id":"Xk9mZp","account_id":"Ab3kL9","name":"New App","active":true}}`)

	p, err := c.Projects.Create(context.Background(), ProjectParams{Name: "New App"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.body["name"] != "New App" {
		t.Errorf("name = %v", got.body["name"])
	}
	if p.Id != "Xk9mZp" {
		t.Errorf("returned project = %+v", p)
	}
}

// An update omits what it was not given, so unset fields are left alone rather
// than blanked — except name, which the schema requires on update as well as
// create, so a caller must always supply it.
func TestCheckInUpdateOmitsUnsetFields(t *testing.T) {
	c, got := captureWrite(t, http.StatusOK, `{"data":{"id":"c1","name":"Nightly"}}`)

	if _, err := c.CheckIns.Update(context.Background(), "Xk9mZp", "c1",
		CheckInParams{Name: "Nightly", GracePeriod: "5m"}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if got.body["grace_period"] != "5m" {
		t.Errorf("grace_period = %v", got.body["grace_period"])
	}
	if got.body["name"] != "Nightly" {
		t.Errorf("name = %v; the schema requires it on update", got.body["name"])
	}
	for _, absent := range []string{"schedule_type", "report_period", "cron_schedule"} {
		if _, present := got.body[absent]; present {
			t.Errorf("%q was sent despite being unset; it would blank the field", absent)
		}
	}
}

// The cron fields exist now, so a cron check-in is expressible.
func TestCheckInCreateSendsCronSchedule(t *testing.T) {
	c, got := captureWrite(t, http.StatusCreated, `{"data":{"id":"c1","name":"Nightly"}}`)

	_, err := c.CheckIns.Create(context.Background(), "Xk9mZp", CheckInParams{
		Name:         "Nightly",
		ScheduleType: "cron",
		CronSchedule: "0 3 * * *",
		// A Rails zone name, not an IANA identifier — the API rejects the latter.
		CronTimezone: "Central Time (US & Canada)",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	for key, want := range map[string]any{
		"schedule_type": "cron",
		"cron_schedule": "0 3 * * *",
		"cron_timezone": "Central Time (US & Canada)",
	} {
		if got.body[key] != want {
			t.Errorf("%s = %v, want %v", key, got.body[key], want)
		}
	}
}

func TestCheckInCreateSendsSpecifiedFields(t *testing.T) {
	c, got := captureWrite(t, http.StatusCreated, `{"data":{"id":"c1","name":"Nightly"}}`)

	_, err := c.CheckIns.Create(context.Background(), "Xk9mZp", CheckInParams{
		Name:         "Nightly",
		ScheduleType: "simple",
		ReportPeriod: "1d",
		GracePeriod:  "1h",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	for key, want := range map[string]any{
		"name": "Nightly", "schedule_type": "simple",
		"report_period": "1d", "grace_period": "1h",
	} {
		if got.body[key] != want {
			t.Errorf("%s = %v, want %v", key, got.body[key], want)
		}
	}
}

// A validation error must arrive typed, with its field details intact — this is
// the array branch of the details oneOf.
func TestWriteValidationErrorCarriesFieldDetails(t *testing.T) {
	c, _ := captureWrite(t, http.StatusUnprocessableEntity,
		`{"error":{"code":"validation_error","message":"Name can't be blank",
		  "details":[{"field":"name","message":"can't be blank"}]},
		  "meta":{"request_id":"req_v"}}`)

	_, err := c.Projects.Create(context.Background(), ProjectParams{})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}

	var apiErr *Error
	if !asError(err, &apiErr) {
		t.Fatal("not an *apiv3.Error")
	}
	if len(apiErr.FieldErrors) != 1 {
		t.Fatalf("FieldErrors = %v, want 1", apiErr.FieldErrors)
	}
	if apiErr.FieldErrors[0].Field != "name" {
		t.Errorf("field = %q, want name", apiErr.FieldErrors[0].Field)
	}
	// The rendered message should name the offending field.
	if !strings.Contains(apiErr.Error(), "name: can't be blank") {
		t.Errorf("Error() = %q", apiErr.Error())
	}
}

// A write refused for scope must still name the scope needed.
func TestWriteInsufficientScopeNamesScope(t *testing.T) {
	c, _ := captureWrite(t, http.StatusForbidden,
		`{"error":{"code":"insufficient_scope","message":"Insufficient scope",
		  "details":{"required_scope":"faults:write","token_scopes":["faults:read"]}}}`)

	err := c.Faults.Ignore(context.Background(), "Xk9mZp", []string{"f1"})
	var apiErr *Error
	if !asError(err, &apiErr) {
		t.Fatalf("err = %T, want *apiv3.Error", err)
	}
	if apiErr.RequiredScope() != "faults:write" {
		t.Errorf("RequiredScope() = %q", apiErr.RequiredScope())
	}
}

// The project write schema carries everything v2 accepted, so a caller can set
// the settings that previously had to be changed in the UI.
func TestProjectsUpdateSendsFullSettings(t *testing.T) {
	c, got := captureWrite(t, http.StatusOK,
		`{"data":{"id":"Xk9mZp","account_id":"Ab3kL9","name":"App","active":true}}`)

	purgeDays := 30
	disablePublicLinks := false
	userURL := "http://example.com/users/[user_id]"

	_, err := c.Projects.Update(context.Background(), "Xk9mZp", ProjectParams{
		Name:               "App",
		PurgeDays:          &purgeDays,
		DisablePublicLinks: &disablePublicLinks,
		UserUrl:            &userURL,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if got.body["purge_days"] != float64(30) {
		t.Errorf("purge_days = %v", got.body["purge_days"])
	}
	// False is a value, not an absence: it must reach the wire.
	if v, present := got.body["disable_public_links"]; !present || v != false {
		t.Errorf("disable_public_links = %v (present %v), want false sent", v, present)
	}
	if got.body["user_url"] != userURL {
		t.Errorf("user_url = %v", got.body["user_url"])
	}
	// Unset fields stay absent.
	if _, present := got.body["source_url"]; present {
		t.Error("source_url was sent unset")
	}
}

// An alarm can now be created with the query and trigger that make it fire.
func TestAlarmsCreateSendsQueryAndTrigger(t *testing.T) {
	c, got := captureWrite(t, http.StatusCreated, `{"data":{"id":"a1","name":"Spike"}}`)

	_, err := c.Alarms.Create(context.Background(), "Xk9mZp", AlarmParams{
		Name:             "Spike",
		Query:            "count() > 100",
		EvaluationPeriod: "5m",
		LookbackLag:      "1m",
		StreamIDs:        []string{"str_1"},
		Trigger:          &AlarmTrigger{Type: "threshold", Operator: ">", Value: 100},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if got.body["query"] != "count() > 100" {
		t.Errorf("query = %v", got.body["query"])
	}
	trigger, ok := got.body["trigger_config"].(map[string]any)
	if !ok {
		t.Fatalf("trigger_config = %v", got.body["trigger_config"])
	}
	if trigger["type"] != "threshold" {
		t.Errorf("trigger type = %v", trigger["type"])
	}
	config, ok := trigger["config"].(map[string]any)
	if !ok || config["operator"] != ">" || config["value"] != float64(100) {
		t.Errorf("trigger config = %v", trigger["config"])
	}
}

// A dashboard with no widgets must send an empty array: the field is required and
// a nil slice would serialise as null.
func TestDashboardsCreateSendsEmptyWidgetArray(t *testing.T) {
	c, got := captureWrite(t, http.StatusCreated, `{"data":{"id":"d1","title":"Ops"}}`)

	if _, err := c.Dashboards.Create(context.Background(), "Xk9mZp",
		DashboardParams{Title: "Ops"}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	widgets, present := got.body["widgets"]
	if !present {
		t.Fatal("widgets was not sent; the schema requires it")
	}
	if widgets == nil {
		t.Error("widgets was null; an empty array is what the schema wants")
	}
	if arr, ok := widgets.([]any); !ok || len(arr) != 0 {
		t.Errorf("widgets = %v, want []", widgets)
	}
}

// Widgets pass through as raw JSON, since the generated widget type is an
// anonymous struct a caller could not build.
func TestDashboardsCreatePassesWidgetsThrough(t *testing.T) {
	c, got := captureWrite(t, http.StatusCreated, `{"data":{"id":"d1","title":"Ops"}}`)

	_, err := c.Dashboards.Create(context.Background(), "Xk9mZp", DashboardParams{
		Title:   "Ops",
		Widgets: []byte(`[{"type":"errors","grid":{"x":0,"y":0,"w":6,"h":4}}]`),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	arr, ok := got.body["widgets"].([]any)
	if !ok || len(arr) != 1 {
		t.Fatalf("widgets = %v", got.body["widgets"])
	}
	widget, _ := arr[0].(map[string]any)
	if widget["type"] != "errors" {
		t.Errorf("widget = %v", widget)
	}
}

// Assignment is expressible now, through a dedicated endpoint.
func TestFaultsAssignAndUnassign(t *testing.T) {
	c, got := captureWrite(t, http.StatusNoContent, "")

	if err := c.Faults.Assign(context.Background(), "Xk9mZp", "f1", "usr_1"); err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if got.method != http.MethodPost {
		t.Errorf("method = %q, want POST", got.method)
	}
	if got.body["assignee_id"] != "usr_1" {
		t.Errorf("assignee_id = %v", got.body["assignee_id"])
	}

	c2, got2 := captureWrite(t, http.StatusNoContent, "")
	if err := c2.Faults.Unassign(context.Background(), "Xk9mZp", "f1"); err != nil {
		t.Fatalf("Unassign: %v", err)
	}
	if got2.method != http.MethodDelete {
		t.Errorf("unassign method = %q, want DELETE", got2.method)
	}
}

// A dashboard update replaces rather than merges, so omitting widgets would clear
// them. The client refuses instead of silently emptying the dashboard.
func TestDashboardsUpdateRefusesWithoutWidgets(t *testing.T) {
	c, _ := captureWrite(t, http.StatusOK, `{"data":{"id":"d1","title":"Ops"}}`)

	_, err := c.Dashboards.Update(context.Background(), "Xk9mZp", "d1",
		DashboardParams{Title: "Ops"})
	if !errors.Is(err, ErrReplacesDashboard) {
		t.Fatalf("err = %v, want ErrReplacesDashboard", err)
	}
}

// Create is different: a new dashboard legitimately has no widgets, and the schema
// requires the field, so an empty array is correct there.
func TestDashboardsCreateAllowsNoWidgets(t *testing.T) {
	c, got := captureWrite(t, http.StatusCreated, `{"data":{"id":"d1","title":"Ops"}}`)

	if _, err := c.Dashboards.Create(context.Background(), "Xk9mZp",
		DashboardParams{Title: "Ops"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if arr, ok := got.body["widgets"].([]any); !ok || len(arr) != 0 {
		t.Errorf("widgets = %v, want []", got.body["widgets"])
	}
}

// An alarm's description must be clearable, which needs absent and empty to be
// distinguishable.
func TestAlarmsUpdateCanClearDescription(t *testing.T) {
	c, got := captureWrite(t, http.StatusOK, `{"data":{"id":"a1","name":"Spike"}}`)

	empty := ""
	if _, err := c.Alarms.Update(context.Background(), "Xk9mZp", "a1",
		AlarmUpdateParams{Description: &empty}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	v, present := got.body["description"]
	if !present {
		t.Fatal("description was dropped; an empty value is how a caller clears it")
	}
	if v != "" {
		t.Errorf("description = %v, want empty", v)
	}
	// A nil name must stay absent rather than blanking the alarm's name.
	if _, present := got.body["name"]; present {
		t.Error("name was sent though it was not supplied")
	}
}

// The fault listing's filters must reach the query string.
func TestFaultsListSendsOrderAndTimeFilters(t *testing.T) {
	var query url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.Query()
		writeJSON(w, 0, `{"data":[]}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	at := time.Unix(1704067200, 500*1000*1000) // fractional seconds are significant

	_, err := c.Faults.ListAll(context.Background(), "Xk9mZp",
		OrderBy("frequent"), CreatedAfter(at), OccurredAfter(at), OccurredBefore(at))
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}

	if got := query.Get("order"); got != "frequent" {
		t.Errorf("order = %q, want frequent", got)
	}
	for _, field := range []string{"created_after", "occurred_after", "occurred_before"} {
		if got := query.Get(field); got != "1704067200.5" {
			t.Errorf("%s = %q, want 1704067200.5 with the fraction intact", field, got)
		}
	}
}

// The counts endpoint takes the same filters, and previously ignored all but q.
func TestFaultsSummarySendsTimeFilters(t *testing.T) {
	var query url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.Query()
		writeJSON(w, 0, `{"data":{"total":1}}`)
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	at := time.Unix(1704067200, 0)

	if _, err := c.Faults.Summary(context.Background(), "Xk9mZp",
		Search("is:unresolved"), OccurredAfter(at)); err != nil {
		t.Fatalf("Summary: %v", err)
	}

	if got := query.Get("q"); got != "is:unresolved" {
		t.Errorf("q = %q", got)
	}
	if got := query.Get("occurred_after"); got == "" {
		t.Error("occurred_after was not sent; the filter would be silently ignored")
	}
}
