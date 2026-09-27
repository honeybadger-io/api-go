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
// is not constructible: the endpoint ignores the query and time filters whenever
// ids are present, so such a request would read as a narrowing and not be one.
//
// The zero value is not a valid selection: see ErrEveryFault.
type FaultSelection struct {
	ids   []int
	query string
	all   bool

	dryRun bool

	// Unix seconds with a fractional part, zero meaning unset. The endpoint takes
	// these as JSON numbers, and the fraction matters: the listing filters send it
	// too, and the same cutoff must select the same faults in both.
	createdAfter   float64
	occurredAfter  float64
	occurredBefore float64
}

func unixSeconds(t time.Time) float64 {
	return float64(t.UnixNano()) / float64(time.Second)
}

// CreatedAfter restricts the change to faults first seen after t.
//
// Like the other filters, it applies only when no ids are named. Chainable:
//
//	apiv3.SelectFaultsMatching("is:unresolved").OccurredBefore(cutoff)
func (sel FaultSelection) CreatedAfter(t time.Time) FaultSelection {
	sel.createdAfter = unixSeconds(t)
	return sel
}

// OccurredAfter restricts the change to faults with a notice after t.
func (sel FaultSelection) OccurredAfter(t time.Time) FaultSelection {
	sel.occurredAfter = unixSeconds(t)
	return sel
}

// OccurredBefore restricts the change to faults with no notice since t.
func (sel FaultSelection) OccurredBefore(t time.Time) FaultSelection {
	sel.occurredBefore = unixSeconds(t)
	return sel
}

// DryRun reports what the change would do without doing it: no state change,
// no comments, no timeline entries. The result has the same shape either way.
func (sel FaultSelection) DryRun() FaultSelection {
	sel.dryRun = true
	return sel
}

// filtered reports whether any filter other than an id list is set.
func (sel FaultSelection) filtered() bool {
	return strings.TrimSpace(sel.query) != "" ||
		sel.createdAfter != 0 || sel.occurredAfter != 0 || sel.occurredBefore != 0
}

// SelectFaults changes the named faults. An id that isn't a fault in the project,
// including one merged away since it was fetched, refuses the whole request with
// 422.
func SelectFaults(ids ...int) FaultSelection {
	return FaultSelection{ids: ids}
}

// SelectFaultsMatching changes every fault the search matches, in the same syntax
// as the fault listing's Search.
func SelectFaultsMatching(query string) FaultSelection {
	return FaultSelection{query: query}
}

// SelectAllFaults changes every fault in the project, or every fault the chained
// time filters match.
//
// It sends all=true, which the endpoint requires before it will make an
// unbounded change. A wildcard query is not the same thing: the search runs
// against notices, so a fault whose notices are not searchable would be missed by
// "*" and caught here.
func SelectAllFaults() FaultSelection {
	return FaultSelection{all: true}
}

// ErrEveryFault is returned when a bulk change names no faults and sets no
// filter, without having asked for the whole project.
//
// The endpoint refuses such a body unless it carries all=true, so sending it
// would only earn a 422. It is refused here instead, with the fix in the message.
// Use SelectAllFaults when the whole project really is the intent.
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
	body := &gen.ResolveFaultsJSONRequestBody{}
	if sel.dryRun {
		t := true
		body.DryRun = &t
	}

	if len(sel.ids) > 0 {
		if sel.filtered() {
			return nil, ErrFilteredIDs
		}
		ids := sel.ids
		body.FaultIds = &ids
		return body, nil
	}

	if sel.all {
		t := true
		body.All = &t
	}
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

// FaultBulkResult reports what a bulk change did.
//
// Count is exact and counts what changed rather than what matched: resolving ten
// faults of which nine were already resolved reports 1. A request naming ids from
// another project succeeds with Count 0, so a caller that expected a change must
// check it. FaultIds is capped at 100; FaultIdsTruncated says when it was.
type FaultBulkResult = gen.FaultBulkResult

// The four bulk endpoints share one body schema, so they share one Go type: the
// generated Unresolve/Ignore/Unignore bodies are structurally identical and
// convertible.

// Resolve marks faults as resolved.
func (s *FaultsService) Resolve(ctx context.Context, projectID string, sel FaultSelection, opts ...Option) (*FaultBulkResult, error) {
	body, err := sel.body()
	if err != nil {
		return nil, err
	}
	return getOne[FaultBulkResult](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().ResolveFaults(ctx, projectID, *body)
	})
}

