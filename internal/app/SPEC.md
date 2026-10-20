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

ArticleSubtitle converts the description and body with PlainText.
It omits empty descriptions and descriptions equal to the body after whitespace normalization.
ArticleByline converts the author to safe plain text and prefixes nonempty text with By.
ArticleDisplayText prints title and URL, optional subtitle and byline, then body paragraphs.

Refresh copies each feed before making its request.
For MetadataPending feeds, it clears ETag and LastModified only on that request copy.
Stored validators and the feed in RefreshResult remain intact.
Failures and HTTP 304 leave metadata pending for another refresh.

ArticleBlocks returns ordered text and image blocks from stored HTML.
Skip script, style, and head elements.
Keep paragraph boundaries and remove terminal controls.
Resolve image sources against the article URL.
Accept only absolute HTTP or HTTPS results without credentials.
Keep invalid images as placeholders without requests.
Flatten and sanitize alt text.
Use [image: alt text], or [image] when alt text is empty.
PlainText includes each placeholder at its original position.
