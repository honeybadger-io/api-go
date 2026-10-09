package apiv3

import (
	"context"
	"net/http"

	"github.com/honeybadger-io/api-go/internal/gen"
)

// Creates and updates.
//
// Input types are aliases of the generated request models, so a field the spec
// adds shows up on the next re-vendor with no change here. The generated package
// is internal, so the named types and enum constants a caller needs to fill one in
// (CheckInScheduleType, AlarmTriggerConfig, OperatorGt and so on) are re-exported
// alongside them. If an input ever needs a hand-written wrapper, that's a sign the
// spec's shape wants fixing instead.
//
// Unset fields are omitted rather than sent empty, so an update touches only what
// it was given. A pointer field is unset when nil. A field the API can clear is a
// nullable.Nullable (github.com/oapi-codegen/nullable): its zero value is unset,
// NewNullableWithValue sets it, and NewNullNullable sends null, which clears it.

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
func (s *ProjectsService) Create(ctx context.Context, p ProjectCreateParams) (*Project, error) {
	return getOne[Project](ctx, s.client, "createProject", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().CreateProject(ctx, p)
	})
}

// Update changes a project. Unset fields are omitted and left unchanged.
func (s *ProjectsService) Update(ctx context.Context, projectID string, p ProjectParams) (*Project, error) {
	return getOne[Project](ctx, s.client, "updateProject", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().UpdateProject(ctx, projectID, p)
	})
}

// Delete removes a project.
func (s *ProjectsService) Delete(ctx context.Context, projectID string) error {
	return noContent(ctx, s.client, "deleteProject", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().DeleteProject(ctx, projectID)
	})
}

// CheckInCreateParams are a new check-in's fields. ReportPeriod is required for a
// simple schedule and CronSchedule for a cron one. Name is optional: an unnamed
// check-in shows its ID.
//
// CronTimezone is a Rails/ActiveSupport zone name rather than an IANA identifier
// — "Central Time (US & Canada)", not "America/Chicago", which the API rejects.
type CheckInCreateParams = gen.CheckInCreateInput

// CheckInUpdateParams are the fields a check-in update can change. Unset fields
// keep their values, and a null clears one.
type CheckInUpdateParams = gen.CheckInInput

// CheckInScheduleType is how a check-in expects its reports.
type CheckInScheduleType = gen.CheckInScheduleType

const (
	// ScheduleSimple expects a report every ReportPeriod.
	ScheduleSimple CheckInScheduleType = gen.Simple
	// ScheduleCron expects reports on CronSchedule.
	ScheduleCron CheckInScheduleType = gen.Cron
)

// Create makes a new check-in.
func (s *CheckInsService) Create(ctx context.Context, projectID string, p CheckInCreateParams) (*CheckIn, error) {
	return getOne[CheckIn](ctx, s.client, "createCheckIn", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().CreateCheckIn(ctx, projectID, p)
	})
}

// Update changes a check-in, leaving whatever p omits as it is.
func (s *CheckInsService) Update(ctx context.Context, projectID, checkInID string, p CheckInUpdateParams) (*CheckIn, error) {
	return getOne[CheckIn](ctx, s.client, "updateCheckIn", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().UpdateCheckIn(ctx, projectID, checkInID, p)
	})
}

// Delete removes a check-in.
func (s *CheckInsService) Delete(ctx context.Context, projectID, checkInID string) error {
	return noContent(ctx, s.client, "deleteCheckIn", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().DeleteCheckIn(ctx, projectID, checkInID)
	})
}

// AlarmTriggerConfig is what turns an alarm on. It's sent whole: an update's
// trigger replaces the stored one rather than merging into it.
type AlarmTriggerConfig = gen.AlarmTriggerConfig

// AlarmTriggerCondition is the comparison a trigger makes, such as gt 10.
type AlarmTriggerCondition = gen.AlarmTriggerCondition

// AlarmTriggerType is a trigger kind.
type AlarmTriggerType = gen.AlarmTriggerConfigType

