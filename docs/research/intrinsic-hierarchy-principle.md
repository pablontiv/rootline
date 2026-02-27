---
estado: Pre-research
fecha: "2026-02-27"
metodo: collaborative-research
---
# Intrinsic Hierarchy Principle

**Context**: Discovered during investigation of parent-child data inconsistencies in a rootline-managed project. Epic README files had `estado: In Progress` while all their child Features were `Completado`. Rootline didn't detect the inconsistency because `aggregate:` wasn't configured — vertical consistency was opt-in, not automatic.

This led to questioning why hierarchy needs to be declared at all when the filesystem already defines it.

---

## Part 1 — The Principle

### Core Insight

In a filesystem-based database, the directory tree IS the relational schema. Parent-child relationships are intrinsic to the medium — not declared, not optional.

### Three Axioms

1. **A file inside a directory is a child of that directory.** Always.
2. **Subdirectories at the same level are siblings.** Always.
3. **These relationships exist before any schema is written.**

These are properties of the medium (filesystem), not features of the engine. They cannot be turned off. A file at `E01/features/F01/README.md` is a descendant of `E01/` by virtue of existing there — no declaration needed.

### The RDBMS Contrast

In an RDBMS, tables are flat. Relationships must be explicitly declared:

| Aspect | RDBMS | Filesystem DB |
|--------|-------|---------------|
| Hierarchy declared by | Schema (`FOREIGN KEY`) | Physical structure (directories) |
| Relationship exists before schema? | No — must be declared | Yes — intrinsic to medium |
| Consistency enforcement | Opt-in (add constraints) | Should be automatic |
| Schema redeclares hierarchy? | Must (FK required) | Should not need to |
| Equivalent of `FOREIGN KEY` | Necessary | Redundant — the structure IS the relationship |

### The Error in Current Rootline

Rootline's `levels:` section in `.stem` files declares hierarchy that already exists:

```yaml
levels:
  epic:
    match: "E*"
    children: [feature]   # ← The filesystem already says this
  feature:
    match: "F*"
    children: [story]     # ← Redundant with directory structure
```

This is the equivalent of declaring `FOREIGN KEY` on a relationship that the medium already guarantees. The `children: [feature]` declaration doesn't create the relationship between `E01/` and `E01/features/F01/` — the directory tree already did.

Similarly, `aggregate:` expressions are required to enable consistency checking between parent and child. But the need for consistency isn't optional — if a field exists at both levels, their values should be consistent by default. `aggregate:` should be the mechanism to *compute* the parent value, not to *enable detection* of inconsistency.

### The Principle

> **In a hierarchical database over a filesystem, the tree topology IS the relational schema. Vertical consistency between parent and children is not an optional feature — it's a property the engine must guarantee by default, because the medium implies it.**

---

## Part 2 — State of the Art

### Closest Systems

No existing system fully implements the intrinsic hierarchy principle. Each has pieces:

| System | Era | What it does well | What it misses |
|--------|-----|-------------------|----------------|
| **IMS** (IBM) | 1966 | Automatic referential integrity in hierarchical model; single-parent rule guarantees uniform data | Proprietary, not filesystem-based; hierarchy declared in DBD macros |
| **OLAP / MDX** | 1990s | Automatic bottom-up rollup through dimension hierarchies; parent values always derived from children | Cube-based, not filesystem; requires explicit dimension/hierarchy declaration |
| **Hugo cascade** | 2020 | `cascade` in `_index.md` propagates values down directory tree | No validation whatsoever; propagates values, not constraints |
| **Dendron schemas** | 2020 | YAML schemas match hierarchy patterns (e.g., `daily.journal.*.md`); template auto-application | Visual-only enforcement (shows "?" indicator); no strict mode, no consistency checks |
| **Statamic cascade** | 2012 | Data inheritance by directory depth; variables set higher cascade down and can be overridden | Cascade is for values, not schema constraints; blueprints are per-collection, not per-depth |
| **Astro Content Collections** | 2022 | Zod schemas validate frontmatter per collection; type-safe at build time | Flat per-collection, no hierarchy, no parent-child consistency |
| **CUE language** | 2019 | Types and data are the same thing; constraints only narrow (never override) | Not filesystem-based; no walk-up discovery |
| **Apache .htaccess** | 1995 | Walk-up discovery + per-directory merge; closer files override farther ones | Configuration, not data validation; no schema concept |
| **EditorConfig** | 2012 | Walk-up discovery with `root=true` sentinel; cross-editor settings | Same — configuration only, no data validation |
| **GNU Recutils** | 2010 | Plain-text database with field typing, auto-increment, constraints | Flat within a single file; no hierarchy, no directory structure |

