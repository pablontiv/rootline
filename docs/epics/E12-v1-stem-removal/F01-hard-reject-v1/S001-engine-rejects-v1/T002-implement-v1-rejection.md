---
estado: Specified
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Implementar rechazo de version 0/1 en ParseStem

**Story**: [S001 Engine rechaza stems v1](README.md)
**Contribuye a**: `rootline validate` con stem v1 produce error, no warning

[[blocks:T001-migrate-test-stems-to-v2]]

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`
- INV2: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

Despues de migrar todos los test stems a v2 (T001), el engine puede rechazar v1 sin romper tests. Hay 4 puntos de cambio: (1) ParseStem en rules.go debe retornar error para version 0/1, (2) rejectLevelsInV2 en rules.go ya no es necesaria, (3) hierarchy.go tiene un branch `if merged.Version == 2` que debe volverse incondicional, (4) stemhealth.go tiene un check "version-deprecated" que ya no aplica (el error ocurre antes).

## Especificacion Tecnica

```yaml
archivos:
  - path: internal/rules/rules.go
    cambios:
      - En ParseStem (o funcion que carga y valida version): agregar check
        "if sf.Version == 0 || sf.Version == 1 { return error }"
        con mensaje: "stem version %d is no longer supported — upgrade with
        rootline v0.x migrate --to-v2 first"
      - Eliminar funcion rejectLevelsInV2() y su invocacion
  - path: internal/rules/hierarchy.go
    cambios:
      - Linea 15 aprox: quitar condicion "if merged.Version == 2"
      - El bloque de match filtering se ejecuta siempre
  - path: internal/rules/stemhealth.go
    cambios:
      - Eliminar check "version-deprecated" (lineas 257-268 aprox)
      - Actualizar slice/registro de checks disponibles
```

## Dependencias

- T001 completado (todos los test stems ya son v2)

## Alcance

**In**:
1. Agregar rechazo de version 0/1 en ParseStem con error descriptivo
2. Eliminar `rejectLevelsInV2()` de rules.go
3. Quitar branch v1/v2 en hierarchy.go (match filtering incondicional)
4. Eliminar check "version-deprecated" de stemhealth.go

**Out**: No modificar tests (eso es T003). No tocar codigo de migracion (eso es F02).

## Estado inicial esperado

- T001 completado: todos los test stems son v2
- `go test ./... -race` pasa verde

## Criterios de Aceptacion

- `go test ./... -race` pasa verde
- `go vet ./...` sin warnings
- `grep -n "rejectLevelsInV2" internal/rules/rules.go` retorna 0 resultados
- `grep -n "version-deprecated" internal/rules/stemhealth.go` retorna 0 resultados
- `grep -n "Version == 2" internal/rules/hierarchy.go` retorna 0 resultados

## Fuente de verdad

- `internal/rules/rules.go` — ParseStem, rejectLevelsInV2
- `internal/rules/hierarchy.go` — branch Version == 2
- `internal/rules/stemhealth.go` — check version-deprecated
