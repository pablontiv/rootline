# Rootline Fase 3 — Per-record resolution (fix `source:` match-scope leak in enrich) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Make the derive/enrich/aggregate pipeline resolve each record's effective `.stem` per-record (via `ResolveForRecord`) instead of per-directory (`DefaultResolver`/merge-only), so that `source:`-extracted schema fields scoped by `match:` populate ONLY the records they match — fixing a cross-scope leak in `EnrichBuiltins`.

**Architecture:** The pipeline's `StemResolver` is currently `func(dir string) *rules.StemFile` returning the merged stem (no match filtering). Only `EnrichBuiltins` (`internal/derive/enrich.go:47-67`) consumes `eff.Schema` — it applies each field's `Extract` (`source:`) rule. With the unfiltered resolver, a field carrying both `source:` and `match:` is extracted for EVERY record in the directory, including ones it does not match. We widen `StemResolver` to `func(dir, recordPath string)` and switch `DefaultResolver` to `rules.ResolveForRecord`, which match-filters `Schema` per record (and leaves `Derive`/`Aggregate` intact). `DeriveRecord` uses only `effective.Derive` (`internal/derive/record.go:50`) and `AggregateAll` only `eff.Aggregate`, so those are switched for consistency but are behavioral no-ops; the observable fix is in enrich.

**Tech Stack:** Go 1.25+, expr-lang, just.

## Global Constraints

- `just check` + `just test` (`-race`) + `just coverage-check` (≥85%) exit 0 at the end of every task.
- Conventional commits; NEVER `Co-Authored-By` / AI attribution.
- Verified facts (do not re-derive): `rules.ResolveForRecord(dir, recordPath string) (*rules.StemFile, error)` (`internal/rules/hierarchy.go:7`) returns the merged stem with `Schema` match-filtered via `FilterSchemaByMatch` and `Derive`/`Aggregate` UNFILTERED. `FilterSchemaByMatch` (`internal/rules/match.go:9-12`): fields without `Match` apply everywhere; fields with `Match` apply only to matching record paths. Only `EnrichBuiltins` reads `eff.Schema`; `DeriveRecord`/`AggregateAll` do not. Reference for `.stem` `match:` syntax in a test fixture: `internal/rules/hierarchy_test.go` `TestResolveForRecord_V2MatchFiltering` (~`:29-106`).

**Setup (once):** On `master`. Create a feature branch before Task 1: `git -C /Users/Shared/harness/rootline checkout -b fix/per-record-enrich-resolution`.

---

### Task 1: Switch the pipeline to per-record resolution; prove the enrich match-scope fix (TDD)

**Files:**
- Modify: `internal/derive/pipeline.go` (`StemResolver` type `:11-12`, `DeriveAll` resolver call `:54`, `DefaultResolver` `:77-86`, the rationale comment `:19-24`)
- Modify: `internal/derive/enrich.go` (resolver call `:37`)
- Modify: `internal/derive/aggregate.go` (resolver call `:64`)
- Modify (test closures → new signature): `internal/derive/pipeline_test.go` (`:23,:53,:73,:87`, and the `resolver(dir)` call `:118`), `internal/derive/enrich_test.go` (`:18,:43,:77,:210,:237,:263`), `internal/derive/aggregate_test.go` (`:23,:49,:76,:105,:128,:147,:182,:208`)
- Test (new): `internal/derive/enrich_test.go` (add the match-scope leak test)

**Interfaces:**
- Produces: `StemResolver = func(dir, recordPath string) *rules.StemFile`; `DefaultResolver()` returns a closure calling `rules.ResolveForRecord(dir, recordPath)`.

- [ ] **Step 1: Write the failing enrich match-scope test**

Add to `internal/derive/enrich_test.go` a test that builds a real-FS fixture: a `.stem` with a field that has BOTH `source:` (body-section extraction) AND `match:` scoping it to only some records, plus two records — one matching, one not. Assert (the FIXED behavior) that the NON-matching record does NOT get the source-derived field, while the matching one does. Use `EnrichBuiltins(ctx, records, root, DefaultResolver())` (the real default resolver, so the test exercises the switch). Mirror the `.stem` `match:` syntax from `internal/rules/hierarchy_test.go:TestResolveForRecord_V2MatchFiltering` and the fixture/helper style of the existing `enrich_test.go` tests. Example shape:

```go
func TestEnrichBuiltins_SourceFieldRespectsMatchScope(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil { t.Fatal(err) }
	// .stem: "resumen" is source-extracted from a body section, scoped to F* records only.
	stem := "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  resumen:\n    type: string\n    source: body.section[\"Resumen\"]\n    match: [\"F*\"]\n"
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stem), 0644)

	fMatch := &extract.Record{Path: "F01.md", Frontmatter: map[string]any{}, Derived: map[string]any{},
		Sections: map[string]string{"Resumen": "feature summary"}}
	tNoMatch := &extract.Record{Path: "T01.md", Frontmatter: map[string]any{}, Derived: map[string]any{},
		Sections: map[string]string{"Resumen": "task summary"}}

	EnrichBuiltins(context.Background(), []*extract.Record{fMatch, tNoMatch}, dir, DefaultResolver())

	if _, ok := fMatch.Derived["resumen"]; !ok {
		t.Errorf("matching record F01 should have source-derived 'resumen'")
	}
	if v, ok := tNoMatch.Derived["resumen"]; ok {
		t.Errorf("non-matching record T01 must NOT get 'resumen' (match-scope leak), got %v", v)
	}
}
```
(Adapt field names — `Sections`, `Derived` — and the section-extraction source string to the real `Record` shape and the real `Extract` format used in `enrich.go:54-61`; read those before writing. If `body.section[...]` is the wrong syntax, use whatever `enrich.go` parses.)

