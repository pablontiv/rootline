---
estado: Specified
---

# Product-Owned Skill Distribution Design

**Date:** 2026-08-28
**Status:** Approved design
**Issue:** #210

## Purpose

Make the Rootline repository the single canonical owner of its agent skill and distribute that skill to supported global runtimes through verifiable symlinks instead of divergent mutable copies.

The current pre-push hook deletes and recopies the canonical skill into the Claude user directory. OpenCode and Pi now share the generic Agents skill directory, while older installations may still contain an obsolete OpenCode-specific copy. Rootline must not retain compatibility code for obsolete destinations. Breaking remediation belongs in `CHANGELOG.md` and is performed deliberately by an agent.

## Decisions

1. The canonical skill tree remains `.claude/skills/rootline` in a stable Rootline checkout.
2. Rootline manages exactly two global destinations: `~/.claude/skills/rootline` and `~/.agents/skills/rootline`.
3. OpenCode and Pi both consume the shared Agents destination.
4. Rootline carries no runtime detection, migration, or compatibility branch for `~/.config/opencode/skills/rootline` or any other obsolete path.
5. Obsolete-path remediation is a breaking change documented in `CHANGELOG.md`; an agent must read and apply that procedure explicitly.
6. Skill distribution is an explicit `rootline skill` CLI lifecycle. Git hooks never mutate skill installations.
7. Every mutation is authorized by a digest of the complete observed plan. A generic confirmation or reusable force flag is insufficient.
8. Every existing supported destination is backed up before replacement.
9. Installations are symlinks. Rootline never falls back to a copied skill tree.
10. Receipts are append-only and preserve enough evidence to verify or restore each accepted operation.
11. Operations are best-effort and observable: a failure rolls back the destination currently being changed, preserves earlier verified actions, and records an incomplete receipt.
12. The CLI emits one versioned JSON contract per verb. It does not maintain a parallel human table representation in this delivery.

## Scope

### In scope

- `rootline skill install`, `status`, `uninstall`, and `restore`.
- Stable-checkout source validation.
- Deterministic canonical-tree and preimage digests.
- Plan-bound approval and time-of-check/time-of-use protection.
- Backups, per-destination rollback, append-only receipts, and restoration.
- Claude and shared Agents destinations.
- Versioned JSON output and global `--field` extraction.
- Removal of skill mutation from `pre-push`.
- Breaking-change guidance in `CHANGELOG.md` and agent-facing documentation.
- Tests with temporary repositories, state directories, and homes on supported platforms.
- A successor ADR that preserves the ban on post-merge mutation and removes the former pre-push exception.

### Out of scope

- Detection or migration of legacy runtime paths.
- A generic artifact or plugin distribution framework.
- Copy-based installation or fallback.
- Installation from remote releases.
- Automatic home mutation during hooks, tests, merges, or upgrades.
- A `--force` bypass.
- A second table or prose output representation.
- Automatic closure of issue #210 before the observed machine migration is verified.

## Public CLI

The root command gains one command group:

```text
rootline skill install [--source <repo>] [--approve <plan-digest>]
rootline skill status [--source <repo>]
rootline skill uninstall [--approve <plan-digest>]
rootline skill restore --receipt <id> [--approve <plan-digest>]
```

All commands use Rootline's existing global JSON output and `--field` machinery. The output-format registry explicitly declares the new command paths as JSON-only.

Mutating commands are two-phase:

1. Without `--approve`, inspect current state and emit an immutable plan. Exit is zero because no application was attempted.
2. With `--approve`, recompute the plan and apply it only if the supplied digest exactly matches.

Approval always uses two invocations. The first invocation emits the plan; a second invocation supplies its exact digest with `--approve`. The CLI never mixes an interactive prompt with its JSON contract.

No verb accepts `--force`.

## Architecture

### Cobra command layer

A new `cmd/rootline/skill.go` owns flag parsing, output envelopes, exit semantics, and calls into the service package. It contains no filesystem mutation logic.

The command layer adds all concrete paths to `commandOutputFormats`, so the repository's command-coverage test prevents accidental format ambiguity.

### `internal/skilldist`

The lifecycle implementation lives in a focused internal package. Its service options inject home, state directory, clock, and receipt-ID generation; production supplies native defaults while tests supply temporary deterministic values. Filesystem behavior remains real rather than mocked so symlink and rename semantics are exercised.

The package contains these units:

#### SourceResolver

- Resolves `--source`, defaulting to the Git repository containing the current directory.
- Canonicalizes the repository path.
- Requires `.claude/skills/rootline/SKILL.md`.
- Requires the source repository's `.git` entry to be a directory.
- Rejects linked worktrees, whose `.git` entry is a file.
- Rejects staged, unstaged, or untracked changes inside the canonical skill tree so commit and digest describe one committed source.
- Computes the source commit and deterministic tree digest.

The linked-worktree rule ensures installed symlinks never target an implementation worktree that may later be removed. A normal primary checkout is the supported source identity.

#### Inventory

