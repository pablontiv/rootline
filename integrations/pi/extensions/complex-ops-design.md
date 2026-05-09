# Complex Operations UX Design for Rootline Pi Extension

**Status:** Design Document  
**Outcome:** [O07 Expose complex operations with guardrails](../../docs/roadmap/O07-expose-complex-operations-with-guardrails/)  
**Date:** 2026-05-09

## Overview

This document specifies the UX design for complex Rootline operations exposed through the Pi extension. Complex operations are commands that affect multiple files, mutate schema, or propose structural changes. Each operation follows a consistent guardrail pattern: **intent → preview → validation → confirmation → rollback guidance**.

## Shared Guardrail Principles

All complex operations must:

1. **Explicit User Intent**: Cannot be triggered silently or by default. Require:
   - Explicit command invocation with clear parameters
   - No auto-execution from context guidance alone
   - Confirmation step in non-interactive mode (flag: `confirmed: true`)

2. **Preview-Before-Apply**: Provide a `--dry-run` mode that:
   - Shows exact changes without writing files
   - Includes file counts, field changes, and modification summary
   - Returns same output structure as apply (diffable)

3. **Post-Operation Validation**: After writing files:
   - Run `rootline validate` on affected scope
   - Report validation errors or warnings
   - Block successful completion if validation fails (except warnings)

4. **Git-Based Rollback**: Always require:
   - User to commit before complex operation (guidance)
   - `git diff` inspection of results before final commit
   - Simple revert via `git reset --hard HEAD` if needed

5. **Scoped Operations**: Restrict to explicit paths:
   - No implicit directory scanning
   - Reject operations on root directory unless explicitly specified
   - Clear error messages for out-of-scope paths

6. **Audit Trail**: JSON output includes:
   - Version, kind, timestamp
   - File-by-file results with change counts
   - Summary statistics
   - Schema suggestions (if applicable)

---

## Operation 1: rootline-fix (Bulk field correction)

**Purpose:** Auto-repair validation errors across multiple documents in a directory.

**Command surface:** `rootline fix --all <path> [--dry-run]`

### User Intent

User explicitly invokes:
```bash
rootline fix --all docs/roadmap/ --dry-run
# Review output
rootline fix --all docs/roadmap/  # Apply
```

**Pi Tool Spec:**
- Tool name: `rootline-fix`
- Must be called explicitly with `path` parameter
- Optional: `dry_run` (boolean, default false)
- Optional: `confirmed` (boolean, default false) for non-interactive mode
- Required for apply: `confirmed: true` in non-interactive sessions

**Activation Rules:**
- Cannot be triggered from agent context guidance alone
- Requires explicit user utterance mentioning "fix" + directory
- In interactive mode: Pi asks for confirmation after showing preview
- In non-interactive mode: requires `confirmed: true` flag

### Preview (`--dry-run`)

Command:
```bash
rootline fix --all <path> --dry-run --output json
```

Output format (JSON):
```json
{
  "version": 1,
  "kind": "rootline/fix-preview",
  "path": "docs/roadmap/",
  "dry_run": true,
  "results": [
    {
      "path": "docs/roadmap/O06/T001.md",
      "fixed": true,
      "fields_added": ["dependency"],
      "values_corrected": [{"field": "estado", "old": "Completd", "new": "Completed"}],
      "changes": [
        "Added required field: dependency",
        "Fixed enum typo: estado (Completd → Completed)"
      ]
    }
  ],
  "summary": {
    "total": 42,
    "fixed": 5,
    "skipped": 37,
    "fields_added": 7,
    "values_corrected": 3
  }
}
```

**Preview expectations:**
- Show file count and change summary upfront
- List each file with specific field changes
- Display before/after values for corrections
- Include why files were skipped (already valid, no applicable fixes)

### Validation

Post-fix workflow:
```bash
# 1. Apply fixes
rootline fix --all docs/roadmap/

# 2. Automatically validate result
rootline validate --all docs/roadmap/ --output json

# 3. If validation passes:
#    - Report shows "validation_passed": true
#    - Exit code 0
# 4. If validation has errors:
#    - Report shows validation errors with file paths
#    - Pi suggests: "Some files still have errors. Review and adjust."
```

