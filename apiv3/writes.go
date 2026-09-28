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
// Nil fields are omitted rather than sent empty, so an update touches only what it
// was given.

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

// CheckInCreateParams are a new check-in's fields. Name is required, and so is
// ReportPeriod for a simple schedule or CronSchedule for a cron one.
//
// CronTimezone is a Rails/ActiveSupport zone name rather than an IANA identifier
// — "Central Time (US & Canada)", not "America/Chicago", which the API rejects.
type CheckInCreateParams = gen.CheckInCreateInput

// CheckInUpdateParams are the fields a check-in update can change. Nil fields keep
// their values.
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
func (s *CheckInsService) Create(ctx context.Context, projectID string, p CheckInCreateParams, opts ...Option) (*CheckIn, error) {
	return getOne[CheckIn](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().CreateCheckIn(ctx, projectID, p)
	})
}

// Update changes a check-in, leaving whatever p omits as it is.
func (s *CheckInsService) Update(ctx context.Context, projectID, checkInID string, p CheckInUpdateParams, opts ...Option) (*CheckIn, error) {
	return getOne[CheckIn](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().UpdateCheckIn(ctx, projectID, checkInID, p)
	})
}

// Delete removes a check-in.
func (s *CheckInsService) Delete(ctx context.Context, projectID, checkInID string, opts ...Option) error {
	return noContent(ctx, s.client, func() (*http.Response, error) {
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

// AlarmCreateParams are a new alarm's fields. Name and Query are required; an
// alarm with no TriggerConfig is created but never fires. Nil StreamIds runs the
// query against every stream on the project, while an empty list means none.
type AlarmCreateParams = gen.AlarmCreateInput

// AlarmUpdateParams are the fields an alarm update can change. Nil fields keep
// their values: pointing Description at "" clears it, and StreamIds distinguishes
// keeping the streams (nil) from none ([]).
type AlarmUpdateParams = gen.AlarmUpdateInput

// Create makes a new alarm.
func (s *AlarmsService) Create(ctx context.Context, projectID string, p AlarmCreateParams, opts ...Option) (*Alarm, error) {
	return getOne[Alarm](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().CreateAlarm(ctx, projectID, p)
	})
}

// Update changes an alarm, including its query, window and trigger, without
// losing its history the way deleting and recreating it would.
func (s *AlarmsService) Update(ctx context.Context, projectID, alarmID string, p AlarmUpdateParams, opts ...Option) (*Alarm, error) {
	return getOne[Alarm](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().UpdateAlarm(ctx, projectID, alarmID, p)
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
