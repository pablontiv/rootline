---
estado: Completed
tipo: refactor
ejecutable_en: 1 sesion
---
# T003: Renombrar task files, story folders y limpiar referencias Cat N

**Story**: [S004 Naming Remediation](README.md)
**Contribuye a**: Task files en F02 no contienen "Cat N" en slugs

[[blocks:T002-remove-category-field]]

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`
- INV3: Coverage >=85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

Los task files y story folders en F02 usan terminologia "Cat N" en slugs y body text. Esto viene del proceso de investigacion y no es autodescriptivo. Los nombres deben comunicar que hace cada componente sin requerir consultar el documento de investigacion original.

## Alcance

**In**:
1. Renombrar 2 story folders via `git mv`:
   - `S001-deterministic-cats-5-7-8-10/` -> `S001-deterministic-inference/`
   - `S002-body-aware-cats-6-12-13/` -> `S002-body-aware-inference/`
2. Renombrar 10 task files (dentro de las carpetas ya renombradas):
   - S001: `T001-cat-5-link-validation.md` -> `T001-link-type-validation.md`
   - S001: `T002-cat-7-back-references.md` -> `T002-back-reference-consistency.md`
   - S001: `T003-cat-8-10-constants-crossrefs.md` -> `T003-constant-detection-crossrefs.md`
   - S001: `T004-deterministic-cats-tests.md` -> `T004-deterministic-inference-tests.md`
   - S002: `T001-cat-6-body-sections.md` -> `T001-body-section-analysis.md`
   - S002: `T002-cat-12-invariant-extraction.md` -> `T002-invariant-extraction.md`
   - S002: `T003-cat-13-sub-schema-detection.md` -> `T003-subschema-detection.md`
   - S002: `T004-body-aware-cats-tests.md` -> `T004-body-analysis-tests.md`
   - S003: `T001-cat-9-formal-deps.md` -> `T001-formal-dependency-stubs.md`
   - S003: `T002-cat-11-traceability-links.md` -> `T002-traceability-link-stubs.md`
3. Actualizar tablas en READMEs padre con nuevos paths:
   - `F02/README.md` — Story table links
   - `S001/README.md` — Task table links
   - `S002/README.md` — Task table links
   - `S003/README.md` — Task table links
4. Buscar y reemplazar "Cat N" en body text de todos los .md en F02:
   - "Cat 5" -> "link-type validation" (o equivalente descriptivo contextual)
   - "Cat 6" -> "body section analysis"
   - "Cat 7" -> "back-reference consistency"
   - "Cat 8" -> "constant field detection"
   - "Cat 9" -> "formal dependency extraction"
   - "Cat 10" -> "cross-reference validation"
   - "Cat 11" -> "traceability link extraction"
   - "Cat 12" -> "invariant extraction"
   - "Cat 13" -> "sub-schema detection"
   - "cats" (plural generico) -> "categories" o "detectors" segun contexto
5. Validar: `rootline validate --all docs/epics/E13-inference-engine/F02-category-expansion/`

**Out**: No modificar codigo Go (ya hecho en T001/T002). No renombrar S003 ni S004 (no contienen "cat" en slug).

## Estado inicial esperado

- T001 y T002 completados — codigo Go ya limpio
- Story folders S001 y S002 aun con "cats" en nombre
- Task files aun con "cat-N" en slugs

## Criterios de Aceptacion

- `find docs/epics/E13-inference-engine/F02-category-expansion/ -name "*cat*" -not -path "*/S004*" | wc -l` retorna 0
- `grep -ri "Cat [0-9]" docs/epics/E13-inference-engine/F02-category-expansion/S001-*/README.md docs/epics/E13-inference-engine/F02-category-expansion/S002-*/README.md docs/epics/E13-inference-engine/F02-category-expansion/S003-*/README.md | wc -l` retorna 0
- `rootline validate --all docs/epics/E13-inference-engine/F02-category-expansion/` no tiene errores
- Todos los links internos en READMEs padre apuntan a archivos que existen

## Fuente de verdad

- `docs/epics/E13-inference-engine/F02-category-expansion/` — todo el arbol F02
- `docs/epics/E13-inference-engine/F02-category-expansion/README.md` — tabla de stories
- `docs/epics/E13-inference-engine/F02-category-expansion/S001-*/README.md` — tabla de tasks S001
- `docs/epics/E13-inference-engine/F02-category-expansion/S002-*/README.md` — tabla de tasks S002
- `docs/epics/E13-inference-engine/F02-category-expansion/S003-*/README.md` — tabla de tasks S003
