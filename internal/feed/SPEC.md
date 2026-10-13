# Feed spec

This package implements domain.Fetcher.
It depends on internal/domain, net/http, and github.com/mmcdole/gofeed.
Fetch takes context.Context first and respects cancellation.
Use domain.NormalizeFeedURL before a request.
Use Feed.ETag and Feed.LastModified as HTTP validators.
Return NotModified for an unchanged feed.
Return parsed articles with stable keys within each feed.
Convert publication timestamps to UTC.
Leave missing publication times empty for the repository to fill.
Keep external parser types inside this package.
Feed requests are not implemented yet.
