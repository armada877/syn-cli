# Image design

Keep HTTP, validation, and cache work outside the reader update function.
Check dimensions before full decoding to limit memory use.
Use the standard GIF decoder to avoid allocating animation frames.
Use the cached x/image module for WebP and image scaling.

Hash complete URLs instead of using remote filenames.
Verify cache contents on every load so corrupt files cannot bypass limits.
Use atomic replacement so interrupted writes do not expose partial images.

Use the existing Kitty encoder and its official coordinate table.
Keep protocol commands outside view text.
Use explicit coordinates on every placeholder cell so clipping preserves offsets.
Use black behind transparent block pixels because terminal background colors can be unknown.
