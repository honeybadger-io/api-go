package apiv3

import (
	"context"
	"net/http"

	"github.com/honeybadger-io/api-go/internal/gen"
)

// IngestionKey is an Ingestion Key (`hbp_`): the credential an app sends errors
// and events with, set as `api_key` in a notifier's config. It can't call the
// API; that takes an API Token.
type IngestionKey = gen.IngestionKey

// IngestionKeyParams are the writable fields of an Ingestion Key.
type IngestionKeyParams = gen.IngestionKeyInput

// IngestionKeysService handles a project's Ingestion Keys.
type IngestionKeysService struct {
	client *Client
}

// List returns one page of a project's Ingestion Keys.
func (s *IngestionKeysService) List(ctx context.Context, projectID string, opts ...Option) (*ListResponse[IngestionKey], error) {
	return s.list(ctx, projectID, resolve(opts))
}

// ListAll returns every Ingestion Key for the project, walking pagination.
func (s *IngestionKeysService) ListAll(ctx context.Context, projectID string, opts ...ListAllOption) ([]IngestionKey, error) {
	ro := resolveListAll(opts)
	return CollectPages(ctx, func(ctx context.Context, page int) (*ListResponse[IngestionKey], error) {
		ro.page = page
		return s.list(ctx, projectID, ro)
	})
}

func (s *IngestionKeysService) list(ctx context.Context, projectID string, ro requestOptions) (*ListResponse[IngestionKey], error) {
	params := &gen.ListIngestionKeysParams{}
	ro.applyOffset(&params.Page, &params.PerPage)

	return listOffset[IngestionKey](ctx, s.client, "listIngestionKeys", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().ListIngestionKeys(ctx, projectID, params)
	})
}

// Create makes a new Ingestion Key.
func (s *IngestionKeysService) Create(ctx context.Context, projectID string, p IngestionKeyParams) (*IngestionKey, error) {
	return getOne[IngestionKey](ctx, s.client, "createIngestionKey", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().CreateIngestionKey(ctx, projectID, gen.CreateIngestionKeyJSONRequestBody(p))
	})
}

// Update changes an Ingestion Key's label.
func (s *IngestionKeysService) Update(ctx context.Context, projectID, ingestionKeyID string, p IngestionKeyParams) (*IngestionKey, error) {
	return getOne[IngestionKey](ctx, s.client, "updateIngestionKey", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().UpdateIngestionKey(ctx, projectID, ingestionKeyID, gen.UpdateIngestionKeyJSONRequestBody(p))
	})
}

// Delete removes an Ingestion Key.
func (s *IngestionKeysService) Delete(ctx context.Context, projectID, ingestionKeyID string) error {
	return noContent(ctx, s.client, "deleteIngestionKey", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().DeleteIngestionKey(ctx, projectID, ingestionKeyID)
	})
}

// Get returns one of a project's Ingestion Keys.
func (s *IngestionKeysService) Get(ctx context.Context, projectID, ingestionKeyID string) (*IngestionKey, error) {
	return getOne[IngestionKey](ctx, s.client, "getIngestionKey", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().GetIngestionKey(ctx, projectID, ingestionKeyID)
	})
}
