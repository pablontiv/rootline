---
estado: Implemented
fecha: "2026-02-27"
metodo: collaborative-research
---
# Intrinsic Hierarchy Principle

## Implementation Status

Parts 1-4 implemented in engine v2 (stem v0/v1 rejected, `levels:` removed, `match:` field scoping, automatic drift detection, configurable index file). Part 5 (v3 entity model) remains open design — see Q1-Q5 in Part 7.

| Proposal | Status | Engine location |
|----------|--------|-----------------|
| Remove `levels:` | Done | `rules.go:255-258` rejects v0/v1 |
| `match:` field scoping | Done | `match.go` — 3 YAML forms |
| Automatic drift detection | Done | `drift.go` — no `aggregate:` required |
| Index file semantics | Done | `validate.go:IsIndexFile()` + `structural.subdirs.require_index` |
| v3 `entities:` model | Open | Not implemented — circular tension unresolved (Part 5) |
| Children type constraints | Open | Not implemented |
| Entity-scoped validation | Open | Not implemented |

---

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
  docs/epics/
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

## Part 4 — The Index File Problem

### Discovery (2026-02-27)

While validating `homeserver/automation/docs/epics/` against a v2 `.stem`, rootline reported 0 errors and 375 valid files. But manual inspection found **11 estado drifts** — parent READMEs with `estado` values inconsistent with their children.

Rootline's `DetectDrift()` actually caught 9 of 11 (as `drift_warnings`, a separate output section from `errors`). The 2 missed cases were non-unanimous children (mix of `Completado` + `Obsoleto`).

But the deeper problem emerged: **rootline doesn't know what a README.md IS**.

### The Missing Concept

The engine knows:
- `require_index: README.md` → "this file must exist in subdirectories"
- `groupByParentDir()` → hardcodes `"README.md"` as the parent record

The engine does NOT know:
- **README.md is the directory node** — not just "a required file" but "the representation of this directory as a record"
- **Other files are children** — T001.md, T002.md etc. are leaf records belonging to the directory
- **estado in README.md should be computed** — it derives from children, not from manual input
- **A file that matches no expected pattern is an intruder** — `notas.md` in `S001/` should be an error

Example of the problem:

```
S006-final-opentofu-cleanup/
├── README.md          → estado: Pending     ← WRONG (should be Completado)
├── T001-xxx.md        → estado: Completado
├── T002-xxx.md        → estado: Completado
├── T003-xxx.md        → estado: Completado
├── T004-xxx.md        → estado: Completado
└── T005-xxx.md        → estado: Completado
```

The README is the Story. The T*.md files are its Tasks. The README's `estado` should be derived from the Tasks, not written by hand. But rootline has no way to know this because the schema doesn't declare the semantic relationship between the index file and its siblings.

### What the Schema Needs to Express

Three things the filesystem cannot tell the engine:

1. **Which file is the index** — "README.md in this directory IS the directory's record"
2. **What children look like** — "files matching T### are valid children; anything else is an intruder"
3. **How derived fields are computed** — "estado in the index = f(children estados)"

Everything else (parent-child relationships, nesting depth, sibling relationships) is intrinsic to the filesystem and should NOT be redeclared.

---

## Part 5 — Design Exploration: v3 Entity Model

### The Circular Problem

Multiple approaches were explored to extend the `.stem` format. Each iteration converged back to re-declaring hierarchy:

| Approach | What it looked like | Why it failed |
|----------|-------------------|---------------|
| v2 `match:` path-aware | `"E*/F*/S*/T*": {prefix: T}` | Verbose, re-encodes the filesystem path |
| `structural.children` | `children: { match: { "S*": "T*.md" } }` | Re-declares what the filesystem shows |
| v3 `entities:` | `E: { children: [F] }` | Essentially `levels:` renamed |
| v3 with `aggregate:` per entity | `E: { aggregate: { estado: "..." } }` | Back to per-level declarations |

**Core tension**: the principle says "don't redeclare hierarchy" but the engine needs to know entity types to scope fields and compute aggregates. Every attempt to define entity types ends up re-declaring the hierarchy as a side effect, because entity types ARE the hierarchy.

### Proposed v3 Format (work in progress)

Despite the circular tension, the entity model is the clearest expression found:

