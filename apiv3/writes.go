package apiv3

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/honeybadger-io/api-go/internal/gen"
)

// Creates and updates.
//
// Input types are aliased from the generated models where a caller can actually
// construct one, and hand-written where they cannot. That line matters: an input
// whose fields are plain types or the public nullable package is usable directly,
// but one containing an enum or an anonymous struct is not, because those types
// live in an internal package. AlarmParams and CheckInParams exist for that
// reason; ProjectParams and FaultParams do not need to.
//
// Unset fields are omitted rather than sent empty, so an update touches only what
// it was given — with one exception the API imposes, noted on CheckIns.Update.

// ProjectParams are the writable fields of a project.
type ProjectParams = gen.ProjectInput

// FaultParams are the writable fields of a fault: resolved, ignored, tags, and
// the assignee. AssigneeId is nullable — an explicit null unassigns, while
// leaving it unspecified changes nothing.
type FaultParams = gen.FaultInput

// Create makes a new project. Name is the only required field.
func (s *ProjectsService) Create(ctx context.Context, p ProjectParams, opts ...Option) (*Project, error) {
	ro := resolve(opts)
	return getOne[Project](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().CreateProject(ctx, s.client.accountID(ro.accountID), p)
	})
}

// Update changes a project.
//
// Name is required even when only another field is changing: the update body is
// the same schema as create. Every other unset field is omitted.
func (s *ProjectsService) Update(ctx context.Context, projectID string, p ProjectParams, opts ...Option) (*Project, error) {
	ro := resolve(opts)
	return getOne[Project](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().UpdateProject(ctx, s.client.accountID(ro.accountID), projectID, p)
	})
}

// Delete removes a project.
func (s *ProjectsService) Delete(ctx context.Context, projectID string, opts ...Option) error {
	ro := resolve(opts)
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().DeleteProject(ctx, s.client.accountID(ro.accountID), projectID)
	})
}

// CheckInParams are the writable fields of a check-in.
//
// The v3 schema now carries everything v2 accepted, including the cron fields
// that define a cron check-in.
type CheckInParams struct {
	// Name is required on create.
	Name string

	// ScheduleType is "simple" or "cron". A simple check-in expects a report every
	// ReportPeriod; a cron one expects them on CronSchedule.
	ScheduleType string

	// ReportPeriod is required for a simple schedule: a count and a unit
	// ("10 minutes", "1 day") or HH:MM:SS.
	ReportPeriod string

	// GracePeriod is how long after the expected time before the check-in counts
	// as missing, in the same format as ReportPeriod.
	GracePeriod string

	// CronSchedule is required when ScheduleType is "cron".
	CronSchedule string

	// CronTimezone is a Rails/ActiveSupport zone name rather than an IANA
	// identifier — "Central Time (US & Canada)", not "America/Chicago", which the
	// API rejects. Required when ScheduleType is "cron".
	CronTimezone string

	// Slug is the short identifier in the check-in's reporting URL. Generated from
	// the name when empty.
	Slug string
}

// apply fills a generated request body, leaving unset fields absent so an update
// touches only what it was given.
func (p CheckInParams) apply(body *gen.CheckInInput) {
	body.Name = p.Name
	if p.ScheduleType != "" {
		st := gen.CheckInInputScheduleType(p.ScheduleType)
		body.ScheduleType = &st
	}
	for field, value := range map[**string]string{
		&body.ReportPeriod: p.ReportPeriod,
		&body.GracePeriod:  p.GracePeriod,
		&body.CronSchedule: p.CronSchedule,
		&body.CronTimezone: p.CronTimezone,
		&body.Slug:         p.Slug,
	} {
		if value != "" {
			v := value
			*field = &v
		}
	}
}

// Create makes a new check-in.
func (s *CheckInsService) Create(ctx context.Context, projectID string, p CheckInParams, opts ...Option) (*CheckIn, error) {
	ro := resolve(opts)
	var body gen.CheckInInput
	p.apply(&body)

	return getOne[CheckIn](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().CreateCheckIn(ctx, s.client.accountID(ro.accountID), projectID, body)
	})
}

// Update changes a check-in.
//
// Name is required even when only another field is changing: the API's update body
// is the same schema as create, with name mandatory. A caller changing just the
// grace period still has to supply the current name.
//
// Every other empty field is omitted, so an update touches only what it was given.
func (s *CheckInsService) Update(ctx context.Context, projectID, checkInID string, p CheckInParams, opts ...Option) (*CheckIn, error) {
	ro := resolve(opts)
	var body gen.CheckInInput
	p.apply(&body)

	return getOne[CheckIn](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().UpdateCheckIn(ctx, s.client.accountID(ro.accountID), projectID, checkInID, body)
	})
}

// Delete removes a check-in.
func (s *CheckInsService) Delete(ctx context.Context, projectID, checkInID string, opts ...Option) error {
	ro := resolve(opts)
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().DeleteCheckIn(ctx, s.client.accountID(ro.accountID), projectID, checkInID)
	})
}

// AlarmTrigger is what turns an alarm on.
type AlarmTrigger struct {
	// Type is the trigger kind, such as "threshold".
	Type string

	// Operator and Value configure a threshold trigger — ">" and 100, say.
	Operator string
	Value    float32
}

// AlarmParams are the writable fields of an alarm.
//
// Hand-written rather than aliased: the generated trigger configuration is an
// anonymous struct inside an internal package, so a caller could not build one.
type AlarmParams struct {
	// Name and Query are required on create.
	Name  string
	Query string

	// EvaluationPeriod is the window each evaluation covers.
	EvaluationPeriod string

	// LookbackLag is how far behind now that window ends, allowing for ingestion
	// delay.
	LookbackLag string

	Description string

	// StreamIDs are the streams the query runs against. Empty means every stream
	// on the project. Ids not belonging to it are dropped by the API.
	StreamIDs []string

	// Trigger is optional; without one the alarm is created but never fires.
	Trigger *AlarmTrigger
}

