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

// An empty page while the metadata promises more is a server inconsistency.
// Returning the partial result would look like a complete collection, which is
// the worst outcome for a caller counting or summarising it.
func TestCollectPagesRejectsEmptyPageWhenMorePromised(t *testing.T) {
	fetch := func(ctx context.Context, page int) (*ListResponse[string], error) {
		if page == 1 {
			return offsetPage([]string{"a"}, 1, 1, 100), nil
		}
		return offsetPage(nil, page, 1, 100), nil
	}
	_, err := CollectPages(context.Background(), fetch)
	if !errors.Is(err, ErrPaginationInconsistent) {
		t.Fatalf("err = %v, want ErrPaginationInconsistent", err)
	}
}

// An empty final page is fine: nothing more was promised.
func TestCollectPagesAcceptsEmptyFinalPage(t *testing.T) {
	fetch := func(ctx context.Context, page int) (*ListResponse[string], error) {
		if page == 1 {
			return offsetPage([]string{"a"}, 1, 1, 1), nil
		}
		t.Fatalf("fetched page %d past the end", page)
		return nil, nil
	}
	got, err := CollectPages(context.Background(), fetch)
	if err != nil {
		t.Fatalf("CollectPages: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("got %v, want 1 item", got)
	}
}

// A server stuck on page 1 would otherwise duplicate that page and then report
// success with the rest missing.
func TestCollectPagesRejectsNonAdvancingPage(t *testing.T) {
	fetch := func(ctx context.Context, page int) (*ListResponse[string], error) {
		return offsetPage([]string{"a"}, 1, 1, 10), nil // always says page 1
	}
	_, err := CollectPages(context.Background(), fetch)
	if !errors.Is(err, ErrPaginationInconsistent) {
		t.Fatalf("err = %v, want ErrPaginationInconsistent", err)
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

// has_older says data remains, so a missing cursor means it is unreachable.
// Stopping quietly would drop it without telling anyone.
func TestCollectCursorRejectsMissingCursor(t *testing.T) {
	fetch := func(ctx context.Context, before string) (*ListResponse[string], error) {
		return cursorPage([]string{"n1"}, true, ""), nil
	}
	_, err := CollectCursor(context.Background(), fetch)
	if !errors.Is(err, ErrPaginationInconsistent) {
		t.Fatalf("err = %v, want ErrPaginationInconsistent", err)
	}
}

// A repeated cursor would loop, re-requesting the same page and appending
// duplicates until the hard cap.
func TestCollectCursorRejectsRepeatedCursor(t *testing.T) {
	fetch := func(ctx context.Context, before string) (*ListResponse[string], error) {
		return cursorPage([]string{"n1"}, true, "same"), nil
	}
	_, err := CollectCursor(context.Background(), fetch)
	if !errors.Is(err, ErrPaginationInconsistent) {
		t.Fatalf("err = %v, want ErrPaginationInconsistent", err)
	}
}

// An explicit null cursor is the same unreachable state as an absent one. Worth
// its own test because nullable.Nullable distinguishes them and the walk must
// treat both as inconsistent when has_older is set.
func TestCollectCursorRejectsNullCursor(t *testing.T) {
	fetch := func(ctx context.Context, before string) (*ListResponse[string], error) {
		return &ListResponse[string]{
			Data: []string{"n1"},
			Cursor: &CursorPagination{
				HasOlder:     true,
				OldestCursor: nullable.NewNullNullable[string](),
			},
		}, nil
	}
	_, err := CollectCursor(context.Background(), fetch)
	if !errors.Is(err, ErrPaginationInconsistent) {
		t.Fatalf("err = %v, want ErrPaginationInconsistent", err)
	}
}

// has_older false with a null cursor is the normal end of a walk.
func TestCollectCursorStopsCleanlyAtEnd(t *testing.T) {
	fetch := func(ctx context.Context, before string) (*ListResponse[string], error) {
		return cursorPage([]string{"n1"}, false, ""), nil
	}
	got, err := CollectCursor(context.Background(), fetch)
	if err != nil {
		t.Fatalf("CollectCursor: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("got %v, want 1 item", got)
	}
}

// Unique cursors forever: the repeated-cursor guard cannot catch this, so the
// hard cap is what stops it.
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
