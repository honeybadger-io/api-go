package apiv3

import (
	"context"
	"net/http"

	"github.com/honeybadger-io/api-go/internal/gen"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Site is an uptime-monitored URL in a project.
type Site = gen.Site

// SiteID identifies a site. Unlike other v3 ids it is a UUID, as the spec
// declares; parse one with uuid.Parse from github.com/google/uuid.
type SiteID = openapi_types.UUID

// SiteCreateParams are a new site's fields. Url is required.
type SiteCreateParams = gen.SiteCreateInput

// SiteUpdateParams are the fields a site update can change. Unset fields keep
// their values, and a null clears a nullable one.
type SiteUpdateParams = gen.SiteInput

// The site settings that take one of a fixed set of values. Use the spec's
// values directly: SiteFrequency is minutes between checks (1, 2, 5 or 15), and
// the rest are strings such as SiteLocation("us-east").
type (
	SiteFrequency     = gen.SiteFrequency
	SiteLocation      = gen.SiteLocation
	SiteMatchType     = gen.SiteMatchType
	SiteRequestMethod = gen.SiteRequestMethod
	SiteState         = gen.SiteState
)

// SitesService handles a project's uptime sites.
type SitesService struct {
	client *Client
}

// List returns one page of a project's sites.
func (s *SitesService) List(ctx context.Context, projectID string, opts ...Option) (*ListResponse[Site], error) {
	return s.list(ctx, projectID, resolve(opts))
}

// ListAll returns every site in the project, walking pagination.
func (s *SitesService) ListAll(ctx context.Context, projectID string, opts ...ListAllOption) ([]Site, error) {
	ro := resolveListAll(opts)
	return CollectPages(ctx, func(ctx context.Context, page int) (*ListResponse[Site], error) {
		ro.page = page
		return s.list(ctx, projectID, ro)
	})
}

func (s *SitesService) list(ctx context.Context, projectID string, ro requestOptions) (*ListResponse[Site], error) {
	params := &gen.ListSitesParams{}
	ro.applyOffset(&params.Page, &params.PerPage)

	return listOffset[Site](ctx, s.client, "listSites", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().ListSites(ctx, projectID, params)
	})
}

// Get returns one of a project's sites.
func (s *SitesService) Get(ctx context.Context, projectID string, siteID SiteID) (*Site, error) {
	return getOne[Site](ctx, s.client, "getSite", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().GetSite(ctx, projectID, siteID)
	})
}

// Create starts monitoring a URL.
func (s *SitesService) Create(ctx context.Context, projectID string, p SiteCreateParams) (*Site, error) {
	return getOne[Site](ctx, s.client, "createSite", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().CreateSite(ctx, projectID, p)
	})
}

// Update changes a site.
func (s *SitesService) Update(ctx context.Context, projectID string, siteID SiteID, p SiteUpdateParams) (*Site, error) {
	return getOne[Site](ctx, s.client, "updateSite", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().UpdateSite(ctx, projectID, siteID, p)
	})
}

// Delete stops monitoring a site and removes it.
func (s *SitesService) Delete(ctx context.Context, projectID string, siteID SiteID) error {
	return noContent(ctx, s.client, "deleteSite", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().DeleteSite(ctx, projectID, siteID)
	})
}

// Outage is one period a site was down, with the status and reason from the
// check that found it.
type Outage = gen.Outage

// UptimeCheck is one check of a site from one location.
type UptimeCheck = gen.UptimeCheck

// ListOutages returns one page of a site's outages, newest first. Limit sizes
// the page; follow the response's older link, or use ListAllOutages, for more.
func (s *SitesService) ListOutages(ctx context.Context, projectID string, siteID SiteID, opts ...Option) (*ListResponse[Outage], error) {
	return s.listOutages(ctx, projectID, siteID, resolve(opts))
}

// ListAllOutages returns every outage for a site, walking from newest to oldest.
func (s *SitesService) ListAllOutages(ctx context.Context, projectID string, siteID SiteID, opts ...ListAllOption) ([]Outage, error) {
	ro := resolveListAll(opts)
	return CollectTimeSeries(ctx, func(ctx context.Context, link string) (*ListResponse[Outage], error) {
		if link != "" {
			return followTimeSeries[Outage](ctx, s.client, link)
		}
		return s.listOutages(ctx, projectID, siteID, ro)
	})
}

func (s *SitesService) listOutages(ctx context.Context, projectID string, siteID SiteID, ro requestOptions) (*ListResponse[Outage], error) {
	params := &gen.ListOutagesParams{}
	if ro.limit > 0 {
		limit := gen.Limit(ro.limit)
		params.Limit = &limit
	}
	return listTimeSeries[Outage](ctx, s.client, "listOutages", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().ListOutages(ctx, projectID, siteID, params)
	})
}

// ListUptimeChecks returns one page of a site's checks, newest first. Limit
// sizes the page; follow the response's older link, or use
// ListAllUptimeChecks, for more.
func (s *SitesService) ListUptimeChecks(ctx context.Context, projectID string, siteID SiteID, opts ...Option) (*ListResponse[UptimeCheck], error) {
	return s.listUptimeChecks(ctx, projectID, siteID, resolve(opts))
}

// ListAllUptimeChecks returns every check for a site, walking from newest to
// oldest. A busy site has many; bound the walk with Limit.
func (s *SitesService) ListAllUptimeChecks(ctx context.Context, projectID string, siteID SiteID, opts ...ListAllOption) ([]UptimeCheck, error) {
	ro := resolveListAll(opts)
	return CollectTimeSeries(ctx, func(ctx context.Context, link string) (*ListResponse[UptimeCheck], error) {
		if link != "" {
			return followTimeSeries[UptimeCheck](ctx, s.client, link)
		}
		return s.listUptimeChecks(ctx, projectID, siteID, ro)
	})
}

func (s *SitesService) listUptimeChecks(ctx context.Context, projectID string, siteID SiteID, ro requestOptions) (*ListResponse[UptimeCheck], error) {
	params := &gen.ListUptimeChecksParams{}
	if ro.limit > 0 {
		limit := gen.Limit(ro.limit)
		params.Limit = &limit
	}
	return listTimeSeries[UptimeCheck](ctx, s.client, "listUptimeChecks", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().ListUptimeChecks(ctx, projectID, siteID, params)
	})
}
