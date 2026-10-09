package apiv3

import (
	"context"
	"net/http"

	"github.com/honeybadger-io/api-go/internal/gen"
)

// StatusPage is a public page showing the state of an account's sites and
// check-ins.
type StatusPage = gen.StatusPage

// StatusPageSite is a site as a status page shows it: its id, display name and
// description. Read the site's live state from the Sites service.
type StatusPageSite = gen.StatusPageSite

// StatusPageCheckIn is a check-in as a status page shows it: its id, display name
// and description. Read the check-in's live state from the CheckIns service.
type StatusPageCheckIn = gen.StatusPageCheckIn

// StatusPageCreateParams are a new status page's fields. Name is required.
type StatusPageCreateParams = gen.StatusPageCreateInput

// StatusPageUpdateParams are the fields a status page update can change. Unset
// fields keep their values and a null clears one. A Sites or CheckIns list
// replaces the page's list, in page order.
type StatusPageUpdateParams = gen.StatusPageInput

// StatusPageSiteInput places a site on a status page, optionally with its own
// display name and description.
type StatusPageSiteInput = gen.StatusPageSiteInput

// StatusPageCheckInInput places a check-in on a status page, optionally with its
// own display name and description.
type StatusPageCheckInInput = gen.StatusPageCheckInInput

// StatusPageFeatures holds a status page's custom captions, home link and CSS.
type StatusPageFeatures = gen.StatusPageFeatures

// StatusPagesService handles the account's status pages. They belong to the
// account the credential resolves to, not to a project.
type StatusPagesService struct {
	client *Client
}

// List returns one page of the account's status pages.
func (s *StatusPagesService) List(ctx context.Context, opts ...Option) (*ListResponse[StatusPage], error) {
	return s.list(ctx, resolve(opts))
}

// ListAll returns every status page in the account, walking pagination.
func (s *StatusPagesService) ListAll(ctx context.Context, opts ...ListAllOption) ([]StatusPage, error) {
	ro := resolveListAll(opts)
	return CollectPages(ctx, func(ctx context.Context, page int) (*ListResponse[StatusPage], error) {
		ro.page = page
		return s.list(ctx, ro)
	})
}

func (s *StatusPagesService) list(ctx context.Context, ro requestOptions) (*ListResponse[StatusPage], error) {
	params := &gen.ListStatusPagesParams{}
	ro.applyOffset(&params.Page, &params.PerPage)

	return listOffset[StatusPage](ctx, s.client, "listStatusPages", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().ListStatusPages(ctx, params)
	})
}

// Get returns a status page.
func (s *StatusPagesService) Get(ctx context.Context, statusPageID string) (*StatusPage, error) {
	return getOne[StatusPage](ctx, s.client, "getStatusPage", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().GetStatusPage(ctx, statusPageID)
	})
}

// Create makes a new status page.
func (s *StatusPagesService) Create(ctx context.Context, p StatusPageCreateParams) (*StatusPage, error) {
	return getOne[StatusPage](ctx, s.client, "createStatusPage", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().CreateStatusPage(ctx, p)
	})
}

// Update changes a status page.
func (s *StatusPagesService) Update(ctx context.Context, statusPageID string, p StatusPageUpdateParams) (*StatusPage, error) {
	return getOne[StatusPage](ctx, s.client, "updateStatusPage", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().UpdateStatusPage(ctx, statusPageID, p)
	})
}

// Delete removes a status page.
func (s *StatusPagesService) Delete(ctx context.Context, statusPageID string) error {
	return noContent(ctx, s.client, "deleteStatusPage", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().DeleteStatusPage(ctx, statusPageID)
	})
}
