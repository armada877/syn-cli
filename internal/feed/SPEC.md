# Feed spec

This package implements domain.Fetcher with gofeed.
New accepts an HTTP client. A nil client selects a default client.
Fetch validates the URL with domain.NormalizeFeedURL.
Fetch sends a GET request with the supplied context.
Each request has a 15-second timeout, including requests through an injected client.
An earlier context deadline or client timeout also applies.
Fetch does not change the injected client.

Feed.ETag sets If-None-Match. Feed.LastModified sets If-Modified-Since.
HTTP 304 returns NotModified without parsing the body.
Missing validators in HTTP 304 retain the supplied values.
Other responses must have a status from 200 through 299.
The response body must not exceed 5 MiB.
Errors identify the operation and URL. Errors do not include response bodies.
Wrapped errors support errors.Is.

RSS and Atom articles retain source titles, links, and content.
Content takes priority over description.
Publication dates use UTC. Missing publication dates remain zero.
Article keys use GUID first, then link, then a SHA-256 digest.
The digest uses title, publication date, and selected content.
Duplicate keys retain the first article in source order.
Successful results include the feed title and response validators.
This package does not change stored data.

Subtitle stores the raw description, even when Content uses that description.
Author uses Authors when available and falls back to Author.
Each author uses a trimmed name, or an email when the name is empty.
Ignore nil authors and empty values.
Join distinct authors with a comma and space in source order.
Author and Subtitle do not enter the fallback key digest.
