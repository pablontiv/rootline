---
tipo: historia
cliente: Platform Owner
---
# S001: MCP Server

**Feature**: [F05 MCP Server and Distribution](../README.md)
**Capacidad**: `rootline serve` expone todos los comandos via JSON-RPC 2.0 (MCP protocol) para consumo por AI assistants

## Antes / Despues

**Antes**: No hay API programatica para AI assistants. Claude Code y otros LLMs no pueden consultar documentacion estructurada directamente. La interaccion requiere CLI output parseado por el LLM.

**Despues**: `rootline serve` inicia un MCP server que expone query, validate, describe, tree, stats como tools JSON-RPC 2.0. AI assistants consumen los mismos contratos JSON que el CLI. Diferenciador: MCP server en Go (mayoria son TypeScript).

## Criterios de Aceptacion (semanticos)

- [ ] MCP server responde a requests JSON-RPC query con rows correctos
- [ ] MCP server expone los mismos contratos que el CLI
- [ ] Funciona con Claude Code como MCP server configurado

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-jsonrpc-server.md) | Implementar JSON-RPC 2.0 server con transporte stdio |
| [T002](T002-mcp-tools-registration.md) | Registrar MCP tools: query, validate, describe, tree, stats |
| [T003](T003-mcp-engine-power-tools.md) | Registrar MCP tools de engine-power: explain, fix, graph |

## Fuente de verdad

- `internal/mcp/mcp.go` — Server wrapper, stdio transport
- `internal/mcp/tools.go` — 8 MCP tool registrations and handlers
- `cmd/rootline/serve.go` — CLI entry point
