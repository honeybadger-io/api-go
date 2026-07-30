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

// CursorPagination is cursor-based pagination, used by time-series endpoints
// such as notices, comments, and deploys.
type CursorPagination = gen.CursorPagination

// maxPages bounds any pagination walk. Nothing legitimate needs this many
// requests — the rate limit is 360/hour — so hitting it means the server's
// counts or cursors are inconsistent, and looping forever would be worse than
// failing.
const maxPages = 500

// ErrTooManyPages is returned when a pagination walk exceeds maxPages, which
// indicates the server never signalled the end of the collection.
var ErrTooManyPages = errors.New("apiv3: pagination exceeded " + fmt.Sprint(maxPages) + " pages")

// ListResponse is one page of a collection.
//
// Exactly one of Pagination and Cursor is set, depending on which scheme the
// endpoint uses. Both may be nil for an endpoint that returns a bare collection.
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
// It stops at the last page reported by the server, or early if a page comes
// back empty — a defence against an inconsistent total_pages. Errors from fetch
// are returned as-is, so errors.Is against the apiv3 sentinels still works.
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
		if resp == nil || len(resp.Data) == 0 {
			return all, nil
		}
		all = append(all, resp.Data...)

		// No pagination block means the endpoint returned everything at once.
		if resp.Pagination == nil {
			return all, nil
		}
		if page >= resp.Pagination.TotalPages {
			return all, nil
		}
	}
}

// CollectCursor walks a cursor-paginated endpoint from newest to oldest and
// returns every item.
//
// It stops when the server reports no older items, or when it reports more but
// supplies no cursor to reach them — an explicit null and an absent cursor are
// both dead ends.
func CollectCursor[T any](ctx context.Context, fetch CursorFetcher[T]) ([]T, error) {
	var all []T
	before := ""

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
		if resp == nil || len(resp.Data) == 0 {
			return all, nil
		}
		all = append(all, resp.Data...)

		if resp.Cursor == nil || !resp.Cursor.HasOlder {
			return all, nil
		}
		// Get reports an error when the cursor is null or unspecified; both mean
		// there is nothing further to follow.
		next, err := resp.Cursor.OldestCursor.Get()
		if err != nil || next == "" {
			return all, nil
		}
		before = next
	}
}
