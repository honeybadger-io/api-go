package apiv3

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

const commentJSON = `{"id":"cmt_1","fault_id":"1","body":"looking into it","created_at":"2026-09-26T00:00:00Z","author":{"name":"Kevin"}}`

// Comments page by time, so ListAll follows links.older to the end.
func TestListAllCommentsFollowsOlderLinks(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/v3/projects/Xk9mZp/faults/1/comments":
			writeJSON(w, 0, `{"data":[`+commentJSON+`],
			  "pagination":{"has_older":true,"limit":1},
			  "links":{"self":"http://`+r.Host+`/v3/self","older":"http://`+r.Host+`/v3/comments/older"}}`)
		case "/v3/comments/older":
			writeJSON(w, 0, `{"data":[{"id":"cmt_0","fault_id":"1","created_at":"2026-09-25T00:00:00Z"}],
			  "pagination":{"has_older":false,"limit":1},
			  "links":{"self":"http://`+r.Host+`/v3/comments/older"}}`)
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
		}
	}))
	defer srv.Close()

	c := NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x")
	all, err := c.Faults.ListAllComments(context.Background(), "Xk9mZp", "1", Limit(1))
	if err != nil {
		t.Fatalf("ListAllComments: %v", err)
	}
	if len(all) != 2 || all[0].Id != "cmt_1" || all[1].Id != "cmt_0" {
		t.Fatalf("got %+v, want both comments newest first", all)
	}
}

// An update answers with the comment as stored.
func TestUpdateCommentReturnsTheComment(t *testing.T) {
	c, got := captureWrite(t, http.StatusOK, `{"data":`+commentJSON+`}`)

	comment, err := c.Faults.UpdateComment(context.Background(), "Xk9mZp", "1", "cmt_1", "looking into it")
	if err != nil {
		t.Fatalf("UpdateComment: %v", err)
	}
	if got.method != http.MethodPatch && got.method != http.MethodPut {
		t.Errorf("method = %q, want PATCH or PUT", got.method)
	}
	if want := "/v3/projects/Xk9mZp/faults/1/comments/cmt_1"; got.path != want {
		t.Errorf("path = %q, want %q", got.path, want)
	}
	if got.body["body"] != "looking into it" {
		t.Errorf("sent body = %v", got.body)
	}
	if comment.Id != "cmt_1" {
		t.Errorf("comment = %+v", comment)
	}
}

func TestGetAndDeleteComment(t *testing.T) {
	c, got := captureWrite(t, http.StatusOK, `{"data":`+commentJSON+`}`)
	comment, err := c.Faults.GetComment(context.Background(), "Xk9mZp", "1", "cmt_1")
	if err != nil || comment.Id != "cmt_1" {
		t.Fatalf("GetComment = %+v, %v", comment, err)
	}
	if want := "/v3/projects/Xk9mZp/faults/1/comments/cmt_1"; got.path != want {
		t.Errorf("path = %q, want %q", got.path, want)
	}

	c2, got2 := captureWrite(t, http.StatusNoContent, "")
	if err := c2.Faults.DeleteComment(context.Background(), "Xk9mZp", "1", "cmt_1"); err != nil {
		t.Fatalf("DeleteComment: %v", err)
	}
	if got2.method != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", got2.method)
	}
}

// An account token cannot write a comment: there is no person to attribute it to.
func TestCommentWriteWithAccountTokenIsTyped(t *testing.T) {
	c, _ := captureWrite(t, http.StatusForbidden,
		`{"error":{"code":"requires_user_token","message":"This endpoint records the person who acted"}}`)
	if _, err := c.Faults.UpdateComment(context.Background(), "Xk9mZp", "1", "cmt_1", "x"); !errors.Is(err, ErrRequiresUserToken) {
		t.Fatalf("err = %v, want ErrRequiresUserToken", err)
	}
}
