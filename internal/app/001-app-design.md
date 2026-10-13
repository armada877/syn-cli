# App design

Use the domain interfaces for storage and network operations.
This keeps the app independent of SQLite and HTTP implementations.
Return one refresh result for each feed.
This lets both interfaces show failures while preserving successful updates.
Use sequential refresh to limit concurrent network and storage work.

Keep text conversion in the app package.
Both interfaces need the same readable article text.
Parse HTML with golang.org/x/net/html instead of removing tags with string rules.
Remove terminal control sequences before either interface displays source text.
