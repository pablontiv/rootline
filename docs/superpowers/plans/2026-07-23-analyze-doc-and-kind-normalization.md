# Analyze Doc + Report-Kind Normalization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Normalize the `analyze` JSON report `kind` to `rootline/analyze` (with backward-compatible reads), then write `docs/analyze.md` reflecting the final behavior.

**Architecture:** Two sequenced changes. First fix the report kind at its single producer and make its two consumers accept both the old and new value (non-breaking). Then author the command reference doc against the now-final output shape, verifying every claim against the real CLI.

**Tech Stack:** Go 1.26, cobra CLI, Go standard `testing`, `just` recipes.

## Global Constraints

- No pinned product versions in living docs (version facts live in CHANGELOG/tags).
- Every documented command/flag/output/example verified against real CLI help, source, or execution.
- `just check` (gofmt + golangci-lint + build) and `just test` (`go test ./... -race`) must pass.
- Trunk-based: direct conventional commits to `master`, push at the very end. No AI attribution. End commit bodies with `Claude-Session: https://claude.ai/code/session_01Fv9yVCrptEWv7YHLiKX37e`.
- Sequence is fixed: Task 1 (kind) precedes Task 2 (doc) — the doc must describe the final kind.

---

### Task 1: Normalize analyze report kind with dual-accept reads

**Files:**
- Modify: `internal/infer/report.go:43-50` (producer + doc comment)
- Modify: `cmd/rootline/schema.go:223` (dispatch probe)
- Modify: `cmd/rootline/schema.go:306-308` (validation guard)
- Test: `internal/e2e/analyze_test.go` (new producer-kind test + fix existing assertion at :115)
- Test: `cmd/rootline/schema_test.go` (new dual-accept test + fix existing assertion at :704)
- Test: `internal/infer/report_test.go:40` (fix existing kind assertion)
- Test: `cmd/rootline/governance_multistem_test.go:94,429` (fix existing kind assertions)

**Pre-existing assertions to migrate (found by self-review):** five tests assert the legacy kind `"analyze"` and will fail once the producer changes. `cmd/rootline/coverage_test.go:28` already tolerates both — leave it. Step 7 migrates the other five.

**Interfaces:**
- Consumes: `infer.NewAnalyzeReport(path string) *infer.AnalyzeReport`; `infer.AnalyzeReport{Version int; Kind string; Path string; Categories []CategoryResult; Summary ReportSummary}`; test helper `executeSchemaApply(t *testing.T, args ...string) (string, error)` (already in `schema_test.go`).
- Produces: `analyze` reports whose `kind` is `rootline/analyze`; `schema apply` that accepts reports with kind `analyze` OR `rootline/analyze`.

- [ ] **Step 1: Write the failing producer test**

Add to `internal/e2e/analyze_test.go`:

```go
func TestAnalyzeReportKindIsNamespaced(t *testing.T) {
	r := infer.NewAnalyzeReport("some/path")
	if r.Kind != "rootline/analyze" {
		t.Errorf("expected kind %q, got %q", "rootline/analyze", r.Kind)
	}
}
```

- [ ] **Step 2: Write the failing dual-accept test**

Add to `cmd/rootline/schema_test.go` (it already imports `os`, `path/filepath`, `encoding/json`, `testing`; add `"github.com/pablontiv/rootline/internal/infer"` to its import block):

```go
// TestSchemaApplyAcceptsBothAnalyzeKinds verifies schema apply dispatches an
// analyze report whether it carries the legacy "analyze" kind or the
// namespaced "rootline/analyze" kind (backward-compatible reads).
func TestSchemaApplyAcceptsBothAnalyzeKinds(t *testing.T) {
	for _, kind := range []string{"analyze", "rootline/analyze"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			report := infer.AnalyzeReport{Version: 1, Kind: kind, Path: root}
			data, _ := json.Marshal(report)
			reportFile := filepath.Join(root, "report.json")
			if err := os.WriteFile(reportFile, data, 0o644); err != nil {
				t.Fatalf("writing report: %v", err)
			}
			if _, err := executeSchemaApply(t, "--report", reportFile); err != nil {
				t.Errorf("expected kind %q to be accepted, got error: %v", kind, err)
			}
		})
	}
}
```

- [ ] **Step 3: Run both tests to verify they fail**

Run: `go test ./internal/e2e/ -run TestAnalyzeReportKindIsNamespaced -v && go test ./cmd/rootline/ -run TestSchemaApplyAcceptsBothAnalyzeKinds -v`
Expected: FAIL — producer still emits `analyze`; the `rootline/analyze` subtest hits `wrong report kind: rootline/analyze (expected analyze)`.

- [ ] **Step 4: Update the producer**

In `internal/infer/report.go`, change the doc comment and constructor:

```go
// NewAnalyzeReport creates a report with version 1 and kind "rootline/analyze".
func NewAnalyzeReport(path string) *AnalyzeReport {
	return &AnalyzeReport{
		Version: 1,
		Kind:    "rootline/analyze",
		Path:    path,
	}
}
```

