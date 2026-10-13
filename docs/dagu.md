# Local Dagu pilot

This optional contributor workflow pins **Dagu v2.19.2**. It does not change hosted
CI, release workflows, installers, or Rootline's product behavior. A local macOS
run does not replace hosted Linux/Windows checks or native Windows installer
coverage. Nothing in this guide claims that a pilot phase has passed: inspect the
run's actual evidence.

## Prerequisites and bootstrap

Run from a clean, committed checkout with Git, Bash/POSIX shell, `just`, a
currently patched Go 1.26+ toolchain (or the newer version required by `go.mod`),
`golangci-lint`, `govulncheck`, and
`gitleaks` on PATH. Bootstrap additionally needs `curl`, `tar`, and a SHA-256
utility, plus network access to the pinned Dagu release. Go dependency downloads
and vulnerability databases may also need network access. `pwsh` is optional
locally: `just test-install` skips PowerShell tests without it; hosted Windows
coverage remains necessary. Forge modes also require `gh`, GitHub authentication,
and permission to push a branch, create a PR, publish commit status, and (for merge)
merge that PR. Repository policies may still block a merge.

```sh
just dagu-bootstrap
just dagu-version
just dagu-validate
just test-dagu
```

The tracked POSIX `.workspace/dagu/bin/dagu` wrapper bootstraps the pinned binary
under ignored `.workspace/dagu/data/`, verifies the release archive, and verifies
the cached executable against a tracked platform checksum on every invocation.
**DAGU_HOME always resolves to this checkout's absolute `.workspace/dagu`
directory**, not a user's global Dagu home. The wrapper exports it before every
command. Tracked configuration and DAG definitions stay in `.workspace/dagu/`;
only `data/` and `logs/` there are ignored. Neither shadow nor forge modes need
a background server.

## Modes and commands

Choose a unique run ID and space ID; use simple identifiers rather than paths.
The recipe arguments are `RUN_ID [MODE [SPACE_ID [ISSUE_NUMBER]]]`, with defaults
`shadow`, `local`, and `0`:

```sh
just dagu-start pilot-001
just dagu-start pilot-002 shadow team-a
just dagu-start pilot-003 pr team-a 261
just dagu-start pilot-004 merge team-a 261
```

- `shadow` runs the local gate only. It cannot push, create PRs, publish status,
  or merge. This is the default, including when MODE is omitted.
- `pr` additionally pushes the tested branch, finds or creates its PR, publishes
  `dagu/rootline-local` for the exact tested commit, waits for every reported
  hosted workflow check, and stops without merging.
- `merge` includes the same PR and hosted-CI flow, then automatically performs a
  guarded squash merge. It is never the default. Both forge modes require a
  positive GitHub issue number; the generated PR title uses the tested commit
  subject and its body closes that issue.

The pilot gates committed source; it does not author changes. `sync` requires the
primary checkout on clean `master`, fetches and fast-forwards from `origin`, then
branches the isolated worktree from that local tip. A forge run therefore needs
the intended commits on the clean local tip before it starts; an unchanged tip
has nothing GitHub can turn into a PR.

The equivalent start invocation is:

```sh
.workspace/dagu/bin/dagu start --run-id pilot-001 \
  .workspace/dagu/dags/rootline-feature.yaml -- \
  SPACE_ID=local RUN_ID=pilot-001 MODE=shadow ISSUE_NUMBER=0
```

Both DAG definitions can be validated with `just dagu-validate`. The recovery DAG
is `.workspace/dagu/dags/rootline-recovery.yaml`; it takes the same SPACE_ID and
RUN_ID owner identity. For straightforward recovery, prefer the recipe below.

## Isolation, gate, and evidence

Each owner gets a deterministic external worktree:

```text
${ROOTLINE_WORKTREE_ROOT:-$HOME/.local/share/rootline/worktrees}/<space_id>/<run_id>
```

Its branch is `dagu/<space_id>/<run_id>`. The shared atomic Git lock lives at
`rootline-dagu.lock` under the repository's **common Git directory**, so linked
worktrees cannot bypass it. It is a regular file published atomically by hard-linking
a fully written temporary owner file.
It records both space ID and run ID, refuses another owner, and only matching
cleanup may release it. Never delete a lock merely because another run is blocked.

State and evidence live at:

```text
$DAGU_HOME/data/rootline-runs/<space_id>/<run_id>/
```

