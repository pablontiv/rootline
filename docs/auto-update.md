# Auto-update — Staged Async Pattern

rootline updates itself automatically using a **staged async** pattern: the new binary is downloaded in the background during run N, and applied at the start of run N+1. The update is transparent — no prompts, no interruptions, no downtime.

The implementation lives upstream in [`github.com/pablontiv/picokit/autoupdate`](https://pkg.go.dev/github.com/pablontiv/picokit/autoupdate); rootline wires it from `cmd/rootline/main.go` via `autoupdate.New("pablontiv/rootline", "rootline")`.

## Flow per Invocation

```
Run N:
  1. ApplyStagedIfAvailable()  — sync: checks ~/.cache/rootline/staged/ for a newer binary
     → if found: atomic rename over current binary, re-exec (process is replaced)
     → if not found or not newer: continues normally
  2. go FetchAndStage(version) — goroutine: downloads next release in background
     → writes to ~/.cache/rootline/staged/<tag>/rootline
     → exits silently on any error

Run N+1:
  → staged binary is detected, applied, and rootline re-execs with the new version
```

The user-visible effect is that the version shown by `rootline --version` changes on the second invocation after a new release.

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

Only `major.minor.patch` participates. `parseSemver` truncates the version at the first `-` or `+`, and `isNewer` then compares with a strict `>`. Pre-release and build metadata are invisible to the comparison:

| Installed | Released | Replaced? | Why |
|-----------|----------|-----------|-----|
| `v4.0.8` | `v4.0.9` | yes | `4.0.9 > 4.0.8` |
| `v4.0.8-3-gabc123` | `v4.0.9` | yes | truncates to `4.0.8` |
| `v4.0.9+local.abc123` | `v4.0.9` | no | tie, and the test is `>` not `>=` |
| `v4.0.9+local.abc123` | `v4.1.0` | yes | `4.1.0 > 4.0.9` |

This is why `just install` versions a local build as `<highest known tag>+local.<sha>` rather than using `git describe`. From a fully synced checkout describe yields the release version and ties safely — but it drops *below* the release whenever tags are stale or `HEAD` predates the newest tag. The release then outranks it and the freshly installed binary is replaced on the very next run. Taking the highest known tag is deterministic regardless of where `HEAD` sits.

Clearing the staging directory does **not** prevent that: every invocation re-stages in a background goroutine, so the next one applies it.

## When Auto-update Does NOT Run

| Condition | Effect |
|-----------|--------|
| `version == "dev"` | Auto-update is always disabled — local builds (without ldflags) never touch network or cache |

There is no opt-out environment variable. The only way to disable auto-update is to build locally without ldflags (produces `version == "dev"`).

## Behavior by OS

| Concern | Unix (Linux / macOS) | Windows |
|---------|---------------------|---------|
| Apply strategy | `os.Rename(staged, current)` — atomic on same filesystem | rename current → `.old`, copy staged → current, remove `.old` |
| Re-exec | `syscall.Exec` — replaces process in same PID, transparent to caller | `exec.Command` + `os.Exit(0)` — new process launched, old exits |
| Binary in use | Not an issue — rename is atomic | Rename of in-use binary fails; copy avoids the lock |

## Silent Failure Policy

All auto-update errors are suppressed — the current command is never interrupted:

- **Network errors** during download: silently skip; staging directory is not created.
- **SHA256 mismatch**: returns an error internally; no file is written to the staging directory.
- **Permission errors** applying the update (e.g., `/usr/local/bin` owned by root): silently skip; the command proceeds normally.

## Troubleshooting

**The binary is not updating:**

1. Verify the installed binary is a release build: `rootline --version` should show a semver tag, not `dev`. A `dev` binary was built without ldflags — reinstall with `just install`.
2. Check that PATH resolves to a single binary: `just doctor-install`. A stale copy shadowing the real one in another directory is the most common cause of "the version never changes".
3. Check the staging directory for your OS (see the table above) — if it stays empty after a run, connectivity or permissions may be blocking.
4. Verify the installed binary sits in a directory you own. Applying an update to a root-owned path such as `/usr/local/bin` fails silently and leaves the old version in place.
5. Verify network access to `api.github.com` and `github.com`.

**Force an update now:**

```bash
rm -rf ~/Library/Caches/rootline/staged/   # macOS — see the table above for other OSes
rootline --version   # run N: downloads release in background
rootline --version   # run N+1: applies and re-execs with new version
```
