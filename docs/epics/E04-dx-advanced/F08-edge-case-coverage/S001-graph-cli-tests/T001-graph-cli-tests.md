---
estado: Completado
tipo: software-test
ejecutable_en: 1 sesion
---
# T001: Crear cmd/rootline/graph_test.go con 8 tests

**Story**: [S001 Graph CLI Tests](README.md)

## Contexto

`cmd/rootline/graph.go` implementa el comando `graph` con 3 modos: JSON output (default), --check mode (texto plano), y diagram output (--format dot|mermaid). Ninguno tiene tests de CLI. El patron existente usa `runCmd(t, args...)` y `setupTestDir(t)` definidos en `commands_test.go`. Los links en markdown se crean con la sintaxis wiki `[[target]]` o `[[type:target]]` que extrae `internal/extract/links.go`.

## Especificacion Tecnica

```yaml
tipo: software-test
proyecto: rootline
paquete: cmd/rootline
cobertura_objetivo: 100% de runGraph, renderDOT, renderMermaid
tipos_test:
  - integration
fixtures:
  - directorio con doc1.md que tiene [[doc2.md]] en el body
  - directorio con ciclo: doc1 -> doc2 -> doc1
  - directorio con broken link: [[nonexistent.md]]
```

## Alcance

**In**: Crear `/opt/rootline/cmd/rootline/graph_test.go` con package main, con estos 8 tests:
1. `TestGraphJSON_Empty` — dir sin links, JSON con `kind: rootline/graph`, `edges: []`, `cycles: []`
2. `TestGraphJSON_WithLinks` — dir con `[[doc2.md]]`, JSON incluye el edge
3. `TestGraphCheck_Clean` — `--check` sin ciclos → output contiene "No cycles or broken links"
4. `TestGraphCheck_WithCycle` — `--check` con A→B→A → retorna error ErrValidationFailed, output contiene "Cycles found: 1"
5. `TestGraphCheck_WithBrokenLink` — `--check` con link a nonexistent.md → error, output "Broken links: 1"
6. `TestGraphFormat_DOT` — `-o table --format dot` → output empieza con "digraph {"
7. `TestGraphFormat_Mermaid` — `--format mermaid` → output empieza con "graph TD;"
8. `TestGraphFormat_Invalid` — `--format xyz` → retorna error

**Out**: Cambios a graph.go, tests de graph internal package

## Estado inicial esperado

- `cmd/rootline/graph.go` existe e implementa runGraph, renderDOT, renderMermaid
- `cmd/rootline/commands_test.go` existe con resetFlags(), runCmd(), setupTestDir()
- `internal/extract/links.go` extrae `[[target]]` del body del markdown
- `cmd/rootline/root.go` define ErrValidationFailed

## Criterios de Aceptacion

- `go test ./cmd/rootline/ -run TestGraph -v` pasa los 8 tests
- `go test ./... -race` pasa sin regresiones
- Cada test usa t.TempDir() y mustWriteFile, no deja archivos temporales

## Fuente de verdad

- `cmd/rootline/graph.go` — lineas 44-138: runGraph, renderDOT, renderMermaid
- `cmd/rootline/commands_test.go` — patron de tests a seguir (TestQueryJSON, TestStatsJSON)
- `cmd/rootline/helpers_test.go` — mustWriteFile disponible
- `internal/extract/links.go:15` — wikilinkRe = `\[\[([^\]]+)\]\]`
