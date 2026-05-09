---
estado: In Progress
tipo: task
---
# T001: Codify command responsibility contracts

**Outcome**: [O09 Separate command responsibilities and replace legacy apply](README.md)
**Contribuye a**: CE1, CE2, CE3 y CE4 del Outcome.

## Preserva

- INV1: `.stem` mutation is explicit schema evolution, never a hidden side effect of data repair.
  - Verificar: the contract explicitly classifies every mutating command by target surface.
- INV3: Existing read-only commands remain read-only.
  - Verificar: read-only commands are listed and have no write responsibilities.

## Contexto

The investigation found that `rootline apply` currently mixes bootstrap (`missing_schema` scaffold), schema mutation, and data correction. `fix --all` also mixes document repairs with schema mutations such as `extend_enum`, `add_aggregate`, and `remove_stem_field`. Before implementation, Rootline needs an explicit command responsibility contract.

## Alcance

**In**:
1. Define command lanes: discovery/read-only, governance validation, data repair, schema proposal, schema apply/evolution, and direct single-record mutation.
2. Classify current commands: `analyze`, `apply`, `fix`, `set`, `migrate`, `init`, `new`, `validate`, `describe`, `query`, `tree`, `stats`, `graph`, `explain`, MCP tools.
3. Define which commands may mutate Markdown, which may mutate `.stem`, and which must remain read-only.
4. Decide deprecation strategy for generic mixed `apply`.

**Out**:
- Implementing new commands.
- Changing `.stem` merge semantics.

## Estado inicial esperado

- Investigation artifacts exist in prior Pi session outputs for `apply` bugs, command API surface, data-first bootstrap, and `.stem` architecture.
- Existing O07 excludes core command bugfixes, so this task creates the core contract O07 can depend on.

## Criterios de Aceptación

- A checked-in ADR/spec or roadmap design document states the command responsibility contract.
- The spec explicitly says normal data repair must not mutate `.stem`.
- The spec identifies the replacement shape for legacy `apply`: schema proposal/apply and repair apply.
- The spec lists compatibility/deprecation expectations for existing CLI and MCP consumers.
- `rootline validate --all docs/roadmap/` returns exit 0 or only accepted warnings.

## Fuente de verdad

- `cmd/rootline/apply.go`
- `cmd/rootline/fix.go`
- `cmd/rootline/migrate.go`
- `cmd/rootline/set.go`
- `cmd/rootline/analyze.go`
- `internal/infer/report.go`
- `internal/proposal/proposal.go`
- `internal/mcp/tools.go`
- `docs/roadmap/O07-expose-complex-operations-with-guardrails/README.md`

## Contrato de Responsabilidades de Comandos

### Carriles de Responsabilidad

Rootline commands organize into six responsibility lanes:

1. **Discovery & Read-Only** — Execute queries, introspection, and analysis without modifying any files.
2. **Governance & Validation** — Validate documents against schema and report structural gaps.
3. **Data Repair** — Correct frontmatter fields and document values (Markdown only, no `.stem` mutations).
4. **Schema Proposal** — Analyze documents to propose schema changes and return proposals (never auto-apply).
5. **Schema Application** — Apply schema inferences to `.stem` files (explicitly separated from data repair).
6. **Direct Single-Record Mutation** — Set individual fields or sections with validation (Markdown only).

### Clasificación de Comandos

#### Lane 1: Discovery & Read-Only
| Command | Mutates | Purpose |
|---------|---------|---------|
| `query [file...]` | None | Search/filter records by frontmatter fields using expr-lang expressions |
| `describe [path]` | None | Display effective `.stem` schema for a directory (with `--by-domain` filtering) |
| `tree [path]` | None | Show hierarchical tree of records with completion counts and metadata |
| `stats [path]` | None | Show aggregate statistics by `estado`, `tipo`, and other fields |
| `graph [path]` | None | Build dependency graph from wiki-links, detect cycles, analyze broken links |
| `explain [file]` | None | Trace document state: field origins, derivation chain, validation errors |
| `validate [files...]` | None | Check documents against `.stem` rules; `--all` validates directory tree |
| MCP: `query` | None | MCP tool wrapper for `query` CLI command |
| MCP: `validate` | None | MCP tool wrapper for `validate` CLI command |
| MCP: `describe` | None | MCP tool wrapper for `describe` CLI command |
| MCP: `tree` | None | MCP tool wrapper for `tree` CLI command |
| MCP: `stats` | None | MCP tool wrapper for `stats` CLI command |
| MCP: `explain` | None | MCP tool wrapper for `explain` CLI command |
| MCP: `graph` | None | MCP tool wrapper for `graph` CLI command |
| MCP: `trace` | None | Follow reference chains through the document graph via BFS traversal |
| MCP: `health` | None | Return server health status: version, uptime, goroutines |

#### Lane 2: Governance & Validation
| Command | Mutates | Purpose |
|---------|---------|---------|
| `analyze [directory]` | None | Run 16 inference detectors (13 data + 3 governance) and return structured report |
| MCP: `fix` | None | Analyze validation errors and return fix proposals (always dry-run, never modifies) |

#### Lane 3: Data Repair
| Command | Mutates | Purpose |
|---------|---------|---------|
| `fix [file...]` | Markdown only | Auto-repair validation errors: add missing required fields, correct enum typos |
| `fix --all [path]` | Markdown only | Bulk fix all files; also proposes aggregate field propagation & stem health issues |
| `migrate --rename old=new [path]` | Markdown + `.stem` | Bulk rename field across all documents and `.stem` schemas |
| `migrate --scaffold [path]` | Markdown only | Add missing required section headings to documents |

