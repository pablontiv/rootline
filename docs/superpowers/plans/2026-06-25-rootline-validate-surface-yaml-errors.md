# Validate Surfacing Malformed YAML Errors Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `rootline validate` report malformed-YAML frontmatter (today's swallowed `record.Errors`) as a blocking `malformed_yaml` error, in both single-file and `--all` paths.

**Architecture:** A new `rules.ExtractionErrors(rec)` helper (mirrors `rules.ValidateStructure`) converts a record's non-fatal `extract.ExtractionError`s into blocking `ValidationError`s. It is appended to the per-record error slice in both `runValidateFiles` and `runValidateAll`. The permissive fallback parser and schema validation are untouched, so a malformed file reports `malformed_yaml` PLUS any schema errors from the salvaged frontmatter ("report everything").

**Tech Stack:** Go 1.25+, `gopkg.in/yaml.v3`, standard `testing`. Existing test harness: `runCmd(t, args...)` (cmd/rootline/commands_test.go:160).

## Global Constraints

- Go 1.25+; no new third-party dependencies.
- `just check` (gofmt + golangci-lint + build) must pass; `just test` runs with `-race`, 0 failures.
- Coverage floor ≥ 85% for `internal/rules` and `cmd/rootline` (`.coverage-floors.toml`).
- Conventional Commits.
- `.stem` fixtures use `version: 2`.
- `malformed_yaml` ValidationError must use exactly: `Rule: "malformed_yaml"`, `Field: "_frontmatter"`, `Severity: "error"`, `Source: rec.Path`, `Message: <the ExtractionError message verbatim>`, `Suggestion: "quote values containing ':' or other YAML-special characters"`.
- Do NOT modify `internal/extract/extract.go`, `fallbackParseFrontmatter`, or the read-path commands (`query`, `tree`, `graph`, `fix`, `set`, `explain`). Only `internal/rules` (new helper) and `cmd/rootline/validate.go` (wiring) change.
- The existing `--strict` flag (warnings→errors) is unrelated; do NOT touch it. `malformed_yaml` is severity `error` and blocks regardless of `--strict`.

---

## File Structure

- **Create:** `internal/rules/extraction_errors.go` — the `ExtractionErrors` helper (one responsibility: extraction-error → ValidationError conversion). Sibling to `validate.go`'s `ValidateStructure`.
- **Create:** `internal/rules/extraction_errors_test.go` — unit tests (mirrors `validate_structure_test.go`).
- **Modify:** `cmd/rootline/validate.go` — append `rules.ExtractionErrors(...)` in `runValidateFiles` (~line 104) and `runValidateAll` (~line 178).
- **Create:** `cmd/rootline/validate_yaml_test.go` — integration tests for both paths.

---

## Task 1: `rules.ExtractionErrors` helper

**Files:**
- Create: `internal/rules/extraction_errors.go`
- Test: `internal/rules/extraction_errors_test.go`

**Interfaces:**
- Consumes: `extract.Record` (`Path string`, `Errors []extract.ExtractionError` where `ExtractionError` has `Line int`, `Message string`); `rules.ValidationError` (`Rule, Field, Message, Source, Severity, Suggestion string`).
- Produces: `func ExtractionErrors(rec *extract.Record) []ValidationError`.

- [ ] **Step 1: Write the failing tests**

Create `internal/rules/extraction_errors_test.go`:

