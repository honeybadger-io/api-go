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

// A 204 carries no body, so it must not be treated as a malformed envelope.
func TestDeleteAcceptsNoContent(t *testing.T) {
	c, got := captureWrite(t, http.StatusNoContent, "")

	if err := c.Faults.Delete(context.Background(), "Xk9mZp", "1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got.method != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", got.method)
	}
}

func TestPauseRecordingSendsBody(t *testing.T) {
	c, got := captureWrite(t, http.StatusOK,
		`{"data":{"id":"1","project_id":"Xk9mZp","recording_paused_until":"2026-09-27T00:00:00Z"}}`)
	fault, err := c.Faults.PauseRecording(context.Background(), "Xk9mZp", "1", PauseDay)
	if err != nil {
		t.Fatalf("pause: %v", err)
	}
	// The fault comes back saying until when recording is paused.
	if until, err := fault.RecordingPausedUntil.Get(); err != nil || until.IsZero() {
		t.Errorf("recording_paused_until = %v, %v", until, err)
	}
	wantPath := "/v3/projects/Xk9mZp/faults/1/pause_recording"
	if got.path != wantPath {
		t.Errorf("path = %q, want %q", got.path, wantPath)
	}
	if got.body == nil {
		t.Fatal("body = nil, want {time: day}")
	}
	if time, ok := got.body["time"]; !ok || time != "day" {
		t.Errorf("body = %v, want time=day", got.body)
	}
}

func TestResumeRecordingSendsNoBody(t *testing.T) {
	c, got := captureWrite(t, http.StatusOK, `{"data":{"id":"1","project_id":"Xk9mZp","recording_paused_until":null}}`)
	if _, err := c.Faults.ResumeRecording(context.Background(), "Xk9mZp", "1"); err != nil {
		t.Fatalf("resume: %v", err)
	}
	wantPath := "/v3/projects/Xk9mZp/faults/1/resume_recording"
	if got.path != wantPath {
		t.Errorf("path = %q, want %q", got.path, wantPath)
	}
	if got.body != nil {
		t.Errorf("body = %v, want none", got.body)
	}
}

