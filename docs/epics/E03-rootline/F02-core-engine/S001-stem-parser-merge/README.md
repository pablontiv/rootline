# S001: Stem Parser and Merge

**Feature**: [F02 Core Engine](../README.md)
**Estado**: Specified
**Cliente**: Platform Owner
**Capacidad**: Rootline resuelve el schema efectivo de cualquier directorio via walk-up discovery y type-driven merge de archivos .stem

## Antes / Despues

**Antes**: El diseno de .stem existe solo en documentos (I5). No hay codigo que parsee archivos .stem ni que compute el schema efectivo de un directorio. La herencia parent-to-child es un concepto, no una implementacion.

**Despues**: Engine carga archivos .stem como YAML, camina hacia arriba hasta .git recolectando .stem files, y los mergea top-down con reglas type-driven (maps merge, arrays replace, scalars replace, null removes). `EffectiveSchema(path)` retorna el schema completo para cualquier directorio.

## Criterios de Aceptacion (semanticos)

- [ ] Un .stem hijo que override un campo scalar reemplaza el valor del padre
- [ ] Un .stem hijo que define schema fields adicionales los agrega al schema del padre
- [ ] Un .stem hijo con `null` en un campo remueve ese campo del schema efectivo
- [ ] Arrays en .stem hijo reemplazan completamente los del padre

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-stem-yaml-parser.md) | Parsear archivos .stem YAML a structs Go |
| [T002](T002-walkup-discovery.md) | Walk-up desde target path hasta .git recolectando .stem files |
| [T003](T003-type-driven-merge.md) | Merge type-driven de lista ordenada de StemFiles |

## Fuente de verdad

- `src/rootline/docs/research/I5-describe-contract.md` — merge algorithm completo
- `src/rootline/docs/intent/v0-rootline.md` — Design Principles
