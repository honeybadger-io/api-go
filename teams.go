package honeybadgerapi

import (
	"context"
	"fmt"
	"net/url"
)

// TeamsService provides methods for interacting with teams
type TeamsService struct {
	client *Client
}

// List retrieves all teams for an account
func (s *TeamsService) List(ctx context.Context, accountID string) ([]Team, error) {
	path := "/teams"

	params := url.Values{}
	params.Set("account_id", accountID)
	path += "?" + params.Encode()

	req, err := s.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response TeamListResponse
	if err := s.client.do(ctx, req, &response); err != nil {
		return nil, err
	}

	return response.Results, nil
}

// Get retrieves a single team by ID
func (s *TeamsService) Get(ctx context.Context, teamID int) (*Team, error) {
	path := fmt.Sprintf("/teams/%d", teamID)

	req, err := s.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var team Team
	if err := s.client.do(ctx, req, &team); err != nil {
		return nil, err
	}

	return &team, nil
}

// Create creates a new team
func (s *TeamsService) Create(ctx context.Context, accountID string, name string) (*Team, error) {
	path := "/teams"

	params := url.Values{}
	params.Set("account_id", accountID)
	path += "?" + params.Encode()

	reqBody := TeamRequest{}
	reqBody.Team.Name = name

	req, err := s.client.newRequest(ctx, "POST", path, reqBody)
	if err != nil {
		return nil, err
	}

	var team Team
	if err := s.client.do(ctx, req, &team); err != nil {
		return nil, err
	}

	return &team, nil
}

// Update updates an existing team
func (s *TeamsService) Update(ctx context.Context, teamID int, name string) error {
	path := fmt.Sprintf("/teams/%d", teamID)

	reqBody := TeamRequest{}
	reqBody.Team.Name = name

	req, err := s.client.newRequest(ctx, "PUT", path, reqBody)
	if err != nil {
		return err
	}

	// Update returns 204 No Content.
	return s.client.do(ctx, req, nil)
}

// Delete deletes a team
func (s *TeamsService) Delete(ctx context.Context, teamID int) error {
	path := fmt.Sprintf("/teams/%d", teamID)

	req, err := s.client.newRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}

// ListMembers retrieves all members of a team
func (s *TeamsService) ListMembers(ctx context.Context, teamID int) ([]TeamMember, error) {
	path := fmt.Sprintf("/teams/%d/team_members", teamID)

	req, err := s.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response TeamMemberListResponse
	if err := s.client.do(ctx, req, &response); err != nil {
		return nil, err
	}

	return response.Results, nil
}

// UpdateMember updates a team member's permissions
func (s *TeamsService) UpdateMember(ctx context.Context, teamID, memberID int, admin bool) error {
	path := fmt.Sprintf("/teams/%d/team_members/%d", teamID, memberID)

	reqBody := TeamMemberUpdateRequest{}
	reqBody.TeamMember.Admin = admin

	req, err := s.client.newRequest(ctx, "PUT", path, reqBody)
	if err != nil {
		return err
	}

	// Update returns 204 No Content.
	return s.client.do(ctx, req, nil)
}

// RemoveMember removes a member from a team
func (s *TeamsService) RemoveMember(ctx context.Context, teamID, memberID int) error {
	path := fmt.Sprintf("/teams/%d/team_members/%d", teamID, memberID)

	req, err := s.client.newRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}

// ListInvitations retrieves all invitations for a team
func (s *TeamsService) ListInvitations(ctx context.Context, teamID int) ([]TeamInvitation, error) {
	path := fmt.Sprintf("/teams/%d/team_invitations", teamID)

	req, err := s.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response TeamInvitationListResponse
	if err := s.client.do(ctx, req, &response); err != nil {
		return nil, err
	}

	return response.Results, nil
}

// GetInvitation retrieves a single invitation by ID
func (s *TeamsService) GetInvitation(ctx context.Context, teamID, invitationID int) (*TeamInvitation, error) {
	path := fmt.Sprintf("/teams/%d/team_invitations/%d", teamID, invitationID)

	req, err := s.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var invitation TeamInvitation
	if err := s.client.do(ctx, req, &invitation); err != nil {
		return nil, err
	}

	return &invitation, nil
}

// CreateInvitation creates a new team invitation
func (s *TeamsService) CreateInvitation(ctx context.Context, teamID int, params TeamInvitationParams) (*TeamInvitation, error) {
	path := fmt.Sprintf("/teams/%d/team_invitations", teamID)

	reqBody := TeamInvitationRequest{TeamInvitation: params}

	req, err := s.client.newRequest(ctx, "POST", path, reqBody)
	if err != nil {
		return nil, err
	}

	var invitation TeamInvitation
	if err := s.client.do(ctx, req, &invitation); err != nil {
		return nil, err
	}

	return &invitation, nil
}

// UpdateInvitation updates an existing team invitation
func (s *TeamsService) UpdateInvitation(ctx context.Context, teamID, invitationID int, params TeamInvitationParams) error {
	path := fmt.Sprintf("/teams/%d/team_invitations/%d", teamID, invitationID)

	reqBody := TeamInvitationRequest{TeamInvitation: params}

	req, err := s.client.newRequest(ctx, "PUT", path, reqBody)
	if err != nil {
		return err
	}

	// Update returns 204 No Content.
	return s.client.do(ctx, req, nil)
}

// DeleteInvitation deletes a team invitation
func (s *TeamsService) DeleteInvitation(ctx context.Context, teamID, invitationID int) error {
	path := fmt.Sprintf("/teams/%d/team_invitations/%d", teamID, invitationID)

	req, err := s.client.newRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}
