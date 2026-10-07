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
func (s *AlarmsService) List(ctx context.Context, projectID string) (*ListResponse[Alarm], error) {
	return listOffset[Alarm](ctx, s.client, "listAlarms", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().ListAlarms(ctx, projectID)
	})
}

// Get returns a single alarm.
func (s *AlarmsService) Get(ctx context.Context, projectID, alarmID string) (*Alarm, error) {
	return getOne[Alarm](ctx, s.client, "getAlarm", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().GetAlarm(ctx, projectID, alarmID)
	})
}

// AlarmHistoryEntry is one evaluation of an alarm: when it ran, what the query
// returned, and whether the alarm was in alarm or ok afterwards.
type AlarmHistoryEntry = gen.AlarmHistoryEntry

// ListHistory returns one page of an alarm's evaluations, newest first.
//
// Page-numbered like the other v3 lists, but with a fixed page size of 25: the
// endpoint takes no per_page, so a per-page value passed to Page is ignored.
func (s *AlarmsService) ListHistory(ctx context.Context, projectID, alarmID string, opts ...Option) (*ListResponse[AlarmHistoryEntry], error) {
	return s.listHistory(ctx, projectID, alarmID, resolve(opts))
}

// ListAllHistory returns every evaluation of an alarm, following links.next.
func (s *AlarmsService) ListAllHistory(ctx context.Context, projectID, alarmID string, opts ...ListAllOption) ([]AlarmHistoryEntry, error) {
	ro := resolveListAll(opts)
	return CollectPages(ctx, func(ctx context.Context, page int) (*ListResponse[AlarmHistoryEntry], error) {
		ro.page = page
		return s.listHistory(ctx, projectID, alarmID, ro)
	})
}

func (s *AlarmsService) listHistory(ctx context.Context, projectID, alarmID string, ro requestOptions) (*ListResponse[AlarmHistoryEntry], error) {
	params := &gen.ListAlarmHistoryParams{}
	if ro.page > 0 {
		page := gen.Page(ro.page)
		params.Page = &page
	}
	return listOffset[AlarmHistoryEntry](ctx, s.client, "listAlarmHistory", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().ListAlarmHistory(ctx, projectID, alarmID, params)
	})
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

	return listOffset[Dashboard](ctx, s.client, "listDashboards", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().ListDashboards(ctx, projectID, params)
	})
}

// Get returns a single dashboard.
func (s *DashboardsService) Get(ctx context.Context, projectID, dashboardID string) (*Dashboard, error) {
	return getOne[Dashboard](ctx, s.client, "getDashboard", func(ctx context.Context) (*http.Response, error) {
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

	return listOffset[Integration](ctx, s.client, "listIntegrations", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().ListIntegrations(ctx, projectID, params)
	})
}

// Get returns a single integration by its public ID.
func (s *IntegrationsService) Get(ctx context.Context, projectID, integrationID string) (*Integration, error) {
	return getOne[Integration](ctx, s.client, "getIntegration", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().GetIntegration(ctx, projectID, integrationID)
	})
}

// IntegrationCreateParams are a new integration's fields. Type picks the
// integration and Config holds that type's settings, the same keys a GET returns
// under config; the spec's IntegrationConfig<Type> schemas list them, and the API
// refuses a missing or unknown one with 422.
type IntegrationCreateParams = gen.IntegrationCreateInput

// IntegrationType names an integration kind: WebHook, Email, Slack, and so on.
type IntegrationType = gen.IntegrationCreateInputType

// IntegrationEvent names an event an integration can notify on.
type IntegrationEvent = gen.IntegrationEvent

// IntegrationUpdateParams are the fields an update can change. Unset fields are
// left as they are and a null clears one; Config carries only the settings
// being changed.
//
// SiteIds and CheckInIds are the exception: null follows every site or
// check-in, including ones added later, and [] follows none. Leaving them out
// on create follows every one, as the UI does; on update it keeps the stored
// value. A read returns null for "every".
type IntegrationUpdateParams = gen.IntegrationUpdateInput

// IntegrationFilter limits one event to the errors matching Query. Filters on an
// integration are an ordered list, replaced whole on update.
type IntegrationFilter = gen.IntegrationFilter

// IntegrationFilterEvent names the event a filter applies to: any
// IntegrationEvent value, or IntegrationFilterAll.
type IntegrationFilterEvent = gen.IntegrationFilterEvent

// IntegrationFilterAll applies a filter to every event.
const IntegrationFilterAll IntegrationFilterEvent = gen.IntegrationFilterEventAll

// Create makes a new integration.
//
// OAuth types (Slack, GitHub and the like) are created unconnected: a person
// connects one at the integration's Links.Web. Active can be true from the start;
// the integration sends nothing until it's connected, then starts on its own.
func (s *IntegrationsService) Create(ctx context.Context, projectID string, p IntegrationCreateParams) (*Integration, error) {
	return getOne[Integration](ctx, s.client, "createIntegration", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().CreateIntegration(ctx, projectID, p)
	})
}

// Update changes an integration's settings.
func (s *IntegrationsService) Update(ctx context.Context, projectID, integrationID string, p IntegrationUpdateParams) (*Integration, error) {
	return getOne[Integration](ctx, s.client, "updateIntegration", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().UpdateIntegration(ctx, projectID, integrationID, p)
	})
}

// Delete removes an integration.
func (s *IntegrationsService) Delete(ctx context.Context, projectID, integrationID string) error {
	return noContent(ctx, s.client, "deleteIntegration", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().DeleteIntegration(ctx, projectID, integrationID)
	})
}
