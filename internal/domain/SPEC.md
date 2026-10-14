# Domain spec

This package uses only the Go standard library.
The types and interfaces in domain.go define the shared contracts.
Network and database operations take context.Context as their first argument.
Timestamps use time.Time. Stored timestamps use UTC.

Feed stores the feed ID, URL, title, HTTP validators, and last fetch time.
Article stores its ID, feed ID, stable key, title, URL, content, publication time, and read state.
ParsedArticle holds the fetched article fields before storage assigns an ID.
FetchResult holds the feed title, parsed articles, HTTP validators, and NotModified state.
ArticleFilter selects one feed by ID. Zero selects all feeds.
UnreadOnly selects unread articles when true.

Fetcher.Fetch returns a FetchResult or an error.
Repository.AddFeed returns an existing feed for a duplicate normalized URL.
Repository.ListFeeds returns feeds by ascending ID.
Repository.RemoveFeed removes the feed and its articles.
Repository.ApplyFetch stores the fetch result and fetch time.
An article key stays stable within its feed.
A later fetch preserves article IDs and read state when content changes.
An article without a publication time uses the current fetch time.
Repository.ListArticles sorts by descending publication time, then descending ID.
Repository.SetRead changes the article read state.
Repository.Close releases resources.
Unknown feed and article IDs return ErrNotFound.
Callers inspect ErrNotFound and ErrInvalidURL with errors.Is.

NormalizeFeedURL requires an HTTP or HTTPS scheme and a host.
It changes the scheme and host to lowercase and removes fragments.
It preserves paths and query strings.
The CLI, store, and fetcher use this helper.

Article and ParsedArticle also store Author and Subtitle strings.
These fields do not change repository methods or article keys.

Feed.MetadataPending tracks a required metadata refresh.
It does not depend on empty article fields.
