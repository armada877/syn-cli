package tui

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"errors"
	"fmt"
	"github.com/armada877/syn-cli/internal/app"
	"github.com/armada877/syn-cli/internal/domain"
	"github.com/charmbracelet/x/ansi"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeRepository struct {
	domain.Repository
	feeds     []domain.Feed
	articles  []domain.Article
	calls     int
	readCalls int
}

func (r *fakeRepository) ListFeeds(ctx context.Context) ([]domain.Feed, error) {
	r.calls++
	return append([]domain.Feed{}, r.feeds...), ctx.Err()
}
func (r *fakeRepository) ListArticles(ctx context.Context, _ domain.ArticleFilter) ([]domain.Article, error) {
	r.calls++
	return append([]domain.Article{}, r.articles...), ctx.Err()
}
func (r *fakeRepository) AddFeed(_ context.Context, url string) (domain.Feed, error) {
	f := domain.Feed{ID: 3, URL: url}
	r.feeds = append(r.feeds, f)
	return f, nil
}
func (r *fakeRepository) RemoveFeed(_ context.Context, id int64) error {
	feeds := []domain.Feed{}
	for _, f := range r.feeds {
		if f.ID != id {
			feeds = append(feeds, f)
		}
	}
	r.feeds = feeds
	articles := []domain.Article{}
	for _, a := range r.articles {
		if a.FeedID != id {
			articles = append(articles, a)
		}
	}
	r.articles = articles
	return nil
}
func (r *fakeRepository) SetRead(_ context.Context, id int64, read bool) error {
	r.readCalls++
	for i := range r.articles {
		if r.articles[i].ID == id {
			r.articles[i].Read = read
			return nil
		}
	}
	return domain.ErrNotFound
}
func (r *fakeRepository) ApplyFetch(_ context.Context, id int64, result domain.FetchResult, _ time.Time) error {
	for i := range r.feeds {
		if r.feeds[i].ID == id {
			r.feeds[i].Title = result.Title
		}
	}
	return nil
}

type fakeFetcher struct {
	calls int
	fail  bool
}

