# Rootline — Intent Document v0

**Fecha**: 2026-02-17
**Estado**: Draft
**Origen**: Research session (docrules-research-2026.md) + reality check + branding

---

## 1. Identity

Rootline is a **file-based database and constraint engine** for structured documentation.

It treats the filesystem as the model: directories are tables, files are records,
metadata is extracted from YAML frontmatter, and structure is inherited along the
directory tree via `.stem` files with parent-to-child inheritance.

The full vision document is the project [README.md](../../README.md).

---

## 2. Problem Statement

### Origin

In the homeserver automation project (`/opt/homeserver/automation/`), documentation
logic for markdown files is scattered across multiple consumers with no single
source of truth.

### Evidence (from codebase analysis, 2026-02-17)

| Finding | Detail |
|---------|--------|
| **7 independent parsing systems** (18 query patterns across 9 distinct consumers — see I1 and I7 for full inventory) | roadmap skill, service skill, module skill, operation skill, instance skill, write hook, PRD queries |
| **4 different regex patterns** | For the same `estado:` field — Python regex, bash grep (x2), agent prompt |
| **5 formats for governance rules** | Markdown prose, Bash patterns, HCL config, Python regex, JSON prompt hooks |
| **0 unified schemas** | Valid values for Task/PRD metadata not formally defined anywhere |
| **Silent failures** | Changing `estado:` format breaks `/roadmap pending` with empty output, no error |

### Concrete Incidents

1. **Tracking drift (E02/F01)**: 3 Tasks were `Completado` in their files but Feature README showed 0%.
   Cause: state lived in two places (Task file + README table). No hook validated consistency.

2. **Format fragility**: `/service` validates Task type with `grep -q "servicio-docker"`.
   `/module` validates with identical grep but different type string. Adding a new type
   requires updating 6 skill files independently.

3. **Prompt-based validation**: The Task Write hook (`.claude/settings.json:100-106`) uses
   an LLM agent prompt to validate structure — not deterministic, not testable, not reliable.

### Root Cause

The logic is dispersed **because there is no canonical place that defines what a valid
document looks like**. Each consumer re-implements discovery, parsing, validation, and
derivation independently.

Rootline solves this by making `.stem` the single source of truth for document structure.

### What Rootline Replaces

| Current logic | Where it lives today | Where it would live |
|---------------|---------------------|---------------------|
| Find Task by ID | 6 skills (`find` + `grep`) | `rootline query --id T005` |
| Validate Task type | 6 skills (inline grep) | `rootline validate T005.md` |
| Validate valid states | Write hook (hardcoded) | `.stem` valid_values |
| Derive Story state | `/roadmap view` (inline Python) | `rootline tree --derive` |
| List by state | `/roadmap pending` (regex) | `rootline query --where 'estado eq Pending'` |
| Full tree view | `/roadmap view` (rebuild each time) | `rootline tree` |
| Add new document type | Edit hook + N skills | Edit `.stem` |

**Estimated reduction**: ~60% of logic in skills eliminated, replaced by CLI calls.

---

## 3. Architecture

