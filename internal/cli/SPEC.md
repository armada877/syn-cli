# CLI spec

Run receives arguments, stdout, and stderr.
RunWith also receives storage, fetcher, and terminal dependencies.
Use the standard flag package for options and explicit dispatch for commands.
No arguments and tui start the terminal reader from stored data.
-h and --help print usage without opening storage.
--db PATH precedes the command and overrides the default database path.
The default path is $XDG_DATA_HOME/syn/articles.db.
When XDG_DATA_HOME is empty, use ~/.local/share/syn/articles.db.
Validate command arguments before opening storage.

feed add URL stores a subscription without fetching it and prints its ID and URL.
feed list prints ID, title, and URL.
feed remove ID removes the feed and its articles.
refresh [ID] refreshes one feed or all feeds and reports each result.
Successful refresh output contains ID, status, and article count.
Failed refresh output contains ID and error on stderr.
article list accepts --feed ID and --unread.
It prints ID, feed ID, read state, title, and URL.
article show ID prints stored plain text without changing read state.
article read ID and article unread ID save the requested read state.

List columns use tabs.
Column values replace whitespace with spaces and remove terminal controls.
Normal output goes to stdout. Errors go to stderr.
Success returns zero. Operation and output failures return one.
Invalid command arguments return two.
A refresh returns one when any feed fails.
Interrupts cancel operations. Each run closes storage before returning.

setup omarchy installs the render template before opening storage.
It accepts exactly two command words.
Dependencies can supply the template target path for isolated checks.
Normal setup uses ~/.config/omarchy/themed/syn.toml.tpl.
Success prints the installed path and explains rendering on the next theme change.
Setup keeps the normal argument, operation, and output failure exit codes.

article show includes an optional plain subtitle and byline after its title and URL.
It suppresses a subtitle that duplicates the displayed body.
It keeps body paragraph breaks and stored read state.
