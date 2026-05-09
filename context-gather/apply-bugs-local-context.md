# Code Context

## Files Retrieved
1. `cmd/rootline/apply.go` (lines 15-27, 30-117, 120-144) - CLI entry point; contains dry-run flag, scaffold pre-phase, target `.stem` selection, schema/data apply calls, and stdout rendering.
2. `internal/infer/apply.go` (lines 14-24, 27-110, 112-230, 334-457) - core apply API/result type, schema write path, data correction loops, and schema mutation helpers.
3. `internal/infer/scaffold.go` (lines 13-65) - `missing_schema` scaffold writer used by apply pre-phase.
4. `internal/infer/report.go` (lines 5-13, 23-34, 43-63, 74-90) - analyze/apply report input contract and versioned analyze result pattern.
5. `internal/rules/discovery.go` (lines 18-21, 35-69) - `WalkUp` ordering semantics used to choose a target `.stem`.
6. `internal/infer/schema_coverage.go` (lines 43-65) - existing closest-stem calculation; explicitly documents `entries[0]` root-most vs `entries[len-1]` closest.
7. `cmd/rootline/coverage_test.go` (lines 70-103) - only CLI apply coverage; uses `apply --dry-run` but does not assert no mutation or valid JSON.
8. `internal/infer/apply_test.go` (lines 233-255, 1005-1042) - dry-run coverage only for data corrections; scaffold tests assert real writes.
9. `internal/e2e/apply_test.go` (lines 87-112) - e2e dry-run coverage only for data corrections.
10. `.claude/skills/rootline/SKILL.md` (lines 23-28) - skill currently warns that `apply --dry-run` is unsafe.
11. `.claude/skills/rootline/ref-advanced.md` (lines 121-138) - apply reference currently documents dry-run as mutating and recommends report inspection.
12. `docs/roadmap/O07-expose-complex-operations-with-guardrails/T004-implement-analyze-apply-workflow.md` (lines 21-46) and `T005-document-complex-operation-risks.md` (lines 21-40) - roadmap follow-ups for approval, post-apply validation, rollback docs.

## Key Code

### Confirmed dry-run mutation paths
`cmd/rootline/apply.go` calls the scaffold pre-phase unconditionally before any dry-run check:

```go
// lines 55-65
if inf.Type != "missing_schema" || inf.RequiresAgent { continue }
if scaffoldErr := infer.ScaffoldSchema(inf.Source); scaffoldErr != nil {
  fmt.Fprintf(cmd.ErrOrStderr(), "scaffold %s: %v\n", inf.Source, scaffoldErr)
} else {
  fmt.Fprintf(cmd.OutOrStdout(), "scaffolded %s/.stem\n", inf.Source)
}
```

`cmd/rootline/apply.go` also calls schema apply without passing `applyDryRun`:

```go
// lines 91-103
schemaResult, err := infer.ApplySchemaInferences(stemPath, schemaInferences)
opts := infer.ApplyOptions{DryRun: applyDryRun, Root: root}
dataResult, err := infer.ApplyDataCorrections(dataInferences, opts)
```

`internal/infer/apply.go` schema apply has no dry-run option and writes directly:

```go
// lines 100-106
out, err := yaml.Marshal(&doc)
if err := os.WriteFile(stemPath, out, 0o644); err != nil { ... }
```

`internal/infer/scaffold.go` always writes `.stem`:

```go
// lines 58-65
b.WriteString("version: 2\nschema:\n")
...
return os.WriteFile(filepath.Join(dirPath, ".stem"), []byte(b.String()), 0o644)
```

### Invalid JSON stdout on `missing_schema`
Default output format is JSON, but successful scaffold writes a human line to stdout before `outputJSON`:

- `cmd/rootline/apply.go` lines 64 and 117.
- Result: stdout can be `scaffolded /path/.stem\n{"applied":...}`, which is invalid JSON for `-o json`/default.

### Root-most vs closest `.stem` targeting
`WalkUp` returns root-to-leaf after reversing discovery order (`internal/rules/discovery.go` lines 18-21, 67-69). `cmd/rootline/apply.go` says “Use the closest stem file” but uses `entries[0]`:

```go
// cmd/rootline/apply.go lines 74-75
// Use the closest stem file.
stemPath := entries[0].Path
```

`internal/infer/schema_coverage.go` already documents the correct interpretation:

```go
// lines 53-55
// entries[0] is root-most, entries[len-1] is leaf-most / closest
closest := entries[len(entries)-1].Path
```

### Partial writes / non-transactionality
Current order is scaffold writes → schema write → data writes. Any later error leaves prior writes in place:

