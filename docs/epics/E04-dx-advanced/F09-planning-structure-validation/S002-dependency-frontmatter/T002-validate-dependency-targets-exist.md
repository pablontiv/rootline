---
estado: Completed
tipo: documentation
ejecutable_en: 1 sesion
---
# T002: Verificar que rootline graph valida targets de wiki-links

**Story**: [S002 Dependency Wiki-Links](README.md)

[[blocks:T001-add-blocks-schema-to-stem]]

## Contexto

Con T001 completado, el `.stem` declara `blocks` como tipo de link valido. `rootline graph --check` ya detecta broken links (targets que no existen como nodos en el grafo) y ciclos. Este task verifica que el pipeline completo funciona: wiki-link `[[blocks:X]]` → graph edge → broken link detection → cycle detection.

No se necesita codigo Go nuevo. Solo verificar con archivos de test que el comportamiento existente cubre el caso de uso.

## Dependencias

- T001 completado (links schema en .stem)

## Alcance

**In**:
1. Crear archivos de test temporales en un directorio de prueba:
   - Task con `[[blocks:T002-existing]]` donde T002-existing.md existe → graph --check OK
   - Task con `[[blocks:T002-nonexistent]]` donde no existe → graph --check reporta broken link
   - Ciclo: T001 tiene `[[blocks:T002]]`, T002 tiene `[[blocks:T001]]` → graph --check reporta ciclo
2. Ejecutar `rootline graph --check` en cada caso y verificar output
3. Documentar resultados en este task

**Out**: No modificar codigo Go. No escribir tests unitarios (es verificacion manual).

## Estado inicial esperado

- T001 completado: `.stem` tiene `links: { allowed: [blocks] }`
- `rootline graph --check` funciona para wiki-links genericos
- No se ha verificado especificamente con tipo `blocks`

## Criterios de Aceptacion

- `rootline graph --check` en directorio con blocks validos retorna sin errores
- `rootline graph --check` en directorio con broken block target reporta broken link
- `rootline graph --check` en directorio con ciclo de blocks reporta ciclo
- No se necesitaron cambios de codigo Go

## Fuente de verdad

- `internal/graph/graph.go` — BrokenLinks(), DetectCycles()
- `internal/extract/links.go` — ParseLinks()
