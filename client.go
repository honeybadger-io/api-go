package honeybadgerapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL     string
	apiToken    string
	bearerToken string
	httpClient  *http.Client

	// Resource services
	Projects     *ProjectsService
	Faults       *FaultsService
	Insights     *InsightsService
	Dashboards   *DashboardsService
	Alarms       *AlarmsService
	Comments     *CommentsService
	Deployments  *DeploymentsService
	CheckIns     *CheckInsService
	Uptime       *UptimeService
	Teams        *TeamsService
	Environments *EnvironmentsService
	Accounts     *AccountsService
	StatusPages  *StatusPagesService
}

func NewClient() *Client {
	c := &Client{
		baseURL: "https://api.honeybadger.io", // Default base URL
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Initialize resource services
	c.Projects = &ProjectsService{client: c}
	c.Faults = &FaultsService{client: c}
	c.Insights = &InsightsService{client: c}
	c.Dashboards = &DashboardsService{client: c}
	c.Alarms = &AlarmsService{client: c}
	c.Comments = &CommentsService{client: c}
	c.Deployments = &DeploymentsService{client: c}
	c.CheckIns = &CheckInsService{client: c}
	c.Uptime = &UptimeService{client: c}
	c.Teams = &TeamsService{client: c}
	c.Environments = &EnvironmentsService{client: c}
	c.Accounts = &AccountsService{client: c}
	c.StatusPages = &StatusPagesService{client: c}
	return c
}

// WithAuthToken sets a Personal Auth Token, sent as HTTP Basic auth.
func (c *Client) WithAuthToken(apiToken string) *Client {
	c.apiToken = apiToken
	return c
}

// WithBearerToken sets an OAuth access token, sent as Authorization: Bearer.
// Takes precedence over WithAuthToken when both are set.
func (c *Client) WithBearerToken(token string) *Client {
	c.bearerToken = token
	return c
}

// WithBaseURL sets the base URL for the client
func (c *Client) WithBaseURL(baseURL string) *Client {
	c.baseURL = baseURL
	return c
}

// WithHTTPClient sets a custom HTTP client
func (c *Client) WithHTTPClient(httpClient *http.Client) *Client {
	c.httpClient = httpClient
	return c
}

// ProjectsAPI returns the projects service
func (c *Client) ProjectsAPI() *ProjectsService {
	return c.Projects
}

func (c *Client) newRequest(ctx context.Context, method, path string, body interface{}) (*http.Request, error) {
	// Add /v2 prefix to all paths
	url := fmt.Sprintf("%s/v2%s", c.baseURL, path)

	var buf io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		buf = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearerToken)
	} else if c.apiToken != "" {
		req.SetBasicAuth(c.apiToken, "")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return req, nil
}

func (c *Client) do(ctx context.Context, req *http.Request, v interface{}) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Check if the error is due to context cancellation
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return WrapError(resp, nil)
	}

	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}
