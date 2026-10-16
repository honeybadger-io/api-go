package apiv3

import (
	"context"
	"net/http"

	"github.com/honeybadger-io/api-go/internal/gen"
)

// Environment is one of a project's environments, such as production.
type Environment = gen.Environment

// EnvironmentCreateParams are a new environment's fields. Name is required.
type EnvironmentCreateParams = gen.EnvironmentCreateInput

// EnvironmentUpdateParams are the fields an environment update can change.
// Unset fields keep their values.
type EnvironmentUpdateParams = gen.EnvironmentInput

// EnvironmentsService handles a project's environments.
type EnvironmentsService struct {
	client *Client
}

// List returns one page of a project's environments.
func (s *EnvironmentsService) List(ctx context.Context, projectID string, opts ...Option) (*ListResponse[Environment], error) {
	return s.list(ctx, projectID, resolve(opts))
}

// ListAll returns every environment in the project, walking pagination.
func (s *EnvironmentsService) ListAll(ctx context.Context, projectID string, opts ...ListAllOption) ([]Environment, error) {
	ro := resolveListAll(opts)
	return CollectPages(ctx, func(ctx context.Context, page int) (*ListResponse[Environment], error) {
		ro.page = page
		return s.list(ctx, projectID, ro)
	})
}

func (s *EnvironmentsService) list(ctx context.Context, projectID string, ro requestOptions) (*ListResponse[Environment], error) {
	params := &gen.ListEnvironmentsParams{}
	ro.applyOffset(&params.Page, &params.PerPage)

	return listOffset[Environment](ctx, s.client, "listEnvironments", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().ListEnvironments(ctx, projectID, params)
	})
}

// Get returns one of a project's environments.
func (s *EnvironmentsService) Get(ctx context.Context, projectID, environmentID string) (*Environment, error) {
	return getOne[Environment](ctx, s.client, "getEnvironment", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().GetEnvironment(ctx, projectID, environmentID)
	})
}

// Create adds an environment to a project.
func (s *EnvironmentsService) Create(ctx context.Context, projectID string, p EnvironmentCreateParams) (*Environment, error) {
	return getOne[Environment](ctx, s.client, "createEnvironment", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().CreateEnvironment(ctx, projectID, p)
	})
}

// Update changes an environment.
func (s *EnvironmentsService) Update(ctx context.Context, projectID, environmentID string, p EnvironmentUpdateParams) (*Environment, error) {
	return getOne[Environment](ctx, s.client, "updateEnvironment", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().UpdateEnvironment(ctx, projectID, environmentID, p)
	})
}

// Delete removes an environment from a project.
func (s *EnvironmentsService) Delete(ctx context.Context, projectID, environmentID string) error {
	return noContent(ctx, s.client, "deleteEnvironment", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().DeleteEnvironment(ctx, projectID, environmentID)
	})
}