### Key Academic Work

**"Reasoning About Integrity Constraints for Tree-Structured Data"** — Czerwinski, David, Murlak, Parys (ICDT 2016, Theory of Computing Systems 2017).

Studies integrity constraints for data trees where nodes have labels and store data values. Proves that validity and containment of constraint queries over tree structures is decidable (doubly exponential, with matching lower bounds). This formalizes the exact problem: given a tree structure (filesystem), what consistency constraints can be automatically checked?

The paper remains theoretical — no filesystem implementation exists.

**"Schema Evolution for XML: A Consistency-Preserving Approach"** (MFCS 2004). Proposes the GREC algorithm: given a new document that may be invalid, build candidate schemas that preserve consistency. Directly analogous to rootline's `infer` package.

### The Gap

The literature and tooling have all the pieces but nobody combined them:

- IMS proved automatic hierarchical consistency works (1966)
- OLAP proved bottom-up rollup is the correct aggregation pattern
- `.htaccess`/EditorConfig proved walk-up + merge works on filesystem
- Czerwinski et al. proved tree constraints are decidable
- Hugo/Dendron/Statamic proved there's demand for filesystem-aware schema tools

**No system combines: filesystem as medium + implicit hierarchy + walk-up schema inheritance + automatic vertical consistency.** This is the space rootline occupies.

---

## Part 3 — The Model for Rootline

### Five Rules

1. **Every subdirectory is a structural child** — by default. The filesystem defines this, not the schema.

2. **`.stemignore` excludes** what doesn't participate (`docs/`, `assets/`, etc.). If it's not indexed, it doesn't exist for rootline.

3. **One `.stem` at the root** defines the schema for the entire domain. No need for one `.stem` per child — that would be redundant duplication.

4. **A `.stem` in a subdirectory inherits from its parent and can extend or override.** It's only an independent domain if it redefines the schema completely. By default, it's a specialization of the parent — same walk-up + top-down merge that rootline already does.

5. **Vertical consistency is automatic** for any field that exists at both parent and child levels. No `aggregate:` required to detect drift — `aggregate:` is for computing the correct value, not for enabling detection.

### Domain Boundaries

`.stem` files define schema scoping through inheritance:

```
project/
  .stem              ← root schema: defines the domain
  .stemignore        ← docs/, assets/ excluded from everything
  epics/
    E01/
      README.md      ← inherits root .stem, consistency checked
      features/
        F01/
          README.md  ← inherits root .stem (child of E01, consistency checked)
      docs/          ← excluded (.stemignore)
    E02/
      README.md
  other-thing/
    .stem            ← inherits root + extends/overrides
    data.md
```

Within a schema scope:
- **Indexed** = structurally participates in hierarchy
- **Not indexed** (`.stemignore`) = doesn't exist for rootline
- **Sub-`.stem`** = specializes parent schema (merge), independent only if it fully redefines

### `levels:` Becomes Unnecessary

What `levels:` encodes today and what replaces each piece:

| `levels:` feature | Why it exists | Replacement |
|---|---|---|
| `match: "E*"` (naming pattern) | Identify which directories are "epics" | Optional: `schema.field.match` for per-pattern field variants |
| `children: [feature]` (hierarchy chain) | Declare parent→child relationships | **Eliminated**: filesystem IS the hierarchy |
| Per-level `schema:` fields | Different fields at different depths | `schema.field.match` for fields that vary by directory pattern |
| Per-level `validate:` rules | Different rules at different depths | Conditional `validate:` with `if: {match: "E*"}` |
| Sequence prefix/digits | E01, F02, S003, T004 auto-numbering | Inferred from directory names, or declared in `schema.id.match` |
| Nesting enforcement | "Task can't be direct child of Epic" | Automatic: everything indexed is a valid child |

### Automatic Vertical Consistency

For any field defined in schema **without** a `match` restriction:
- It applies to ALL levels in the domain
- The engine automatically checks parent-child consistency:
  - If **all direct children** of a parent have the same value X, and parent has value Y ≠ X → **warning**
- `aggregate:` is the mechanism to **compute** the correct parent value
- Detection of drift is **free** — it comes from the structure, not from configuration

