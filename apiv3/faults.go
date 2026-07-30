package apiv3

import (
	"context"

	"github.com/honeybadger-io/api-go/internal/gen"
)

// Fault is an error group.
type Fault = gen.Fault

// Notice is a single occurrence of a fault.
type Notice = gen.Notice

// FaultsService handles the faults resource and its notices.
type FaultsService struct {
	client *Client
}

// FaultListOptions are the options for listing faults.
//
// Query is v3's single filter parameter. Environment, resolved, and ignored
// filters are expressed inside it rather than as separate fields —
// "environment:production", "is:resolved", "is:ignored", each negatable with a
// leading "-". This mirrors the API rather than inventing a friendlier surface
// that would need updating whenever the filter language grows.
type FaultListOptions struct {
	Query     string
	Page      int
	PerPage   int
	AccountID string
}

// FaultGetOptions are the options for fetching one fault.
type FaultGetOptions struct {
	AccountID string
}

// NoticeListOptions are the options for listing a fault's notices.
//
// Notices are cursor-paginated rather than offset-paginated. Before fetches
// older notices, After fetches newer ones; leave both empty to start at the
// newest.
type NoticeListOptions struct {
	Limit     int
	Before    string
	After     string
	AccountID string
}

// List returns one page of faults for a project.
func (s *FaultsService) List(ctx context.Context, projectID string, opts FaultListOptions) (*ListResponse[Fault], error) {
	params := &gen.ListFaultsParams{}
	if opts.Query != "" {
		params.Q = &opts.Query
	}
	if opts.Page > 0 {
		page := gen.Page(opts.Page)
		params.Page = &page
	}
	if opts.PerPage > 0 {
		perPage := gen.PerPage(opts.PerPage)
		params.PerPage = &perPage
	}

	resp, err := s.client.gen().ListFaultsWithResponse(ctx, s.client.accountID(opts.AccountID), projectID, params)
	if err != nil {
		return nil, err
	}
	if err := s.client.checkResponse(resp.HTTPResponse, resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, s.client.malformed(resp.HTTPResponse, resp.Body)
	}

	out := &ListResponse[Fault]{
		Pagination: resp.JSON200.Pagination,
		Links:      derefMap(resp.JSON200.Links),
	}
	if resp.JSON200.Data != nil {
		out.Data = *resp.JSON200.Data
	}
	if resp.JSON200.Meta != nil && resp.JSON200.Meta.RequestId != nil {
		out.RequestID = *resp.JSON200.Meta.RequestId
	}
	return out, nil
}

// ListAll returns every fault matching the options, walking pagination.
// opts.Page is ignored.
func (s *FaultsService) ListAll(ctx context.Context, projectID string, opts FaultListOptions) ([]Fault, error) {
	return CollectPages(ctx, func(ctx context.Context, page int) (*ListResponse[Fault], error) {
		pageOpts := opts
		pageOpts.Page = page
		return s.List(ctx, projectID, pageOpts)
	})
}

// Get returns a single fault.
func (s *FaultsService) Get(ctx context.Context, projectID, faultID string, opts FaultGetOptions) (*Fault, error) {
	resp, err := s.client.gen().GetFaultWithResponse(ctx, s.client.accountID(opts.AccountID), projectID, faultID)
	if err != nil {
		return nil, err
	}
	if err := s.client.checkResponse(resp.HTTPResponse, resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil || resp.JSON200.Data == nil {
		return nil, s.client.malformed(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200.Data, nil
}

// ListNotices returns one page of a fault's notices, newest first.
func (s *FaultsService) ListNotices(ctx context.Context, projectID, faultID string, opts NoticeListOptions) (*ListResponse[Notice], error) {
	params := &gen.ListNoticesParams{}
	if opts.Limit > 0 {
		limit := gen.Limit(opts.Limit)
		params.Limit = &limit
	}
	if opts.Before != "" {
		before := gen.Before(opts.Before)
		params.Before = &before
	}
	if opts.After != "" {
		after := gen.After(opts.After)
		params.After = &after
	}

	resp, err := s.client.gen().ListNoticesWithResponse(ctx, s.client.accountID(opts.AccountID), projectID, faultID, params)
	if err != nil {
		return nil, err
	}
	if err := s.client.checkResponse(resp.HTTPResponse, resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, s.client.malformed(resp.HTTPResponse, resp.Body)
	}

	// Cursor endpoints put CursorPagination under the same "pagination" key that
	// offset endpoints use for Pagination, so it lands in Cursor here.
	out := &ListResponse[Notice]{
		Cursor: resp.JSON200.Pagination,
		Links:  derefMap(resp.JSON200.Links),
	}
	if resp.JSON200.Data != nil {
		out.Data = *resp.JSON200.Data
	}
	if resp.JSON200.Meta != nil && resp.JSON200.Meta.RequestId != nil {
		out.RequestID = *resp.JSON200.Meta.RequestId
	}
	return out, nil
}

// ListAllNotices returns every notice for a fault, walking the cursor from
// newest to oldest. opts.Before and opts.After are ignored.
func (s *FaultsService) ListAllNotices(ctx context.Context, projectID, faultID string, opts NoticeListOptions) ([]Notice, error) {
	return CollectCursor(ctx, func(ctx context.Context, before string) (*ListResponse[Notice], error) {
		pageOpts := opts
		pageOpts.Before = before
		pageOpts.After = ""
		return s.ListNotices(ctx, projectID, faultID, pageOpts)
	})
}
