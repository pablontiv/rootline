# Code Context

## Files Retrieved
1. `internal/rules/discovery.go` (lines 1-62) - `.stem` walk-up discovery; returns entries root-to-leaf.
2. `internal/rules/merge.go` (lines 1-152) - effective schema merge semantics and source tracking.
3. `internal/rules/hierarchy.go` (lines 1-31) - `ResolveForRecord` merge + match-scoped schema filtering.
4. `internal/rules/match.go` (lines 1-92) - per-record schema field filtering by path components.
5. `internal/rules/stem_cache.go` (lines 1-88) - cached walk-up variant with different no-git behavior.
6. `internal/index/index.go` (lines 24-112) and `internal/index/scope.go` (lines 1-24) - scan-time scope resolver and `scope.match` filtering.
7. `cmd/rootline/analyze.go` (lines 45-180, 221-239) - analyze scan, root effective stem use, detector wiring.
8. `internal/infer/schema_coverage.go` (lines 1-81) - `missing_schema` vs `implicit_schema` detector and closest stem calculation.
9. `internal/infer/validation_gaps.go` (lines 1-99) and `internal/infer/delta.go` (lines 1-75) - governance/incremental filtering against one effective stem.
10. `cmd/rootline/apply.go` (lines 31-110) and `internal/infer/apply.go` (lines 1-189, 321-418) - apply pre-scaffold, stem target selection, schema/data writes.
11. `internal/infer/scaffold.go` (lines 1-58) - minimal local `.stem` scaffolding for missing schema.
12. `cmd/rootline/migrate.go` (lines 45-217, 321-444), `internal/migrate/source.go` (lines 1-91), `internal/migrate/split.go` (lines 1-269), `internal/migrate/scaffold.go` (lines 1-164), `internal/migrate/rename.go` (lines 1-313) - diff/split/scaffold/rename mutation flows.
13. `cmd/rootline/fix.go` (lines 66-260, 261-394) and `internal/fix/fix.go` (lines 22-125, 247-352, 581-628) - single/all fix flows, proposal application, `.stem` writes.
14. `internal/proposal/proposal.go` (lines 70-180, 214-520, 647-721), `internal/proposal/propagate.go` (lines 1-111), `internal/proposal/stem_health.go` (lines 1-76) - proposal generation and target paths.
15. `cmd/rootline/set.go` (lines 102-327) - direct mutation uses effective schema for section/enum validation and rollback.
16. `internal/derive/pipeline.go` (lines 1-83), `internal/derive/enrich.go` (lines 1-55), `internal/derive/aggregate.go` (lines 1-145) - derive/aggregate use default unfiltered effective stems per directory.

## Key Code

- Discovery order is root-to-leaf:

```go
// internal/rules/discovery.go:12-16,58-61
// Returns entries ordered root-to-leaf (ready for top-down merge).
...
slices.Reverse(entries)
return entries, nil
```

- Effective merge tracks source on schema fields, but child field replacement is whole-field except severity tightening:

```go
// internal/rules/merge.go:31-47,64-79
result.Schema = mergeSchemaFields(result.Schema, s.Schema, path)
...
for k, v := range child {
    v.Source = source
    if parentField, exists := out[k]; exists { v = mergeFieldSeverity(parentField, v) }
    out[k] = v
}
```

- Record-level effective schema is only produced by `ResolveForRecord`; it applies `match` filtering after merge:

```go
// internal/rules/hierarchy.go:7-30
entries, err := WalkUp(dir)
merged := MergeStemFiles(entries)
filtered := FilterSchemaByMatch(ptrSchema, recordPath)
merged.Schema = effective
```

- Analyze scans with per-directory merged scope but uses one stem at report root for governance and incremental filtering:

