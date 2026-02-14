package honeybadgerapi

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// FaultsService handles operations for the faults resource
type FaultsService struct {
	client *Client
}

// FaultListOptions represents options for listing faults
type FaultListOptions struct {
	Q              string // Search string
	CreatedAfter   int64  // Filter faults created after this time (seconds since Unix epoch)
	OccurredAfter  int64  // Filter faults that occurred after this time (seconds since Unix epoch)
	OccurredBefore int64  // Filter faults that occurred before this time (seconds since Unix epoch)
	Limit          int    // Max 25
	Order          string // "recent" or "frequent"
	Page           int    // Page number for pagination
}

// FaultListResponse represents the API response for listing faults
type FaultListResponse = ListResponse[Fault]

// List returns a list of faults for a project with optional filtering and ordering.
//
// Honeybadger API docs: https://docs.honeybadger.io/api/faults/#get-a-fault-list-or-fault-details
//
// GET /v2/projects/{projectID}/faults
func (f *FaultsService) List(ctx context.Context, projectID int, options FaultListOptions) (*FaultListResponse, error) {
	path := fmt.Sprintf("/projects/%d/faults", projectID)

	// Build query parameters using url.Values
	params := url.Values{}
	if options.Q != "" {
		params.Set("q", options.Q)
	}
	if options.CreatedAfter > 0 {
		params.Set("created_after", strconv.FormatInt(options.CreatedAfter, 10))
	}
	if options.OccurredAfter > 0 {
		params.Set("occurred_after", strconv.FormatInt(options.OccurredAfter, 10))
	}
	if options.OccurredBefore > 0 {
		params.Set("occurred_before", strconv.FormatInt(options.OccurredBefore, 10))
	}
	if options.Limit > 0 {
		params.Set("limit", strconv.Itoa(options.Limit))
	}
	if options.Order != "" {
		params.Set("order", options.Order)
	}
	if options.Page > 0 {
		params.Set("page", strconv.Itoa(options.Page))
	}

	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	req, err := f.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response FaultListResponse
	if err := f.client.do(ctx, req, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// Get returns a single fault by ID with full fault details.
//
// Honeybadger API docs: https://docs.honeybadger.io/api/faults/#get-a-fault-list-or-fault-details
//
// GET /v2/projects/{projectID}/faults/{faultID}
func (f *FaultsService) Get(ctx context.Context, projectID, faultID int) (*Fault, error) {
	path := fmt.Sprintf("/projects/%d/faults/%d", projectID, faultID)
	req, err := f.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result Fault
	if err := f.client.do(ctx, req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// FaultListNoticesOptions represents options for listing notices for a fault
type FaultListNoticesOptions struct {
	CreatedAfter  int64 // Filter notices created after this time (seconds since Unix epoch)
	CreatedBefore int64 // Filter notices created before this time (seconds since Unix epoch)
	Limit         int   // Max 25
}

// FaultNoticesResponse represents the API response for listing fault notices
type FaultNoticesResponse = ListResponse[Notice]

// ListNotices returns a list of notices for a specific fault with optional filtering.
//
// Honeybadger API docs: https://docs.honeybadger.io/api/faults/#get-a-list-of-notices
//
// GET /v2/projects/{projectID}/faults/{faultID}/notices
func (f *FaultsService) ListNotices(ctx context.Context, projectID, faultID int, options FaultListNoticesOptions) (*FaultNoticesResponse, error) {
	path := fmt.Sprintf("/projects/%d/faults/%d/notices", projectID, faultID)

	// Build query parameters using url.Values
	params := url.Values{}
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

	req, err := f.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response FaultNoticesResponse
	if err := f.client.do(ctx, req, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// FaultAffectedUser represents a user affected by a fault
type FaultAffectedUser struct {
	User  string `json:"user"`  // Email or user identifier
	Count int    `json:"count"` // Number of occurrences for this user
}

// FaultListAffectedUsersOptions represents options for listing affected users for a fault
type FaultListAffectedUsersOptions struct {
	Q string // Search string
}

// ListAffectedUsers returns a list of users affected by a specific fault.
//
// Honeybadger API docs: https://docs.honeybadger.io/api/faults/#get-a-list-of-affected-users
//
// GET /v2/projects/{projectID}/faults/{faultID}/affected_users
func (f *FaultsService) ListAffectedUsers(ctx context.Context, projectID, faultID int, options FaultListAffectedUsersOptions) ([]FaultAffectedUser, error) {
	path := fmt.Sprintf("/projects/%d/faults/%d/affected_users", projectID, faultID)

	// Build query parameters if search provided
	if options.Q != "" {
		path += fmt.Sprintf("?q=%s", url.QueryEscape(options.Q))
	}

	req, err := f.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var users []FaultAffectedUser
	if err := f.client.do(ctx, req, &users); err != nil {
		return nil, err
	}

	return users, nil
}

// FaultCountsEnvironment represents fault counts grouped by environment with resolution and ignored status
type FaultCountsEnvironment struct {
	Environment string `json:"environment"` // Environment name (e.g., "production", "staging")
	Resolved    bool   `json:"resolved"`    // Whether these faults are resolved
	Ignored     bool   `json:"ignored"`     // Whether these faults are ignored
	Count       int    `json:"count"`       // Number of faults in this group
}

// FaultCounts represents fault count statistics for a project
type FaultCounts struct {
	Total        int                      `json:"total"`        // Total count of all faults
	Environments []FaultCountsEnvironment `json:"environments"` // Counts grouped by environment, resolution, and ignored status
}

// GetCounts returns fault count statistics for a project with optional filtering.
//
// Honeybadger API docs: https://docs.honeybadger.io/api/faults/#get-a-count-of-faults
//
// GET /v2/projects/{projectID}/faults/summary
func (f *FaultsService) GetCounts(ctx context.Context, projectID int, options FaultListOptions) (*FaultCounts, error) {
	path := fmt.Sprintf("/projects/%d/faults/summary", projectID)

	// Build query parameters using url.Values (reuse same filtering options as List)
	params := url.Values{}
	if options.Q != "" {
		params.Set("q", options.Q)
	}
	if options.CreatedAfter > 0 {
		params.Set("created_after", strconv.FormatInt(options.CreatedAfter, 10))
	}
	if options.OccurredAfter > 0 {
		params.Set("occurred_after", strconv.FormatInt(options.OccurredAfter, 10))
	}
	if options.OccurredBefore > 0 {
		params.Set("occurred_before", strconv.FormatInt(options.OccurredBefore, 10))
	}

	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	req, err := f.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var counts FaultCounts
	if err := f.client.do(ctx, req, &counts); err != nil {
		return nil, err
	}

	return &counts, nil
}
