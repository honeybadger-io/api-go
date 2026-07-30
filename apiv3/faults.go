package apiv3

import (
	"context"
	"net/http"

	"github.com/honeybadger-io/api-go/internal/gen"
)

// Fault is an error group.
type Fault = gen.Fault

// Notice is a single occurrence of a fault.
//
// Note a notice is addressed by its token UUID rather than the short opaque
// public id every other v3 resource uses, so Notice.Id is a uuid.UUID.
type Notice = gen.Notice

// FaultsService handles the faults resource and its notices.
type FaultsService struct {
	client *Client
}

// List returns one page of faults for a project. Use Search to filter, Page to
// select which page, and InAccount to address a specific account.
func (s *FaultsService) List(ctx context.Context, projectID string, opts ...Option) (*ListResponse[Fault], error) {
	return s.list(ctx, projectID, resolve(opts))
}

// ListAll returns every fault matching the options, walking pagination.
func (s *FaultsService) ListAll(ctx context.Context, projectID string, opts ...ListAllOption) ([]Fault, error) {
	ro := resolveListAll(opts)
	return CollectPages(ctx, func(ctx context.Context, page int) (*ListResponse[Fault], error) {
		ro.page = page
		return s.list(ctx, projectID, ro)
	})
}

func (s *FaultsService) list(ctx context.Context, projectID string, ro requestOptions) (*ListResponse[Fault], error) {
	params := &gen.ListFaultsParams{}
	ro.applyOffset(&params.Page, &params.PerPage)
	if ro.query != "" {
		params.Q = &ro.query
	}
	if ro.order != "" {
		order := gen.ListFaultsParamsOrder(ro.order)
		params.Order = &order
	}
	if ro.createdAfter != 0 {
		params.CreatedAfter = &ro.createdAfter
	}
	if ro.occurredAfter != 0 {
		params.OccurredAfter = &ro.occurredAfter
	}
	if ro.occurredBefore != 0 {
		params.OccurredBefore = &ro.occurredBefore
	}

	return listOffset[Fault](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().ListFaults(ctx, s.client.accountID(ro.accountID), projectID, params)
	})
}

// Get returns a single fault.
func (s *FaultsService) Get(ctx context.Context, projectID, faultID string, opts ...Option) (*Fault, error) {
	ro := resolve(opts)
	return getOne[Fault](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().GetFault(ctx, s.client.accountID(ro.accountID), projectID, faultID)
	})
}

// AffectedUser is one person a fault has reached, with how often it reached them.
type AffectedUser struct {
	// User is the affected user as the API reports it. Untyped because the spec
	// does not describe the member, and the app renders whatever the user object
	// serialises to.
	User map[string]any `json:"user"`

	// Count is how many of the fault's notices affected this user.
	Count int `json:"count"`
}

// AffectedUsers returns the users a fault has affected.
//
// The spec declares the data member as an object, but the endpoint renders an
// array of {user, count} pairs — a real server answers with the array, and
// decoding it as an object failed outright. Modelled on what the app sends, not
// on what the document claims; see openapi/GAPS.md.
//
// Search accepts the same filter syntax as the fault listing.
func (s *FaultsService) AffectedUsers(ctx context.Context, projectID, faultID string, opts ...Option) ([]AffectedUser, error) {
	ro := resolve(opts)
	params := &gen.ListFaultAffectedUsersParams{}
	if ro.query != "" {
		q := gen.SearchFilter(ro.query)
		params.Q = &q
	}

	data, err := getOne[[]AffectedUser](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().ListFaultAffectedUsers(ctx, s.client.accountID(ro.accountID), projectID, faultID, params)
	})
	if err != nil {
		return nil, err
	}
	return *data, nil
}

// ListNotices returns one page of a fault's notices, newest first. Use Limit to
// size the page, and Before or After to position within the collection.
func (s *FaultsService) ListNotices(ctx context.Context, projectID, faultID string, opts ...Option) (*ListResponse[Notice], error) {
	return s.listNotices(ctx, projectID, faultID, resolve(opts))
}

// ListAllNotices returns every notice for a fault, walking from newest to oldest.
//
// After the first page it follows links.older rather than re-deriving a cursor,
// which is what the spec instructs and the only mechanism that also works for
// collections that page on a timestamp.
func (s *FaultsService) ListAllNotices(ctx context.Context, projectID, faultID string, opts ...ListAllOption) ([]Notice, error) {
	ro := resolveListAll(opts)
	return CollectTimeSeries(ctx, func(ctx context.Context, link string) (*ListResponse[Notice], error) {
		if link != "" {
			return followTimeSeries[Notice](ctx, s.client, link)
		}
		return s.listNotices(ctx, projectID, faultID, ro)
	})
}

func (s *FaultsService) listNotices(ctx context.Context, projectID, faultID string, ro requestOptions) (*ListResponse[Notice], error) {
	params := &gen.ListNoticesParams{}
	ro.applyTimeSeries(&params.Limit, &params.Before, &params.After)

	return listTimeSeries[Notice](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().ListNotices(ctx, s.client.accountID(ro.accountID), projectID, faultID, params)
	})
}
