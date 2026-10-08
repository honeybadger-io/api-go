package apiv3

import (
	"context"
	"net/http"

	"github.com/honeybadger-io/api-go/internal/gen"
)

// Project is a Honeybadger project. It carries no Ingestion Key, since a
// project can have several; list them with IngestionKeys.List.
type Project = gen.Project

// ProjectsService handles the projects resource.
type ProjectsService struct {
	client *Client
}

// List returns one page of projects. Use Page to select which.
func (s *ProjectsService) List(ctx context.Context, opts ...Option) (*ListResponse[Project], error) {
	return s.list(ctx, resolve(opts))
}

// ListAll returns every project, walking pagination.
func (s *ProjectsService) ListAll(ctx context.Context, opts ...ListAllOption) ([]Project, error) {
	ro := resolveListAll(opts)
	return CollectPages(ctx, func(ctx context.Context, page int) (*ListResponse[Project], error) {
		ro.page = page
		return s.list(ctx, ro)
	})
}

func (s *ProjectsService) list(ctx context.Context, ro requestOptions) (*ListResponse[Project], error) {
	params := &gen.ListProjectsParams{}
	ro.applyOffset(&params.Page, &params.PerPage)
	if ro.name != "" {
		params.Name = &ro.name
	}

	return listOffset[Project](ctx, s.client, "listProjects", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().ListProjects(ctx, params)
	})
}

// Get returns a single project by its opaque id.
func (s *ProjectsService) Get(ctx context.Context, projectID string) (*Project, error) {
	return getOne[Project](ctx, s.client, "getProject", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().GetProject(ctx, projectID)
	})
}