```go
// cmd/rootline/analyze.go:62-70,87-91,139-150,166-168
resolver := func(dir string) *rules.StemFile { ... return rules.MergeStemFiles(entries) }
records, err := index.Scan(ctx, root, reg, index.WithScopeResolver(resolver))
...
entries, walkErr := rules.WalkUp(root); stem = rules.MergeStemFiles(entries)
...
infer.DetectMissingDomains(stem)
infer.DetectValidationGaps(stem, records, prior)
...
inferences = infer.FilterCoveredInferences(inferences, stem)
```

- Schema coverage correctly uses closest stem for implicit-schema messaging:

```go
// internal/infer/schema_coverage.go:52-71
closest := entries[len(entries)-1].Path
... if depth >= 2 { Type: "implicit_schema", Source: dir, Value: closest }
```

- Apply says closest but selects `entries[0]` (root-most), then writes schema changes ignoring `--dry-run`:

```go
// cmd/rootline/apply.go:48-58,60-67,86-90
infer.ScaffoldSchema(inf.Source) // no dry-run guard
...
// Use the closest stem file.
stemPath := entries[0].Path
...
schemaResult, err := infer.ApplySchemaInferences(stemPath, schemaInferences)
```

- `ApplySchemaInferences` mutates one concrete `.stem` and only existing field nodes/properties:

```go
// internal/infer/apply.go:21-32,72-84,341-418
func ApplySchemaInferences(stemPath string, inferences []ReportInference) ...
...
os.WriteFile(stemPath, out, 0o644)
...
fieldNode := findSchemaFieldNode(doc, inf.Field); if fieldNode == nil { return false }
```

- `fix --all` validates each record with its own effective schema, then collapses proposals to the "richest" schema:

```go
// cmd/rootline/fix.go:183-211
allErrs[rec.Path] = errs
...
if effective == nil || len(s.Schema) > len(effective.Schema) { effective = s }
report := proposal.Analyze(records, effective, allErrs)
```

- `fix` enum extension walks root-to-leaf and breaks on first matching enum, so root-most matching `.stem` wins:

```go
// internal/fix/fix.go:247-260
for _, e := range entries {
    if sf, ok := e.Stem.Schema[p.Field]; ok && sf.Type == "enum" {
        stemPath = e.Path
        break
    }
}
```

## Architecture

- Normal read/validation path: `WalkUp(target)` -> root-to-leaf entries -> `MergeStemFiles` -> optional `ResolveForRecord` match filtering -> `Validate` uses schema fields with `Source` from the defining/overriding `.stem`.
- Scan path (`analyze`, `fix --all`, migrate scaffold/rename): `index.Scan(... WithScopeResolver)` calls a resolver per directory. Resolver usually returns merged `.stem`; scan uses only `scope.match` to include/exclude files. It does not use `ResolveForRecord` field filtering.
- Mutation paths diverge:
  - `apply` reads an analyze report, scaffolds every engine-resolved `missing_schema`, then applies all schema inferences to a single selected `.stem` and data corrections to report paths.
  - `fix --all` validates per record but proposal generation takes one representative effective schema, then `fix.ApplyProposals` mutates `.stem`/records sequentially.
  - `migrate --split` reads only `<target>/.stem`, infers hierarchy from unscoped scan, and writes multiple generated `.stem` files.
  - `migrate --rename` scans scoped records and edits every parsed `.stem` under/above root that directly contains the field.
  - `migrate --scaffold` and `set` use `ResolveForRecord` for record-specific section/frontmatter mutation.

## Concrete Bug / Debt Candidates

1. **`apply` root-vs-closest bug (high confidence).** `WalkUp` returns root-to-leaf, but `cmd/rootline/apply.go:64-67` labels `entries[0]` as closest. This targets the root-most `.stem`, not the nearest/local schema. Multi-directory reports and inherited child overrides will be written to the wrong file.

2. **`apply --dry-run` is not safe for schema/scaffold writes (high confidence).** `applyDryRun` is only passed to `ApplyDataCorrections` (`cmd/rootline/apply.go:93-96`). The missing-schema pre-phase always calls `ScaffoldSchema` (`cmd/rootline/apply.go:48-58`) and `ApplySchemaInferences` always writes (`internal/infer/apply.go:79-84`).

