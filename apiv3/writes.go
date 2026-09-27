package apiv3

import (
	"context"
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
// it was given.

// ProjectCreateParams are the fields of a new project. Name is required.
type ProjectCreateParams = gen.ProjectCreateInput

// ProjectParams are the writable fields of an existing project. Every field is
// optional; an update changes only what it sets.
type ProjectParams = gen.ProjectInput

// FaultParams are the writable fields of a fault: resolved, ignored, tags, and
// the assignee. AssigneeId is nullable — an explicit null unassigns, while
// leaving it unspecified changes nothing.
type FaultParams = gen.FaultInput

// Create makes a new project. Name is the only required field.
func (s *ProjectsService) Create(ctx context.Context, p ProjectCreateParams, opts ...Option) (*Project, error) {
	return getOne[Project](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().CreateProject(ctx, p)
	})
}

// Update changes a project. Unset fields are omitted and left unchanged.
func (s *ProjectsService) Update(ctx context.Context, projectID string, p ProjectParams, opts ...Option) (*Project, error) {
	return getOne[Project](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().UpdateProject(ctx, projectID, p)
	})
}

// Delete removes a project.
func (s *ProjectsService) Delete(ctx context.Context, projectID string, opts ...Option) error {
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().DeleteProject(ctx, projectID)
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

// setOptional points each target at its value, leaving empty values absent so an
// update touches only what it was given.
func setOptional(fields map[**string]string) {
	for field, value := range fields {
		if value != "" {
			v := value
			*field = &v
		}
	}
}

func (p CheckInParams) toCreate() gen.CheckInCreateInput {
	body := gen.CheckInCreateInput{Name: p.Name}
	if p.ScheduleType != "" {
		st := gen.CheckInScheduleType(p.ScheduleType)
		body.ScheduleType = &st
	}
	setOptional(map[**string]string{
		&body.ReportPeriod: p.ReportPeriod,
		&body.GracePeriod:  p.GracePeriod,
		&body.CronSchedule: p.CronSchedule,
		&body.CronTimezone: p.CronTimezone,
		&body.Slug:         p.Slug,
	})
	return body
}

func (p CheckInParams) toUpdate() gen.CheckInInput {
	var body gen.CheckInInput
	if p.ScheduleType != "" {
		st := gen.CheckInScheduleType(p.ScheduleType)
		body.ScheduleType = &st
	}
	setOptional(map[**string]string{
		&body.Name:         p.Name,
		&body.ReportPeriod: p.ReportPeriod,
		&body.GracePeriod:  p.GracePeriod,
		&body.CronSchedule: p.CronSchedule,
		&body.CronTimezone: p.CronTimezone,
		&body.Slug:         p.Slug,
	})
	return body
}

// Create makes a new check-in.
func (s *CheckInsService) Create(ctx context.Context, projectID string, p CheckInParams, opts ...Option) (*CheckIn, error) {
	body := p.toCreate()

	return getOne[CheckIn](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().CreateCheckIn(ctx, projectID, body)
	})
}

// Update changes a check-in. Empty fields are omitted and left unchanged, so a
// caller changing just the grace period sends just the grace period.
func (s *CheckInsService) Update(ctx context.Context, projectID, checkInID string, p CheckInParams, opts ...Option) (*CheckIn, error) {
	body := p.toUpdate()

	return getOne[CheckIn](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().UpdateCheckIn(ctx, projectID, checkInID, body)
	})
}

// Delete removes a check-in.
func (s *CheckInsService) Delete(ctx context.Context, projectID, checkInID string, opts ...Option) error {
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().DeleteCheckIn(ctx, projectID, checkInID)
	})
}

// AlarmTrigger is what turns an alarm on.
//
// The vocabulary is the API's, not arithmetic: Type is a trigger kind such as
// "alert_result_count", and Operator is a name such as "gt" or "lt" rather than a
// symbol. Verified against a running server — an alarm created with
// {alert_result_count, gt, 10} stores and reports exactly that.
type AlarmTrigger struct {
	// Type is the trigger kind, such as "alert_result_count".
	Type string

	// Operator and Value configure the comparison — "gt" and 10, say.
	Operator string
	Value    float64
}

