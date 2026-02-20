# F02: Document Lifecycle

**Epic**: [E04](../README.md)
**Objetivo**: Rootline cubre el ciclo completo de documentos: inferir schema desde archivos existentes, crear documentos validos desde schema, y reparar errores automaticamente
**Beneficio**: Elimina la barrera de adopcion (no hay que escribir .stem desde cero), reduce friccion diaria (new), y cierra el gap entre detectar errores y corregirlos (fix)
**Milestone**: `rootline init docs/` genera .stem inferido, `rootline new docs/prd/feature.md` crea documento con campos pre-poblados, `rootline fix docs/prd/feature.md` agrega campos faltantes

## Scope

**In**: Schema inference desde archivos existentes, scaffolding de documentos desde schema efectivo, auto-fix de errores de validacion
**Out**: Schema migrations (cambios entre versiones de .stem), schema versioning, templates externos

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [Schema Inference](S001-schema-inference/) | Rootline analiza archivos existentes e infiere un .stem schema |
| S002 | [Scaffolding and Repair](S002-scaffolding-repair/) | Rootline genera documentos validos y repara errores de validacion |

## Dependencias

- Ninguna (usa core engine de E03)

## Fuente de verdad

- `internal/extract/` — Record type, Registry, MarkdownExtractor
- `internal/rules/` — StemFile, WalkUp, MergeStemFiles, Validate
- `internal/index/` — Scan
