package apiv3

import (
	"context"
	"net/http"

	"github.com/honeybadger-io/api-go/internal/gen"
)

// Project is a Honeybadger project.
type Project = gen.Project

// ProjectsService handles the projects resource.
type ProjectsService struct {
	client *Client
}

// ProjectListOptions are the options for listing projects.
//
// AccountID is optional on every options struct: leaving it empty uses the
// client's account, which defaults to the `me` sentinel that v3 resolves from
// the credential. Set it when a credential covers several accounts, which is the
// case that returns ambiguous_account.
type ProjectListOptions struct {
	Page      int
	PerPage   int
	AccountID string
}

// ProjectGetOptions are the options for fetching one project.
type ProjectGetOptions struct {
	AccountID string
}

// List returns one page of projects.
func (s *ProjectsService) List(ctx context.Context, opts ProjectListOptions) (*ListResponse[Project], error) {
	params := &gen.ListProjectsParams{}
	if opts.Page > 0 {
		page := gen.Page(opts.Page)
		params.Page = &page
	}
	if opts.PerPage > 0 {
		perPage := gen.PerPage(opts.PerPage)
		params.PerPage = &perPage
	}

	resp, err := s.client.gen().ListProjectsWithResponse(ctx, s.client.accountID(opts.AccountID), params)
	if err != nil {
		return nil, err
	}
	if err := s.client.checkResponse(resp.HTTPResponse, resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, s.client.malformed(resp.HTTPResponse, resp.Body)
	}

	out := &ListResponse[Project]{
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

// ListAll returns every project, walking pagination. opts.Page is ignored.
func (s *ProjectsService) ListAll(ctx context.Context, opts ProjectListOptions) ([]Project, error) {
	return CollectPages(ctx, func(ctx context.Context, page int) (*ListResponse[Project], error) {
		pageOpts := opts
		pageOpts.Page = page
		return s.List(ctx, pageOpts)
	})
}

// Get returns a single project by its opaque id.
func (s *ProjectsService) Get(ctx context.Context, projectID string, opts ProjectGetOptions) (*Project, error) {
	resp, err := s.client.gen().GetProjectWithResponse(ctx, s.client.accountID(opts.AccountID), projectID)
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

// derefMap flattens the generated `*map[string]interface{}` into something
// callers can index without a nil check.
func derefMap(m *map[string]interface{}) map[string]any {
	if m == nil {
		return nil
	}
	return *m
}

// checkResponse converts a non-2xx response into an *Error, attaching the
// rate-limit snapshot so a 429 tells the caller when to retry.
func (c *Client) checkResponse(resp *http.Response, body []byte) error {
	if resp == nil {
		return nil
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	apiErr := parseError(resp.StatusCode, body)
	apiErr.RateLimit = c.LastRateLimit()
	return apiErr
}

// malformed reports a 2xx whose body did not match the documented envelope.
// Returning an error beats handing back a zero value that looks like real data.
func (c *Client) malformed(resp *http.Response, body []byte) error {
	status := 0
	if resp != nil {
		status = resp.StatusCode
	}
	apiErr := parseError(status, body)
	apiErr.Message = "response did not match the documented envelope: " + apiErr.Message
	return apiErr
}
