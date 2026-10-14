package app

import (
	"context"
	"fmt"
	"github.com/armada877/syn-cli/internal/domain"
	"time"
)

type App struct {
	Repository domain.Repository
	Fetcher    domain.Fetcher
	Now        func() time.Time
}
type RefreshResult struct {
	Feed        domain.Feed
	Err         error
	NotModified bool
	Count       int
}

func New(repository domain.Repository, fetcher domain.Fetcher) *App {
	return &App{repository, fetcher, time.Now}
}
func (a *App) AddFeed(ctx context.Context, raw string) (domain.Feed, error) {
	url, err := domain.NormalizeFeedURL(raw)
	if err != nil {
		return domain.Feed{}, err
	}
	return a.Repository.AddFeed(ctx, url)
}
func (a *App) ListFeeds(ctx context.Context) ([]domain.Feed, error) {
	return a.Repository.ListFeeds(ctx)
}
func (a *App) RemoveFeed(ctx context.Context, id int64) error {
	return a.Repository.RemoveFeed(ctx, id)
}
func (a *App) ListArticles(ctx context.Context, f domain.ArticleFilter) ([]domain.Article, error) {
	return a.Repository.ListArticles(ctx, f)
}
func (a *App) SetRead(ctx context.Context, id int64, read bool) error {
	return a.Repository.SetRead(ctx, id, read)
}
func (a *App) Article(ctx context.Context, id int64) (domain.Article, error) {
	articles, err := a.ListArticles(ctx, domain.ArticleFilter{})
	if err != nil {
		return domain.Article{}, err
	}
	for _, article := range articles {
		if article.ID == id {
			return article, nil
		}
	}
	return domain.Article{}, domain.ErrNotFound
}
func (a *App) Refresh(ctx context.Context, id int64) ([]RefreshResult, error) {
	feeds, err := a.ListFeeds(ctx)
	if err != nil {
		return nil, err
	}
	results := []RefreshResult{}
	found := id == 0
	for _, feed := range feeds {
		if id != 0 && feed.ID != id {
			continue
		}
		found = true
		if err := ctx.Err(); err != nil {
			return results, err
		}
		requestFeed := feed
		if requestFeed.MetadataPending {
			requestFeed.ETag = ""
			requestFeed.LastModified = ""
		}
		result, err := a.Fetcher.Fetch(ctx, requestFeed)
		if err == nil {
			err = a.Repository.ApplyFetch(ctx, feed.ID, result, a.Now())
		}
		results = append(results, RefreshResult{feed, err, result.NotModified, len(result.Articles)})
	}
	if !found {
		return nil, fmt.Errorf("feed %d: %w", id, domain.ErrNotFound)
	}
	return results, nil
}
