---
estado: In Progress
tipo: task
---
# T011: Establish a single supported install path with a duplicate-binary guard

**Contribuye a**: eliminating the class of "the CLI is not updating" reports caused by more than one `rootline` on `PATH`, and making the Definition of Done capable of detecting it.

## Context

The repository had no install recipe at all. `just --list` offered `check`, `coverage`, `coverage-check`, `default`, `fix-docs`, `fmt`, `test`, and `validate` — every historical installation was therefore ad-hoc, to whatever directory was convenient. Two binaries resulted:

| Path | Version | Origin |
|------|---------|--------|
| `~/bin/rootline` | current release | maintained by `picokit/autoupdate` |
| `~/.local/bin/rootline` | `dev` | `go build` without ldflags, copied manually |

The duplicate survived because the Definition of Done verified installation with `which rootline`, which returns only the first `PATH` hit. The newer binary in `~/bin` answered, the check passed, and the shadowed `dev` copy stayed invisible. The check could not detect the condition it was meant to catch.

Two further defects surfaced while fixing this:

1. `docs/auto-update.md` documented the staging directory as `~/.cache/rootline/staged/`. The real base is Go's `os.UserCacheDir()`, which is `~/Library/Caches` on macOS. Following the troubleshooting steps led to a non-existent path and the conclusion that auto-update was broken.
2. The same doc claimed staging directories are cleaned up when the binary is applied. Applying moves the binary out but leaves the empty version directory behind; they accumulate one per release.

## Three competing install mechanisms

Discovered while merging: `just install` is not yet the only path. The repository already installs the binary in two other ways, and they disagree on both destination and version.

| Mechanism | Destination | Version |
|-----------|-------------|---------|
| `install.sh` (public `curl \| sh` installer) | `~/.local/bin` when it is on `PATH`, else `/usr/local/bin` | downloaded release |
| `.githooks/post-merge`, `.githooks/pre-push` | whatever `command -v rootline` resolves to | `git describe --tags` |
| `just install` | `~/bin` | `<highest tag>+local.<sha>` |

`install.sh:40-41` is what produced the orphan: it prefers `$HOME/.local/bin` whenever that directory is on `PATH`. The hooks then follow `PATH`, so which binary they overwrite depends on directory order.

Two further defects in `.githooks/post-merge`: it runs `rootline fix --all "$REPO_ROOT/docs/epics/"`, a directory that does not exist (the roadmap lives at `docs/roadmap/`), silenced by `2>/dev/null || true`; and it versions rebuilds with `git describe`, which falls below the current release whenever tags are stale.

## Acceptance criteria

- `just install` targets `~/bin` and builds with the same ldflags as goreleaser so an installed binary never reports `dev`. **Done.**
- `just install` versions the build as `<highest known tag>+local.<sha>`. Auto-update compares only `major.minor.patch` (`parseSemver` truncates at the first `-` or `+`; `isNewer` uses a strict `>`). A synced `git describe` ties safely, but it drops below the release whenever tags are stale or `HEAD` predates the newest tag, and the binary is then replaced on the next run. Taking the highest tag is deterministic regardless of where `HEAD` sits. **Done.**
- **Open:** reconcile `install.sh`, the two git hooks, and `just install` onto one destination and one version scheme, so a single `rootline` can exist on `PATH` by construction rather than by convention.
- **Open:** fix or remove the dead `docs/epics/` call in `.githooks/post-merge`.
- `just doctor-install` uses `which -a` and fails when `PATH` resolves to more than one `rootline`, when it resolves outside `~/bin`, or when the binary reports `dev`. It prints every path with its version.
- Both copies of the Definition of Done (repo and user-global) call `just doctor-install` instead of `which rootline`.
- `docs/auto-update.md` states the staging base per OS, describes the empty-directory accumulation accurately, and documents the version-comparison semantics.

## Notes

Clearing the staging directory before installing was evaluated and rejected: every `rootline` invocation re-stages in a background goroutine, so the next invocation applies the release anyway. The version-matching approach is what actually holds.
