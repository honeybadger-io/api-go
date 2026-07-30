package apiv3

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// captureWrite records what a write actually put on the wire, which is the only
// thing worth asserting about a request body assembled from a partial schema.
type captured struct {
	method string
	path   string
	body   map[string]any
}

func captureWrite(t *testing.T, status int, response string) (*Client, *captured) {
	t.Helper()
	got := &captured{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.method, got.path = r.Method, r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &got.body)
		}
		if response == "" {
			w.WriteHeader(status)
			return
		}
		writeJSON(w, status, response)
	}))
	t.Cleanup(srv.Close)
	return NewClient().WithBaseURL(srv.URL).WithBearerToken("hbt_x"), got
}

func TestFaultsResolveSendsFaultIDs(t *testing.T) {
	c, got := captureWrite(t, http.StatusNoContent, "")

	if err := c.Faults.Resolve(context.Background(), "Xk9mZp", []string{"f1", "f2"}); err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if got.method != http.MethodPost {
		t.Errorf("method = %q, want POST", got.method)
	}
	if want := "/v3/accounts/me/projects/Xk9mZp/faults/resolve"; got.path != want {
		t.Errorf("path = %q, want %q", got.path, want)
	}
	ids, ok := got.body["fault_ids"].([]any)
	if !ok || len(ids) != 2 || ids[0] != "f1" {
		t.Errorf("fault_ids = %v", got.body["fault_ids"])
	}
}

// A 204 carries no body, so it must not be treated as a malformed envelope.
func TestDeleteAcceptsNoContent(t *testing.T) {
	c, got := captureWrite(t, http.StatusNoContent, "")

	if err := c.Faults.Delete(context.Background(), "Xk9mZp", "f1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got.method != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", got.method)
	}
}

func TestPauseAndResumeRecordingSendNoBody(t *testing.T) {
	for _, tc := range []struct {
		name string
		call func(*Client) error
		path string
	}{
		{"pause", func(c *Client) error {
			return c.Faults.PauseRecording(context.Background(), "Xk9mZp", "f1")
		}, "/v3/accounts/me/projects/Xk9mZp/faults/f1/pause_recording"},
		{"resume", func(c *Client) error {
			return c.Faults.ResumeRecording(context.Background(), "Xk9mZp", "f1")
		}, "/v3/accounts/me/projects/Xk9mZp/faults/f1/resume_recording"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, got := captureWrite(t, http.StatusNoContent, "")
			if err := tc.call(c); err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			if got.path != tc.path {
				t.Errorf("path = %q, want %q", got.path, tc.path)
			}
			if got.body != nil {
				t.Errorf("body = %v, want none", got.body)
			}
		})
	}
}

// An account token can hold faults:write and still be refused, because a comment
// attributes text to a person.
func TestAddCommentRequiresUserToken(t *testing.T) {
	c, _ := captureWrite(t, http.StatusForbidden,
		`{"error":{"code":"requires_user_token","message":"This endpoint records the person who acted"}}`)

	err := c.Faults.AddComment(context.Background(), "Xk9mZp", "f1", "looking into it")
	if !errors.Is(err, ErrRequiresUserToken) {
		t.Fatalf("err = %v, want ErrRequiresUserToken", err)
	}
}

func TestProjectsCreateSendsRequiredName(t *testing.T) {
	c, got := captureWrite(t, http.StatusCreated,
		`{"data":{"id":"Xk9mZp","account_id":"Ab3kL9","name":"New App","active":true}}`)

	p, err := c.Projects.Create(context.Background(), "New App")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.body["name"] != "New App" {
		t.Errorf("name = %v", got.body["name"])
	}
	if p.Id != "Xk9mZp" {
		t.Errorf("returned project = %+v", p)
	}
}

// An update omits what it was not given, so unset fields are left alone rather
// than blanked.
func TestCheckInUpdateOmitsUnsetFields(t *testing.T) {
	c, got := captureWrite(t, http.StatusOK, `{"data":{"id":"c1","name":"Nightly"}}`)

	if _, err := c.CheckIns.Update(context.Background(), "Xk9mZp", "c1",
		CheckInParams{GracePeriod: "5m"}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if got.body["grace_period"] != "5m" {
		t.Errorf("grace_period = %v", got.body["grace_period"])
	}
	for _, absent := range []string{"name", "schedule_type", "report_period"} {
		if _, present := got.body[absent]; present {
			t.Errorf("%q was sent despite being unset; it would blank the field", absent)
		}
	}
}

func TestCheckInCreateSendsSpecifiedFields(t *testing.T) {
	c, got := captureWrite(t, http.StatusCreated, `{"data":{"id":"c1","name":"Nightly"}}`)

	_, err := c.CheckIns.Create(context.Background(), "Xk9mZp", CheckInParams{
		Name:         "Nightly",
		ScheduleType: "simple",
		ReportPeriod: "1d",
		GracePeriod:  "1h",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	for key, want := range map[string]any{
		"name": "Nightly", "schedule_type": "simple",
		"report_period": "1d", "grace_period": "1h",
	} {
		if got.body[key] != want {
			t.Errorf("%s = %v, want %v", key, got.body[key], want)
		}
	}
}

// A validation error must arrive typed, with its field details intact — this is
// the array branch of the details oneOf.
func TestWriteValidationErrorCarriesFieldDetails(t *testing.T) {
	c, _ := captureWrite(t, http.StatusUnprocessableEntity,
		`{"error":{"code":"validation_error","message":"Name can't be blank",
		  "details":[{"field":"name","message":"can't be blank"}]},
		  "meta":{"request_id":"req_v"}}`)

	_, err := c.Projects.Create(context.Background(), "")
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}

	var apiErr *Error
	if !asError(err, &apiErr) {
		t.Fatal("not an *apiv3.Error")
	}
	if len(apiErr.FieldErrors) != 1 {
		t.Fatalf("FieldErrors = %v, want 1", apiErr.FieldErrors)
	}
	if apiErr.FieldErrors[0].Field != "name" {
		t.Errorf("field = %q, want name", apiErr.FieldErrors[0].Field)
	}
	// The rendered message should name the offending field.
	if !strings.Contains(apiErr.Error(), "name: can't be blank") {
		t.Errorf("Error() = %q", apiErr.Error())
	}
}

// A write refused for scope must still name the scope needed.
func TestWriteInsufficientScopeNamesScope(t *testing.T) {
	c, _ := captureWrite(t, http.StatusForbidden,
		`{"error":{"code":"insufficient_scope","message":"Insufficient scope",
		  "details":{"required_scope":"faults:write","token_scopes":["faults:read"]}}}`)

	err := c.Faults.Ignore(context.Background(), "Xk9mZp", []string{"f1"})
	var apiErr *Error
	if !asError(err, &apiErr) {
		t.Fatalf("err = %T, want *apiv3.Error", err)
	}
	if apiErr.RequiredScope() != "faults:write" {
		t.Errorf("RequiredScope() = %q", apiErr.RequiredScope())
	}
}
