# Front page design

Use the wide main page for a headline index and an article preview.
Keep the narrow stream and reader to preserve useful controls on small terminals.
Use the same stable article copy for focused reading in either layout.
Selection alone performs no writes. Enter remains the explicit read action.

Keep the feed picker separate from article focus.
Allow picker access from both main layouts and focused wide reading.
Keep f inactive in the narrow reader to preserve its controls.
Opening it preserves the focused article and scroll until Enter applies a filter.
This lets users manage feeds without a permanent feed pane.

Store the preferred index width separately from the effective panel width.
Window size can limit the effective width without losing the saved preference.
Inject theme, state, and clock inputs so tests cannot access personal state.
Keep the compatibility theme constructor free of state persistence.

Use one pending width and typed save results to serialize atomic file replacements.
Disable storage actions while a save runs, and defer saves while storage operations run.
Navigation stays available after file errors.

Build both panels inside the existing exact terminal frame.
Use existing text helpers and theme styles for safe text and live colors.
Account for wrapped index entries when placing the selected entry in the viewport.
Keep the masthead outside the preview scroll range and the key bar on the final row.
