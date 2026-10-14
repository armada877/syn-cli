# Local store

`Open(path string) (*Store, error)` opens a SQLite database at the supplied path.
`Store` implements `domain.Repository`.
The caller selects the path and closes the store.

## Initialization

The store creates missing parent directories with mode 0700.
It creates a new database file with mode 0600 on Unix.
It preserves permissions on existing directories and files.
The store uses one connection with foreign keys enabled and a 5000 millisecond busy timeout.
Each new connection receives these settings through the driver connection parameters.

Schema version 2 uses the SQLite `user_version` field.
Initialization creates the schema and sets its version in one transaction.
The store rejects a newer version and preserves existing data.
A failed initialization rolls back all schema changes.

## Schema

Feeds have INTEGER primary keys and unique normalized URLs.
Feeds store title, ETag, Last-Modified, and the last fetch time.
Articles have INTEGER primary keys and reference a feed with cascading deletion.
The feed ID and article key form a unique pair.
Articles store title, URL, content, author, subtitle, publication time, and read state.
Read state defaults to zero and permits only zero or one.

## Operations

The store normalizes feed URLs with `domain.NormalizeFeedURL` before insertion.
Adding a duplicate URL returns the existing feed.
Feeds appear in ascending ID order.
Removing a feed removes its articles.
Unknown IDs return `domain.ErrNotFound` for removal, fetch application, and read state changes.
All queries use parameters for user values.

Fetch application updates feed metadata and articles in one transaction.
A failed fetch transaction preserves all existing data.
Article updates preserve IDs and read state.
Repeated article keys update the same article within their feed.
A missing publication time uses the fetch time on first insertion.
A later update without a publication time preserves the stored time.
A supplied publication time replaces the stored time.
A NotModified result updates validators and fetch time only.
It preserves the feed title and all articles.

Article filters select a feed, unread articles, or both.
A zero feed ID selects all feeds.
Articles appear by descending publication time, then descending ID.
The store compares parsed times to order fractional timestamps correctly.

## Timestamps

The database stores timestamps as UTC RFC3339Nano text.
An empty last fetch field represents a feed that has no fetch result.
Publication times always contain a timestamp, including a zero fetch time when supplied.

Schema version 2 adds author and subtitle as non-null text with empty defaults.
A version 1 migration adds article metadata and feed pending state, then updates user_version in one transaction.
A migration failure restores the original schema and version.
Migration preserves all rows, IDs, keys, timestamps, validators, and read states.
Existing rows receive empty metadata.
A normal refresh updates both fields while preserving IDs, read state, and the existing timestamp rules.
A NotModified result preserves both fields.
Reopening version 2 does not repeat migration.

Feeds include metadata_pending, an integer with a zero default and values limited to zero or one.
The version 1 migration adds this column and marks existing feeds pending in the same migration transaction.
Migration preserves stored ETag and Last-Modified values.
Fresh feeds start without pending metadata.
A successful full fetch clears pending state in the same transaction as article updates.
HTTP 304 and failed transactions keep the previous pending state.
Empty author or subtitle values do not set pending state.
