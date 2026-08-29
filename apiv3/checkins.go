package apiv3

import (
	"context"
	"net/http"

	"github.com/honeybadger-io/api-go/internal/gen"
)

// CheckIn is a cron or heartbeat monitor.
type CheckIn = gen.CheckIn

// CheckInEvent is one report against a check-in.
type CheckInEvent = gen.CheckInEvent

// CheckInsService handles the check-ins resource and its events.
type CheckInsService struct {
	client *Client
}

// List returns one page of a project's check-ins.
func (s *CheckInsService) List(ctx context.Context, projectID string, opts ...Option) (*ListResponse[CheckIn], error) {
	return s.list(ctx, projectID, resolve(opts))
}

// ListAll returns every check-in for a project, walking pagination.
func (s *CheckInsService) ListAll(ctx context.Context, projectID string, opts ...ListAllOption) ([]CheckIn, error) {
	ro := resolveListAll(opts)
	return CollectPages(ctx, func(ctx context.Context, page int) (*ListResponse[CheckIn], error) {
		ro.page = page
		return s.list(ctx, projectID, ro)
	})
}

func (s *CheckInsService) list(ctx context.Context, projectID string, ro requestOptions) (*ListResponse[CheckIn], error) {
	params := &gen.ListCheckInsParams{}
	ro.applyOffset(&params.Page, &params.PerPage)

	return listOffset[CheckIn](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().ListCheckIns(ctx, projectID, params)
	})
}

// Get returns a single check-in.
func (s *CheckInsService) Get(ctx context.Context, projectID, checkInID string, opts ...Option) (*CheckIn, error) {
	return getOne[CheckIn](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().GetCheckIn(ctx, projectID, checkInID)
	})
}

// ListEvents returns one page of a check-in's events, newest first.
//
// Only Limit applies here. The endpoint's other parameter, created_before, is
// deliberately not exposed: the spec types it as `number`, which generates a
// float32, and a float32 cannot represent an epoch second — at current
// timestamps its precision is coarser than two minutes, so paging by it would
// silently skip or repeat events. Walk with ListAllEvents instead, which follows
// links and never has to name a timestamp.
func (s *CheckInsService) ListEvents(ctx context.Context, projectID, checkInID string, opts ...Option) (*ListResponse[CheckInEvent], error) {
	return s.listEvents(ctx, projectID, checkInID, resolve(opts))
}

// ListAllEvents returns every event for a check-in, walking from newest to
// oldest by following links.older.
func (s *CheckInsService) ListAllEvents(ctx context.Context, projectID, checkInID string, opts ...ListAllOption) ([]CheckInEvent, error) {
	ro := resolveListAll(opts)
	return CollectTimeSeries(ctx, func(ctx context.Context, link string) (*ListResponse[CheckInEvent], error) {
		if link != "" {
			return followTimeSeries[CheckInEvent](ctx, s.client, link)
		}
		return s.listEvents(ctx, projectID, checkInID, ro)
	})
}

func (s *CheckInsService) listEvents(ctx context.Context, projectID, checkInID string, ro requestOptions) (*ListResponse[CheckInEvent], error) {
	params := &gen.ListCheckInEventsParams{}
	if ro.limit > 0 {
		limit := gen.Limit(ro.limit)
		params.Limit = &limit
	}

	return listTimeSeries[CheckInEvent](ctx, s.client, func() (*http.Response, error) {
		return s.client.gen().ListCheckInEvents(ctx, projectID, checkInID, params)
	})
}
