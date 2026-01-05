package honeybadgerapi

import (
	"context"
	"fmt"
)

// StatusPagesService provides methods for interacting with status pages
type StatusPagesService struct {
	client *Client
}

// List retrieves all status pages for an account
func (s *StatusPagesService) List(ctx context.Context, accountID int) ([]StatusPage, error) {
	path := fmt.Sprintf("/accounts/%d/status_pages", accountID)

	req, err := s.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response StatusPageListResponse
	if err := s.client.do(ctx, req, &response); err != nil {
		return nil, err
	}

	return response.Results, nil
}

// Get retrieves a single status page by ID
func (s *StatusPagesService) Get(ctx context.Context, accountID, statusPageID int) (*StatusPage, error) {
	path := fmt.Sprintf("/accounts/%d/status_pages/%d", accountID, statusPageID)

	req, err := s.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var statusPage StatusPage
	if err := s.client.do(ctx, req, &statusPage); err != nil {
		return nil, err
	}

	return &statusPage, nil
}

// Create creates a new status page
func (s *StatusPagesService) Create(ctx context.Context, accountID int, params StatusPageParams) (*StatusPage, error) {
	path := fmt.Sprintf("/accounts/%d/status_pages", accountID)

	reqBody := StatusPageRequest{StatusPage: params}

	req, err := s.client.newRequest(ctx, "POST", path, reqBody)
	if err != nil {
		return nil, err
	}

	var statusPage StatusPage
	if err := s.client.do(ctx, req, &statusPage); err != nil {
		return nil, err
	}

	return &statusPage, nil
}

// Update updates an existing status page
func (s *StatusPagesService) Update(ctx context.Context, accountID, statusPageID int, params StatusPageParams) error {
	path := fmt.Sprintf("/accounts/%d/status_pages/%d", accountID, statusPageID)

	reqBody := StatusPageRequest{StatusPage: params}

	req, err := s.client.newRequest(ctx, "POST", path, reqBody)
	if err != nil {
		return err
	}

	// Update returns 204 No Content
	return s.client.do(ctx, req, nil)
}

// Delete deletes a status page
func (s *StatusPagesService) Delete(ctx context.Context, accountID, statusPageID int) error {
	path := fmt.Sprintf("/accounts/%d/status_pages/%d", accountID, statusPageID)

	req, err := s.client.newRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}
