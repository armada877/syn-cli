package store

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/armada877/syn-cli/internal/domain"
	_ "modernc.org/sqlite"
)

type Store struct{ db *sql.DB }

var _ domain.Repository = (*Store)(nil)

func Open(path string) (*Store, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(filepath.Dir(absolute), 0700); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(absolute, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err == nil {
		err = file.Close()
	} else if os.IsExist(err) {
		err = nil
	}
	if err != nil {
		return nil, err
	}
	location := url.URL{Scheme: "file", Path: absolute}
	query := location.Query()
	query.Add("_pragma", "foreign_keys(1)")
	query.Add("_pragma", "busy_timeout(5000)")
	location.RawQuery = query.Encode()
	db, err := sql.Open("sqlite", location.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	store := &Store{db: db}
	if err = store.initialize(); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) initialize() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var version int
	if err = tx.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	if version > 2 {
		return fmt.Errorf("unsupported schema version: %d", version)
	}
	if version == 0 {
		statements := []string{
			`CREATE TABLE feeds (id INTEGER PRIMARY KEY, url TEXT NOT NULL UNIQUE, title TEXT NOT NULL DEFAULT '', etag TEXT NOT NULL DEFAULT '', last_modified TEXT NOT NULL DEFAULT '', last_fetched_at TEXT NOT NULL DEFAULT '', metadata_pending INTEGER NOT NULL DEFAULT 0 CHECK (metadata_pending IN (0, 1)))`,
			`CREATE TABLE articles (id INTEGER PRIMARY KEY, feed_id INTEGER NOT NULL REFERENCES feeds(id) ON DELETE CASCADE, key TEXT NOT NULL, title TEXT NOT NULL, url TEXT NOT NULL, content TEXT NOT NULL, author TEXT NOT NULL DEFAULT '', subtitle TEXT NOT NULL DEFAULT '', published_at TEXT NOT NULL, read INTEGER NOT NULL DEFAULT 0 CHECK (read IN (0, 1)), UNIQUE(feed_id, key))`,
			`PRAGMA user_version = 2`,
		}
		for _, statement := range statements {
			if _, err = tx.Exec(statement); err != nil {
				return err
			}
		}
	}
	if version == 1 {
		for _, statement := range []string{"ALTER TABLE feeds ADD COLUMN metadata_pending INTEGER NOT NULL DEFAULT 0 CHECK (metadata_pending IN (0, 1))", "UPDATE feeds SET metadata_pending = 1", "ALTER TABLE articles ADD COLUMN author TEXT NOT NULL DEFAULT ''", "ALTER TABLE articles ADD COLUMN subtitle TEXT NOT NULL DEFAULT ''", "PRAGMA user_version = 2"} {
			if _, err = tx.Exec(statement); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func (s *Store) Close() error { return s.db.Close() }

func stamp(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }
func parseStamp(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339Nano, value)
}

type scanner interface{ Scan(...any) error }

func scanFeed(row scanner) (domain.Feed, error) {
	var feed domain.Feed
	var fetched string
	err := row.Scan(&feed.ID, &feed.URL, &feed.Title, &feed.ETag, &feed.LastModified, &fetched, &feed.MetadataPending)
	if err != nil {
		return feed, err
	}
	feed.LastFetchedAt, err = parseStamp(fetched)
	return feed, err
}

func (s *Store) AddFeed(ctx context.Context, raw string) (domain.Feed, error) {
	normalized, err := domain.NormalizeFeedURL(raw)
	if err != nil {
		return domain.Feed{}, err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO feeds(url) VALUES (?) ON CONFLICT(url) DO NOTHING`, normalized)
	if err != nil {
		return domain.Feed{}, err
	}
	return scanFeed(s.db.QueryRowContext(ctx, `SELECT id, url, title, etag, last_modified, last_fetched_at, metadata_pending FROM feeds WHERE url = ?`, normalized))
}

func (s *Store) ListFeeds(ctx context.Context) ([]domain.Feed, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, url, title, etag, last_modified, last_fetched_at, metadata_pending FROM feeds ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	feeds := []domain.Feed{}
	for rows.Next() {
		feed, err := scanFeed(rows)
		if err != nil {
			return nil, err
		}
		feeds = append(feeds, feed)
	}
	return feeds, rows.Err()
}

func requireRow(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) RemoveFeed(ctx context.Context, id int64) error {
	return requireRow(s.db.ExecContext(ctx, `DELETE FROM feeds WHERE id = ?`, id))
}

func (s *Store) ApplyFetch(ctx context.Context, feedID int64, result domain.FetchResult, fetchedAt time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if result.NotModified {
		err = requireRow(tx.ExecContext(ctx, `UPDATE feeds SET etag = ?, last_modified = ?, last_fetched_at = ? WHERE id = ?`, result.ETag, result.LastModified, stamp(fetchedAt), feedID))
	} else {
		err = requireRow(tx.ExecContext(ctx, `UPDATE feeds SET title = ?, etag = ?, last_modified = ?, last_fetched_at = ?, metadata_pending = 0 WHERE id = ?`, result.Title, result.ETag, result.LastModified, stamp(fetchedAt), feedID))
	}
	if err != nil {
		return err
	}
	if !result.NotModified {
		for _, article := range result.Articles {
			published := article.PublishedAt
			missing := published.IsZero()
			if missing {
				published = fetchedAt
			}
			_, err = tx.ExecContext(ctx, `INSERT INTO articles(feed_id, key, title, url, content, author, subtitle, published_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(feed_id, key) DO UPDATE SET title = excluded.title, url = excluded.url, content = excluded.content, author = excluded.author, subtitle = excluded.subtitle, published_at = CASE WHEN ? THEN articles.published_at ELSE excluded.published_at END`, feedID, article.Key, article.Title, article.URL, article.Content, article.Author, article.Subtitle, stamp(published), missing)
			if err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func (s *Store) ListArticles(ctx context.Context, filter domain.ArticleFilter) ([]domain.Article, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, feed_id, key, title, url, content, author, subtitle, published_at, read FROM articles WHERE (? = 0 OR feed_id = ?) AND (? = 0 OR read = 0)`, filter.FeedID, filter.FeedID, filter.UnreadOnly)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	articles := []domain.Article{}
	for rows.Next() {
		var article domain.Article
		var published string
		if err = rows.Scan(&article.ID, &article.FeedID, &article.Key, &article.Title, &article.URL, &article.Content, &article.Author, &article.Subtitle, &published, &article.Read); err != nil {
			return nil, err
		}
		article.PublishedAt, err = parseStamp(published)
		if err != nil {
			return nil, err
		}
		articles = append(articles, article)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(articles, func(i, j int) bool {
		if articles[i].PublishedAt.Equal(articles[j].PublishedAt) {
			return articles[i].ID > articles[j].ID
		}
		return articles[i].PublishedAt.After(articles[j].PublishedAt)
	})
	return articles, nil
}

func (s *Store) SetRead(ctx context.Context, articleID int64, read bool) error {
	return requireRow(s.db.ExecContext(ctx, `UPDATE articles SET read = ? WHERE id = ?`, read, articleID))
}
