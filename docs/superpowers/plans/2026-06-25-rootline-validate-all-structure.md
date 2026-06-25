# Validate --all Structural Check Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Run `rules.ValidateStructure` per-file in `runValidateAll` so `validate --all` catches `multiple_yaml_documents` like single-file validate already does.

**Architecture:** Single edit in `cmd/rootline/validate.go`'s `runValidateAll` per-record loop: re-read each file's raw content (`os.ReadFile(absPath)`) and append `rules.ValidateStructure(content, rec.Path)` to the record's errors. The `Record` does not retain raw content, so re-reading mirrors the single-file path with minimal blast radius.

**Tech Stack:** Go 1.25+, standard `testing`. Existing harness: `runCmd(t, args...) (string, error)`.

## Global Constraints

- Go 1.25+; no new third-party dependencies.
- `just check` (gofmt + golangci-lint + build) must pass; `just test` runs with `-race`, 0 failures.
- Coverage floor ≥ 85% for `cmd/rootline` (`.coverage-floors.toml`).
- Conventional Commits.
- `.stem` fixtures use `version: 2`.
- Only modify `cmd/rootline/validate.go` (one block in `runValidateAll`) and add a test file. Do NOT touch `extract.go`, the `Record`, read-path commands, or `runValidateFiles` (it already runs the check).

---

## Task 1: Run ValidateStructure per-file in `runValidateAll`

**Files:**
- Modify: `cmd/rootline/validate.go` (`runValidateAll` per-record loop, ~line 178)
- Test: `cmd/rootline/validate_structure_all_test.go`

**Interfaces:**
- Consumes: `rules.ValidateStructure(content []byte, path string) []rules.ValidationError` (existing, internal/rules/validate.go:413); `runCmd(t, args...) (string, error)` (commands_test.go:160).

- [ ] **Step 1: Write the failing tests**

Create `cmd/rootline/validate_structure_all_test.go`:

```go
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeStructFixture(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Two YAML documents in one file → ValidateStructure flags multiple_yaml_documents.
const multiDocFile = "---\nfoo: 1\n---\nbody1\n---\nbar: 2\n---\nbody2\n"

func TestValidateAll_MultipleYAMLDocuments(t *testing.T) {
	dir := t.TempDir()
	writeStructFixture(t, dir, ".stem", "version: 2\nschema:\n  foo:\n    type: string\n")
	writeStructFixture(t, dir, "multi.md", multiDocFile)

	out, err := runCmd(t, "validate", "--all", dir)
	if err == nil {
		t.Fatalf("expected validate --all to fail on multiple YAML documents, got nil; out=%s", out)
	}
	if !strings.Contains(out, "multiple_yaml_documents") {
		t.Errorf("expected multiple_yaml_documents in --all output, got: %s", out)
	}
}

func TestValidateAll_SingleDocument_NoStructuralError(t *testing.T) {
	dir := t.TempDir()
	writeStructFixture(t, dir, ".stem", "version: 2\nschema:\n  foo:\n    type: string\n")
	writeStructFixture(t, dir, "ok.md", "---\nfoo: bar\n---\n# Body\n")

	out, _ := runCmd(t, "validate", "--all", dir)
	if strings.Contains(out, "multiple_yaml_documents") {
		t.Errorf("did not expect multiple_yaml_documents for a single-document file, got: %s", out)
	}
}
```

- [ ] **Step 2: Run tests to verify the failing one fails**

Run: `go test ./cmd/rootline/ -run TestValidateAll_MultipleYAMLDocuments -v`
Expected: FAIL — `err == nil` (current `--all` never runs the structural check, so the multi-doc file passes).

- [ ] **Step 3: Wire `ValidateStructure` into `runValidateAll`**

In `cmd/rootline/validate.go`, find the per-record loop body in `runValidateAll` (around line 178):

```go
		errs := rules.Validate(ctx, rec, effective)
		errs = append(errs, rules.ExtractionErrors(rec)...)

		results = append(results, rules.NewValidationResult(rec.Path, errs))
```

Insert the structural check (re-reading the raw content) before building the result:

```go
		errs := rules.Validate(ctx, rec, effective)
		errs = append(errs, rules.ExtractionErrors(rec)...)
		if content, readErr := os.ReadFile(absPath); readErr == nil {
			errs = append(errs, rules.ValidateStructure(content, rec.Path)...)
		}

		results = append(results, rules.NewValidationResult(rec.Path, errs))
```

(`absPath` is already computed earlier in the loop as `filepath.Join(root, rec.Path)`; `os` is already imported — `runValidateFiles` uses `os.ReadFile`.)

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/rootline/ -run TestValidateAll -v`
Expected: PASS (2 tests) — multi-doc file now fails `--all` with `multiple_yaml_documents`; single-doc file stays clean.

- [ ] **Step 5: Full suite, check, coverage**

Run: `just test && just check`
Expected: all packages PASS with `-race`; gofmt clean; lint clean; build OK.

Run: `go test ./cmd/rootline/ -coverprofile=/tmp/cov.out && go tool cover -func=/tmp/cov.out | tail -1`
Expected: `cmd/rootline` coverage ≥ 85%.

- [ ] **Step 6: Commit**

```bash
git add cmd/rootline/validate.go cmd/rootline/validate_structure_all_test.go
git commit -m "fix(validate): run structural check per-file in --all mode"
```

---

## Self-Review (completed by plan author)

**Spec coverage:**
- Run `ValidateStructure` per-record in `runValidateAll` (spec §"Arquitectura") → Task 1 Step 3. ✓
- Re-read content via `os.ReadFile(absPath)`, skip on read error (spec §"Arquitectura") → Task 1 Step 3. ✓
- Tests: multi-doc fails `--all`; single-doc clean (spec §"Plan de tests") → Task 1 Step 1. ✓
- DoD: check/test/coverage (spec §"Verificación") → Task 1 Step 5. ✓
- Out of scope: extract.go/Record/read-paths/runValidateFiles untouched → only validate.go + new test changed. ✓

**Placeholder scan:** none — full code and exact commands in every step.

**Type consistency:** `rules.ValidateStructure(content []byte, path string) []rules.ValidationError` matches its definition (internal/rules/validate.go:413); `absPath`, `os.ReadFile`, `errs` all match the existing loop in validate.go.