3. **Analyze report lacks enough source metadata for safe schema writes (high confidence).** `ReportInference` has `Source`, but `ApplySchemaInferences` ignores it and receives one `stemPath` (`internal/infer/report.go:26-36`, `cmd/rootline/apply.go:86-90`). Inferred schema changes from multiple directories cannot be routed to the `.stem` that owns/should own the field.

4. **`ApplySchemaInferences` cannot create schema fields (medium-high confidence).** Field type, required, default, enum, and sequence updates all require `findSchemaFieldNode` to find an existing field (`internal/infer/apply.go:341-418`). Thus fresh `field_type` inferences for fields not already in `.stem` are silently skipped, except after a separate `missing_schema` scaffold happens to create them.

5. **`fix --all` may generate proposals using the wrong effective schema (medium-high confidence).** It validates per-record with correct effective schemas, but then picks the schema with the most fields for all proposal detectors (`cmd/rootline/fix.go:183-211`). This can mix sibling/child field domains, enum values, and links across directories.

6. **`fix` enum extension writes root-most matching `.stem` (medium-high confidence).** `applyExtendEnum` breaks on the first root-to-leaf matching enum (`internal/fix/fix.go:247-260`). If a child overrides an enum, validation error source would be the child field, but the proposal does not carry that source and the root may be extended instead.

7. **Governance/incremental analyze is rooted to one effective schema (medium confidence).** `DetectMissingDomains`, `DetectValidationGaps`, and `FilterCoveredInferences` receive only `stem` for the report root (`cmd/rootline/analyze.go:87-91,139-150,166-168`). For a scan spanning multiple `.stem` islands, inherited/missing/untyped/required-understatement results may be false positives/negatives.

8. **Partial/non-atomic schema writes (medium confidence).** `migrate --split` writes generated `.stem` files one-by-one (`cmd/rootline/migrate.go:436-440`), `fix.ApplyProposals` mutates schema/data sequentially (`internal/fix/fix.go:33-123`), and `apply` scaffolds before later schema errors (`cmd/rootline/apply.go:48-67`). Failures can leave partial schema state.

9. **Derive/aggregate ignore match-scoped schema filtering (medium confidence).** `DeriveAllSimple`/`AggregateAllSimple` use `DefaultResolver` (merge only) rather than `ResolveForRecord` (`internal/derive/pipeline.go:59-75`, `internal/derive/aggregate.go:45-67`). This may be intended for non-schema `derive/aggregate`, but index detection and aggregate validation can span fields outside a record's match scope.

10. **`StemCache.WalkUp` has different no-git behavior than `rules.WalkUp` (medium confidence).** `rules.WalkUp` errors if no `.git` is found (`internal/rules/discovery.go:45-52`), while cache walk returns `nil, nil` at filesystem root (`internal/rules/stem_cache.go:51-56`). Any future mutation path using cache may silently treat no-git as no schema.

## Start Here

Open `cmd/rootline/apply.go` first. It contains the clearest architecture mismatch: report-wide apply, pre-scaffold side effects, root-vs-closest target selection, and dry-run bypass. Then inspect `internal/fix/fix.go:247-260` and `cmd/rootline/fix.go:183-211` for the same source-routing problem in proposal-based fixes.

## Gaps / Clarification Questions

- Should schema-mutating inferences be applied to the field's defining `.stem` (`SchemaField.Source`), the closest `.stem` for each affected record/directory, or a newly scaffolded local `.stem`?
- Is `implicit_schema` intended to be actionable by engine apply, or always human/agent planning only?
- Should `apply --dry-run` cover scaffold and schema mutations, or is current data-only dry-run intentional?
- For multi-directory analyze reports, should apply batch by inference `Source`/paths, or should analyze emit one report per governance root?
- Should child `.stem` partial overrides be supported, or is whole-field replacement the intended merge contract?