func (p AlarmParams) toCreate() gen.AlarmCreateInput {
	body := gen.AlarmCreateInput{Name: p.Name, Query: p.Query}
	for field, value := range map[**string]string{
		&body.EvaluationPeriod: p.EvaluationPeriod,
		&body.LookbackLag:      p.LookbackLag,
		&body.Description:      p.Description,
	} {
		if value != "" {
			v := value
			*field = &v
		}
	}
	if len(p.StreamIDs) > 0 {
		body.StreamIds = &p.StreamIDs
	}
	if p.Trigger != nil {
		body.TriggerConfig = &struct {
			Config *struct {
				Operator *string  `json:"operator,omitempty"`
				Value    *float32 `json:"value,omitempty"`
			} `json:"config,omitempty"`
			Type string `json:"type"`
		}{Type: p.Trigger.Type}
		if p.Trigger.Operator != "" || p.Trigger.Value != 0 {
			op, val := p.Trigger.Operator, p.Trigger.Value
			body.TriggerConfig.Config = &struct {
				Operator *string  `json:"operator,omitempty"`
				Value    *float32 `json:"value,omitempty"`
			}{Operator: &op, Value: &val}
		}
	}
	return body
}

// Create makes a new alarm. Name and Query are required; an alarm with no
// trigger is created but never fires.
func (s *AlarmsService) Create(ctx context.Context, projectID string, p AlarmParams, opts ...Option) (*Alarm, error) {
	ro := resolve(opts)
	body := p.toCreate()
	return getOne[Alarm](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().CreateAlarm(ctx, s.client.accountID(ro.accountID), projectID, body)
	})
}

// Update changes an alarm's name or description.
//
// Those are the only fields the update schema declares — unlike create, which
// takes the query, trigger and evaluation settings. Changing an alarm's
// behaviour is therefore not yet possible through the API.
func (s *AlarmsService) Update(ctx context.Context, projectID, alarmID string, name, description string, opts ...Option) (*Alarm, error) {
	ro := resolve(opts)
	var body gen.AlarmUpdateInput
	if name != "" {
		body.Name = &name
	}
	if description != "" {
		body.Description = &description
	}

	return getOne[Alarm](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().UpdateAlarm(ctx, s.client.accountID(ro.accountID), projectID, alarmID, body)
	})
}

// Delete removes an alarm.
func (s *AlarmsService) Delete(ctx context.Context, projectID, alarmID string, opts ...Option) error {
	ro := resolve(opts)
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().DeleteAlarm(ctx, s.client.accountID(ro.accountID), projectID, alarmID)
	})
}

// DashboardParams are the writable fields of a dashboard.
//
// Widgets is raw JSON rather than a typed slice. The generated widget type is a
// deeply nested anonymous struct inside an internal package, so a caller could
// not construct one — passing the array through untouched is the only way to
// support widgets at all until the spec names those schemas. The dashboards
// reference topic documents the shape.
type DashboardParams struct {
	// Title is required.
	Title string

	// DefaultTs is the dashboard's default time range, such as "P1D" or "week".
	DefaultTs string

	// Widgets is a JSON array of widget objects. Nil sends an empty array, which
	// is what the schema requires for a dashboard with no widgets.
	Widgets json.RawMessage
}

// body builds the request payload.
//
// Marshalled here rather than through the generated struct because widgets must
// reach the API as an empty array when there are none: the field is required and
// not omitempty, so a nil slice would serialise as null.
func (p DashboardParams) body() ([]byte, error) {
	payload := struct {
		Title     string          `json:"title"`
		DefaultTs *string         `json:"default_ts,omitempty"`
		Widgets   json.RawMessage `json:"widgets"`
	}{Title: p.Title, Widgets: p.Widgets}

	if p.DefaultTs != "" {
		payload.DefaultTs = &p.DefaultTs
	}
	if len(payload.Widgets) == 0 {
		payload.Widgets = json.RawMessage("[]")
	}
	return json.Marshal(payload)
}

// Create makes a new dashboard.
//
// Title is the field on both sides now: the spec previously wrote it as name
// while reading it as title, and settled on title.
func (s *DashboardsService) Create(ctx context.Context, projectID string, p DashboardParams, opts ...Option) (*Dashboard, error) {
	ro := resolve(opts)
	raw, err := p.body()
	if err != nil {
		return nil, err
	}

	return getOne[Dashboard](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().CreateDashboardWithBody(ctx, s.client.accountID(ro.accountID), projectID,
			"application/json", bytes.NewReader(raw))
	})
}

// Update changes a dashboard. Title is required, as it is on create.
func (s *DashboardsService) Update(ctx context.Context, projectID, dashboardID string, p DashboardParams, opts ...Option) (*Dashboard, error) {
	ro := resolve(opts)
	raw, err := p.body()
	if err != nil {
		return nil, err
	}

	return getOne[Dashboard](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().UpdateDashboardWithBody(ctx, s.client.accountID(ro.accountID), projectID,
			dashboardID, "application/json", bytes.NewReader(raw))
	})
}

// Delete removes a dashboard.
func (s *DashboardsService) Delete(ctx context.Context, projectID, dashboardID string, opts ...Option) error {
	ro := resolve(opts)
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().DeleteDashboard(ctx, s.client.accountID(ro.accountID), projectID, dashboardID)
	})
}
