package apiv3

import (
	"context"
	"errors"
	"net/http"
	"strings"

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

// FaultSelection chooses which faults a bulk state change applies to.
//
// Its fields are unexported and it is built through the three constructors
// below, so the three intents stay distinct: named faults, faults matching a
// search, or every fault in the project. A struct with both ids and a query set
// is not constructible, which matters because the spec and the app disagree about
// what that would mean — the spec says the query is ignored when ids are present,
// while the Rails side filters by query first and applies the ids to the result.
// A destructive operation should not depend on which of those is true.
//
// The zero value is not a valid selection: see ErrEveryFault.
type FaultSelection struct {
	ids   []string
	query string
	all   bool
}

// SelectFaults changes the named faults. Ids outside the project select nothing.
func SelectFaults(ids ...string) FaultSelection {
	return FaultSelection{ids: ids}
}

// SelectFaultsMatching changes every fault the search matches, in the same syntax
// as the fault listing's Search.
func SelectFaultsMatching(query string) FaultSelection {
	return FaultSelection{query: query}
}

// SelectAllFaults changes every fault in the project.
//
// This sends no filter at all, which is what the endpoint treats as "everything".
// A wildcard query is not the same thing: the search runs against notices, so a
// fault whose notices are not searchable would be missed by "*" and caught here.
func SelectAllFaults() FaultSelection {
	return FaultSelection{all: true}
}

// ErrEveryFault is returned when a bulk change would apply to the whole project
// without having asked to.
//
// The request body is optional and an absent one means "everything the filters
// match", so the zero-value selection is not a no-op: it resolves, ignores or
// unignores every fault in the project. That is a plausible typo and an
// implausible intention, so it is refused here rather than sent. Use
// SelectAllFaults when the whole project really is the intent.
var ErrEveryFault = errors.New(
	"apiv3: a bulk fault change with no ids and no query applies to every fault in the project — " +
		"name the faults with SelectFaults, filter them with SelectFaultsMatching, or say so " +
		"with SelectAllFaults")

// body renders the selection, or refuses an unbounded one.
func (sel FaultSelection) body() (*gen.ResolveFaultsJSONRequestBody, error) {
	switch {
	case len(sel.ids) > 0:
		ids := sel.ids
		return &gen.ResolveFaultsJSONRequestBody{FaultIds: &ids}, nil
	case strings.TrimSpace(sel.query) != "":
		query := strings.TrimSpace(sel.query)
		return &gen.ResolveFaultsJSONRequestBody{Q: &query}, nil
	case sel.all:
		return &gen.ResolveFaultsJSONRequestBody{}, nil
	}
	return nil, ErrEveryFault
}

// The four bulk endpoints share one body schema, so they share one Go type: the
// generated Unresolve/Ignore/Unignore bodies are structurally identical and
// convertible.

// Resolve marks faults as resolved.
func (s *FaultsService) Resolve(ctx context.Context, projectID string, sel FaultSelection, opts ...Option) error {
	ro := resolve(opts)
	body, err := sel.body()
	if err != nil {
		return err
	}
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().ResolveFaults(ctx, s.client.accountID(ro.accountID), projectID, *body)
	})
}

// Unresolve returns faults to the unresolved state.
func (s *FaultsService) Unresolve(ctx context.Context, projectID string, sel FaultSelection, opts ...Option) error {
	ro := resolve(opts)
	body, err := sel.body()
	if err != nil {
		return err
	}
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().UnresolveFaults(ctx, s.client.accountID(ro.accountID), projectID,
			gen.UnresolveFaultsJSONRequestBody(*body))
	})
}

// Ignore marks faults as ignored, which also stops collecting data for them.
func (s *FaultsService) Ignore(ctx context.Context, projectID string, sel FaultSelection, opts ...Option) error {
	ro := resolve(opts)
	body, err := sel.body()
	if err != nil {
		return err
	}
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().IgnoreFaults(ctx, s.client.accountID(ro.accountID), projectID,
			gen.IgnoreFaultsJSONRequestBody(*body))
	})
}

// Unignore stops ignoring faults.
func (s *FaultsService) Unignore(ctx context.Context, projectID string, sel FaultSelection, opts ...Option) error {
	ro := resolve(opts)
	body, err := sel.body()
	if err != nil {
		return err
	}
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().UnignoreFaults(ctx, s.client.accountID(ro.accountID), projectID,
			gen.UnignoreFaultsJSONRequestBody(*body))
	})
}

// FaultMerge is a merge accepted for background processing.
type FaultMerge struct {
	// BatchID identifies the background merge.
	BatchID string `json:"batch_id"`

	// SourceID is the fault that was merged away.
	SourceID string `json:"source_id"`

	// TargetID is the fault that was kept.
	TargetID string `json:"target_id"`
}

// ErrMergeIntoSelf is returned when both fault ids are the same.
var ErrMergeIntoSelf = errors.New("apiv3: a fault cannot be merged into itself")

// Merge folds sourceFaultID into targetFaultID.
//
// The source is destroyed: its notices move to the target and the fault itself is
// removed. Passing the two the wrong way round therefore deletes the fault the
// caller meant to keep, and nothing about the request would look wrong.
//
// The merge runs in the background, so a successful call means accepted, not
// done, and the returned ids let a caller confirm the direction the API applied.
func (s *FaultsService) Merge(ctx context.Context, projectID, sourceFaultID, targetFaultID string, opts ...Option) (*FaultMerge, error) {
	if sourceFaultID == targetFaultID {
		return nil, ErrMergeIntoSelf
	}
	ro := resolve(opts)
	body := gen.MergeFaultsJSONRequestBody{TargetFaultId: targetFaultID}
	return getOne[FaultMerge](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().MergeFaults(ctx, s.client.accountID(ro.accountID), projectID, sourceFaultID, body)
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
