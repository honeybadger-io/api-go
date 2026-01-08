package honeybadgerapi

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// DeploymentsService provides methods for interacting with deployments
type DeploymentsService struct {
	client *Client
}

// List retrieves all deployments for a project with optional filtering
func (s *DeploymentsService) List(ctx context.Context, projectID int, options DeploymentListOptions) ([]Deployment, error) {
	path := fmt.Sprintf("/projects/%d/deploys", projectID)

	// Build query parameters
	params := url.Values{}
	if options.Environment != "" {
		params.Set("environment", options.Environment)
	}
	if options.LocalUsername != "" {
		params.Set("local_username", options.LocalUsername)
	}
	if options.CreatedAfter > 0 {
		params.Set("created_after", strconv.FormatInt(options.CreatedAfter, 10))
	}
	if options.CreatedBefore > 0 {
		params.Set("created_before", strconv.FormatInt(options.CreatedBefore, 10))
	}
	if options.Limit > 0 {
		params.Set("limit", strconv.Itoa(options.Limit))
	}

	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	req, err := s.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response DeploymentListResponse
	if err := s.client.do(ctx, req, &response); err != nil {
		return nil, err
	}

	return response.Results, nil
}

// Get retrieves a single deployment by ID
func (s *DeploymentsService) Get(ctx context.Context, projectID, deploymentID int) (*Deployment, error) {
	path := fmt.Sprintf("/projects/%d/deploys/%d", projectID, deploymentID)

	req, err := s.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var deployment Deployment
	if err := s.client.do(ctx, req, &deployment); err != nil {
		return nil, err
	}

	return &deployment, nil
}

// Delete deletes a deployment
func (s *DeploymentsService) Delete(ctx context.Context, projectID, deploymentID int) error {
	path := fmt.Sprintf("/projects/%d/deploys/%d", projectID, deploymentID)

	req, err := s.client.newRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}
