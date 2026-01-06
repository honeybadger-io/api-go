package honeybadgerapi

import (
	"context"
	"fmt"
)

// AccountsService provides methods for interacting with accounts
type AccountsService struct {
	client *Client
}

// List retrieves all accounts for the authenticated user
func (s *AccountsService) List(ctx context.Context) ([]Account, error) {
	path := "/accounts"

	req, err := s.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response AccountListResponse
	if err := s.client.do(ctx, req, &response); err != nil {
		return nil, err
	}

	return response.Results, nil
}

// Get retrieves a single account by ID with quota and API stats
func (s *AccountsService) Get(ctx context.Context, accountID string) (*Account, error) {
	path := fmt.Sprintf("/accounts/%s", accountID)

	req, err := s.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var account Account
	if err := s.client.do(ctx, req, &account); err != nil {
		return nil, err
	}

	return &account, nil
}

// ListUsers retrieves all users for an account
func (s *AccountsService) ListUsers(ctx context.Context, accountID string) ([]AccountUser, error) {
	path := fmt.Sprintf("/accounts/%s/users", accountID)

	req, err := s.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var users []AccountUser
	if err := s.client.do(ctx, req, &users); err != nil {
		return nil, err
	}

	return users, nil
}

// GetUser retrieves a single user by ID
func (s *AccountsService) GetUser(ctx context.Context, accountID string, userID int) (*AccountUser, error) {
	path := fmt.Sprintf("/accounts/%s/users/%d", accountID, userID)

	req, err := s.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var user AccountUser
	if err := s.client.do(ctx, req, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// UpdateUser updates a user's role
func (s *AccountsService) UpdateUser(ctx context.Context, accountID string, userID int, role string) (*AccountUser, error) {
	path := fmt.Sprintf("/accounts/%s/users/%d", accountID, userID)

	reqBody := AccountUserUpdateRequest{}
	reqBody.User.Role = role

	req, err := s.client.newRequest(ctx, "PUT", path, reqBody)
	if err != nil {
		return nil, err
	}

	var user AccountUser
	if err := s.client.do(ctx, req, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// RemoveUser removes a user from an account
func (s *AccountsService) RemoveUser(ctx context.Context, accountID string, userID int) error {
	path := fmt.Sprintf("/accounts/%s/users/%d", accountID, userID)

	req, err := s.client.newRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}

// ListInvitations retrieves all invitations for an account
func (s *AccountsService) ListInvitations(ctx context.Context, accountID string) ([]AccountInvitation, error) {
	path := fmt.Sprintf("/accounts/%s/invitations", accountID)

	req, err := s.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var invitations []AccountInvitation
	if err := s.client.do(ctx, req, &invitations); err != nil {
		return nil, err
	}

	return invitations, nil
}

// GetInvitation retrieves a single invitation by ID
func (s *AccountsService) GetInvitation(ctx context.Context, accountID string, invitationID int) (*AccountInvitation, error) {
	path := fmt.Sprintf("/accounts/%s/invitations/%d", accountID, invitationID)

	req, err := s.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var invitation AccountInvitation
	if err := s.client.do(ctx, req, &invitation); err != nil {
		return nil, err
	}

	return &invitation, nil
}

// CreateInvitation creates a new account invitation
func (s *AccountsService) CreateInvitation(ctx context.Context, accountID string, params AccountInvitationParams) (*AccountInvitation, error) {
	path := fmt.Sprintf("/accounts/%s/invitations", accountID)

	reqBody := AccountInvitationRequest{Invitation: params}

	req, err := s.client.newRequest(ctx, "POST", path, reqBody)
	if err != nil {
		return nil, err
	}

	var invitation AccountInvitation
	if err := s.client.do(ctx, req, &invitation); err != nil {
		return nil, err
	}

	return &invitation, nil
}

// UpdateInvitation updates an existing account invitation
func (s *AccountsService) UpdateInvitation(ctx context.Context, accountID string, invitationID int, params AccountInvitationParams) (*AccountInvitation, error) {
	path := fmt.Sprintf("/accounts/%s/invitations/%d", accountID, invitationID)

	reqBody := AccountInvitationRequest{Invitation: params}

	req, err := s.client.newRequest(ctx, "PUT", path, reqBody)
	if err != nil {
		return nil, err
	}

	var invitation AccountInvitation
	if err := s.client.do(ctx, req, &invitation); err != nil {
		return nil, err
	}

	return &invitation, nil
}

// DeleteInvitation deletes an account invitation
func (s *AccountsService) DeleteInvitation(ctx context.Context, accountID string, invitationID int) error {
	path := fmt.Sprintf("/accounts/%s/invitations/%d", accountID, invitationID)

	req, err := s.client.newRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}
