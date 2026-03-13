# Contributing to Rootline

## Development Setup

```bash
git clone https://github.com/pablontiv/rootline.git
cd rootline

# Set up git hooks
git config core.hooksPath .githooks

# Verify environment
just check
just test
```

Requires Go 1.24+ and [just](https://github.com/casey/just).

## Just Recipes

Run `just --list` to see all available recipes. Key ones:

| Recipe | What it does |
|--------|-------------|
| `just check` | Format check + golangci-lint + go build |
| `just test` | Run all tests with race detector |
| `just fmt` | Auto-format code |
| `just validate` | Validate docs/epics with rootline |
| `just fix-docs` | Fix and propagate doc aggregates |

## Workflow

1. Fork the repository
2. Create a feature branch from `master`
3. Make your changes
4. Run `just check` and `just test`
5. Commit using [Conventional Commits](https://www.conventionalcommits.org/)
6. Open a Pull Request

## Releasing

Releases are fully automated via CI. On push to `master`, the CI pipeline analyzes conventional commit prefixes, auto-tags, and triggers goreleaser for cross-platform binaries and GitHub Releases. No manual release steps needed — just push with conventional commit messages.

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

## Git Hooks

Hooks live in `.githooks/` and are activated with `git config core.hooksPath .githooks`.

| Hook | What it does |
|------|-------------|
| `pre-commit` | gofmt check + golangci-lint + gitleaks secret scan |
| `commit-msg` | Validates conventional commit format |
| `pre-push` | Validates docs/epics, checks code-docs drift, syncs skills, rebuilds binary |
| `post-merge` | Syncs skills, rebuilds binary, propagates doc aggregates |

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
