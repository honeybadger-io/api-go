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

// CursorPagination is opaque-cursor pagination, used by time-series endpoints
// such as notices, comments, and deploys. Walk it with CollectCursor.
type CursorPagination = gen.CursorPagination

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
	// contradicts itself: a page number that does not advance, a repeated
	// cursor, an empty page while more are promised, or more items promised with
	// no cursor to reach them.
	//
	// These are reported rather than absorbed. Absorbing them means returning a
	// partial collection that looks complete, which is the worst outcome for a
	// caller summarising or counting the results.
	ErrPaginationInconsistent = errors.New("apiv3: server pagination metadata is inconsistent")
)

// ListResponse is one page of a collection.
//
// Exactly one of Pagination and Cursor is set, depending on the endpoint's
// scheme. Both are nil for an endpoint that returns a bare collection.
type ListResponse[T any] struct {
	Data      []T
	Links     map[string]any
	RequestID string

	// Pagination is set for offset-paginated endpoints (page / per_page).
	Pagination *Pagination

	// Cursor is set for cursor-paginated endpoints (limit / before / after).
	Cursor *CursorPagination
}

// PageFetcher retrieves a single page by 1-indexed page number.
type PageFetcher[T any] func(ctx context.Context, page int) (*ListResponse[T], error)

// CursorFetcher retrieves a page older than the given cursor. The first call
// receives an empty cursor, meaning "start at the newest".
type CursorFetcher[T any] func(ctx context.Context, before string) (*ListResponse[T], error)

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

// CollectCursor walks a cursor-paginated endpoint from newest to oldest and
// returns every item.
//
// Like CollectPages, it reports inconsistency rather than truncating: a server
// that promises older items but supplies no cursor, or repeats one, is a bug
// worth surfacing.
func CollectCursor[T any](ctx context.Context, fetch CursorFetcher[T]) ([]T, error) {
	var all []T
	before := ""
	seen := make(map[string]bool)

	for i := 0; ; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if i >= maxPages {
			return nil, ErrTooManyPages
		}

		resp, err := fetch(ctx, before)
		if err != nil {
			return nil, err
		}
		if resp == nil {
			return all, nil
		}
		all = append(all, resp.Data...)

		if resp.Cursor == nil || !resp.Cursor.HasOlder {
			return all, nil
		}

		// has_older says more data exists, so a missing or null cursor leaves no
		// way to reach it. Silently stopping here would drop that data without
		// telling anyone.
		next, err := resp.Cursor.OldestCursor.Get()
		if err != nil || next == "" {
			return nil, fmt.Errorf("%w: has_older is true but oldest_cursor is absent or null",
				ErrPaginationInconsistent)
		}
		if seen[next] {
			return nil, fmt.Errorf("%w: cursor %q was returned twice", ErrPaginationInconsistent, next)
		}
		seen[next] = true
		before = next
	}
}
