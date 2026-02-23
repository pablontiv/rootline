---
estado: Pending
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Tests y documentacion de linked values en derive expressions

**Story**: [S001 Graph Field Resolution](README.md)

[[blocks:T001-resolve-linked-fields]]

## Contexto

Con T001 completado, el derive engine inyecta valores de documentos enlazados en el env. Esta task verifica que las expresiones expr-lang funcionan correctamente con estos valores y documenta la API.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/derive
interfaces: []
dependencias_externas: []
tests:
  - all(blocked_by, {. == 'Completado'}) retorna true cuando todos los blockers estan Completados
  - any(blocked_by, {. == 'Bloqueada'}) retorna true si algun blocker esta Bloqueada
  - blocked_by == nil retorna true cuando no hay links de tipo blocks
  - len(blocked_by) retorna numero correcto de links
  - Expresion con variable de link inexistente retorna error claro
```

## Dependencias

- T001 completado (RecordResolver wired, linked values en env)

## Alcance

**In**:
1. Integration tests con expresiones derive que usan linked values
2. Test con .stem que define `links.blocks.field: blocked_by` y `derive.estado` expression
3. Documentar en package doc comment: que variables se inyectan, que tipos son ([]string), como usar con all/any/len

**Out**: Performance benchmarks, new builtins, aggregate changes

## Estado inicial esperado

- T001 completado — RecordResolver funcional
- expr-lang/expr builtins (all, any, len) disponibles

## Criterios de Aceptacion

- Tests en `internal/derive/links_test.go` cubren all/any/nil/len patterns
- Package doc en `internal/derive/links.go` documenta variables inyectadas
- `go test ./internal/derive/ -run TestLinked -v` pasa

## Fuente de verdad

- `internal/derive/links.go` (nuevo, de T001)
- `internal/derive/builtins.go` (funciones existentes)
