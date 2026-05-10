---
estado: Completed
tipo: task
---
# T001: Remove Rootline MCP support

**Outcome**: [Decommission Pi, MCP, and marketplace publishing](README.md)

## Preserva

- Core CLI commands other than serve keep their existing behavior.
- Rootline remains usable as a CLI-first tool for Pi integrations.

## Contexto

The new direction explicitly removes MCP support from Rootline. Prior scouting found the entry point in cmd/rootline/serve.go, implementation in internal/mcp, tests in internal/mcp and cmd/rootline/commands_test.go, and the github.com/modelcontextprotocol/go-sdk dependency in go.mod.

## Alcance

**In**:
1. Remove cmd/rootline/serve.go and command registration for serve.
2. Remove internal/mcp implementation and MCP-specific tests.
3. Remove the modelcontextprotocol Go dependency via go mod tidy.
4. Update command tests and active docs that mention rootline serve, MCP, JSON-RPC, Claude Desktop MCP setup, or MCP tools.

**Out**:
1. Do not replace MCP with another RPC/API surface.
2. Do not change core Rootline CLI semantics unrelated to MCP.
3. Do not remove local Claude skills solely because they mention roadmap workflows unless those references are active MCP instructions.

## Estado inicial esperado

Rootline has cmd/rootline/serve.go, internal/mcp/**, MCP tests, go.mod dependency github.com/modelcontextprotocol/go-sdk, README/docs MCP references, and roadmap history mentioning MCP.

## Criterios de Aceptación

- rg 'internal/mcp|modelcontextprotocol|rootline serve|Start MCP server' returns no active code or active product-doc references outside historical roadmap records that are intentionally retained.
- go test ./... returns exit 0.
- go mod tidy leaves no direct github.com/modelcontextprotocol/go-sdk requirement.
- rootline --help no longer lists serve.

## Fuente de verdad

- cmd/rootline/serve.go
- internal/mcp/
- go.mod
- README.md
- docs/json-rpc.md
- CLAUDE.md
- .claude/skills/rootline/
