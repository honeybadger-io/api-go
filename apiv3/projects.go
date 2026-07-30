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

// List returns one page of projects. Use Page to select which, and InAccount to
// address a specific account.
func (s *ProjectsService) List(ctx context.Context, opts ...Option) (*ListResponse[Project], error) {
	ro := resolve(opts)
	return s.list(ctx, ro)
}

// ListAll returns every project, walking pagination.
func (s *ProjectsService) ListAll(ctx context.Context, opts ...ListAllOption) ([]Project, error) {
	ro := resolveListAll(opts)
	return CollectPages(ctx, func(ctx context.Context, page int) (*ListResponse[Project], error) {
		pageOpts := ro
		pageOpts.page = page
		return s.list(ctx, pageOpts)
	})
}

func (s *ProjectsService) list(ctx context.Context, ro requestOptions) (*ListResponse[Project], error) {
	params := &gen.ListProjectsParams{}
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
		resp, err := s.client.gen().ListProjects(ctx, s.client.accountID(ro.accountID), params)
		if resp != nil {
			status = resp.StatusCode
		}
		return resp, err
	})
	if err != nil {
		return nil, err
	}
	return decodeOffsetList[Project](status, body)
}

// Get returns a single project by its opaque id.
func (s *ProjectsService) Get(ctx context.Context, projectID string, opts ...Option) (*Project, error) {
	ro := resolve(opts)

	var status int
	body, err := s.client.do(ctx, func() (*http.Response, error) {
		resp, err := s.client.gen().GetProject(ctx, s.client.accountID(ro.accountID), projectID)
		if resp != nil {
			status = resp.StatusCode
		}
		return resp, err
	})
	if err != nil {
		return nil, err
	}
	return decodeSingle[Project](status, body)
}
