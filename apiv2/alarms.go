package apiv2

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// AlarmsService handles operations for the alarms resource
type AlarmsService struct {
	client *Client
}

// Alarm represents a Honeybadger Insights alarm
type Alarm struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	State            string                 `json:"state"`
	Query            string                 `json:"query"`
	StreamIDs        []string               `json:"stream_ids"`
	EvaluationPeriod string                 `json:"evaluation_period"`
	LookbackLag      string                 `json:"lookback_lag"`
	TriggerConfig    map[string]interface{} `json:"trigger_config"`
	Error            string                 `json:"error,omitempty"`
	LastCheckedAt    *time.Time             `json:"last_checked_at"`
	NextCheckAt      *time.Time             `json:"next_check_at"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	URL              string                 `json:"url"`
	ProjectID        int                    `json:"project_id"`
}

// AlarmListResponse represents the API response for listing alarms
type AlarmListResponse = ListResponse[Alarm]

// AlarmRequest represents the request body for creating or updating an alarm
type AlarmRequest struct {
	Name             string                 `json:"name"`
	Description      string                 `json:"description,omitempty"`
	Query            string                 `json:"query"`
	StreamIDs        []string               `json:"stream_ids,omitempty"`
	EvaluationPeriod string                 `json:"evaluation_period"`
	LookbackLag      string                 `json:"lookback_lag,omitempty"`
	TriggerConfig    map[string]interface{} `json:"trigger_config"`
}

// AlarmTrigger represents a single trigger event in alarm history
type AlarmTrigger struct {
	ID        string                 `json:"id"`
	State     string                 `json:"state"`
	Result    map[string]interface{} `json:"result"`
	CreatedAt time.Time              `json:"created_at"`
}

// AlarmHistoryResponse represents the API response for alarm history
type AlarmHistoryResponse struct {
	Triggers []AlarmTrigger  `json:"triggers"`
	Links    PaginationLinks `json:"links"`
}

// List returns all alarms for a project.
//
// GET /v2/projects/{projectID}/alarms
func (a *AlarmsService) List(ctx context.Context, projectID int) (*AlarmListResponse, error) {
	path := fmt.Sprintf("/projects/%d/alarms", projectID)

	req, err := a.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response AlarmListResponse
	if err := a.client.do(ctx, req, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// Get returns a single alarm by ID.
//
// GET /v2/projects/{projectID}/alarms/{alarmID}
func (a *AlarmsService) Get(ctx context.Context, projectID int, alarmID string) (*Alarm, error) {
	path := fmt.Sprintf("/projects/%d/alarms/%s", projectID, alarmID)

	req, err := a.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result Alarm
	if err := a.client.do(ctx, req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Create creates a new alarm for a project.
//
// POST /v2/projects/{projectID}/alarms
func (a *AlarmsService) Create(ctx context.Context, projectID int, alarmReq AlarmRequest) (*Alarm, error) {
	body := map[string]interface{}{
		"alarm": alarmReq,
	}

	path := fmt.Sprintf("/projects/%d/alarms", projectID)
	req, err := a.client.newRequest(ctx, "POST", path, body)
	if err != nil {
		return nil, err
	}

	var result Alarm
	if err := a.client.do(ctx, req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Update updates an existing alarm.
//
// PUT /v2/projects/{projectID}/alarms/{alarmID}
func (a *AlarmsService) Update(ctx context.Context, projectID int, alarmID string, alarmReq AlarmRequest) (*UpdateResult, error) {
	body := map[string]interface{}{
		"alarm": alarmReq,
	}

	path := fmt.Sprintf("/projects/%d/alarms/%s", projectID, alarmID)
	req, err := a.client.newRequest(ctx, "PUT", path, body)
	if err != nil {
		return nil, err
	}

	if err := a.client.do(ctx, req, nil); err != nil {
		return nil, err
	}

	return &UpdateResult{
		Success: true,
		Message: fmt.Sprintf("Alarm %s was successfully updated", alarmID),
	}, nil
}

// Delete deletes an alarm.
//
// DELETE /v2/projects/{projectID}/alarms/{alarmID}
func (a *AlarmsService) Delete(ctx context.Context, projectID int, alarmID string) (*DeleteResult, error) {
	path := fmt.Sprintf("/projects/%d/alarms/%s", projectID, alarmID)

	req, err := a.client.newRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return nil, err
	}

	if err := a.client.do(ctx, req, nil); err != nil {
		return nil, err
	}

	return &DeleteResult{
		Success: true,
		Message: fmt.Sprintf("Alarm %s deleted successfully", alarmID),
	}, nil
}

// History returns the trigger history for an alarm.
//
// GET /v2/projects/{projectID}/alarms/{alarmID}/history
func (a *AlarmsService) History(ctx context.Context, projectID int, alarmID string, page int) (*AlarmHistoryResponse, error) {
	path := fmt.Sprintf("/projects/%d/alarms/%s/history", projectID, alarmID)
	params := url.Values{}
	if page > 0 {
		params.Set("page", strconv.Itoa(page))
	}
	if len(params) > 0 {
		path = fmt.Sprintf("%s?%s", path, params.Encode())
	}

	req, err := a.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result AlarmHistoryResponse
	if err := a.client.do(ctx, req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
