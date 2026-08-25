# Justfile for Rootline
set shell := ["bash", "-c"]

# Default recipe
default: check test

# Run all checks (fmt, lint, build)
check:
    gofmt -l . | grep . && { echo "ERROR: Code not formatted. Run: just fmt"; exit 1; } || true
    golangci-lint run
    go build ./...

# Format code
fmt:
    gofmt -l -w .

# Run tests
test:
    go test ./... -race

# Run deterministic offline tests for the public installers. PowerShell coverage
# runs when pwsh is available locally and unconditionally on the Windows CI leg.
test-install:
    sh tests/installers/install-sh-test.sh
    if command -v pwsh >/dev/null 2>&1; then pwsh -NoProfile -File tests/installers/install-ps1-test.ps1; else echo "pwsh not found; PowerShell installer tests skipped locally"; fi

# Show coverage per package and total
coverage:
    go test ./... -coverprofile=coverage.out
    go run github.com/pablontiv/picokit/cmd/pkcov report

# Check coverage meets per-package floors
coverage-check: coverage
    go run github.com/pablontiv/picokit/cmd/pkcov check

# Build and install a local development binary to ~/.local/bin
#
# The single canonical install destination. install.sh (the public installer)
# defaults here too. Run `just install` explicitly when you want to install the
# current checkout; the pre-push hook intentionally does not install unmerged
# branch builds.
install: && doctor-install
    #!/usr/bin/env bash
    set -euo pipefail
    dest="$HOME/.local/bin"
    mkdir -p "$dest"
    git fetch --tags --quiet origin 2>/dev/null || true
    # Version as <highest known tag>+local.<sha>, NOT `git describe`.
    #
    # Auto-update compares only major.minor.patch: picokit's parseSemver strips
    # everything from the first "-" or "+", then isNewer uses a strict ">".
    # From a synced checkout `git describe` yields the release version and ties,
    # but it silently drops below the release whenever tags are stale or HEAD
    # predates the newest tag — and then a released 4.0.9 outranks a describe of
    # v4.0.8-3-gabc and replaces this binary on the next run. Taking the highest
    # known tag is deterministic regardless of where HEAD sits, so the local
    # build survives until a genuinely newer release ships — and then correctly
    # gives way to it. The +local.<sha> suffix keeps the build identifiable.
    #
    # Clearing the staging directory instead does NOT work: every rootline run
    # re-stages in a background goroutine, so the next invocation applies it.
    tag="$(git tag --sort=-v:refname | head -1)"
    [ -n "$tag" ] || tag="v0.0.0"
    sha="$(git rev-parse --short HEAD)"
    git diff --quiet || sha="${sha}.dirty"
    version="${tag}+local.${sha}"
    # Same ldflags as goreleaser, so an installed binary never reports "dev".
    # A "dev" build disables auto-update entirely (see docs/auto-update.md).
    go build -ldflags "-X main.version=${version}" -o "$dest/rootline" ./cmd/rootline
    echo "installed $dest/rootline ($version)"

# Fail if PATH resolves rootline to anything but a single ~/.local/bin binary
doctor-install:
    #!/usr/bin/env bash
    set -euo pipefail
    dest="$HOME/.local/bin/rootline"
    # which -a, never which: a bare which returns only the first PATH hit, so a
    # stale binary shadowed in another directory passes the check unseen.
    found="$(which -a rootline 2>/dev/null || true)"
    count="$(printf '%s' "$found" | grep -c . || true)"
    if [ "$count" -gt 1 ]; then
        echo "ERROR: $count rootline binaries in PATH — only $dest is supported:"
        printf '%s\n' "$found" | while read -r bin; do
            [ -n "$bin" ] || continue
            printf '  %-40s %s\n' "$bin" "$("$bin" --version 2>/dev/null || echo '<unreadable>')"
        done
        echo "Remove every copy except $dest, then re-run."
        exit 1
    fi
    if [ "$count" -eq 0 ]; then
        echo "ERROR: rootline is not in PATH — run: just install"
        exit 1
    fi
    if [ "$found" != "$dest" ]; then
        echo "ERROR: rootline resolves to $found, expected $dest"
        echo "A copy in an earlier PATH entry is shadowing it — remove it."
        exit 1
    fi
    case "$("$found" --version)" in
        *dev*) echo "ERROR: $found reports version 'dev' — auto-update is disabled on dev builds"; exit 1 ;;
    esac
    echo "install ok: $found ($("$found" --version))"

# Validate docs
validate:
    rootline validate --all docs/roadmap/

# Fix docs (propagate aggregates)
fix-docs:
    rootline fix --all docs/roadmap/
