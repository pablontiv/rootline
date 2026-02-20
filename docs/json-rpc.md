---
estado: Planned
---
# JSON-RPC Protocol (Planned)

Rootline will use **JSON-RPC 2.0** as its interaction protocol via the MCP server.

All core capabilities will be exposed as methods:

- `query`
- `describe`
- `explain`
- `validate`
- `tree`
- `stats`

The query contract maps directly to JSON-RPC `params`.

## Example Request

```json
{
  "jsonrpc": "2.0",
  "id": "q1",
  "method": "query",
  "params": {
    "from": "docs/",
    "where": {
      "eq": ["state.visibility", "public"]
    }
  }
}
```

## Example Response

```json
{
  "jsonrpc": "2.0",
  "id": "q1",
  "result": {
    "version": 1,
    "kind": "rootline/query",
    "meta": { "count": 1 },
    "rows": []
  }
}
```
