# Configuration spec

Load configuration once at reader startup.
Read $XDG_CONFIG_HOME/syn/config.toml when XDG_CONFIG_HOME has a value.
Otherwise read ~/.config/syn/config.toml.
Never create or change this file.

The images setting accepts auto and off.
The image_fallback setting accepts blocks and placeholder.
The defaults are auto and blocks.
Ignore unknown keys.
Use defaults when the file is missing.
Keep defaults for invalid values and show a safe status message.
Use all defaults when TOML syntax is invalid.
Limit configuration reads to 64 KiB.
Accept explicit configuration or an explicit path for tests.
An empty injected path disables file access.
