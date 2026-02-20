# S002: Scaffolding and Repair

**Feature**: [F02 Document Lifecycle](../README.md)
**Estado**: Specified
**Cliente**: Platform Owner
**Capacidad**: Rootline genera documentos Markdown pre-poblados desde el schema efectivo y repara automaticamente errores de validacion

## Antes / Despues

**Antes**: Crear un documento valido requiere recordar que campos pide el schema, que valores son validos para enums, y que campos son required. Corregir errores de validacion es manual — el usuario lee el output de `rootline validate`, abre cada archivo, y agrega campos uno por uno.

**Despues**: `rootline new docs/prd/feature.md` genera un archivo con todos los campos required pre-poblados con defaults, campos enum con primer valor, y comentarios indicando valores validos. `rootline fix docs/prd/feature.md` lee errores de validacion y los corrige automaticamente (agrega campos faltantes, corrige enums al valor mas cercano).

## Criterios de Aceptacion (semanticos)

- [ ] `rootline new` genera documento con todos los campos required del schema efectivo
- [ ] `rootline fix` agrega campos required faltantes con valores default
- [ ] `rootline fix` corrige valores enum invalidos al match mas cercano
- [ ] `rootline fix --dry-run` muestra cambios sin modificar archivos

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-new-command.md) | Comando new que scaffold documentos desde schema efectivo |
| [T002](T002-fix-command.md) | Comando fix que repara errores de validacion automaticamente |

## Fuente de verdad

- `internal/rules/` — WalkUp, MergeStemFiles, Validate, SchemaField
- `internal/extract/` — Record, MarkdownExtractor
