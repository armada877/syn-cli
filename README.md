# syn

syn is an RSS reader project. The current executable provides help.

## Setup

Install mise. Run these commands in this directory.

```sh
mise install
mise exec -- go mod download all
```

mise.toml selects Go 1.27.2 for this project.
The module path is github.com/armada877/syn-cli.
Dependency installation needs a network connection.

## Build and test

```sh
mise exec -- go build ./cmd/syn
./syn --help
mise exec -- go vet ./...
mise exec -- go test ./...
```

After dependency installation, run checks without network access.

```sh
GOPROXY=off GOSUMDB=off mise exec -- go vet ./...
GOPROXY=off GOSUMDB=off mise exec -- go test ./...
GOPROXY=off GOSUMDB=off mise exec -- go build ./cmd/syn
```

## Packages

- cmd/syn starts the executable.
- internal/domain defines shared types, interfaces, errors, and URL normalization.
- internal/feed will fetch and parse feeds with gofeed.
- internal/store will store feeds and articles with SQLite.
- internal/app will connect the fetcher and repository.
- internal/cli handles arguments and output streams.
- internal/tui will provide the reader with Bubble Tea v2.

Each package has a SPEC.md file with its behavior and dependencies.
