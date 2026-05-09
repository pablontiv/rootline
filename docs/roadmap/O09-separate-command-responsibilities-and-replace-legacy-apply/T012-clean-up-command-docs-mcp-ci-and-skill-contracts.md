---
estado: In Progress
tipo: task
---
# T012: Clean up command docs MCP CI and skill contracts

**Outcome**: [O09 Separate command responsibilities and replace legacy apply](README.md)
**Contribuye a**: CE5 del Outcome.

[[blocked_by:./T011-deprecate-legacy-apply-and-update-command-surfaces.md]]

## Preserva

- INV2: Every machine-readable command used by agents emits parseable, versioned JSON.
  - Verificar: docs and skill references match implemented JSON kinds/tool catalog.

## Contexto

Investigation found several contract drifts: docs mention 9 MCP tools while code registers 12, `Justfile` and CI reference `docs/epics/`, `docs/set.md` and `docs/migrate.md` have behavior/kind drift, and skills currently warn that `apply --dry-run` is unsafe.

## Alcance

**In**:
1. Update README command tables and docs for replacement schema/repair workflows.
2. Update `.claude/skills/rootline` references and remove/replace stale unsafe `apply` guidance after fixes.
3. Reconcile MCP tool catalog docs with code and any new tools.
4. Fix or explicitly document stale CI/Justfile/pre-push validation paths and root agent-guide references.
5. Update O07 task dependencies and wording so Pi guardrails build on safe core commands.
6. Run roadmap/docs validation.

**Out**:
- Implementing command behavior.
- Publishing marketplace/package releases.

## Estado inicial esperado

- T011 has established final command surfaces.

## Criterios de Aceptación

- README/docs/skills no longer recommend unsafe generic `apply` workflows.
- MCP docs match registered tools and any newly exposed tools.
- `Justfile`/CI/pre-push validation paths and agent-guide command counts/paths are corrected or intentionally explained.
- O07 tasks have `blocked_by` links to the relevant O09 tasks.
- `rootline validate --all docs/roadmap/` and the relevant Go/doc checks pass.

## Fuente de verdad

- `README.md`
- `docs/fix.md`
- `docs/migrate.md`
- `docs/set.md`
- `docs/json-rpc.md`
- `docs/validate.md`
- `.claude/skills/rootline/SKILL.md`
- `.claude/skills/rootline/ref-advanced.md`
- `Justfile`
- `.github/workflows/ci.yml`
- `.githooks/pre-push`
- `CLAUDE.md`
- `CONTRIBUTING.md`
- `internal/mcp/tools.go`
