# Configuration design

Use TOML for the two image settings.
Keep configuration separate from theme colors and saved reader state.
Inject paths or parsed values to prevent access to personal settings during tests.
Return fixed status messages so invalid input cannot inject terminal controls.
Read the file once to keep updates independent of file access.

Apply SYN_IMAGES after file validation so launchers can select a mode for each terminal.
Pass the environment value from the production constructor to keep tests independent of personal settings.
Keep invalid overrides separate from file errors and retain the configured mode.
