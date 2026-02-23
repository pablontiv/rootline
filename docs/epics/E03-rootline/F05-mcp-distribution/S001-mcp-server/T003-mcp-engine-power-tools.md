---
estado: Specified
tipo: software-module
ejecutable_en: 1 sesion
---
# T003: Registrar MCP tools de engine-power: explain, fix, graph

**Story**: [S001 MCP Server](README.md)

[[blocks:T002-mcp-tools-registration]]

## Contexto

Con el MCP server funcional (T001) y los 5 tools core registrados (T002), esta task expande el surface area para AI consumers registrando 3 tools adicionales que exponen capacidades avanzadas del engine: explain (field tracing), fix (proposal-based corrections), y graph (dependency visualization).

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/mcp
interfaces:
  - nombre: ToolHandler (reuse from T002)
    metodos:
      - nombre: Handle
        input: "ctx context.Context, params json.RawMessage"
        output: "any, error"
dependencias_externas:
  - github.com/modelcontextprotocol/go-sdk
tests:
  - Tool explain acepta path y retorna ExplainResult JSON
  - Tool fix acepta path + dry_run bool y retorna proposals JSON
  - Tool graph acepta path + check bool + format string y retorna graph data
  - Cada tool declara inputSchema correcto
```

## Dependencias

- T001 completado (server framework)
- T002 completado (core tools registrados, pattern establecido)

## Alcance

**In**:
1. Registrar MCP tool `explain` con params `{"path": "string"}`
2. Registrar MCP tool `fix` con params `{"path": "string", "dry_run": bool, "all": bool}`
3. Registrar MCP tool `graph` con params `{"path": "string", "check": bool, "format": "dot|mermaid"}`
4. InputSchema JSON Schema por tool
5. Output matchea contratos JSON existentes del CLI

**Out**: MCP resources, MCP prompts, new tool types not in CLI

## Estado inicial esperado

- MCP server funcional (T001)
- 5 tools core registrados con pattern establecido (T002)
- explain, fix, graph commands funcionales en CLI

## Criterios de Aceptacion

- `tools/list` JSON-RPC retorna 8 tools (5 core + 3 engine-power)
- `tools/call` con tool "explain" retorna ExplainResult JSON valido
- `tools/call` con tool "fix" con dry_run=true retorna proposals sin modificar archivos
- `tools/call` con tool "graph" con check=true retorna cycle/broken link analysis
- Input schemas son JSON Schema validos

## Fuente de verdad

- `internal/mcp/tools.go` (pattern de T002)
- `cmd/rootline/explain.go`, `cmd/rootline/fix.go`, `cmd/rootline/graph.go` (contratos JSON)
