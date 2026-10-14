package feed

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/armada877/syn-cli/internal/domain"
	"github.com/mmcdole/gofeed"
)

const defaultTimeout = 15 * time.Second
const maxBodySize = 5 * 1024 * 1024

type Fetcher struct {
	client *http.Client
}

var _ domain.Fetcher = (*Fetcher)(nil)

func New(client *http.Client) *Fetcher {
	if client == nil {
		client = &http.Client{}
	}
	return &Fetcher{client: client}
}

func (f *Fetcher) Fetch(ctx context.Context, feed domain.Feed) (domain.FetchResult, error) {
	feedURL, err := domain.NormalizeFeedURL(feed.URL)
	if err != nil {
		return domain.FetchResult{}, fmt.Errorf("validate feed URL %q: %w", feed.URL, err)
	}
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return domain.FetchResult{}, fmt.Errorf("create request %q: %w", feedURL, err)
	}
	if feed.ETag != "" {
		request.Header.Set("If-None-Match", feed.ETag)
	}
	if feed.LastModified != "" {
		request.Header.Set("If-Modified-Since", feed.LastModified)
	}
	response, err := f.client.Do(request)
	if err != nil {
		return domain.FetchResult{}, fmt.Errorf("request feed %q: %w", feedURL, err)
	}
	defer response.Body.Close()
	result := domain.FetchResult{ETag: response.Header.Get("ETag"), LastModified: response.Header.Get("Last-Modified")}
	if response.StatusCode == http.StatusNotModified {
		result.NotModified = true
		if result.ETag == "" {
			result.ETag = feed.ETag
		}
		if result.LastModified == "" {
			result.LastModified = feed.LastModified
		}
		return result, nil
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return domain.FetchResult{}, fmt.Errorf("request feed %q: HTTP status %d", feedURL, response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxBodySize+1))
	if err != nil {
		return domain.FetchResult{}, fmt.Errorf("read feed %q: %w", feedURL, err)
	}
	if len(body) > maxBodySize {
		return domain.FetchResult{}, fmt.Errorf("read feed %q: body exceeds %d bytes", feedURL, maxBodySize)
	}
	parsed, err := gofeed.NewParser().Parse(strings.NewReader(string(body)))
	if err != nil {
		return domain.FetchResult{}, fmt.Errorf("parse feed %q: %w", feedURL, err)
	}
	result.Title = parsed.Title
	seen := make(map[string]bool)
	for _, item := range parsed.Items {
		article := normalizeArticle(item)
		if seen[article.Key] {
			continue
		}
		seen[article.Key] = true
		result.Articles = append(result.Articles, article)
	}
	return result, nil
}

func normalizeArticle(item *gofeed.Item) domain.ParsedArticle {
	article := domain.ParsedArticle{Title: item.Title, URL: item.Link, Content: item.Content, Subtitle: item.Description, Author: articleAuthors(item)}
	if article.Content == "" {
		article.Content = item.Description
	}
	if item.PublishedParsed != nil {
		article.PublishedAt = item.PublishedParsed.UTC()
	}
	article.Key = item.GUID
	if article.Key == "" {
		article.Key = item.Link
	}
	if article.Key == "" {
		fields, _ := json.Marshal([3]string{article.Title, article.PublishedAt.Format(time.RFC3339Nano), article.Content})
		digest := sha256.Sum256(fields)
		article.Key = hex.EncodeToString(digest[:])
	}
	return article
}

func articleAuthors(item *gofeed.Item) string {
	authors := item.Authors
	if len(authors) == 0 && item.Author != nil {
		authors = []*gofeed.Person{item.Author}
	}
	names := []string{}
	seen := map[string]bool{}
	for _, author := range authors {
		if author == nil {
			continue
		}
		name := strings.TrimSpace(author.Name)
		if name == "" {
			name = strings.TrimSpace(author.Email)
		}
		if name != "" && !seen[name] {
			names = append(names, name)
			seen[name] = true
		}
	}
	return strings.Join(names, ", ")
}
