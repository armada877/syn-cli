# Contributing

Follow the [security policy](SECURITY.md) to report vulnerabilities privately.

Install mise. Run these commands from the repository root.

```sh
mise install
mise exec -- go mod download all
mise exec -- gofmt -w .
mise exec -- go build ./cmd/syn
mise exec -- go vet ./...
mise exec -- go test ./...
CGO_ENABLED=1 mise exec -- go test -race ./...
```

mise selects Go 1.27.2. Race tests need a C compiler and CGO.
After dependencies are installed, set GOPROXY=off and GOSUMDB=off for offline checks.
Check formatting before you submit a change.

```sh
mise exec -- sh -c 'files=$(gofmt -l .); test -z "$files" || { printf "%s\n" "$files"; exit 1; }'
```

Keep changes small. Include tests for changed behavior.
Write a SPEC.md file next to the code to define required behavior.
Write decision records next to the code to explain design choices.
Do not write code comments. Tool directives, such as shebangs and lint directives, are permitted.
Use Simplified Technical English (ASD-STE100) for documentation and commit messages.
Use short sentences, common words, and the active voice.
