package apiv3

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

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
	ids   []int
	query string
	all   bool

	// Unix seconds, zero meaning unset. The endpoint takes these as JSON numbers.
	createdAfter   float64
	occurredAfter  float64
	occurredBefore float64
}

// CreatedAfter restricts the change to faults first seen after t.
//
// Like the other filters, it applies only when no ids are named. Chainable:
//
//	apiv3.SelectFaultsMatching("is:unresolved").OccurredBefore(cutoff)
func (sel FaultSelection) CreatedAfter(t time.Time) FaultSelection {
	sel.createdAfter = float64(t.Unix())
	return sel
}

// OccurredAfter restricts the change to faults with a notice after t.
func (sel FaultSelection) OccurredAfter(t time.Time) FaultSelection {
	sel.occurredAfter = float64(t.Unix())
	return sel
}

// OccurredBefore restricts the change to faults with no notice since t.
func (sel FaultSelection) OccurredBefore(t time.Time) FaultSelection {
	sel.occurredBefore = float64(t.Unix())
	return sel
}

// filtered reports whether any filter other than an id list is set.
func (sel FaultSelection) filtered() bool {
	return strings.TrimSpace(sel.query) != "" ||
		sel.createdAfter != 0 || sel.occurredAfter != 0 || sel.occurredBefore != 0
}

// SelectFaults changes the named faults. Ids outside the project select nothing.
func SelectFaults(ids ...int) FaultSelection {
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

// ErrFilteredIDs is returned when a selection names ids and also filters.
//
// The endpoint applies its filters only when fault_ids is absent, so a request
// carrying both changes every named fault and silently ignores the filter. That
// reads as a narrowing and is not one, which on a destructive operation is worth
// refusing rather than sending.
var ErrFilteredIDs = errors.New(
	"apiv3: a fault selection names ids and also filters, but the endpoint ignores " +
		"filters when ids are present — name the faults, or filter them, not both")

// body renders the selection, or refuses one that is unbounded or contradictory.
func (sel FaultSelection) body() (*gen.ResolveFaultsJSONRequestBody, error) {
	if len(sel.ids) > 0 {
		if sel.filtered() {
			return nil, ErrFilteredIDs
		}
		ids := sel.ids
		return &gen.ResolveFaultsJSONRequestBody{FaultIds: &ids}, nil
	}

	body := &gen.ResolveFaultsJSONRequestBody{}
	if query := strings.TrimSpace(sel.query); query != "" {
		body.Q = &query
	}
	for field, value := range map[**float64]float64{
		&body.CreatedAfter:   sel.createdAfter,
		&body.OccurredAfter:  sel.occurredAfter,
		&body.OccurredBefore: sel.occurredBefore,
	} {
		if value != 0 {
			v := value
			*field = &v
		}
	}

	// A filter of any kind is a bound. Only a selection with none at all, and no
	// explicit request for the whole project, is the accident worth refusing.
	if body.Q == nil && !sel.filtered() && !sel.all {
		return nil, ErrEveryFault
	}
	return body, nil
}

// The four bulk endpoints share one body schema, so they share one Go type: the
// generated Unresolve/Ignore/Unignore bodies are structurally identical and
// convertible.

// Resolve marks faults as resolved.
func (s *FaultsService) Resolve(ctx context.Context, projectID string, sel FaultSelection, opts ...Option) error {
	body, err := sel.body()
	if err != nil {
		return err
	}
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().ResolveFaults(ctx, projectID, *body)
	})
}

// Unresolve returns faults to the unresolved state.
func (s *FaultsService) Unresolve(ctx context.Context, projectID string, sel FaultSelection, opts ...Option) error {
	body, err := sel.body()
	if err != nil {
		return err
	}
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().UnresolveFaults(ctx, projectID,
			gen.UnresolveFaultsJSONRequestBody(*body))
	})
}

// Ignore marks faults as ignored, which also stops collecting data for them.
func (s *FaultsService) Ignore(ctx context.Context, projectID string, sel FaultSelection, opts ...Option) error {
	body, err := sel.body()
	if err != nil {
		return err
	}
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().IgnoreFaults(ctx, projectID,
			gen.IgnoreFaultsJSONRequestBody(*body))
	})
}

