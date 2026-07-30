package apiv3

// Option configures a single request. Options are variadic, so the common call
// carries none:
//
//	c.Faults.Get(ctx, projectID, faultID)
//	c.Faults.Get(ctx, projectID, faultID, apiv3.InAccount("Ab3kL9"))
//
// Options that do not apply to an operation are ignored rather than rejected —
// each operation reads only the fields it understands. The exception is paging:
// ListAll methods take ListAllOption, a narrower interface that Page does not
// satisfy, so a page number cannot be passed to a call that walks every page.
type Option interface {
	apply(*requestOptions)
}

// ListAllOption configures a ListAll call. Every Option except paging is also a
// ListAllOption; Page deliberately is not, because ListAll starts at the first
// page by definition.
type ListAllOption interface {
	Option
	listAll()
}

// requestOptions is the resolved set of per-request settings.
type requestOptions struct {
	accountID string
	page      int
	perPage   int
	limit     int
	before    string
	after     string
	query     string
}

func resolve(opts []Option) requestOptions {
	var ro requestOptions
	for _, o := range opts {
		if o != nil {
			o.apply(&ro)
		}
	}
	return ro
}

func resolveListAll(opts []ListAllOption) requestOptions {
	var ro requestOptions
	for _, o := range opts {
		if o != nil {
			o.apply(&ro)
		}
	}
	return ro
}

// accountOption selects the account a request addresses.
type accountOption struct{ id string }

func (o accountOption) apply(ro *requestOptions) { ro.accountID = o.id }
func (o accountOption) listAll()                 {}

// InAccount addresses a specific account instead of resolving one from the
// credential. Needed when a credential covers more than one account, which is
// the case that returns ambiguous_account.
func InAccount(accountID string) accountOption {
	return accountOption{id: accountID}
}

// pageOption selects one page of an offset-paginated collection.
type pageOption struct {
	page    int
	perPage int
}

func (o pageOption) apply(ro *requestOptions) {
	ro.page = o.page
	ro.perPage = o.perPage
}

// Page requests a single page of an offset-paginated collection. Pages are
// 1-indexed; perPage caps at 100 and defaults to 25 when zero.
//
// Page is not a ListAllOption: ListAll walks from the first page by definition,
// so a page number there would be silently ignored. The compiler rejects it
// instead.
func Page(page, perPage int) pageOption {
	return pageOption{page: page, perPage: perPage}
}

// limitOption caps how many items one request returns.
type limitOption struct{ n int }

func (o limitOption) apply(ro *requestOptions) { ro.limit = o.n }
func (o limitOption) listAll()                 {}

// Limit caps how many items a single request returns, for cursor-paginated
// collections. It caps at 100 and defaults to 25 when zero. On a ListAll call it
// sets the size of each underlying request rather than a total.
func Limit(n int) limitOption {
	return limitOption{n: n}
}

// cursorOption positions a request within a cursor-paginated collection.
type cursorOption struct {
	before string
	after  string
}

func (o cursorOption) apply(ro *requestOptions) {
	ro.before = o.before
	ro.after = o.after
}

// Before requests items older than the given cursor, taken from a previous
// response's Cursor.OldestCursor.
//
// Not a ListAllOption: ListAll follows cursors itself from the newest item.
func Before(cursor string) cursorOption {
	return cursorOption{before: cursor}
}

// After requests items newer than the given cursor, taken from a previous
// response's Cursor.NewestCursor.
//
// Not a ListAllOption, for the same reason as Before.
func After(cursor string) cursorOption {
	return cursorOption{after: cursor}
}

// searchOption carries v3's single filter parameter.
type searchOption struct{ q string }

func (o searchOption) apply(ro *requestOptions) { ro.query = o.q }
func (o searchOption) listAll()                 {}

// Search filters a fault listing.
//
// v3 expresses environment and status filters inside this one query rather than
// as separate parameters — "environment:production", "is:resolved",
// "is:ignored", each negatable with a leading "-". This mirrors the API instead
// of inventing a surface that would need updating as the filter language grows.
func Search(query string) searchOption {
	return searchOption{q: query}
}
