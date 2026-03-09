# Contributing to Rootline

## Development Setup

```bash
git clone https://github.com/pablontiv/rootline.git
cd rootline

# Set up git hooks
git config core.hooksPath .githooks

# Build
go build ./cmd/rootline/

# Test
go test ./... -race

# Lint
golangci-lint run
```

Requires Go 1.24+.

## Workflow

1. Fork the repository
2. Create a feature branch from `master`
3. Make your changes
4. Run tests and linter
5. Commit using [Conventional Commits](https://www.conventionalcommits.org/)
6. Open a Pull Request

## Commit Convention

```
type(scope): description
```

| Type | When to use |
|------|-------------|
| `feat` | New user-facing functionality |
| `fix` | Bug fix |
| `refactor` | Internal restructuring, no behavior change |
| `perf` | Performance improvement |
| `test` | Adding or updating tests |
| `docs` | Documentation only |
| `chore` | Build, CI, dependency updates |

Breaking changes use `!` suffix: `feat!: remove deprecated flag`

## Quality Gates

All PRs must pass:
- `go build ./...`
- `go test ./... -race` (85% coverage threshold)
- `go mod tidy` (no uncommitted changes)
- `golangci-lint run`
- `govulncheck ./...`
- `rootline validate --all docs/epics/`

## Reporting Issues

- **Bugs**: Use the bug report template
- **Features**: Use the feature request template
- **Security**: See [SECURITY.md](SECURITY.md) for responsible disclosure
