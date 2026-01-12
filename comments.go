package honeybadgerapi

import (
	"context"
	"fmt"
)

// CommentsService provides methods for interacting with comments
type CommentsService struct {
	client *Client
}

// List retrieves all comments for a specific fault
func (s *CommentsService) List(ctx context.Context, projectID, faultID int) ([]Comment, error) {
	path := fmt.Sprintf("/projects/%d/faults/%d/comments", projectID, faultID)

	req, err := s.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response CommentListResponse
	if err := s.client.do(ctx, req, &response); err != nil {
		return nil, err
	}

	return response.Results, nil
}

// Get retrieves a single comment by ID
func (s *CommentsService) Get(ctx context.Context, projectID, faultID, commentID int) (*Comment, error) {
	path := fmt.Sprintf("/projects/%d/faults/%d/comments/%d", projectID, faultID, commentID)

	req, err := s.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var comment Comment
	if err := s.client.do(ctx, req, &comment); err != nil {
		return nil, err
	}

	return &comment, nil
}

// Create creates a new comment on a fault
func (s *CommentsService) Create(ctx context.Context, projectID, faultID int, body string) (*Comment, error) {
	path := fmt.Sprintf("/projects/%d/faults/%d/comments", projectID, faultID)

	reqBody := CommentRequest{}
	reqBody.Comment.Body = body

	req, err := s.client.newRequest(ctx, "POST", path, reqBody)
	if err != nil {
		return nil, err
	}

	var comment Comment
	if err := s.client.do(ctx, req, &comment); err != nil {
		return nil, err
	}

	return &comment, nil
}

// Update updates an existing comment
func (s *CommentsService) Update(ctx context.Context, projectID, faultID, commentID int, body string) error {
	path := fmt.Sprintf("/projects/%d/faults/%d/comments/%d", projectID, faultID, commentID)

	reqBody := CommentRequest{}
	reqBody.Comment.Body = body

	req, err := s.client.newRequest(ctx, "PUT", path, reqBody)
	if err != nil {
		return err
	}

	// Update returns 204 No Content.
	return s.client.do(ctx, req, nil)
}

// Delete deletes a comment
func (s *CommentsService) Delete(ctx context.Context, projectID, faultID, commentID int) error {
	path := fmt.Sprintf("/projects/%d/faults/%d/comments/%d", projectID, faultID, commentID)

	req, err := s.client.newRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}