**Validation report attachment:**
- Include `validation_summary` in fix output:
  ```json
  {
    "validation_summary": {
      "valid": true,
      "total_files": 42,
      "valid_count": 42,
      "error_count": 0,
      "warning_count": 0
    }
  }
  ```

### Rollback

**Pre-operation checkpoint:**
```bash
# 1. Before running fix, user must:
cd <project-root>
git add -A && git commit -m "chore: checkpoint before rootline fix"

# 2. Apply fix
rootline fix --all docs/roadmap/

# 3. Review changes
git diff HEAD

# 4. If satisfied:
git add -A && git commit -m "fix(docs): auto-corrected validation errors"

# 5. If not satisfied:
git reset --hard HEAD~1
```

**Rollback guidance in Pi:**
```
If the fixes are not what you expected:
  git reset --hard HEAD~1
Then review what went wrong and try again.
```

### Tool Implementation

**Parameters:**
```typescript
interface RootlineFixOptions {
  path: string;           // Directory to fix (required)
  dry_run?: boolean;      // Preview only, don't write (default: false)
  confirmed?: boolean;    // Explicit confirmation in non-interactive mode
}
```

**Guardrails (TypeScript extension):**
1. Reject if `path` outside project root (use `validateTargetPath`)
2. In non-interactive mode: require `confirmed: true`
3. Run `validate` on affected directory after apply
4. Return validation result in output

---

## Operation 2: rootline-migrate (Schema migration and bulk changes)

**Purpose:** Rename fields, migrate values, or restructure `.stem` files across directories.

**Command surface:** `rootline migrate [path] [--rename old=new | --split | --scaffold] [--dry-run]`

### User Intent

User explicitly specifies the migration operation:

```bash
# Option A: Explicit field rename
rootline migrate docs/roadmap/ --rename old_field=new_field --dry-run
rootline migrate docs/roadmap/ --rename old_field=new_field

# Option B: Split flat .stem to hierarchical
rootline migrate docs/roadmap/ --split --dry-run
rootline migrate docs/roadmap/ --split

# Option C: Scaffold missing fields in documents
rootline migrate docs/roadmap/ --scaffold --dry-run
rootline migrate docs/roadmap/ --scaffold
```

**Pi Tool Spec:**
- Tool name: `rootline-migrate`
- Required: `path`
- Exactly one of: `rename`, `split`, or `scaffold` (mutually exclusive)
- Optional: `dry_run` (boolean, default false)
- Optional: `confirmed` (boolean, default false) for non-interactive mode

**Activation Rules:**
- Cannot be triggered from agent context guidance alone
- Requires explicit operation selection (rename field name, split instruction, etc.)
- In interactive mode: Pi confirms the migration strategy before proceeding
- In non-interactive mode: requires `confirmed: true` flag

### Preview (`--dry-run`)

Command:
```bash
rootline migrate docs/roadmap/ --rename foo=bar --dry-run --output json
```

Output format (JSON):
```json
{
  "version": 1,
  "kind": "rootline/migrate-preview",
  "path": "docs/roadmap/",
  "operation": "rename",
  "rename": {"old": "foo", "new": "bar"},
  "dry_run": true,
  "results": [
    {
      "path": "docs/roadmap/.stem",
      "type": "stem_file",
      "changes": ["Field definition renamed: foo → bar"]
    },
    {
      "path": "docs/roadmap/O06/T001.md",
      "type": "document",
      "changes": ["Frontmatter field renamed: foo → bar"]
    }
  ],
  "summary": {
    "total_files": 42,
    "stem_files_modified": 1,
    "documents_modified": 8,
    "breaking_changes": []
  },
  "breaking_changes_detected": false
}
```

**Breaking changes detection:**
- Type widening (int → string)
- Enum values removed
- Required field made optional (data loss risk)
- Enum values changed or removed

### Validation

Post-migrate workflow:
```bash
# 1. Apply migration
rootline migrate docs/roadmap/ --rename foo=bar

# 2. Validate result
rootline validate --all docs/roadmap/ --output json

# 3. Report includes validation state
```

