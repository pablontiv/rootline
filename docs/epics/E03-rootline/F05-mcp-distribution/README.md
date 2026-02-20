# F05: MCP Server and Distribution

**Epic**: [E03](../README.md)
**Objetivo**: AI assistants pueden consultar Rootline via MCP; binario disponible via Homebrew
**Beneficio**: Diferenciador (MCP server en Go); instalacion zero-friction para usuarios
**Milestone**: MCP server responde a JSON-RPC query; `brew install rootline` funciona

## Scope

**In**: JSON-RPC 2.0 MCP server (stdio + SSE), goreleaser config, Homebrew tap
**Out**: MCP tools beyond CLI commands, custom distribution channels, Docker image

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [MCP Server](S001-mcp-server/) | `rootline serve` expone todos los comandos via JSON-RPC 2.0 |
| S002 | [Release Pipeline](S002-release-pipeline/) | Tags producen binarios multi-plataforma y formula Homebrew |

## Dependencias

- F03 y F04 completados (comandos CLI funcionales para exponer via MCP)

## Fuente de verdad

