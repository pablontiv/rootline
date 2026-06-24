# Rootline Fase 2 — Port `update_stem` to schema apply, then remove `apply` — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Make `schema apply` actually apply schema-field modifications to existing `.stem` files (by consuming an `analyze` report), then delete the deprecated `apply` command and its orphaned data-correction code — without losing any capability.

**Architecture:** The deprecated `apply` command read an `analyze` report and applied its schema-modifying inferences via `infer.ApplySchemaInferences`. `schema apply` currently only handles `SchemaProposalsReport` (create_stem; `update_stem` is a no-op stub at `cmd/rootline/schema.go:260-266`). We relocate `apply`'s schema half into `schema apply` via report-kind dispatch (Option D), keeping `infer.ApplySchemaInferences` unchanged. The data-correction half (`infer.ApplyDataCorrections` + helpers) is already covered by `repair apply` (`internal/fix/repair.go` handles `CorrectValue`/`MigrateValue`/`AddField` with post-validation+rollback), so it is deleted as orphaned. Then `apply` is removed.

**Tech Stack:** Go 1.25+, cobra, gopkg.in/yaml.v3, just.

## Global Constraints

- `just check` (gofmt + lint + build) and `just test` (`go test ./... -race`) exit 0 at the end of every task.
- `just coverage-check` must pass (≥85% per `.coverage-floors.toml`) before the final commit — deletions remove both code and its tests; the new schema-apply path adds tested code.
- Conventional commits; NEVER `Co-Authored-By` / AI attribution.
- Verified facts (do not re-derive): `infer.ApplySchemaInferences(stemPath string, inferences []ReportInference, dryRun bool) (*ApplyResult, error)` applies schema-modifying inference types `enum_values`, `required_field`, `constant_field`, `field_type`, `untyped_field`, `sequence_incomplete` to a `.stem` (respects `dryRun`). `AnalyzeReport` (`internal/infer/report.go:6`) has `Version int`, `Kind string` = `"analyze"`, `Path string`, `Inferences []ReportInference`. `SchemaProposalsReport` (`cmd/rootline/schema.go:33`) has `Kind` = `"rootline/schema-proposals"`. `apply.go:36-133` (`runApply`) is the working reference for the schema half. `ApplyDataCorrections` non-test caller is ONLY `apply.go:117`; `rewriteFrontmatter`/`parseFrontmatter`/`applyDocCorrection`/`applyAddField` have NO non-test callers outside `internal/infer/apply.go`. `repair apply` covers data corrections (`internal/fix/repair.go:88-127`).

**Setup (once):** Currently on `master`. Create a feature branch before Task 1: `git -C /Users/Shared/harness/rootline checkout -b feat/remove-apply-port-schema-update`. The `SKILL.md` edits (Task 4) live OUTSIDE the repo at `/Users/pones/.claude/skills/rootline/` — not part of repo commits.

---

### Task 1: `schema apply` consumes an `analyze` report (port apply's schema capability) + drop the dead `update_stem` stub

This is the functionality-preserving step. Do it BEFORE removing `apply`.

**Files:**
- Modify: `cmd/rootline/schema.go` (`runSchemaApply` ~`:201`; the `update_stem` stub `:260-266`)
- Test: `cmd/rootline/schema_test.go` (or a new `schema_apply_analyze_test.go`)
- Reference (read, do not modify yet): `cmd/rootline/apply.go:36-133`

**Interfaces:**
- Consumes: `infer.ApplySchemaInferences(stemPath, []infer.ReportInference, dryRun)`, `infer.AnalyzeReport`, `rules.Resolve` (closest-stem resolution as in `apply.go:76-83`).
- Produces: `schema apply --report <analyze.json>` applies schema-modifying inferences to the closest `.stem`; returns the existing `SchemaApplyResult` shape (`version:1`, `kind:"rootline/schema-apply"`).

- [ ] **Step 1: Write the failing test** — `schema apply` on an analyze report extends an enum on an existing `.stem`.

Create a test that: writes a `.stem` with an enum field (e.g. `estado: {type: enum, values: [Pending, Done]}`) and a doc using a new value `Blocked`; runs `analyze` to produce a report JSON with an `enum_values` inference; writes that report to a file; runs `schema apply --report <file>`; asserts the `.stem` now contains `Blocked` in the enum values. Mirror the fixture style of existing `cmd/rootline/*_test.go` (use the `runCmd(t, ...)` helper).

