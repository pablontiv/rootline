---
estado: Specified
tipo: historia
cliente: Platform Owner
---
# S002: Proposal Analysis Engine

**Feature**: [F10 Proposal-Based Fix Engine](../README.md)
**Capacidad**: `proposal.Analyze()` toma records, schema y errores de validacion, y produce un Report con propuestas categorizadas y priorizadas en vez de correcciones mecanicas planas

## Antes / Despues

**Antes**: `applyFixes()` en fix.go muta records directamente: agrega primer valor de enum para campos faltantes, aplica Levenshtein para enums invalidos. No distingue si el error es del schema o de los datos. Pierde informacion semantica (ej: "Obsoleto" → "Completado", "Pending (blocked by X)" → "Pending").

**Despues**: `proposal.Analyze()` categoriza cada error de validacion en uno de 6 tipos (extend_enum, migrate_value, extract_body, infer_from_children, add_field, correct_value), agrupa archivos afectados, y produce un Report JSON con propuestas priorizadas que preservan toda la informacion semantica.

## Criterios de Aceptacion (semanticos)

- [ ] 3 archivos con "Obsoleto" → 1 propuesta extend_enum (no correct_value)
- [ ] "Pending (blocked by T001)" → propuesta migrate_value con Bloqueada + wiki-links
- [ ] README con `**Estado**: Completada` en body → propuesta extract_body con mapping
- [ ] README sin hints pero hijos Completado → propuesta infer_from_children
- [ ] `go test ./internal/proposal/ -v` pasa

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-create-proposal-types-and-basic-detectors.md) | Types, Analyze(), extend_enum, correct_value, add_field |
| [T002](T002-implement-migrate-value-detector.md) | migrate_value + ParseBlockingInfo() |
| [T003](T003-implement-body-and-child-inference-detectors.md) | extract_body + infer_from_children |

## Fuente de verdad

- `internal/proposal/` — package nuevo a crear
- `internal/extract/extract.go` — ScanBodyFields (de S001)
- `internal/rules/validate.go` — ValidationError types
- `internal/rules/rules.go` — StemFile, SchemaField types
- `cmd/rootline/fix.go` — logica existente de applyFixes/closestMatch a migrar