For fields **with** a `match` restriction:
- They only apply to matching directories
- No cross-level consistency check (the field doesn't exist at all levels)

### .stem Format: Before and After

**Current (with `levels:`):**
```yaml
version: 1

schema:
  estado:
    type: enum
    values: [Pending, In Progress, Completado]

levels:
  epic:
    match: "E*"
    children: [feature]
    schema:
      id:
        type: sequence
        prefix: E
        digits: 2
  feature:
    match: "F*"
    children: [story]
    schema:
      id:
        type: sequence
        prefix: F
        digits: 2
      tipo:
        type: enum
        values: [servicio-docker, modulo-sistema, documentation]
  story:
    match: "S*"
    children: [task]
    schema:
      id:
        type: sequence
        prefix: S
        digits: 3
  task:
    match: "T*"
    children: []
    schema:
      id:
        type: sequence
        prefix: T
        digits: 3
      tipo:
        type: enum
        required: true
        values: [servicio-docker, modulo-sistema, documentation]
```

**Proposed (no `levels:`):**
```yaml
version: 2

schema:
  estado:
    type: enum
    values: [Pending, In Progress, Completado]
    # No match → applies everywhere → vertical consistency automatic

  id:
    type: sequence
    match:
      "E*": {prefix: E, digits: 2}
      "F*": {prefix: F, digits: 2}
      "S*": {prefix: S, digits: 3}
      "T*": {prefix: T, digits: 3}

  tipo:
    type: enum
    values: [servicio-docker, modulo-sistema, documentation]
    match: ["F*", "T*"]
    required:
      match: ["T*"]    # required only at task level
```

Key differences:
- **34 lines** instead of 47 — less repetition
- No hierarchy declaration — filesystem provides it
- `estado` without `match` = automatic vertical consistency
- `tipo` with `match` = only applies where specified
- `required` can be scoped with `match` (instead of duplicating the field across levels)

---

## Part 4 — Open Questions

### Q1: Sequence inference vs declaration
Should `id` prefix/digits be fully inferred from existing directory names (zero config) or always require declaration? Inference is more aligned with the principle (the filesystem tells you) but may be fragile for new projects with no directories yet.

### Q2: Match pattern syntax
Should `match` use glob patterns only (like current `levels.match`), or also support depth-based selectors (`depth: 2`) or path patterns (`epics/*/features/*`)? Globs are simpler but less expressive.

### Q3: Migration path
Three options for transitioning from `levels:` to the new model:
- **a)** `levels:` deprecated but supported indefinitely (backwards compat)
- **b)** `levels:` supported in v0.x, removed in v1.0 (breaking change)
- **c)** `rootline migrate` auto-converts `levels:` to `match`-based schema

### Q4: Partial exclusion
Current model: `.stemignore` = doesn't exist at all. Is there a need for "index this file (validate its frontmatter) but don't include it in parent-child consistency"? Current assessment: no — if it's indexed, it participates. If it shouldn't participate, exclude it.

---

## References

- IBM IMS Hierarchical Database Model — [Wikipedia](https://en.wikipedia.org/wiki/Hierarchical_database_model), [O'Reilly](https://www.oreilly.com/library/view/an-introduction-to/9780132886987/ch07.html)
- Czerwinski, David, Murlak, Parys — "Reasoning About Integrity Constraints for Tree-Structured Data" (ICDT 2016) — [Dagstuhl](https://drops.dagstuhl.de/entities/document/10.4230/LIPIcs.ICDT.2016.20), [Springer](https://link.springer.com/article/10.1007/s00224-017-9771-z)
- Schema Evolution for XML — [ResearchGate](https://www.researchgate.net/publication/220976005)
- OLAP Hierarchical Aggregation — [Medium](https://medium.com/learning-sql/olap-hierarchical-aggregation-with-sql-6c45ebc206d7), [SIGMOD Record](https://sigmodrecord.org/publications/sigmodRecord/0003/pourabbas.pdf)
- Hugo cascade — [Hugo docs](https://gohugo.io/content-management/front-matter/)
- Dendron schemas — [Dendron wiki](https://wiki.dendron.so/notes/c5e5adde-5459-409b-b34d-a0d75cbb1052/)
- Statamic data inheritance — [Statamic docs](https://statamic.dev/data-inheritance)
- Astro Content Collections — [Astro docs](https://docs.astro.build/en/guides/content-collections/)
- CUE language — [CUE discussion](https://github.com/cue-lang/cue/discussions/669)
- Apache .htaccess — [Apache docs](https://httpd.apache.org/docs/current/howto/htaccess.html)
- JSON Schema inheritance — [json-schema.org](https://json-schema.org/blog/posts/modelling-inheritance)
- GNU Recutils — [gnu.org](https://www.gnu.org/software/recutils/)
