# Image service spec

Accept absolute HTTP and HTTPS image URLs without credentials.
Apply the same rules to redirects.
Permit at most four redirects.
Use a ten-second request timeout and caller cancellation.
Run at most two downloads at the same time.

Read at most 10 MiB for each response or cached file.
Require PNG, JPEG, GIF, or WebP content types for responses.
Require the decoded format to match the content type.
Check dimensions before decoding pixels.
Reject zero dimensions, dimensions above 8192, and images above 16 million pixels.
Decode only the first GIF frame.

Cache verified encoded data under a SHA-256 URL key.
Use temporary files and atomic replacement.
Validate cached files with the same byte and dimension limits.
Treat cache failures as nonfatal.
Accept injected clients, cache paths, and stricter limits.
An empty cache path disables disk access.

Generate terminal output only from decoded pixels and numeric identifiers.
Use PNG uploads with protocol chunks of at most 4096 encoded bytes.
Use quiet virtual placements and Unicode placeholder cells.
Encode each cell with explicit row, column, image ID, and placement ID.
Keep row and column coordinates when clipping text cells.
Delete only images with identifiers owned by this reader.

Use true-color upper half blocks when graphics support is absent.
Composite transparent pixels against black for block rendering.
Use a cell width-to-height ratio of 0.5 when measurements are absent.
Limit generated image height to 256 rows.
