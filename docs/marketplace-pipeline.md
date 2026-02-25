---
estado: Pending
---
# Marketplace Distribution Pipeline

Automated pipeline that syncs skills from the rootline repo to [pablontiv/agent-marketplace](https://github.com/pablontiv/agent-marketplace).

## Architecture

```
rootline repo                        agent-marketplace repo
┌─────────────────┐                  ┌─────────────────────┐
│ .claude/skills/  │                  │ skills/rootline-*/   │
│  ├── roadmap/    │   publish-       │  ├── rootline-roadmap│
│  ├── validate/   │──marketplace──►  │  ├── rootline-validate│
│  ├── describe/   │   workflow       │  ├── rootline-describe│
│  └── new-doc/    │                  │  └── rootline-new-doc│
└─────────────────┘                  └─────────────────────┘

Trigger Chain:
  push to master (paths: .claude/**)
    → CI passes
    → publish-marketplace.yml runs
    → hash comparison (skip if unchanged)
    → rsync skills with rootline- prefix
    → validate marketplace structure
    → update version from git tag
    → commit & push to agent-marketplace
```

## PAT Setup

The workflow needs a Personal Access Token (PAT) to push to the agent-marketplace repo.

### Step 1: Create a Fine-Grained PAT

1. Go to [GitHub Settings > Developer settings > Personal access tokens > Fine-grained tokens](https://github.com/settings/personal-access-tokens/new)
2. Set:
   - **Token name**: `marketplace-sync`
   - **Expiration**: 90 days (recommended) or custom
   - **Repository access**: Select `pablontiv/agent-marketplace` only
   - **Permissions**: Contents → Read and write
3. Click **Generate token** and copy the value

### Step 2: Add as Repository Secret

1. Go to [rootline repo Settings > Secrets and variables > Actions](https://github.com/pablontiv/rootline/settings/secrets/actions)
2. Click **New repository secret**
3. Set:
   - **Name**: `MARKETPLACE_TOKEN`
   - **Value**: paste the PAT from Step 1
4. Click **Add secret**

### Step 3: Verify

Push a change to any file under `.claude/` and check the Actions tab for the "Publish Marketplace" workflow run.

## Manual Re-Sync

To force a sync even when no changes are detected:

1. Go to [Actions > Publish Marketplace](https://github.com/pablontiv/rootline/actions/workflows/publish-marketplace.yml)
2. Click **Run workflow**
3. Select branch: `master`
4. Click **Run workflow**

The `workflow_dispatch` trigger bypasses the hash comparison and forces a full sync.

## Troubleshooting

### Token Expired

**Symptom**: Workflow fails at "Checkout marketplace" step with `authentication failed`.

**Fix**: Create a new PAT (Step 1) and update the `MARKETPLACE_TOKEN` secret (Step 2).

### Incorrect Token Scope

**Symptom**: Workflow fails at "Commit and push" with `permission denied` or `403`.

**Fix**: Ensure the PAT has `Contents: Read and write` permission specifically for `pablontiv/agent-marketplace`.

### Secret Not Found

**Symptom**: Workflow fails with `Error: Input required and not supplied: token`.

**Fix**: Verify the secret is named exactly `MARKETPLACE_TOKEN` (case-sensitive) in the rootline repo settings.

### No Changes Detected (False Negative)

**Symptom**: Skills changed but workflow reports "No changes detected, skipping push".

**Fix**: Use manual re-sync via `workflow_dispatch` (see above). The hash comparison uses file content hashes — if files were reformatted without content changes, the hash may match.

### Marketplace Validation Fails

**Symptom**: Workflow fails at "Validate marketplace structure" with error annotations.

**Fix**: Check the error messages:
- `marketplace.json is not valid JSON` → Fix JSON syntax in `.claude-plugin/marketplace.json` in the marketplace repo
- `Missing SKILL.md in skills/rootline-X` → Ensure skill directory has a SKILL.md file
- `missing 'name'/'description' in frontmatter` → Add required fields to the SKILL.md frontmatter
