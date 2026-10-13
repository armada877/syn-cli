# Template installation design

Embed the template beside the installer so the binary carries the exact template.
Create the target with exclusive creation to preserve existing files.
Inspect existing targets without following symlinks before comparing regular file contents.
Inject the target path to isolate tests from user configuration.
