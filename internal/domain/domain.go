package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrInvalidURL = errors.New("invalid feed URL")
)

type Feed struct {
	ID              int64
	URL             string
	Title           string
	ETag            string
	LastModified    string
	LastFetchedAt   time.Time
	MetadataPending bool
}

type Article struct {
	ID          int64
	FeedID      int64
	Key         string
	Title       string
	URL         string
	Author      string
	Subtitle    string
	Content     string
	PublishedAt time.Time
	Read        bool
}

type ParsedArticle struct {
	Key         string
	Title       string
	URL         string
	Author      string
	Subtitle    string
	Content     string
	PublishedAt time.Time
}

type FetchResult struct {
	Title        string
	Articles     []ParsedArticle
	ETag         string
	LastModified string
	NotModified  bool
}

type ArticleFilter struct {
	FeedID     int64
	UnreadOnly bool
}

type Fetcher interface {
	Fetch(ctx context.Context, feed Feed) (FetchResult, error)
}

type Repository interface {
	AddFeed(ctx context.Context, url string) (Feed, error)
	ListFeeds(ctx context.Context) ([]Feed, error)
	RemoveFeed(ctx context.Context, id int64) error
	ApplyFetch(ctx context.Context, feedID int64, result FetchResult, fetchedAt time.Time) error
	ListArticles(ctx context.Context, filter ArticleFilter) ([]Article, error)
	SetRead(ctx context.Context, articleID int64, read bool) error
	Close() error
}
