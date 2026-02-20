# Contributing to Rootline

Thank you for your interest in contributing to Rootline.

## Reporting Bugs

Open an issue using the **Bug Report** template. Include:

- Steps to reproduce the problem
- Expected vs actual behavior
- Rootline version (`rootline --version`)
- OS and Go version

## Suggesting Features

Open an issue using the **Feature Request** template. Describe the use case and why existing functionality doesn't cover it.

## Development Setup

```bash
# Clone the repository
git clone https://github.com/pablontiv/rootline.git
cd rootline

# Build
go build ./cmd/rootline/

# Run tests
go test ./... -race

# Lint
go vet ./...
```

Linting is configured via `.golangci.yml`. Pre-commit hooks are defined in `.pre-commit-config.yaml`.

## Code Conventions

- Follow standard Go conventions (`gofmt`, `go vet`)
- All JSON output carries `"version": 1` for contract stability
- Tests use the standard `testing` package
- No external test frameworks

## Pull Requests

1. Fork the repository and create a branch from `master`
2. Make your changes
3. Ensure tests pass: `go test ./... -race`
4. Ensure code is clean: `go vet ./...`
5. Open a PR with a clear description of the change

Keep PRs focused on a single concern. If your change involves both a bug fix and a new feature, split them into separate PRs.
