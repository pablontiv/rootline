---
estado: Completed
tipo: software-test
ejecutable_en: 1 sesion
---
# T002: Tests e2e con dependency chains reales

**Story**: [S002 Dependency State Propagation](README.md)

[[blocks:T001-blocked-unblocked-derivation]]

## Contexto

Verificar que el pipeline completo (extract → derive con RecordResolver → query) funciona end-to-end con dependency chains realistas. Cubrir edge cases: chains lineales, multiple blockers, ciclos, target inexistente.

## Especificacion Tecnica

```yaml
tipo: software-test
proyecto: rootline
paquete: internal/e2e
cobertura_objetivo: 100% de edge cases documentados
tipos_test:
  - e2e
fixtures:
  - testdata/links/ con .stem y documentos con [[blocks:]] links
```

## Dependencias

- S002/T001 completado (derive expression configurada)
- S001/T001-T002 completados (RecordResolver funcional)

## Alcance

**In**:
1. Fixture: A blocks B, B blocks C, C=Completado → B derived Pending, A derived Bloqueada
2. Fixture: A blocks B, A blocks C, B=Completado, C=Pending → A derived Bloqueada
3. Fixture: A blocks B, B blocks A → deteccion de ciclo, error graceful
4. Fixture: A blocks NonExistent → skip graceful, A estado sin modificar
5. Fixture: A sin links → A estado original preservado

**Out**: Performance benchmarks, stress tests, integration con CI

## Estado inicial esperado

- Derive pipeline con RecordResolver funcional
- .stem con derive expression para blocked_by

## Criterios de Aceptacion

- `go test ./internal/e2e/ -run TestDependencyChain -v` pasa
- Chain lineal: valores derivados correctos en cada nivel
- Ciclo: error reportado, no infinite loop
- Target inexistente: skip sin error
- Sin links: estado preservado
- Todos los tests pasan con `-race`

## Fuente de verdad

- `internal/e2e/` (test package existente)
- `internal/derive/links.go` (RecordResolver)
