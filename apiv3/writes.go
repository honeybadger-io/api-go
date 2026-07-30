package apiv3

import (
	"context"
	"net/http"

	"github.com/honeybadger-io/api-go/internal/gen"
)

// Creates and updates for the remaining resources.
//
// Creates take their required fields as arguments; updates take pointers under
// the hood so an omitted field is left unchanged rather than blanked.
//
// Each takes only the fields the spec declares for its request body. Several of
// those bodies are still narrower than v2's — a dashboard write, for instance,
// declares `name` where v2 accepted title, default_ts, and widgets. The methods
// here follow the spec rather than v2, because a field absent from the schema is
// dropped by the generated request type and would be silently discarded on the
// way out. openapi/README.md tracks which bodies are still incomplete.

// Create makes a new project in the account.
func (s *ProjectsService) Create(ctx context.Context, name string, opts ...Option) (*Project, error) {
	ro := resolve(opts)
	body := gen.CreateProjectJSONRequestBody{Name: name}
	return getOne[Project](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().CreateProject(ctx, s.client.accountID(ro.accountID), body)
	})
}

// Update renames a project.
func (s *ProjectsService) Update(ctx context.Context, projectID, name string, opts ...Option) (*Project, error) {
	ro := resolve(opts)
	body := gen.UpdateProjectJSONRequestBody{Name: name}
	return getOne[Project](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().UpdateProject(ctx, s.client.accountID(ro.accountID), projectID, body)
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

// Create makes a new alarm.
//
// The spec declares only name for this body. An alarm's query, evaluation
// period, trigger configuration, lookback lag, and streams are not yet in the
// request schema, so an alarm created here will need configuring elsewhere.
func (s *AlarmsService) Create(ctx context.Context, projectID, name string, opts ...Option) (*Alarm, error) {
	ro := resolve(opts)
	body := gen.CreateAlarmJSONRequestBody{Name: name}
	return getOne[Alarm](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().CreateAlarm(ctx, s.client.accountID(ro.accountID), projectID, body)
	})
}

// Update renames an alarm. Same caveat as Create: name is all the spec declares.
func (s *AlarmsService) Update(ctx context.Context, projectID, alarmID, name string, opts ...Option) (*Alarm, error) {
	ro := resolve(opts)
	body := gen.UpdateAlarmJSONRequestBody{Name: &name}
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

// Create makes a new dashboard.
//
// Title is the field on both sides now: the spec previously wrote it as name
// while reading it as title, and settled on title.
func (s *DashboardsService) Create(ctx context.Context, projectID, title string, opts ...Option) (*Dashboard, error) {
	ro := resolve(opts)
	body := gen.CreateDashboardJSONRequestBody{Title: title, Widgets: nil}
	return getOne[Dashboard](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().CreateDashboard(ctx, s.client.accountID(ro.accountID), projectID, body)
	})
}

// Update renames a dashboard.
//
// Widgets are not yet exposed here: the generated widget type is a deep anonymous
// struct, so a caller could not construct one. Renaming works, which is what MCP
// needs today.
func (s *DashboardsService) Update(ctx context.Context, projectID, dashboardID, title string, opts ...Option) (*Dashboard, error) {
	ro := resolve(opts)
	body := gen.UpdateDashboardJSONRequestBody{Title: title, Widgets: nil}
	return getOne[Dashboard](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().UpdateDashboard(ctx, s.client.accountID(ro.accountID), projectID, dashboardID, body)
	})
}

// Delete removes a dashboard.
func (s *DashboardsService) Delete(ctx context.Context, projectID, dashboardID string, opts ...Option) error {
	ro := resolve(opts)
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().DeleteDashboard(ctx, s.client.accountID(ro.accountID), projectID, dashboardID)
	})
}
