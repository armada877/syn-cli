package feed

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/armada877/syn-cli/internal/domain"
	"github.com/mmcdole/gofeed"
)

func TestFixtures(t *testing.T) {
	for _, name := range []string{"rss", "atom"} {
		t.Run(name, func(t *testing.T) {
			body, err := os.ReadFile("testdata/" + name + ".xml")
			if err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("method = %s", r.Method)
				}
				w.Header().Set("ETag", "new-tag")
				w.Header().Set("Last-Modified", "new-date")
				w.Write(body)
			}))
			defer server.Close()
			result, err := New(server.Client()).Fetch(context.Background(), domain.Feed{URL: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			if result.ETag != "new-tag" || result.LastModified != "new-date" || result.NotModified {
				t.Fatalf("result = %+v", result)
			}
			expectedTitle := "RSS sample"
			if name == "atom" {
				expectedTitle = "Atom sample"
			}
			if result.Title != expectedTitle {
				t.Fatalf("title = %q", result.Title)
			}
			first := result.Articles[0]
			if !first.PublishedAt.Equal(time.Date(2026, 10, 13, 10, 0, 0, 0, time.UTC)) || first.PublishedAt.Location() != time.UTC {
				t.Fatalf("date = %v", first.PublishedAt)
			}
			if name == "rss" {
				if len(result.Articles) != 3 || first.Key != "one" || first.Title != "First" || first.URL != "https://example.com/one" || first.Content != "<p>Full content</p>" {
					t.Fatalf("articles = %+v", result.Articles)
				}
				second := result.Articles[1]
				if second.Key != second.URL || second.Content != "Second summary" || !second.PublishedAt.IsZero() {
					t.Fatalf("second = %+v", second)
				}
				fallback := result.Articles[2]
				if len(fallback.Key) != 64 || !fallback.PublishedAt.IsZero() {
					t.Fatalf("fallback = %+v", fallback)
				}
				again, err := New(nil).Fetch(context.Background(), domain.Feed{URL: server.URL})
				if err != nil || again.Articles[2].Key != fallback.Key {
					t.Fatalf("unstable key: %+v, %v", again, err)
				}
			} else {
				if len(result.Articles) != 2 || first.Key != "urn:article" || first.Title != "Atom article" || first.URL != "https://example.com/atom" || first.Content != "<p>Atom content</p>" || !result.Articles[1].PublishedAt.IsZero() {
					t.Fatalf("articles = %+v", result.Articles)
				}
			}
		})
	}
}

func TestNotModified(t *testing.T) {
	for _, replace := range []bool{false, true} {
		t.Run(fmt.Sprint(replace), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("If-None-Match") != "old-tag" || r.Header.Get("If-Modified-Since") != "old-date" {
					t.Errorf("headers = %v", r.Header)
				}
				if replace {
					w.Header().Set("ETag", "new-tag")
					w.Header().Set("Last-Modified", "new-date")
				}
				w.WriteHeader(http.StatusNotModified)
			}))
			defer server.Close()
			result, err := New(nil).Fetch(context.Background(), domain.Feed{URL: server.URL, ETag: "old-tag", LastModified: "old-date"})
			tag, date := "old-tag", "old-date"
			if replace {
				tag, date = "new-tag", "new-date"
			}
			if err != nil || !result.NotModified || result.ETag != tag || result.LastModified != date || len(result.Articles) != 0 {
				t.Fatalf("result = %+v, error = %v", result, err)
			}
		})
	}
}

func TestFailures(t *testing.T) {
	cases := []struct {
		name    string
		status  int
		body    string
		message string
	}{
		{"HTTP", 503, "private response", "HTTP status 503"},
		{"redirect", 302, "private response", "HTTP status 302"},
		{"malformed", 200, "<rss><channel>", "parse feed"},
		{"oversized", 200, strings.Repeat("x", maxBodySize+1), "body exceeds"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(tc.status); io.WriteString(w, tc.body) }))
			defer server.Close()
			_, err := New(nil).Fetch(context.Background(), domain.Feed{URL: server.URL})
			if err == nil || !strings.Contains(err.Error(), tc.message) || !strings.Contains(err.Error(), server.URL) || strings.Contains(err.Error(), "private response") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestBodyBoundary(t *testing.T) {
	body, err := os.ReadFile("testdata/rss.xml")
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, []byte(strings.Repeat(" ", maxBodySize-len(body)))...)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(body) }))
	defer server.Close()
	if _, err := New(nil).Fetch(context.Background(), domain.Feed{URL: server.URL}); err != nil {
		t.Fatal(err)
	}
}

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRequestContext(t *testing.T) {
	client := &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		deadline, ok := r.Context().Deadline()
		remaining := time.Until(deadline)
		if !ok || remaining <= 14*time.Second || remaining > defaultTimeout {
			t.Errorf("remaining timeout = %v", remaining)
		}
		return nil, errors.New("transport failure")
	})}
	_, err := New(client).Fetch(context.Background(), domain.Feed{URL: "https://example.invalid/feed"})
	if err == nil || !strings.Contains(err.Error(), "request feed") || !strings.Contains(err.Error(), "https://example.invalid/feed") {
		t.Fatalf("error = %v", err)
	}
	if client.Timeout != 0 {
		t.Fatal("client changed")
	}
}

func TestCancellation(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(started); <-r.Context().Done() }))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	finished := make(chan error, 1)
	go func() { _, err := New(nil).Fetch(ctx, domain.Feed{URL: server.URL}); finished <- err }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("request did not start")
	}
	cancel()
	select {
	case err := <-finished:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("request did not stop")
	}
}

func TestInvalidURL(t *testing.T) {
	_, err := New(nil).Fetch(context.Background(), domain.Feed{URL: "file:///feed"})
	if !errors.Is(err, domain.ErrInvalidURL) || !strings.Contains(err.Error(), "file:///feed") {
		t.Fatalf("error = %v", err)
	}
}

func TestNormalization(t *testing.T) {
	for _, name := range []string{"rss", "atom"} {
		t.Run(name, func(t *testing.T) {
			file, err := os.Open("testdata/" + name + ".xml")
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			parsed, err := gofeed.NewParser().Parse(file)
			if err != nil {
				t.Fatal(err)
			}
			first := normalizeArticle(parsed.Items[0])
			expectedContent := "<p>Full content</p>"
			if name == "atom" {
				expectedContent = "<p>Atom content</p>"
			}
			if first.Content != expectedContent || !first.PublishedAt.Equal(time.Date(2026, 10, 13, 10, 0, 0, 0, time.UTC)) {
				t.Fatalf("article = %+v", first)
			}
			missing := normalizeArticle(parsed.Items[len(parsed.Items)-1])
			if !missing.PublishedAt.IsZero() || missing.Content == "" {
				t.Fatalf("article = %+v", missing)
			}
		})
	}
	base := &gofeed.Item{Title: "Title", Description: "Content"}
	key := normalizeArticle(base).Key
	if len(key) != 64 || key != normalizeArticle(base).Key {
		t.Fatal("fallback key is not stable")
	}
	for _, item := range []*gofeed.Item{{Title: "Other", Description: "Content"}, {Title: "Title", Description: "Other"}, {Title: "Title", Description: "Content", PublishedParsed: new(time.Time)}} {
		if item.PublishedParsed != nil {
			*item.PublishedParsed = time.Unix(1, 0)
		}
		if normalizeArticle(item).Key == key {
			t.Fatal("fallback key did not change")
		}
	}
}
