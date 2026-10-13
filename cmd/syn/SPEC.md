# Executable spec

The executable is syn.
main passes command arguments and process output streams to cli.Run.
main uses the returned value as the process exit code.
The entry point contains no network or database operations.
