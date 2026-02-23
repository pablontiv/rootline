---
estado: Specified
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Implementar JSON-RPC 2.0 server con transporte stdio

**Story**: [S001 MCP Server](README.md)

## Contexto

Rootline usa MCP (Model Context Protocol) basado en JSON-RPC 2.0 como su protocolo de interaccion para AI assistants. El SDK oficial de Go (`modelcontextprotocol/go-sdk`) provee la implementacion del protocolo. El server wrappea el core engine — las mismas funciones que el CLI usa directamente.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/mcp
interfaces:
  - nombre: Server
    metodos:
      - nombre: Start
        input: "ctx context.Context"
        output: error
      - nombre: RegisterTool
        input: "name string, handler ToolHandler"
        output: (none)
dependencias_externas:
  - github.com/modelcontextprotocol/go-sdk
tests:
  - Server inicia y responde a JSON-RPC request
  - Server maneja method not found
  - Server maneja request malformado
  - stdio transport funciona con stdin/stdout
```

## Dependencias

- F01/S001 completado (Go module con dependencia del SDK)

## Alcance

**In**:
1. Server struct que wrappea MCP SDK
2. stdio transport (primary): lee JSON-RPC de stdin, escribe a stdout
3. Cobra command `serve` que inicia el server
4. Manejo de errores JSON-RPC (method not found, invalid params, internal error)
5. Graceful shutdown con context cancellation

**Out**: SSE transport (futuro), authentication, rate limiting

## Estado inicial esperado

- Go module con `modelcontextprotocol/go-sdk` en go.mod
- Cobra skeleton con serve stub

## Criterios de Aceptacion

- `rootline serve` inicia y espera input en stdin
- Request JSON-RPC valido recibe response JSON-RPC
- Method no registrado retorna error -32601
- Request malformado retorna error -32700
- Ctrl+C hace graceful shutdown

## Fuente de verdad

- `src/rootline/README.md` seccion "JSON-RPC protocol"
