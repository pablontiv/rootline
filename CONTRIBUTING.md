# Contributing to Rootline

## Requirements

### Product Requirement
- **Go 1.26+** — Required to build and run Rootline

### Contributor Workflow
- **Git** — For pre-commit hooks, tests, and CI integration
- **[just](https://github.com/casey/just)** — Task runner for development recipes

**Note**: Rootline itself does not require Git to operate. Git is only needed for the contributor development workflow, not for end users.

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

## Just Recipes

Run `just --list` to see all available recipes. Key ones:

| Recipe | What it does |
|--------|-------------|
| `just check` | Format check + golangci-lint + go build |
| `just test` | Run all tests with race detector |
| `just fmt` | Auto-format code |
| `just validate` | Validate docs/roadmap with rootline |
| `just fix-docs` | Fix and propagate doc aggregates |
| `just test-install` | Offline installer tests (PowerShell when available) |
| `just dagu-version` | Report the user-level Dagu version |
| `just dagu-validate` | Validate the versioned feature and PoC DAGs |
| `just dagu-enqueue RUN_ID ISSUE PRIMARY BRANCH OMP_SESSION_ID` | Enqueue one issue lifecycle |
| `just dagu-complete-implementation RUN_ID` | Complete the implementation checkpoint after one clean commit |

## Workflow

1. Fork the repository
2. Create a feature branch from `master`
3. Make your changes
4. Run `just check` and `just test`
5. Commit using [Conventional Commits](https://www.conventionalcommits.org/)
6. Open a Pull Request

### Optional Dagu lifecycle

The [Dagu workflow](docs/dagu.md) runs the versioned `feature.yaml` and
`poc.yaml` on a user-level daemon. Enqueue from a clean primary clone on
`master`, using the `OMP_SESSION_ID` returned to the coordinating session. The
feature run owns triage, optional feasibility PoC routing, the fixed
`.workspace/worktrees/issue-<n>` worktree, dependency and conflict review,
local gates, guarded auto-merge, reconciliation, primary synchronization, and
terminal cleanup. The `candidate` GitHub check remains the required remote
merge gate.

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
| `pre-push` | Validates docs/roadmap changes, code-docs drift/coverage gates, syncs skills; no binary installation |

## Quality Gates

All PRs must pass:
- `go build ./...`
- `go test ./... -race` (85% coverage threshold)
- `go mod tidy` (no uncommitted changes)
- `golangci-lint run`
- `govulncheck ./...`
- `rootline validate --all docs/roadmap/`
- Secret scanning (`gitleaks`)
- Installer resolver tests on Linux, macOS, and Windows
- Native atomic-write and repair-rollback durability tests on Linux, macOS, and Windows (the Bash step must discover exactly 14 matching tests before running them with `-count=1`)
- The packaged PR candidate gate described below

### Packaged PR candidates

On `pull_request` events targeting `main` or `master`, `candidate-build` waits
for `ci`, `gitleaks`, `docs-validate`, `installer-tests`, and `durability`. It calls
[`go-candidate.yml`](https://github.com/pablontiv/crossbeam/blob/b012cf53f083e637a013080ef8c677d1af5e0161/.github/workflows/go-candidate.yml)
at the immutable Crossbeam revision
`b012cf53f083e637a013080ef8c677d1af5e0161`, with Go `1.26.2` and the event's
exact head repository, head SHA, base SHA, and PR number. Crossbeam reads the
trusted `.goreleaser.yml` from the base SHA, not the PR head; the snapshot template
must already exist at that base. The candidate version is
`<next-patch>-pr.<number>.g<short-sha>`, derived from a stable tag reachable from
the base. A stale base or a head that does not descend from it fails the build;
rebase before regenerating the candidate.

The PR-only `candidate` job runs even when dependencies fail or are skipped and
requires every dependency to succeed. It checks the returned artifact ID, name,
SHA-256 digest, source SHA, current workflow-run ID, and `expired: false` against
GitHub's artifact REST metadata, then downloads that exact ID rather than
searching by name. The pinned producer retains candidate artifacts for seven
days; an expired artifact cannot pass this gate.
It requires exactly eight regular files: `candidate.json`, `checksums.txt`,
and six `rootline_<candidate-version>_<os>_<arch>` archives for Linux, macOS
(`darwin`), and Windows on amd64 and arm64 (`.tar.gz` on Unix, `.zip` on Windows).
The manifest must match the PR, head SHA, base SHA, and candidate version; every
archive must match its manifest declaration and the exact checksum inventory.

The pinned Crossbeam producer already checks packaged `--version` in a
network-isolated smoke. The consumer independently extracts the declared Linux
amd64 archive and invokes its absolute, executable `rootline` binary, not a
source build. It requires `--version` to contain the exact candidate version
and rejects `dev`. An isolated working directory, HOME, TMPDIR, and XDG
directories are provided through `env -i` with only `/usr/bin:/bin` on PATH;
this consumer smoke isolates state, not the network. A temporary fixture
declares a `root: true` schema with a required string `title` and one valid
Markdown record. The packaged `validate --all` must return the version-2
`rootline/validate-batch` envelope, exactly one valid record and no invalid
records, structural errors, stem-health errors, or error notices. Temporary
smoke state is removed on exit. Both the consumer gate and native durability
jobs have 20-minute timeouts.

### Stable releases

Candidates are PR artifacts only: they do not create tags or GitHub Releases.
The stable release job remains on pushes to `main`/`master` and deliberate
major-release dispatches on `master`, using Crossbeam's `go-release.yml@v2`.
It requires the same five quality gates (`ci`, `gitleaks`, `docs-validate`,
`installer-tests`, and `durability`) and declares all five in
`quality-gate-jobs`. The existing post-release installer smoke matrix still
installs the published release on native Linux, macOS, and Windows runners.

## Reporting Issues

- **Bugs**: Use the bug report template
- **Features**: Use the feature request template
- **Security**: See [SECURITY.md](SECURITY.md) for responsible disclosure
