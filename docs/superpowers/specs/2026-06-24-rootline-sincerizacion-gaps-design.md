# Rootline — Sincerización post-O14 + cierre de gaps

**Fecha:** 2026-06-24
**Estado:** Diseño aprobado, pendiente de plan de implementación

## Contexto y problema

Una investigación cruzada (backscroll, docs wiki en `/Users/Shared/wiki`, y el código del repo)
buscó oportunidades de mejora, cierre de gaps y bugfixes en rootline. El hallazgo más fuerte no fue
un bug de runtime sino una **brecha de honestidad**: la documentación, el skill y restos de código
describen un subsistema (`domain:` types, `domain_coverage`, `DetectSubSchemas`) y un conteo de
detectores que **ya no existen** — se eliminaron en el refactor field-agnostic (O14). Además quedan
un comando deprecado totalmente reemplazado (`apply`), una limitación real de resolución
multi-`.stem` en derive/aggregate, y una interfaz de extractores lista pero con un solo formato.

El objetivo es un programa de trabajo que cubra los cinco hallazgos, cortado en cuatro fases de
**riesgo ascendente + dependencias**: primero la verdad barata e independiente, después remoción de
código muerto/deprecado, después el cambio de comportamiento, y al final la feature más grande (la
única con prerequisito).

### Evidencia base (verificada en código)

| Hallazgo | Evidencia |
|---|---|
| Detectores reales: **14 (12 data + 2 governance)**, no 15 ni 16 | `cmd/rootline/analyze.go:111-162`; CLAUDE.md dice 15, CHANGELOG dice 16 |
| Subsistema `domain:` **removido** en O14 | `internal/rules/domains.go` no existe; `internal/e2e/governance_test.go:26` |
| `DetectSubSchemas` removido a propósito (anti field-agnostic) | `internal/infer/subschema_detection.go` (solo comentario-tumba) |
| `missing_domain` es **código muerto** | `cmd/rootline/analyze.go:44`, `internal/e2e/analyze_test.go:42`; ningún detector lo emite |
| `apply` removido; reemplazado por `schema apply` + `repair apply` | `cmd/rootline/apply.go` (deleteable); `schema.go:88` applies schema inferences from analyze reports; `repair.go:32` applies data repairs |
| derive/aggregate usan `DefaultResolver` (merge-only), no per-record | `internal/derive/pipeline.go:77`, `aggregate.go:162`; `ResolveForRecord` ya existe y se usa en describe/validate/fix/set/repair |
| Extractor interface pluggable pero writes hardcodean frontmatter `---` | `internal/extract/extract.go:21`, `registry.go`; `fix/fix.go:122`, `infer/apply.go:241`, `migrate/rename.go` |
| `type: section` **sigue vivo** (verificado end-to-end); O14 removió solo el append `+=` | `new.go:144-160` emite la sección si es required/default; `describe` la expone; `rules.go:167-169` la parsea; `set.go:263` y `SKILL.md:109` afirman erróneamente que `type:section` fue removido |

### Descartado con evidencia (no accionable)

- **Resucitar `DetectSubSchemas` / `domain_coverage`:** revierte la decisión de arquitectura O14
  (field-agnostic). No se hace.
- **Líneas de investigación pausadas:** barrera descriptivo/normativo (un marco mental ya resuelto,
  no una feature) y modelo de entidad v3 (deferido por tensiones filosóficas sin resolver).

---

## Fase 1 — Sincerización: docs + skill + código muerto

**Objetivo:** que docs, skill y código digan la verdad post-O14, y documentar el contrato
engine/agente.

**Cambios:**

- **Conteo de detectores → 14 (12 data + 2 governance)** en `CLAUDE.md:9,31,38`, `README.md:185`,
  `CHANGELOG.md:21`, y `docs/roadmap/O09-.../T001-codify-command-responsibility-contracts.md:96`.
- **Remover docs `domain:` stale:** sección "Domain Types" del `README.md:137-162` (reemplazar por
  nota de deprecación o eliminar), fila de tabla `README.md:16`, y menciones en `CLAUDE.md:31`
  (incl. `describe --by-domain`) y `CLAUDE.md:38`.
- **Código muerto:** quitar `"missing_domain"` de `agentRequiredTypes` (`cmd/rootline/analyze.go:44`)
  y del fixture (`internal/e2e/analyze_test.go:42`); borrar el archivo-tumba
  `internal/infer/subschema_detection.go`.
- **Corregir `type: section`:** `cmd/rootline/set.go:263` debe referirse al append `+=` removido, no
  a "type:section has been removed"; ajustar `SKILL.md:109` para reflejar que el tipo sigue
  soportado (`source: body.section[...]` como alternativa, no reemplazo).
- **Contrato engine/agente en README** (`AI-Native` ~`:298` o `Fix & Proposals` ~`:275`): el engine
  resuelve lo computable **por forma** (frecuencia, acuerdo unánime, estructura) y marca como
  `requires_agent` lo que requiere **juicio semántico**; `analyze` expone **evidencia porcentual, no
  opiniones**. El rationale (barrera estadístico/semántico) queda en el wiki, no en el README.
- **Skill apply guidance:** `SKILL.md:118` (y `:3`, `:127`) → recomendar `rootline schema apply
  --report <f>` y `rootline repair apply --report <f>` en vez de `apply`.

