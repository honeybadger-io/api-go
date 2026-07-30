package apiv3

import (
	"context"
	"net/http"

	"github.com/honeybadger-io/api-go/internal/gen"
)

// Fault state changes.
//
// v3 replaced v2's single mutable PUT — which took resolved, ignored, and
// assignee together — with one endpoint per action. Each takes a list of fault
// ids, so a bulk change is one request rather than one per fault.
//
// Two fault writes are deliberately absent: Update and Assign. Both declare
// their request body as a bare `type: object` in the spec, so there is nothing to
// build a typed method from, and guessing field names would produce a method
// that compiles and silently does nothing. See openapi/README.md.

// Resolve marks faults as resolved.
func (s *FaultsService) Resolve(ctx context.Context, projectID string, faultIDs []string, opts ...Option) error {
	ro := resolve(opts)
	body := gen.ResolveFaultsJSONRequestBody{FaultIds: faultIDs}
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().ResolveFaults(ctx, s.client.accountID(ro.accountID), projectID, body)
	})
}

// Unresolve returns faults to the unresolved state.
func (s *FaultsService) Unresolve(ctx context.Context, projectID string, faultIDs []string, opts ...Option) error {
	ro := resolve(opts)
	body := gen.UnresolveFaultsJSONRequestBody{FaultIds: faultIDs}
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().UnresolveFaults(ctx, s.client.accountID(ro.accountID), projectID, body)
	})
}

// Ignore marks faults as ignored, suppressing their notifications.
func (s *FaultsService) Ignore(ctx context.Context, projectID string, faultIDs []string, opts ...Option) error {
	ro := resolve(opts)
	body := gen.IgnoreFaultsJSONRequestBody{FaultIds: faultIDs}
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().IgnoreFaults(ctx, s.client.accountID(ro.accountID), projectID, body)
	})
}

// Unignore stops ignoring faults.
func (s *FaultsService) Unignore(ctx context.Context, projectID string, faultIDs []string, opts ...Option) error {
	ro := resolve(opts)
	body := gen.UnignoreFaultsJSONRequestBody{FaultIds: faultIDs}
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().UnignoreFaults(ctx, s.client.accountID(ro.accountID), projectID, body)
	})
}

// Merge folds the source faults into the target fault.
func (s *FaultsService) Merge(ctx context.Context, projectID, targetFaultID string, sourceFaultIDs []string, opts ...Option) error {
	ro := resolve(opts)
	body := gen.MergeFaultsJSONRequestBody{SourceFaultIds: sourceFaultIDs}
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().MergeFaults(ctx, s.client.accountID(ro.accountID), projectID, targetFaultID, body)
	})
}

// PauseRecording stops recording new notices for a fault.
func (s *FaultsService) PauseRecording(ctx context.Context, projectID, faultID string, opts ...Option) error {
	ro := resolve(opts)
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().PauseFaultRecording(ctx, s.client.accountID(ro.accountID), projectID, faultID)
	})
}

// ResumeRecording starts recording notices for a fault again.
func (s *FaultsService) ResumeRecording(ctx context.Context, projectID, faultID string, opts ...Option) error {
	ro := resolve(opts)
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().ResumeFaultRecording(ctx, s.client.accountID(ro.accountID), projectID, faultID)
	})
}

// Delete removes a fault and its notices.
func (s *FaultsService) Delete(ctx context.Context, projectID, faultID string, opts ...Option) error {
	ro := resolve(opts)
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().DeleteFault(ctx, s.client.accountID(ro.accountID), projectID, faultID)
	})
}

// AddComment attaches a comment to a fault.
//
// Commenting attributes text to a person, so an account token holding
// faults:write is still refused with requires_user_token — there is nobody to
// attribute it to. Check errors.Is(err, ErrRequiresUserToken).
func (s *FaultsService) AddComment(ctx context.Context, projectID, faultID, comment string, opts ...Option) error {
	ro := resolve(opts)
	body := gen.CreateCommentJSONRequestBody{Body: comment}
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().CreateComment(ctx, s.client.accountID(ro.accountID), projectID, faultID, body)
	})
}
