---
estado: Specified
tipo: refactor
ejecutable_en: 1 sesion
---
# T006: Reemplazar terminología de investigación en documentos del roadmap

**Story**: [S004 Naming Remediation](README.md)
**Contribuye a**: Cero lenguaje de investigación en documentos de producto

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`

## Contexto

Los documentos .md del roadmap E13 usan terminología del proceso de investigación que no es autodescriptiva: "Cat N" (números de categoría del research), "porciones Go/LLM" (descomposición research), porcentajes como "30% Go / 70% LLM", y referencias a teorías como "computation-then-understanding". El código Go ya fue limpiado en T001/T002/T004. Los directorios se renombran en T005. Este task limpia el contenido textual.

## Alcance

**In**:
1. Reemplazar toda referencia "Cat N" por nombre descriptivo del detector:
   - Cat 5 → link-type validation
   - Cat 6 → body section analysis
   - Cat 7 → back-reference consistency
   - Cat 8 → constant field detection
   - Cat 9 → formal dependency extraction
   - Cat 10 → cross-reference validation
   - Cat 11 → traceability link extraction
   - Cat 12 → invariant extraction
   - Cat 13 → sub-schema detection
2. Eliminar o reescribir frases con "porciones Go/LLM", "engine portion", porcentajes "N% Go / M% LLM"
3. Eliminar referencias a "computation-then-understanding" (teoría de research)
4. Reemplazar "semantic residue" / "residuo semántico" por "análisis semántico pendiente" o similar
5. Reescribir "categorias 5-13", "categorias 100% engine", "cats con alto % LLM" con vocabulario de producto
6. Actualizar títulos de Story refs en tasks: "Deterministic Categories 5/7/8/10" → usar nombre descriptivo del Story
7. Archivos afectados (~19, todos bajo `docs/epics/E13-inference-engine/`):
   - `README.md`
   - `F02-.../README.md`
   - `F02-.../S001/.../T001` a `T004` (4 archivos)
   - `F02-.../S002/.../README.md`, `T001` a `T004` (5 archivos)
   - `F02-.../S003/.../README.md`, `T001` a `T003` (4 archivos)
   - `F01/.../S002/.../T002-extract-codeblocks-tables.md`
   - `F03/.../README.md`
   - `F03/.../S001/.../T004-analyze-integration-tests.md`

**Out**: No tocar S004-naming-remediation/ (documenta la remediación misma). No tocar docs/research/. No modificar código Go (ya hecho en T001/T002/T004). No renombrar directorios (eso es T005).

## Estado inicial esperado

- T005 completado — directorios ya renombrados (paths actualizados)
- ~45 referencias "Cat N" y términos research en documentos .md

## Criterios de Aceptacion

- `grep -rni "cat [0-9]\|cat[0-9]" docs/epics/E13-inference-engine/ --include="*.md" | grep -v S004-naming` retorna vacío
- `grep -rni "porcion.*LLM\|porcion.*Go\|engine.portion\|% Go\|% LLM" docs/epics/E13-inference-engine/ --include="*.md" | grep -v S004-naming` retorna vacío
- `grep -rni "semantic.residue\|residuo.semantic\|computation.then" docs/epics/E13-inference-engine/ --include="*.md" | grep -v S004-naming` retorna vacío
- `rootline validate --all docs/epics/E13-inference-engine/` no tiene errores de validación

## Fuente de verdad

- Todos los .md bajo `docs/epics/E13-inference-engine/` (excepto S004)
- Tabla de mapping Cat N → nombre descriptivo en sección Alcance arriba
