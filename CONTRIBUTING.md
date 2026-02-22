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

# Install pre-commit hooks
pip install pre-commit   # or: brew install pre-commit
pre-commit install

# Build
go build ./cmd/rootline/

# Run tests
go test ./... -race

# Lint
golangci-lint run ./...
```

Pre-commit hooks run `golangci-lint` and `gofmt` automatically on each commit (configured in `.pre-commit-config.yaml`). Lint rules are defined in `.golangci.yml`.

## Code Conventions

- Follow standard Go conventions (`gofmt`, `go vet`)
- The project includes an `.editorconfig` file — configure your editor to respect it (Go uses tabs, YAML/Markdown use 2 spaces)
- All JSON output carries `"version": 1` for contract stability
- Tests use the standard `testing` package
- No external test frameworks

## Commit Messages

This project uses **conventional commits**. The format is:

```
type(scope): description
```

Valid types: `feat`, `fix`, `docs`, `test`, `refactor`, `ci`, `chore`, `perf`, `style`

Examples:
- `feat(graph): add cycle detection to dependency graph`
- `fix(extract): handle missing trailing newline in frontmatter`
- `docs(roadmap): update task states for F05`

A commit-msg hook validates the format automatically.

## Pull Requests

1. Fork the repository and create a branch from `master`
2. Make your changes
3. Ensure tests pass: `go test ./... -race`
4. Ensure lint passes: `golangci-lint run ./...`
5. Use conventional commit format for your commit messages
6. Open a PR with a clear description of the change

Keep PRs focused on a single concern. If your change involves both a bug fix and a new feature, split them into separate PRs.
