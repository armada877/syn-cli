# App spec

App connects domain.Fetcher and domain.Repository.
New accepts both interfaces and uses time.Now for the fetch time.
Tests can replace Now.
All storage and network operations receive the supplied context.
AddFeed normalizes the URL before storage.
Article searches stored articles and returns ErrNotFound for an unknown ID.
ListFeeds, RemoveFeed, ListArticles, and SetRead use the repository.

Refresh selects one feed by ID or all feeds with ID zero.
Refresh processes feeds sequentially in repository order.
Each successful fetch reaches ApplyFetch with the current time.
A failed fetch or storage update produces a separate RefreshResult.
Other feeds continue after an operation failure.
Successful updates remain stored when another feed fails.
Cancellation stops further fetches and returns the context error.
An unknown feed ID returns ErrNotFound.

PlainText converts HTML to text and separates block content with blank lines.
It removes script, style, and head content.
SafeText removes terminal escape sequences, control characters, and Unicode format characters.
It retains newlines and tabs for text layout.
ArticleText combines the safe title, safe URL, and converted content.
