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

	return listOffset[CheckIn](ctx, s.client, "listCheckIns", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().ListCheckIns(ctx, projectID, params)
	})
}

// Get returns a single check-in.
func (s *CheckInsService) Get(ctx context.Context, projectID, checkInID string) (*CheckIn, error) {
	return getOne[CheckIn](ctx, s.client, "getCheckIn", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().GetCheckIn(ctx, projectID, checkInID)
	})
}

// ListEvents returns one page of a check-in's events, newest first. Limit sizes
// the page and OlderThan pages back; ListAllEvents walks them all.
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
	ro.applyOlderThan(&params.Limit, &params.CreatedBefore)

	return listTimeSeries[CheckInEvent](ctx, s.client, "listCheckInEvents", func(ctx context.Context) (*http.Response, error) {
		return s.client.gen().ListCheckInEvents(ctx, projectID, checkInID, params)
	})
}
