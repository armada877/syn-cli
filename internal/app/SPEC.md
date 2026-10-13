# App spec

This package connects domain.Fetcher and domain.Repository.
It depends on internal/domain and the Go standard library.
Network and database operations take context.Context first.
Pass cancellation to each operation.
Pass the current fetch time to Repository.ApplyFetch.
Keep HTTP, SQL, and terminal details outside this package.
Application operations are not implemented yet.
