# TUI spec

The reader uses charm.land/bubbletea/v2 v2.0.10.
New receives a context and an app.
Init loads stored feeds and articles without a network request.
Storage and network work runs in commands that return typed messages.
Update changes view state and schedules commands.
Only one operation runs at a time.
Run cancels pending operations when the program exits.
The CLI closes storage after Run returns.

WideLayoutMinColumns is 81.
At 81 columns or more, show feed and article panes side by side.
At 80 columns or less, show one article stream.
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

The open reader uses a centered body column with a preferred width of 60 cells.
The width is max(1, min(60, terminal width minus 4)).
The left margin is half the remaining terminal width, rounded down.
The newspaper starts with a full-width double rule, uppercase feed title, optional date, and another full-width double rule.
The date uses the format 2 January 2006.
The article FeedID selects the feed title, independent of the active filter.
The uppercase bold headline follows, then a full-width single rule, optional subtitle, and optional byline.
A blank line separates metadata from body paragraphs.
The feed title and date wrap and center across the terminal.
The headline, subtitle, and byline wrap and center over the body column.
Rules and masthead use accent styles.
The headline uses emphasis. Subtitle and byline use muted styles.
The body inherits the terminal foreground.
The list status header does not appear above the open article.
The entire column scrolls. The help footer and operation messages stay outside the column.
The reader viewport uses the terminal height minus the key bar and any operation row, with a zero minimum.
Page movement uses at least one row. Rendering and scroll limits use the same viewport height.
Body lines wrap greedily at word gaps. Long words split at grapheme boundaries.
Glyphs wider than the column are omitted.
Nonfinal paragraph lines distribute extra spaces evenly, with remaining spaces placed from left to right.
Final lines and single-word lines stay left aligned.
One blank line separates paragraphs.
Source text is safe before layout. Styles follow layout.
Resize reflows the open article and clamps scroll. Theme changes preserve reader state.

Every view contains exactly the terminal height in rows.
Each row clips and pads to the exact terminal width, measured in display cells.
This includes lists, picker, reader, help, URL input, and removal confirmation.
The key bar always occupies the final row, including terminals with only one row.
Blank rows fill unused space without repeated entries.
List operation messages form part of the header.
At heights below four rows, an operation message shares the main header.
All remaining rows between the header and key bar belong to the list.
Both wide panes fill that height, including a divider on blank rows.
The feed pane uses one third of the terminal width after reserving one cell for the divider.
The article pane uses the remaining width.
Each pane clips and pads its own rows before adding the divider.
Scrolling keeps the selected row visible whenever the frame has a list row.
Reader operation messages appear above the final key bar, outside the body.
Reader rules span the entire terminal width.
Source text remains safe after padding and clipping. Generated styles remain intact.