**Validation in output:**
```json
{
  "validation_summary": {
    "valid": true,
    "total_files": 42,
    "breaking_changes_found": 0,
    "schema_conflicts": 0
  },
  "migration_log": {
    "timestamp": "2026-05-09T10:00:00Z",
    "operation": "rename",
    "old_field": "foo",
    "new_field": "bar",
    "affected_files": ["docs/roadmap/.stem", "docs/roadmap/O06/T001.md", ...],
    "rollback_steps": [
      "git reset --hard HEAD~1",
      "OR manually rename bar → foo in .stem and documents"
    ]
  }
}
```

### Rollback

**Pre-operation checkpoint:**
```bash
# 1. Before migration:
cd <project-root>
git add -A && git commit -m "chore: checkpoint before rootline migrate"

# 2. Apply migration
rootline migrate docs/roadmap/ --rename foo=bar

# 3. Review changes
git diff HEAD
rootline validate --all docs/roadmap/

# 4. If satisfied:
git add -A && git commit -m "refactor(schema): migrate foo → bar"

# 5. If not satisfied:
git reset --hard HEAD~1
# Files revert to original state; any unsaved changes in other files are lost
```

**Rollback guidance in Pi:**
```
If the migration result is not correct:
  git reset --hard HEAD~1
to revert all changes and start over.
```

### Tool Implementation

**Parameters:**
```typescript
interface RootlineMigrateOptions {
  path: string;          // Directory to migrate (required)
  rename?: string;       // "old_field=new_field" (mutually exclusive with split, scaffold)
  split?: boolean;       // Split flat to hierarchical (mutually exclusive with rename, scaffold)
  scaffold?: boolean;    // Scaffold missing fields (mutually exclusive with rename, split)
  dry_run?: boolean;     // Preview only (default: false)
  confirmed?: boolean;   // Explicit confirmation in non-interactive mode
}
```

**Guardrails (TypeScript extension):**
1. Validate `path` is within project root
2. Enforce mutual exclusion: exactly one of `rename`, `split`, `scaffold`
3. Parse `rename` format: `old=new` (reject invalid format)
4. In non-interactive mode: require `confirmed: true`
5. After apply: validate and report breaking changes
6. Return migration log with rollback steps

---

## Operation 3: rootline-analyze + Schema Propose + Schema Apply (Inference and Schema Evolution)

**Purpose:** Analyze documents, infer schema patterns, generate schema proposals, and optionally apply them to `.stem` files.

**Command surfaces:**
- Read-only: `rootline analyze <path> [--incremental]`
- Read-only: `rootline schema propose <path> --report <file> [--incremental]`
- Mutating: `rootline schema apply --report <proposals.json> [--dry-run]`

### User Intent

This is a three-step workflow where the first two are read-only (no confirmation needed), and the third requires explicit approval:

```bash
# Step 1: Analyze documents (read-only)
rootline analyze docs/roadmap/ --output json > analyze-report.json

# Step 2: Review analysis, then generate schema proposals (read-only)
rootline schema propose docs/roadmap/ --report analyze-report.json \
  --incremental --output json > schema-proposals.json

# Step 3: Review proposals, then explicitly apply schema changes
rootline schema apply --report schema-proposals.json --dry-run
# Review the preview
rootline schema apply --report schema-proposals.json  # Apply
```

**Pi Tool Spec (3 separate tools):**

1. **rootline-analyze** (read-only, no confirmation)
   - Parameters: `path`, `incremental` (optional)
   - Returns: analysis report with inferred fields, enums, coverage gaps

2. **rootline-schema-propose** (read-only, no confirmation)
   - Parameters: `analyze_report` (JSON), `path`, `incremental` (optional)
   - Returns: schema proposals report (JSON) with proposals and `requires_agent` markers

3. **rootline-schema-apply** (mutating, requires confirmation)
   - Parameters: `proposals_report` (JSON), `dry_run` (optional), `confirmed` (optional)
   - Returns: schema-apply result with modified `.stem` files and validation state

### Preview

**analyze (read-only, informational):**
```bash
rootline analyze docs/roadmap/ --output json
```

Output:
```json
{
  "version": 1,
  "kind": "rootline/analyze-report",
  "path": "docs/roadmap/",
  "categories": [
    {
      "category": "data_inference",
      "inferences": [
        {
          "type": "field_type_inferred",
          "source": "docs/roadmap/O06/",
          "field": "estado",
          "inferred_type": "string",
          "confidence": 0.95,
          "enum_values": ["Pending", "In Progress", "Completed"],
          "requires_agent": false
        }
      ]
    },
    {
      "category": "governance",
      "inferences": [
        {
          "type": "missing_schema",
          "source": "docs/roadmap/O08/",
          "message": "Directory has no .stem file",
          "requires_agent": true,
          "recommended_fields": ["estado", "tipo", "priority"]
        }
      ]
    }
  ]
}
```

