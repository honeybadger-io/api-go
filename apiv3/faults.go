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
		pageOpts := ro
		pageOpts.page = page
		return s.list(ctx, projectID, pageOpts)
	})
}

func (s *FaultsService) list(ctx context.Context, projectID string, ro requestOptions) (*ListResponse[Fault], error) {
	params := &gen.ListFaultsParams{}
	if ro.query != "" {
		params.Q = &ro.query
	}
	if ro.page > 0 {
		page := gen.Page(ro.page)
		params.Page = &page
	}
	if ro.perPage > 0 {
		perPage := gen.PerPage(ro.perPage)
		params.PerPage = &perPage
	}

	var status int
	body, err := s.client.do(ctx, func() (*http.Response, error) {
		resp, err := s.client.gen().ListFaults(ctx, s.client.accountID(ro.accountID), projectID, params)
		if resp != nil {
			status = resp.StatusCode
		}
		return resp, err
	})
	if err != nil {
		return nil, err
	}
	return decodeOffsetList[Fault](status, body)
}

// Get returns a single fault.
func (s *FaultsService) Get(ctx context.Context, projectID, faultID string, opts ...Option) (*Fault, error) {
	ro := resolve(opts)

	var status int
	body, err := s.client.do(ctx, func() (*http.Response, error) {
		resp, err := s.client.gen().GetFault(ctx, s.client.accountID(ro.accountID), projectID, faultID)
		if resp != nil {
			status = resp.StatusCode
		}
		return resp, err
	})
	if err != nil {
		return nil, err
	}
	return decodeSingle[Fault](status, body)
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
	return CollectTimeSeries(ctx, func(ctx context.Context, url string) (*ListResponse[Notice], error) {
		if url != "" {
			return followTimeSeries[Notice](ctx, s.client, url)
		}
		return s.listNotices(ctx, projectID, faultID, ro)
	})
}

func (s *FaultsService) listNotices(ctx context.Context, projectID, faultID string, ro requestOptions) (*ListResponse[Notice], error) {
	params := &gen.ListNoticesParams{}
	if ro.limit > 0 {
		limit := gen.Limit(ro.limit)
		params.Limit = &limit
	}
	if ro.before != "" {
		before := gen.Before(ro.before)
		params.Before = &before
	}
	if ro.after != "" {
		after := gen.After(ro.after)
		params.After = &after
	}

	var status int
	body, err := s.client.do(ctx, func() (*http.Response, error) {
		resp, err := s.client.gen().ListNotices(ctx, s.client.accountID(ro.accountID), projectID, faultID, params)
		if resp != nil {
			status = resp.StatusCode
		}
		return resp, err
	})
	if err != nil {
		return nil, err
	}
	return decodeTimeSeriesList[Notice](status, body)
}