- Scaffold writes before `WalkUp` and before schema/data application (`cmd/rootline/apply.go` lines 55-67).
- Schema mutations are written as a batch to one `.stem` (`internal/infer/apply.go` lines 96-106) before data corrections.
- Data corrections write per file inside loops; a later missing/unwritable path returns an error after earlier files may already be modified (`internal/infer/apply.go` lines 157-184, 198-228).

No apply rollback/post-validation exists. A rollback pattern exists for `set` but only single-file: `cmd/rootline/set.go` lines 217-235 and 297-326.

### ApplyResult contract gap
`internal/infer/apply.go` lines 14-19:

```go
type ApplyResult struct {
  Applied []string `json:"applied"`
  Skipped []string `json:"skipped"`
  DryRun bool `json:"dry_run,omitempty"`
}
```

Most other JSON outputs include `version` and `kind` (e.g. `AnalyzeReport` in `internal/infer/report.go` lines 5-13, `NewAnalyzeReport` lines 43-49). `ApplyResult` lacks both.

### Existing test coverage constraints
- CLI apply test only checks the command returns no error after `apply --dry-run`; it does not assert no `.stem`/document changes and does not JSON-parse stdout (`cmd/rootline/coverage_test.go` lines 70-83).
- Dry-run no-write tests exist only for `ApplyDataCorrections`, not `ApplySchemaInferences` or `runApply` scaffold (`internal/infer/apply_test.go` lines 233-255; `internal/e2e/apply_test.go` lines 87-112).
- `ScaffoldSchema` tests assert write behavior only (`internal/infer/apply_test.go` lines 1005-1042).

## Architecture

`rootline analyze` emits `infer.AnalyzeReport` (`version: 1`, `kind: "analyze"`) with categories of `ReportInference`. `rootline apply` reads that JSON, resolves `report.Path`, scaffolds any non-agent `missing_schema`, finds `.stem` entries with `rules.WalkUp`, splits inferences into schema vs data, then calls:

1. `infer.ApplySchemaInferences(stemPath, schemaInferences)` - currently always writes one `.stem`.
2. `infer.ApplyDataCorrections(dataInferences, ApplyOptions{DryRun, Root})` - honors dry-run for document frontmatter only.

The result is rendered as table or JSON. Because scaffold messages are printed separately and schema/scaffold ignore dry-run, CLI behavior and JSON contract are inconsistent with the `--dry-run` flag.

## Start Here
Open `cmd/rootline/apply.go` first. The main bugs are visible there: unconditional scaffold/write-before-result, `entries[0]` target selection, schema apply without dry-run options, and pre-JSON stdout logging.

## Recommended Implementation Order
1. Add failing CLI tests first: `apply --dry-run` with a schema inference does not modify `.stem`; `apply --dry-run` with `missing_schema` does not create `.stem`; default JSON stdout remains parseable; `ApplyResult` includes `version` and `kind`.
2. Extend apply planning/result model: add `Version int`, `Kind string` (likely `rootline/apply`), and route scaffold messages into `Applied`/`Skipped` instead of direct stdout for JSON mode.
3. Add dry-run support to schema/scaffold paths: pass `ApplyOptions` (or a schema-specific option) into `ApplySchemaInferences`; split scaffold into build/plan vs write, or add dry-run-aware wrapper.
4. Fix `.stem` targeting deliberately: at minimum replace `entries[0]` with `entries[len(entries)-1]` if “closest” is intended; better, target `inf.Source` when it points at a physical `.stem`, then fall back to closest for report path.
5. Address partial writes: compute a full plan first, snapshot all target files/stems before writes, apply writes, and rollback snapshots on any error. Include scaffolded `.stem` creation/deletion in rollback.
6. Add post-apply validation/guardrail work after write semantics are correct; this aligns with roadmap T004.
7. Update `.claude/skills/rootline/*` and add/adjust user docs after the dry-run guarantee is true; remove the current “apply --dry-run is unsafe” warning.

## Supervisor coordination
No blocker encountered. Remaining implementation questions that matter:

- Should schema inferences target the closest `.stem`, the root-most `.stem`, or the physical source `.stem` per inference when `ReportInference.Source` is set?
- Should `apply --dry-run` report planned schema/scaffold changes in `applied` (current data dry-run pattern) or a new `planned`/`would_apply` field?
- Is full transactionality required across schema + all document writes, or is best-effort rollback on error acceptable for this repair roadmap?
- Should `rootline apply` perform post-apply `validate --all <report.Path>` automatically or only document/enforce it in protected workflows?
