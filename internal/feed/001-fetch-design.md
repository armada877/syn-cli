# Fetch design

Fetch uses a context deadline to apply the default timeout.
This preserves the injected client and its transport settings.
The body reader accepts one extra byte to detect responses above the limit.
Parsing starts after the size check.

Fallback keys hash a JSON array with three fields.
The fields contain title, UTC publication date, and selected content.
JSON preserves field boundaries when source strings contain separator characters.
The key uses lowercase hexadecimal text.
The first article wins when keys repeat.
