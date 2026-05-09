# Code Context

## Files Retrieved
1. `docs/.stem` (lines 1-7) - parent docs schema; requires `estado` for all docs markdown.
2. `docs/roadmap/.stem` (lines 1-35) - roadmap-specific constraints for outcome/task files, IDs, and `blocked_by` links.
3. `docs/roadmap/O01-map-rootline-integration-surface/README.md` (lines 1-43) - establishes command inventory / JSON contract / risk classification as prerequisites and excludes changing Rootline command semantics.
4. `docs/roadmap/O01-map-rootline-integration-surface/T002-capture-json-contracts.md` (lines 1-45) - task template and JSON contract pattern for `cmd/rootline/*.go`.
5. `docs/roadmap/O01-map-rootline-integration-surface/T003-classify-pi-exposure.md` (lines 1-45) - task template and explicit command risk-class requirement.
6. `docs/roadmap/O03-build-read-only-pi-package-mvp/T003-implement-query-describe-tools.md` (lines 1-46) - existing pattern for grouping related Rootline command tools.
7. `docs/roadmap/O03-build-read-only-pi-package-mvp/T004-implement-validate-tree-stats-tools.md` (lines 1-46) - existing validation/tree/stats tool pattern and acceptance criteria.
8. `docs/roadmap/O04-package-skill-prompts-and-commands/T004-implement-validation-and-tree-commands.md` (lines 1-44) - slash-command validation/tree UX pattern.
9. `docs/roadmap/O06-add-safe-mutation-tools/README.md` (lines 1-43) - safe narrow mutation scope; explicitly excludes bulk fix/migrate/apply.
10. `docs/roadmap/O07-expose-complex-operations-with-guardrails/README.md` (lines 1-43) - O07 boundaries; includes guarded complex workflows and excludes core command bugfixes.
11. `docs/roadmap/O07-expose-complex-operations-with-guardrails/T001-design-complex-operation-ux.md` (lines 1-47) - O07 design task; sources are `analyze`, `fix`, `migrate`, `apply` command files.
12. `docs/roadmap/O07-expose-complex-operations-with-guardrails/T002-implement-protected-fix-workflow.md` (lines 1-46) - guarded `fix` workflow task pattern.
13. `docs/roadmap/O07-expose-complex-operations-with-guardrails/T003-implement-protected-migrate-workflow.md` (lines 1-46) - guarded `migrate` workflow task pattern.
14. `docs/roadmap/O07-expose-complex-operations-with-guardrails/T004-implement-analyze-apply-workflow.md` (lines 1-46) - guarded `analyze/apply` workflow task pattern.
15. `docs/roadmap/O07-expose-complex-operations-with-guardrails/T005-document-complex-operation-risks.md` (lines 1-44) - risk/rollback documentation task pattern.
16. `docs/roadmap/O08-productionize-testing-release-and-adoption/README.md` (lines 1-45) - production/release scope; not a fit for Rootline core behavior changes.
17. `Justfile` (lines 1-27) - repo check/test recipes and currently stale docs validation target.
18. `intake/.stem` (lines 1-12) and `intake/inference-engine-architecture.md` (lines 1-23, 55-63) - current non-roadmap intake/reference pattern; useful only for exploratory/research records, not executable task plans.
19. `MAP.md` (lines 1-67) - confirms old research/intake map; references backlog paths that are not present locally.

## Key Code

### Roadmap schema/invariants

`docs/roadmap/.stem` defines only lightweight frontmatter constraints:

```yaml
schema:
  estado:
    type: string
    required:
      match: ["O*", "T*"]
    match: ["O*", "T*"]
  tipo:
    type: enum
    required:
      match: ["O*", "T*"]
    match: ["O*", "T*"]
    values: [outcome, task]
  id:
    type: sequence
    match:
      "O*": { prefix: O, digits: 2 }
      "T*": { prefix: T, digits: 3 }
links:
  blocked_by:
    target: '^(\./|\.\./|.*/)T[0-9]{3}-[^/]+\.md$'
```

Practical constraints:
- Outcome README frontmatter pattern: `estado: Pending`, `tipo: outcome`.
- Task frontmatter pattern: `estado: Specified`, `tipo: task`.
- File naming matters: outcomes `O##-slug/README.md`, tasks `T###-slug.md`.
- `blocked_by` links must target task files named `T###-*.md` using `./`, `../`, or another path prefix.
- The schema does **not** enforce sections beyond frontmatter/link shape; section structure is conventional.

### Existing outcome pattern

Outcome READMEs use:
- frontmatter
- `# O##: Title`
- `## Objetivo`
- `## Criterios de Éxito`
- `## Invariantes`
- `## Alcance` with `**In**` / `**Out**`
- `## Tasks` table

O07 is the key boundary file:
- In: `rootline_fix`, `rootline_migrate`, `rootline_analyze/apply`, rollback/preview docs (`docs/roadmap/O07.../README.md` lines 23-29).
- Out: `Fixing Rootline core command bugs`, `Publishing package` (lines 31-33).

### Existing task pattern

Tasks use:
- frontmatter
- `# T###: Title.`
- `**Outcome**: [...]`
- `**Contribuye a**: CE# del Outcome.`
- optional `[[blocked_by:...]]`
- `## Preserva` with copied invariant
- `## Contexto`
- `## Alcance` with numbered In and Out
- `## Estado inicial esperado`
- `## Criterios de Aceptación`
- `## Fuente de verdad`