Inventories only the supported destinations:

| Destination ID | Path |
|---|---|
| `claude` | `<home>/.claude/skills/rootline` |
| `agents` | `<home>/.agents/skills/rootline` |

Each path is classified with `Lstat` as:

- absent;
- regular directory;
- correct symlink;
- divergent symlink;
- unsupported file type.

For a directory, Inventory computes a deterministic tree digest. For a symlink, it records both the lexical target and the canonical resolved target. Unsupported types fail closed.

Inventory contains no obsolete OpenCode-specific path constant or probe.

#### Planner

Planner converts source and inventory snapshots into ordered actions:

- `create_symlink`;
- `replace_with_symlink`;
- `remove_managed_symlink`;
- `restore_preimage`;
- `no_op`.

A plan digest binds:

- operation;
- canonical source path, commit, and tree digest;
- destination IDs and paths;
- every preimage type, digest, and symlink target;
- ordered proposed actions;
- referenced receipt for restore.

Any source, destination, or action change creates a different digest.

#### Executor

Executor applies an approved plan one destination at a time:

1. Re-inventory source and destination.
2. Refuse if the recomputed plan digest differs.
3. Create the destination's backup without overwriting existing state.
4. Move the current destination to a unique sibling staging path when present.
5. Re-inventory the moved preimage and refuse if it no longer matches the approved evidence.
6. Create the symlink directly at the now-absent destination; creation fails instead of overwriting an entry recreated concurrently.
7. Verify lexical target, canonical target, and source digest.
8. Remove the staged preimage only after successful verification.
9. Mark the destination action successful in the receipt.

If steps 4–8 fail, Executor restores that destination's preimage when the final path remains available. If another actor recreated the final path, Executor preserves that external state, retains the independent backup, reports rollback as incomplete, and never overwrites the new entry. Earlier verified destinations remain installed. The run is sealed incomplete and reports exactly which destination failed.

Rootline uses `os.Symlink`; it never shells out to `ln`, `cp`, or platform checksum commands.

#### ReceiptStore

ReceiptStore uses the native per-user state directory under a `rootline/skill` namespace. On Unix it uses `XDG_STATE_HOME` when set and otherwise `$HOME/.local/state`. On Windows it uses `os.UserConfigDir`. Tests inject a state root and never depend on those production defaults.

```text
rootline/skill/
├── receipts.jsonl
└── backups/
    └── <receipt-id>/
        ├── claude/
        └── agents/
```

Receipts are appended, never rewritten. Each line contains:

- version and kind;
- receipt ID and timestamp;
- operation and completion state;
- source path, commit, and digest;
- approved plan digest;
- per-destination preimage and final evidence;
- backup locations;
- errors and rollback results.

Backup paths are unique and created with exclusive semantics. A regular-directory preimage is copied only as recovery evidence; copied trees are never used as active installations. A symlink preimage is preserved as symlink metadata and target evidence.

#### Verifier

Verifier checks all of the following:

- the destination is lexically a symlink;
- the lexical target equals the intended source path;
- canonical resolution remains inside and equals the canonical skill source;
- the resolved tree digest equals the planned source digest;
- the receipt evidence matches the installed state.

## Operation Flows

### Install

`install` plans one action per supported destination. An absent destination requires no backup. Any existing directory or divergent symlink is backed up before replacement. A correct symlink is a no-op, making repeated installs idempotent.

Installing from a different primary checkout is a replacement, not a silent retarget. Its plan digest changes and requires new approval.

### Status

`status` is read-only. It reports source validity, destination classifications, convergence, `receipt_drift` from the latest receipt, and available restoration evidence. A committed source update may leave both symlinks converged while setting `receipt_drift: true`; approving the resulting idempotent install plan records the new evidence. Status never creates the state directory merely to report that no receipt exists.

### Uninstall

`uninstall` removes only symlinks that still match a Rootline receipt and the observed approved source. A directory, changed symlink, or unreceipted path is a conflict. The command emits a plan first and requires digest-bound approval.

Uninstall does not automatically restore a preinstallation directory; restoration remains a separate explicit verb.

### Restore

`restore` selects one receipt and plans restoration of its recorded preimages. It validates backup existence and digest before approval. If a current destination differs from the state described by the receipt, the conflict is part of the plan and therefore changes the required approval digest. Restore never overwrites unobserved state.

## Output Contracts

Each verb emits one versioned envelope. Install uses this shape; sibling verbs use their own stable kind and relevant additive fields:

```json
{
  "version": 1,
  "kind": "rootline/skill-install",
  "complete": false,
  "source": {
    "path": "/stable/rootline/.claude/skills/rootline",
    "commit": "<commit>",
    "digest": "sha256:<digest>"
  },
  "plan_digest": "sha256:<digest>",
  "receipt_drift": false,
  "destinations": [],
  "backups": [],
  "receipt": null,
  "errors": []
}
```

The payload is always written before a non-zero exit. `complete` means the accepted operation carried through every planned action. A plan-only run is not an attempted operation and therefore has `complete: false` with no error.