```go
package rules

import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func TestExtractionErrors_NoErrors(t *testing.T) {
	rec := &extract.Record{Path: "ok.md"}
	got := ExtractionErrors(rec)
	if got != nil {
		t.Errorf("expected nil for a record without extraction errors, got %v", got)
	}
}

func TestExtractionErrors_MalformedYAML(t *testing.T) {
	rec := &extract.Record{
		Path: "broken.md",
		Errors: []extract.ExtractionError{
			{Line: 1, Message: "malformed YAML frontmatter: yaml: mapping values are not allowed in this context"},
		},
	}
	got := ExtractionErrors(rec)
	if len(got) != 1 {
		t.Fatalf("expected 1 validation error, got %d", len(got))
	}
	e := got[0]
	if e.Rule != "malformed_yaml" {
		t.Errorf("expected rule malformed_yaml, got %q", e.Rule)
	}
	if e.Severity != "error" {
		t.Errorf("expected severity error, got %q", e.Severity)
	}
	if e.Field != "_frontmatter" {
		t.Errorf("expected field _frontmatter, got %q", e.Field)
	}
	if e.Source != "broken.md" {
		t.Errorf("expected source broken.md, got %q", e.Source)
	}
	if e.Message != "malformed YAML frontmatter: yaml: mapping values are not allowed in this context" {
		t.Errorf("expected message passthrough, got %q", e.Message)
	}
	if e.Suggestion == "" {
		t.Errorf("expected a non-empty suggestion")
	}
}

func TestExtractionErrors_MultipleErrors(t *testing.T) {
	rec := &extract.Record{
		Path: "broken.md",
		Errors: []extract.ExtractionError{
			{Line: 1, Message: "err one"},
			{Line: 2, Message: "err two"},
		},
	}
	got := ExtractionErrors(rec)
	if len(got) != 2 {
		t.Fatalf("expected 2 validation errors, got %d", len(got))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/rules/ -run TestExtractionErrors -v`
Expected: FAIL — `undefined: ExtractionErrors`.

- [ ] **Step 3: Implement the helper**

Create `internal/rules/extraction_errors.go`:

