# Project contracts

## Decision

Use the local module path syn-cli and the executable name syn.
Select Go 1.27.2 through mise.toml for consistent builds.
Keep shared types independent of external libraries.
Define fetcher and repository interfaces in internal/domain.
Use one URL helper to keep URL identity consistent across packages.
Inject command output streams to support tests without process changes.

Pin Bubble Tea v2.0.10, gofeed v1.5.0, and modernc.org/sqlite v1.39.1.
These versions provide the required major versions and APIs.
gofeed v1.5.0 includes parser fixes and uses current golang.org/x/net.
Pin golang.org/x/sys v0.47.0 and golang.org/x/sync v0.22.0 as transitive dependencies.
These versions support the library dependencies and keep the module graph compatible with Go 1.27.2.
The resolved graph selects golang.org/x/net v0.58.0 and golang.org/x/text v0.41.0.
All three libraries build offline with Go 1.27.2.
Keep module checksums in go.sum for offline checks after dependency installation.

## Sources

- [Bubble Tea](https://pkg.go.dev/charm.land/bubbletea/v2@v2.0.10)
- [gofeed](https://pkg.go.dev/github.com/mmcdole/gofeed@v1.5.0)
- [SQLite](https://pkg.go.dev/modernc.org/sqlite@v1.39.1)

## Result

Adapters own external library types.
Shared interfaces keep application code independent of HTTP, SQL, and terminal details.
