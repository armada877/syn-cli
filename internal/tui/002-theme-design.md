# Theme design

Use ANSI defaults to follow the terminal palette without fixed RGB colors.
Keep the palette and styles on the model to prevent shared mutable state.
Use one rendered file to keep the overlay tied to the current Omarchy theme.
Inject its path and the color choice for isolated tests.

Check machine colors against emphasis and background references before using them.
Use reference colors only for readability checks.
A bad key must not remove valid values from other keys.

Watch directories because Omarchy can replace the complete theme directory.
Watch existing ancestors when the target directories are absent.
Rebuild watches after changes and after each loaded message.
Use a cancellable timer to wait for the final rendered file.
Keep watching outside Update so the reader remains responsive.

Sanitize source text before rendering styles.
Use ANSI width functions after rendering to retain styles during clipping.
