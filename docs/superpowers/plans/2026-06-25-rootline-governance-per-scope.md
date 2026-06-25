# Governance Per-Scope Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make rootline's governance/inference detectors (`DetectValidationGaps`, `FilterCoveredInferences`, and `schema propose --incremental`) descend into per-directory `.stem` scopes instead of using a single root-merged stem, so they work correctly on multi-`.stem` repos.

**Architecture:** A new internal `internal/infer/scopes.go` groups scanned records by their closest `.stem` (effective stem), via an injected `StemResolver` (Fase-3 style, with a per-directory cache). The two infer detectors run per-scope; the analyze/schema callers pass `DefaultStemResolver()`. The permissive/core per-record pipeline is untouched.

**Tech Stack:** Go 1.25+, standard `testing`. Reuses `rules.WalkUp`, `rules.MergeStemFiles`, `extract.Record`, `infer.isCovered`.

## Global Constraints

- Go 1.25+; no new third-party dependencies.
- `just check` (gofmt + golangci-lint + build) must pass; `just test` runs with `-race`, 0 failures.
- Coverage floor ≥ 85% for `internal/infer` and `cmd/rootline` (`.coverage-floors.toml`).
- Conventional Commits; `.stem` fixtures use `version: 2`.
- `SchemaField.Source` is the `.stem` path that declared a field (already populated; used by current inferences). It is the dedup key.
- Group key = the leaf-most `.stem` path: `entries[len-1].Path` from `rules.WalkUp(dir)` (same convention as `internal/infer/schema_coverage.go:55`).
- Behavior MUST be a no-op for single-`.stem` repos (one scope group → identical output to today).
- Do NOT touch the per-record core (derive/aggregate/enrich, validate, ValidateStemHealth, DetectMissingSchemata, DetectStructural) or the out-of-scope sites (link-schema, query --sort, tree render).

---

## File Structure

- **Create** `internal/infer/scopes.go` — `StemResolver`, `ScopeGroup`, `GroupByScope`, `DefaultStemResolver`.
- **Create** `internal/infer/scopes_test.go`.
- **Modify** `internal/infer/validation_gaps.go` — extract `detectGapsForScope`, new per-scope `DetectValidationGaps` + dedup.
- **Modify** `internal/infer/validation_gaps_test.go` — repoint existing tests to `detectGapsForScope`; add multi-scope test.
- **Modify** `internal/infer/delta.go` — per-scope `FilterCoveredInferences` (keep `isCovered`).
- **Modify** `internal/infer/delta_test.go` — update `FilterCoveredInferences` tests to the new signature.
- **Modify** `cmd/rootline/analyze.go` — wire `DefaultStemResolver` into both detectors; drop the root-merged `stem` var.
- **Modify** `cmd/rootline/schema.go` — apply per-scope coverage to `schema propose --incremental`.
- **Create** a shared multi-`.stem` test fixture + e2e tests (Task 5).

---

## Task 1: `scopes.go` — grouping helper + resolver

**Files:**
- Create: `internal/infer/scopes.go`
- Test: `internal/infer/scopes_test.go`

**Interfaces:**
- Produces: `StemResolver func(dir string) (*rules.StemFile, string)`; `ScopeGroup{Key string; Stem *rules.StemFile; Records []*extract.Record}`; `GroupByScope(records []*extract.Record, root string, resolve StemResolver) []ScopeGroup`; `DefaultStemResolver() StemResolver`.

- [ ] **Step 1: Write the failing tests**

Create `internal/infer/scopes_test.go`:

```go
package infer

import (
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestGroupByScope_BucketsByClosestStem(t *testing.T) {
	conceptsStem := &rules.StemFile{Schema: map[string]rules.SchemaField{"kind": {Type: "enum"}}}
	sourcesStem := &rules.StemFile{Schema: map[string]rules.SchemaField{"tipo": {Type: "enum"}}}
	resolve := func(dir string) (*rules.StemFile, string) {
		if filepath.Base(dir) == "concepts" {
			return conceptsStem, "concepts/.stem"
		}
		return sourcesStem, "sources/.stem"
	}
	records := []*extract.Record{
		{Path: "concepts/a.md"}, {Path: "concepts/b.md"}, {Path: "sources/p.md"},
	}
	groups := GroupByScope(records, ".", resolve)
	if len(groups) != 2 {
		t.Fatalf("expected 2 scope groups, got %d", len(groups))
	}
	byKey := map[string]ScopeGroup{}
	for _, g := range groups {
		byKey[g.Key] = g
	}
	if len(byKey["concepts/.stem"].Records) != 2 || len(byKey["sources/.stem"].Records) != 1 {
		t.Errorf("wrong bucketing: %+v", byKey)
	}
}

func TestGroupByScope_NoStem(t *testing.T) {
	resolve := func(dir string) (*rules.StemFile, string) { return nil, "" }
	groups := GroupByScope([]*extract.Record{{Path: "x.md"}}, ".", resolve)
	if len(groups) != 1 || groups[0].Stem != nil || groups[0].Key != "" {
		t.Errorf("expected one nil-stem group, got %+v", groups)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/infer/ -run TestGroupByScope -v`
