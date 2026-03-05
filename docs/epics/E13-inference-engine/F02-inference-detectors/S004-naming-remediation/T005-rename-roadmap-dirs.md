---
estado: Completed
tipo: refactor
ejecutable_en: 1 sesion
---
# T005: Rename roadmap directories — eliminar nomenclatura de investigación

**Story**: [S004 Naming Remediation](README.md)
**Contribuye a**: Nombres de directorios describen qué contienen, no de qué categoría de research vienen

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`

## Contexto

Los directorios del roadmap E13/F02 usan nombres que vienen del proceso de investigación: "category-expansion", "deterministic-inference" (clasificación research, no descripción de producto), "semantic-stubs-9-11" (números de categoría). Deben describir qué contienen en vocabulario de producto.

## Alcance

**In**:
1. Renombrar `F02-category-expansion/` → `F02-inference-detectors/`
2. Renombrar `S001-deterministic-inference/` → `S001-structural-detectors/`
3. Renombrar `S003-semantic-stubs-9-11/` → `S003-semantic-extraction/`
4. Actualizar todas las referencias internas:
   - Links en READMEs padre (E13/README.md, F02/README.md)
   - Story refs en tasks (`**Story**: [S001 ...]`, `**Feature**: [F02 ...]`)
   - Links relativos entre archivos dentro de los directorios renombrados
   - Cualquier .stem path que referencie estos directorios

**Out**: Cambiar el contenido textual de los documentos (eso es T006). Solo paths y links.

## Estado inicial esperado

- `docs/epics/E13-inference-engine/F02-category-expansion/` existe
- `docs/epics/E13-inference-engine/F02-category-expansion/S001-deterministic-inference/` existe
- `docs/epics/E13-inference-engine/F02-category-expansion/S003-semantic-stubs-9-11/` existe

## Criterios de Aceptacion

- `ls docs/epics/E13-inference-engine/ | grep F02` muestra `F02-inference-detectors`
- `ls docs/epics/E13-inference-engine/F02-inference-detectors/ | grep S001` muestra `S001-structural-detectors`
- `ls docs/epics/E13-inference-engine/F02-inference-detectors/ | grep S003` muestra `S003-semantic-extraction`
- `grep -rn "category-expansion\|deterministic-inference\|semantic-stubs-9-11" docs/epics/E13-inference-engine/ | grep -v S004` retorna vacío
- `rootline validate --all docs/epics/E13-inference-engine/` sin errores

## Fuente de verdad

- `docs/epics/E13-inference-engine/README.md` — links a F02
- `docs/epics/E13-inference-engine/F02-category-expansion/README.md` — links a Stories
- Todos los archivos dentro de S001, S003 — Story refs y links relativos
