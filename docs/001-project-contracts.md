# Project contracts

## Decision

Use the module path github.com/armada877/syn-cli and the executable name syn.
Select Go 1.27.2 through mise.toml for consistent builds.
Keep shared types independent of external libraries.
Define fetcher and repository interfaces in internal/domain.
Use one URL helper to keep URL identity consistent across packages.
Inject command output streams to support tests without process changes.

Pin gofeed v1.5.0 and modernc.org/sqlite v1.39.1.
Use charm.land/bubbletea/v2 v2.0.10 for the implemented terminal reader.
Use charm.land/lipgloss/v2 v2.0.6 for semantic terminal styles.
Use github.com/charmbracelet/x/ansi v0.11.8 for text width, clipping, and wrapping.
Use github.com/fsnotify/fsnotify v1.10.1 for live theme file watching.
Use golang.org/x/net v0.58.0 directly for HTML conversion and through gofeed for feed parsing.
These versions provide the required major versions and APIs.
gofeed v1.5.0 includes parser fixes and uses current golang.org/x/net.
Use golang.org/x/sys v0.47.0 as a transitive dependency.
This version supports the library dependencies with Go 1.27.2.
The resolved graph selects golang.org/x/net v0.58.0 and golang.org/x/text v0.41.0.
The feed parser, SQLite driver, and terminal reader build offline with Go 1.27.2.
Keep only imported libraries and their dependencies in the module files.
Keep module checksums in go.sum for offline checks after dependency installation.

## Sources

- [Bubble Tea](https://pkg.go.dev/charm.land/bubbletea/v2@v2.0.10)
- [gofeed](https://pkg.go.dev/github.com/mmcdole/gofeed@v1.5.0)
- [SQLite](https://pkg.go.dev/modernc.org/sqlite@v1.39.1)

## Result

Adapters own external library types.
Shared interfaces keep application code independent of HTTP, SQL, and terminal details.
