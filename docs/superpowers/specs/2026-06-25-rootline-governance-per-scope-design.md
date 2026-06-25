# Diseño — Governance per-scope en repos multi-`.stem`

**Fecha:** 2026-06-25
**Estado:** aprobado, plan de implementación adjunto
**Tipo:** corrección sistémica (anti-patrón up-vs-down en governance)

## Contexto y problema

Rootline es una DB jerárquica: los `.stem` gobiernan directorios; el merge es parent→child
(DESCENDENTE); el schema está **distribuido** por el árbol. Una auditoría encontró un anti-patrón:
varios comandos hacen `index.Scan` para DESCENDER y traer records (✓), pero una resolución de schema
secundaria hace `MergeStemFiles(WalkUp(root))` — resuelve en el scan-root **subiendo** — y usa ese
único stem-de-root para una decisión sobre todo el árbol. En un repo cuyos `.stem` viven en
subdirectorios (ej. wiki: `concepts/.stem`, `entities/.stem`, `sources/.stem`, SIN `.stem` raíz),
`WalkUp(root)` no devuelve nada → no-op o resultado incorrecto.

El núcleo del pipeline ya es per-record (post-Fase 3) y correcto: derive/aggregate/enrich,
`validate --all`, `ValidateStemHealth`, `DetectMissingSchemata` (desciende con `filepath.WalkDir`),
`DetectStructural`. NO se tocan.

## Alcance

Los **3 sitios de governance/inferencia** con output incorrecto:

1. `internal/infer/validation_gaps.go` `DetectValidationGaps` — itera un stem root-merged. Caller
   `cmd/rootline/analyze.go:159`.
2. `internal/infer/delta.go` `FilterCoveredInferences`/`isCovered` — cobertura contra stem root-merged.
   `analyze --incremental` (`analyze.go:166`).
3. `cmd/rootline/schema.go` `schema propose --incremental` — filtra proposals contra stem root-only.

**Fuera de scope (follow-up documentado):** link-schema (`analyze.go:238`, `graph.go:85`), `query --sort`
(`query.go:96`), `tree` render (`tree.go:220`). Hoy degradan con gracia; su fix correcto es per-record
(mergear scopes hermanos es ambiguo ante conflictos de tipo).

## Diseño

**Resolución inyectada estilo Fase 3 + agrupamiento por scope** (helper interno en `internal/infer`,
no una API pública grande).

### Helper `internal/infer/scopes.go`

- `StemResolver func(dir string) (effective *rules.StemFile, closestStem string)`: devuelve el stem
  efectivo del directorio y el path del `.stem` leaf-most (group key). `closestStem == ""` si ninguno
  gobierna.
- `ScopeGroup{Key string; Stem *rules.StemFile; Records []*extract.Record}`.
- `GroupByScope(records, root, resolve) []ScopeGroup`: agrupa records por su `.stem` más cercano,
  preservando orden de aparición.
- `DefaultStemResolver() StemResolver`: respaldado por `rules.WalkUp(dir)` + `rules.MergeStemFiles`,
  con **cache per-directorio**; group key = `entries[len-1].Path` (leaf-most, como schema_coverage.go:55).

### Site 1 — `DetectValidationGaps` per-scope

- El cuerpo actual de checks se extrae a un interno `detectGapsForScope(stem, records, prior)` (lógica
  idéntica de hoy). `DetectValidationGaps(records, prior, root, resolve)` agrupa por scope, corre
  `detectGapsForScope` por grupo (saltando `Stem == nil`), y **deduplica** por `(Type, Field, Source)`
  (un campo heredado aparece en varios scopes; `SchemaField.Source` es la clave).
- `required_understatement` se cuenta dentro de `group.Records` → scoping correcto. **No-op para
  repos de un solo `.stem`** (un solo grupo).

### Sites 2-3 — cobertura per-scope (esquiva el trap de conflicto)

- `isCovered(inf, stem)` (chequeo a nivel campo) se conserva.
- `FilterCoveredInferences(inferences, records, root, resolve)`: una inferencia está **cubierta** sii
  para TODO scope "relevante" (el campo está en `group.Stem.Schema` o en algún record del grupo) la
  cubre `isCovered`; si algún scope relevante NO la cubre, se MANTIENE. Conservador → no suprime
  conflictos entre hermanos. Reduce al comportamiento actual para single-scope.
- `schema propose --incremental` usa la misma lógica per-scope.

## Tests

Fixture multi-`.stem` compartido (espeja el wiki: `.stem` en subárboles, sin raíz). Asserts:
validation_gaps detecta el gap de un `.stem` de subárbol (hoy 0); `required_understatement` scopeado;
dedup de heredados; incremental filtra lo cubierto por un scope y mantiene lo no cubierto / conflictos;
single-`.stem` = output actual (no-op); e2e CLI de los 3 comandos.

## Verificación (DoD)

`just check`; `just test -race`; cobertura `internal/infer` y `cmd/rootline` ≥ 85%; `analyze --all`
sobre `docs/roadmap` sin regresión; verificación manual contra el wiki real (`/Users/Shared/wiki`):
`analyze` ahora reporta gaps de los `.stem` de subárbol. Commits convencionales.
