# Spec: Stem Domain Types

## Context

Rootline is a file-based database where `.stem` files define schemas (DDL) for markdown documents. Today, field identification is purely by name — tools that consume rootline data (like Kedral via MCP, or AI agents) must know that a lifecycle field is called `estado` in one repo and `status` in another. There is no semantic layer to bridge this gap.

The `stats` command and MCP tool hardcode lookups for `estado` and `tipo`. This breaks for any project using different field names.

Industry precedent (SQL DOMAINs, JSON Schema `format`, GraphQL custom scalars, Protobuf well-known types) shows that a semantic type layer over base types is the standard solution. Empirical research on MCP tool descriptions (Feb 2026) confirms that AI agents perform better with well-known identifiers than freeform metadata for common concepts.

## Objective

Add `domain` as an optional property on `.stem` schema fields — a semantic type that declares what a field *means* (lifecycle state, record type, identifier) independently of what it's *named*. Domains imply a base type, enable virtual alias resolution in expressions, and provide a stable contract for consumer tools.

## Design

### 1. Domain Property

Each field in a `.stem` schema can declare an optional `domain`:

```yaml
version: 2
schema:
  estado:
    domain: lifecycle_state
    values: [draft, active, closed]
    # type inferred as "enum" from lifecycle_state's base type

  id:
    domain: identifier
    prefix: "T"
    digits: 3
    # type inferred as "sequence"

  titulo:
    domain: title
    # type inferred as "string"

  mi_campo_custom:
    domain: acme/sprint_velocity
    type: integer
    # custom domain — type must be declared explicitly
```

### 2. Core Domain Registry

Rootline defines a set of well-known domains with implied base types:

| Domain | Base Type | Required Attrs | Description |
|--------|-----------|----------------|-------------|
| `lifecycle_state` | enum | `values` | Lifecycle stage of a record |
| `record_type` | enum | `values` | Type discriminator / classifier |
| `identifier` | sequence | `prefix`, `digits` | Unique record ID |
| `title` | string | — | Human-readable name |
| `created_date` | string | — | Creation date |
| `due_date` | string | — | Deadline |
| `owner` | string | — | Responsible person/team |
| `parent_ref` | string | — | Reference to parent record |
| `priority` | enum | `values` | Priority level |
| `description` | string | — | Extended description |
| `confidence` | enum | `values` | Confidence / maturity level |
| `source` | string | — | Origin of information |

**Extension convention**: Strings without `/` are core domains (validated by rootline). Strings with `/` are custom domains (e.g., `acme/sprint_velocity`) — rootline transports them without validation.

### 3. Type Resolution Rules