// Unignore stops ignoring faults.
func (s *FaultsService) Unignore(ctx context.Context, projectID string, sel FaultSelection, opts ...Option) error {
	body, err := sel.body()
	if err != nil {
		return err
	}
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().UnignoreFaults(ctx, projectID,
			gen.UnignoreFaultsJSONRequestBody(*body))
	})
}

// FaultMerge is a merge accepted for background processing.
type FaultMerge struct {
	// BatchID identifies the background merge.
	BatchID string `json:"batch_id"`

	// SourceID is the fault that was merged away.
	SourceID int `json:"source_id"`

	// TargetID is the fault that was kept.
	TargetID int `json:"target_id"`
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
func (s *FaultsService) Merge(ctx context.Context, projectID string, sourceFaultID, targetFaultID int, opts ...Option) (*FaultMerge, error) {
	if sourceFaultID == targetFaultID {
		return nil, ErrMergeIntoSelf
	}
	body := gen.MergeFaultsJSONRequestBody{TargetFaultId: targetFaultID}
	return getOne[FaultMerge](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().MergeFaults(ctx, projectID, sourceFaultID, body)
	})
}

// PauseDuration controls how long recording is paused.
type PauseDuration = gen.PauseFaultRecordingJSONBodyTime

const (
	PauseHour PauseDuration = gen.PauseFaultRecordingJSONBodyTimeHour
	PauseDay  PauseDuration = gen.PauseFaultRecordingJSONBodyTimeDay
	PauseWeek PauseDuration = gen.PauseFaultRecordingJSONBodyTimeWeek
)

// PauseRecording stops recording new notices for a fault for the given duration.
func (s *FaultsService) PauseRecording(ctx context.Context, projectID string, faultID int, duration PauseDuration, opts ...Option) error {
	if !duration.Valid() {
		return fmt.Errorf("apiv3: invalid pause duration %q (use PauseHour, PauseDay, or PauseWeek)", duration)
	}
	body := gen.PauseFaultRecordingJSONRequestBody{Time: duration}
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().PauseFaultRecording(ctx, projectID, faultID, body)
	})
}

// ResumeRecording starts recording notices for a fault again.
func (s *FaultsService) ResumeRecording(ctx context.Context, projectID string, faultID int, opts ...Option) error {
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().ResumeFaultRecording(ctx, projectID, faultID)
	})
}

// Delete removes a fault and its notices.
func (s *FaultsService) Delete(ctx context.Context, projectID string, faultID int, opts ...Option) error {
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().DeleteFault(ctx, projectID, faultID)
	})
}

// AddComment attaches a comment to a fault.
//
// Commenting attributes text to a person, so an account token holding
// faults:write is still refused with requires_user_token — there is nobody to
// attribute it to. Check errors.Is(err, ErrRequiresUserToken).
func (s *FaultsService) AddComment(ctx context.Context, projectID string, faultID int, comment string, opts ...Option) error {
	body := gen.CreateCommentJSONRequestBody{Body: comment}
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().CreateComment(ctx, projectID, faultID, body)
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
func (s *FaultsService) Update(ctx context.Context, projectID string, faultID int, p FaultParams, opts ...Option) (*Fault, error) {
	return getOne[Fault](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().UpdateFault(ctx, projectID, faultID, p)
	})
}

// Assign gives a fault to a user.
//
// The id must belong to a member of the project; one that does not is rejected
// with 422 rather than silently unassigning.
func (s *FaultsService) Assign(ctx context.Context, projectID string, faultID int, assigneeID string, opts ...Option) error {
	body := gen.AssignFaultJSONRequestBody{AssigneeId: assigneeID}
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().AssignFault(ctx, projectID, faultID, body)
	})
}

// Unassign removes a fault's assignee.
func (s *FaultsService) Unassign(ctx context.Context, projectID string, faultID int, opts ...Option) error {
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().UnassignFault(ctx, projectID, faultID)
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
		return s.client.gen().GetFaultSummary(ctx, projectID, params)
	})
	if err != nil {
		return nil, err
	}
	return *data, nil
}