**Verificación:** `just check` + `just test` verdes; `rg "15 detectors|16 detectors|13 data"` y
`rg -i "domain:"` sobre repo+skill sin claims vivos; `rootline analyze docs/roadmap/` corre.

**Nota wiki:** `/Users/Shared/wiki/wiki/entities/rootline.md` es artefacto ingerido; se regenera vía
wiki-ingest tras corregir las fuentes del repo. No editar a mano.

---

## Fase 2 — Remover el comando `apply` deprecado

**Objetivo:** eliminar `apply` CLI. Capacidades preservadas: `schema apply --report <analyze.json>` 
consumes inference reports from `analyze` to apply schema-field changes to `.stem` files (replaces the 
legacy `update_stem` capability); `repair apply --report` applies data repairs. Verificado: borrarlo 
no pierde funcionalidad tras Option D (schema apply consumes analyze reports).

**Cambios:**

- Borrar `cmd/rootline/apply.go` y desregistrar el comando en el setup de cobra (root).
- Migrar/eliminar ~22 referencias en tests (`apply_deprecation_test.go`, `apply_table_test.go`,
  `fix_apply_test.go`, `coverage_test.go` incl. los characterization tests que hacían `t.Skip` en
  `:526/:595/:655`, `coverage_boost_test.go`, `repair_test.go`); cubrir paridad con tests de
  `schema apply`/`repair apply` donde falte.
- **No** borrar `internal/infer/apply.go` (`ApplySchemaInferences`/`ScaffoldSchema` siguen usándose
  por schema/repair apply); solo el comando CLI. Revisar `rewriteFrontmatter`
  (`internal/infer/apply.go:241`): si queda sin uso tras la remoción, eliminarlo (reduce el hardcode
  de frontmatter que estorba a Fase 4).
- Actualizar docs que describen `apply` como vigente/deprecado → "removido".

**Verificación:** `rg '"apply"|applyCmd'` sin comando vivo; `rootline --help` no lista `apply`;
`just test` verde; flujo `analyze → schema apply → repair apply` funciona en un fixture.

---

## Fase 3 — Resolución per-record en derive/enrich

**Objetivo:** que campos derivados/extraídos respeten el schema efectivo **por record** en islas
multi-nivel con `match:` distintos, no el merge directorio-nivel.

**Cambios:**

- `internal/derive/enrich.go` (`EnrichBuiltins`): usa resolución per-record vía `ResolveForRecord(dir, recordPath)`
  (línea 33) para extraer campos `source:` (como `source: body.h1`) escoped por `match:`. Así, un campo
  derivado con `match:` solo se extrae en records coincidentes, no en toda la isla. Esta es la **verdadera
  fix observable** (extracciones respetan `match:`).
- `internal/derive/pipeline.go` (`DeriveRecord`) y `internal/derive/aggregate.go` (`AggregateAll`): usan solo
  los mapas `Derive` y `Aggregate` de la schema, que match-filtering ya no afecta (no acceden a Schema
  directamente). El cambio a `ResolveForRecord` es **consistency no-op**: derive/aggregate no tienen
  comportamiento visible distinto, solo siguen el mismo patrón que enrich para coherencia arquitectónica.
- Verificación: `record.go:50`, `enrich.go:47-67` prueban la extracción per-record; `hierarchy.go` (infer)
  solo filtra Schema en el pipeline (no Derive/Aggregate).

**Verificación:** `just check` + `just test` verdes; no debería romper golden tests pues DeriveRecord/AggregateAll
comportamiento es invariante (solo acceden a Derive/Aggregate maps, no Schema).

---

## Fase 4 — Expansión de extractores no-markdown

**Objetivo:** habilitar extractores no-markdown (p.ej. YAML/JSON) sin romper fix/repair/migrate.

**Cambios:**

- **Prerequisito (abstracción de escritura):** mover la serialización/deserialización de frontmatter
  detrás del `Extractor` (o un writer asociado) para que cada formato sepa reescribir lo suyo. Hoy
  el rewrite hardcodea `---`: `internal/fix/fix.go:122` (`RewriteFrontmatter`),
  `internal/migrate/rename.go` (`renameFrontmatterField`), y `internal/infer/apply.go:241` (si
  sobrevivió a Fase 2). El `Record` ya es format-neutral (`internal/extract/extract.go`), así que el
  cambio se concentra en los writers.
- **Read-path primero:** implementar un extractor adicional simple (YAML o JSON) registrándolo en
  `internal/extract/registry.go`. Con eso, query/validate/describe/tree funcionan sobre el nuevo
  formato sin tocar el resto del pipeline.
- **Write-path después:** una vez abstraído el rewrite, habilitar fix/repair/set/migrate para el
  formato nuevo. (Depende del prerequisito y del read-path.)
- Documentar el formato soportado y los no-objetivos en `docs/extensibility.md` y README.

**Verificación:** tests de extracción para el formato nuevo; `rootline validate`/`query` sobre un
fixture no-markdown; `rootline fix`/`set` reescriben correctamente; `just check` + `just test`.

---

## Secuenciación y notas

- Las fases son independientes y se ejecutan/commitean por separado; Fase 1 desbloquea claridad para
  las demás.
- Fase 4 depende de su propio prerequisito interno (abstracción de writers) y se beneficia de que
  Fase 2 elimine `infer/apply.go:241` si queda muerto.
- Cada cambio respeta la Definition of Done del repo: `just check` + `just test` verdes, commit
  convencional, y cobertura ≥85% por `.coverage-floors.toml`.