O07 task dependency chain:
- `T001` blocked by O06 `T005`.
- `T002`, `T003`, `T004` blocked by O07 `T001`.
- `T005` blocked by O07 `T004`.

## Architecture

The local `docs/roadmap/` is a Pi integration roadmap, not a general Rootline core backlog:
- O01 maps Rootline command contracts/risk and explicitly excludes changing command semantics (`O01/README.md` lines 23-35).
- O03/O04 expose read-only commands and slash commands.
- O06 exposes narrow safe mutations (`new`, `set`) and explicitly excludes bulk `fix/migrate/apply` (`O06/README.md` lines 23-33).
- O07 handles protected interfaces around complex/bulk operations but explicitly excludes fixing Rootline core command bugs (`O07/README.md` lines 23-33).
- O08 handles productionization/release/adoption and excludes data-model/editor-integration expansion (`O08/README.md` lines 23-34).

Local filesystem facts:
- There is no `docs/epics/` directory despite `Justfile` and inherited docs referencing it.
- There is no `integrations/pi/` directory yet, though multiple roadmap tasks cite `integrations/pi/extensions/` as future source of truth.
- There are no child `.stem` files under `docs/roadmap/`; all roadmap outcomes/tasks inherit the root roadmap schema.

Validation commands observed:
- `rootline validate --all docs/roadmap/` exits valid: 51 total validation records, 0 invalid, 2 warnings.
  - Warnings are stem-health only: `scope.match "*.md" matches no files in directory` and `field "estado" overrides parent definition`.
- `rootline validate --all docs/` exits valid: 69 total records, same 2 warnings.
- `rootline tree docs/roadmap --output table` shows 8 outcomes and 41 tasks; all outcomes `Pending`, all tasks `Specified`.
- `rootline stats docs/roadmap --output json` reports `by_estado: {Pending: 8, Specified: 41}`, `by_tipo: {outcome: 8, task: 41}`, `total: 49`.
- `just validate` is currently stale/broken because it runs `rootline validate --all docs/epics/`, and `docs/epics/` does not exist (`Justfile` lines 21-23).

## Start Here

Start with `docs/roadmap/O07-expose-complex-operations-with-guardrails/README.md` because it defines the exact O07 scope boundary and explicitly says core Rootline command bugfixes are out of scope. Then open `docs/roadmap/.stem` to ensure any repair-plan files obey filename/frontmatter/link constraints.

## Suggested roadmap decomposition

If the repair plan is about Pi guardrails around already-working Rootline commands, keep it inside existing O07 tasks:
1. Update/execute `T001` design for user intent, preview, validation, rollback.
2. Implement protected `fix` in `T002`.
3. Implement protected `migrate` in `T003`.
4. Implement protected `analyze/apply` in `T004`.
5. Document risk/rollback in `T005`.

If the repair plan requires fixing Rootline core command bugs or changing core command semantics:
- Do **not** put those bugfix tasks inside O07; O07 explicitly excludes them.
- Do **not** hide them in O03/O04/O06; those outcomes are Pi exposure/UX layers, not core repairs.
- Best local fit is a new roadmap outcome, e.g. `docs/roadmap/O09-repair-rootline-command-contracts/`, with tasks like:
  - `T001-reproduce-command-bugs.md` — minimal repros and failing tests for affected `cmd/rootline/*.go` commands.
  - `T002-normalize-json-contracts.md` — fix/add stable `version`/`kind` JSON output where Pi must parse it.
  - `T003-fix-command-behavior.md` — actual Rootline CLI/internal fixes, scoped by command.
  - `T004-add-regression-tests.md` — Go tests plus CLI smoke cases.
  - `T005-update-command-docs-and-roadmap-dependencies.md` — docs and `blocked_by` links back into O07 tasks.
- Then add `[[blocked_by:../O09-repair-rootline-command-contracts/T###-...md]]` from the relevant O07 task(s), preserving the `T###-*.md` link regex.

Alternative if the bugfix is exploratory rather than committed implementation: use `intake/` first (schema only requires `estado` and `fecha`), then promote to a roadmap outcome/task once the repair scope is concrete. Existing `intake/` is research/reference oriented, not a task execution queue.

## Constraints / invariants to preserve

- Preserve O07 INV1: bulk operations require explicit user intent and cannot be triggered silently by agent context guidance.
- For mutating or bulk workflows, acceptance criteria should include preview/dry-run, explicit target/report, user approval, post-run validation, and rollback guidance.
- Keep Pi integration boundary honest: command contract/risk documentation should precede tool exposure (O01 INV1).
- Keep core command repairs separate from extension wrappers; O07 can depend on them but should not absorb them.
- After any roadmap edits, run `rootline validate --all docs/roadmap/`; expect only the existing two warnings unless schema/docs are changed.

## Remaining clarification questions

1. Should core Rootline command bugfixes be tracked in this Pi roadmap as a new O09 dependency outcome, or in an external issue tracker/backlog not present in the repo?
2. If adding a new outcome, should numbering append as O09, or should the roadmap be renumbered/inserted before O07 to reflect dependency order?
3. Which concrete commands are bugged (`fix`, `migrate`, `analyze`, `apply`, or shared JSON/output behavior), and are they blockers for O07 or independent cleanup?
4. Should `Justfile validate` be updated from `docs/epics/` to `docs/roadmap/`, or is `docs/epics/` expected to be restored later?
