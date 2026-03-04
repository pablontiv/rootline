---
estado: Specified
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Implementar ParseLinksAST con equivalencia a ParseLinks

**Story**: [S001 goldmark Pipeline Integration](README.md)
**Contribuye a**: ParseLinksAST produce output identico a ParseLinks

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`
- INV3: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

ParseLinks en `internal/extract/links.go` usa regex con exclusion manual de fenced code blocks para extraer wiki-links `[[target]]` del body. Una version AST-based seria mas precisa (no confundiria links en code blocks) y habilitaria extraccion contextual futura.

## Alcance

**In**:
1. Implementar `ParseLinksAST(node ast.Node, source []byte) []Link` en links.go (o links_ast.go)
2. Walk del AST excluyendo FencedCodeBlock y CodeBlock nodes
3. Extraer wiki-links del texto de los nodos restantes con el mismo regex
4. Test de equivalencia: ParseLinksAST produce mismo output que ParseLinks en los 14 casos de test existentes

**Out**: Reemplazar ParseLinks por ParseLinksAST (decision futura post-benchmark). Extraccion de secciones (S002).

## Estado inicial esperado

- T001 completado (Record tiene AST)
- ParseLinks funciona con 14 test cases
- links_test.go tiene cobertura de edge cases

## Criterios de Aceptacion

- `ParseLinksAST` existe y compila
- Test de equivalencia compara output de ParseLinks vs ParseLinksAST en todos los test cases existentes — 0 diferencias
- Test adicional: link dentro de fenced code block — ParseLinksAST lo excluye correctamente
- `go test ./internal/extract/ -race` pasa verde
- Coverage de links.go o links_ast.go ≥85%

## Fuente de verdad

- `internal/extract/links.go` — ParseLinks, regex patterns, fenced code block exclusion
- `internal/extract/links_test.go` — 14 test cases existentes