**schema propose (read-only, proposes changes):**
```bash
rootline schema propose docs/roadmap/ --report analyze-report.json --incremental --output json
```

Output:
```json
{
  "version": 1,
  "kind": "rootline/schema-proposals",
  "path": "docs/roadmap/",
  "incremental": true,
  "proposals": [
    {
      "type": "extend_enum",
      "surface": "schema",
      "source": "docs/roadmap/O06/.stem",
      "field": "estado",
      "reason": "Documents contain value 'On Hold' not in enum",
      "action": "Add 'On Hold' to estado enum values",
      "requires_agent": false
    },
    {
      "type": "create_stem",
      "surface": "schema",
      "source": "docs/roadmap/O08/",
      "reason": "Directory has documents but no .stem file",
      "action": "Scaffold .stem with inferred fields",
      "requires_agent": true,
      "recommended_fields": ["estado", "tipo"]
    }
  ],
  "summary": {
    "total_proposals": 3,
    "requires_agent_count": 1,
    "auto_applicable_count": 2
  }
}
```

**schema apply with `--dry-run`:**
```bash
rootline schema apply --report schema-proposals.json --dry-run --output json
```

Output:
```json
{
  "version": 1,
  "kind": "rootline/schema-apply",
  "dry_run": true,
  "path": "docs/roadmap/",
  "results": [
    {
      "proposal_type": "extend_enum",
      "source": "docs/roadmap/O06/.stem",
      "field": "estado",
      "action": "Add 'On Hold' to enum",
      "status": "would_apply",
      "affected_stem_file": "docs/roadmap/O06/.stem"
    }
  ],
  "summary": {
    "total_proposals_in_report": 3,
    "applied_count": 2,
    "skipped_count": 1,
    "requires_agent_count": 1
  },
  "validation": {
    "valid": true,
    "schema_conflicts": 0,
    "stem_health_warnings": []
  }
}
```

### Validation

Post-schema-apply workflow:
```bash
# 1. After applying proposals
rootline schema apply --report schema-proposals.json

# 2. Validate updated schema
rootline validate --all docs/roadmap/ --output json

# 3. Fix any data issues if schema changes require document updates
rootline fix --all docs/roadmap/ --dry-run
# Review and apply if needed
rootline fix --all docs/roadmap/
```

**Validation in schema-apply output:**
```json
{
  "validation": {
    "valid": true,
    "total_files_validated": 42,
    "error_count": 0,
    "warning_count": 0,
    "stem_health": {
      "yaml_valid": true,
      "scope_match": true,
      "type_consistency": true
    }
  },
  "next_steps": [
    "Review schema changes: git diff docs/roadmap/.stem",
    "If data needs updating: rootline fix --all docs/roadmap/",
    "Commit: git add -A && git commit -m 'chore(schema): apply inferred proposals'"
  ]
}
```

### Rollback

**Workflow:**
```bash
# 1. Pre-checkpoint
cd <project-root>
git add -A && git commit -m "chore: checkpoint before schema evolution"

# 2. Analyze → Propose (read-only, safe)
rootline analyze docs/roadmap/ > analyze-report.json
rootline schema propose docs/roadmap/ --report analyze-report.json > proposals.json

# 3. Review proposals in proposals.json
# Check which proposals have requires_agent: true

# 4. Apply (with confirmation in interactive mode)
rootline schema apply --report proposals.json --dry-run
# Review preview carefully

rootline schema apply --report proposals.json
# Apply schema changes

# 5. Validate and optionally fix data
rootline validate --all docs/roadmap/
rootline fix --all docs/roadmap/ --dry-run
rootline fix --all docs/roadmap/

# 6. Commit
git add -A && git commit -m "chore(schema): apply inferred proposals"

# 7. If schema changes are wrong:
git reset --hard HEAD~2  # Go back before schema apply
# (or HEAD~3 if you also fixed data)
```

