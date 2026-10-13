package store

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/armada877/syn-cli/internal/domain"
)

func openTest(t *testing.T) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "nested", "reader.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s, path
}

func TestRefreshAndReopen(t *testing.T) {
	s, path := openTest(t)
	ctx := context.Background()
	feed, err := s.AddFeed(ctx, "HTTPS://EXAMPLE.COM/feed#fragment")
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := s.AddFeed(ctx, "https://example.com/feed")
	if err != nil || duplicate.ID != feed.ID {
		t.Fatalf("duplicate: %+v %v", duplicate, err)
	}
	if _, err = s.AddFeed(ctx, "file:///feed"); !errors.Is(err, domain.ErrInvalidURL) {
		t.Fatal(err)
	}
	at := time.Date(2026, 1, 1, 12, 0, 0, 0, time.FixedZone("offset", 3600))
	result := domain.FetchResult{Title: "Feed", ETag: "one", Articles: []domain.ParsedArticle{{Key: "a", Title: "old"}, {Key: "a", Title: "new"}}}
	if err = s.ApplyFetch(ctx, feed.ID, result, at); err != nil {
		t.Fatal(err)
	}
	articles, err := s.ListArticles(ctx, domain.ArticleFilter{})
	if err != nil || len(articles) != 1 {
		t.Fatalf("articles: %+v %v", articles, err)
	}
	original := articles[0]
	if original.Title != "new" || !original.PublishedAt.Equal(at) || original.PublishedAt.Location() != time.UTC {
		t.Fatal(original)
	}
	if err = s.SetRead(ctx, original.ID, true); err != nil {
		t.Fatal(err)
	}
	result.Articles[0].Title = "updated"
	result.Articles = result.Articles[:1]
	if err = s.ApplyFetch(ctx, feed.ID, result, at.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err = s.ApplyFetch(ctx, feed.ID, domain.FetchResult{NotModified: true, Title: "ignored", ETag: "two", LastModified: "validator", Articles: []domain.ParsedArticle{{Key: "ignored"}}}, at.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	articles, err = s.ListArticles(ctx, domain.ArticleFilter{})
	if err != nil || len(articles) != 1 {
		t.Fatalf("articles: %+v %v", articles, err)
	}
	if articles[0].ID != original.ID || !articles[0].Read || !articles[0].PublishedAt.Equal(at) || articles[0].Title != "updated" {
		t.Fatal(articles[0])
	}
	feeds, err := s.ListFeeds(ctx)
	if err != nil || len(feeds) != 1 {
		t.Fatalf("feeds: %+v %v", feeds, err)
	}
	if feeds[0].Title != "Feed" || feeds[0].ETag != "two" || feeds[0].LastModified != "validator" || !feeds[0].LastFetchedAt.Equal(at.Add(2*time.Hour)) {
		t.Fatal(feeds[0])
	}
	unread, err := s.ListArticles(ctx, domain.ArticleFilter{UnreadOnly: true})
	if err != nil || len(unread) != 0 {
		t.Fatalf("unread: %+v %v", unread, err)
	}
}

func TestOrderingFilteringAndDeletion(t *testing.T) {
	s, _ := openTest(t)
	ctx := context.Background()
	first, _ := s.AddFeed(ctx, "https://example.com/1")
	second, _ := s.AddFeed(ctx, "https://example.com/2")
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, feed := range []domain.Feed{first, second} {
		err := s.ApplyFetch(ctx, feed.ID, domain.FetchResult{Articles: []domain.ParsedArticle{
			{Key: "zero", PublishedAt: at}, {Key: "fraction", PublishedAt: at.Add(100 * time.Millisecond)},
			{Key: "later", PublishedAt: at.Add(200 * time.Millisecond)}, {Key: "tie", PublishedAt: at.Add(200 * time.Millisecond)},
		}}, at)
		if err != nil {
			t.Fatal(err)
		}
	}
	feeds, err := s.ListFeeds(ctx)
	if err != nil || len(feeds) != 2 || feeds[0].ID != first.ID || feeds[1].ID != second.ID {
		t.Fatalf("feeds: %+v %v", feeds, err)
	}
	articles, err := s.ListArticles(ctx, domain.ArticleFilter{FeedID: first.ID})
	if err != nil || len(articles) != 4 {
		t.Fatalf("articles: %+v %v", articles, err)
	}
	for i, key := range []string{"tie", "later", "fraction", "zero"} {
		if articles[i].Key != key {
			t.Fatal(articles)
		}
	}
	if err = s.SetRead(ctx, articles[0].ID, true); err != nil {
		t.Fatal(err)
	}
	unread, err := s.ListArticles(ctx, domain.ArticleFilter{FeedID: first.ID, UnreadOnly: true})
	if err != nil || len(unread) != 3 {
		t.Fatalf("unread: %+v %v", unread, err)
	}
	if err = s.RemoveFeed(ctx, first.ID); err != nil {
		t.Fatal(err)
	}
	all, err := s.ListArticles(ctx, domain.ArticleFilter{})
	if err != nil || len(all) != 4 {
		t.Fatalf("remaining: %+v %v", all, err)
	}
	for _, article := range all {
		if article.FeedID != second.ID {
			t.Fatal(article)
		}
	}
	for _, err := range []error{s.RemoveFeed(ctx, first.ID), s.SetRead(ctx, articles[0].ID, false), s.ApplyFetch(ctx, first.ID, domain.FetchResult{}, at)} {
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatal(err)
		}
	}
}

func TestRollbackAndNewerSchema(t *testing.T) {
	s, path := openTest(t)
	ctx := context.Background()
	feed, _ := s.AddFeed(ctx, "https://example.com/feed")
	if _, err := s.db.Exec(`CREATE TRIGGER reject_article BEFORE INSERT ON articles WHEN NEW.key = 'fail' BEGIN SELECT RAISE(ABORT, 'test failure'); END`); err != nil {
		t.Fatal(err)
	}
	err := s.ApplyFetch(ctx, feed.ID, domain.FetchResult{Title: "changed", Articles: []domain.ParsedArticle{{Key: "ok"}, {Key: "fail"}}}, time.Now())
	if err == nil {
		t.Fatal("expected transaction failure")
	}
	feeds, err := s.ListFeeds(ctx)
	if err != nil || feeds[0].Title != "" || !feeds[0].LastFetchedAt.IsZero() {
		t.Fatalf("feeds: %+v %v", feeds, err)
	}
	articles, err := s.ListArticles(ctx, domain.ArticleFilter{})
	if err != nil || len(articles) != 0 {
		t.Fatalf("articles: %+v %v", articles, err)
	}
	if _, err = s.db.Exec("PRAGMA user_version = 2"); err != nil {
		t.Fatal(err)
	}
	s.Close()
	if opened, err := Open(path); err == nil {
		opened.Close()
		t.Fatal("expected schema rejection")
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	s = &Store{db: db}
	defer s.Close()
	var version int
	if err = s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 2 {
		t.Fatalf("version: %d %v", version, err)
	}
	feeds, err = s.ListFeeds(ctx)
	if err != nil || len(feeds) != 1 || feeds[0].ID != feed.ID {
		t.Fatalf("feeds: %+v %v", feeds, err)
	}
}

func TestPermissionsAndInitializationRollback(t *testing.T) {
	s, path := openTest(t)
	if runtime.GOOS != "windows" {
		for name, want := range map[string]os.FileMode{path: 0600, filepath.Dir(path): 0700} {
			info, err := os.Stat(name)
			if err != nil || info.Mode().Perm() != want {
				t.Fatalf("permissions: %s %v %v", name, info, err)
			}
		}
	}
	var foreignKeys, timeout int
	if err := s.db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil || foreignKeys != 1 {
		t.Fatalf("foreign keys: %d %v", foreignKeys, err)
	}
	if err := s.db.QueryRow("PRAGMA busy_timeout").Scan(&timeout); err != nil || timeout != 5000 {
		t.Fatalf("timeout: %d %v", timeout, err)
	}
	badPath := filepath.Join(t.TempDir(), "partial.db")
	db, err := sql.Open("sqlite", badPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec("CREATE TABLE articles (value TEXT); INSERT INTO articles VALUES ('preserved')"); err != nil {
		t.Fatal(err)
	}
	if opened, err := Open(badPath); err == nil {
		opened.Close()
		t.Fatal("expected initialization failure")
	}
	var value string
	if err = db.QueryRow("SELECT value FROM articles").Scan(&value); err != nil || value != "preserved" {
		t.Fatalf("value: %s %v", value, err)
	}
	var count, version int
	if err = db.QueryRow("SELECT count(*) FROM sqlite_master WHERE name = 'feeds'").Scan(&count); err != nil || count != 0 {
		t.Fatalf("tables: %d %v", count, err)
	}
	if err = db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 0 {
		t.Fatalf("version: %d %v", version, err)
	}
}