func (f *fakeFetcher) Fetch(context.Context, domain.Feed) (domain.FetchResult, error) {
	f.calls++
	if f.fail {
		return domain.FetchResult{}, errors.New("refresh failed")
	}
	return domain.FetchResult{Title: "Updated"}, nil
}
func fixture(t *testing.T) (*Model, *fakeRepository, *fakeFetcher) {
	t.Helper()
	r := &fakeRepository{feeds: []domain.Feed{{ID: 1, Title: "One"}, {ID: 2, Title: "Two"}}, articles: []domain.Article{{ID: 20, FeedID: 2, Title: "Newest", Content: strings.Repeat("word ", 300)}, {ID: 10, FeedID: 1, Title: "Older", Content: "<p>text</p>"}}}
	f := &fakeFetcher{}
	m := NewWithTheme(context.Background(), app.New(r, f), filepath.Join(t.TempDir(), "syn.toml"), false)
	finish(m, m.work(nil))
	return m, r, f
}
func finish(m *Model, cmd tea.Cmd) {
	if cmd != nil {
		m.Update(cmd())
	}
}
func press(m *Model, key string) tea.Cmd { _, cmd := m.key(key); return cmd }
func resize(m *Model, width int)         { m.Update(tea.WindowSizeMsg{Width: width, Height: 20}) }
func TestLayoutsAndResize(t *testing.T) {
	m, r, f := fixture(t)
	m.filter = domain.ArticleFilter{FeedID: 1, UnreadOnly: true}
	m.selected = 10
	m.reconcile()
	reader := m.visible()[0]
	m.reader = &reader
	reader.Content = strings.Repeat("Long text ", 100)
	calls := r.calls
	for _, width := range []int{79, 80, 81, 80, 79, 81, 79} {
		resize(m, width)
		if m.wide() != (width >= 81) || m.filter.FeedID != 1 || !m.filter.UnreadOnly || m.selected != 10 || m.reader == nil || m.reader.ID != 10 {
			t.Fatalf("width %d: %+v", width, m)
		}
		for _, line := range m.readerLines() {
			if ansi.StringWidth(line) > width {
				t.Fatalf("width %d: %q", width, line)
			}
		}
	}
	if r.calls != calls || f.calls != 0 {
		t.Fatalf("resize made requests: %d %d", r.calls, f.calls)
	}
	resize(m, 40)
	narrow := len(m.readerLines())
	resize(m, 100)
	if len(m.readerLines()) >= narrow {
		t.Fatal("reader did not reflow")
	}
}
func TestNavigationReadAndFilters(t *testing.T) {
	for _, width := range []int{79, 80, 81} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			m, r, f := fixture(t)
			resize(m, width)
			if !strings.Contains(m.View().Content, "Newest") {
				t.Fatal(m.View())
			}
			press(m, "j")
			if m.selected != 10 {
				t.Fatal(m.selected)
			}
			finish(m, press(m, "enter"))
			if m.reader == nil || !m.reader.Read || r.readCalls != 1 || f.calls != 0 {
				t.Fatalf("reader %+v, calls %d", m.reader, f.calls)
			}
			press(m, "esc")
			finish(m, press(m, "m"))
			if m.visible()[m.articleIndex].Read {
				t.Fatal("toggle failed")
			}
			press(m, "u")
			if !m.filter.UnreadOnly {
				t.Fatal("unread filter")
			}
			press(m, "k")
			if m.selected != 20 {
				t.Fatal(m.selected)
			}
			finish(m, press(m, "enter"))
			press(m, "pgdown")
			if m.scroll == 0 {
				t.Fatal("scroll")
			}
			press(m, "pgup")
			if m.scroll != 0 {
				t.Fatal("page up")
			}
			press(m, "down")
			press(m, "up")
			if m.scroll != 0 {
				t.Fatal("arrows")
			}
			press(m, "esc")
			if m.selected != 10 {
				t.Fatal("selection after unread removal", m.selected)
			}
			if width < 81 {
				before := m.focusFeeds
				press(m, "tab")
				if before != m.focusFeeds {
					t.Fatal("narrow Tab")
				}
			} else {
				press(m, "tab")
				press(m, "j")
				if m.filter.FeedID != 1 {
					t.Fatal("wide feed navigation")
				}
				press(m, "tab")
				if m.focusFeeds {
					t.Fatal("wide Tab")
				}
			}
		})
	}
}
func TestFeedPicker(t *testing.T) {
	m, r, f := fixture(t)
	resize(m, 79)
	press(m, "f")
	if !m.picker {
		t.Fatal("picker")
	}
	press(m, "j")
	press(m, "j")
	press(m, "k")
	press(m, "enter")
	if m.picker || m.filter.FeedID != 1 || m.selected != 10 {
		t.Fatal("apply feed")
	}
	press(m, "f")
	press(m, "j")
	press(m, "esc")
	if m.picker || m.filter.FeedID != 1 {
		t.Fatal("cancel changed filter")
	}
	press(m, "f")
	press(m, "k")
	press(m, "enter")
	if m.filter.FeedID != 0 || len(m.visible()) != 2 {
		t.Fatal("All feeds")
	}
	press(m, "f")
	press(m, "j")
	finish(m, press(m, "r"))
	if f.calls != 1 || !strings.Contains(m.status, "updated") {
		t.Fatal("refresh", m.status)
	}
	press(m, "a")
	if !m.input {
		t.Fatal("input")
	}
	for _, key := range "http://local.test/feed" {
		press(m, string(key))
	}
	finish(m, press(m, "enter"))
	if len(r.feeds) != 3 || m.input {
		t.Fatal("add feed")
	}
	press(m, "d")
	if !m.confirm {
		t.Fatal("confirmation")
	}
	press(m, "esc")
	if m.confirm || len(r.feeds) != 3 {
		t.Fatal("cancel removal")
	}
	press(m, "enter")
	press(m, "f")
	press(m, "d")
	finish(m, press(m, "y"))
	if len(r.feeds) != 2 || m.filter.FeedID != 0 || m.confirm || m.selected != 20 {
		t.Fatal("remove", m.filter, m.selected)
	}
	press(m, "a")
	press(m, "q")
	if m.url != "q" {
		t.Fatal("input q")
	}
	press(m, "esc")
}
func TestEmptyLoadingErrorsAndRefresh(t *testing.T) {
	r := &fakeRepository{}
	f := &fakeFetcher{}
	m := NewWithTheme(context.Background(), app.New(r, f), filepath.Join(t.TempDir(), "syn.toml"), false)
	if !strings.Contains(m.View().Content, "Loading") {
		t.Fatal("loading")
	}
	finish(m, m.work(nil))
	for _, width := range []int{79, 80} {
		resize(m, width)
		if !strings.Contains(m.View().Content, "No articles") {
			t.Fatal("empty")
		}
		for _, key := range []string{"j", "k", "enter", "m", "d", "u"} {
			finish(m, press(m, key))
		}
	}
	m.Update(dataMsg{err: errors.New("database failed")})
	if !strings.Contains(m.View().Content, "database failed") {
		t.Fatal("error")
	}
	m, r, f = fixture(t)
	f.fail = true
	finish(m, press(m, "R"))
	if f.calls != 2 || !strings.Contains(m.View().Content, "refresh failed") {
		t.Fatal("refresh error", m.status)
	}
	f.fail = false
	finish(m, press(m, "R"))
	if f.calls != 4 || m.feeds[0].Title != "Updated" {
		t.Fatal("refresh success")
	}
	press(m, "?")
	if !m.help {
		t.Fatal("help")
	}
	press(m, "esc")
	if m.help {
		t.Fatal("help close")
	}
	if press(m, "q") == nil || press(m, "ctrl+c") == nil {
		t.Fatal("quit")
	}
	_ = r
}
func TestTerminalText(t *testing.T) {
	m, _, _ := fixture(t)
	m.articles[0].Title = "safe\x1b[31mred\x1b]0;attack\a\u202e"
	m.feeds[0].Title = "bad\nfeed"
	view := m.View().Content
	if strings.ContainsAny(ansi.Strip(view), "\x1b\a\u202e") || strings.Contains(view, "attack") {
		t.Fatalf("unsafe %q", view)
	}
	for _, width := range []int{1, 10, 79, 80} {
		resize(m, width)
		for _, line := range strings.Split(m.View().Content, "\n") {
			if ansi.StringWidth(line) > width {
				t.Fatal(width, line)
			}
		}
	}
}

func TestKeyMessagesAndPaste(t *testing.T) {
	m, _, f := fixture(t)
	resize(m, 79)
	m.Update(tea.KeyPressMsg{Code: 'f', Text: "f"})
	if !m.picker {
		t.Fatal("key message did not open picker")
	}
	m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	m.Update(tea.PasteMsg{Content: "http://local.test/\x1b[31mfeed\x1b[0m"})
	if m.url != "http://local.test/feed" {
		t.Fatal(m.url)
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	m.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	finish(m, cmd)
	if m.input || len(m.feeds) != 3 {
		t.Fatal("input submission")
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	finish(m, press(m, "enter"))
	id := m.reader.ID
	press(m, "u")
	finish(m, press(m, "R"))
	if m.reader == nil || m.reader.ID != id || !m.filter.UnreadOnly || f.calls != 3 {
		t.Fatal("shared reader keys")
	}
}
