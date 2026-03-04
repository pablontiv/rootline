---
estado: Specified
tipo: test
ejecutable_en: 1 sesion
---
# T003: Tests de integracion para apply

**Story**: [S003 Apply Command](README.md)
**Contribuye a**: Apply command validado end-to-end

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`
- INV3: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

Apply modifica tanto .stem como documentos. Tests de integracion validan que el pipeline analyze → apply produce resultados correctos y que archivos modificados siguen siendo validos.

## Alcance

**In**:
1. Fixture: directorio con .stem incompleto + documentos → analyze → apply → validate
2. Test: apply extend_enum → .stem tiene nuevo valor → validate pasa
3. Test: apply add_field → documentos tienen nuevo campo → validate pasa
4. Test: --dry-run no modifica archivos
5. Test: inferencias requires_agent se saltan con warning

**Out**: Tests de rendimiento. Tests con agent real.

## Estado inicial esperado

- S003/T001-T002 completados (apply funcional)
- Fixtures de analyze disponibles

## Criterios de Aceptacion

- Test: pipeline analyze → apply → validate sin errores
- Test: --dry-run output muestra cambios pero archivos unchanged
- Test: requires_agent inferencias producen warning en stderr
- `go test ./internal/e2e/ -race` pasa verde

## Fuente de verdad

- `internal/e2e/` — tests e2e existentes como referencia
- `cmd/rootline/apply.go` — comando bajo test