- [ ] **Step 2: Run it — confirm it FAILS**

Run: `go test ./internal/derive/ -run TestEnrichBuiltins_SourceFieldRespectsMatchScope -v`
Expected: FAIL on the second assertion — current `DefaultResolver` returns the unfiltered schema, so `T01` (non-matching) also gets `resumen`.

- [ ] **Step 3: Widen `StemResolver` and switch `DefaultResolver`**

In `internal/derive/pipeline.go`:
```go
// :11-12
type StemResolver func(dir, recordPath string) *rules.StemFile

// :77-86
func DefaultResolver() StemResolver {
	return func(dir, recordPath string) *rules.StemFile {
		stem, err := rules.ResolveForRecord(dir, recordPath)
		if err != nil {
			return nil
		}
		return stem
	}
}
```

- [ ] **Step 4: Pass the record path at the three call sites**

- `internal/derive/pipeline.go` `DeriveAll` `:54`: change `eff = resolver(dir)` to `eff := resolver(dir, rec.Path)` and remove the per-directory `stemCache` (resolution is now per record; the cache keyed by dir is no longer valid — resolve per record). Keep the `len(eff.Derive) == 0` guard.
- `internal/derive/enrich.go` `:37`: change `eff = resolver(dir)` to `eff := resolver(dir, rec.Path)`; remove its per-directory `stemCache` similarly.
- `internal/derive/aggregate.go` `:64`: change `eff = resolver(dir)` to `eff := resolver(dir, idx.Path)`; remove its per-directory `stemCache`.

(Removing the directory caches is required because two records in the same directory can now resolve to different schemas. If profiling later shows this matters, a per-record cache can be added — do NOT add it now, YAGNI.)

- [ ] **Step 5: Update every custom-resolver test closure to the new signature**

In `pipeline_test.go`, `enrich_test.go`, `aggregate_test.go`, change each `func(dir string) *rules.StemFile` to `func(dir, recordPath string) *rules.StemFile` (the bodies that `return stem`/`return nil` are unchanged — they ignore the args). At `pipeline_test.go:118`, change `_ = resolver(dir)` to `_ = resolver(dir, "")`. The exact lines: pipeline_test.go 23/53/73/87/118; enrich_test.go 18/43/77/210/237/263; aggregate_test.go 23/49/76/105/128/147/182/208.

- [ ] **Step 6: Run the new test — confirm it PASSES**

Run: `go test ./internal/derive/ -run TestEnrichBuiltins_SourceFieldRespectsMatchScope -v`
Expected: PASS.

- [ ] **Step 7: Full suite + coverage**

Run: `just check` → exit 0 (signature change compiles everywhere — grep for any missed `func(dir string) *rules.StemFile` if it fails: `rg "func\(dir string\) \*rules\.StemFile"`).
Run: `just test` → exit 0 (existing derive/aggregate tests must still pass — those fields have no `match:`, so per-record resolution returns the same schema).
Run: `just coverage-check` → exit 0.

- [ ] **Step 8: Update the rationale comment**

Replace the now-inaccurate `internal/derive/pipeline.go:19-24` comment. New comment: resolution is per-record via `ResolveForRecord` so that `match:`-scoped schema fields (and their `source:` extraction in enrich) apply only to records they match; `Derive`/`Aggregate` rules are unaffected by match filtering and behave identically per record.

- [ ] **Step 9: Commit**

```bash
git add internal/derive/
git commit -m "fix(derive): resolve effective stem per-record so source fields respect match scope"
```

---

### Task 2: Docs + spec sync

**Files:**
- Modify: `CLAUDE.md` (the `internal/derive/` paragraph, if it describes directory-level resolution)
- Modify: `docs/superpowers/specs/2026-06-24-rootline-sincerizacion-gaps-design.md` (Fase 3 section)

- [ ] **Step 1: CLAUDE.md** — if the `internal/derive/` description (or any resolution description) states directory-level/merge-only resolution, update it to: derive/enrich/aggregate resolve the effective stem per-record via `ResolveForRecord`, so `match:`-scoped `source:` fields apply only to matching records. If CLAUDE.md does not describe this, make no change and note so.

- [ ] **Step 2: Spec sync** — in the Fase 3 section of the design spec, correct the premise: the real fix is `enrich`'s `source:` extraction respecting `match:` scope (not "wrong derived/aggregated values" — `DeriveRecord`/`AggregateAll` use only `Derive`/`Aggregate`, which match-filtering does not touch, so switching them is a consistency no-op). Note the verified evidence (`record.go:50`, `enrich.go:47-67`, `hierarchy.go` filters Schema only).

- [ ] **Step 3: Verify + commit**

Run: `just check` + `just test` → exit 0.
```bash
git add CLAUDE.md docs/superpowers/specs/2026-06-24-rootline-sincerizacion-gaps-design.md
git commit -m "docs: document per-record stem resolution; correct Fase 3 scope"
```

---

## Self-Review (spec coverage — Fase 3)

- Per-record resolution (`StemResolver` widened, `DefaultResolver`→`ResolveForRecord`, three call sites, caches removed) → Task 1 ✓
- Real behavioral fix proven by TDD (enrich source field respects match scope) → Task 1 Steps 1-6 ✓
- All custom-resolver test closures updated → Task 1 Step 5 ✓
- Rationale comment corrected → Task 1 Step 8 ✓
- Docs/spec corrected to accurate scope → Task 2 ✓

No placeholders: the failing test is given (adapt field names to the real `Record`/`Extract` shapes — read `enrich.go:47-67` and `extract.Record` first); every call-site and test-closure line is enumerated; verification uses `just check`/`just test`/`just coverage-check`. The fix is observable (enrich) — derive/aggregate switches are consistency no-ops, stated honestly.
```