Stable error codes include:

- `source_not_canonical`;
- `linked_worktree_refused`;
- `source_digest_changed`;
- `preimage_digest_changed`;
- `unsupported_file_type`;
- `symlink_permission_denied`;
- `backup_failed`;
- `verification_failed`;
- `restore_conflict`.

Error messages add remediation but are not machine-parsed.

## Platform Behavior

The contract requires a real directory symlink on every platform. On Windows, `os.Symlink` may require Developer Mode or an appropriate token privilege. If the platform denies symlink creation, Rootline emits `symlink_permission_denied`, restores the affected preimage, and explains how to enable the capability. It does not create a junction or copied directory as an implicit alternative.

Tests exercise successful symlink behavior where the runner grants it and the precise fail-closed path where permission is unavailable. Both outcomes must leave no active copied installation.

## Breaking-Change Documentation

`CHANGELOG.md` receives a breaking entry that states:

1. OpenCode and Pi consume `~/.agents/skills/rootline`.
2. The former OpenCode-specific destination is unsupported and has no compatibility handling in Rootline.
3. An agent upgrading an affected environment must inspect the obsolete path, preserve any desired preimage externally, retire it deliberately, run `rootline skill install` from the stable canonical checkout, and verify with `rootline skill status`.
4. Rootline will not automatically discover or mutate the obsolete path.

`docs/skill.md` documents install, update, status, uninstall, restore, backup ownership, receipts, approval, and platform requirements. `README.md` links to that guide.

The canonical `.claude/skills/rootline/SKILL.md` tells agents to read the relevant breaking entry in `CHANGELOG.md` before correcting an installation created under an older contract. This instruction does not add legacy-path logic to the CLI.

## Hook and ADR Changes

`.githooks/pre-push` loses the block that deletes and copies the Claude skill. It retains its existing validation and coverage gates.

`TestPrePushSyncsSkillWithoutInstallingBinary` is replaced with a negative contract test: executing the hook with a temporary home must not create, remove, or modify any path under that home.

A successor to ADR 0003 preserves these decisions:

- no `post-merge` hook;
- no automatic binary installation;
- no documentation mutation from merge hooks;
- explicit installation of checkout-owned artifacts.

It supersedes the former consequence that allowed `pre-push` to synchronize the skill. Skill distribution becomes explicit through `rootline skill`.

## Testing Strategy

### Unit tests

`internal/skilldist` tests cover:

- deterministic tree digests independent of directory enumeration order;
- source validation and linked-worktree refusal;
- every supported inventory classification;
- plan digest changes for source, preimage, or action changes;
- idempotent no-op planning;
- backup-before-replacement ordering;
- per-destination rollback;
- exclusive backup creation;
- append-only receipt behavior;
- uninstall and restore drift refusal;
- verifier checks for lexical and canonical targets;
- an obsolete OpenCode-path sentinel remains byte-identical throughout every operation, proving production code does not discover or mutate it.

### CLI and integration tests

Temporary repositories, homes, and state directories verify:

1. install without approval emits a plan and performs no mutation;
2. matching approval creates both symlinks;
3. repeated install is a no-op;
4. source or preimage drift invalidates approval;
5. status reports convergence and drift;
6. uninstall removes only intact receipted links;
7. restore reproduces exact preimages;
8. unsupported types and symlink permission failures preserve prior state;
9. partial execution records complete evidence and rolls back the failing destination;
10. pre-push never mutates the temporary home;
11. envelopes, exit codes, output format validation, and `--field` follow Rootline's global contracts.

Tests never mutate the real user home. The existing CI matrix provides platform execution. Local verification runs:

```text
just test
just check
just coverage-check
rootline validate --all docs/adr/
lens diagnostics for edited files
```

## Delivery and Operational Acceptance

Delivery is split into two verifiable gates.

### Product gate

The pull request contains code, tests, documentation, changelog, and the accepted successor ADR. It references issue #210 but does not auto-close it. The pull request must pass the full repository quality gates.

### Observed-environment gate

After merge, the primary stable checkout is updated to the merged commit. From that checkout:

1. run `rootline skill install` against the real home without approval;
2. retain the source commit/digest and exact supported preimage evidence;
3. review the plan;
4. apply the exact plan digest;
5. verify Claude, OpenCode, and Pi discovery plus lexical and canonical paths;
6. retain the receipt and demonstrate that restoration evidence is readable.

Only after this operational verification is issue #210 closed. The implementation worktree is never a symlink target.

## Security and Maintainability Rationale

Digest-bound plans prevent approval replay after source or destination drift. Exclusive backups and append-only receipts make destructive replacement attributable and recoverable. Symlink-only installation structurally prevents future content divergence.

The design deliberately avoids a generic manifest engine, shell-command delegation, compatibility branches, copy fallbacks, and force flags. Those alternatives add state or exceptions without improving the two verified supported destinations. The result is a small lifecycle service with a narrow public contract and no remembered legacy behavior.
