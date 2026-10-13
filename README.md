# syn

syn stores RSS and Atom feeds in a local SQLite database.
It provides CLI commands and a Bubble Tea reader.
Refresh runs only when requested.

## Setup and checks

Install mise. Run these commands in this directory.

```sh
mise install
mise exec -- go mod download all
mise exec -- go build ./cmd/syn
./syn --help
```

mise.toml selects Go 1.27.2.
Dependency installation needs network access.
After installation, run checks offline.

```sh
GOPROXY=off GOSUMDB=off mise exec -- go vet ./...
GOPROXY=off GOSUMDB=off mise exec -- go test ./...
GOPROXY=off GOSUMDB=off mise exec -- go build ./cmd/syn
```

## Commands

```sh
syn
syn tui
syn --help
syn feed add https://example.com/feed.xml
syn feed list
syn feed remove 1
syn refresh
syn refresh 1
syn article list --feed 1 --unread
syn article show 1
syn article read 1
syn article unread 1
syn --db /tmp/syn-smoke.db feed list
```

The default database is $XDG_DATA_HOME/syn/articles.db.
When XDG_DATA_HOME is empty, syn uses ~/.local/share/syn/articles.db.
Put --db PATH before the command to override this path.
Adding a feed does not fetch it.
Showing an article does not mark it as read.
Lists use tab-separated columns. Errors go to stderr.
Success returns zero. Operation failures return one. Invalid arguments return two.
Refresh reports each feed and preserves successful updates after another feed fails.

## Reader keys

| Key | Action |
| --- | --- |
| Arrows or j/k | Move selection or scroll the reader |
| Enter | Open an article and mark it as read |
| Escape | Close the reader, picker, help, or input |
| m | Toggle article read state |
| u | Toggle the unread filter |
| R | Refresh all feeds |
| ? | Show key help |
| q or Ctrl+C | Quit outside URL input |
| PageUp or PageDown | Scroll the reader by one page |

At 80 columns or more, feeds and articles appear side by side.
Tab changes focus. a adds a feed. d requests removal. r refreshes the selected feed.
Below 80 columns, articles appear in one stream. Tab has no effect.
f opens the feed picker.
In the picker, arrows or j/k move selection. Enter applies the feed filter.
All feeds removes the feed filter. Escape cancels without changing the filter.
The picker also accepts a, d, and r for feed management.
URL input accepts typing and paste. Enter saves it. Escape cancels it.
y or Enter confirms removal. n or Escape cancels removal.
Ctrl+C also quits from URL input.
Resize preserves filters, article selection, and the open reader.

## Local smoke procedure

Serve an RSS or Atom fixture on a local port.
Use a separate database for this check.

```sh
./syn --db /tmp/syn-smoke.db feed add http://127.0.0.1:8000/feed.xml
./syn --db /tmp/syn-smoke.db refresh
./syn --db /tmp/syn-smoke.db feed list
./syn --db /tmp/syn-smoke.db article list
./syn --db /tmp/syn-smoke.db article show 1
./syn --db /tmp/syn-smoke.db article read 1
./syn --db /tmp/syn-smoke.db article unread 1
./syn --db /tmp/syn-smoke.db tui
```

Disconnect the fixture server. Open a stored article in the reader.
Quit and restart. Check that the article remains read.
Check navigation, refresh feedback, URL input, and removal confirmation.
Reduce the terminal width to 79 columns. Check the feed picker and Tab.
Open an article. Resize through 79, 80, and 81 columns in both directions.
Check that the article and filters remain selected.
Cancel removal once. Confirm removal once. Quit with q.

Each package has a SPEC.md file with its required behavior.
Decision records describe the app, CLI, and reader designs next to their code.
