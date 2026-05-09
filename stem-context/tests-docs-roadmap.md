Note: I did **not** write `/home/shared/rootline/stem-context/tests-docs-roadmap.md` because the task also said “do not edit”; treating artifact creation as a file edit.

# Code Context

## Files Retrieved

1. `docs/roadmap/.stem` (lines 1-35) — actual roadmap schema.
2. `docs/.stem` (lines 1-7) — inherited parent schema.
3. `.claude/skills/roadmap/base.stem` (lines 1-36) — canonical roadmap template.
4. `.claude/roadmap.local.md` (lines 1-17) — configured statuses/root.
5. `.claude/skills/roadmap/SKILL.md` (lines 49-190) — bootstrap, config validation, rootline dependency.
6. `.claude/skills/roadmap/plan-subcommand.md` (lines 45-169) — materialization, `.stem` bootstrap, validate commands.
7. `.claude/skills/roadmap/common-logic.md` (lines 1-95) — canonical structure, no fallback, auto-numbering.
8. `.claude/skills/roadmap/task-guide.md` (lines 45-141) — effective `.stem`, task template/statuses.
9. `.github/workflows/ci.yml` (lines 19-27) and `Justfile` (lines 21-27) — docs validation target.
10. `docs/validate.md` (lines 28-35) — documented stem-health phases.
11. `internal/rules/discovery_test.go` (lines 36-174) — `.stem` walk-up assumptions.
12. `internal/rules/merge_test.go` (lines 48-125, 228-455) — merge/inheritance assumptions.
13. `internal/rules/match_test.go` (lines 7-246) — match/required-match assumptions.
14. `internal/rules/stemhealth_test.go` (lines 20-240, 397-738) — stem-health assumptions.
15. `cmd/rootline/nested_stem_test.go` (lines 13-260) — CLI inheritance assumptions.
16. `cmd/rootline/validate_test.go` (lines 314-428) — validation/stem-health CLI assumptions.
17. `cmd/rootline/describe_test.go` (lines 23-190) — describe/merged schema contract.
18. `internal/e2e/match_hierarchy_test.go` (lines 13-240) — v2 match hierarchy coverage.
19. `internal/index/scope_test.go` (lines 12-219) — scope filtering assumptions.
20. `internal/rules/rules.go` (lines 14-276), `internal/rules/merge.go` (lines 3-150) — relevant implementation contracts.

## Key Findings

### High-confidence likely bugs

- **CI/docs validation target is stale.**  
  `.github/workflows/ci.yml` and `Justfile` validate `docs/epics/`, but this repo has `docs/roadmap/` and no `docs/epics/`. Local command `./rootline validate --all docs/epics/` exits 1 with missing path.  
  Files: `.github/workflows/ci.yml` lines 19-27; `Justfile` lines 21-27.  
  Confidence: high.

- **Roadmap schema allows any `estado`, while roadmap skill expects status enum validation.**  
  Actual `docs/roadmap/.stem` has `estado.type: string` with no values (lines 5-10). Skill canonical `base.stem` has `estado.type: enum` and values `[Pending, Specified, In Progress, Completed, Blocked, On Hold, Obsolete]` (lines 5-11). `SKILL.md` says to verify configured statuses exist in `schema.estado` (lines 157-171), but current effective schema has no enum values.  
  Confidence: high that this hides invalid statuses; product decision needed for fix because parent `docs/.stem` also defines `estado: string`.

- **Validation docs are stale.**  
  `docs/validate.md` says stem health has “8 diagnostics” including `version-deprecated` (lines 28-35). Tests cover additional domain checks and v1 parse rejection, not `version-deprecated` as a live check (`internal/rules/stemhealth_test.go` lines 397-738).  
  Confidence: high.

### Product/API decisions masquerading as bugs

- **Can child `.stem` narrow parent `estado: string` to `enum`?**  
  Stem-health treats child type changes as fail (`internal/rules/stemhealth_test.go` lines 153-188; CLI test lines 343-381). If `docs/roadmap/.stem` changes to enum under `docs/.stem`, validation likely fails unless parent changes or type-consistency semantics change.  
  Classification: product/architecture decision.

- **Does `null` removal apply to `schema` fields?**  
  README says null removes inherited keys (`README.md` lines 106-114), but `mergeSchemaFields` uses `map[string]SchemaField`, and test explicitly says schema null removal is not supported, using derive as proxy (`internal/rules/merge_test.go` lines 249-275).  
  Classification: product/API decision or docs bug.

- **Current roadmap validation warnings policy is incomplete.**  
  Local `./rootline validate --all docs/roadmap/` exits 0 with 51 valid, 2 warnings: `scope-match` and `field-override`. `plan-subcommand.md` only declares `scope.match "*.md" matches no files` acceptable (line 161), not `field-override`.  
  Classification: docs/product decision.

## Existing Coverage Summary

- Good coverage: walk-up discovery, `.git` boundary, top-down merge, match-scoped fields, required-match, sequence per prefix, stem-health surfacing in CLI, nested CLI inheritance.
- Weak coverage: actual `docs/roadmap/.stem`, skill `base.stem`, `.claude/roadmap.local.md` status-values vs effective schema, CI validation target, warning whitelist, canonical roadmap creation flow.

## Test Ideas

1. Add fixture/test asserting `docs/roadmap/.stem` effective `schema.estado` matches `.claude/roadmap.local.md` statuses, or explicitly allows string if that is the decision.
2. Add CI/recipe test that validation target path exists and `rootline validate --all docs/roadmap/` passes.
3. Parse `.claude/skills/roadmap/base.stem` in tests and assert enum statuses, `tipo` values, and O/T sequence config.
4. E2E temp roadmap using `base.stem`: create `O01/README.md`, `O01/T001.md`, and root `T001.md`; assert `rootline new`, `describe schema.id.next`, `validate --all`, and `graph --check`.
5. Add warning policy test for roadmap validation: either zero warnings or only explicitly accepted warnings.
6. Add test for schema-null removal if intended; otherwise update docs to say null removal is only generic maps like derive/aggregate.
7. Add skill contract test: existing `docs/roadmap/.stem` should not drift from `.claude/skills/roadmap/base.stem` unless explicitly approved.

## Start Here

Open `docs/roadmap/.stem` and `.claude/skills/roadmap/base.stem` first. The central mismatch is `estado: string` in the actual roadmap versus `estado: enum` in the canonical skill template.

## Remaining Clarification Questions

1. Should roadmap `estado` be an enum governed by `status-values`?
2. If yes, should parent `docs/.stem` change, or should stem-health allow child string→enum narrowing?
3. Should CI/Justfile replace `docs/epics/` with `docs/roadmap/`?
4. Are current roadmap warnings acceptable, especially `field-override`?
5. Is schema-field null removal part of the promised `.stem` architecture?