- [ ] **Step 5: Update the dispatch probe**

In `cmd/rootline/schema.go` (~line 223):

```go
	if probe.Kind == "analyze" || probe.Kind == "rootline/analyze" {
		return runSchemaApplyFromAnalyze(cmd, data)
	}
```

- [ ] **Step 6: Update the validation guard**

In `cmd/rootline/schema.go` `runSchemaApplyFromAnalyze` (~line 306):

```go
	if report.Kind != "analyze" && report.Kind != "rootline/analyze" {
		return fmt.Errorf("wrong report kind: %s (expected analyze or rootline/analyze)", report.Kind)
	}
```

- [ ] **Step 7: Migrate the five pre-existing kind assertions**

These tests emit an `analyze` report and assert the old kind; update each to the namespaced kind.

- `internal/e2e/analyze_test.go:115` — `if decoded.Kind != "analyze" {` → `if decoded.Kind != "rootline/analyze" {` (and its error string).
- `internal/infer/report_test.go:40` — `if decoded.Kind != "analyze" {` → `if decoded.Kind != "rootline/analyze" {` (and its error string).
- `cmd/rootline/schema_test.go:704` — `if report["kind"] != "analyze" {` → `if report["kind"] != "rootline/analyze" {` (and its error string).
- `cmd/rootline/governance_multistem_test.go:94` — `if report["kind"] != "analyze" {` → `if report["kind"] != "rootline/analyze" {` (and its error string).
- `cmd/rootline/governance_multistem_test.go:429` — `if report["kind"] != "analyze" {` → `if report["kind"] != "rootline/analyze" {` (and its error string).

Leave `cmd/rootline/coverage_test.go:28` unchanged — it already accepts both.

- [ ] **Step 8: Run the affected tests to verify they pass**

Run: `go test ./internal/e2e/ -run TestAnalyzeReportKindIsNamespaced -v && go test ./cmd/rootline/ -run TestSchemaApplyAcceptsBothAnalyzeKinds -v && go test ./internal/infer/ ./internal/e2e/ ./cmd/rootline/ -run 'Kind|Analyze|SchemaApply|Governance'`
Expected: PASS — new tests green and migrated assertions green.

- [ ] **Step 9: Full gates**

Run: `just check && just test`
Expected: gofmt clean, golangci-lint `0 issues`, build ok, all packages `ok` (race clean).

- [ ] **Step 10: Commit**

```bash
git add internal/infer/report.go cmd/rootline/schema.go internal/e2e/analyze_test.go cmd/rootline/schema_test.go internal/infer/report_test.go cmd/rootline/governance_multistem_test.go
git commit -m "$(cat <<'EOF'
fix(schema): normalize analyze report kind to rootline/analyze

analyze emitted kind "analyze" while every other command uses
rootline/<name>. Producer now emits rootline/analyze; schema apply
accepts both the legacy and namespaced kind so reports saved by an
older (auto-updating) binary still apply.

Claude-Session: https://claude.ai/code/session_01Fv9yVCrptEWv7YHLiKX37e
EOF
)"
```

---

### Task 2: Write docs/analyze.md

**Files:**
- Create: `docs/analyze.md`
- Reference (verify against, do not modify): `cmd/rootline/analyze.go`, `internal/infer/report.go`

**Interfaces:**
- Consumes: the finalized report shape from Task 1 (`kind: rootline/analyze`).
- Produces: `docs/analyze.md` (human + agent reference for the `analyze` command).

- [ ] **Step 1: Capture the real output shape**

Run (dev build avoids the auto-updater; build once if absent: `go build -o /tmp/rl ./cmd/rootline`):
`/tmp/rl analyze docs/roadmap/O16-autoupdate-integration -o json | jq '{version, kind, path, categories: (.categories[0]), summary}'`
Expected: `kind` is `rootline/analyze`; `summary` has `total_inferences`, `agent_required`, `engine_resolved`; each category has `id`, `name`, `inference_count`, `inferences[]` where an inference has `type`, `field`, `value`, `message`, `requires_agent`. Use this captured JSON verbatim as the doc's output example.

- [ ] **Step 2: Confirm flags and detector list against source**

Verify against `cmd/rootline/analyze.go`: flags are `--incremental` (bool, "report only inferences not covered by existing .stem") and `--threshold` (float, default `0.60`, "section pattern detection threshold (0.0-1.0)") — lines 34-35. The 14 detector categories (id → name) come verbatim from the `categories` slice (lines 105-156): `field_types`, `required_fields`, `enum_values`, `constant_fields`, `link_types`, `back_references`, `cross_references`, `section_patterns`, `invariants`, `formal_dependencies`, `traceability`, `structural` (12 data) + `schema_coverage`, `validation_gaps` (2 governance).

- [ ] **Step 3: Write the doc**

Create `docs/analyze.md` following the existing `docs/<cmd>.md` skeleton (see `docs/validate.md` / `docs/query.md` for heading style). Sections:

