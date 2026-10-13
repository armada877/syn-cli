# CLI spec

This package parses command arguments and writes command output.
It depends on the Go standard library.
Later commands can depend on internal/app and internal/domain.
Run accepts arguments, stdout, and stderr through parameters.
No arguments, -h, and --help write help to stdout and return zero.
Invalid arguments write an error to stderr and return two.
Output failures return one.
Later feed commands use domain.NormalizeFeedURL.
Complete feed commands are not implemented yet.
