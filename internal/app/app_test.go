package app

import (
	"context"
	"errors"
	"github.com/armada877/syn-cli/internal/domain"
	"testing"
	"time"
)

type repository struct {
	domain.Repository
	feeds   []domain.Feed
	applied []int64
	times   []time.Time
}

func (r *repository) ListFeeds(context.Context) ([]domain.Feed, error) { return r.feeds, nil }
func (r *repository) ApplyFetch(_ context.Context, id int64, _ domain.FetchResult, at time.Time) error {
	r.applied = append(r.applied, id)
	r.times = append(r.times, at)
	return nil
}

type fetcher struct{ calls []int64 }

func (f *fetcher) Fetch(_ context.Context, feed domain.Feed) (domain.FetchResult, error) {
	f.calls = append(f.calls, feed.ID)
	if feed.ID == 2 {
		return domain.FetchResult{}, errors.New("fetch failed")
	}
	return domain.FetchResult{Title: "updated"}, nil
}
func TestRefreshPartialFailure(t *testing.T) {
	r := &repository{feeds: []domain.Feed{{ID: 1}, {ID: 2}, {ID: 3}}}
	f := &fetcher{}
	a := New(r, f)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a.Now = func() time.Time { return now }
	results, err := a.Refresh(context.Background(), 0)
	if err != nil || len(results) != 3 || results[1].Err == nil {
		t.Fatalf("results %v, error %v", results, err)
	}
	if len(r.applied) != 2 || r.applied[0] != 1 || r.applied[1] != 3 || r.times[0] != now {
		t.Fatalf("applied %v at %v", r.applied, r.times)
	}
	if len(f.calls) != 3 || f.calls[0] != 1 || f.calls[1] != 2 || f.calls[2] != 3 {
		t.Fatal(f.calls)
	}
	if _, err = a.Refresh(context.Background(), 99); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = a.Refresh(ctx, 0); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
func TestText(t *testing.T) {
	source := "<h1>Title &amp; more</h1><p>First <b>bold</b>.</p><script>bad()</script><style>bad</style><p>Second\x1b[31m red\x1b[0m\x1b]0;attack\x07\x00\u009b</p>"
	want := "Title & more\n\nFirst bold.\n\nSecond red"
	if got := PlainText(source); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if got := SafeText("hello\x1bPbad\x1b\\ world\r\x08\u202e"); got != "hello world" {
		t.Fatalf("got %q", got)
	}
}
