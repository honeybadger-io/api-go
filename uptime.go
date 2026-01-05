package honeybadgerapi

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// UptimeService provides methods for interacting with uptime monitoring sites
type UptimeService struct {
	client *Client
}

// List retrieves all uptime sites for a project
func (s *UptimeService) List(ctx context.Context, projectID int) ([]Site, error) {
	path := fmt.Sprintf("/projects/%d/sites", projectID)

	req, err := s.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var sites []Site
	if err := s.client.do(ctx, req, &sites); err != nil {
		return nil, err
	}

	return sites, nil
}

// Get retrieves a single uptime site by ID
func (s *UptimeService) Get(ctx context.Context, projectID int, siteID string) (*Site, error) {
	path := fmt.Sprintf("/projects/%d/sites/%s", projectID, siteID)

	req, err := s.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var site Site
	if err := s.client.do(ctx, req, &site); err != nil {
		return nil, err
	}

	return &site, nil
}

// Create creates a new uptime site
func (s *UptimeService) Create(ctx context.Context, projectID int, params SiteParams) (*Site, error) {
	path := fmt.Sprintf("/projects/%d/sites", projectID)

	reqBody := SiteCreateRequest{Site: params}

	req, err := s.client.newRequest(ctx, "POST", path, reqBody)
	if err != nil {
		return nil, err
	}

	var site Site
	if err := s.client.do(ctx, req, &site); err != nil {
		return nil, err
	}

	return &site, nil
}

// Update updates an existing uptime site
func (s *UptimeService) Update(ctx context.Context, projectID int, siteID string, params SiteParams) (*Site, error) {
	path := fmt.Sprintf("/projects/%d/sites/%s", projectID, siteID)

	reqBody := SiteCreateRequest{Site: params}

	req, err := s.client.newRequest(ctx, "PUT", path, reqBody)
	if err != nil {
		return nil, err
	}

	var site Site
	if err := s.client.do(ctx, req, &site); err != nil {
		return nil, err
	}

	return &site, nil
}

// Delete deletes an uptime site
func (s *UptimeService) Delete(ctx context.Context, projectID int, siteID string) error {
	path := fmt.Sprintf("/projects/%d/sites/%s", projectID, siteID)

	req, err := s.client.newRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}

// ListOutages retrieves outages for a specific site
func (s *UptimeService) ListOutages(ctx context.Context, projectID int, siteID string, options OutageListOptions) ([]Outage, error) {
	path := fmt.Sprintf("/projects/%d/sites/%s/outages", projectID, siteID)

	// Build query parameters
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

	req, err := s.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var outages []Outage
	if err := s.client.do(ctx, req, &outages); err != nil {
		return nil, err
	}

	return outages, nil
}

// ListUptimeChecks retrieves uptime checks for a specific site
func (s *UptimeService) ListUptimeChecks(ctx context.Context, projectID int, siteID string, options UptimeCheckListOptions) ([]UptimeCheck, error) {
	path := fmt.Sprintf("/projects/%d/sites/%s/uptime_checks", projectID, siteID)

	// Build query parameters
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

	req, err := s.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var checks []UptimeCheck
	if err := s.client.do(ctx, req, &checks); err != nil {
		return nil, err
	}

	return checks, nil
}