**Rollback guidance in Pi:**
```
If the schema changes are not correct:
  git diff HEAD~1  # Compare before/after
  git reset --hard HEAD~1  # Revert schema changes
Then review what went wrong and try a different approach.
```

### Tool Implementation

**Parameters:**

```typescript
// rootline-analyze (read-only)
interface RootlineAnalyzeOptions {
  path: string;
  incremental?: boolean;  // Only report inferences not covered by .stem
}

// rootline-schema-propose (read-only)
interface RootlineSchemaProposesOptions {
  path: string;
  analyze_report: AnalyzeReport;  // JSON from rootline-analyze
  incremental?: boolean;
}

// rootline-schema-apply (mutating)
interface RootlineSchemaApplyOptions {
  proposals_report: SchemaProposalsReport;  // JSON from rootline-schema-propose
  dry_run?: boolean;
  confirmed?: boolean;  // Required in non-interactive mode
}
```

**Guardrails (TypeScript extension):**

For `rootline-analyze` and `rootline-schema-propose`:
- No confirmation needed (read-only operations)
- Validate `path` is within project root
- Return structured reports for inspection

For `rootline-schema-apply`:
1. Validate `proposals_report` is valid JSON with kind "rootline/schema-proposals"
2. Reject proposals with `requires_agent: true` (user must review first)
3. In non-interactive mode: require `confirmed: true`
4. After apply: run validation, report validation state
5. Suggest next steps (commit, fix data, etc.)

---

## Operation 4: rootline-repair-apply (Data-only repair from proposals)

**Purpose:** Apply data-repair proposals to document frontmatter without modifying `.stem` files.

**Command surface:** `rootline repair apply --report <proposals.json> [--dry-run]`

### User Intent

User explicitly passes a repair proposals report:

```bash
# Step 1: Generate repair proposals from analyze (read-only)
rootline analyze docs/roadmap/ --output json > analyze-report.json

# Step 2: Review analyze report for data repair suggestions

# Step 3: Apply repairs with explicit report file
rootline repair apply --report analyze-report.json --dry-run
# Review changes

rootline repair apply --report analyze-report.json
# Apply repairs
```

**Pi Tool Spec:**
- Tool name: `rootline-repair-apply`
- Required: `report` (file path or inline JSON)
- Optional: `dry_run` (boolean, default false)
- Optional: `confirmed` (boolean, default false) for non-interactive mode

**Activation Rules:**
- Cannot trigger without explicit `--report` argument
- Requires explicit report file passed by user
- In non-interactive mode: requires `confirmed: true` flag
- Pi must show preview before applying in interactive mode

### Preview (`--dry-run`)

Command:
```bash
rootline repair apply --report analyze-report.json --dry-run --output json
```

Output format (JSON):
```json
{
  "version": 1,
  "kind": "rootline/repair-apply",
  "dry_run": true,
  "report_source": "analyze-report.json",
  "path": "docs/roadmap/",
  "results": [
    {
      "path": "docs/roadmap/O06/T001.md",
      "status": "would_repair",
      "changes": [
        {
          "field": "estado",
          "old_value": "Completd",
          "new_value": "Completed",
          "reason": "Enum correction (typo)"
        },
        {
          "field": "priority",
          "old_value": null,
          "new_value": "medium",
          "reason": "Add missing required field (default value)"
        }
      ]
    }
  ],
  "summary": {
    "total_files": 42,
    "would_repair": 5,
    "skipped": 37,
    "total_fields_changed": 8,
    "fields_added": 3,
    "values_corrected": 5
  },
  "schema_surface_count": 0,
  "repair_surface_count": 8
}
```

**Preview expectations:**
- Show file count and change summary
- List each file with field-by-field changes
- Include reason for each change (enum correction, required field, type migration, etc.)
- Separate repair-surface changes from schema-surface (which are skipped)

### Validation

Post-repair-apply workflow:
```bash
# 1. Apply repairs
rootline repair apply --report analyze-report.json

# 2. Validate result
rootline validate --all docs/roadmap/ --output json

# 3. If validation passes:
#    - Report shows "validation_passed": true
# 4. If validation fails:
#    - Report shows remaining errors
#    - Pi suggests: "Some files still have errors. Review manually."
```

**Validation in output:**
```json
{
  "validation_summary": {
    "valid": true,
    "total_files": 42,
    "error_count": 0,
    "warning_count": 0
  }
}
```