// config renders the trigger in the shape create, update and the alarm response
// all share.
func (t AlarmTrigger) config() *gen.AlarmTriggerConfig {
	cfg := &gen.AlarmTriggerConfig{Type: t.Type}
	if t.Operator != "" || t.Value != 0 {
		op, val := t.Operator, t.Value
		cfg.Config = &gen.AlarmTriggerCondition{Operator: &op, Value: &val}
	}
	return cfg
}

// AlarmParams are the writable fields of an alarm.
//
// Hand-written rather than aliased so the trigger reads as three flat fields
// instead of a nested config object. The generated trigger type is now named and
// constructible, so this is an ergonomic choice rather than a workaround.
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
	// on the project. An id that isn't one of its streams is refused with 422.
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
		body.TriggerConfig = p.Trigger.config()
	}
	return body
}

// Create makes a new alarm. Name and Query are required; an alarm with no
// trigger is created but never fires.
func (s *AlarmsService) Create(ctx context.Context, projectID string, p AlarmParams, opts ...Option) (*Alarm, error) {
	body := p.toCreate()
	return getOne[Alarm](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().CreateAlarm(ctx, projectID, body)
	})
}

// AlarmUpdateParams are the fields an alarm update can change. Nil fields are
// omitted and left unchanged.
//
// Pointers rather than values so absent and empty are distinguishable: pointing
// Description at "" clears it, which a plain string could not express.
type AlarmUpdateParams struct {
	Name        *string
	Description *string

	// Query is the BadgerQL query evaluated on each check.
	Query *string

	// EvaluationPeriod and LookbackLag are compact durations: "10m", "1h".
	EvaluationPeriod *string
	LookbackLag      *string

	// StreamIDs replaces the streams the query runs against.
	StreamIDs *[]string

	// Trigger replaces the whole trigger configuration.
	Trigger *AlarmTrigger
}

// Update changes an alarm, including its query, window and trigger, without
// losing its history the way deleting and recreating it would.
func (s *AlarmsService) Update(ctx context.Context, projectID, alarmID string, p AlarmUpdateParams, opts ...Option) (*Alarm, error) {
	body := gen.AlarmUpdateInput{
		Name:             p.Name,
		Description:      p.Description,
		Query:            p.Query,
		EvaluationPeriod: p.EvaluationPeriod,
		LookbackLag:      p.LookbackLag,
		StreamIds:        p.StreamIDs,
	}
	if p.Trigger != nil {
		body.TriggerConfig = p.Trigger.config()
	}

	return getOne[Alarm](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().UpdateAlarm(ctx, projectID, alarmID, body)
	})
}

// Delete removes an alarm.
func (s *AlarmsService) Delete(ctx context.Context, projectID, alarmID string, opts ...Option) error {
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().DeleteAlarm(ctx, projectID, alarmID)
	})
}

// DashboardCreateParams are a new dashboard's fields. Only Title is required;
// Widgets defaults to none.
type DashboardCreateParams = gen.DashboardInput

// DashboardUpdateParams are the fields a dashboard update can change. Nil fields
// keep their values. A Widgets list replaces the dashboard's widgets: leave one
// out to remove it, and keep each widget's Id to keep its identity.
type DashboardUpdateParams = gen.DashboardUpdateInput

// DashboardWidget is a widget as written. Config holds the settings for the
// widget's Type; the spec's DashboardWidgetConfig<Type> schemas list them.
type DashboardWidget = gen.DashboardWidgetInput

// DashboardWidgetType names a widget kind: insights_vis, alarms, errors, and so on.
type DashboardWidgetType = gen.DashboardWidgetInputType

// Create makes a new dashboard.
func (s *DashboardsService) Create(ctx context.Context, projectID string, p DashboardCreateParams, opts ...Option) (*Dashboard, error) {
	return getOne[Dashboard](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().CreateDashboard(ctx, projectID, p)
	})
}

// Update changes a dashboard, leaving whatever p omits as it is.
func (s *DashboardsService) Update(ctx context.Context, projectID, dashboardID string, p DashboardUpdateParams, opts ...Option) (*Dashboard, error) {
	return getOne[Dashboard](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().UpdateDashboard(ctx, projectID, dashboardID, p)
	})
}

// Delete removes a dashboard.
func (s *DashboardsService) Delete(ctx context.Context, projectID, dashboardID string, opts ...Option) error {
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().DeleteDashboard(ctx, projectID, dashboardID)
	})
}