```yaml
version: 3

common:
  estado:
    type: enum
    required: true
    values: [Pending, In Progress, Specified, Completado, Diferida, Bloqueada, Obsoleto]
  cliente:
    type: string

entities:
  E:
    index: README.md
    extends: [common]
    children: [F]
    fields:
      id: { type: sequence, prefix: E, digits: 2 }
    aggregate:
      estado: <expr>    # computed from children

  F:
    index: README.md
    extends: [common]
    children: [S]
    fields:
      id: { type: sequence, prefix: F, digits: 2 }
      tipo: { type: enum, severity: warn, values: [...] }
    aggregate:
      estado: <expr>

  S:
    index: README.md
    extends: [common]
    children: [T]
    fields:
      id: { type: sequence, prefix: S, digits: 3 }
    aggregate:
      estado: <expr>    # may differ from E/F

  T:
    extends: [common]
    children: []
    fields:
      id: { type: sequence, prefix: T, digits: 3 }
      tipo: { type: enum, required: true, values: [...] }
      ejecutable_en: { type: string, severity: error }
      hold: { type: string }
    derive:
      estado: |
        hold != nil ? "On Hold" :
        blocked_by != nil && !all(blocked_by, {# == "Completado"}) ? "Bloqueada" :
        estado

links:
  allowed: [blocks, reference]
  rules:
    blocks:
      target_entity: T
      field: blocked_by

validate:
  - rule: requires
    when: { entity: T, tipo: software-module }
    then: { fields: [ejecutable_en] }
    severity: error
```

### What v3 Solves

- **Index semantics**: `index: README.md` means "this file IS the directory node"
- **Children validation**: entity type regex (derived from `id` prefix+digits) validates what belongs; anything else is an intruder
- **Derived vs written fields**: entities with `aggregate:` have computed fields; leaf entities (`children: []`) have written fields
- **Validate scoping**: `when: { entity: T }` eliminates cross-level constraint conflicts (e.g., `tipo: ci-cd` requiring `ejecutable_en` only fires for T entities, not F entities that also have `tipo`)
- **Bug fix**: v2 `match:` couldn't prevent a validate rule from requiring a field that doesn't exist at that level

### What v3 Doesn't Solve

- **Hierarchy re-declaration**: `children: [F]` is functionally identical to v1's `levels.epic.children: [feature]` — the filesystem already says E01/ contains F01/
- **Aggregate repetition**: if E, F, S all use the same aggregation formula, it's repeated three times (or requires a named reference mechanism)
- **The principle violation**: the core insight was "don't declare what the filesystem provides" — but entity types with `children:` do exactly that

### Possible Resolution

The `children:` declaration may not be hierarchy re-declaration but **type constraint**: "inside an E directory, only F-type entities are valid." The filesystem says E01/ contains F01/, but it doesn't say F01/ MUST be an F-type entity. A misplaced S001/ directory inside E01/ would be structurally valid to the filesystem but semantically wrong.

This reframes `children:` from "declaring hierarchy" to "constraining what entity types are valid at each nesting level" — which IS new information that the filesystem alone cannot provide.

Whether this reframing resolves the philosophical tension or merely rationalizes it remains an open question.

---

## Part 6 — State of the Art (Extended)

### Additional Systems Surveyed

| System | How it handles index files | How it handles hierarchy | Pattern syntax |
|--------|--------------------------|-------------------------|----------------|
| **Hugo** | `_index.md` = branch bundle (has children); `index.md` = leaf bundle (no children). The `_` prefix is the entire declaration. | Implicit from directory structure. `cascade` propagates values down. | N/A — convention-based |
| **Zola** | `_index.md` creates a section. Without it, directory is invisible. `transparent: true` flattens hierarchy. | Implicit from directory structure. Front matter controls sorting, pagination. | N/A — convention-based |
| **Next.js App Router** | `page.tsx` = the route, `layout.tsx` = wraps children, `loading.tsx` = skeleton. Each filename IS the semantic role. | Implicit from directory nesting. Dynamic segments via `[slug]`. | Convention: reserved filenames |
| **Remix** | `route.tsx` inside a folder, or flat files with dot notation. | Dot notation: `concerts.trending.tsx` = `/concerts/trending`. Parent found by longest common prefix. | Dots encode hierarchy in filenames |
| **EditorConfig** | N/A — configuration, not content. | Walk-up discovery. If pattern contains `/`, it's path-relative. Without `/`, matches at any depth. | Glob with path-awareness via `/` |
| **directory-schema** | N/A — validates directory tree as JSON against JSON Schema. | Converts `tree -J` to JSON, validates structure with standard JSON Schema. | JSON Schema patterns on names |
| **CUE** | N/A — not filesystem-based. | Lattice-based unification. Constraints only narrow. Values and types are the same thing. | CUE expressions |

### Key Insight from EditorConfig

EditorConfig's `/` convention elegantly separates "match anywhere" from "match at specific path":
- `*.md` → matches at any depth
- `docs/**/*.md` → matches only inside `docs/`

This principle could apply to `.stem` match patterns: `T*` means "any entity named T-something regardless of depth" while `S*/T*` means "T-something that's a direct child of S-something." However, in practice this re-encodes hierarchy in patterns — the same circular problem.

### Key Insight from Hugo/Zola

