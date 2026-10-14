# Newspaper design

Use one body column with a preferred width of 60 display cells.
This keeps long paragraphs readable on wide terminals.
Reduce the column width and margins on narrow terminals.

Build all article lines from safe text before applying styles.
Use cached ANSI functions to measure and split graphemes.
This keeps combining characters intact and avoids new dependencies.
Omit glyphs that cannot fit into an extremely narrow column.

Keep metadata in the same line sequence as the body.
Use one viewport height for rendering, page movement, and scroll limits.
This keeps the masthead in the scroll range and the footer outside it.
Use existing model styles so live theme changes redraw the full article.

Keep the shared display helpers in the app package.
The CLI and terminal reader then suppress duplicate descriptions in the same way.
Store the raw description so later display changes do not lose source data.

Span newspaper rules across the terminal width.
Center the feed title and date across the terminal.
Keep the headline, subtitle, and byline over the body column.
This separates the masthead from the narrower reading area.

Reserve the last row for the reader key bar.
Place operation messages above it and outside the body.
Calculate the reader viewport after reserving these rows.
Fill short articles with blank rows to keep the bar in a fixed position.