The run's `tested.sha`, per-phase `phases/<phase>` evidence, final local
`evidence`, immutable `issue`, and forge-stage `ci.sha`/`ci.checks` files record
the commit and completed gates. Forge actions read this completed evidence
rather than trusting caller-supplied identifiers.

The local gate consists of these phases, in order:

1. `go mod tidy`, requiring no resulting changes (`tidy-clean`).
2. `just check` (format check, lint, build).
3. `just test` (race-enabled tests).
4. `just coverage-check` (per-package coverage floors).
5. `just test-install` (offline installer tests).
6. Build Rootline from the isolated worktree and use that binary to run
   `validate --all docs/roadmap/` (never a preinstalled Rootline).
7. `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...` (pinned; no
   globally installed binary required).
8. Gitleaks secret scanning.

Every normal phase uses the repository's selected Go toolchain; the vulnerability
phase alone pins patched `go1.27.2`. Every phase verifies a clean worktree and an
unchanged tested SHA. Final evidence is bound to that exact SHA; a branch name,
old log, or partially completed gate is not sufficient proof.
Check phases normalize the process umask to `0022`, so file-mode assertions do
not depend on the shell that started Dagu. They also run with **GH_TOKEN and
GITHUB_TOKEN unset**. Implicit dotenv loading is disabled. This isolates those
GitHub credential variables, not every credential mechanism on the host (for
example Git credential helpers or SSH agents); do not treat the pilot as a
general sandbox.

## Automatic merge and forge behavior

PR lookup happens before creation, so retries reuse the existing PR. Publishing
`dagu/rootline-local` is skipped when that exact commit already has a successful
status. The `wait_ci` step then uses `gh pr checks --watch --fail-fast`, requires
at least one hosted workflow check, excludes non-workflow commit statuses from
that evidence, and records each passing or intentionally skipped check plus the
exact SHA. A PR with no hosted check fails closed. Push, PR, status, CI wait, and
merge operations are designed to be idempotent.

After `wait_ci` persists successful exact-SHA evidence, `pr` mode ends without
merging. In `merge` mode, the downstream merge runs automatically. Starting a
run in `merge` mode authorizes that guarded action; use `pr` mode when review
must stop before merge.

Immediately before merging, the forge action rechecks all hosted CI, the
successful `dagu/rootline-local` commit status, and that the PR head equals the
evidence's tested SHA, then uses:

```text
gh pr merge --squash --match-head-commit <tested-sha>
```

A moved PR head, missing local success status, or failed/pending hosted check
cannot reuse old evidence. GitHub branch protection and permissions still apply.
After a successful squash, the workflow verifies that the primary checkout is
still clean and unchanged from its recorded starting commit, then realigns local
`master` to `origin/master`; this removes the pre-squash local commit topology
without discarding uncommitted work.

## Retry, recovery, and fault progression

```sh
just dagu-retry pilot-001 FAILED_STEP
just dagu-recover pilot-001 local
```

Retry uses Dagu's stored parameters for the original run and resumes the named
failed step plus its downstream steps, not a new mode or checkout. Inspect the
failed step and evidence before retrying; never bypass preconditions to obtain
a green result. If terminal cleanup already removed the worktree, the retry
recreates it at the immutable `tested.sha` and phase checkpoints prevent
successful upstream gate work from rerunning. A changed source SHA or a dirty
worktree is not recoverable evidence: commit the intended source change and
start a fresh run ID instead.

Normal success, exhausted failure, and abort clean up the owned
local worktree, branch, and lock. After a process crash or interrupted cleanup,
`dagu-recover` invokes
`cleanup.sh SPACE_ID RUN_ID`; it must refuse to release another owner's lock or
remove another owner's resources. Evidence remains in the run directory for
inspection. Recovery does not undo remote branches, PRs, or published statuses.

Advance deliberately. Phase 1 requires five successful `shadow` runs plus
restart/checkpoint, injected E2E crash, lock-contention, stale-SHA, and
post-creation network-failure scenarios. Phase 2 may create one controlled
`docs` or `chore` PR, publish `dagu/rootline-local`, wait for all hosted CI, and
finish in `pr` mode without merging. Only phase 3 may select `merge`, after the
prior evidence has been reviewed and repository rules protect the PR merge path.
`just test-dagu` exercises the offline fault contracts; real GitHub permissions,
protections, and merge behavior still need deliberate live evaluation. Do not
enable forge modes based solely on an offline test result.

The pilot is local and serialized by its Git lock. It is not a replacement for
hosted platform coverage or a distributed worker system. Automatic merge is
limited to an explicit `merge` run and remains fail-closed on stale or incomplete
evidence.
