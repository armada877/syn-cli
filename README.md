# syn

Syndicates your RSS and Atom feeds into a newspaper in your terminal.

It stores RSS and Atom feeds in a local SQLite database,
and provides CLI commands and a Bubble Tea reader.

## Installation

Install from source with Go 1.27.2 or later:

```sh
go install github.com/armada877/syn-cli/cmd/syn@latest
```

Add Go's binary directory to PATH. It is usually $(go env GOPATH)/bin.

When a release is available, download it from the
[releases page](https://github.com/armada877/syn-cli/releases).
Choose linux or darwin for your operating system.
Choose amd64 for Intel or AMD processors, or arm64 for ARM processors.
Download the matching tar.gz archive and checksums.txt into the same directory.
Verify the archive before extraction. Replace ARCHIVE with its file name.

```sh
sha256sum --ignore-missing -c checksums.txt
```

On macOS, use this command and check the line for your archive:

```sh
shasum -a 256 -c checksums.txt
```

Continue only if your archive reports OK.
Extract the archive and install syn into a directory on PATH:

```sh
tar -xzf ARCHIVE
mkdir -p "$HOME/.local/bin"
install -m 755 syn "$HOME/.local/bin/syn"
export PATH="$HOME/.local/bin:$PATH"
```

Add the PATH setting to your shell configuration to keep it after restart.

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

At 81 columns or more, a headline index and article preview appear side by side.
Moving selection updates the preview without changing read state.
Enter marks the article read and focuses its preview. Tab or Escape returns to the index.
< and > change index width. Drag the divider with the left mouse button.
The reader saves the preferred width in $XDG_STATE_HOME/syn/front-page.json.
When XDG_STATE_HOME is empty, it uses ~/.local/state/syn/front-page.json.
At 80 columns or less, articles appear in one stream. Tab has no effect.
f opens the feed picker from both main layouts and the focused wide preview.
f has no effect inside the narrow full-screen reader.
In the picker, arrows or j/k move selection. Enter applies the feed filter.
All feeds removes the feed filter. Escape cancels without changing the filter.
In the picker, a adds a feed. d requests removal. r refreshes the selected feed.
URL input accepts typing and paste. Enter saves it. Escape cancels it.
y or Enter confirms removal. n or Escape cancels removal.
Ctrl+C also quits from URL input.
Resize preserves filters, article selection, and the open reader.

## Mouse wheel

The vertical wheel moves three entries in the article stream or feed picker.
It scrolls article content by three lines in the narrow reader.
In the wide layout, the pointer selects the index or preview, independent of keyboard focus.
The index wheel changes article selection and updates the preview while preserving focus.
The preview wheel scrolls article content.
Movement stops at both ends. Wheel navigation preserves read state and performs no storage work.
The wide layout ignores wheel events over the masthead, divider, footer, and outside the frame.
Help, URL input, removal confirmation, and divider dragging ignore wheel events.
The reader, stream, and picker enable mouse events. The wide layout retains cell motion for divider dragging.

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
Reduce the terminal width to 80 columns. Check the feed picker and Tab.
Open an article. Resize through 79, 80, and 81 columns in both directions.
Check that the article and filters remain selected.
Cancel removal once. Confirm removal once. Quit with q.

Each package has a SPEC.md file with its required behavior.
Decision records describe the app, CLI, and reader designs next to their code.

## Omarchy colors

Run `syn setup omarchy` to install ~/.config/omarchy/themed/syn.toml.tpl.
Omarchy renders the template on the next theme change.
Setup preserves a different existing template and refuses a symlink target.
It does not open the database or run a theme change.

The reader uses terminal ANSI colors by default.
It reads ~/.local/state/omarchy/current/theme/syn.toml when that file exists.
It follows live changes without restarting or fetching feeds.
Missing or invalid keys retain their default styles.
Unread markers and selected rows use the accent.
Read articles use faint text or the rendered muted color.
Set a nonempty NO_COLOR to disable generated styles.

## Newspaper reader

The preview and open reader use a centered body column with a preferred width of 60 cells.
Full-width rules frame the feed, date, and headline.
The headline, optional subtitle, and author appear over the body column.
The body uses justified paragraphs. Short final lines stay left aligned.
Metadata scrolls with the article. Current themes supply the colors.
Descriptions that duplicate the body do not appear twice.
The CLI also shows the subtitle and author without changing read state.
Opening an existing version 1 database migrates it to version 2 in one transaction.
The migration preserves articles, subscriptions, IDs, and read state.

Migrated feeds request a full body on their next refresh to fill article metadata.
Stored validators remain intact. Later refreshes use conditional requests again after a successful full refresh.

Every terminal view fills the exact terminal width and height.
The key bar stays on the final row.
Lists use all rows between their header and key bar, with blank padding for short lists.
The reader centers its feed title and date across the full terminal width.
Operation messages stay above the key bar.

## License

syn uses the [MIT License](LICENSE).

## Article images

The reader displays images under the headline and between body paragraphs.
It downloads only images that intersect the preview or reader viewport.
Image work runs separately from navigation.
The CLI shows safe image placeholders and never downloads images.

In auto mode, the reader queries Kitty graphics support once at startup.
It uses Unicode placeholder cells when the terminal confirms support.
Other terminals use true-color half blocks.
Transparent block pixels use a black background.
The reader uses measured cell dimensions when available.
Otherwise it assumes cells are twice as high as they are wide.
Image height has a limit of 256 text rows.

Set image options in $XDG_CONFIG_HOME/syn/config.toml.
When XDG_CONFIG_HOME is empty, use ~/.config/syn/config.toml.
Missing settings use these defaults:

```toml
images = "auto"
image_fallback = "blocks"
```

The images setting accepts "auto", "kitty", "blocks", "placeholder", and "off".
Set images to "kitty" to force Kitty graphics without a capability query.
Set images to "blocks" to force true-color half blocks without Kitty queries or uploads.
Set images to "placeholder" or "off" to show plain placeholders without queries or downloads.
The kitty and blocks modes still request cell measurements.
The image_fallback setting applies only to auto mode.

Set SYN_IMAGES to any accepted images value to override the configuration.
For example, run `SYN_IMAGES=blocks syn` to force blocks.
An unset or empty SYN_IMAGES keeps the configuration value.
An invalid SYN_IMAGES keeps that value and shows a safe warning.

Herdr answers Kitty queries through its embedded terminal.
The outer terminal can lack Kitty support even when Herdr reports success.
Auto mode trusts that response, which can leave empty image areas in foot with Herdr.
For foot with Herdr, set `images = "blocks"` or launch with `SYN_IMAGES=blocks`.
Use `SYN_IMAGES=kitty` when the viewing terminal supports Kitty graphics.
Set image_fallback to "placeholder" to prevent downloads when Kitty support is absent.
A nonempty NO_COLOR also selects plain placeholders without queries or downloads.
Invalid settings show a status message and retain defaults.
The reader never creates or changes your configuration file.

Verified images use $XDG_CACHE_HOME/syn/images, or ~/.cache/syn/images when XDG_CACHE_HOME is empty.
Stored images remain available without network access.
The reader accepts PNG, JPEG, GIF first frames, and WebP.
Each image has a ten-second timeout and a 10 MiB response limit.
