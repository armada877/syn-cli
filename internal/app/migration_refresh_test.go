package app_test

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/armada877/syn-cli/internal/app"
	"github.com/armada877/syn-cli/internal/domain"
	"github.com/armada877/syn-cli/internal/feed"
	"github.com/armada877/syn-cli/internal/store"
)

func conditionalHandler(t *testing.T) http.Handler {
	t.Helper()
	requests := 0
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		etag, modified := r.Header.Get("If-None-Match"), r.Header.Get("If-Modified-Since")
		if requests <= 3 && (etag != "" || modified != "") {
			t.Errorf("Pending request includes validators: %q %q", etag, modified)
		}
		if requests >= 4 && (etag != "original" || modified != "original-date") {
			t.Errorf("Conditional request lacks validators: %q %q", etag, modified)
		}
		w.Header().Set("ETag", "original")
		w.Header().Set("Last-Modified", "original-date")
		if requests == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		if requests == 2 || etag == "original" || modified == "original-date" {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		io.WriteString(w, `<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/"><channel><title>Feed</title><item><guid>key</guid><title>Title</title><link>https://example.com/article</link><author>alice@example.com</author><description>Summary</description><content:encoded>Body</content:encoded></item><item><guid>empty</guid><title>Empty</title></item></channel></rss>`)
	})
}

func verifyMigrationRefresh(t *testing.T, url string, client *http.Client) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "v1.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	schema := `CREATE TABLE feeds (id INTEGER PRIMARY KEY, url TEXT NOT NULL UNIQUE, title TEXT NOT NULL DEFAULT '', etag TEXT NOT NULL DEFAULT '', last_modified TEXT NOT NULL DEFAULT '', last_fetched_at TEXT NOT NULL DEFAULT '');
CREATE TABLE articles (id INTEGER PRIMARY KEY, feed_id INTEGER NOT NULL REFERENCES feeds(id) ON DELETE CASCADE, key TEXT NOT NULL, title TEXT NOT NULL, url TEXT NOT NULL, content TEXT NOT NULL, published_at TEXT NOT NULL, read INTEGER NOT NULL DEFAULT 0 CHECK (read IN (0, 1)), UNIQUE(feed_id, key));
PRAGMA user_version = 1;`
	if _, err = db.Exec(schema); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	if _, err = db.Exec(`INSERT INTO feeds VALUES (7, ?, 'Feed', 'original', 'original-date', ?);`, url, at.Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO articles VALUES (9, 7, 'key', 'Title', 'https://example.com/article', 'Body', ?, 1)`, at.Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	repository, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer repository.Close()
	ctx := context.Background()
	a := app.New(repository, feed.New(client))
	a.Now = func() time.Time { return at.Add(time.Hour) }
	original, err := a.ListFeeds(ctx)
	if err != nil || len(original) != 1 || !original[0].MetadataPending || original[0].ETag != "original" || original[0].LastModified != "original-date" {
		t.Fatal(original, err)
	}
	for step := 0; step < 4; step++ {
		results, err := a.Refresh(ctx, 7)
		if err != nil || len(results) != 1 {
			t.Fatal(results, err)
		}
		if (step == 0) != (results[0].Err != nil) {
			t.Fatal(step, results)
		}
		if results[0].NotModified != (step == 1 || step == 3) {
			t.Fatal(step, results)
		}
		if !reflect.DeepEqual(results[0].Feed, original[0]) && step == 0 {
			t.Fatal("Request copy changed result feed")
		}
		feeds, err := a.ListFeeds(ctx)
		if err != nil || len(feeds) != 1 || feeds[0].MetadataPending != (step < 2) || feeds[0].ETag != "original" || feeds[0].LastModified != "original-date" {
			t.Fatal(step, feeds, err)
		}
		articles, err := a.ListArticles(ctx, domain.ArticleFilter{})
		if err != nil {
			t.Fatal(err)
		}
		var article domain.Article
		for _, stored := range articles {
			if stored.Key == "key" {
				article = stored
			}
		}
		want := domain.Article{ID: 9, FeedID: 7, Key: "key", Title: "Title", URL: "https://example.com/article", Content: "Body", PublishedAt: at, Read: true}
		if step >= 2 {
			want.Author, want.Subtitle = "alice@example.com", "Summary"
		}
		if !reflect.DeepEqual(article, want) {
			t.Fatal(step, article, want)
		}
	}
}

func TestMigrationConditionalServer(t *testing.T) {
	server := httptest.NewServer(conditionalHandler(t))
	defer server.Close()
	verifyMigrationRefresh(t, server.URL, server.Client())
}

type conditionalTransport struct{ handler http.Handler }

func (transport conditionalTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	response := httptest.NewRecorder()
	transport.handler.ServeHTTP(response, request)
	return response.Result(), nil
}

func TestMigrationConditionalTransport(t *testing.T) {
	verifyMigrationRefresh(t, "https://example.com/feed", &http.Client{Transport: conditionalTransport{conditionalHandler(t)}})
}

func TestFreshFeedEmptyMetadata(t *testing.T) {
	repository, err := store.Open(filepath.Join(t.TempDir(), "fresh.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repository.Close()
	ctx := context.Background()
	f, err := repository.AddFeed(ctx, "https://example.com/feed")
	if err != nil || f.MetadataPending {
		t.Fatal(f, err)
	}
	if err = repository.ApplyFetch(ctx, f.ID, domain.FetchResult{ETag: "tag", LastModified: "date", Articles: []domain.ParsedArticle{{Key: "empty"}}}, time.Time{}); err != nil {
		t.Fatal(err)
	}
	a := app.New(repository, feed.New(&http.Client{Transport: conditionalTransport{http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-None-Match") != "tag" || r.Header.Get("If-Modified-Since") != "date" {
			t.Error("Empty metadata disabled validators")
		}
		w.WriteHeader(http.StatusNotModified)
	})}}))
	results, err := a.Refresh(ctx, f.ID)
	if err != nil || len(results) != 1 || results[0].Err != nil || !results[0].NotModified {
		t.Fatal(results, err)
	}
	articles, err := a.ListArticles(ctx, domain.ArticleFilter{})
	if err != nil || len(articles) != 1 || strings.TrimSpace(articles[0].Author+articles[0].Subtitle) != "" {
		t.Fatal(articles, err)
	}
}
