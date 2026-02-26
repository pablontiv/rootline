---
estado: Specified
tipo: software-module
ejecutable_en: 1 sesion
---
# T003: Fix describe to expand levels for file targets

**Story**: [S002 Describe, Infer and Stemhealth](README.md)
**Contribuye a**: describe soporta levels como herramienta first-class

## Preserva

- INV1: Todos los tests existentes pasan sin modificacion
  - Verificar: `go test ./... -race`

## Contexto

`rootline describe` usa `WalkUp + MergeStemFiles` para construir el effective schema. Esto funciona para directorios, pero cuando el target es un archivo (ej: un task file T001-xxx.md), no expande `levels:` y el schema mostrado no incluye campos per-level como `tipo` o `ejecutable_en`. `validate` y `fix` ya usan `ResolveForRecord` que si expande levels.

## Especificacion Tecnica

En `cmd/rootline/describe.go`, funcion `runDescribe`:

Agregar branching por tipo de target:
- Si es archivo (`os.Stat` + `!IsDir()`): usar `ResolveForRecord(dir, path)` para obtener schema con levels expandidos
- Si es directorio: mantener `WalkUp + MergeStemFiles` actual (no hay record especifico contra el cual expandir)

## Alcance

**In**:
1. Agregar `os.Stat` check en `runDescribe` para detectar file vs directory
2. Para file targets, usar `rules.ResolveForRecord(dir, args[0])` en vez de `WalkUp + MergeStemFiles`
3. Agregar import `os`
4. Agregar e2e test en `internal/e2e/hierarchy_test.go`

**Out**: Cambios al describe output format, cambios al table rendering

## Estado inicial esperado

- `cmd/rootline/describe.go` no importa `os` y no distingue file vs directory
- `rootline describe T001-xxx.md` muestra solo schema raiz (estado, cliente)

## Criterios de Aceptacion

- `go test ./internal/e2e/ -run TestHierarchyLevelsDescribeFile -v` pasa
- `rootline describe <task-file.md>` muestra campos task-level (tipo, ejecutable_en, id)
- `rootline describe <directory>/` mantiene comportamiento actual (sin levels expansion)
- `go test ./... -race` pasa sin regresiones

## Fuente de verdad

- `cmd/rootline/describe.go` — runDescribe function
- `internal/rules/hierarchy.go` — ResolveForRecord function
- `internal/e2e/hierarchy_test.go` — e2e tests
