---
estado: Completed
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

## Three install mechanisms, now reconciled

`just install` was not initially the only install path. The repository installed the binary in two other ways that disagreed on both destination and version, so a single `rootline` on `PATH` was a matter of convention, not construction:

| Mechanism | Was | Now |
|-----------|-----|-----|
| `install.sh` (public `curl \| sh` installer) | `~/.local/bin` only when on `PATH`, else `/usr/local/bin` | `~/.local/bin` unconditionally (override: `ROOTLINE_INSTALL_DIR`) |
| `.githooks/post-merge`, `.githooks/pre-push` | own `install_rootline_binary` over `command -v rootline`, versioned with `git describe` | delegate to `just install` |
| `just install` | `~/bin`, `<highest tag>+local.<sha>` | `~/.local/bin`, `<highest tag>+local.<sha>` |

`install.sh:40-41` was what produced the orphan: it preferred `$HOME/.local/bin` only when that directory was on `PATH`, and the hooks then followed `PATH`, so which binary they overwrote depended on directory order. All three now converge on `~/.local/bin` with the `<highest tag>+local.<sha>` scheme, and the two duplicated copies of `install_rootline_binary` are gone.

`.githooks/post-merge` also ran `rootline fix --all "$REPO_ROOT/docs/epics/"`, a directory that does not exist (the roadmap lives at `docs/roadmap/`), silenced by `2>/dev/null || true`; `.githooks/pre-push` printed `docs/epics/ validation passed` while actually validating `docs/roadmap/`. Both `docs/epics/` references are corrected.

## Acceptance criteria

- `just install` targets `~/.local/bin` and builds with the same ldflags as goreleaser so an installed binary never reports `dev`. **Done.**
- `just install` versions the build as `<highest known tag>+local.<sha>`. Auto-update compares only `major.minor.patch` (`parseSemver` truncates at the first `-` or `+`; `isNewer` uses a strict `>`). A synced `git describe` ties safely, but it drops below the release whenever tags are stale or `HEAD` predates the newest tag, and the binary is then replaced on the next run. Taking the highest tag is deterministic regardless of where `HEAD` sits. **Done.**
- `install.sh` and both git hooks converge on `~/.local/bin`; the hooks delegate to `just install` instead of carrying their own build-and-install logic. **Done.**
- The dead `docs/epics/` references in `.githooks/post-merge` and `.githooks/pre-push` are corrected to `docs/roadmap/`. **Done.**
- `just doctor-install` uses `which -a` and fails when `PATH` resolves to more than one `rootline`, when it resolves outside `~/.local/bin`, or when the binary reports `dev`. It prints every path with its version. **Done.**
- Both copies of the Definition of Done (repo and user-global) call `just doctor-install` instead of `which rootline`.
- `docs/auto-update.md` states the staging base per OS, describes the empty-directory accumulation accurately, and documents the version-comparison semantics.

## Checksum verification portability

End-to-end validation of `install.sh` on macOS surfaced a pre-existing bug (introduced in `dfeee57`, unrelated to the path reconciliation): checksum verification used `sha256sum --check --status`, GNU coreutils syntax that the BSD/macOS `sha256sum` rejects. The public installer therefore aborted on every macOS run with `Checksum verification failed`, before installing anything. Linux with GNU coreutils was unaffected.

Fixed by computing the digest and comparing hex strings instead of relying on the `--check` flag — the same approach `install.ps1` already uses via `Get-FileHash`. A `sha256_hex` helper prefers `shasum -a 256` (present on macOS and most Linux) and falls back to `sha256sum` (GNU / busybox). Verified end-to-end on macOS, plus a negative test confirming a corrupted checksum is rejected.

## Notes

Clearing the staging directory before installing was evaluated and rejected: every `rootline` invocation re-stages in a background goroutine, so the next invocation applies the release anyway. The version-matching approach is what actually holds.
