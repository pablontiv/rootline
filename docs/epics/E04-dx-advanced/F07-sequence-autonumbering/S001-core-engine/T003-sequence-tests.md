---
estado: Completado
tipo: software-test
ejecutable_en: 1 sesion
---
# T003: Tests unitarios y CLI end-to-end para sequence

**Story**: [S001 Core Engine](README.md)

## Contexto

Con T001 y T002 implementados, el comportamiento de sequence debe estar cubierto por tests. Dos niveles: unitario en `internal/rules/describe_test.go` (testeando computeNextSequence directamente) y CLI en `cmd/rootline/commands_test.go` (testeando `rootline describe --field schema.id.next` end-to-end).

## Especificacion Tecnica

```yaml
tipo: software-test
proyecto: rootline
paquete: internal/rules y cmd/rootline
cobertura_objetivo: 100% de computeNextSequence
tipos_test:
  - unit
  - integration
fixtures:
  - directorio temporal con archivos T001-*.md, T002-*.md
  - .stem con id: {type: sequence, prefix: T, digits: 3}
```

## Alcance

**In**:
1. En `internal/rules/describe_test.go`, agregar:
   - `TestComputeNextSequence_Empty` — directorio vacio → "T001"
   - `TestComputeNextSequence_WithExisting` — T001+T002 → "T003"
   - `TestComputeNextSequence_IgnoresNonMatching` — README.md, .stem ignorados
   - `TestComputeNextSequence_PrefixE` — E01+E02 con digits=2 → "E03"
   - `TestDescribeResult_SequenceNext` — DescribeResult.Schema["id"].Next tiene valor
2. En `cmd/rootline/commands_test.go`, agregar:
   - `TestDescribeSequenceField` — crea dir con .stem sequence + T001-x.md, ejecuta `rootline describe --field schema.id.next`, verifica retorna "T002"

**Out**: Tests de otros features, refactor de helpers de test existentes

## Estado inicial esperado

- T001 y T002 completados
- `go test ./internal/rules/ -race` pasa con tests existentes

## Criterios de Aceptacion

- `go test ./internal/rules/ -run TestComputeNextSequence -v` pasa los 4 casos
- `go test ./internal/rules/ -run TestDescribeResult_SequenceNext -v` pasa
- `go test ./cmd/rootline/ -run TestDescribeSequenceField -v` pasa
- `go test ./... -race` pasa sin regresiones

## Fuente de verdad

- `internal/rules/describe_test.go` — archivo a extender
- `cmd/rootline/commands_test.go` — tests CLI existentes a seguir como patron
- `cmd/rootline/helpers_test.go` — mustWriteFile, setupTestDir disponibles