### Rollback

**Workflow:**
```bash
# 1. Pre-checkpoint
cd <project-root>
git add -A && git commit -m "chore: checkpoint before rootline repair apply"

# 2. Generate report from analyze (read-only)
rootline analyze docs/roadmap/ > analyze-report.json

# 3. Apply repairs
rootline repair apply --report analyze-report.json --dry-run
# Review preview

rootline repair apply --report analyze-report.json
# Apply

# 4. Validate result
rootline validate --all docs/roadmap/

# 5. Commit or revert
git diff HEAD
git add -A && git commit -m "fix(docs): apply data repairs from analysis"

# OR if not satisfied:
git reset --hard HEAD~1
```

**Rollback guidance in Pi:**
```
If the repairs are not correct:
  git reset --hard HEAD~1
to revert and try again, or fix individual files manually.
```

### Tool Implementation

**Parameters:**
```typescript
interface RootlineRepairApplyOptions {
  report: string | object;  // File path or inline JSON from analyze/repair
  dry_run?: boolean;
  confirmed?: boolean;      // Required in non-interactive mode
}
```

**Guardrails (TypeScript extension):**
1. Validate `report` is valid JSON with kind "rootline/analyze-report" or "rootline/repair-proposals"
2. Extract only repair-surface proposals (reject schema-surface)
3. In non-interactive mode: require `confirmed: true`
4. After apply: validate affected files
5. Return validation state in output

---

## Implementation Notes for T002-T004

### File Structure

Each tool implementation should:
1. **Create extension file**: `extensions/rootline-<operation>.ts`
2. **Define guardrail functions**: Use existing `guardrails.ts` as base (validateTargetPath, requiresConfirmation, validateAfterWrite)
3. **Call rootline CLI**: Use existing `runner.ts` to execute commands and parse JSON output
4. **Register tool**: Add to `package.json` extensions list

### Tool Registration Pattern

```typescript
// extensions/rootline-fix.ts
export default function (pi: ExtensionAPI) {
  pi.tool("rootline-fix", {
    description: "Bulk fix validation errors across multiple files",
    parameters: {
      path: { type: "string", description: "Directory to fix" },
      dry_run: { type: "boolean", description: "Preview without applying" },
      confirmed: { type: "boolean", description: "Explicit confirmation" }
    },
    execute: async (params) => {
      // Implementation
    }
  });
}
```

### Validation Pattern

After each mutation, always run:
```typescript
const validateResult = await run<ValidateReport>(
  ["validate", affectedPath, "--output", "json"],
  { cwd }
);
// Check validateResult.data.summary
```

### Error Handling

Catch and report:
- Path validation failures (outside root, traversal attempts)
- Missing report files (for apply operations)
- Invalid JSON in reports
- Rootline CLI errors (capture stderr)
- Post-validation failures (validation errors in result)

### Testing

Each tool should have:
- Unit tests for guardrails (path validation, confirmation logic)
- Integration tests with mock rootline CLI responses
- E2E tests with sample documents and `.stem` files

See existing tests: `extensions/*.test.ts`

---

## Acceptance Criteria Mapping

1. ✓ Design covers analyze, fix, migrate, and replacement schema/repair apply surfaces
   - Section: Operations 1-4 above
   - Legacy `apply` is deprecated (see `cmd/rootline/apply.go` deprecation warning)

2. ✓ Each operation has user-intent, preview, validation, and rollback notes
   - User Intent: How Pi tool is triggered
   - Preview: `--dry-run` output examples
   - Validation: Post-apply validation workflow
   - Rollback: Git-based checkpoint and reset procedures

3. ✓ `rootline validate --all docs/roadmap/` returns exit 0
   - Verified in Pi README (section "Local Verification")
   - Baseline: 79 records, 0 errors, 2 schema-health warnings

---

## Summary

This design ensures that complex operations exposed through the Pi extension:
- **Never execute silently**: Require explicit user intent
- **Always show preview**: Dry-run before applying
- **Always validate**: Post-apply validation catches errors
- **Always enable rollback**: Git-based checkpoint/reset workflow
- **Maintain audit trail**: JSON output with version, kind, file-by-file results

Implementors (T002-T004) should follow these patterns to create consistent, safe complex operation tools.