1. **Core domain, no explicit type** → `type` = domain's base type (inferred)
2. **Core domain + explicit type** → explicit type wins; rootline validates compatibility (warn if incompatible)
3. **Custom domain (contains `/`), no explicit type** → `error` (rootline can't infer type for unknown domains)
4. **No domain** → current behavior, unchanged

### 4. Virtual Alias Resolution in Expressions

Domains are resolved as virtual field names in any expr-lang expression. This works across all commands that accept `--where`:

```bash
# These are equivalent when 'estado' has domain: lifecycle_state
rootline query --where 'lifecycle_state == "active"' docs/epics/
rootline query --where 'estado == "active"' docs/epics/
```

**Injection mechanism:** Domain aliases are injected at **runtime per-record**, not at compile time. `BuildEnv` in `expr_eval.go` receives the record's effective schema (passed as a new parameter). For each field with a domain, it adds `domain_name → field_value` to the env map alongside the existing `field_name → field_value` entries. This is runtime injection because different records may resolve the same domain to different fields (due to `match:` scoping). The compiled expression uses `AllowUndefinedVariables()`, so if a domain has no match for a given record, it resolves to nil (falsy) rather than erroring.

Resolution is scope-aware: rootline evaluates the record's effective schema (after `match:` filtering) to resolve the domain to the correct field name.

```yaml
schema:
  estado_epic:
    domain: lifecycle_state
    match: "E*"
    type: enum
    values: [draft, active, closed]

  estado_feature:
    domain: lifecycle_state
    match: "F*"
    type: enum
    values: [specified, in_progress, done]
```

For a record in `E01-infra/`, `lifecycle_state` resolves to `estado_epic`.
For a record in `F01-auth/`, `lifecycle_state` resolves to `estado_feature`.

### 5. Domain Uniqueness

Domain must be unique **per effective scope** (after match filtering):

- Two fields with the same domain and **mutually exclusive match patterns** → allowed
- Two fields with the same domain and **overlapping match patterns** (or both global) → stem-health error
- Uniqueness is validated post-merge (on the effective schema, not individual `.stem` files)

### 6. Merge Semantics

`domain` is a scalar → **replace** on merge (child overrides parent), consistent with existing string merge behavior in rootline.

### 7. Describe Extension

`rootline describe` output includes domain:

```
$ rootline describe docs/epics/
  estado     enum (lifecycle_state)  required  [draft, active, closed]  Source: .stem
  tipo       enum (record_type)      required  [epic, feature]          Source: .stem
  id         sequence (identifier)              prefix=E, digits=2      Source: .stem
```

New `--by-domain` flag filters output:

```
$ rootline describe docs/epics/ --by-domain lifecycle_state
estado    enum    required    [draft, active, closed]
```

JSON output includes `"domain"` key on each field.

### 8. Hardcoded Field Decoupling

Replace all hardcoded `"estado"` and `"tipo"` lookups with domain-based resolution:

**Affected sites:**
- `cmd/rootline/stats.go` — `by_estado` / `by_tipo` aggregation
- `internal/mcp/tools.go` — `handleStats()` uses `EffectiveField("estado")` and `EffectiveField("tipo")`
- `internal/mcp/tools.go` — `buildTree()` / `treeNode.Estado` uses `EffectiveField("estado")` and compares against `"Completed"`
- `cmd/rootline/tree.go` — same pattern as MCP tree

**Resolution strategy:**
- Look for field with `domain: lifecycle_state` instead of field named `estado`
- Look for field with `domain: record_type` instead of field named `tipo`
- Fallback to current hardcoded names if no domain is declared (backward compatibility)

**JSON output contract:** The `StatsResult` JSON keys `by_estado` and `by_tipo` remain unchanged for backward compatibility. Internally, the values are populated by domain lookup, but the JSON key names stay the same. A future version may add a parallel `by_domain` structure, but this spec does not change the JSON contract.

**Out of scope:**
- `cmd/rootline/analyze.go` hardcodes `"tipo"` in `DetectSubSchemas`. This is inference-layer code that operates on raw data, not schema-resolved data. It will be addressed separately when inference gains domain awareness.
- `internal/derive/derive.go` has its own `BuildEnv` for derivation expressions. Domain aliases in derive expressions are deferred — derive operates on concrete field names and adding alias support there is a separate concern.

### 9. Stem-Health Validation

New checks added to the stem-health diagnostic phase:

| Check | Severity | Condition |
|-------|----------|-----------|
| `domain-type-compat` | warn | Core domain assigned to incompatible base type |
| `domain-missing-attrs` | warn | Core domain missing required attributes (e.g., `lifecycle_state` without `values`) |
| `domain-duplicate-scope` | error | Two fields share domain with overlapping match patterns |
| `domain-custom-no-type` | error | Custom domain (with `/`) has no explicit type |

### 10. Version Compatibility

- **Stem version**: Remains v2. `domain` is an optional additive extension.
- **Backward compatibility**: Stems without `domain` work identically. Older rootline versions that don't understand `domain` will ignore the unknown YAML key during custom unmarshal (verified: `gopkg.in/yaml.v3` behavior).
- **Forward compatibility**: New stems with `domain` degrade gracefully on older rootline — fields work normally, domain metadata is simply absent.

## Files Modified

| File | Change |
|------|--------|
| `internal/rules/rules.go` | Add `Domain string` to `SchemaField`, update `UnmarshalYAML` |
| `internal/rules/domains.go` (new) | Core domain registry with `DomainDef` and `Resolve()` |
| `internal/rules/validate.go` | Stem-health checks for domain coherence |
| `internal/query/expr_eval.go` | Inject domain aliases into expr-lang environment |
| `cmd/rootline/describe.go` | Display domain in output, `--by-domain` flag |
| `cmd/rootline/stats.go` | Replace hardcoded field names with domain lookup |
| `internal/mcp/tools.go` | Extend describe tool output, domain filter; update stats and tree tools |
| `cmd/rootline/tree.go` | Replace hardcoded `estado` lookup with domain resolution |

## Verification

1. **Unit tests**: Domain registry resolution, type inference, uniqueness validation
2. **Integration tests**: `.stem` with domains parsed correctly, merge behavior, describe output
3. **E2E tests**: Query with domain alias, stats with domain-based lookup, MCP describe with domain filter
4. **Backward compat**: Existing stems without domain pass all existing tests unchanged
5. **Manual**: `rootline validate --all docs/epics/` passes after adding domains to existing stems
