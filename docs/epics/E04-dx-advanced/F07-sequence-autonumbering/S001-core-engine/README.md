---
estado: Specified
tipo: historia
cliente: Platform Owner
---
# S001: Core Engine

**Feature**: [F07 Sequence Auto-numbering](../README.md)
**Capacidad**: `rootline describe <dir> --field schema.id.next` retorna el proximo ID de secuencia basado en archivos existentes en el directorio

## Antes / Despues

**Antes**: `SchemaField` no tiene campos Prefix, Digits ni Next. El comando `rootline describe --field schema.id.next` falla con "key 'id' not found". No existe mecanismo nativo de auto-numbering en rootline.

**Despues**: Un `.stem` con `id: {type: sequence, prefix: T, digits: 3}` produce que `rootline describe <dir> --field schema.id.next` retorne "T004" si T001, T002, T003 ya existen. El campo `next` es computado en tiempo de describe, no almacenado.

## Criterios de Aceptacion (semanticos)

- [ ] `rootline describe <dir-con-T001-T002> --field schema.id.next` retorna "T003"
- [ ] Directorio vacio con sequence stem retorna "T001"
- [ ] Archivos que no matchean el patron (ej: README.md) son ignorados
- [ ] `go test ./internal/rules/ -run TestSequence` pasa
- [ ] `go test ./cmd/rootline/ -run TestDescribeSequence` pasa

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-extend-schema-field.md) | Agregar Prefix/Digits/Next a SchemaField struct |
| [T002](T002-compute-next-sequence.md) | Implementar computeNextSequence y wiring en NewDescribeResult |
| [T003](T003-sequence-tests.md) | Tests unitarios y CLI end-to-end para sequence |

## Fuente de verdad

- `internal/rules/rules.go:88` — SchemaField struct actual
- `internal/rules/describe.go` — NewDescribeResult a modificar
- `internal/rules/describe_test.go` — tests a extender
