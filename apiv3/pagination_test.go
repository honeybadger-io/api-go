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

func timeSeriesPage(items []string, hasOlder bool, olderLink string) *ListResponse[string] {
	page := &ListResponse[string]{
		Data:            items,
		TimeSeries:      &TimeSeriesPagination{HasOlder: hasOlder, Limit: len(items)},
		TimeSeriesLinks: &TimeSeriesLinks{Self: "https://app.honeybadger.io/v3/self"},
	}
	if olderLink != "" {
		page.TimeSeriesLinks.Older = nullable.NewNullableWithValue(olderLink)
	}
	return page
}

func TestCollectTimeSeriesFollowsOlderLinks(t *testing.T) {
	var followed []string
	fetch := func(ctx context.Context, link string) (*ListResponse[string], error) {
		followed = append(followed, link)
		switch link {
		case "":
			return timeSeriesPage([]string{"n1", "n2"}, true, "https://app.honeybadger.io/v3/older/1"), nil
		case "https://app.honeybadger.io/v3/older/1":
			return timeSeriesPage([]string{"n3"}, false, ""), nil
		}
		t.Fatalf("unexpected link %q", link)
		return nil, nil
	}

	got, err := CollectTimeSeries(context.Background(), fetch)
	if err != nil {
		t.Fatalf("CollectTimeSeries: %v", err)
	}
	if want := []string{"n1", "n2", "n3"}; !equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if want := []string{"", "https://app.honeybadger.io/v3/older/1"}; !equal(followed, want) {
		t.Errorf("followed %v, want %v", followed, want)
	}
}

// has_older says data remains, so a missing link means it is unreachable.
// Stopping quietly would drop it without telling anyone.
func TestCollectTimeSeriesRejectsMissingOlderLink(t *testing.T) {
	fetch := func(ctx context.Context, link string) (*ListResponse[string], error) {
		return timeSeriesPage([]string{"n1"}, true, ""), nil
	}
	_, err := CollectTimeSeries(context.Background(), fetch)
	if !errors.Is(err, ErrPaginationInconsistent) {
		t.Fatalf("err = %v, want ErrPaginationInconsistent", err)
	}
}

// A repeated link would loop, re-requesting the same page and appending
// duplicates until the hard cap.
func TestCollectTimeSeriesRejectsRepeatedLink(t *testing.T) {
	fetch := func(ctx context.Context, link string) (*ListResponse[string], error) {
		return timeSeriesPage([]string{"n1"}, true, "https://app.honeybadger.io/v3/same"), nil
	}
	_, err := CollectTimeSeries(context.Background(), fetch)
	if !errors.Is(err, ErrPaginationInconsistent) {
		t.Fatalf("err = %v, want ErrPaginationInconsistent", err)
	}
}

// An explicit null link is the same unreachable state as an absent one.
func TestCollectTimeSeriesRejectsNullOlderLink(t *testing.T) {
	fetch := func(ctx context.Context, link string) (*ListResponse[string], error) {
		return &ListResponse[string]{
			Data:       []string{"n1"},
			TimeSeries: &TimeSeriesPagination{HasOlder: true},
			TimeSeriesLinks: &TimeSeriesLinks{
				Self:  "https://app.honeybadger.io/v3/self",
				Older: nullable.NewNullNullable[string](),
			},
		}, nil
	}
	_, err := CollectTimeSeries(context.Background(), fetch)
	if !errors.Is(err, ErrPaginationInconsistent) {
		t.Fatalf("err = %v, want ErrPaginationInconsistent", err)
	}
}

// has_older false is the normal end of a walk.
func TestCollectTimeSeriesStopsCleanlyAtEnd(t *testing.T) {
	fetch := func(ctx context.Context, link string) (*ListResponse[string], error) {
		return timeSeriesPage([]string{"n1"}, false, ""), nil
	}
	got, err := CollectTimeSeries(context.Background(), fetch)
	if err != nil {
		t.Fatalf("CollectTimeSeries: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("got %v, want 1 item", got)
	}
}

// Unique links forever: the repeated-link guard cannot catch this, so the hard
// cap is what stops it.
func TestCollectTimeSeriesEnforcesHardCap(t *testing.T) {
	i := 0
	fetch := func(ctx context.Context, link string) (*ListResponse[string], error) {
		i++
		return timeSeriesPage([]string{fmt.Sprintf("n%d", i)}, true,
			fmt.Sprintf("https://app.honeybadger.io/v3/older/%d", i)), nil
	}
	_, err := CollectTimeSeries(context.Background(), fetch)
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
