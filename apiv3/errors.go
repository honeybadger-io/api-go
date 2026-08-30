package apiv3

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Code is a machine-readable v3 error code.
//
// Deliberately an open string rather than the generated enum. The bundle's enum
// has grown repeatedly during v3's development, and a closed set would drop any
// code the client has not caught up with — silently turning a real API error
// into an unrecognised one. Compare against the Code* constants, or use
// errors.Is with the Err* sentinels.
type Code string

// The codes v3 declares. Auth and authorization first, then the rest, matching
// the spec's own ordering.
const (
	CodeUnauthorized          Code = "unauthorized"
	CodeUnsupportedAuthScheme Code = "unsupported_auth_scheme"
	CodeCredentialInQuery     Code = "credential_in_query"
	CodeAccessDenied          Code = "access_denied"
	CodeInsufficientScope     Code = "insufficient_scope"
	CodeRequiresUserToken     Code = "requires_user_token"
	CodeProjectRestricted     Code = "project_restricted"
	CodeAccountInactive       Code = "account_inactive"
	CodeAccountParked         Code = "account_parked"
	CodeFeatureUnavailable    Code = "feature_unavailable"
	CodeNotFound              Code = "not_found"
	CodeValidationError       Code = "validation_error"
	CodeDeleteFailed          Code = "delete_failed"
	CodeLimitReached          Code = "limit_reached"
	CodeRateLimitExceeded     Code = "rate_limit_exceeded"
	CodeMaintenanceMode       Code = "maintenance_mode"
	CodeInvalidID             Code = "invalid_id"
	CodeForbiddenAttributes   Code = "forbidden_attributes"
	CodeServiceUnavailable    Code = "service_unavailable"
	CodeAmbiguousAccount      Code = "ambiguous_account"
)

// Sentinels for errors.Is. Each matches any Error carrying the same code.
var (
	ErrUnauthorized          = &Error{Code: CodeUnauthorized}
	ErrUnsupportedAuthScheme = &Error{Code: CodeUnsupportedAuthScheme}
	ErrCredentialInQuery     = &Error{Code: CodeCredentialInQuery}
	ErrAccessDenied          = &Error{Code: CodeAccessDenied}
	ErrInsufficientScope     = &Error{Code: CodeInsufficientScope}
	ErrRequiresUserToken     = &Error{Code: CodeRequiresUserToken}
	ErrProjectRestricted     = &Error{Code: CodeProjectRestricted}
	ErrAccountInactive       = &Error{Code: CodeAccountInactive}
	ErrAccountParked         = &Error{Code: CodeAccountParked}
	ErrFeatureUnavailable    = &Error{Code: CodeFeatureUnavailable}
	ErrNotFound              = &Error{Code: CodeNotFound}
	ErrValidation            = &Error{Code: CodeValidationError}
	ErrDeleteFailed          = &Error{Code: CodeDeleteFailed}
	ErrLimitReached          = &Error{Code: CodeLimitReached}
	ErrRateLimited           = &Error{Code: CodeRateLimitExceeded}
	ErrMaintenanceMode       = &Error{Code: CodeMaintenanceMode}
	ErrInvalidID             = &Error{Code: CodeInvalidID}
	ErrForbiddenAttributes   = &Error{Code: CodeForbiddenAttributes}
	ErrServiceUnavailable    = &Error{Code: CodeServiceUnavailable}
	ErrAmbiguousAccount      = &Error{Code: CodeAmbiguousAccount}
)

// FieldError is one entry from a validation error's details.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error is a v3 API error.
type Error struct {
	StatusCode int
	Code       Code
	Message    string
	RequestID  string

	// FieldErrors holds details when it arrives as an array, which is what
	// validation errors send.
	FieldErrors []FieldError

	// Details holds details when it arrives as an object, which is what
	// insufficient_scope sends. Read it through RequiredScope and TokenScopes.
	Details map[string]any

	// RateLimit is the rate-limit snapshot from the failing response, when the
	// response carried those headers.
	RateLimit *RateLimit

	// Body is the raw response body, kept for diagnosing responses that do not
	// match the documented envelope.
	Body []byte
}

func (e *Error) Error() string {
	var b strings.Builder
	b.WriteString("honeybadger: ")
	fmt.Fprintf(&b, "%d", e.StatusCode)
	if e.Code != "" {
		b.WriteString(" " + string(e.Code))
	}
	if e.Message != "" {
		b.WriteString(": " + e.Message)
	}

	// An insufficient_scope error is only actionable if it names the scope.
	if scope := e.RequiredScope(); scope != "" {
		b.WriteString(" (requires " + scope)
		if held := e.TokenScopes(); len(held) > 0 {
			b.WriteString("; token has " + strings.Join(held, ", "))
		}
		b.WriteString(")")
	}

	for _, fe := range e.FieldErrors {
		b.WriteString(" [" + fe.Field + ": " + fe.Message + "]")
	}
	if e.RequestID != "" {
		b.WriteString(" (request " + e.RequestID + ")")
	}
	return b.String()
}

// Is reports whether target is a sentinel with the same code, so callers can
// write errors.Is(err, apiv3.ErrNotFound).
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// RequiredScope returns the scope an insufficient_scope error says is needed,
// or "" for other errors.
func (e *Error) RequiredScope() string {
	scope, _ := e.Details["required_scope"].(string)
	return scope
}

// TokenScopes returns the scopes an insufficient_scope error says the credential
// holds, or nil for other errors.
func (e *Error) TokenScopes() []string {
	raw, ok := e.Details["token_scopes"].([]any)
	if !ok {
		return nil
	}
	scopes := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			scopes = append(scopes, s)
		}
	}
	return scopes
}

// parseError builds an Error from a failing response.
//
// It tolerates both shapes of the details field. The bundle types details as an
// array of {field, message} while the insufficient_scope response sends an
// object, so neither shape can be assumed. It also tolerates bodies that are not
// the documented envelope at all — a proxy's HTML 502, for instance — because
// those are exactly the cases where a useful error matters most.
func parseError(statusCode int, body []byte) *Error {
	e := &Error{StatusCode: statusCode, Body: body}

	var envelope struct {
		Error struct {
			Code    string          `json:"code"`
			Message string          `json:"message"`
			Details json.RawMessage `json:"details"`
		} `json:"error"`
		Meta struct {
			RequestID string `json:"request_id"`
		} `json:"meta"`
	}

	if len(body) > 0 && json.Unmarshal(body, &envelope) == nil {
		e.Code = Code(envelope.Error.Code)
		e.Message = envelope.Error.Message
		e.RequestID = envelope.Meta.RequestID
		e.parseDetails(envelope.Error.Details)
	}

	if e.Message == "" {
		e.Message = fallbackMessage(statusCode, body)
	}
	return e
}

// parseDetails tries the array shape, then the object shape. Neither succeeding
// is fine: details is optional.
func (e *Error) parseDetails(raw json.RawMessage) {
	if len(raw) == 0 {
		return
	}
	var fields []FieldError
	if json.Unmarshal(raw, &fields) == nil {
		e.FieldErrors = fields
		return
	}
	var object map[string]any
	if json.Unmarshal(raw, &object) == nil {
		e.Details = object
	}
}

// fallbackMessage produces something readable when the body is missing or is not
// the documented envelope.
func fallbackMessage(statusCode int, body []byte) string {
	if text := strings.TrimSpace(string(body)); text != "" {
		const max = 200
		if len(text) > max {
			text = text[:max] + "…"
		}
		return text
	}
	if status := http.StatusText(statusCode); status != "" {
		return status
	}
	return fmt.Sprintf("unexpected status %d", statusCode)
}