```go
package rules

import "github.com/pablontiv/rootline/internal/extract"

// ExtractionErrors converts a record's non-fatal extraction errors (e.g. malformed
// YAML frontmatter that fell back to the permissive line-by-line parser) into
// blocking ValidationErrors, so `validate` surfaces them instead of swallowing them.
// Returns nil when the record has no extraction errors.
func ExtractionErrors(rec *extract.Record) []ValidationError {
	if len(rec.Errors) == 0 {
		return nil
	}
	out := make([]ValidationError, 0, len(rec.Errors))
	for _, ee := range rec.Errors {
		out = append(out, ValidationError{
			Rule:       "malformed_yaml",
			Field:      "_frontmatter",
			Message:    ee.Message,
			Source:     rec.Path,
			Severity:   "error",
			Suggestion: "quote values containing ':' or other YAML-special characters",
		})
	}
	return out
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/rules/ -run TestExtractionErrors -v`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/rules/extraction_errors.go internal/rules/extraction_errors_test.go
git commit -m "feat(rules): add ExtractionErrors to convert extraction errors to validation errors"
```

---

## Task 2: Wire into both validate paths + integration tests

**Files:**
- Modify: `cmd/rootline/validate.go` (`runValidateFiles` ~line 104; `runValidateAll` ~line 178)
- Test: `cmd/rootline/validate_yaml_test.go`

**Interfaces:**
- Consumes: `rules.ExtractionErrors(rec *extract.Record) []ValidationError` (Task 1); `runCmd(t, args...) (string, error)` (commands_test.go:160).

- [ ] **Step 1: Write the failing integration tests**

Create `cmd/rootline/validate_yaml_test.go`:

```go
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFile is a tiny helper for these fixtures.
func writeYAMLFixture(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// malformed: title has an unquoted internal colon → yaml.v3 rejects it.
const malformedDoc = "---\ntitle: Foo: Bar\n---\n# Body\n"

func TestValidate_MalformedYAML_SingleFile(t *testing.T) {
	dir := t.TempDir()
	writeYAMLFixture(t, dir, ".stem", "version: 2\nschema:\n  title:\n    type: string\n    required: true\n")
	f := writeYAMLFixture(t, dir, "broken.md", malformedDoc)

	out, err := runCmd(t, "validate", f)
	if err == nil {
		t.Fatalf("expected validate to fail (exit error) on malformed YAML, got nil; out=%s", out)
	}
	if !strings.Contains(out, "malformed_yaml") {
		t.Errorf("expected malformed_yaml in output, got: %s", out)
	}
}

func TestValidate_MalformedYAML_All(t *testing.T) {
	dir := t.TempDir()
	writeYAMLFixture(t, dir, ".stem", "version: 2\nschema:\n  title:\n    type: string\n    required: true\n")
	writeYAMLFixture(t, dir, "broken.md", malformedDoc)

	out, err := runCmd(t, "validate", "--all", dir)
	if err == nil {
		t.Fatalf("expected validate --all to fail on malformed YAML, got nil; out=%s", out)
	}
	if !strings.Contains(out, "malformed_yaml") {
		t.Errorf("expected malformed_yaml in --all output, got: %s", out)
	}
}

func TestValidate_MalformedYAML_ReportsEverything(t *testing.T) {
	dir := t.TempDir()
	// estado is required; the malformed doc omits it entirely.
	writeYAMLFixture(t, dir, ".stem", "version: 2\nschema:\n  title:\n    type: string\n    required: true\n  estado:\n    type: string\n    required: true\n")
	f := writeYAMLFixture(t, dir, "broken.md", malformedDoc)

	out, err := runCmd(t, "validate", f)
	if err == nil {
		t.Fatalf("expected failure, got nil; out=%s", out)
	}
	if !strings.Contains(out, "malformed_yaml") {
		t.Errorf("expected malformed_yaml in output, got: %s", out)
	}
	// "report everything": the schema error for the missing required field also appears.
	if !strings.Contains(out, "estado") {
		t.Errorf("expected the missing-required schema error for 'estado' to also appear, got: %s", out)
	}
}

func TestValidate_ValidYAML_NoMalformedError(t *testing.T) {
	dir := t.TempDir()
	writeYAMLFixture(t, dir, ".stem", "version: 2\nschema:\n  title:\n    type: string\n    required: true\n")
	f := writeYAMLFixture(t, dir, "ok.md", "---\ntitle: \"Foo: Bar\"\n---\n# Body\n")

	out, err := runCmd(t, "validate", f)
	if err != nil {
		t.Fatalf("expected valid file to pass, got error; out=%s", out)
	}
	if strings.Contains(out, "malformed_yaml") {
		t.Errorf("did not expect malformed_yaml for a valid (quoted) title, got: %s", out)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/rootline/ -run TestValidate_MalformedYAML -v`
Expected: FAIL — `TestValidate_MalformedYAML_SingleFile` and `_All` get `err == nil` (current code swallows the parse error and the fallback salvages `title`, so validate passes). `_ValidYAML_NoMalformedError` already passes.

- [ ] **Step 3: Wire `ExtractionErrors` into `runValidateFiles`**

In `cmd/rootline/validate.go`, locate the block in `runValidateFiles` (around line 100-106):

```go
		// Structural integrity check: detect multiple YAML documents.
		structErrs := rules.ValidateStructure(content, file)
		var errs []rules.ValidationError

		// Validate
		errs = append(errs, rules.Validate(ctx, record, effective)...)
		errs = append(errs, structErrs...)
```

Add the extraction-errors append immediately after the `rules.Validate` line:

```go
		// Structural integrity check: detect multiple YAML documents.
		structErrs := rules.ValidateStructure(content, file)
		var errs []rules.ValidationError

		// Validate
		errs = append(errs, rules.Validate(ctx, record, effective)...)
		errs = append(errs, rules.ExtractionErrors(record)...)
		errs = append(errs, structErrs...)
```

- [ ] **Step 4: Wire `ExtractionErrors` into `runValidateAll`**

In `cmd/rootline/validate.go`, locate the per-record loop in `runValidateAll` (around line 178):

```go
		errs := rules.Validate(ctx, rec, effective)

		results = append(results, rules.NewValidationResult(rec.Path, errs))
```

Change it to append extraction errors before building the result:

```go
		errs := rules.Validate(ctx, rec, effective)
		errs = append(errs, rules.ExtractionErrors(rec)...)

		results = append(results, rules.NewValidationResult(rec.Path, errs))
```

(`rec` is `*extract.Record` from the index scanner, which preserves `rec.Errors` — index.go:165-170 stores the full record and returns nil `extractErr` for malformed YAML.)

- [ ] **Step 5: Run the integration tests to verify they pass**

Run: `go test ./cmd/rootline/ -run TestValidate_MalformedYAML -v`
Expected: PASS (4 tests) — single-file and `--all` now fail with `malformed_yaml`; report-everything shows both `malformed_yaml` and the `estado` schema error; valid file stays clean.

- [ ] **Step 6: Run full suite, check, coverage, and the repo's own docs gate**

Run: `just test && just check`
Expected: all packages PASS with `-race`; gofmt clean; lint clean; build OK.

Run: `go test ./internal/rules/ ./cmd/rootline/ -coverprofile=/tmp/cov.out && go tool cover -func=/tmp/cov.out | tail -1`
Expected: combined coverage ≥ 85% (per-package gate enforced separately by `just coverage-check`).

Run (use the freshly-built code, NOT the installed `rootline` binary which predates this change): `go run ./cmd/rootline validate --all docs/roadmap >/tmp/docsgate.txt 2>&1; echo "exit=$?"; grep -c malformed_yaml /tmp/docsgate.txt || true`
Expected: exit=0 and 0 `malformed_yaml` matches — the repo's own roadmap docs have valid YAML and must not regress under the new error. (Use `docs/roadmap`; `docs/epics` does not exist in this repo despite older CLAUDE.md references.) If `malformed_yaml` appears, a real malformed doc in the repo is a legitimate find to fix in this task; quote the offending title/value and fix by quoting it.

- [ ] **Step 7: Commit**

```bash
git add cmd/rootline/validate.go cmd/rootline/validate_yaml_test.go
git commit -m "feat(validate): surface malformed YAML frontmatter as a blocking error"
```

---

## Self-Review (completed by plan author)

**Spec coverage:**
- `rules.ExtractionErrors` helper (spec §"Helper nuevo") → Task 1. ✓
- Exact ValidationError shape (Rule/Field/Severity/Source/Message/Suggestion) → Task 1 Step 3 + Global Constraints. ✓
- Wire into `runValidateFiles` (spec §"Wire") → Task 2 Step 3. ✓
- Wire into `runValidateAll` (spec §"Wire", the `--all`/wiki path) → Task 2 Step 4. ✓
- "Report everything" (malformed_yaml + schema errors) → Task 2 Step 1 `TestValidate_MalformedYAML_ReportsEverything`. ✓
- Fallback + extract.go untouched (spec §"Sin cambios") → no task modifies them; Global Constraints forbid it. ✓
- Default-blocking, no new flag (spec §"Decisión"/§"Fuera de scope") → Severity "error", `--strict` untouched. ✓
- Regression on valid files → Task 2 `TestValidate_ValidYAML_NoMalformedError`. ✓
- DoD: just check/test, coverage ≥85%, repo docs gate → Task 2 Step 6. ✓
- Out of scope: removing fallback, `--strict` flag, read-path commands, `multiple_yaml_documents` in `--all` → no task touches them. ✓

**Placeholder scan:** none — every step has full code or exact commands.

**Type consistency:** `ExtractionErrors(rec *extract.Record) []ValidationError` is defined in Task 1 and consumed with the same signature in Task 2 Steps 3-4. `ValidationError` field names match the struct (validate.go:16-23). `extract.ExtractionError{Line, Message}` and `extract.Record{Path, Errors}` match extract.go:30-57.
