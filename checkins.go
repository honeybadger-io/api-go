package honeybadgerapi

import (
	"context"
	"fmt"
)

// CheckInsService provides methods for interacting with check-ins
type CheckInsService struct {
	client *Client
}

// List retrieves all check-ins for a project
func (s *CheckInsService) List(ctx context.Context, projectID int) ([]CheckIn, error) {
	path := fmt.Sprintf("/projects/%d/check_ins", projectID)

	req, err := s.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response CheckInListResponse
	if err := s.client.do(ctx, req, &response); err != nil {
		return nil, err
	}

	return response.Results, nil
}

// Get retrieves a single check-in by ID
func (s *CheckInsService) Get(ctx context.Context, projectID, checkInID int) (*CheckIn, error) {
	path := fmt.Sprintf("/projects/%d/check_ins/%d", projectID, checkInID)

	req, err := s.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var checkIn CheckIn
	if err := s.client.do(ctx, req, &checkIn); err != nil {
		return nil, err
	}

	return &checkIn, nil
}

// Create creates a new check-in
func (s *CheckInsService) Create(ctx context.Context, projectID int, params CheckInParams) (*CheckIn, error) {
	path := fmt.Sprintf("/projects/%d/check_ins", projectID)

	reqBody := CheckInRequest{CheckIn: params}

	req, err := s.client.newRequest(ctx, "POST", path, reqBody)
	if err != nil {
		return nil, err
	}

	var checkIn CheckIn
	if err := s.client.do(ctx, req, &checkIn); err != nil {
		return nil, err
	}

	return &checkIn, nil
}

// Update updates an existing check-in
func (s *CheckInsService) Update(ctx context.Context, projectID, checkInID int, params CheckInParams) (*CheckIn, error) {
	path := fmt.Sprintf("/projects/%d/check_ins/%d", projectID, checkInID)

	reqBody := CheckInRequest{CheckIn: params}

	req, err := s.client.newRequest(ctx, "PUT", path, reqBody)
	if err != nil {
		return nil, err
	}

	var checkIn CheckIn
	if err := s.client.do(ctx, req, &checkIn); err != nil {
		return nil, err
	}

	return &checkIn, nil
}

// BulkUpdate updates all check-ins for a project
// WARNING: An empty payload will delete all existing check-ins
func (s *CheckInsService) BulkUpdate(ctx context.Context, projectID int, checkIns []CheckInParams) (*CheckInBulkUpdateResponse, error) {
	path := fmt.Sprintf("/projects/%d/check_ins", projectID)

	// Wrap the check-ins in the expected format
	reqBody := struct {
		CheckIns []CheckInParams `json:"check_ins"`
	}{
		CheckIns: checkIns,
	}

	req, err := s.client.newRequest(ctx, "PUT", path, reqBody)
	if err != nil {
		return nil, err
	}

	var response CheckInBulkUpdateResponse
	if err := s.client.do(ctx, req, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// Delete deletes a check-in
func (s *CheckInsService) Delete(ctx context.Context, projectID, checkInID int) error {
	path := fmt.Sprintf("/projects/%d/check_ins/%d", projectID, checkInID)

	req, err := s.client.newRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}
