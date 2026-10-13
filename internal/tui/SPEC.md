# TUI spec

This package provides the terminal reader with charm.land/bubbletea/v2.
It depends on internal/app and internal/domain.
Keep Bubble Tea types inside this package.
Use application operations for network and database work.
Pass context.Context to those operations.
Keep feed and article state in domain types.
The reader interface is not implemented yet.
