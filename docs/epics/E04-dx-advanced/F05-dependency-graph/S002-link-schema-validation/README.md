# S002: Link Schema and Validation

**Feature**: [F05 Dependency Graph](../README.md)
**Estado**: Specified
**Cliente**: Platform Owner
**Capacidad**: .stem define que tipos de links estan permitidos y hacia que targets, y la validacion detecta links invalidos

## Antes / Despues

**Antes**: `links:` en .stem se parsea como map[string]any opaco sin semantica. No hay forma de definir que tipos de link son validos en un directorio ni hacia donde pueden apuntar. Links invalidos o con tipos desconocidos pasan sin deteccion.

**Despues**: .stem define `links.allowed: [blocks, parent, reference]` y reglas por tipo como `links.blocks.target: "../tasks/*.md"`. La validacion reporta errores cuando un documento usa un tipo de link no permitido o apunta a un target que no matchea el pattern esperado.

## Criterios de Aceptacion (semanticos)

- [ ] .stem con `links.allowed` restringe tipos de link validos
- [ ] Link con tipo no listado en allowed genera error de validacion
- [ ] Target pattern en .stem se valida con glob matching
- [ ] Links sin schema de links en .stem no se validan (permisivo por default)

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-links-stem-schema.md) | Tipificar links en .stem con allowed types y target patterns |
| [T002](T002-link-validation-rules.md) | Validar links contra schema definido en .stem |

## Fuente de verdad

- `internal/rules/rules.go` — StemFile.Links (map[string]any → tipificar)
- `internal/rules/validate.go` — Validate function (agregar link validation)