// AlarmTriggerOperator is a named comparison. The API names them rather than
// using symbols.
type AlarmTriggerOperator = gen.AlarmTriggerConditionOperator

const (
	// TriggerResultCount compares the number of results the query returns.
	TriggerResultCount AlarmTriggerType = gen.AlertResultCount

	OperatorGt  AlarmTriggerOperator = gen.Gt
	OperatorGte AlarmTriggerOperator = gen.Gte
	OperatorLt  AlarmTriggerOperator = gen.Lt
	OperatorLte AlarmTriggerOperator = gen.Lte
	OperatorEq  AlarmTriggerOperator = gen.Eq
	OperatorNeq AlarmTriggerOperator = gen.Neq
)

// AlarmCreateParams are a new alarm's fields. Name, Query, EvaluationPeriod,
// LookbackLag and TriggerConfig are required. Nil StreamIds runs the query
// against every current stream on the project; a list must name at least one
// stream, and an empty one is refused with 422.
type AlarmCreateParams = gen.AlarmCreateInput

// AlarmUpdateParams are the fields an alarm update can change. Unset fields keep
// their values and a null Description clears it. StreamIds left unset keeps the
// stored streams, a list replaces them (at least one; an empty list is a 422),
// and null resets them to every current stream on the project.
type AlarmUpdateParams = gen.AlarmUpdateInput

// Create makes a new alarm.
func (s *AlarmsService) Create(ctx context.Context, projectID string, p AlarmCreateParams) (*Alarm, error) {
	return getOne[Alarm](ctx, s.client, "createAlarm", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().CreateAlarm(ctx, projectID, p)
	})
}

// Update changes an alarm, including its query, window and trigger, without
// losing its history the way deleting and recreating it would.
func (s *AlarmsService) Update(ctx context.Context, projectID, alarmID string, p AlarmUpdateParams) (*Alarm, error) {
	return getOne[Alarm](ctx, s.client, "updateAlarm", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().UpdateAlarm(ctx, projectID, alarmID, p)
	})
}

// Delete removes an alarm.
func (s *AlarmsService) Delete(ctx context.Context, projectID, alarmID string) error {
	return noContent(ctx, s.client, "deleteAlarm", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().DeleteAlarm(ctx, projectID, alarmID)
	})
}

// DashboardCreateParams are a new dashboard's fields. Only Title is required;
// Widgets defaults to none.
type DashboardCreateParams = gen.DashboardInput

// DashboardUpdateParams are the fields a dashboard update can change. Unset fields
// keep their values, and a null DefaultTs clears it. A Widgets list replaces the dashboard's widgets: leave one
// out to remove it, and keep each widget's Id to keep its identity.
type DashboardUpdateParams = gen.DashboardUpdateInput

// DashboardWidget is a widget as written. Config holds the settings for the
// widget's Type; the spec's DashboardWidgetConfig<Type> schemas list them.
type DashboardWidget = gen.DashboardWidgetInput

// DashboardWidgetType names a widget kind: insights_vis, alarms, errors, and so on.
type DashboardWidgetType = gen.DashboardWidgetType

// Create makes a new dashboard.
func (s *DashboardsService) Create(ctx context.Context, projectID string, p DashboardCreateParams) (*Dashboard, error) {
	return getOne[Dashboard](ctx, s.client, "createDashboard", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().CreateDashboard(ctx, projectID, p)
	})
}

// Update changes a dashboard, leaving whatever p omits as it is.
func (s *DashboardsService) Update(ctx context.Context, projectID, dashboardID string, p DashboardUpdateParams) (*Dashboard, error) {
	return getOne[Dashboard](ctx, s.client, "updateDashboard", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().UpdateDashboard(ctx, projectID, dashboardID, p)
	})
}

// Delete removes a dashboard.
func (s *DashboardsService) Delete(ctx context.Context, projectID, dashboardID string) error {
	return noContent(ctx, s.client, "deleteDashboard", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().DeleteDashboard(ctx, projectID, dashboardID)
	})
}
