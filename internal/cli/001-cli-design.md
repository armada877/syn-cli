# CLI design

Select the database path in the CLI because storage accepts a caller-supplied path.
Use the XDG data directory for persistent articles.
Keep --db available for separate databases and local checks.

Parse and validate commands before opening storage.
This prevents help and invalid commands from creating database files.
Inject dependencies through RunWith for checks without sockets or a terminal.
Use a local HTTP transport fixture to check the full command sequence offline.
Close each database so repeated command tests also check stored state after restart.

Dispatch setup before resolving the database path or opening storage.
This keeps template installation independent of stored articles.
Inject the template target through the existing dependency structure.
