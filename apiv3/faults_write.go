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
// Update and Assign exist now: their request bodies used to be bare objects in
// the spec, so there was nothing to build a typed method from.

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

// Update changes a fault's state.
//
// Prefer the single-purpose methods above for the common cases — they take a list
// of ids and change many faults in one request. This one exists for the fields
// they do not cover, chiefly tags, and for changing several attributes of one
// fault at once.
//
// AssigneeId is nullable: an explicit null unassigns, while leaving it
// unspecified changes nothing.
func (s *FaultsService) Update(ctx context.Context, projectID, faultID string, p FaultParams, opts ...Option) (*Fault, error) {
	ro := resolve(opts)
	return getOne[Fault](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().UpdateFault(ctx, s.client.accountID(ro.accountID), projectID, faultID, p)
	})
}

// Assign gives a fault to a user.
//
// The id must belong to a member of the project; one that does not is rejected
// with 422 rather than silently unassigning.
func (s *FaultsService) Assign(ctx context.Context, projectID, faultID, assigneeID string, opts ...Option) error {
	ro := resolve(opts)
	body := gen.AssignFaultJSONRequestBody{AssigneeId: assigneeID}
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().AssignFault(ctx, s.client.accountID(ro.accountID), projectID, faultID, body)
	})
}

// Unassign removes a fault's assignee.
func (s *FaultsService) Unassign(ctx context.Context, projectID, faultID string, opts ...Option) error {
	ro := resolve(opts)
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().UnassignFault(ctx, s.client.accountID(ro.accountID), projectID, faultID)
	})
}

// Summary returns fault counts for a project, honouring the same search filter as
// the fault listing.
//
// Untyped: the endpoint's payload is a counts object the spec does not pin.
func (s *FaultsService) Summary(ctx context.Context, projectID string, opts ...Option) (map[string]any, error) {
	ro := resolve(opts)
	params := &gen.GetFaultSummaryParams{}
	if ro.query != "" {
		params.Q = &ro.query
	}
	if ro.createdAfter != 0 {
		params.CreatedAfter = &ro.createdAfter
	}
	if ro.occurredAfter != 0 {
		params.OccurredAfter = &ro.occurredAfter
	}
	if ro.occurredBefore != 0 {
		params.OccurredBefore = &ro.occurredBefore
	}

	data, err := getOne[map[string]any](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().GetFaultSummary(ctx, s.client.accountID(ro.accountID), projectID, params)
	})
	if err != nil {
		return nil, err
	}
	return *data, nil
}
