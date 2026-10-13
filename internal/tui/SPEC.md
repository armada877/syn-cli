# TUI spec

The reader uses charm.land/bubbletea/v2 v2.0.10.
New receives a context and an app.
Init loads stored feeds and articles without a network request.
Storage and network work runs in commands that return typed messages.
Update changes view state and schedules commands.
Only one operation runs at a time.
Run cancels pending operations when the program exits.
The CLI closes storage after Run returns.

WideLayoutMinColumns is 80.
At 80 columns or more, show feed and article panes side by side.
Below 80 columns, show one article stream.
Each article row shows its date, feed title, title, and read state.
The repository supplies newest-first article order.
Show the active feed title and unread filter.
Show empty lists, loading state, operation results, and errors.
Keep article selection valid after data and filter changes.

Arrow keys and j/k move selection.
Wide Tab changes focus. Narrow Tab has no effect.
Enter opens an article in a full-screen reader and saves read state.
Escape closes the reader.
Arrows, j/k, PageUp, and PageDown scroll the reader.
m toggles read state. u toggles the unread filter.
R refreshes all feeds. ? opens key help.
q and Ctrl+C quit outside URL input. Ctrl+C also quits from input.
In wide mode, a opens URL input and d opens removal confirmation.
r refreshes the active feed or the highlighted feed when feed focus is active.
Enter adds the URL. Escape cancels input.
The input accepts typed and pasted text.
Removal requires y or Enter. n or Escape cancels removal.

Narrow f opens the feed picker.
Arrow keys and j/k move feed selection.
Enter applies the highlighted filter and closes the picker.
All feeds selects every feed.
a opens URL input. d requests confirmed removal. r refreshes the highlighted feed.
Escape closes the picker without changing the active filter.

Resize changes the layout immediately at the breakpoint.
It preserves the active filter, selected article ID, unread filter, and open reader.
Resize does not load data or start network work.
Reader text wraps again at the new width.
Scrolling stays within valid bounds.
Displayed source text contains no terminal control sequences.

Theme state and styles belong to each model.
NewWithTheme receives the rendered file path and the NO_COLOR choice.
New resolves ~/.local/state/omarchy/current/theme/syn.toml for normal use.
A nonempty NO_COLOR disables generated styles.
Normal text inherits terminal colors.
The default accent is ANSI bright blue. Errors use ANSI red.
Read articles use faint text unless a muted overlay supplies a color.
Selected rows use bold accent text and take priority over read styles.
Unread markers use the accent. Active focus indicators use bold accent text.
Emphasis prefers bright_foreground over foreground.
The application never paints a full-screen background.

The loader accepts quoted three-digit and six-digit RGB hex colors.
It accepts single quotes, double quotes, and trailing comments.
It ignores unknown keys, invalid values, and unresolved placeholders.
Each absent or invalid key keeps its own default.
Missing, removed, and unreadable files restore ANSI defaults.
The loader reads no other theme file.
It recognizes mode, accent, selection, muted, foreground, error, bright_foreground, background, blue, and bright_blue.
Accent must differ from emphasis by at least 64 sRGB units when emphasis exists.
With a background, accent contrast must reach five or match the reference blue contrast.
The reference uses bright_blue, then blue, then ANSI bright blue.
A selection background requires accent contrast of at least 4.5.

Watching runs in commands and stops when the context ends.
The watcher observes the theme directory and its parents.
When directories are absent, it observes existing ancestors.
Changes settle for 250 milliseconds before loading the file.
Update receives a typed palette message and rebuilds styles.
Theme updates preserve filters, selection, focus, picker state, reader state, and scroll.
Theme updates do not access storage or feeds.
Source text is sanitized before styling.
ANSI-aware clipping preserves generated styles and terminal width limits.
