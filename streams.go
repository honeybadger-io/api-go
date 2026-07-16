package honeybadgerapi

import (
	"context"
	"fmt"
)

// StreamsService provides methods for interacting with Insights streams
type StreamsService struct {
	client *Client
}

// List retrieves all streams for a project
func (s *StreamsService) List(ctx context.Context, projectID int) ([]Stream, error) {
	path := fmt.Sprintf("/projects/%d/streams", projectID)

	req, err := s.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response StreamListResponse
	if err := s.client.do(ctx, req, &response); err != nil {
		return nil, err
	}

	return response.Results, nil
}