```markdown
# analyze

Run all inference detectors over a directory and produce a structured report
of schema and content patterns. The report feeds `schema apply` (schema
proposals) and `repair apply` (data-only repairs).

## Usage

​```bash
rootline analyze [directory]           # defaults to .
rootline analyze docs/ -o json
​```

## Flags

| Flag | Description |
|------|-------------|
| `--incremental` | Report only inferences not covered by existing `.stem` files |
| `--threshold <0.0-1.0>` | Section-pattern detection threshold (default `0.60`) |

Global flags `--output json|table` and `--field <path>` also apply.

## Detectors

Fourteen detectors run per invocation — twelve data detectors and two
governance detectors.

**Data:** field types, required fields, enum values, constant fields, link
types, back references, cross references, section patterns, invariants,
formal dependencies, traceability links, structural rules.

**Governance:** schema coverage (directories without a `.stem`), validation
gaps (enum without values, untyped fields, incomplete sequences, required
understatement).

## JSON output

​```json
<PASTE the captured JSON from Step 1 here — real shape, kind rootline/analyze>
​```

- `version` — contract version.
- `kind` — `rootline/analyze`.
- `categories[]` — one per detector: `id`, `name`, `inference_count`, `inferences[]`.
- Each inference: `type`, `field`, `value`, `message`, `requires_agent`.
- `summary` — `total_inferences`, `agent_required`, `engine_resolved`.

## Consuming the report

​```bash
# Schema proposals → .stem files
rootline analyze docs/ -o json > analyze.json
rootline schema apply --report analyze.json --dry-run
rootline schema apply --report analyze.json

# Data-only repairs → document frontmatter
rootline repair apply --report analyze.json --dry-run
rootline repair apply --report analyze.json
​```

Inferences with `requires_agent: true` are skipped by the engine and need
human/agent disambiguation.
```

Replace the `<PASTE …>` placeholder with the real JSON captured in Step 1. No literal product versions anywhere.

- [ ] **Step 4: Verify the doc's commands actually run**

Run each command block from the doc against the dev build in a temp copy; confirm `schema apply`/`repair apply` accept the generated report and `--incremental`/`--threshold` are honored.
Expected: no errors; report `kind` in the doc matches live output.

- [ ] **Step 5: Cross-link from README command list (if present)**

Run: `rg -n "docs/validate.md|docs/query.md" README.md`
If the README references per-command docs in a list, add the analogous `docs/analyze.md` link in the same place. If no such list exists, skip (do not invent one).

- [ ] **Step 6: Commit**

```bash
git add docs/analyze.md README.md
git commit -m "$(cat <<'EOF'
docs: add analyze command reference

analyze was the only major command without a docs/<cmd>.md page.
Covers the 14 detectors, --incremental/--threshold, the real JSON
report shape (kind rootline/analyze), and the schema apply / repair
apply consumption loop. All claims verified against the CLI.

Claude-Session: https://claude.ai/code/session_01Fv9yVCrptEWv7YHLiKX37e
EOF
)"
```

---

### Task 3: Roadmap closeout and push

**Files:**
- Modify: `docs/roadmap/T009-write-analyze-command-doc.md` (estado)
- Modify: `docs/roadmap/T010-normalize-analyze-report-kind.md` (estado)

**Interfaces:**
- Consumes: completed Tasks 1 and 2.
- Produces: roadmap records marked `Completed`; all commits on `origin/master`.

- [ ] **Step 1: Mark both records Completed (dogfood `rootline set`)**

Run:
```bash
rootline set docs/roadmap/T009-write-analyze-command-doc.md estado=Completed
rootline set docs/roadmap/T010-normalize-analyze-report-kind.md estado=Completed
```
Expected: each prints `set estado = "Completed"`.

- [ ] **Step 2: Validate the roadmap**

Run: `just validate`
Expected: summary `valid N/N, errors 0` (N unchanged from before; only estado values changed).

- [ ] **Step 3: Final gates**

Run: `just check && just test`
Expected: all green.

- [ ] **Step 4: Commit and push**

```bash
git add docs/roadmap/T009-write-analyze-command-doc.md docs/roadmap/T010-normalize-analyze-report-kind.md
git commit -m "$(cat <<'EOF'
docs(roadmap): mark T009/T010 complete

analyze command reference written and report kind normalized.

Claude-Session: https://claude.ai/code/session_01Fv9yVCrptEWv7YHLiKX37e
EOF
)"
git push
```
Expected: push succeeds (the pre-push docs-sync guard is satisfied — docs and roadmap changed alongside code).

---

## Verification (end-to-end)

- `rootline analyze docs/roadmap -o json | jq .kind` → `"rootline/analyze"`.
- `rootline schema apply --report <analyze.json>` accepts both a legacy `analyze` report and a `rootline/analyze` report (Task 1 test).
- `docs/analyze.md` exists; every command block runs; no pinned product versions.
- `just check` + `just test` green; `just validate` roadmap green.
- `git status` clean; `master` == `origin/master`.
