---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T003: Skip correct_value cuando extend_enum hizo el valor valido

**Story**: [S001 Fix Priority Conflicts](README.md)

## Contexto

`applyProposals` en fix.go aplica `extend_enum` primero (agrega "Obsoleto" al enum del .stem), luego aplica `correct_value` que fue generado ANTES del extend. El `correct_value` tiene `From: "Obsoleto", To: "Completado"` — un fix que ya no aplica porque "Obsoleto" es ahora un valor valido del enum.

Resultado: archivos con `estado: Obsoleto` se cambian incorrectamente a `Completado`.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
archivo: cmd/rootline/fix.go
funcion: applyProposals
cambio: |
  Despues del loop de extend_enum:
  1. Re-leer el stem con rules.WalkUp(root) + MergeStemFiles
  2. En el case CorrectValue, antes de aplicar:
     - Verificar si p.From es ahora valido en el enum actualizado
     - Si es valido, skip (continue)
     - Si no, aplicar normalmente
tests:
  - Archivos con estado Obsoleto NO se cambian a Completado despues de extend_enum
  - Archivos con valores realmente invalidos SI se corrigen
```

## Dependencias

- T001 (stem selection correcta para que extend_enum se genere)

## Alcance

**In**: Agregar guard en applyProposals para correct_value post-extend
**Out**: Cambios en el analisis de proposals (eso es T002)

## Estado inicial esperado

- extend_enum agrega "Obsoleto" al .stem
- correct_value proposals para "Obsoleto" -> "Completado" existen en el report

## Criterios de Aceptacion

- `rootline fix --all` no cambia archivos con `estado: Obsoleto` a `Completado`
- `rootline fix --all` SI corrige valores realmente invalidos (ej: typos)
- `go test ./cmd/rootline/...` pasa

## Fuente de verdad

- `cmd/rootline/fix.go` funcion applyProposals
