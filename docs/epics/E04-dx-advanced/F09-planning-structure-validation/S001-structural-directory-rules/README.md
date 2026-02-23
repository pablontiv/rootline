---
tipo: historia
cliente: Platform Owner
---
# S001: Structural Directory Rules

**Feature**: [F09 Planning Structure Validation](../README.md)
**Estado**: Specified
**Cliente**: Platform Owner
**Capacidad**: .stem files pueden definir reglas estructurales sobre directorios (archivos requeridos, conteo minimo de hijos) y `rootline validate --all` las verifica

## Antes / Despues

**Antes**: `.stem` solo valida campos de frontmatter en archivos individuales. No puede expresar "cada subdirectorio debe tener README.md" ni "minimo 2 features por epic". Las inconsistencias estructurales (E03 sin README, E03 con 1 sola feature) pasan desapercibidas hasta revision manual.

**Despues**: `.stem` acepta bloque `structural:` con reglas de directorio. `rootline validate --all docs/epics/` reporta automaticamente directorios sin index file y directorios con menos hijos del minimo requerido.

## Criterios de Aceptacion (semanticos)

- [ ] Un .stem con `structural.subdirs.require_index: README.md` causa que validate reporte directorios sin README.md
- [ ] Un .stem con `structural.subdirs.min_children: 2` causa que validate reporte directorios con menos de 2 subdirectorios
- [ ] Los resultados estructurales aparecen en BatchValidationResult junto con los resultados de archivo

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-extend-stemfile-structural-types.md) | Agregar StructuralRules y SubdirRules structs a StemFile |
| [T002](T002-implement-validate-directory.md) | Implementar ValidateDirectory con require_index y min_children |
| [T003](T003-integrate-structural-into-validate-all.md) | Integrar structural checks en validate --all pipeline |

## Fuente de verdad

- `internal/rules/rules.go` — StemFile struct (lineas 14-24)
- `internal/rules/validate.go` — Validate function (linea 25)
- `cmd/rootline/validate.go` — runValidateAll function
