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
	body := gen.UpdateProjectJSONRequestBody{Name: &name}
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
// The spec declares these four. v2 also accepted a slug, a cron schedule, and a
// timezone, which are not in the v3 request schema yet — see openapi/README.md.
type CheckInParams struct {
	Name         string
	ScheduleType string
	ReportPeriod string
	GracePeriod  string
}

// Create makes a new check-in.
func (s *CheckInsService) Create(ctx context.Context, projectID string, p CheckInParams, opts ...Option) (*CheckIn, error) {
	ro := resolve(opts)
	body := gen.CreateCheckInJSONRequestBody{Name: p.Name}
	if p.ScheduleType != "" {
		body.ScheduleType = &p.ScheduleType
	}
	if p.ReportPeriod != "" {
		body.ReportPeriod = &p.ReportPeriod
	}
	if p.GracePeriod != "" {
		body.GracePeriod = &p.GracePeriod
	}

	return getOne[CheckIn](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().CreateCheckIn(ctx, s.client.accountID(ro.accountID), projectID, body)
	})
}

// Update changes a check-in. Empty fields are omitted, leaving them unchanged.
func (s *CheckInsService) Update(ctx context.Context, projectID, checkInID string, p CheckInParams, opts ...Option) (*CheckIn, error) {
	ro := resolve(opts)
	body := gen.UpdateCheckInJSONRequestBody{}
	if p.Name != "" {
		body.Name = &p.Name
	}
	if p.ScheduleType != "" {
		body.ScheduleType = &p.ScheduleType
	}
	if p.ReportPeriod != "" {
		body.ReportPeriod = &p.ReportPeriod
	}
	if p.GracePeriod != "" {
		body.GracePeriod = &p.GracePeriod
	}

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
// The spec declares only name. v2 accepted title, default_ts, and widgets, so a
// dashboard created here has no widgets on it.
func (s *DashboardsService) Create(ctx context.Context, projectID, name string, opts ...Option) (*Dashboard, error) {
	ro := resolve(opts)
	body := gen.CreateDashboardJSONRequestBody{Name: name}
	return getOne[Dashboard](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().CreateDashboard(ctx, s.client.accountID(ro.accountID), projectID, body)
	})
}

// Update renames a dashboard. Same caveat as Create.
func (s *DashboardsService) Update(ctx context.Context, projectID, dashboardID, name string, opts ...Option) (*Dashboard, error) {
	ro := resolve(opts)
	body := gen.UpdateDashboardJSONRequestBody{Name: &name}
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