Expected: FAIL — `undefined: GroupByScope`.

- [ ] **Step 3: Implement `scopes.go`**

Create `internal/infer/scopes.go`:

```go
package infer

import (
	"path/filepath"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

// StemResolver returns the effective (merged) stem for a directory and the path
// of the closest (leaf-most) .stem governing it — the scope group key.
// Returns (nil, "") when no .stem governs the directory.
type StemResolver func(dir string) (*rules.StemFile, string)

// ScopeGroup is the set of records governed by one effective .stem.
type ScopeGroup struct {
	Key     string
	Stem    *rules.StemFile
	Records []*extract.Record
}

// GroupByScope buckets records by the closest .stem governing them, preserving
// first-appearance order of scope keys.
func GroupByScope(records []*extract.Record, root string, resolve StemResolver) []ScopeGroup {
	var order []string
	byKey := make(map[string]*ScopeGroup)
	for _, rec := range records {
		dir := filepath.Dir(filepath.Join(root, rec.Path))
		stem, key := resolve(dir)
		g, ok := byKey[key]
		if !ok {
			g = &ScopeGroup{Key: key, Stem: stem}
			byKey[key] = g
			order = append(order, key)
		}
		g.Records = append(g.Records, rec)
	}
	groups := make([]ScopeGroup, 0, len(order))
	for _, k := range order {
		groups = append(groups, *byKey[k])
	}
	return groups
}

// DefaultStemResolver resolves each directory's effective stem via WalkUp +
// MergeStemFiles, caching per directory to avoid re-walking per record. The
// group key is the leaf-most .stem path.
func DefaultStemResolver() StemResolver {
	type entry struct {
		stem *rules.StemFile
		key  string
	}
	cache := make(map[string]entry)
	return func(dir string) (*rules.StemFile, string) {
		if e, ok := cache[dir]; ok {
			return e.stem, e.key
		}
		var e entry
		entries, err := rules.WalkUp(dir)
		if err == nil && len(entries) > 0 {
			e.stem = rules.MergeStemFiles(entries)
			e.key = entries[len(entries)-1].Path
		}
		cache[dir] = e
		return e.stem, e.key
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/infer/ -run TestGroupByScope -v`
Expected: PASS (2 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/infer/scopes.go internal/infer/scopes_test.go
git commit -m "feat(infer): add scope grouping helper for multi-stem governance"
```

---

## Task 2: `DetectValidationGaps` per-scope

**Files:**
- Modify: `internal/infer/validation_gaps.go`
- Modify: `internal/infer/validation_gaps_test.go`
- Modify: `cmd/rootline/analyze.go`

**Interfaces:**
- Consumes: `GroupByScope`, `StemResolver`, `DefaultStemResolver` (Task 1).
- Produces: `DetectValidationGaps(records []*extract.Record, prior []Inference, root string, resolve StemResolver) []Inference`; internal `detectGapsForScope(stem *rules.StemFile, records []*extract.Record, prior []Inference) []Inference`.

- [ ] **Step 1: Refactor the existing function into a per-scope core**

In `internal/infer/validation_gaps.go`, RENAME the current `DetectValidationGaps` to the unexported `detectGapsForScope` (keep its body byte-for-byte; only the name and the doc comment change). Then add the new exported orchestrator:

```go
// DetectValidationGaps runs the per-scope schema-gap checks across every .stem
// scope in the tree (grouped by GroupByScope), deduplicating findings by
// (Type, Field, Source) since inherited fields appear in multiple scopes.
func DetectValidationGaps(records []*extract.Record, prior []Inference, root string, resolve StemResolver) []Inference {
	groups := GroupByScope(records, root, resolve)
	seen := make(map[string]bool)
	var out []Inference
	for _, g := range groups {
		if g.Stem == nil || len(g.Stem.Schema) == 0 {
			continue
		}
		for _, inf := range detectGapsForScope(g.Stem, g.Records, prior) {
			k := inf.Type + "|" + inf.Field + "|" + inf.Source
			if seen[k] {
				continue
			}
			seen[k] = true
			out = append(out, inf)
		}
	}
	return out
}
```

- [ ] **Step 2: Update existing tests + add a multi-scope test (write, then run)**

In `internal/infer/validation_gaps_test.go`, the 7 existing tests call `DetectValidationGaps(stem, records, prior)`. Repoint each to `detectGapsForScope(stem, records, prior)` (identical behavior, single scope) — a mechanical rename of the call in each test. Then ADD:

```go
func TestDetectValidationGaps_MultiScope(t *testing.T) {
	conceptsStem := &rules.StemFile{Schema: map[string]rules.SchemaField{
		"status": {Type: "enum", Values: []string{"a"}, Source: "concepts/.stem"},
	}}
	// sources scope declares an enum WITHOUT values → a gap that root-only would miss.
	sourcesStem := &rules.StemFile{Schema: map[string]rules.SchemaField{
		"tipo": {Type: "enum", Source: "sources/.stem"},
	}}
	resolve := func(dir string) (*rules.StemFile, string) {
		if filepath.Base(dir) == "sources" {
			return sourcesStem, "sources/.stem"
		}
		return conceptsStem, "concepts/.stem"
	}
	records := []*extract.Record{{Path: "concepts/a.md"}, {Path: "sources/p.md"}}

	got := DetectValidationGaps(records, nil, ".", resolve)

	found := false
	for _, inf := range got {
		if inf.Type == "enum_without_values" && inf.Field == "tipo" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected enum_without_values for sources/.stem:tipo, got %+v", got)
	}
}
```

Add `"path/filepath"` and the `rules`/`extract` imports if missing.

Run: `go test ./internal/infer/ -run 'TestDetectValidationGaps' -v`
Expected: PASS — existing tests (now on `detectGapsForScope`) green; the new multi-scope test green.

- [ ] **Step 3: Wire `analyze.go` (validation_gaps only — keep `stem` for now)**

In `cmd/rootline/analyze.go`: DO NOT remove the root-merged `stem` block yet (the incremental block still uses it until Task 3, so removing it now would break the build). Add, near where `records` is obtained (after the `index.Scan` call):

```go
	gapsResolver := infer.DefaultStemResolver()
```

Change ONLY the `validation_gaps` category closure (currently `return infer.DetectValidationGaps(stem, records, prior)`) to:

```go
			return infer.DetectValidationGaps(records, prior, root, gapsResolver)
```

Leave the `prior` construction and the incremental `FilterCoveredInferences(inferences, stem)` block untouched (Task 3 converts it and then removes the `stem` var). `gapsResolver` is now used by validation_gaps; `stem` is still used by the incremental block — both compile.

- [ ] **Step 4: Verify build + tests**

Run: `go build ./... && go test ./internal/infer/ ./cmd/rootline/ -run 'ValidationGaps|Analyze' -count=1`
Expected: builds (both `stem` and `gapsResolver` are referenced); tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/infer/validation_gaps.go internal/infer/validation_gaps_test.go cmd/rootline/analyze.go
git commit -m "feat(infer): resolve validation gaps per-scope in multi-stem trees"
```

---

## Task 3: `FilterCoveredInferences` per-scope

**Files:**
- Modify: `internal/infer/delta.go`
- Modify: `internal/infer/delta_test.go`
- Modify: `cmd/rootline/analyze.go` (incremental call site)

**Interfaces:**
- Consumes: `GroupByScope`, `StemResolver` (Task 1), existing `isCovered`.
- Produces: `FilterCoveredInferences(inferences []Inference, records []*extract.Record, root string, resolve StemResolver) []Inference`.

- [ ] **Step 1: Write the per-scope behavior tests**

In `internal/infer/delta_test.go`: the `TestIsCovered_*` tests are UNCHANGED (isCovered is untouched). UPDATE the `TestFilterCoveredInferences_*` tests to the new signature using a single-scope resolver helper. Add this helper and a multi-scope test:

```go
func singleScope(stem *rules.StemFile) StemResolver {
	return func(dir string) (*rules.StemFile, string) { return stem, "/.stem" }
}

func TestFilterCoveredInferences_MultiScope(t *testing.T) {
	conceptsStem := &rules.StemFile{Schema: map[string]rules.SchemaField{
		"status": {Type: "enum", Values: []string{"a", "b"}},
	}}
	sourcesStem := &rules.StemFile{Schema: map[string]rules.SchemaField{}}
	resolve := func(dir string) (*rules.StemFile, string) {
		if filepath.Base(dir) == "sources" {
			return sourcesStem, "sources/.stem"
		}
		return conceptsStem, "concepts/.stem"
	}
	records := []*extract.Record{
		{Path: "concepts/a.md", Frontmatter: map[string]any{"status": "a"}},
		{Path: "sources/p.md", Frontmatter: map[string]any{"ref": "x"}},
	}
	inferences := []Inference{
		{Type: "enum_values", Field: "status"}, // covered in concepts scope
		{Type: "field_type", Field: "ref", Value: "string"}, // not covered anywhere
	}
	got := FilterCoveredInferences(inferences, records, ".", resolve)
	if len(got) != 1 || got[0].Field != "ref" {
		t.Errorf("expected only 'ref' to survive, got %+v", got)
	}
}
```

For each existing `TestFilterCoveredInferences_*` test, change the call from
`FilterCoveredInferences(inferences, stem)` to
`FilterCoveredInferences(inferences, records, ".", singleScope(stem))`, where `records` is a minimal
`[]*extract.Record` whose frontmatter contains the inference's field so the scope is "relevant"
(e.g. `[]*extract.Record{{Path: "a.md", Frontmatter: map[string]any{"<field>": "x"}}}`). Preserve each
test's expected outcome.

- [ ] **Step 2: Run to verify the new test fails**

Run: `go test ./internal/infer/ -run TestFilterCoveredInferences_MultiScope -v`
Expected: FAIL — signature mismatch / not yet per-scope.

- [ ] **Step 3: Implement per-scope `FilterCoveredInferences`**

In `internal/infer/delta.go`, REPLACE `FilterCoveredInferences` (keep `isCovered` unchanged) with:

```go
// FilterCoveredInferences removes inferences already covered by the existing
// per-scope .stem schema. An inference for a field is covered iff EVERY scope
// that is relevant to that field (the field is in the scope's schema or in one
// of its records) covers it via isCovered. Conservative: if any relevant scope
// is uncovered, the inference is kept. Reduces to the single-stem behavior when
// there is one scope.
func FilterCoveredInferences(inferences []Inference, records []*extract.Record, root string, resolve StemResolver) []Inference {
	groups := GroupByScope(records, root, resolve)
	var deltas []Inference
	for _, inf := range inferences {
		if coveredEverywhere(inf, groups) {
			continue
		}
		deltas = append(deltas, inf)
	}
	return deltas
}

func coveredEverywhere(inf Inference, groups []ScopeGroup) bool {
	relevant := false
	for _, g := range groups {
		if g.Stem == nil {
			continue
		}
		_, inSchema := g.Stem.Schema[inf.Field]
		inRecords := false
		for _, rec := range g.Records {
			if _, ok := rec.Frontmatter[inf.Field]; ok {
				inRecords = true
				break
			}
		}
		if !inSchema && !inRecords {
			continue
		}
		relevant = true
		if !isCovered(inf, g.Stem) {
			return false
		}
	}
	return relevant
}
```

- [ ] **Step 4: Wire the analyze incremental call site**

In `cmd/rootline/analyze.go`, change the incremental block (currently `FilterCoveredInferences(inferences, stem)`) to reuse `gapsResolver` from Task 2:

```go
		inferences = infer.FilterCoveredInferences(inferences, records, root, gapsResolver)
```

Then REMOVE the now-orphaned root-merged `stem` block (the comment + `var stem *rules.StemFile` + the `WalkUp`/`MergeStemFiles` lines, ~analyze.go:90-98). Run `rg -n '\bstem\b' cmd/rootline/analyze.go` to confirm no remaining references to that variable (do not match `gapsResolver`/`linkSchema`/field names).

- [ ] **Step 5: Run tests + build**

Run: `go build ./... && go test ./internal/infer/ ./cmd/rootline/ -run 'FilterCovered|IsCovered|Analyze' -count=1`
Expected: builds; all green.

- [ ] **Step 6: Commit**

```bash
git add internal/infer/delta.go internal/infer/delta_test.go cmd/rootline/analyze.go
git commit -m "feat(infer): per-scope incremental coverage filtering"
```

---

## Task 4: `schema propose --incremental` per-scope

**Files:**
- Modify: `cmd/rootline/schema.go`
- Test: `cmd/rootline/schema_test.go` (or the existing schema test file)

**Interfaces:**
- Consumes: `infer.FilterCoveredInferences` (Task 3) / `infer.DefaultStemResolver`.

- [ ] **Step 1: Read the current incremental path**

Read `cmd/rootline/schema.go` around the `schema propose` handler (the `--incremental` filtering, ~lines 130-180, including `generateSchemaProposals(...)` and its `existingStem`/`incremental` handling). Identify exactly where proposals are filtered against the root-merged `existingStem`.

- [ ] **Step 2: Write a failing multi-scope test**

Add a test (in the schema command's test file) that builds a temp multi-`.stem` fixture (sibling `concepts/.stem` defining a covered field + `sources/.stem`), runs `schema propose --incremental <dir>`, and asserts a proposal already covered by a SUBTREE `.stem` is filtered out (today it is NOT, because filtering uses root-only). Use the existing command test harness (`runCmd`/`executeValidate`-style). Run it; expect FAIL.

- [ ] **Step 3: Apply per-scope filtering**

Replace the root-merged `existingStem` filtering in the incremental path with the per-scope call: build `resolve := infer.DefaultStemResolver()` and filter the generated proposals/inferences with `infer.FilterCoveredInferences(inferences, records, root, resolve)` (mirroring analyze.go). If `generateSchemaProposals` does its own coverage internally against `existingStem`, refactor it to take `records + resolve` and delegate to `FilterCoveredInferences`, or filter after generation. Keep proposals that no scope covers.

- [ ] **Step 4: Run the test + build**

Run: `go build ./... && go test ./cmd/rootline/ -run 'Schema' -count=1`
Expected: builds; the new multi-scope test passes; existing schema tests stay green.

- [ ] **Step 5: Commit**

```bash
git add cmd/rootline/schema.go cmd/rootline/schema_test.go
git commit -m "feat(schema): per-scope incremental filtering for schema propose"
```

---

## Task 5: Shared fixture + e2e + DoD

**Files:**
- Create: a multi-`.stem` fixture (temp-dir helper in tests, or `internal/infer/testdata/`).
- Test: `cmd/rootline/governance_multistem_test.go` (e2e for the 3 commands).

- [ ] **Step 1: Write the e2e tests**

Create `cmd/rootline/governance_multistem_test.go` building a temp dir that mirrors the wiki (NO root `.stem`):

```
<dir>/.git/                      (marker for WalkUp boundary)
<dir>/concepts/.stem             (status: enum [a,b], required)
<dir>/concepts/a.md, b.md
<dir>/sources/.stem              (tipo: type enum, NO values  ← gap)
<dir>/sources/p.md, q.md
```

Assertions via the command harness:
- `rootline analyze <dir>` (JSON) contains `enum_without_values` for `tipo` (today: absent).
- `rootline analyze --incremental <dir>` filters proposals covered by the subtree `.stem`s.
- `rootline schema propose --incremental <dir>` likewise.
- A single-`.stem` temp dir (root `.stem` only) yields the SAME analyze output as before (no-op regression guard).

- [ ] **Step 2: Run e2e**

Run: `go test ./cmd/rootline/ -run 'Governance' -v`
Expected: PASS.

- [ ] **Step 3: Full suite, check, coverage**

Run: `just test && just check`
Expected: all packages PASS with `-race`; gofmt clean; lint clean; build OK.

Run: `go test ./internal/infer/ ./cmd/rootline/ -coverprofile=/tmp/cov.out && go tool cover -func=/tmp/cov.out | tail -1`
Expected: combined ≥ 85% (per-package gate via `just coverage-check`).

- [ ] **Step 4: Regression + real-repo manual check**

Run: `go run ./cmd/rootline analyze --all docs/roadmap >/tmp/g.txt 2>&1; echo exit=$?`
Expected: exit 0 (no regression on the repo's own docs).

Run: `go run ./cmd/rootline analyze /Users/Shared/wiki/wiki -o json | rg -c 'validation_gaps|enum_without_values' || true`
Expected: now reports governance findings for the wiki's subtree `.stem` files (was 0 before).

- [ ] **Step 5: Commit**

```bash
git add cmd/rootline/governance_multistem_test.go
git commit -m "test(infer): e2e governance on multi-stem fixture + wiki regression guard"
```

---

## Self-Review (completed by plan author)

**Spec coverage:** scopes helper → T1; validation_gaps per-scope + dedup → T2; incremental per-scope (analyze) → T3; schema propose --incremental → T4; fixture + e2e + no-op regression + wiki check → T5. ✓

**Placeholder scan:** T4 Step 1/3 are intentionally investigative (the schema incremental path must be read before editing) — they name the exact location and the delegation target (`FilterCoveredInferences`); not a placeholder.

**Type consistency:** `StemResolver func(dir string) (*rules.StemFile, string)`, `GroupByScope(records, root, resolve)`, `DetectValidationGaps(records, prior, root, resolve)`, `FilterCoveredInferences(inferences, records, root, resolve)` are used identically across tasks and call sites. `isCovered` and `detectGapsForScope` are internal and unchanged in behavior.
