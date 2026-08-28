# Rootline Skill Distribution

Rootline distributes the canonical agent skill from a primary Rootline checkout to the active user destinations with an explicit plan-and-approve workflow. The Git hooks do not install, update, warn about, or repair skills.

## Destinations

`rootline skill install` manages exactly these destinations:

- Claude: `~/.claude/skills/rootline`
- OpenCode and Pi: `~/.agents/skills/rootline`

Both destinations are symlinks to the canonical source tree at `<source>/.claude/skills/rootline`. Rootline does not scan other runtime paths.

## Source checkout requirement

Use the primary, stable Rootline checkout as `--source`. The source must:

- be a Git worktree root with a real `.git/` directory, not a linked worktree metadata file;
- contain `.claude/skills/rootline/SKILL.md`;
- have no uncommitted changes under `.claude/skills/rootline`;
- resolve to a non-empty `HEAD` commit.

A linked worktree is refused with `linked_worktree_refused` so agents do not publish a transient development checkout as a user-level skill.

## Install and verify

Preview first, then approve the exact `plan_digest` emitted by the preview. `--field` returns a JSON value, so command substitutions intended for shell variables must pipe through `jq -r` to remove JSON string quotes.

```bash
plan=$(rootline skill install --source /stable/rootline --field plan_digest | jq -r .)
receipt_id=$(rootline skill install --source /stable/rootline --approve "$plan" --field receipt.id | jq -r .)
rootline skill status --source /stable/rootline
```

Example preview shape:

```json
{
  "version": 1,
  "kind": "rootline/skill-install",
  "complete": false,
  "source": {
    "repo_root": "/stable/rootline",
    "path": "/stable/rootline/.claude/skills/rootline",
    "commit": "abc123",
    "digest": "sha256:source"
  },
  "plan_digest": "sha256:plan",
  "destinations": [
    {"id": "claude", "path": "/home/agent/.claude/skills/rootline", "kind": "absent"},
    {"id": "agents", "path": "/home/agent/.agents/skills/rootline", "kind": "absent"}
  ],
  "backups": [],
  "receipt": null,
  "receipt_drift": false,
  "errors": []
}
```

Example applied result shape:

```json
{
  "version": 1,
  "kind": "rootline/skill-install",
  "complete": true,
  "plan_digest": "sha256:plan",
  "receipt": {
    "version": 1,
    "kind": "rootline/skill-receipt",
    "id": "0123456789abcdef",
    "operation": "install",
    "complete": true,
    "plan_digest": "sha256:plan"
  },
  "receipt_drift": false,
  "errors": []
}
```

`receipt_id` is the exact value emitted by `--field receipt.id`; save it if you may need a targeted restore.

## Uninstall

Uninstall is receipt-bound. It removes intact managed symlinks only when the latest complete install receipt still matches the current destinations.

```bash
uninstall_plan=$(rootline skill uninstall --field plan_digest | jq -r .)
rootline skill uninstall --approve "$uninstall_plan"
```

If `rootline skill status --source /stable/rootline` reports `receipt_drift: true` after a committed source update, do not uninstall immediately. First approve an idempotent install plan from the current primary checkout, then rerun uninstall planning:

```bash
plan=$(rootline skill install --source /stable/rootline --field plan_digest | jq -r .)
rootline skill install --source /stable/rootline --approve "$plan"
uninstall_plan=$(rootline skill uninstall --field plan_digest | jq -r .)
rootline skill uninstall --approve "$uninstall_plan"
```

This no-op reapproval refreshes receipt evidence for current symlinks and avoids treating a normal source digest change as an uninstall conflict.

## Restore

Restore uses an install receipt ID and replaces the managed symlink with the backed-up preimage recorded for each destination.

```bash
restore_plan=$(rootline skill restore --receipt "$receipt_id" --field plan_digest | jq -r .)
rootline skill restore --receipt "$receipt_id" --approve "$restore_plan"
```

Example restore preview shape:

```json
{
  "version": 1,
  "kind": "rootline/skill-restore",
  "complete": false,
  "plan_digest": "sha256:restore-plan",
  "receipt": {"id": "0123456789abcdef", "operation": "install", "complete": true},
  "receipt_drift": false,
  "errors": []
}
```

## State, receipts, and backups

Rootline stores skill distribution state under the user state root:

- `$XDG_STATE_HOME/rootline/skill` on non-Windows systems when `XDG_STATE_HOME` is set;
- otherwise `~/.local/state/rootline/skill` on non-Windows systems;
- `%USERPROFILE%` is not used for state on Windows; Rootline uses `os.UserConfigDir()` and stores receipts under `<UserConfigDir>\\rootline\\skill`.

`receipts.jsonl` is append-only JSON Lines evidence for install, uninstall, and restore operations. Backups live below `backups/<receipt-id>/` in the same state root. The user owns those backups; Rootline writes and verifies them, but agents must preserve the state directory if they want uninstall or restore to remain possible.

Receipts are best-effort operational evidence, not a transaction log with global rollback. A failed operation can write an incomplete receipt that records how far it got. Use `complete: true` receipts for automated uninstall or restore decisions, and inspect `errors[]`, `actions[]`, and `backups[]` when `complete` is false.

## Plan drift and safety errors

Approval digests bind the observed source, destination preimages, and operation. If the source tree or a destination changes between preview and approve, Rootline refuses the stale approval with `source_digest_changed`, `preimage_digest_changed`, or `restore_conflict`. Re-run the preview, inspect the new plan, and approve only the new digest.

Unsupported destination file types are not overwritten. Directories and divergent symlinks can be backed up and replaced by the managed symlink; regular files or other unsupported entries require manual remediation before retrying.

On Windows, creating symlinks can require Developer Mode or elevated privileges. If the OS denies symlink creation, Rootline reports `symlink_permission_denied`; fix the Windows permission model and rerun the same preview/approve workflow.

## Post-merge operational gate

For issue #210, after the skill-distribution change is merged into the primary checkout, an operator must run the install workflow from that primary checkout and verify `rootline skill status --source /stable/rootline` before depending on agents to consume the new contract. This is an operational gate, not hook behavior.
