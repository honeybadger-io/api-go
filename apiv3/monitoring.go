package apiv3

import (
	"context"
	"net/http"

	"github.com/honeybadger-io/api-go/internal/gen"
)

// Alarm is an Insights alarm.
type Alarm = gen.Alarm

// Dashboard is an Insights dashboard.
type Dashboard = gen.Dashboard

// AlarmsService handles the alarms resource.
type AlarmsService struct {
	client *Client
}

// List returns every alarm for a project.
//
// There is no ListAll counterpart because this endpoint is not paginated: it
// declares no page parameters and returns no pagination object, so one call is
// the whole collection.
func (s *AlarmsService) List(ctx context.Context, projectID string, opts ...Option) (*ListResponse[Alarm], error) {
	return listOffset[Alarm](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().ListAlarms(ctx, projectID)
	})
}

// Get returns a single alarm.
func (s *AlarmsService) Get(ctx context.Context, projectID, alarmID string, opts ...Option) (*Alarm, error) {
	return getOne[Alarm](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().GetAlarm(ctx, projectID, alarmID)
	})
}

// AlarmHistoryEntry is one state change of an alarm.
//
// Untyped because the endpoint passes the query service's rows through
// unchanged, the same arrangement as an Insights query result.
type AlarmHistoryEntry = map[string]any

// ListHistory returns one page of an alarm's state changes.
//
// This endpoint's pagination object is the query service's own — page and
// total_pages only, without the per_page and total_count the rest of v3 reports —
// so it is exposed as a plain page rather than through ListResponse.Pagination.
func (s *AlarmsService) ListHistory(ctx context.Context, projectID, alarmID string, opts ...Option) ([]AlarmHistoryEntry, error) {
	ro := resolve(opts)
	// Page only: this endpoint takes no per_page, another consequence of it
	// passing the query service's paging through rather than using v3's.
	params := &gen.ListAlarmHistoryParams{}
	if ro.page > 0 {
		page := gen.Page(ro.page)
		params.Page = &page
	}

	resp, err := listOffset[AlarmHistoryEntry](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().ListAlarmHistory(ctx, projectID, alarmID, params)
	})
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// DashboardsService handles the dashboards resource.
type DashboardsService struct {
	client *Client
}

// List returns one page of a project's dashboards.
func (s *DashboardsService) List(ctx context.Context, projectID string, opts ...Option) (*ListResponse[Dashboard], error) {
	return s.list(ctx, projectID, resolve(opts))
}

// ListAll returns every dashboard for a project, walking pagination.
func (s *DashboardsService) ListAll(ctx context.Context, projectID string, opts ...ListAllOption) ([]Dashboard, error) {
	ro := resolveListAll(opts)
	return CollectPages(ctx, func(ctx context.Context, page int) (*ListResponse[Dashboard], error) {
		ro.page = page
		return s.list(ctx, projectID, ro)
	})
}

func (s *DashboardsService) list(ctx context.Context, projectID string, ro requestOptions) (*ListResponse[Dashboard], error) {
	params := &gen.ListDashboardsParams{}
	ro.applyOffset(&params.Page, &params.PerPage)

	return listOffset[Dashboard](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().ListDashboards(ctx, projectID, params)
	})
}

// Get returns a single dashboard.
func (s *DashboardsService) Get(ctx context.Context, projectID, dashboardID string, opts ...Option) (*Dashboard, error) {
	return getOne[Dashboard](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().GetDashboard(ctx, projectID, dashboardID)
	})
}

// Integration is a notification integration — a Slack hook, a webhook, an email
// destination, PagerDuty, and so on.
type Integration = gen.Integration

// IntegrationsService handles notification integrations.
type IntegrationsService struct {
	client *Client
}

// List returns one page of a project's integrations.
func (s *IntegrationsService) List(ctx context.Context, projectID string, opts ...Option) (*ListResponse[Integration], error) {
	return s.list(ctx, projectID, resolve(opts))
}

// ListAll returns every integration for the project, walking pagination.
func (s *IntegrationsService) ListAll(ctx context.Context, projectID string, opts ...ListAllOption) ([]Integration, error) {
	ro := resolveListAll(opts)
	return CollectPages(ctx, func(ctx context.Context, page int) (*ListResponse[Integration], error) {
		ro.page = page
		return s.list(ctx, projectID, ro)
	})
}

func (s *IntegrationsService) list(ctx context.Context, projectID string, ro requestOptions) (*ListResponse[Integration], error) {
	params := &gen.ListIntegrationsParams{}
	ro.applyOffset(&params.Page, &params.PerPage)

	return listOffset[Integration](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().ListIntegrations(ctx, projectID, params)
	})
}

// Get returns a single integration by its public ID.
func (s *IntegrationsService) Get(ctx context.Context, projectID, integrationID string, opts ...Option) (*Integration, error) {
	return getOne[Integration](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().GetIntegration(ctx, projectID, integrationID)
	})
}

// IntegrationParams are the writable fields of an integration.
type IntegrationParams = gen.IntegrationInput

// Create makes a new integration.
//
// Type is required on create and determines which config fields are valid.
// OAuth integration types (Slack, GitHub, etc.) cannot be created via the API.
func (s *IntegrationsService) Create(ctx context.Context, projectID string, p IntegrationParams, opts ...Option) (*Integration, error) {
	return getOne[Integration](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().CreateIntegration(ctx, projectID, gen.CreateIntegrationJSONRequestBody(p))
	})
}

// Update changes an integration's settings.
func (s *IntegrationsService) Update(ctx context.Context, projectID, integrationID string, p IntegrationParams, opts ...Option) (*Integration, error) {
	return getOne[Integration](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().UpdateIntegration(ctx, projectID, integrationID, gen.UpdateIntegrationJSONRequestBody(p))
	})
}

// Delete removes an integration.
func (s *IntegrationsService) Delete(ctx context.Context, projectID, integrationID string, opts ...Option) error {
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().DeleteIntegration(ctx, projectID, integrationID)
	})
}
