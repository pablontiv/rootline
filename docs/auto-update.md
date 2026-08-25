# Auto-update — Staged Async Pattern

rootline updates itself automatically using a **staged async** pattern within its compatibility boundary: the new binary is downloaded in the background during run N, and an eligible staged binary is applied at the start of run N+1. Same-major stable releases are eligible. Cross-major releases are deliberately withheld, leave the staged binary in place, and print a stderr notice instructing the user to run the installer for a deliberate upgrade.

The implementation lives upstream in [`github.com/pablontiv/picokit/autoupdate`](https://pkg.go.dev/github.com/pablontiv/picokit/autoupdate); rootline wires it from `cmd/rootline/main.go` via `autoupdate.New("pablontiv/rootline", "rootline")`.

## Flow per Invocation

```
Run N:
  1. ApplyStagedIfAvailable()  — sync: checks ~/.cache/rootline/staged/ for a newer binary
     → if found, newer, and same-major compatible: atomic rename over current binary, re-exec (process is replaced)
     → if found and newer but outside the compatibility boundary: keep staged binary, print a notice, continue
     → if not found or not newer: continues normally
  2. go FetchAndStage(version) — goroutine: downloads next release in background
     → writes to ~/.cache/rootline/staged/<tag>/rootline
     → exits silently on any error

Run N+1:
  → eligible staged binary is detected, applied, and rootline re-execs with the new version
```

For same-major releases, the user-visible effect is that the version shown by `rootline --version` changes on the second invocation after a new release. For cross-major releases, the version does not change automatically; reinstall deliberately with the installer command for the new major.

## Staging Directory

The base is Go's `os.UserCacheDir()`, so the path is OS-dependent:

| OS | Staging base |
|----|--------------|
| macOS | `~/Library/Caches/rootline/staged/` |
| Linux | `~/.cache/rootline/staged/` |
| Windows | `%LocalAppData%\rootline\staged\` |

```
<staging base>/
  vX.Y.Z/
    rootline          # Linux / macOS
    rootline.exe      # Windows
```

The directory is created by the downloader. Each staged release occupies its own version subdirectory.

Applying an update renames the staged binary over the running one, so the binary leaves the staging directory — but the now-empty version directory is **not** removed. They accumulate one empty directory per release. Harmless (zero bytes), and safe to delete at any time.

## Version Comparison

Only `major.minor.patch` participates in ordering. `parseSemver` truncates the version at the first `-` or `+`, and `isNewer` then compares with a strict `>`. Ordering is necessary but not sufficient: a release must also satisfy the same-major compatibility policy. Pre-release and build metadata are invisible to the comparison:

| Installed | Released | Replaced? | Why |
|-----------|----------|-----------|-----|
| `v4.0.8` | `v4.0.9` | yes | `4.0.9 > 4.0.8` |
| `v4.0.8-3-gabc123` | `v4.0.9` | yes | truncates to `4.0.8` |
| `v4.0.9+local.abc123` | `v4.0.9` | no | tie, and the test is `>` not `>=` |
| `v4.0.9+local.abc123` | `v4.1.0` | yes | `4.1.0 > 4.0.9` and same major |
| `v4.0.9` | `v5.0.0` | no | newer, but cross-major; withheld for deliberate reinstall |

This is why `just install` versions a local build as `<highest known tag>+local.<sha>` rather than using `git describe`. From a fully synced checkout describe yields the release version and ties safely — but it drops *below* the release whenever tags are stale or `HEAD` predates the newest tag. The release then outranks it and the freshly installed binary is replaced on the very next run. Taking the highest known tag is deterministic regardless of where `HEAD` sits.

Clearing the staging directory does **not** prevent that: every invocation re-stages in a background goroutine, so the next one applies it.

## When Auto-update Does NOT Apply

| Condition | Effect |
|-----------|--------|
| `version == "dev"` | Auto-update is always disabled — local builds (without ldflags) never touch network or cache |
| Staged release crosses the same-major compatibility boundary | The update is withheld, the staged binary remains in cache, a stderr notice is printed, and the current command continues with the installed binary |

There is no opt-out environment variable. The only way to disable all auto-update activity is to build locally without ldflags (produces `version == "dev"`). To cross a major-version boundary, run the installer deliberately.

## Behavior by OS

| Concern | Unix (Linux / macOS) | Windows |
|---------|---------------------|---------|
| Apply strategy | `os.Rename(staged, current)` — atomic on same filesystem | rename current → `.old`, copy staged → current, remove `.old` |
| Re-exec | `syscall.Exec` — replaces process in same PID, transparent to caller | `exec.Command` + `os.Exit(0)` — new process launched, old exits |
| Binary in use | Not an issue — rename is atomic | Rename of in-use binary fails; copy avoids the lock |

## Silent Failure Policy

Most auto-update failures are suppressed — the current command is never interrupted. Compatibility-boundary withholding is the deliberate exception: it prints a notice so the user knows to reinstall intentionally.

- **Network errors** during download: silently skip; staging directory is not created.
- **SHA256 mismatch**: returns an error internally; no file is written to the staging directory.
- **Permission errors** applying the update (e.g., `/usr/local/bin` owned by root): silently skip; the command proceeds normally.

## Troubleshooting

**The binary is not updating:**

1. Verify the installed binary is a release build: `rootline --version` should show a semver tag, not `dev`. A `dev` binary was built without ldflags — reinstall with `just install`.
2. Check stderr for `incompatible update withheld`. If present, the newest staged release crosses the compatibility boundary; run the installer deliberately to move to that major version.
3. Check that PATH resolves to a single binary: `just doctor-install`. A stale copy shadowing the real one in another directory is another common cause of "the version never changes".
4. Check the staging directory for your OS (see the table above) — if it stays empty after a run, connectivity or permissions may be blocking. If it contains a newer cross-major release, the boundary is working as designed.
5. Verify the installed binary sits in a directory you own. Applying an eligible update to a root-owned path such as `/usr/local/bin` fails silently and leaves the old version in place.
6. Verify network access to `api.github.com` and `github.com`.

**Force an eligible same-major update now:**

```bash
rm -rf ~/Library/Caches/rootline/staged/   # macOS — see the table above for other OSes
rootline --version   # run N: downloads release in background
rootline --version   # run N+1: applies and re-execs with new version
```