**Critical requirement for Lane 3**: Normal data repair operations (`fix`, `fix --all` single-file mode, `migrate --scaffold`) **MUST NOT mutate `.stem` files**. The only exception is `migrate --rename`, which updates field references in both documents and `.stem` schemas as a data migration operation.

#### Lane 4: Schema Proposal
| Command | Mutates | Purpose |
|---------|---------|---------|
| `apply [report.json]` | Both | **LEGACY, MIXED: currently handles both schema & data**. Pre-phase scaffolds `.stem` for `missing_schema`; separates inferences and applies schema-modifying & data-correcting proposals together. Planned for deprecation. |

#### Lane 5: Schema Application
| Command | Mutates | Purpose |
|---------|---------|---------|
| `migrate --split [path]` | `.stem` only | Split flat `.stem` into hierarchical per-level files; auto-generates aggregates where needed |
| `migrate --diff [path]` | None | Compare current `.stem` against git HEAD or previous version; report changes (schema evolution tracking, not mutation) |

#### Lane 6: Direct Single-Record Mutation
| Command | Mutates | Purpose |
|---------|---------|---------|
| `set <file> <field=value>...` | Markdown only | Set individual frontmatter fields or body section content; validates post-mutation |
| `new <filepath>` | Markdown only | Create new document with frontmatter scaffolded from effective `.stem` schema |
| `init [path]` | `.stem` only | Generate or update `.stem` from existing documents (inference-based bootstrap) |
| `init --template owner/repo[@tag]` | `.stem` only | Fetch and import `.stem` from remote GitHub repo |
| MCP: `new` | Markdown only | MCP tool wrapper for `new` CLI command |
| MCP: `set` | Markdown only | MCP tool wrapper for `set` CLI command |

### Key Design Rules

**Rule 1: Separation of Concerns**
- Data repair commands fix frontmatter/body but **never touch `.stem` schemas**.
- Schema evolution commands modify `.stem` files but are intentional (not side effects).
- Exception: `migrate --rename` (bulk field rename) modifies both documents and `.stem` as a coordinated data migration.

**Rule 2: Markdown vs `.stem` Boundaries**
| Target | Read-only | Data Repair | Schema Application | Mutation |
|--------|-----------|-------------|-------------------|----------|
| Markdown | query, validate, describe, tree, graph, explain, analyze, fix (dry-run) | fix, fix --all, set, migrate --scaffold | — | fix, fix --all, set, new, migrate --rename |
| `.stem` | validate, analyze, migrate --diff | — | init, migrate --split | init, init --template, migrate --split, apply (legacy), apply (schema phase) |

**Rule 3: Proposal vs Application**
- `analyze` returns inferences; `apply` (currently) consumes them.
- `fix` (MCP, always dry-run) and `fix --all` (with `--dry-run`) generate proposals via the proposal engine (`internal/proposal/Analyze`).
- There is no standalone "proposal generation" command; proposals are generated as a phase of `analyze` → `apply` or `fix` → `fix`.

**Rule 4: Bulk Operations Require Explicit Intent**
- `fix --all` requires explicit `--all` flag (no implicit "repair everything").
- `apply` requires explicit report input (from `analyze` output or stdin).
- `migrate --rename`, `migrate --split` require explicit flags.

**Rule 5: All Mutations Support `--dry-run`**
- All mutating commands in Lanes 3 and 6 support `--dry-run` to preview changes without writing.
- `fix` (MCP) is always dry-run in the MCP context (never modifies).
- This aligns with O07 guardrails: users see proposals before accepting bulk changes.

### Replacement Shape for Legacy `apply`

The legacy `apply` command currently mixes responsibilities: it scaffolds `.stem` for `missing_schema`, applies schema-modifying inferences, and applies data corrections in one pass. This violates the separation-of-concerns principle and makes its behavior hard to reason about.

**Replacement (planned)**:

1. **Schema Proposal & Application Surface**: `schema propose` / `schema apply` (or rename `apply` to `schema apply`)
   - Input: `analyze` report
   - Output: Apply only schema-modifying inferences (`extend_enum`, `add_aggregate`, etc.) to `.stem` files
   - Scaffolding: Pre-phase to create `.stem` files for `missing_schema` before applying inferences
   - Guardrail: Require `--confirm` or interactive approval for breaking changes

2. **Data Repair Application Surface**: `repair apply` (separate from schema)
   - Input: Proposals with type `migrate_value`, `correct_value`, `add_field`, etc.
   - Output: Apply data corrections to Markdown files only (never `.stem`)
   - Guardrail: Support `--dry-run`; validate post-mutation

**Compatibility & Deprecation**:
- Legacy `apply` remains available for backward compatibility during v0.x
- Mark as `@deprecated` in help text and documentation
- New code should use `schema apply` for schema changes and `repair apply` (or extended `fix --all`) for data corrections
- MCP `fix` tool (always proposals, never mutations) becomes the canonical "proposal generation" interface
- Deprecation timeline: Plan removal in v1.0 or next major release

### Invariants Preserved

- **INV1** — `.stem` mutation is explicit schema evolution, never a hidden side effect of data repair.
  - Verified: Lanes 3 and 6 (data repair & direct mutation) explicitly exclude `.stem` modification.
  - Only Lane 5 (`migrate --split`, `apply` schema phase) and Lane 1 initialization (`init`) modify `.stem`.

- **INV3** — Existing read-only commands remain read-only.
  - Verified: All Lane 1 commands are read-only by definition.

### MCP Consistency

All MCP tools implement the same responsibility lanes as their CLI counterparts:
- MCP tools in Lane 1 are read-only wrappers
- MCP `fix` (Lane 4) is always proposals + dry-run, never modifies files
- MCP `set` (Lane 6) mutates Markdown with validation
- MCP does not expose schema mutation commands (Lane 5); schema changes remain CLI-only for now
