# Article metadata design

Store author and subtitle text in the existing article row.
Add both columns with empty defaults in one version 1 migration transaction.
Update the schema version only after both changes succeed.
This preserves existing rows and permits rollback after an incomplete migration.

Keep author and subtitle outside the article key algorithm.
A metadata change then updates the same row and preserves its read state.
Normal refreshes replace both values. HTTP 304 leaves article values unchanged.

Track required metadata refreshes with an explicit feed flag.
Migration sets this flag while preserving validators.
The app omits validators from the request copy until a full refresh succeeds.
The store clears the flag in the article update transaction.
This permits retry after HTTP 304 or failures without treating legitimately empty metadata as incomplete.