```go
func TestSchemaApply_AnalyzeReport_ExtendsEnum(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil { t.Fatal(err) }
	stem := "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: enum\n    values: [Pending, Done]\n"
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stem), 0644)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\nestado: Pending\n---\n# A\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\nestado: Blocked\n---\n# B\n"), 0644)

	out, err := runCmd(t, "analyze", dir)
	if err != nil { t.Fatalf("analyze: %v\n%s", err, out) }
	reportFile := filepath.Join(dir, "report.json")
	mustWriteFile(t, reportFile, []byte(out), 0644)

	if _, err := runCmd(t, "schema", "apply", "--report", reportFile); err != nil {
		t.Fatalf("schema apply: %v", err)
	}
	got := string(mustReadFile(t, filepath.Join(dir, ".stem")))
	if !strings.Contains(got, "Blocked") {
		t.Fatalf("expected enum extended with Blocked, got:\n%s", got)
	}
}
```
(Adapt helper names — `runCmd`, `mustWriteFile`, `mustReadFile` — to whatever the existing tests in `cmd/rootline/` use; check `coverage_test.go` for the exact helpers.)

- [ ] **Step 2: Run it to confirm it fails**

Run: `go test ./cmd/rootline/ -run TestSchemaApply_AnalyzeReport_ExtendsEnum -v`
Expected: FAIL — currently `schema apply` rejects kind `analyze` ("wrong report kind") at `schema.go:219-221`.

- [ ] **Step 3: Implement report-kind dispatch in `runSchemaApply`**

In `cmd/rootline/schema.go` `runSchemaApply` (after reading `data`, before unmarshaling into `SchemaProposalsReport`), probe the kind and branch. When kind is `"analyze"`, handle via a new function that mirrors `apply.go`'s schema half:

```go
	// Probe report kind to dispatch.
	var probe struct {
		Kind string `json:"kind"`
	}
	_ = json.Unmarshal(data, &probe)
	if probe.Kind == "analyze" {
		return runSchemaApplyFromAnalyze(cmd, data)
	}
	// ... existing SchemaProposalsReport path unchanged ...
```

