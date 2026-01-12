package honeybadgerapi

import (
	"context"
	"fmt"
)

// EnvironmentsService provides methods for interacting with environments
type EnvironmentsService struct {
	client *Client
}

// List retrieves all environments for a project
func (s *EnvironmentsService) List(ctx context.Context, projectID int) ([]Environment, error) {
	path := fmt.Sprintf("/projects/%d/environments", projectID)

	req, err := s.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response EnvironmentListResponse
	if err := s.client.do(ctx, req, &response); err != nil {
		return nil, err
	}

	return response.Results, nil
}

// Get retrieves a single environment by ID
func (s *EnvironmentsService) Get(ctx context.Context, projectID, environmentID int) (*Environment, error) {
	path := fmt.Sprintf("/projects/%d/environments/%d", projectID, environmentID)

	req, err := s.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var environment Environment
	if err := s.client.do(ctx, req, &environment); err != nil {
		return nil, err
	}

	return &environment, nil
}

// Create creates a new environment
func (s *EnvironmentsService) Create(ctx context.Context, projectID int, params EnvironmentParams) (*Environment, error) {
	path := fmt.Sprintf("/projects/%d/environments", projectID)

	reqBody := EnvironmentRequest{Environment: params}

	req, err := s.client.newRequest(ctx, "POST", path, reqBody)
	if err != nil {
		return nil, err
	}

	var environment Environment
	if err := s.client.do(ctx, req, &environment); err != nil {
		return nil, err
	}

	return &environment, nil
}

// Update updates an existing environment
func (s *EnvironmentsService) Update(ctx context.Context, projectID, environmentID int, params EnvironmentParams) error {
	path := fmt.Sprintf("/projects/%d/environments/%d", projectID, environmentID)

	reqBody := EnvironmentRequest{Environment: params}

	req, err := s.client.newRequest(ctx, "PUT", path, reqBody)
	if err != nil {
		return err
	}

	// Update returns 204 No Content
	return s.client.do(ctx, req, nil)
}

// Delete deletes an environment
func (s *EnvironmentsService) Delete(ctx context.Context, projectID, environmentID int) error {
	path := fmt.Sprintf("/projects/%d/environments/%d", projectID, environmentID)

	req, err := s.client.newRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}
