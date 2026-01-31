package honeybadgerapi

import (
	"context"
	"fmt"
	"time"
)

// DashboardsService handles operations for the dashboards resource
type DashboardsService struct {
	client *Client
}

// Dashboard represents a Honeybadger Insights dashboard
type Dashboard struct {
	ID        string                   `json:"id"`
	Title     string                   `json:"title"`
	Widgets   []map[string]interface{} `json:"widgets"`
	IsDefault bool                     `json:"is_default"`
	Shared    bool                     `json:"shared"`
	CreatedAt time.Time                `json:"created_at"`
	UpdatedAt time.Time                `json:"updated_at"`
	ProjectID int                      `json:"project_id"`
}

// DashboardListResponse represents the API response for listing dashboards
type DashboardListResponse = ListResponse[Dashboard]

// DashboardRequest represents the request body for creating or updating a dashboard
type DashboardRequest struct {
	Title     string                   `json:"title"`
	DefaultTs string                   `json:"default_ts,omitempty"`
	Widgets   []map[string]interface{} `json:"widgets"`
}

// List returns all dashboards for a project.
//
// GET /v2/projects/{projectID}/dashboards
func (d *DashboardsService) List(ctx context.Context, projectID int) (*DashboardListResponse, error) {
	path := fmt.Sprintf("/projects/%d/dashboards", projectID)

	req, err := d.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response DashboardListResponse
	if err := d.client.do(ctx, req, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// Get returns a single dashboard by ID.
//
// GET /v2/projects/{projectID}/dashboards/{dashboardID}
func (d *DashboardsService) Get(ctx context.Context, projectID int, dashboardID string) (*Dashboard, error) {
	path := fmt.Sprintf("/projects/%d/dashboards/%s", projectID, dashboardID)

	req, err := d.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result Dashboard
	if err := d.client.do(ctx, req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Create creates a new dashboard for a project.
//
// POST /v2/projects/{projectID}/dashboards
func (d *DashboardsService) Create(ctx context.Context, projectID int, dashboardReq DashboardRequest) (*Dashboard, error) {
	body := map[string]interface{}{
		"dashboard": dashboardReq,
	}

	path := fmt.Sprintf("/projects/%d/dashboards", projectID)
	req, err := d.client.newRequest(ctx, "POST", path, body)
	if err != nil {
		return nil, err
	}

	var result Dashboard
	if err := d.client.do(ctx, req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Update updates an existing dashboard.
//
// PUT /v2/projects/{projectID}/dashboards/{dashboardID}
func (d *DashboardsService) Update(ctx context.Context, projectID int, dashboardID string, dashboardReq DashboardRequest) (*UpdateResult, error) {
	body := map[string]interface{}{
		"dashboard": dashboardReq,
	}

	path := fmt.Sprintf("/projects/%d/dashboards/%s", projectID, dashboardID)
	req, err := d.client.newRequest(ctx, "PUT", path, body)
	if err != nil {
		return nil, err
	}

	if err := d.client.do(ctx, req, nil); err != nil {
		return nil, err
	}

	return &UpdateResult{
		Success: true,
		Message: fmt.Sprintf("Dashboard %s was successfully updated", dashboardID),
	}, nil
}

// Delete deletes a dashboard.
//
// DELETE /v2/projects/{projectID}/dashboards/{dashboardID}
func (d *DashboardsService) Delete(ctx context.Context, projectID int, dashboardID string) (*DeleteResult, error) {
	path := fmt.Sprintf("/projects/%d/dashboards/%s", projectID, dashboardID)

	req, err := d.client.newRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return nil, err
	}

	if err := d.client.do(ctx, req, nil); err != nil {
		return nil, err
	}

	return &DeleteResult{
		Success: true,
		Message: fmt.Sprintf("Dashboard %s deleted successfully", dashboardID),
	}, nil
}
