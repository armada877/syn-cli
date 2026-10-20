# Configuration spec

Load configuration once at reader startup.
Read $XDG_CONFIG_HOME/syn/config.toml when XDG_CONFIG_HOME has a value.
Otherwise read ~/.config/syn/config.toml.
Never create or change this file.

The images setting accepts auto, kitty, blocks, placeholder, and off.
The image_fallback setting accepts blocks and placeholder.
The defaults are auto and blocks.
Ignore unknown keys.
Use defaults when the file is missing.
Keep defaults for invalid values and show a safe status message.
Use all defaults when TOML syntax is invalid.
Limit configuration reads to 64 KiB.
Accept explicit configuration or an explicit path for tests.
An empty injected path disables file access.

Read SYN_IMAGES once at the production boundary.
A valid nonempty value overrides images after configuration loading.
An unset or empty value keeps the configured mode.
An invalid value keeps the configured mode and shows a fixed warning.
Never include the invalid value in terminal output.
Do not save environment overrides to configuration files.
Inject the override value explicitly for tests.
Injected models never read SYN_IMAGES from the process environment.
