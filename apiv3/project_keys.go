package apiv3

import (
	"context"
	"net/http"

	"github.com/honeybadger-io/api-go/internal/gen"
)

// ProjectKey is an ingestion key for a project.
type ProjectKey = gen.ProjectKey

// ProjectKeyParams are the writable fields of a project key.
type ProjectKeyParams = gen.ProjectKeyInput

// ProjectKeysService handles project ingestion keys.
type ProjectKeysService struct {
	client *Client
}

// List returns one page of a project's keys.
func (s *ProjectKeysService) List(ctx context.Context, projectID string, opts ...Option) (*ListResponse[ProjectKey], error) {
	return s.list(ctx, projectID, resolve(opts))
}

// ListAll returns every key for the project, walking pagination.
func (s *ProjectKeysService) ListAll(ctx context.Context, projectID string, opts ...ListAllOption) ([]ProjectKey, error) {
	ro := resolveListAll(opts)
	return CollectPages(ctx, func(ctx context.Context, page int) (*ListResponse[ProjectKey], error) {
		ro.page = page
		return s.list(ctx, projectID, ro)
	})
}

func (s *ProjectKeysService) list(ctx context.Context, projectID string, ro requestOptions) (*ListResponse[ProjectKey], error) {
	params := &gen.ListProjectKeysParams{}
	ro.applyOffset(&params.Page, &params.PerPage)

	return listOffset[ProjectKey](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().ListProjectKeys(ctx, projectID, params)
	})
}

// Create makes a new project key.
func (s *ProjectKeysService) Create(ctx context.Context, projectID string, p ProjectKeyParams, opts ...Option) (*ProjectKey, error) {
	return getOne[ProjectKey](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().CreateProjectKey(ctx, projectID, gen.CreateProjectKeyJSONRequestBody(p))
	})
}

// Update changes a key's label.
func (s *ProjectKeysService) Update(ctx context.Context, projectID, keyID string, p ProjectKeyParams, opts ...Option) (*ProjectKey, error) {
	return getOne[ProjectKey](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().UpdateProjectKey(ctx, projectID, keyID, gen.UpdateProjectKeyJSONRequestBody(p))
	})
}

// Delete removes a project key.
func (s *ProjectKeysService) Delete(ctx context.Context, projectID, keyID string, opts ...Option) error {
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().DeleteProjectKey(ctx, projectID, keyID)
	})
}
