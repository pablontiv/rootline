---
estado: Completed
---
# MCP Server

Rootline exposes its engine via the **Model Context Protocol (MCP)** over JSON-RPC 2.0.

Two transport modes:

| Mode | Command | Use case |
|------|---------|----------|
| **HTTP** (default) | `rootline serve --addr 127.0.0.1:9200` | Multi-consumer: kedral, hooks, dashboards |
| **Stdio** (legacy) | `rootline serve --stdio` | Claude Code MCP client config |

HTTP mode uses **Streamable HTTP** (stateless) — each POST to `/mcp` is an independent MCP session. A `/health` endpoint returns server status.

AI assistants call the same contracts as the CLI — no separate API layer.

---

## Setup

### Claude Desktop (stdio)

Add to your Claude Desktop configuration (`~/.claude/mcp.json` or project `.mcp.json`):

```json
{
  "mcpServers": {
    "rootline": {
      "command": "rootline",
      "args": ["serve", "--stdio"]
    }
  }
}
```

### HTTP daemon

```bash
rootline serve --addr 127.0.0.1:9200
# Health check: curl http://127.0.0.1:9200/health
# MCP endpoint: POST http://127.0.0.1:9200/mcp
```

### Any MCP Client

**Stdio:** Connect to `rootline serve --stdio` as a subprocess (JSON-RPC 2.0 over stdin/stdout).

**HTTP:** POST JSON-RPC requests to `http://<addr>/mcp`. Requires MCP initialize handshake. Responses use SSE format (`text/event-stream`).

---

## Tool Catalog

Rootline provides 12 functional MCP tools that map 1:1 to CLI commands.

### query

Search and filter records by frontmatter fields using expr-lang expressions.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | yes | Directory to scan (absolute path) |
| `where` | string[] | no | Filter expressions (expr-lang syntax) |
| `count` | bool | no | Return count instead of records |
| `limit` | int | no | Limit number of results (0 = unlimited) |

**Returns**: `rootline/query` — rows with frontmatter, derived, and aggregated fields.

### validate

Check documents against `.stem` schema rules.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | yes | Directory to validate (absolute path) |
| `where` | string[] | no | Filter expressions for validation scope |

**Returns**: `rootline/validate-batch` — per-file validation results with errors and warnings.

### describe

Show the effective `.stem` schema for a directory.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | yes | Directory to describe (absolute path) |

**Returns**: `rootline/describe` — merged schema with field types, sources, and inheritance chain.

### tree

Show hierarchical tree of records with completion counts.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | yes | Directory to scan (absolute path) |
| `where` | string[] | no | Filter expressions |

**Returns**: `rootline/tree` — nested tree with completed/total counts per node.

### stats

Show aggregate statistics (by estado, by tipo) for records.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | yes | Directory to scan (absolute path) |
| `where` | string[] | no | Filter expressions |

**Returns**: `rootline/stats` — counts grouped by `estado` and `tipo`.

### explain

Trace why a document has a given state: field origins, derivation chain, and validation errors.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | yes | File to explain (absolute path) |

**Returns**: `rootline/explain` — per-field origin (frontmatter, schema, derived, aggregate), `.stem` chain, and validation errors.

### fix

Analyze validation errors and return fix proposals. Always read-only — never modifies files.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | yes | Directory to scan (absolute path) |
| `dry_run` | bool | no | Always true for MCP (proposals only) |
| `all` | bool | no | Fix all files in scope |

**Returns**: `rootline/proposals` — typed proposals (extend_enum, correct_value, add_field, migrate_value, etc.).

### set

Mutate frontmatter fields or section bodies in a document, with schema validation. This is the first mutation capability in the MCP server — all other tools are read-only.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | yes | File to mutate (absolute path) |
| `fields` | map[string]string | no | Frontmatter or section assignments (`field → value`) |
| `append_fields` | map[string]string | no | Section append assignments (`field → content`) |
| `create_sections` | bool | no | Create sections that do not exist when appending |
| `dry_run` | bool | no | Return proposed diff without writing to disk |

**Returns**: `rootline/set` — list of applied mutations and post-validation result. If `dry_run` is true, returns the proposed diff only.

**SetInput schema**:

```json
{
  "path": "/home/user/project/docs/overview.md",
  "fields": {
    "estado": "Completed",
    "## Summary": "Describes the API surface."
  },
  "append_fields": {
    "## Changelog": "- Added /v2/status endpoint"
  },
  "dry_run": false
}
```

**Notes**:
- Pre-validation checks enum membership and type before applying changes.
- Post-validation runs the full schema check after applying; failures trigger rollback.
- `append_fields` requires the section to already exist unless `create_sections` is true.

### graph

Build dependency graph from `[[wiki-links]]` with cycle detection and broken link analysis.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | yes | Directory to scan (absolute path) |
| `check` | bool | no | Validate only (cycles + broken links) |
| `format` | string | no | Output format: `dot` or `mermaid` (default: JSON) |

**Returns**: `rootline/graph` — nodes, edges, cycles, and broken links (with fuzzy `suggestions` for close matches). Or DOT/Mermaid text if `format` is set.

### trace

Follow reference chains through the document graph via BFS traversal. Shows connected documents and their estado.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | yes | Absolute path to the starting file |
| `reverse` | bool | no | Follow incoming references instead of outgoing |
| `depth` | int | no | Max traversal depth (0 = unlimited) |
| `edge_type` | string | no | Filter by edge type (field name) |

**Returns**: Graph traversal result with connected documents and their fields.

### new

Create a new document with frontmatter scaffolded from the effective `.stem` schema of the target directory.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | yes | Absolute path to the new document file |
| `content` | string | no | Full document content (frontmatter + body); if provided, used directly instead of generating from schema |
| `force` | bool | no | Overwrite existing file |
| `dry_run` | bool | no | Show generated content without writing file |

**Returns**: `rootline/new` or text output — created file path and status, or full content if `dry_run` is true.

### health

Return server health status: version, uptime, goroutines.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|

**Returns**: Server health status including version, uptime, and runtime metrics.

---

## Example

### Request

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "query",
    "arguments": {
      "path": "/home/user/project/docs",
      "where": ["estado == 'Pending'"],
      "count": true
    }
  }
}
```

### Response

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"version\":1,\"kind\":\"rootline/query\",\"meta\":{\"count\":3},\"rows\":[]}"
      }
    ]
  }
}
```

---

## Protocol Details

- **Transport**: stdio (stdin/stdout)
- **Protocol**: JSON-RPC 2.0 via MCP SDK
- **All results** are JSON with `"version": 1` for contract stability
- **No authentication** — the server runs as a local subprocess
- **Source**: `internal/mcp/mcp.go` (server), `internal/mcp/tools.go` (tool handlers)
