---
estado: Specified
tipo: software-module
ejecutable_en: 1 sesion
hold: Pendiente aprobación de usuario
---
# T002: Registrar MCP tools: query, validate, describe, tree, stats

**Story**: [S001 MCP Server](README.md)

## Contexto

Cada comando CLI se expone como un MCP tool con input/output schemas que matchean los contratos JSON del CLI. Los MCP tools son el mapeo 1:1 de CLI commands a JSON-RPC methods. El AI assistant invoca el mismo contrato que un usuario de CLI.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/mcp
interfaces:
  - nombre: ToolHandler
    metodos:
      - nombre: Handle
        input: "ctx context.Context, params json.RawMessage"
        output: "any, error"
dependencias_externas:
  - github.com/modelcontextprotocol/go-sdk
tests:
  - Tool query acepta params y retorna QueryResult
  - Tool validate acepta file path y retorna ValidationResult
  - Tool describe acepta dir path y retorna DescribeResult
  - Tool tree acepta path y retorna tree JSON
  - Tool stats retorna stats JSON
  - Cada tool declara input schema correcto
```

## Dependencias

- T001 completado (server framework)
- F03 y F04 completados (commands funcionales para exponer)

## Alcance

**In**:
1. Registrar 5 MCP tools: query, validate, describe, tree, stats
2. Cada tool con inputSchema (JSON Schema) describiendo parametros
3. Cada tool delegando al core engine (mismas funciones que CLI)
4. Output schemas matchean contratos JSON del CLI

**Out**: MCP resources, MCP prompts, explain tool (deferred)

## Estado inicial esperado

- MCP server funcional (T001)
- Query engine, validation engine, describe, tree, stats funcionales

## Criterios de Aceptacion

- `tools/list` JSON-RPC retorna 5 tools con nombres y schemas
- `tools/call` con tool "query" y params validos retorna QueryResult JSON
- `tools/call` con tool "validate" retorna ValidationResult JSON
- `tools/call` con tool "describe" retorna DescribeResult JSON
- Input schemas son JSON Schema validos

## Fuente de verdad

- `src/rootline/README.md` seccion "AI-native" y "JSON-RPC protocol"
