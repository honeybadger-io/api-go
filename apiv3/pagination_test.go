package apiv3

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/oapi-codegen/nullable"
)

func offsetPage(items []string, page, perPage, total int) *ListResponse[string] {
	return &ListResponse[string]{
		Data: items,
		Pagination: &Pagination{
			Page:       page,
			PerPage:    perPage,
			TotalCount: total,
			TotalPages: (total + perPage - 1) / perPage,
		},
	}
}

func TestCollectPagesWalksEveryPage(t *testing.T) {
	var requested []int
	fetch := func(ctx context.Context, page int) (*ListResponse[string], error) {
		requested = append(requested, page)
		switch page {
		case 1:
			return offsetPage([]string{"a", "b"}, 1, 2, 5), nil
		case 2:
			return offsetPage([]string{"c", "d"}, 2, 2, 5), nil
		case 3:
			return offsetPage([]string{"e"}, 3, 2, 5), nil
		}
		t.Fatalf("fetched page %d beyond the last", page)
		return nil, nil
	}

	got, err := CollectPages(context.Background(), fetch)
	if err != nil {
		t.Fatalf("CollectPages: %v", err)
	}
	if want := []string{"a", "b", "c", "d", "e"}; !equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if want := []int{1, 2, 3}; !equalInt(requested, want) {
		t.Errorf("requested pages %v, want %v", requested, want)
	}
}

func TestCollectPagesSinglePage(t *testing.T) {
	fetch := func(ctx context.Context, page int) (*ListResponse[string], error) {
		return offsetPage([]string{"only"}, 1, 25, 1), nil
	}
	got, err := CollectPages(context.Background(), fetch)
	if err != nil {
		t.Fatalf("CollectPages: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("got %v, want one item", got)
	}
}

// A response with no pagination block is a single page, not an error.
func TestCollectPagesWithoutPaginationBlock(t *testing.T) {
	fetch := func(ctx context.Context, page int) (*ListResponse[string], error) {
		return &ListResponse[string]{Data: []string{"x"}}, nil
	}
	got, err := CollectPages(context.Background(), fetch)
	if err != nil {
		t.Fatalf("CollectPages: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("got %v, want one item", got)
	}
}

func TestCollectPagesPropagatesError(t *testing.T) {
	boom := errors.New("boom")
	fetch := func(ctx context.Context, page int) (*ListResponse[string], error) {
		if page == 2 {
			return nil, boom
		}
		return offsetPage([]string{"a"}, 1, 1, 3), nil
	}
	_, err := CollectPages(context.Background(), fetch)
	if !errors.Is(err, boom) {
		t.Errorf("err = %v, want boom", err)
	}
}

// An empty page ends the walk even when the counts claim more, so a
// miscounted total_pages cannot spin forever.
func TestCollectPagesStopsOnEmptyPage(t *testing.T) {
	calls := 0
	fetch := func(ctx context.Context, page int) (*ListResponse[string], error) {
		calls++
		if page == 1 {
			return offsetPage([]string{"a"}, 1, 1, 100), nil
		}
		return offsetPage(nil, page, 1, 100), nil
	}
	got, err := CollectPages(context.Background(), fetch)
	if err != nil {
		t.Fatalf("CollectPages: %v", err)
	}
	if len(got) != 1 || calls != 2 {
		t.Errorf("got %v after %d calls, want 1 item after 2 calls", got, calls)
	}
}

// A server that always claims another page must not hang the caller.
func TestCollectPagesEnforcesHardCap(t *testing.T) {
	fetch := func(ctx context.Context, page int) (*ListResponse[string], error) {
		return offsetPage([]string{fmt.Sprintf("item-%d", page)}, page, 1, 1<<30), nil
	}
	_, err := CollectPages(context.Background(), fetch)
	if err == nil {
		t.Fatal("want an error when the page count never ends")
	}
	if !errors.Is(err, ErrTooManyPages) {
		t.Errorf("err = %v, want ErrTooManyPages", err)
	}
}

func TestCollectPagesHonorsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	fetch := func(ctx context.Context, page int) (*ListResponse[string], error) {
		cancel()
		return offsetPage([]string{"a"}, page, 1, 100), nil
	}
	_, err := CollectPages(ctx, fetch)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func cursorPage(items []string, hasOlder bool, oldest string) *ListResponse[string] {
	c := &CursorPagination{HasOlder: hasOlder, Limit: len(items)}
	if oldest != "" {
		c.OldestCursor = nullable.NewNullableWithValue(oldest)
	}
	return &ListResponse[string]{Data: items, Cursor: c}
}

func TestCollectCursorWalksBackwards(t *testing.T) {
	var cursors []string
	fetch := func(ctx context.Context, before string) (*ListResponse[string], error) {
		cursors = append(cursors, before)
		switch before {
		case "":
			return cursorPage([]string{"n1", "n2"}, true, "cur2"), nil
		case "cur2":
			return cursorPage([]string{"n3"}, false, ""), nil
		}
		t.Fatalf("unexpected cursor %q", before)
		return nil, nil
	}

	got, err := CollectCursor(context.Background(), fetch)
	if err != nil {
		t.Fatalf("CollectCursor: %v", err)
	}
	if want := []string{"n1", "n2", "n3"}; !equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if want := []string{"", "cur2"}; !equal(cursors, want) {
		t.Errorf("cursors %v, want %v", cursors, want)
	}
}

// has_older true but no cursor to follow is a dead end, not a loop.
func TestCollectCursorStopsWhenCursorMissing(t *testing.T) {
	calls := 0
	fetch := func(ctx context.Context, before string) (*ListResponse[string], error) {
		calls++
		return cursorPage([]string{"n1"}, true, ""), nil
	}
	got, err := CollectCursor(context.Background(), fetch)
	if err != nil {
		t.Fatalf("CollectCursor: %v", err)
	}
	if len(got) != 1 || calls != 1 {
		t.Errorf("got %v after %d calls, want 1 item after 1 call", got, calls)
	}
}

// An explicit null cursor is the same dead end as an absent one. Worth its own
// test because nullable.Nullable distinguishes the two and the walk must not.
func TestCollectCursorStopsOnNullCursor(t *testing.T) {
	fetch := func(ctx context.Context, before string) (*ListResponse[string], error) {
		return &ListResponse[string]{
			Data: []string{"n1"},
			Cursor: &CursorPagination{
				HasOlder:     true,
				OldestCursor: nullable.NewNullNullable[string](),
			},
		}, nil
	}
	got, err := CollectCursor(context.Background(), fetch)
	if err != nil {
		t.Fatalf("CollectCursor: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("got %v, want 1 item", got)
	}
}

func TestCollectCursorEnforcesHardCap(t *testing.T) {
	i := 0
	fetch := func(ctx context.Context, before string) (*ListResponse[string], error) {
		i++
		return cursorPage([]string{fmt.Sprintf("n%d", i)}, true, fmt.Sprintf("cur%d", i)), nil
	}
	_, err := CollectCursor(context.Background(), fetch)
	if !errors.Is(err, ErrTooManyPages) {
		t.Errorf("err = %v, want ErrTooManyPages", err)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalInt(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