// Unresolve returns faults to the unresolved state.
func (s *FaultsService) Unresolve(ctx context.Context, projectID string, sel FaultSelection, opts ...Option) (*FaultBulkResult, error) {
	body, err := sel.body()
	if err != nil {
		return nil, err
	}
	return getOne[FaultBulkResult](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().UnresolveFaults(ctx, projectID,
			gen.UnresolveFaultsJSONRequestBody(*body))
	})
}

// Ignore marks faults as ignored, which also stops collecting data for them.
func (s *FaultsService) Ignore(ctx context.Context, projectID string, sel FaultSelection, opts ...Option) (*FaultBulkResult, error) {
	body, err := sel.body()
	if err != nil {
		return nil, err
	}
	return getOne[FaultBulkResult](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().IgnoreFaults(ctx, projectID,
			gen.IgnoreFaultsJSONRequestBody(*body))
	})
}

// Unignore stops ignoring faults.
func (s *FaultsService) Unignore(ctx context.Context, projectID string, sel FaultSelection, opts ...Option) (*FaultBulkResult, error) {
	body, err := sel.body()
	if err != nil {
		return nil, err
	}
	return getOne[FaultBulkResult](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().UnignoreFaults(ctx, projectID,
			gen.UnignoreFaultsJSONRequestBody(*body))
	})
}

// FaultMerge is a merge accepted for background processing: BatchId identifies
// it, SourceId is the fault merged away and TargetId the fault kept.
type FaultMerge = gen.FaultMerge

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
type PauseDuration = gen.PauseDuration

const (
	PauseHour PauseDuration = gen.PauseDurationHour
	PauseDay  PauseDuration = gen.PauseDurationDay
	PauseWeek PauseDuration = gen.PauseDurationWeek
)

// PauseRecording stops recording new notices for a fault for the given duration,
// and returns the fault, whose RecordingPausedUntil says until when.
func (s *FaultsService) PauseRecording(ctx context.Context, projectID string, faultID int, duration PauseDuration, opts ...Option) (*Fault, error) {
	if !duration.Valid() {
		return nil, fmt.Errorf("apiv3: invalid pause duration %q (use PauseHour, PauseDay, or PauseWeek)", duration)
	}
	body := gen.PauseFaultRecordingJSONRequestBody{Time: duration}
	return getOne[Fault](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().PauseFaultRecording(ctx, projectID, faultID, body)
	})
}

// ResumeRecording starts recording notices for a fault again, and returns the
// fault as it now stands.
func (s *FaultsService) ResumeRecording(ctx context.Context, projectID string, faultID int, opts ...Option) (*Fault, error) {
	return getOne[Fault](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().ResumeFaultRecording(ctx, projectID, faultID)
	})
}

// Delete removes a fault and its notices.
func (s *FaultsService) Delete(ctx context.Context, projectID string, faultID int, opts ...Option) error {
	return noContent(ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().DeleteFault(ctx, projectID, faultID)
	})
}

// Comment is a comment on a fault.
type Comment = gen.Comment

// AddComment attaches a comment to a fault.
//
// Commenting attributes text to a person, so an account token holding
// faults:write is still refused with requires_user_token — there is nobody to
// attribute it to. Check errors.Is(err, ErrRequiresUserToken).
func (s *FaultsService) AddComment(ctx context.Context, projectID string, faultID int, comment string, opts ...Option) (*Comment, error) {
	body := gen.CreateCommentJSONRequestBody{Body: comment}
	return getOne[Comment](ctx, s.client, func() (*http.Response, error) {
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
// with 422 rather than silently unassigning. Returns the fault as it now stands.
func (s *FaultsService) Assign(ctx context.Context, projectID string, faultID int, assigneeID string, opts ...Option) (*Fault, error) {
	body := gen.AssignFaultJSONRequestBody{AssigneeId: assigneeID}
	return getOne[Fault](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().AssignFault(ctx, projectID, faultID, body)
	})
}

// Unassign removes a fault's assignee and returns the fault as it now stands.
func (s *FaultsService) Unassign(ctx context.Context, projectID string, faultID int, opts ...Option) (*Fault, error) {
	return getOne[Fault](ctx, s.client, func() (*http.Response, error) {
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
