package apiv3

import (
	"context"
	"errors"
	"fmt"

	"github.com/honeybadger-io/api-go/internal/gen"
)

// Pagination is offset-based pagination, used by most list endpoints.
//
// Aliased from the generated models so there is one definition rather than a
// hand-written copy that can drift.
type Pagination = gen.Pagination

// TimeSeriesPagination is pagination for a time-ordered collection: notices,
// comments, deploys, check-in events, uptime checks, outages.
//
// v3 unified what were previously two separate schemes. Cursor-paged
// collections populate OldestCursor and NewestCursor; collections that page on a
// timestamp leave them absent or null. Both kinds report HasOlder and HasNewer
// and supply links to follow, so following links works for either — which is
// what CollectTimeSeries does.
type TimeSeriesPagination = gen.TimeSeriesPagination

// TimeSeriesLinks are the navigation links for a time-ordered collection.
type TimeSeriesLinks = gen.TimeSeriesLinks

// maxPages bounds any pagination walk. Nothing legitimate needs this many
// requests — the rate limit is 360/hour — so hitting it means the server's
// counts or cursors are inconsistent, and looping forever would be worse than
// failing.
const maxPages = 500

var (
	// ErrTooManyPages is returned when a walk exceeds maxPages, which means the
	// server never signalled the end of the collection.
	ErrTooManyPages = errors.New("apiv3: pagination exceeded " + fmt.Sprint(maxPages) + " pages")

	// ErrPaginationInconsistent is returned when a server's pagination metadata
	// contradicts itself: a page number that does not advance, a repeated link,
	// an empty page while more are promised, or more items promised with no way
	// to reach them.
	//
	// These are reported rather than absorbed. Absorbing them means returning a
	// partial collection that looks complete, which is the worst outcome for a
	// caller summarising or counting the results.
	ErrPaginationInconsistent = errors.New("apiv3: server pagination metadata is inconsistent")

	// ErrUntrustedLink is returned when a pagination link points at a different
	// host than the configured base URL. Following it would send the credential
	// somewhere it was not issued for.
	ErrUntrustedLink = errors.New("apiv3: pagination link points at an untrusted host")
)

// ListResponse is one page of a collection.
//
// Exactly one of Pagination and TimeSeries is set, depending on the endpoint's
// scheme. Both are nil for an endpoint that returns a bare collection.
type ListResponse[T any] struct {
	Data      []T
	RequestID string

	// Pagination is set for offset-paginated endpoints (page / per_page).
	Pagination *Pagination

	// TimeSeries is set for time-ordered endpoints (limit / before / after).
	TimeSeries *TimeSeriesPagination

	// TimeSeriesLinks are the navigation links for a time-ordered endpoint.
	TimeSeriesLinks *TimeSeriesLinks

	// Links holds an offset endpoint's untyped links object.
	Links map[string]any
}

// PageFetcher retrieves a single page by 1-indexed page number.
type PageFetcher[T any] func(ctx context.Context, page int) (*ListResponse[T], error)

// TimeSeriesFetcher retrieves a page of a time-ordered collection. The first call
// receives an empty url, meaning "the newest page"; later calls receive the
// `older` link from the previous response.
type TimeSeriesFetcher[T any] func(ctx context.Context, url string) (*ListResponse[T], error)

// CollectPages walks an offset-paginated endpoint and returns every item.
//
// It reports ErrPaginationInconsistent rather than returning a partial result
// when the server's metadata does not add up.
func CollectPages[T any](ctx context.Context, fetch PageFetcher[T]) ([]T, error) {
	var all []T

	for page := 1; ; page++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if page > maxPages {
			return nil, ErrTooManyPages
		}

		resp, err := fetch(ctx, page)
		if err != nil {
			return nil, err
		}
		if resp == nil {
			return all, nil
		}

		// No pagination block means the endpoint returned everything at once.
		if resp.Pagination == nil {
			return append(all, resp.Data...), nil
		}

		// The server must be answering the page that was asked for. A server
		// that keeps returning page 1 would otherwise duplicate that page and
		// then report success with the rest missing.
		if resp.Pagination.Page != 0 && resp.Pagination.Page != page {
			return nil, fmt.Errorf("%w: requested page %d, server returned page %d",
				ErrPaginationInconsistent, page, resp.Pagination.Page)
		}

		morePromised := page < resp.Pagination.TotalPages
		if len(resp.Data) == 0 {
			if morePromised {
				return nil, fmt.Errorf("%w: page %d of %d was empty",
					ErrPaginationInconsistent, page, resp.Pagination.TotalPages)
			}
			return all, nil
		}
		all = append(all, resp.Data...)

		if !morePromised {
			return all, nil
		}
	}
}

// CollectTimeSeries walks a time-ordered endpoint from newest to oldest and
// returns every item.
//
// It follows links.older rather than cursors. The spec instructs exactly that —
// "when has_older is true, follow links.older" — and it is the only mechanism
// that works for both cursor-paged and timestamp-paged collections, since the
// latter leave the cursor fields null.
func CollectTimeSeries[T any](ctx context.Context, fetch TimeSeriesFetcher[T]) ([]T, error) {
	var all []T
	next := ""
	seen := make(map[string]bool)

	for i := 0; ; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if i >= maxPages {
			return nil, ErrTooManyPages
		}

		resp, err := fetch(ctx, next)
		if err != nil {
			return nil, err
		}
		if resp == nil {
			return all, nil
		}
		all = append(all, resp.Data...)

		if resp.TimeSeries == nil || !resp.TimeSeries.HasOlder {
			return all, nil
		}

		// has_older says more data exists, so no link to follow leaves it
		// unreachable. Stopping quietly would drop it without telling anyone.
		if resp.TimeSeriesLinks == nil {
			return nil, fmt.Errorf("%w: has_older is true but the response carries no links",
				ErrPaginationInconsistent)
		}
		older, err := resp.TimeSeriesLinks.Older.Get()
		if err != nil || older == "" {
			return nil, fmt.Errorf("%w: has_older is true but links.older is absent or null",
				ErrPaginationInconsistent)
		}
		if seen[older] {
			return nil, fmt.Errorf("%w: link %q was returned twice", ErrPaginationInconsistent, older)
		}
		seen[older] = true
		next = older
	}
}
