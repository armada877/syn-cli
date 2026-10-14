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
At 81 columns or more, show a headline index and article preview side by side.
At 80 columns or less, show one article stream.
Each narrow article row shows its date, feed title, title, and read state.
The repository supplies newest-first article order.
Show the active feed title and unread filter.
Show empty lists, loading state, operation results, and errors.
Keep article selection valid after data and filter changes.

Arrow keys and j/k move selection.
Wide Tab returns preview focus to the index. Index Tab and narrow Tab have no effect.
Enter saves read state and focuses the wide preview or opens the narrow reader.
Escape returns to the index or closes the narrow reader.
Arrows, j/k, PageUp, and PageDown scroll the reader.
m toggles read state. u toggles the unread filter.
R refreshes all feeds. ? opens key help.
q and Ctrl+C quit outside URL input. Ctrl+C also quits from input.
In the picker, a opens URL input and d opens removal confirmation.
r refreshes the highlighted feed.
Enter adds the URL. Escape cancels input.
The input accepts typed and pasted text.
Removal requires y or Enter. n or Escape cancels removal.

f opens the feed picker from both main layouts and the focused wide preview.
f has no effect inside the narrow full-screen reader.
Escape restores the previous reading focus and scroll.
Enter applies the feed filter and returns to index or stream focus.
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
NewWithOptions receives explicit theme and state paths, a clock, and the NO_COLOR choice.
An empty state path disables persistence.
NewWithTheme remains compatible and disables persistence.
Tests inject temporary paths or disable persistence.
Tests inject a fixed clock for date checks.
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

The narrow open reader uses a centered body column with a preferred width of 60 cells.
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
The wide index defaults to 30 percent of terminal width.
The preview uses the remaining width after reserving one divider cell.
Each pane clips and pads its own rows before adding the divider.
Scrolling keeps the selected row visible whenever the frame has a list row.
Reader operation messages appear above the final key bar, outside the body.
Reader rules span the entire terminal width.
Source text remains safe after padding and clipping. Generated styles remain intact.

## Wide front page

Start with index focus.
Selection updates the preview without storage writes.
Selection changes reset preview scroll.
Keep a stable article copy after Enter, including when unread filtering removes its index entry.
m toggles the focused article or selected index article.
Arrows and j/k select index entries or scroll the focused preview.
PageUp and PageDown scroll the focused preview.

Use full-width double rules around the active feed title and today's local date.
Use ALL FEEDS when no feed filter applies.
Use capitals for the safe feed title and 2 January 2006 for the date.
Clip the title to reserve date space.
Show focus, unread filtering, and operation messages below the masthead.
Compress the masthead at short heights. Always reserve the final key bar.

Each index entry shows selection, read state, up to two headline rows, and publication date.
Wrap safe headlines at grapheme boundaries and omit extra rows.
Keep the selected entry visible. Pad unused panel rows without repeated entries.
Use accent styles for selection and unread markers. Use muted styles for read entries.

The preview shows the uppercase bold headline and a panel-width single rule.
Show optional subtitle and byline with the existing text helpers.
Center the justified body in a column of at most 60 cells with safe margins.
Keep final paragraph lines left aligned and blank rows between paragraphs.
Do not repeat the masthead in the preview.
Show No article matches when the filtered index is empty.

Reserve one cell for the vertical divider across every panel row.
Keep the index at least 20 columns and the preview at least 40 columns.
Clamp effective index width to [20, terminal width minus 41].
< and > change index width by one column on the wide main page.
Enable cell motion mouse events on that page.
Start dragging on a left click on the divider within panel rows.
Resize on left-button motion during dragging.
Stop dragging and save on left release, including release outside the divider.
Ignore unrelated mouse events and disable resizing in all modal views.
Cancel dragging on resize or when opening a modal view.
Reflow the preview and clamp scroll after width changes.

Load index_width from $XDG_STATE_HOME/syn/front-page.json when XDG_STATE_HOME is nonempty.
Otherwise use ~/.local/state/syn/front-page.json.
Accept one positive integer field. Use the default width for missing, invalid, or unreadable files.
Create the state directory only on save. Replace the state file atomically.
Save absolute preferred width after keyboard changes and completed drags.
Window resize clamps effective width without changing the preferred width.
A larger window restores the preferred width.
Serialize saves and keep only the latest pending width.
Run saves separately from feed and read operations. Keep navigation available.
Report save errors in the operation area.

Resize at 80 and 81 preserves filters, selection, preferred width, and focused article.
Wide preview focus becomes the narrow reader. The narrow reader becomes wide preview focus.
An unfocused wide preview becomes the stream without storage writes.
Theme changes preserve all navigation and reading state.
