# Reader design

Use one model for both layouts.
Keep filters and selected article IDs independent of terminal width.
Resize only changes dimensions and reader scroll bounds.
This prevents layout changes from resetting reading state or loading data.

Load all stored articles and apply view filters in the model.
This lets filter changes use stored data without extra storage requests.
Use stable IDs to preserve selection when refresh changes article order.
Keep a copy of the open article so an unread filter cannot close the reader.

Use a feed picker at 80 columns or less to keep article rows readable.
Keep picker selection separate from the active feed filter until Enter applies it.
Use typed command results to return storage and network data to Update.
Disable operation keys while a command runs to keep writes sequential.
Use ANSI width helpers for Unicode-aware clipping and reader wrapping.

Use the wide layout from 81 columns.
Build each view inside one exact terminal frame.
Reserve the final row for its key bar before adding content.
Clip and pad rows with ANSI display widths to preserve generated styles.

Calculate list height from the actual header rows and key bar.
Put operation messages in the list header so both panes share the same height.
At short heights, combine the message with the main header to keep selection visible.
Fill short lists with blank rows and keep the divider through both wide panes.
