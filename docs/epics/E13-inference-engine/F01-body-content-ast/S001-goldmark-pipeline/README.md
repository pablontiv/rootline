---
estado: Specified
tipo: historia
---
# S001: goldmark Pipeline Integration

**Feature**: [F01 Body Content AST Infrastructure](../README.md)
**Capacidad**: goldmark parsea body content a AST sin romper contratos JSON existentes
**Cubre**: Milestone de F01 — Record tiene AST opcional

## Antes / Despues

**Antes**: Record.Body es un string plano. ParseLinks usa regex con exclusion de fenced code blocks. No hay AST disponible para analisis estructural. fix.go:236 usa `strings.Index("\n## ")` para insertar wiki-links.

**Despues**: Record tiene campo AST (`json:"-"`) poblado opcionalmente por goldmark. ParseLinksAST produce resultados identicos a ParseLinks pero con precision AST. El overhead es ≤20%.

## Criterios de Aceptacion (semanticos)

- [ ] `go get github.com/yuin/goldmark` añade dependencia sin conflictos
- [ ] Record.AST existe como `ast.Node` con tag `json:"-"` — no se serializa
- [ ] ParseLinksAST produce output identico a ParseLinks en todos los casos existentes
- [ ] Benchmark muestra overhead ≤20% sobre extraccion actual
- [ ] `go test ./... -race` pasa verde

## Invariantes

- INV1: `go test ./... -race` pasa verde en cada commit
  - Verificar: `go test ./... -race`
- INV2: Contratos JSON no cambian (AST no se serializa)
  - Verificar: `go test ./internal/extract/ -run TestJSON -race` (si existe) o verificar tag `json:"-"`
- INV3: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-add-goldmark-ast-field.md) | Añadir goldmark y campo AST a Record |
| [T002](T002-implement-parse-links-ast.md) | Implementar ParseLinksAST con equivalencia |
| [T003](T003-benchmark-ast-overhead.md) | Benchmark overhead y feature flag |

## Fuente de verdad

- `internal/extract/extract.go` — Record struct (lineas 26-34)
- `internal/extract/links.go` — ParseLinks
- `go.mod` — dependencias
