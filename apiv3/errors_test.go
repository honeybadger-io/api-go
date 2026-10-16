package apiv3

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestParseErrorBasicFields(t *testing.T) {
	body := []byte(`{"error":{"code":"not_found","message":"Resource not found"},"meta":{"request_id":"req_9"}}`)
	err := parseError(http.StatusNotFound, body)

	if err.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want 404", err.StatusCode)
	}
	if err.Code != CodeNotFound {
		t.Errorf("Code = %q, want %q", err.Code, CodeNotFound)
	}
	if err.Message != "Resource not found" {
		t.Errorf("Message = %q", err.Message)
	}
	if err.RequestID != "req_9" {
		t.Errorf("RequestID = %q, want req_9", err.RequestID)
	}
	if want := "honeybadger: 404 not_found: Resource not found (request req_9)"; err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
}

func TestErrorsIsMatchesSentinels(t *testing.T) {
	tests := []struct {
		code     string
		sentinel error
	}{
		{"not_found", ErrNotFound},
		{"unauthorized", ErrUnauthorized},
		{"access_denied", ErrAccessDenied},
		{"validation_error", ErrValidation},
		{"rate_limit_exceeded", ErrRateLimited},
		{"ambiguous_account", ErrAmbiguousAccount},
		{"insufficient_scope", ErrInsufficientScope},
		{"credential_in_query", ErrCredentialInQuery},
		{"project_restricted", ErrProjectRestricted},
		{"service_unavailable", ErrServiceUnavailable},
	}
	for _, tt := range tests {
		err := parseError(400, []byte(`{"error":{"code":"`+tt.code+`","message":"m"}}`))
		if !errors.Is(err, tt.sentinel) {
			t.Errorf("code %q: errors.Is(err, %v) = false, want true", tt.code, tt.sentinel)
		}
		if errors.Is(err, ErrMaintenanceMode) && tt.code != "maintenance_mode" {
			t.Errorf("code %q matched ErrMaintenanceMode", tt.code)
		}
	}
}

// Three codes the API documents are missing from the bundle's enum, so apiv3
// treats Code as an open string. An unrecognized code must still round-trip.
func TestUnknownCodeIsPreserved(t *testing.T) {
	err := parseError(418, []byte(`{"error":{"code":"brand_new_code","message":"m"}}`))
	if err.Code != "brand_new_code" {
		t.Errorf("Code = %q, want the raw value preserved", err.Code)
	}
	if errors.Is(err, ErrNotFound) {
		t.Error("unknown code matched an unrelated sentinel")
	}
}

// Spec bug: details is declared as an array of {field, message} but the
// InsufficientScope response uses an object. apiv3 accepts both.
func TestParseErrorDetailsArrayForm(t *testing.T) {
	body := []byte(`{"error":{"code":"validation_error","message":"Name can't be blank",
	  "details":[{"field":"name","message":"can't be blank"}]}}`)
	err := parseError(422, body)

	if len(err.FieldErrors) != 1 {
		t.Fatalf("FieldErrors = %v, want 1 entry", err.FieldErrors)
	}
	if err.FieldErrors[0].Field != "name" || err.FieldErrors[0].Message != "can't be blank" {
		t.Errorf("FieldErrors[0] = %+v", err.FieldErrors[0])
	}
}

func TestParseErrorDetailsObjectForm(t *testing.T) {
	body := []byte(`{"error":{"code":"insufficient_scope","message":"Insufficient scope",
	  "details":{"required_scope":"faults:write","token_scopes":["faults:read","insights:read"]}}}`)
	err := parseError(http.StatusForbidden, body)

	if err.RequiredScope() != "faults:write" {
		t.Errorf("RequiredScope() = %q, want faults:write", err.RequiredScope())
	}
	got := err.TokenScopes()
	if len(got) != 2 || got[0] != "faults:read" || got[1] != "insights:read" {
		t.Errorf("TokenScopes() = %v", got)
	}
	if len(err.FieldErrors) != 0 {
		t.Errorf("FieldErrors should be empty for the object form, got %v", err.FieldErrors)
	}
}

// An insufficient_scope error should say what to grant. That text is the whole
// point of the spec carrying required_scope.
func TestInsufficientScopeErrorMessageNamesTheScope(t *testing.T) {
	body := []byte(`{"error":{"code":"insufficient_scope","message":"Insufficient scope",
	  "details":{"required_scope":"checkins:write","token_scopes":["faults:read"]}}}`)
	err := parseError(http.StatusForbidden, body)

	msg := err.Error()
	for _, want := range []string{"checkins:write", "faults:read"} {
		if !strings.Contains(msg, want) {
			t.Errorf("Error() = %q, want it to mention %q", msg, want)
		}
	}
}

// A body that is not the documented envelope must not produce an empty error.
func TestParseErrorFallsBackForNonEnvelopeBodies(t *testing.T) {
	err := parseError(http.StatusBadGateway, []byte("<html>502 Bad Gateway</html>"))
	if err.StatusCode != http.StatusBadGateway {
		t.Errorf("StatusCode = %d", err.StatusCode)
	}
	if err.Message == "" {
		t.Error("Message is empty; a non-JSON body must still yield something readable")
	}
	if !strings.Contains(err.Error(), "502") {
		t.Errorf("Error() = %q, want it to mention the status", err.Error())
	}
}

func TestParseErrorEmptyBody(t *testing.T) {
	err := parseError(http.StatusNotFound, nil)
	if err.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %d", err.StatusCode)
	}
	if err.Message == "" {
		t.Error("Message is empty for a bodyless error response")
	}
}

// Rate limiting needs the reset time, so callers can say when to retry rather
// than blocking.
func TestRateLimitErrorCarriesReset(t *testing.T) {
	body := []byte(`{"error":{"code":"rate_limit_exceeded","message":"Rate limit exceeded"}}`)
	err := parseError(http.StatusTooManyRequests, body)
	err.RateLimit = &RateLimit{Limit: 360, Remaining: 0, Reset: fixedReset}

	if !errors.Is(err, ErrRateLimited) {
		t.Fatal("errors.Is(err, ErrRateLimited) = false")
	}
	if err.RateLimit.Remaining != 0 {
		t.Errorf("Remaining = %d, want 0", err.RateLimit.Remaining)
	}
	if err.RateLimit.Reset != fixedReset {
		t.Errorf("Reset = %v, want %v", err.RateLimit.Reset, fixedReset)
	}
}

var fixedReset = time.Unix(1784000000, 0)
