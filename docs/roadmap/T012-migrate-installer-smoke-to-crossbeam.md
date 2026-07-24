---
estado: Pending
tipo: task
---
# T012: Migrate the installer smoke test to a reusable crossbeam workflow

**Contribuye a**: keeping install-script CI generic and reusable across pablontiv Go CLIs, instead of duplicating it inline per repo.

## Context

`ci.yml` gained an `installer-smoke` job (added alongside T011): a post-release matrix over `ubuntu-latest`, `macos-latest`, and `windows-latest` that runs the public install scripts against the freshly published release and asserts `rootline --version`. It exists because a GNU-only `sha256sum --check` in `install.sh` silently broke installation on every Mac and nothing in CI executed the installers to catch it — the release smoke only checked the binary's `--version`/`--help`, never the install path.

The job is inline for now, mirroring how `docs-validate` stays inline because it is repo-specific. But installer smoke testing is **not** repo-specific: any Go CLI distributed via `install.sh` / `install.ps1` + goreleaser benefits from the same three-OS check. It is a natural sibling of the existing `go-ci` and `go-release` reusable workflows in `pablontiv/crossbeam`.

## Why not migrate immediately

- The inline job should prove its value and stabilize (flakiness tuning, retry counts) before being generalized.
- Generalizing requires parameterizing the binary name, the install-script paths/names, and the per-OS invocation, plus deciding how the reusable workflow discovers the "latest" release across consuming repos.

## Acceptance criteria

- A reusable workflow (e.g. `pablontiv/crossbeam/.github/workflows/installer-smoke.yml@v1`) accepts at least `binary-name` and runs `install.sh` on Ubuntu/macOS and `install.ps1` on Windows against the latest published release.
- `rootline`'s `ci.yml` replaces the inline `installer-smoke` job with a `uses:` reference to it, preserving `needs: [release]` and the post-release trigger.
- Behavior is unchanged: the same three OSes, the same full-path `--version` assertion, the same retry-on-network-flake.
- The inline job's explanatory comment and this task are removed or updated to point at the crossbeam workflow.

## Notes

Possible enhancement to weigh during migration: also run the smoke on pull requests that touch `install.sh` / `install.ps1` (path-filtered) against the previous release, so install-script regressions are caught before merge rather than only after release. The checksum bug this whole line of work uncovered would have been caught at PR time by such a gate.