Add `runSchemaApplyFromAnalyze(cmd *cobra.Command, data []byte) error` that mirrors `apply.go:44-132` BUT only the schema-modifying half (no `ApplyDataCorrections` — data corrections are `repair apply`'s job): unmarshal `infer.AnalyzeReport`; resolve `report.Path`; resolve the closest `.stem` (as `apply.go:76-83` does via `rules.Resolve`); filter inferences to the schema-modifying types (`enum_values`, `required_field`, `constant_field`, `field_type`, `untyped_field`, `sequence_incomplete` — copy the separation logic from `apply.go:92-104`, keeping only the schema set); call `infer.ApplySchemaInferences(stemPath, schemaInferences, schemaApplyDryRun)`; return a `SchemaApplyResult` (`version:1`, `kind:"rootline/schema-apply"`) rendered via the existing table/JSON output path. Read `apply.go:36-133` and replicate its proven structure.

- [ ] **Step 4: Run the test to confirm it passes**

Run: `go test ./cmd/rootline/ -run TestSchemaApply_AnalyzeReport_ExtendsEnum -v`
Expected: PASS.

- [ ] **Step 5: Add a dry-run safety test**

Write `TestSchemaApply_AnalyzeReport_DryRunNoWrite`: same fixture, run `schema apply --report <file> --dry-run`, assert the `.stem` bytes are byte-identical before/after (the analyze path must respect `schemaApplyDryRun`; `ApplySchemaInferences` honors `dryRun`). Run it; expected PASS.

- [ ] **Step 6: Drop the dead `update_stem` stub**

In `runSchemaApply`'s SchemaProposalsReport `switch proposal.Operation` (`schema.go:251-266`), remove the `case "update_stem"` no-op block (`:260-266`) — `schema propose` never emits `update_stem` (verified), and real schema updates now flow through the analyze-report path. If a test asserts the stub's "update_stem: ..." output, update/remove it (grep `rg -n "update_stem" cmd/rootline/`).

- [ ] **Step 7: Full suite + commit**

Run: `just test` → exit 0. Run: `just check` → exit 0.
```bash
git add cmd/rootline/schema.go cmd/rootline/schema_test.go
git commit -m "feat(schema): apply schema inferences from analyze reports via schema apply"
```

---

### Task 2: Remove the deprecated `apply` command and its CLI/e2e tests

**Files:**
- Delete: `cmd/rootline/apply.go`, `cmd/rootline/apply_deprecation_test.go`, `cmd/rootline/apply_table_test.go`
- Modify: `cmd/rootline/coverage_test.go` (remove `TestApplySafety_EnumExtensionDryRun` ~`:505-535`, `TestApplySafety_DryRunMissingSchema` `:540-606`, `TestApplySafety_JSONPurityWithMissingSchema` `:612-661`)
- Modify: `cmd/rootline/coverage_boost_test.go` (remove `TestRunApplyDeprecated` `:30-51`)
- Modify: `cmd/rootline/commands_test.go` (remove `applyDryRun` reset `:81` and the `applyCmd` flag-reset block `:146-148`)
- Modify: `internal/e2e/apply_test.go` (retarget the schema-applying tests to `schema apply`; remove the data-correction tests covered by repair)

- [ ] **Step 1: Baseline green** — `just test` exit 0.

- [ ] **Step 2: Delete the command + its dedicated tests**

```bash
git rm cmd/rootline/apply.go cmd/rootline/apply_deprecation_test.go cmd/rootline/apply_table_test.go
```
(`apply.go` defines `applyCmd`, `applyDryRun`, `runApply`, `renderApplyTable`, and the `init()` that calls `rootCmd.AddCommand(applyCmd)` — all removed together.)

- [ ] **Step 3: Remove the apply-only characterization + reset references**

In `cmd/rootline/coverage_test.go` delete the three `TestApplySafety_*` functions (they characterized `apply`'s dry-run bug, which dies with the command). In `cmd/rootline/coverage_boost_test.go` delete `TestRunApplyDeprecated`. In `cmd/rootline/commands_test.go` delete the `applyDryRun = false` line and the `applyCmd.Flags().Lookup("dry-run")` reset block.

- [ ] **Step 4: Retarget e2e schema tests; drop data-correction e2e tests**

In `internal/e2e/apply_test.go`: the tests that exercise schema application end-to-end (`TestApply_AnalyzeThenApplyThenValidate`, `TestApply_RequiresAgent_Skipped`, `TestApply_AddField_ThenValidate` if schema-typed, `TestApply_DryRun_NoFileChanges`) should be rewritten to invoke `schema apply --report <analyze.json>` instead of `apply` (the capability moved there in Task 1) — preserving their assertions. The pure data-correction test (`TestApply_DataCorrections_MigrateAndValidate`) is covered by `repair apply` tests (`cmd/rootline/repair_test.go`); delete it here. Rename the file to `schema_apply_e2e_test.go` if all remaining tests target schema apply.

- [ ] **Step 5: Build + full suite**

Run: `just check` → exit 0 (no dangling references to `applyCmd`/`runApply`/`renderApplyTable`).
Run: `just test` → exit 0.
Verify the command is gone: `rg -n '"apply"|applyCmd|runApply' cmd/rootline/` returns nothing (the `schema apply`/`repair apply` subcommands use their own command vars, not `applyCmd`).

- [ ] **Step 6: Commit**

```bash
git add -A cmd/rootline/ internal/e2e/
git commit -m "refactor: remove deprecated apply command (schema apply + repair apply cover it)"
```

---

### Task 3: Cascade-delete the orphaned data-correction code in `internal/infer/apply.go`

After Task 2, `ApplyDataCorrections` and its helpers have no callers. `ApplySchemaInferences` and its node helpers + `ScaffoldSchema` SURVIVE (used by schema apply).

**Files:**
- Modify: `internal/infer/apply.go` (delete `ApplyDataCorrections`, `applyDocCorrection`, `applyAddField`, `rewriteFrontmatter`, `parseFrontmatter`)
- Modify: `internal/infer/apply_test.go` (delete tests for the removed functions)

- [ ] **Step 1: Confirm orphaned (no non-test callers)**

Run: `rg -n "ApplyDataCorrections|rewriteFrontmatter|parseFrontmatter|applyDocCorrection|applyAddField" --type go -g '!*_test.go'`
Expected: only definitions inside `internal/infer/apply.go` (no external callers). If ANY other caller appears, STOP and report NEEDS_CONTEXT.

- [ ] **Step 2: Delete the orphaned functions**

In `internal/infer/apply.go` remove: `ApplyDataCorrections` (`:119-153`), `applyDocCorrection` (`:156-194`), `applyAddField` (`:197-238`), `rewriteFrontmatter` (`:241-269`), `parseFrontmatter` (`:272-288`). Keep `ApplySchemaInferences`, `ApplyResult`, `ApplyOptions` (if still used — check), and all `*Node` helpers used by `ApplySchemaInferences`. If `ApplyOptions` becomes unused after removing `ApplyDataCorrections`, remove it too.

- [ ] **Step 3: Delete their tests**

In `internal/infer/apply_test.go` remove the test functions that target the deleted functions (e.g. `TestApplyDataCorrections_*`, any `rewriteFrontmatter`/`parseFrontmatter`/`applyAddField` unit tests). Keep tests for `ApplySchemaInferences`.

- [ ] **Step 4: Build + suite + coverage**

Run: `just check` → exit 0. Run: `just test` → exit 0. Run: `just coverage-check` → exit 0 (≥85% all packages; if `internal/infer` dips, the cause is removed-but-still-counted lines — confirm the deleted functions and their tests both went). If coverage fails, report the failing package and numbers.

- [ ] **Step 5: Commit**

```bash
git add internal/infer/apply.go internal/infer/apply_test.go
git commit -m "refactor: remove orphaned data-correction code (covered by repair apply)"
```

---

### Task 4: Docs + skill — `apply` removed, `schema apply` analyze-report capability documented; sync the spec

**Files:**
- Modify: `README.md` (the CLI block listing `rootline apply ...`), `CLAUDE.md` (apply mentions)
- Modify: `docs/superpowers/specs/2026-06-24-rootline-sincerizacion-gaps-design.md` (Fase 2 section — correct the false "nothing lost" premise)
- Modify (skill, outside repo): `/Users/pones/.claude/skills/rootline/SKILL.md`, `ref-advanced.md`

- [ ] **Step 1: README** — remove the `rootline apply [file] [--dry-run]` line from the CLI block (around `:165`). Under the `schema apply` line, note it also accepts an `analyze` report to apply schema-field changes to existing `.stem` files.

- [ ] **Step 2: CLAUDE.md** — change the `apply` description from "deprecated, remains functional" to removed; state that `schema apply --report <analyze.json>` applies schema-modifying inferences and `repair apply` applies data repairs. Remove `apply.go` from the `cmd/rootline/` file list and the "apply.go uses this API" mention in the `internal/rules/` paragraph.

- [ ] **Step 3: Spec sync** — in the Fase 2 section of the design spec, replace the "borrarlo no pierde nada" premise with the verified reality: `schema apply`'s `update_stem` was a stub; the port (Option D — schema apply consumes analyze reports) preserves the schema-modification capability; data corrections were already covered by `repair apply`; then `apply` was removed.

- [ ] **Step 4: Skill (outside repo)** — in `/Users/pones/.claude/skills/rootline/SKILL.md` and `ref-advanced.md`, remove any guidance presenting `apply` as a usable command; ensure `analyze → schema apply --report` (schema) and `repair apply --report` (data) are the documented flows.

- [ ] **Step 5: Verify + commit (repo only)**

Run: `rg -n "rootline apply|legacy apply| apply\.go" README.md CLAUDE.md` → no live-command references remain.
Run: `just check` + `just test` → exit 0.
```bash
git add README.md CLAUDE.md docs/superpowers/specs/2026-06-24-rootline-sincerizacion-gaps-design.md
git commit -m "docs: remove apply command references; document schema apply analyze-report flow"
```

---

## Self-Review (spec coverage — Fase 2)

- Port `update_stem`/apply schema capability into `schema apply` (Option D) → Task 1 ✓
- Drop dishonest `update_stem` stub → Task 1 Step 6 ✓
- Remove `apply` command + CLI/e2e tests → Task 2 ✓
- Cascade-delete orphaned data-correction code (covered by repair apply) → Task 3 ✓
- Docs/skill/spec sync → Task 4 ✓

No placeholders: Task 1 gives the failing test verbatim and points the implementer at `apply.go:36-133` as the in-repo reference to replicate; deletions name exact symbols and line ranges; every task verifies with `just check`/`just test` (+ `just coverage-check` in Task 3). Order preserves capability before deletion.
```