func TestProjectsCreateSendsRequiredName(t *testing.T) {
	c, got := captureWrite(t, http.StatusCreated,
		`{"data":{"id":"Xk9mZp","account_id":"Ab3kL9","name":"New App","active":true}}`)

	p, err := c.Projects.Create(context.Background(), ProjectCreateParams{Name: "New App"})
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
// than blanked. Name included: the update schema no longer requires it.
func TestCheckInUpdateOmitsUnsetFields(t *testing.T) {
	c, got := captureWrite(t, http.StatusOK, `{"data":{"id":"c1","name":"Nightly"}}`)

	grace := "5m"
	if _, err := c.CheckIns.Update(context.Background(), "Xk9mZp", "c1",
		CheckInUpdateParams{GracePeriod: &grace}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if got.body["grace_period"] != "5m" {
		t.Errorf("grace_period = %v", got.body["grace_period"])
	}
	for _, absent := range []string{"name", "schedule_type", "report_period", "cron_schedule"} {
		if _, present := got.body[absent]; present {
			t.Errorf("%q was sent despite being unset; it would blank the field", absent)
		}
	}
}

// The cron fields exist now, so a cron check-in is expressible.
func TestCheckInCreateSendsCronSchedule(t *testing.T) {
	c, got := captureWrite(t, http.StatusCreated, `{"data":{"id":"c1","name":"Nightly"}}`)

	cron, schedule := "0 3 * * *", ScheduleCron
	// A Rails zone name, not an IANA identifier — the API rejects the latter.
	zone := "Central Time (US & Canada)"
	_, err := c.CheckIns.Create(context.Background(), "Xk9mZp", CheckInCreateParams{
		Name: "Nightly", ScheduleType: &schedule, CronSchedule: &cron, CronTimezone: &zone,
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

	schedule, period, grace := ScheduleSimple, "1d", "1h"
	_, err := c.CheckIns.Create(context.Background(), "Xk9mZp", CheckInCreateParams{
		Name: "Nightly", ScheduleType: &schedule, ReportPeriod: &period, GracePeriod: &grace,
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

	_, err := c.Projects.Create(context.Background(), ProjectCreateParams{})
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

	_, err := c.Faults.Ignore(context.Background(), "Xk9mZp", SelectFaults("1"))
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
	// Unset fields stay absent, name included: an update is partial.
	for _, absent := range []string{"source_url", "name"} {
		if _, present := got.body[absent]; present {
			t.Errorf("%s was sent unset", absent)
		}
	}
}

// An alarm can now be created with the query and trigger that make it fire.
func TestAlarmsCreateSendsQueryAndTrigger(t *testing.T) {
	c, got := captureWrite(t, http.StatusCreated, `{"data":{"id":"a1","name":"Spike"}}`)

	period, lag := "5m", "1m"
	streams := []string{"str_1"}
	_, err := c.Alarms.Create(context.Background(), "Xk9mZp", AlarmCreateParams{
		Name:             "Spike",
		Query:            "count() > 100",
		EvaluationPeriod: &period,
		LookbackLag:      &lag,
		StreamIds:        &streams,
		TriggerConfig: &AlarmTriggerConfig{Type: TriggerResultCount,
			Config: AlarmTriggerCondition{Operator: OperatorGt, Value: 100}},
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
	if trigger["type"] != "alert_result_count" {
		t.Errorf("trigger type = %v", trigger["type"])
	}
	config, ok := trigger["config"].(map[string]any)
	if !ok || config["operator"] != "gt" || config["value"] != float64(100) {
		t.Errorf("trigger config = %v", trigger["config"])
	}
}

// A widget's config is the settings for its type, and every key survives — not
// just the insights ones.
func TestDashboardsCreateSendsWidgetConfig(t *testing.T) {
	c, got := captureWrite(t, http.StatusCreated, `{"data":{"id":"d1","title":"Ops"}}`)

	widgets := []DashboardWidget{{
		Type:   "alarms",
		Config: &map[string]interface{}{"limit": 5, "filter_state": "triggered"},
	}}
	if _, err := c.Dashboards.Create(context.Background(), "Xk9mZp",
		DashboardCreateParams{Title: "Ops", Widgets: &widgets}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	arr, ok := got.body["widgets"].([]any)
	if !ok || len(arr) != 1 {
		t.Fatalf("widgets = %v", got.body["widgets"])
	}
	widget, _ := arr[0].(map[string]any)
	config, _ := widget["config"].(map[string]any)
	if widget["type"] != "alarms" || config["limit"] != float64(5) || config["filter_state"] != "triggered" {
		t.Errorf("widget = %v", widget)
	}
}

// An update merges: a rename sends only the title, leaving the widgets alone.
func TestDashboardsUpdateSendsOnlyWhatChanged(t *testing.T) {
	c, got := captureWrite(t, http.StatusOK, `{"data":{"id":"d1","title":"Renamed"}}`)

	title := "Renamed"
	if _, err := c.Dashboards.Update(context.Background(), "Xk9mZp", "d1",
		DashboardUpdateParams{Title: &title}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.body["title"] != "Renamed" {
		t.Errorf("title = %v", got.body["title"])
	}
	for _, absent := range []string{"widgets", "default_ts"} {
		if _, present := got.body[absent]; present {
			t.Errorf("%s was sent though it was not supplied", absent)
		}
	}
}

// Assignment goes through a dedicated endpoint, which answers with the fault as
// it now stands.
func TestFaultsAssignAndUnassign(t *testing.T) {
	c, got := captureWrite(t, http.StatusOK,
		`{"data":{"id":"1","project_id":"Xk9mZp","assignee":{"id":"usr_1","email":"a@example.com"}}}`)

	fault, err := c.Faults.Assign(context.Background(), "Xk9mZp", "1", "usr_1")
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if got.method != http.MethodPost {
		t.Errorf("method = %q, want POST", got.method)
	}
	if got.body["assignee_id"] != "usr_1" {
		t.Errorf("assignee_id = %v", got.body["assignee_id"])
	}
	if a, err := fault.Assignee.Get(); err != nil || a.Id == nil || *a.Id != "usr_1" {
		t.Errorf("returned assignee = %+v, %v", a, err)
	}

	c2, got2 := captureWrite(t, http.StatusOK,
		`{"data":{"id":"1","project_id":"Xk9mZp","assignee":null}}`)
	fault, err = c2.Faults.Unassign(context.Background(), "Xk9mZp", "1")
	if err != nil {
		t.Fatalf("Unassign: %v", err)
	}
	if got2.method != http.MethodDelete {
		t.Errorf("unassign method = %q, want DELETE", got2.method)
	}
	if !fault.Assignee.IsNull() {
		t.Errorf("returned assignee = %v, want null", fault.Assignee)
	}
}

// A comment comes back as created, with its id.
func TestAddCommentReturnsTheComment(t *testing.T) {
	c, got := captureWrite(t, http.StatusCreated,
		`{"data":{"id":"cmt_1","fault_id":"1","body":"looking into it","created_at":"2026-09-26T00:00:00Z"}}`)

	comment, err := c.Faults.AddComment(context.Background(), "Xk9mZp", "1", "looking into it")
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	if got.body["body"] != "looking into it" {
		t.Errorf("sent body = %v", got.body)
	}
	if comment.Id != "cmt_1" || comment.FaultId != "1" {
		t.Errorf("comment = %+v", comment)
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

// An alarm's query and evaluation window can change in place, so it keeps its
// history; only the fields supplied are sent.
func TestAlarmsUpdateChangesBehaviour(t *testing.T) {
	c, got := captureWrite(t, http.StatusOK, `{"data":{"id":"a1","name":"Spike"}}`)

	query, period := "filter level::str == \"error\"", "15m"
	streams := []string{"s1"}
	if _, err := c.Alarms.Update(context.Background(), "Xk9mZp", "a1", AlarmUpdateParams{
		Query: &query, EvaluationPeriod: &period, StreamIds: &streams,
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if got.body["query"] != query || got.body["evaluation_period"] != period {
		t.Errorf("body = %v", got.body)
	}
	if ids, ok := got.body["stream_ids"].([]any); !ok || len(ids) != 1 || ids[0] != "s1" {
		t.Errorf("stream_ids = %v", got.body["stream_ids"])
	}
	for _, absent := range []string{"name", "description", "lookback_lag", "trigger_config"} {
		if _, present := got.body[absent]; present {
			t.Errorf("%s was sent though it was not supplied", absent)
		}
	}
}

// An update can change the trigger, and the value keeps its precision.
func TestAlarmsUpdateSendsTrigger(t *testing.T) {
	c, got := captureWrite(t, http.StatusOK, `{"data":{"id":"a1","name":"Spike"}}`)

	if _, err := c.Alarms.Update(context.Background(), "Xk9mZp", "a1", AlarmUpdateParams{
		TriggerConfig: &AlarmTriggerConfig{Type: TriggerResultCount,
			Config: AlarmTriggerCondition{Operator: OperatorGte, Value: 0.5}},
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	trigger, ok := got.body["trigger_config"].(map[string]any)
	if !ok || trigger["type"] != "alert_result_count" {
		t.Fatalf("trigger_config = %v", got.body["trigger_config"])
	}
	config, ok := trigger["config"].(map[string]any)
	if !ok || config["operator"] != "gte" || config["value"] != 0.5 {
		t.Errorf("trigger config = %v", trigger["config"])
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

// Common fields sit at the top level and the type's settings under config, the
// shape a GET returns.
func TestIntegrationsCreateNestsSettingsUnderConfig(t *testing.T) {
	c, got := captureWrite(t, http.StatusCreated,
		`{"data":{"id":"i1","project_id":"Xk9mZp","type":"WebHook","active":true,"links":{"web":"https://app/x"}}}`)

	events := []IntegrationEvent{"occurred", "resolved"}
	if _, err := c.Integrations.Create(context.Background(), "Xk9mZp", IntegrationCreateParams{
		Type:   "WebHook",
		Events: &events,
		Config: &map[string]interface{}{"url": "https://example.com/hook", "label": "Deploys"},
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if got.body["type"] != "WebHook" {
		t.Errorf("type = %v", got.body["type"])
	}
	if evs, ok := got.body["events"].([]any); !ok || len(evs) != 2 {
		t.Errorf("events = %v", got.body["events"])
	}
	config, ok := got.body["config"].(map[string]any)
	if !ok || config["url"] != "https://example.com/hook" || config["label"] != "Deploys" {
		t.Errorf("config = %v", got.body["config"])
	}
	if _, flat := got.body["url"]; flat {
		t.Error("url was sent at the top level; settings belong under config")
	}
}

// Nil StreamIds runs the query against every stream, while an empty list means
// none; the two must reach the API differently.
func TestAlarmsCreateDistinguishesNoStreamsFromEvery(t *testing.T) {
	c, got := captureWrite(t, http.StatusCreated, `{"data":{"id":"a1","name":"Spike"}}`)
	if _, err := c.Alarms.Create(context.Background(), "Xk9mZp",
		AlarmCreateParams{Name: "Spike", Query: "q"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, present := got.body["stream_ids"]; present {
		t.Error("stream_ids was sent though it was nil")
	}

	c2, got2 := captureWrite(t, http.StatusCreated, `{"data":{"id":"a1","name":"Spike"}}`)
	none := []string{}
	if _, err := c2.Alarms.Create(context.Background(), "Xk9mZp",
		AlarmCreateParams{Name: "Spike", Query: "q", StreamIds: &none}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if ids, ok := got2.body["stream_ids"].([]any); !ok || len(ids) != 0 {
		t.Errorf("stream_ids = %v, want []", got2.body["stream_ids"])
	}
}
