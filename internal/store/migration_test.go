package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/armada877/syn-cli/internal/domain"
)

const originalSchema = `CREATE TABLE feeds (id INTEGER PRIMARY KEY, url TEXT NOT NULL UNIQUE, title TEXT NOT NULL DEFAULT '', etag TEXT NOT NULL DEFAULT '', last_modified TEXT NOT NULL DEFAULT '', last_fetched_at TEXT NOT NULL DEFAULT '');
CREATE TABLE articles (id INTEGER PRIMARY KEY, feed_id INTEGER NOT NULL REFERENCES feeds(id) ON DELETE CASCADE, key TEXT NOT NULL, title TEXT NOT NULL, url TEXT NOT NULL, content TEXT NOT NULL, published_at TEXT NOT NULL, read INTEGER NOT NULL DEFAULT 0 CHECK (read IN (0, 1)), UNIQUE(feed_id, key));
PRAGMA user_version = 1;`

func TestMigrationPreservesAndRefreshes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v1.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 1, 2, 3, 4, 5, 600, time.UTC)
	for _, statement := range []string{originalSchema, `INSERT INTO feeds VALUES (7, 'https://example.com/feed', 'Feed', 'etag', 'modified', '` + stamp(at) + `')`, `INSERT INTO articles VALUES (9, 7, 'key', 'Title', 'https://example.com/article', 'Body', '` + stamp(at) + `', 1)`} {
		if _, err = db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	feeds, err := s.ListFeeds(ctx)
	if err != nil {
		t.Fatal(err)
	}
	wantFeed := domain.Feed{ID: 7, URL: "https://example.com/feed", Title: "Feed", ETag: "etag", LastModified: "modified", LastFetchedAt: at, MetadataPending: true}
	if len(feeds) != 1 || !reflect.DeepEqual(feeds[0], wantFeed) {
		t.Fatal(feeds)
	}
	articles, err := s.ListArticles(ctx, domain.ArticleFilter{})
	want := domain.Article{ID: 9, FeedID: 7, Key: "key", Title: "Title", URL: "https://example.com/article", Content: "Body", PublishedAt: at, Read: true}
	if err != nil || len(articles) != 1 || !reflect.DeepEqual(articles[0], want) {
		t.Fatal(articles, err)
	}
	var version int
	if err = s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 2 {
		t.Fatal(version, err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	feeds, err = s.ListFeeds(ctx)
	if err != nil || len(feeds) != 1 || !reflect.DeepEqual(feeds[0], wantFeed) {
		t.Fatal(feeds, err)
	}
	if err = s.ApplyFetch(ctx, 7, domain.FetchResult{NotModified: true, ETag: "etag", LastModified: "modified"}, at); err != nil {
		t.Fatal(err)
	}
	feeds, err = s.ListFeeds(ctx)
	if err != nil || !reflect.DeepEqual(feeds[0], wantFeed) {
		t.Fatal(feeds, err)
	}
	if _, err = s.db.Exec(`CREATE TRIGGER reject_metadata BEFORE INSERT ON articles BEGIN SELECT RAISE(ABORT, 'test failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err = s.ApplyFetch(ctx, 7, domain.FetchResult{Articles: []domain.ParsedArticle{{Key: "key"}}}, at); err == nil {
		t.Fatal("Expected transaction failure")
	}
	feeds, err = s.ListFeeds(ctx)
	if err != nil || !reflect.DeepEqual(feeds[0], wantFeed) {
		t.Fatal(feeds, err)
	}
	if _, err = s.db.Exec("DROP TRIGGER reject_metadata"); err != nil {
		t.Fatal(err)
	}
	result := domain.FetchResult{Title: "Feed", Articles: []domain.ParsedArticle{{Key: want.Key, Title: want.Title, URL: want.URL, Content: want.Content, Author: "Alice", Subtitle: "Summary"}}}
	if err = s.ApplyFetch(ctx, 7, result, at.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	feeds, err = s.ListFeeds(ctx)
	if err != nil || feeds[0].MetadataPending {
		t.Fatal(feeds, err)
	}
	want.Author, want.Subtitle = "Alice", "Summary"
	for _, notModified := range []bool{false, true} {
		if notModified {
			if err = s.ApplyFetch(ctx, 7, domain.FetchResult{NotModified: true}, at.Add(2*time.Hour)); err != nil {
				t.Fatal(err)
			}
		}
		articles, err = s.ListArticles(ctx, domain.ArticleFilter{})
		if err != nil || len(articles) != 1 || !reflect.DeepEqual(articles[0], want) {
			t.Fatal(articles, err)
		}
	}
}

func TestMigrationRollsBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken-v1.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(originalSchema + "ALTER TABLE articles ADD COLUMN subtitle TEXT NOT NULL DEFAULT ''; INSERT INTO articles VALUES (1, 1, 'key', 'Title', 'URL', 'Body', '2026-01-01T00:00:00Z', 1, 'Summary');"); err != nil {
		t.Fatal(err)
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	if s, err := Open(path); err == nil {
		s.Close()
		t.Fatal("Expected migration failure")
	}
	db, err = sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var version, authors int
	if err = db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 1 {
		t.Fatal(version, err)
	}
	if err = db.QueryRow("SELECT count(*) FROM pragma_table_info('articles') WHERE name = 'author'").Scan(&authors); err != nil || authors != 0 {
		t.Fatal(authors, err)
	}
	if err = db.QueryRow("SELECT count(*) FROM pragma_table_info('feeds') WHERE name = 'metadata_pending'").Scan(&authors); err != nil || authors != 0 {
		t.Fatal(authors, err)
	}
	var subtitle string
	if err = db.QueryRow("SELECT subtitle FROM articles WHERE id = 1").Scan(&subtitle); err != nil || subtitle != "Summary" {
		t.Fatal(subtitle, err)
	}
}
