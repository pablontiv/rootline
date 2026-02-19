# I3 Pre-Research: Derivation Functions

**Date**: 2026-02-18
**Status**: Pre-research (investigation deferred)
**Method**: Web research on expression languages + analysis of rootline's derivation needs

---

## 1. Why Deferred

Reality check: the primary motivator for I3 (tracking drift, E02/F01) is solvable with
**hierarchical validation** without derivation. Derivation adds convenience
(auto-computed `progress`, `slug`) but isn't blocking.

Deferring lets us choose the expression language with evidence from real usage patterns
instead of designing in abstract.

**Architecture is ready**: pipeline slot reserved (`Extraction > ... > [Derivation] > Query`),
Record type accommodates derived fields (`map[string]any`), .stem `derive:` key parseable
but no-op, query contract has `derived.*` namespace, describe contract has `"derive": {}` placeholder.

---

## 2. Derivation Levels Identified

| Level | Example | Evaluator needs |
|-------|---------|-----------------|
| 1. Field transform | `slug = slugify(title)` | Access to one record's fields |
| 2. Cross-record aggregation | `feature.status = f(children)` | Access to child records as collection |
| 3. Link traversal | `blocked = any(links.blocks, .status != "done")` | Navigate link graph |
| 4. Transitive/recursive | `health = all(descendants, .has_owner)` | Recursive traversal |

Initial derivation should cover levels 1-2. Levels 3-4 depend on link features (D9).

---

## 3. Expression Languages Evaluated

### Expr (`expr-lang/expr`)

- **Type**: Non-Turing complete expression evaluator
- **Go package**: `github.com/expr-lang/expr`
- **Syntax**: Go/JS-like (`lower(replace(title, " ", "-"))`)
- **Safety**: Memory-safe, side-effect-free, always terminates
- **Types**: Compile-time type checking
- **Built-ins**: `all`, `none`, `any`, `filter`, `map`, `len`, `lower`, `upper`, `trim`
- **Performance**: Bytecode VM with optimizing compiler
- **Size**: ~3MB, zero transitive deps
- **Users**: Google Cloud, Uber, Argo Workflows, ByteDance
- **Levels**: 1-2 (collections supported via `filter`/`map` if engine passes children as array)
- **Source**: https://expr-lang.org/

### CEL (`google/cel-go`)

- **Type**: Non-Turing complete expression language
- **Go package**: `cel.dev/expr`
- **Syntax**: C-like (`request.auth.claims.sub`)
- **Safety**: Non-Turing complete, always terminates
- **Users**: Kubernetes CRD validation, Envoy, Tekton
- **Size**: Heavy (protobuf dependency)
- **Levels**: 1-2 (cross-field validation strength, less natural for transforms)
- **Source**: https://cel.dev/

### Starlark (`google/starlark-go`)

- **Type**: Python dialect, Turing-complete (bounded)
- **Go package**: `go.starlark.net/starlark`
- **Syntax**: Python (`[c for c in children if c.status == "done"]`)
- **Safety**: No I/O, no unsafe imports, sandboxed. `Thread.SetMaxExecutionSteps()` for termination
- **Users**: Bazel, Buck2, Tilt, Drone CI
- **Levels**: 1-4 (full language, can express anything with custom builtins)
- **Source**: https://github.com/google/starlark-go

### Rego (OPA)

- **Type**: Datalog-inspired policy language, non-Turing complete
- **Go package**: `github.com/open-policy-agent/opa/v1/rego`
- **Syntax**: Declarative/Datalog (`status := "done" { count(undone) == 0 }`)
- **Safety**: Non-Turing complete, always terminates
- **Built-ins**: 150+ (count, sum, min, max, regex, glob, graph, etc.)
- **Users**: Kubernetes, Terraform Sentinel, Envoy, Conftest
- **Size**: Heavy (~15MB OPA dependency)
- **Levels**: 1-4 (Datalog = natural recursion over graphs)
- **Source**: https://www.openpolicyagent.org/

### Also Reviewed (less relevant)

- **CUE** (`cuelang.org/go/cue`): Lattice-based unification. Philosophically aligned (schema=derivation) but excessive complexity for rootline's scope.
- **Nickel** (`nickel-lang`): Contracts + merge with idempotency. Interesting concept (idempotent derivation: `slugify(slugify(x)) == slugify(x)`) but Rust-based, no Go embedding.
- **Contentlayer**: `computedFields` pattern with `resolve: (doc) => value` signature. Clean but project abandoned (2024). Pattern adoptable.
- **Dataview** (Obsidian): Expression engine with `Result` type error handling. Cross-page computed fields NOT supported (known limitation). Error handling pattern adoptable.
- **MarkdownDB**: `computedFields: [(fileInfo, ast) => {...}]`. No schema declaration. Counter-example.

---

## 4. Two Architectures Identified

### Architecture A: Single Powerful Language

```
Starlark / Rego handles everything:
  field transforms + aggregation + graph traversal
```

- One language, one file (`.derive.star` or `.derive.rego`)
- User writes all derivation logic
- Engine passes context (record, children, links)
- Inheritance: walk-up like `.stem`, child can override parent

**Starlark variant**: imperative (Python-like). Familiar syntax, high adoption potential.
**Rego variant**: declarative (Datalog). Coherent with `.stem` declarative nature. Natural graph recursion.

### Architecture B: Engine + Expression

```
Rootline engine (Go) controls traversal.
Expression language (Expr) evaluates leaf computations.
```

- Engine handles: which records to visit, in what order, what context to pass
- Expressions handle: individual value computations
- `.stem` declares derivation declaratively, expressions inline in YAML

**Pro**: rootline controls the walk (coherent with per-directory inheritance as differentiator).
**Con**: two concepts (engine keywords + expression syntax).

---

## 5. Where Files Would Live

Expressions inline in `.stem` (Architecture B):
```yaml
derive:
  slug:
    expr: "lower(replace(title, ' ', '-'))"
```

Separate file alongside `.stem` (Architecture A):
```
docs/epics/E01-infra/
  .stem              # schema + validation
  .derive.star       # derivation logic (Starlark)
```

Inheritance applies to both: walk-up discovery, child overrides parent.

---

## 6. Key Insight

The expression language choice benefits from real usage patterns.
Once rootline ships with validation + query, we'll know:
- How complex real derivation needs are (level 1 only? or 2-4?)
- Whether inline expressions suffice or separate files are needed
- What the actual adoption barrier is (syntax familiarity matters)

This evidence will make I3 a much better investigation.

---

## 7. References

- Expr: https://expr-lang.org/, https://github.com/expr-lang/expr
- CEL: https://cel.dev/, https://codelabs.developers.google.com/codelabs/cel-go
- Starlark: https://github.com/google/starlark-go
- OPA/Rego: https://www.openpolicyagent.org/docs, https://www.openpolicyagent.org/docs/policy-language
- CUE: https://cuelang.org/docs/introduction/
- Nickel: https://nickel-lang.org/user-manual/contracts/, https://github.com/tweag/nickel/blob/master/RATIONALE.md
- Dataview: https://deepwiki.com/blacksmithgu/obsidian-dataview
- WunderGraph Expr analysis: https://wundergraph.com/blog/expr-lang-go-centric-expression-language
