# Store spec

This package implements domain.Repository with modernc.org/sqlite.
It depends on internal/domain and database/sql.
Use domain.NormalizeFeedURL before adding a feed.
Database operations take context.Context first and respect cancellation.
Store timestamps in UTC.
Return feeds by ascending ID.
Return articles by descending publication time, then descending ID.
Zero ArticleFilter.FeedID selects all feeds.
UnreadOnly selects unread articles when true.
AddFeed returns the existing feed for a duplicate normalized URL.
RemoveFeed removes its articles.
Unknown feed and article IDs return domain.ErrNotFound.
ApplyFetch uses fetchedAt when an article has no publication time.
Match articles by their stable key within the feed.
Preserve article IDs and read state when content changes.
Apply each fetch in one transaction.
Close releases database resources.
Database operations are not implemented yet.
