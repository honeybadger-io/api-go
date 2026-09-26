package apiv3

import (
	"context"
	"net/http"

	"github.com/honeybadger-io/api-go/internal/gen"
)

// Comments live under a fault in v3's paths, so they are FaultsService methods
// alongside AddComment rather than a service of their own.
//
// Writing a comment attributes text to a person, so an account token holding
// faults:write is refused with requires_user_token for create, update and
// delete. Check errors.Is(err, ErrRequiresUserToken).

// ListComments returns one page of a fault's comments, newest first.
func (s *FaultsService) ListComments(ctx context.Context, projectID string, faultID int, opts ...Option) (*ListResponse[Comment], error) {
	return s.listComments(ctx, projectID, faultID, resolve(opts))
}

// ListAllComments returns every comment on a fault, walking from newest to
// oldest by following links.older.
func (s *FaultsService) ListAllComments(ctx context.Context, projectID string, faultID int, opts ...ListAllOption) ([]Comment, error) {
	ro := resolveListAll(opts)
	return CollectTimeSeries(ctx, func(ctx context.Context, link string) (*ListResponse[Comment], error) {
		if link != "" {
			return followTimeSeries[Comment](ctx, s.client, link)
		}
		return s.listComments(ctx, projectID, faultID, ro)
	})
}

func (s *FaultsService) listComments(ctx context.Context, projectID string, faultID int, ro requestOptions) (*ListResponse[Comment], error) {
	params := &gen.ListCommentsParams{}
	ro.applyTimeSeries(&params.Limit, &params.Before, &params.After)

	return listTimeSeries[Comment](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().ListComments(ctx, projectID, faultID, params)
	})
}

// GetComment returns one comment on a fault.
func (s *FaultsService) GetComment(ctx context.Context, projectID string, faultID int, commentID string, opts ...Option) (*Comment, error) {
	return getOne[Comment](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().GetComment(ctx, projectID, faultID, commentID)
	})
}

// UpdateComment replaces a comment's body and returns the comment as stored.
func (s *FaultsService) UpdateComment(ctx context.Context, projectID string, faultID int, commentID, body string, opts ...Option) (*Comment, error) {
	input := gen.UpdateCommentJSONRequestBody{Body: body}
	return getOne[Comment](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().UpdateComment(ctx, projectID, faultID, commentID, input)
	})
}

// DeleteComment removes a comment from a fault.
func (s *FaultsService) DeleteComment(ctx context.Context, projectID string, faultID int, commentID string, opts ...Option) error {
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().DeleteComment(ctx, projectID, faultID, commentID)
	})
}
