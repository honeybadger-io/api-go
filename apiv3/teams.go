package apiv3

import (
	"context"
	"net/http"

	"github.com/honeybadger-io/api-go/internal/gen"
)

// Team is a group of account members with access to a set of projects.
type Team = gen.Team

// TeamCreateParams are a new team's fields. Name is required.
type TeamCreateParams = gen.TeamCreateInput

// TeamUpdateParams are the fields a team update can change. Unset fields keep
// their values, and a ProjectIds list replaces the team's projects.
type TeamUpdateParams = gen.TeamInput

// TeamsService handles the account's teams. Teams belong to the account the
// credential resolves to, not to a project.
type TeamsService struct {
	client *Client
}

// List returns one page of the account's teams. Named narrows it to the team
// with exactly that name.
func (s *TeamsService) List(ctx context.Context, opts ...Option) (*ListResponse[Team], error) {
	return s.list(ctx, resolve(opts))
}

// ListAll returns every team in the account, walking pagination.
func (s *TeamsService) ListAll(ctx context.Context, opts ...ListAllOption) ([]Team, error) {
	ro := resolveListAll(opts)
	return CollectPages(ctx, func(ctx context.Context, page int) (*ListResponse[Team], error) {
		ro.page = page
		return s.list(ctx, ro)
	})
}

func (s *TeamsService) list(ctx context.Context, ro requestOptions) (*ListResponse[Team], error) {
	params := &gen.ListTeamsParams{}
	ro.applyOffset(&params.Page, &params.PerPage)
	if ro.name != "" {
		params.Name = &ro.name
	}

	return listOffset[Team](ctx, s.client, "listTeams", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().ListTeams(ctx, params)
	})
}

// Get returns a team.
func (s *TeamsService) Get(ctx context.Context, teamID string) (*Team, error) {
	return getOne[Team](ctx, s.client, "getTeam", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().GetTeam(ctx, teamID)
	})
}

// Create makes a new team.
func (s *TeamsService) Create(ctx context.Context, p TeamCreateParams) (*Team, error) {
	return getOne[Team](ctx, s.client, "createTeam", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().CreateTeam(ctx, p)
	})
}

// Update changes a team.
func (s *TeamsService) Update(ctx context.Context, teamID string, p TeamUpdateParams) (*Team, error) {
	return getOne[Team](ctx, s.client, "updateTeam", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().UpdateTeam(ctx, teamID, p)
	})
}

// Delete removes a team.
func (s *TeamsService) Delete(ctx context.Context, teamID string) error {
	return noContent(ctx, s.client, "deleteTeam", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().DeleteTeam(ctx, teamID)
	})
}