### Core Model

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  CLI        │────→│              │     │ docs/*.md   │
│  (cobra)    │     │ Core Engine  │←───→│ .stem files │
└─────────────┘     │ (Go pkg)     │     └─────────────┘
                    │              │
┌─────────────┐     │ - parser/    │
│  MCP Server │────→│ - rules/     │
│  (serve)    │     │ - index/     │
└─────────────┘     │ - query/     │
                    └──────────────┘
```

- **CLI** calls Core Engine directly (Go function calls, no serialization)
- **MCP Server** wraps Core Engine via JSON-RPC 2.0 (MCP protocol)
- Both produce the same data contracts
- Core Engine is the single source of truth

### Pipeline

```
Extraction → Parsing → Rule Loading → Validation → [Derivation] → Query
```

Each stage is an extension point (see Design Principles).
Derivation is deferred — pipeline slot reserved but not active initially (I3).

### Commands

| Command | Purpose | Output |
|---------|---------|--------|
| `rootline query` | Search and filter records | JSON rows |
| `rootline validate` | Check documents against rules | JSON errors |
| `rootline describe <path>` | Show effective schema for a directory | JSON contract |
| `rootline explain <file>` | Trace why a document has a given state | JSON trace |
| `rootline tree` | Hierarchical view with derived state | JSON/text tree |
| `rootline stats` | Summary counts by type/state | JSON stats |
| `rootline serve` | Start MCP server (stdio or SSE) | JSON-RPC 2.0 |

---

## 4. Design Principles

### Extensibility by Design

Every stage of the pipeline should be extensible. The core ships with built-in
implementations, but the architecture must allow extension without modifying core code.

| Pipeline Stage | Extension Point | Built-in | Planned |
|---------------|-----------------|----------|---------|
| **Extraction** | Extractors | Markdown (frontmatter) | YAML, JSON, TOML, OpenAPI |
| **Parsing** | Field parsers | YAML frontmatter | Inline metadata, custom formats |
| **Derivation** | Functions | `slugify`, `lowercase`, `count` | Custom functions |
| **Validation** | Rules | `non_empty`, `requires_if`, `enum` | Custom validators |
| **State** | Derivation rules | `when/then` conditions | Graph-based |
| **Output** | Formatters | JSON, table | Custom formatters |

**Plugin architecture**: TBD — requires dedicated investigation session.
Options: Go interfaces + registry, WASM plugins, expression DSL, or external process.

### Derived State is Never Written Back

Rootline computes derived fields and state at query time.
Source files are never modified. This preserves human-first authoring
and avoids circular dependencies between source and computed data.

### Stable JSON Contracts

All CLI and MCP outputs follow versioned JSON contracts (`"version": 1`).
Breaking changes require version bumps. This enables:
- CI pipeline integration
- Editor/tooling integration
- AI assistant consumption
- Backward-compatible evolution

### Versioning Model (D20)

JSON output schemas carry their own version (`"version": 1`), independent of any future release version.
Breaking changes to output schemas require a version bump. This enables backward-compatible evolution
for CI pipelines, editors, and AI assistants that consume Rootline output.

### Explainability as First-Class

Every validation error, derived state, or computed field must be traceable
to the `.stem` rules that produced it. "Why is this document invalid?" and
"Why does this have state X?" must always be answerable.

---

## 5. Decisions

### Closed

| # | Decision | Result | Rationale |
|---|----------|--------|-----------|
| D1 | Language | **Go** | Portfolio signal for DevOps/SRE/AI audiences; coherence with IaC portfolio; lowest abandonment risk |
| D2 | CLI name | **rootline** | Free on GitHub, npm, pkg.go.dev, Homebrew. Metaphor: "the line from root to leaves" |
| D3 | Rules file | **`.stem`** | "The stem is where everything grows from" — parent→child inheritance |
| D4 | Rules format | **YAML** | Coherence with frontmatter of indexed documents |
| D5 | Inheritance model | **parent→child** (.htaccess) | Rules flow top-down through hierarchy |
| D6 | Scope | **Standalone from day 1** | Publishable, contributable, dogfooding-ready |
| D7 | Protocol | **MCP (JSON-RPC 2.0)** | Single protocol layer. CLI calls engine directly. MCP for external consumers. |
| D8 | Cache | **No cache initially** | YAGNI — ~200 files in <200ms. Add mtime-based cache if needed. |
| D9 | Links | **Separate feature** | Link extraction is deferred. Graph-based state derivation is a separate future concern. Evaluate need first. |
| D10 | CLI↔Engine | **Direct Go calls** | No JSON-RPC between CLI and engine. MCP is for external consumers only. |
| D19 | Schema model | **Flat field map** | Research doc proposed levels-array (`schema.levels[].name`). Final design uses flat map (`schema.<field>.type/values`). Flat model is simpler, composes better with type-driven merge, and doesn't hardcode hierarchy levels. |
| D20 | Versioning | **Contract version independent** | JSON output schemas carry `"version": 1` independent of any future release version. Breaking output changes require version bump. |
| D11 | GitHub org/user | **github.com/pones** | Personal account. Module path: `github.com/pones/rootline` |

### Open

*No open decisions.*

---

## 6. Investigations Pending

Each investigation is a separate session. Results update this document.

### Motivation Categories

- **Problem-driven**: Addresses a documented problem from §2 (Evidence)
- **Feature-driven**: Enables a differentiator from the vision, no current pain point
- **Futureproofing**: No current problem or urgency, prepares for scale/extensibility

### Problem ↔ Investigation Cross-Reference

| Documented Problem (§2) | Root Issue | Investigation |
|--------------------------|------------|---------------|
| 7 independent parsing systems | No unified extraction pipeline | I7 |
| 4 different regex patterns for `estado:` | No query language, each consumer re-invents filtering | I1 |
| 5 formats for governance rules | Solved by `.stem` design, no investigation needed | — |
| 0 unified schemas | No way to express or query effective schema | I5 |
| Silent failures (empty output, no error) | Untyped queries can't distinguish "no results" from "bad filter" | I1 |
| Tracking drift (E02/F01, state in 2 places) | State manually maintained instead of derived | I3 |
| Format fragility (add type = edit 6 skills) | No centralized schema, validation scattered | I5 |
| Prompt-based validation (non-deterministic) | Solved by `.stem` + `validate` design, no investigation needed | — |

### Investigation Registry

| # | Topic | Question | Approach | Motivation | Solves | Status |
|---|-------|----------|----------|------------|--------|--------|
| I1 | **Query operators** | What operator set to adopt? | Derived from 18 real consumers. Result: 5 operators (`eq`, `ne`, `in`, `contains`, `exists`) + `and` + `count`/`limit`. `--field` for extraction. `order_by` removed (sequence is I5 concern). See [I1 research](../research/I1-query-operators.md). | Problem-driven | 4 regex patterns, silent failures, format fragility | **Done** |
| I2 | **Plugin architecture** | How to implement extensibility? | **Deferred.** Only Markdown needed for dogfooding. Extractor interface exists (I7, D28). Plugin distribution mechanism when non-Markdown extractors are demanded. Options identified: Go interfaces + registry, WASM, expression DSL, external process. | Futureproofing | — | **Deferred** |
| I3 | **Derivation functions** | Built-in set or extensible? | **Deferred.** Reality check (2026-02-18): tracking drift is solvable with hierarchical validation. Derivation adds convenience but isn't blocking. Pipeline slot reserved, Record type accommodates derived fields, .stem `derive:` key is a no-op until engine exists. Expression language (Starlark/Rego/Expr) to be chosen with evidence from real usage. See [I3 pre-research](../research/I3-derivation-pre-research.md). | Problem-driven | Tracking drift (state in 2 places) | **Deferred** |
| I4 | **Explain tracing** | What depth of tracing? | **Deferred.** `validate` errors include `source` field + rule details. `describe` shows full schema cascade with `source`. Together they cover explainability. Deep tracing (`explain` command) adds value when derivation (I3) exists. | Feature-driven | — (differentiator: explainability) | **Deferred** |
| I5 | **Describe contract** | Exact format of effective schema? | Walk-up discovery + top-down merge. Type-driven merge (maps merge, arrays replace, scalars replace, null removes). Validated against 12 dotfile systems + 250 homeserver sessions. See [I5 research](../research/I5-describe-contract.md). | Problem-driven | 0 unified schemas, format fragility | **Done** |
| I6 | **Cache strategy** | When to add cache? | **Deferred.** D8 says YAGNI (<200ms for ~200 files). Benchmark reactively post-implementation. mtime-based cache is standard pattern, no design decision needed upfront. | Futureproofing | — | **Deferred** |
| I7 | **Extractors architecture** | How to abstract extraction pipeline? | `Extractor` interface (3 methods: Extract, Extensions, Name) + `Record` type (Path, Type, Frontmatter map, Body, Errors) + Registry. Only Markdown built-in; all other formats are plugins (D28). Contract serializable for cross-process plugins (D29). See [I7 research](../research/I7-extractors-architecture.md). | Problem-driven | 7 independent parsing systems | **Done** |
| I8 | **MCP tools design** | What MCP tools to expose? | **Deferred.** MCP tools are 1:1 mapping of CLI commands. Input/output schemas already defined by I1 (query contract) and I5 (describe contract). Implement mechanically when CLI is stable. | Feature-driven | — (differentiator: AI-native) | **Deferred** |

### Suggested Execution Order

Based on problem-driven priority and dependency chain:

```
Done:     I1 (Query ops) + I5 (Describe) + I7 (Extractors) → ready to implement
Deferred: I3 (Derivation) + I4 (Explain) + I8 (MCP) + I2 (Plugins) + I6 (Cache)
```

Reality check (2026-02-18): all deferred investigations remain valid but none are blocking.
The three Done investigations provide the complete foundation for initial implementation.

---

## 7. Viability Assessment

Based on reality check (2026-02-17):

| Feature | Viability | Risk | Stack |
|---------|-----------|------|-------|
| .stem loading + parent→child inheritance | High | Low | `gopkg.in/yaml.v3` + `filepath.Walk` |
| Schema validation (types, required, enums) | High | Low | Custom (JSON Schema-lite) |
| Frontmatter parsing | High | Trivial | `gopkg.in/yaml.v3` |
| CLI (query, validate, tree, stats) | High | Low | `cobra` + `viper` |
| `describe` command (effective contract) | High | Low | Subproduct of merge logic |
| MCP server | High | Low | `modelcontextprotocol/go-sdk` |
| JSON output (stable contracts) | High | Trivial | `encoding/json` stdlib |
| Derived fields (built-in set) | Medium | Medium | Deferred — pipeline slot reserved (I3) |
| Derived state (when/then) | Medium | Medium | Deferred — depends on expression language choice (I3) |
| `explain` tracing | Medium | Medium | Instrumented rule engine (I4) |
| Wiki-link extraction | High | Low | `go.abhg.dev/goldmark/wikilink` |
| Links as queryable relationships | Medium-Low | High | Graph query on extracted links |
| Link-based state derivation | Low | High | Graph evaluation — complex |
| Extractors beyond Markdown | Medium | High | Extractor interface (I7) |
| LSP integration | Low | Very High | Entire project scope by itself |

### Stack Summary

| Component | Library/Tool | Rationale |
|-----------|-------------|-----------|
| CLI framework | `cobra` + `viper` | De facto standard (kubectl, gh, hugo) |
| YAML parsing | `gopkg.in/yaml.v3` | Mature, battle-tested |
| Testing | `go test` + `testify` | Built-in + readable assertions |
| MCP server | `modelcontextprotocol/go-sdk` | Official SDK, Go team + Anthropic |
| Wiki-links | `go.abhg.dev/goldmark/wikilink` | Goldmark extension, maintained |
| CI/CD | GitHub Actions + goreleaser | Multi-platform releases |
| Distribution | `go install` + Homebrew tap | Maximum accessibility |

---

## 8. Strategic Context

### The Project IS the Portfolio Piece

Rootline is not a utility buried in a monorepo. Published as open source,
a well-executed Go CLI demonstrates:

| Audience | What they value | How Rootline signals it |
|----------|----------------|------------------------|
| **DevOps / SRE / Platform** | Go, tooling, pragmatism | CLI in Go like kubectl/terraform |
| **Software Engineering** | Clean code, testing, design patterns | Modular architecture, tests, CI/CD |
| **AI/LLM Engineering** | Native AI integration, MCP | MCP server in Go (differentiator vs TS default) |

### Differentiators (vs existing tools)

No existing tool combines all three:

1. **Frontmatter indexing** — MarkdownDB, frontmatter-mcp do this
2. **Per-directory rules with parent→child inheritance** — **nobody does this**
3. **Hierarchical state derivation** — nobody (Obsidian Dataview is closest, no inheritance)

Rootline fills this empty space.

### Dogfooding

The homeserver automation project (`/opt/homeserver/automation/docs/epics/`) is
the first consumer. Its current 7 parsing systems would be replaced by `.stem`
files + `rootline` CLI calls.

---

## 9. References

### Research
- [Original research document](../research/docrules-research-2026.md) — competitive analysis, tech stack decision

### Competitors
- [MarkdownDB (mddb)](https://github.com/datopian/markdowndb) — Node.js, SQLite backend, no per-dir rules
- [markdown-to-sqlite](https://github.com/simonw/markdown-to-sqlite) — Python, Datasette ecosystem
- [frontmatter-mcp (DuckDB)](https://github.com/kzmshx/frontmatter-mcp) — DuckDB, SQL over frontmatter, no inheritance

### MCP Ecosystem
- [Go MCP SDK (official)](https://github.com/modelcontextprotocol/go-sdk) — Go team + Anthropic
- [markdown-frontmatter-mcp](https://github.com/caffeinatedwes/markdown-frontmatter-mcp) — Obsidian-focused
- [Knowledge Base MCP](https://lobehub.com/mcp/cwente25-knowledgebasemcp) — flat structure

### Per-Directory Config Patterns
- [Apache .htaccess](https://httpd.apache.org/docs/current/howto/htaccess.html) — parent→child model
- [EditorConfig spec](https://editorconfig.org/) — child→parent model (contrast)
- [.gitattributes](https://git-scm.com/docs/gitattributes) — parent→child, pattern matching

### Query Language References
- [OData $filter](https://www.odata.org/getting-started/basic-tutorial/) — operator set reference for I1
- [Obsidian Dataview](https://blacksmithgu.github.io/obsidian-dataview/) — query language for frontmatter

### Go Libraries
- [goldmark-wikilink](https://pkg.go.dev/go.abhg.dev/goldmark/wikilink) — wiki-link extraction
- [sourcegraph/jsonrpc2](https://pkg.go.dev/github.com/sourcegraph/jsonrpc2) — JSON-RPC 2.0 for Go
