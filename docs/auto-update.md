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

```
~/.cache/rootline/staged/
  v1.2.0/
    rootline          # Linux / macOS
    rootline.exe      # Windows
```

The directory is created by the downloader. Each staged release occupies its own version subdirectory. Staging directories are cleaned up automatically when the binary is applied.

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

1. Verify the installed binary is a release build: `rootline --version` should show a semver tag, not `dev`.
2. Check staging directory: `ls ~/.cache/rootline/staged/` — if empty after running, connectivity or permissions may be blocking.
3. Verify network access to `api.github.com` and `github.com`.

**Force an update now:**

```bash
rm -rf ~/.cache/rootline/staged/
rootline --version   # run N: downloads release in background
rootline --version   # run N+1: applies and re-execs with new version
```
