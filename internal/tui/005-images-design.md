# Article image design

Use one block layout for image spans and text lines.
This prevents duplicate alt text from changing request visibility.
Move the first image to the headline area before building the remaining body.

Use the existing Ultraviolet graphics response event through Bubble Tea.
This keeps terminal input under one consumer.
Use bounded query time and explicit identifiers instead of terminal environment guesses.

Keep request tokens separate from article generations.
An image can leave and return to one viewport before an old request completes.
Use tokens to reject that old result.
Keep failed sources separate from active entries so failures cannot fill all eight slots.

Queue protocol writes through one serialized sequence.
Return a completion message after each raw write.
Publish placeholder cells after that completion message.
Keep resource deletion behind active uploads and remove obsolete queued uploads.
This prevents late writes from restoring released resources.

Retain reduced pixels only while visible images load or render.
Release pixels when rendering completes.
Completed images do not consume active job slots.
Bound retained rendered rows by viewport height and two intersecting image edges.
Keep dimensions after pixel eviction so revisiting an image preserves its reserved height.
Use the verified disk cache to reload images without another network request.

Keep true-color output after Kitty confirms support.
Placeholder foreground colors contain image identifiers.
Color reduction would change those identifiers and display the wrong image.

Keep auto detection based on the matching Kitty reply.
Herdr answers that query through its embedded terminal, even when the outer terminal cannot display Kitty graphics.
Auto mode keeps Kitty support inside Herdr because the viewing terminal can support the protocol.
Do not use XTVERSION or inherited KITTY variables to select a mode.
A server can have clients with different terminal support.
For foot with Herdr, use `images = "blocks"` or `SYN_IMAGES=blocks` to avoid empty image areas.
Explicit modes let each launcher select rendering without capability queries.
Request cell measurements for forced graphics modes to preserve image proportions.
Force true-color output for blocks and Kitty placeholder identifiers.
Keep NO_COLOR above all mode choices to prevent downloads and terminal queries.