The `_index.md` convention is maximally elegant: **zero configuration, pure convention**. The presence and name of the file IS the declaration. Rootline's `index: README.md` is close but requires explicit declaration per entity type.

A fully convention-based approach would be: "any `README.md` is the directory's index file, always." No declaration needed. This is what `groupByParentDir()` already hardcodes — the question is whether to formalize or keep it as convention.

---

## Part 7 — Open Questions (Revised)

### Q1: Is `children:` hierarchy re-declaration or type constraint?
If `children: [F]` means "only F-type entities are valid inside E", it's new information. If it means "E contains F", it's redundant. The answer depends on whether the engine should reject a misplaced entity (S001/ directly inside E01/) or silently accept it.

### Q2: Should `index:` be convention or declaration?
Hugo chose convention (`_index.md`). Rootline currently mixes both (hardcoded in code, declared in structural). If every entity uses `README.md`, declaration is boilerplate. If different entities could use different index filenames, declaration has value.

### Q3: Can aggregate formulas be inherited?
If E, F, S all compute `estado` the same way, should `common` carry the aggregate expression with per-entity override capability? Or should aggregate always be per-entity because the semantics genuinely differ at each level?

### Q4: v2 → v3 migration path
v2 with `match:` works today. v3 with `entities:` is cleaner but requires engine changes. Options:
- **a)** v2 and v3 coexist — engine supports both
- **b)** `rootline migrate` converts v2 → v3
- **c)** v3 is a future major version (post-1.0)

### Q5: External review feedback — cross-level constraint conflicts
An external review identified that v2's flat `validate:` rules can create impossible constraints: `tipo: ci-cd` in F-level triggers `requires ejecutable_en`, but `ejecutable_en` only exists at T-level. v3's `when: { entity: T }` scoping fixes this, but v2 has no mechanism to prevent it. This is a concrete bug that argues for entity-scoped validation.

---

## References

### Hierarchical Database Theory
- IBM IMS Hierarchical Database Model — [Wikipedia](https://en.wikipedia.org/wiki/Hierarchical_database_model), [O'Reilly](https://www.oreilly.com/library/view/an-introduction-to/9780132886987/ch07.html)
- Czerwinski, David, Murlak, Parys — "Reasoning About Integrity Constraints for Tree-Structured Data" (ICDT 2016) — [Dagstuhl](https://drops.dagstuhl.de/entities/document/10.4230/LIPIcs.ICDT.2016.20), [Springer](https://link.springer.com/article/10.1007/s00224-017-9771-z)
- Schema Evolution for XML — [ResearchGate](https://www.researchgate.net/publication/220976005)
- OLAP Hierarchical Aggregation — [Medium](https://medium.com/learning-sql/olap-hierarchical-aggregation-with-sql-6c45ebc206d7), [SIGMOD Record](https://sigmodrecord.org/publications/sigmodRecord/0003/pourabbas.pdf)

### Filesystem Content Systems
- Hugo Page Bundles (`_index.md` vs `index.md`) — [Hugo docs](https://gohugo.io/content-management/page-bundles/)
- Hugo Content Organization — [Hugo docs](https://gohugo.io/content-management/organization/)
- Hugo cascade — [Hugo docs](https://gohugo.io/content-management/front-matter/)
- Zola Sections (`_index.md`) — [Zola docs](https://www.getzola.org/documentation/content/section/)
- Dendron schemas — [Dendron wiki](https://wiki.dendron.so/notes/c5e5adde-5459-409b-b34d-a0d75cbb1052/)
- Statamic data inheritance — [Statamic docs](https://statamic.dev/data-inheritance)
- Astro Content Collections — [Astro docs](https://docs.astro.build/en/guides/content-collections/)

### Filesystem Routing Conventions
- Next.js App Router file conventions — [Next.js docs](https://nextjs.org/docs/app/getting-started/project-structure)
- Remix flat routes (dot notation) — [Remix docs](https://remix.run/docs/en/main/file-conventions/route-files-v2)

### Configuration Inheritance
- EditorConfig specification (glob patterns, walk-up discovery) — [EditorConfig spec](https://spec.editorconfig.org/index.html)
- Apache .htaccess — [Apache docs](https://httpd.apache.org/docs/current/howto/htaccess.html)

### Schema & Constraint Systems
- CUE language (lattice unification) — [CUE docs](https://cuelang.org/docs/introduction/), [Discussion](https://github.com/cue-lang/cue/discussions/669)
- JSON Schema inheritance — [json-schema.org](https://json-schema.org/blog/posts/modelling-inheritance)
- directory-schema (HubMAP, archived) — [GitHub](https://github.com/hubmapconsortium/directory-schema)
- GNU Recutils — [gnu.org](https://www.gnu.org/software/recutils/)
