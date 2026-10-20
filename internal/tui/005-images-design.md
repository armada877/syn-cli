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
