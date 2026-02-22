---
estado: Pending
tipo: feature
---
# F10: Proposal-Based Fix Engine

**Epic**: [E04](../README.md)
**Objetivo**: `rootline fix` produce propuestas diagnosticas categorizadas que guian al usuario (humano o AI) a traves de correcciones multi-paso, en vez de aplicar correcciones mecanicas con perdida de informacion
**Beneficio**: Elimina la necesidad de investigar manualmente como corregir inconsistencias entre datos y schema — la herramienta diagnostica, categoriza y propone la solucion correcta en ambas direcciones (schema y data)
**Milestone**: `rootline fix --all --dry-run` en un proyecto con errores mixtos (enum invalidos, campos faltantes, metadata en body) produce propuestas agrupadas por tipo con cero perdida de informacion

## Scope

**In**: Proposal engine (`internal/proposal/`), body text scanning (`internal/extract/`), enhanced `rootline fix` output (dry-run y apply), soporte para modificar .stem ademas de archivos de datos
**Out**: Cambios a `rootline init`, cambios a `rootline doctor`, UI interactiva para seleccionar propuestas

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| [S001](S001-body-text-scanning/) | Body Text Scanning | Rootline detecta metadata en body markdown (`**Key**: Value`) |
| [S002](S002-proposal-analysis-engine/) | Proposal Analysis Engine | `proposal.Analyze()` categoriza errores en propuestas priorizadas |
| [S003](S003-fix-command-integration/) | Fix Command Integration | `rootline fix --dry-run` muestra propuestas; `rootline fix` las aplica |

## Dependencias

- Ninguna — independiente de F07, F08, F09

## Fuente de verdad

- `cmd/rootline/fix.go` — comando fix actual a extender
- `internal/extract/extract.go` — extractor a extender con body scanning
- `internal/rules/validate.go` — errores de validacion que alimentan las propuestas
- `internal/rules/rules.go` — StemFile/SchemaField types
