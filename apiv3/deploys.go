package apiv3

import (
	"context"
	"net/http"

	"github.com/honeybadger-io/api-go/internal/gen"
)

// Deploy is a deploy recorded for a project. Fault.LastNoticeDeploy is one of
// these.
type Deploy = gen.Deploy

// DeploysService handles a project's deploys.
type DeploysService struct {
	client *Client
}

// List returns one page of a project's deploys, newest first. Limit sizes the
// page, Before and After position within it, and InEnvironment and DeployedBy
// filter it.
func (s *DeploysService) List(ctx context.Context, projectID string, opts ...Option) (*ListResponse[Deploy], error) {
	return s.list(ctx, projectID, resolve(opts))
}

// ListAll returns every deploy for the project, walking from newest to oldest.
func (s *DeploysService) ListAll(ctx context.Context, projectID string, opts ...ListAllOption) ([]Deploy, error) {
	ro := resolveListAll(opts)
	return CollectTimeSeries(ctx, func(ctx context.Context, link string) (*ListResponse[Deploy], error) {
		if link != "" {
			return followTimeSeries[Deploy](ctx, s.client, link)
		}
		return s.list(ctx, projectID, ro)
	})
}

func (s *DeploysService) list(ctx context.Context, projectID string, ro requestOptions) (*ListResponse[Deploy], error) {
	params := &gen.ListDeploysParams{}
	ro.applyTimeSeries(&params.Limit, &params.Before, &params.After)
	if ro.environment != "" {
		params.Environment = &ro.environment
	}
	if ro.localUsername != "" {
		params.LocalUsername = &ro.localUsername
	}

	return listTimeSeries[Deploy](ctx, s.client, "listDeploys", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().ListDeploys(ctx, projectID, params)
	})
}

// Get returns one of a project's deploys.
func (s *DeploysService) Get(ctx context.Context, projectID, deployID string) (*Deploy, error) {
	return getOne[Deploy](ctx, s.client, "getDeploy", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().GetDeploy(ctx, projectID, deployID)
	})
}
