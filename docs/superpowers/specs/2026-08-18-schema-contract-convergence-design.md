---
estado: Specified
---

# Schema and Body-Source Contract Convergence Design

**Date:** 2026-08-18
**Status:** Approved design
**Issues:** #142, #144, #148, #151, #152
**Related follow-up:** #190

## Purpose

Make Rootline's field declarations executable, deterministic contracts across schema generation, body extraction, validation, inheritance, diagnostics, query, scaffolding, and machine-readable output.

This design supports the approved product north star:

> Rootline turns Markdown into a governed, queryable knowledge system.

A governed system cannot accept values that contradict declared types, assign two meanings to the same field, reject a valid child refinement in one subsystem while accepting it in another, or emit machine contracts that change with the checkout location.

## Decisions

1. Stem inheritance is always monotonic. No `semantics: monotonic` flag or permissive compatibility mode will be introduced.
2. Child `string → enum` is valid narrowing.
3. `schema.<field>.type` is an enforceable value constraint for every supported Rootline type.
4. Markdown body location is declared through `source`; it is not a value type.
5. `type: section`, `heading`, and `ordered` are legacy declarations and will not remain as a parallel schema dialect.
6. The canonical section form is a real type plus `source: body.section["<exact heading>"]`.
7. An inherited field's source binding is stable: a child may inherit it but may not change or remove it.
8. `required` means presence only. Empty values remain present.
9. `non_empty` remains a separate explicit content constraint.
10. `exists` continues to require an effective field, including derived fields.
11. A present heading with an empty body resolves as `present=true, value=""`.
12. Duplicate matching headings are ambiguous and produce an actionable error; Rootline never silently selects the first or last occurrence.
13. Path-like `results[].errors[].source` values are relative to the governance root. Symbolic sources remain symbolic.
14. Multiple sections materialized in one operation are ordered lexically by their exact source heading.
15. `boolean` and `integer` are enforceable value types; legacy `type: bool` migrates to canonical `type: boolean`.
16. Enums accept one or more declared values; only an empty domain is invalid.
17. Section inference rejects logical-name collisions and requires explicit field names; it never invents suffixes.
18. One shared field contract owns value conformance, source resolution, and parent-to-child compatibility.
19. No schema mode, envelope version, migration verb, or automatic value coercion will be added.

## Scope

### In scope

- Central supported-type vocabulary and value-conformance rules.
- Canonical body-source parsing and resolution.
- Monotonic type and source-binding compatibility.
- Record-level type validation.
- Stem-health rejection of unknown, incomplete, or legacy field declarations.
- Section inference and schema generation that converge with validation.
- `new` and `migrate --scaffold` materialization of required body-sourced fields.
- Preservation of source declarations through serializers and schema splitting.
- Query, describe, and explain consumption of the same effective field.
- Portable validation error provenance.
- Focused runtime, CLI contract, and documentation tests.
- Living documentation and Rootline agent-skill synchronization.

### Out of scope

- Automatic conversion between YAML representations.
- A permissive legacy inheritance mode.
- A new `.stem` or validation-envelope version.
- New meanings for whitespace-only strings or empty lists under `non_empty`.
- Guessing a heading when a legacy declaration does not identify one.
- Ancestor-qualified section selectors; hierarchical source paths are tracked separately in #190.
- Treating `set` as a Markdown body editor; it remains a frontmatter override operation.
- Replacing the independent repair-surface `set_section` operation.
- Rewriting completed roadmap records or historical designs.
- Unrelated validation, inference, migration, or documentation issues.

## Architecture

### Shared field contract

`internal/rules` owns one field-contract component. It answers:

1. Is a declared value type supported and sufficiently configured?
2. Is a declared source supported and unambiguous?
3. Does a resolved value conform to its declared type?
4. Is a child field a valid monotonic refinement of its inherited field?

Validation, layered resolution, stem health, query enrichment, and inspection consume this contract instead of maintaining independent field semantics.

The implementation may use focused internal files such as `type_contract.go` and `field_source.go`; filenames are not normative.

### Supported value-type vocabulary

| Type | Accepted representation | Additional semantic constraint |
|---|---|---|
| `string` | YAML or extracted string | None |
| `list` | YAML sequence | None on item values |
| `enum` | YAML or extracted string | Value MUST occur in `values`; one or more declared values are valid and an empty domain is invalid |
| `sequence` | YAML or extracted string | Value MUST conform to the declared `prefix` plus exactly `digits` decimal digits; both settings are required |
| `link` | String, or list of strings | Every accepted string MUST contain the currently supported wiki-link syntax |
| `boolean` | YAML boolean | No string coercion; legacy `type: bool` is rejected with migration guidance |
| `integer` | YAML integer | Floating-point and quoted numeric values are not integers |

`section` is absent from this table because it identifies a storage location, not a value representation.

An unknown type is not forward-compatible data. It is an uninterpretable governance declaration and MUST produce a failing stem-health diagnostic.

### Canonical source binding

A body-backed field composes a real value type with a source directive:

```yaml
summary:
  type: string
  source: body.section["## Summary"]
  required: true
```

The source resolver returns three independent outputs:

- the resolved value;
- whether the source is present;
- a resolution error, including ambiguity.

This distinction prevents an empty-but-present section from being treated as absent.

The supported body-source directives are `body.h1` and `body.section[...]`. `body.section[...]` matches the exact heading level and text declared in the directive. Any other source directive fails as an unsupported schema declaration.

A body source is a fallback. Existing frontmatter precedence remains explicit: a frontmatter field with the same logical name overrides the extracted body value. `set` therefore writes a deliberate frontmatter override; it does not mutate the body section.

### Duplicate headings

When a section source matches more than one heading in one document, resolution fails as ambiguous. AST-backed and text-backed extraction MUST preserve enough occurrence information to produce the same result. A map that silently overwrites duplicate headings is not an admissible contract boundary.

### Stable source inheritance

For an inherited logical field:

- an omitted child source inherits the parent source;
- the same explicit source is valid;
- a changed source is incompatible;
- explicitly removing the source is incompatible.

A child that needs a different body section defines a different logical field. This keeps query and agent-facing field identity stable across the directory hierarchy.

### Monotonic type relation

| Parent | Child | Result |
|---|---|---|
| Any supported type | Same type | Valid |
| `string` | `enum` | Valid narrowing |
| `enum` | `enum` with a subset of parent values | Valid narrowing |
| `enum` | `enum` with additional values | Invalid widening |
| `enum` | `string` | Invalid widening |
| Any other unequal pair | Different type | Invalid unless a future approved contract explicitly adds the relation |

The generic stem-health behavior that rejects every type difference MUST be replaced by this relation. A valid narrowing MUST NOT also produce `field-override` noise.

## Validation Flow

For each effective schema field, Rootline performs these steps in order:

1. Resolve frontmatter precedence and the declared fallback source into `(value, present, error)`.
2. If source resolution is ambiguous or invalid, emit its actionable error.
3. Apply presence semantics.
4. If the value is present, validate its representation against the shared type contract.
5. If representation succeeds, run type-specific semantic checks.
6. Run applicable explicit validation rules.

### Presence semantics

- `required`: the schema-resolved field must be present. `""` and `[]` count as present.
- `non_empty`: the effective field must exist and MUST NOT be the exact empty string. Existing whitespace and collection behavior remains unchanged.
- `exists`: the effective field must exist; derived fields count and empty values remain allowed.

These constraints are intentionally separate. Presence, representation, and content quality are different dimensions.

### Error precedence

A representation mismatch produces one `ValidationError` with:

- `rule: "type"`;
- the field name;
- the expected Rootline type;
- the received YAML representation;
- the source `.stem`;
- the field's configured severity.

After a type mismatch, Rootline MUST skip semantic checks that depend on the expected representation. A string supplied for `list`, for example, must not also produce derivative enum, sequence, or link errors.

Invalid schema declarations belong to stem health, not record validation. A record cannot meaningfully conform to an unknown, incomplete, or legacy field declaration.

## Section Inference and Materialization

### Inference

Section-pattern detection MUST:

1. preserve the exact heading level and text;
2. count every record in the denominator, including documents without a body and headings with empty bodies;
3. infer a candidate field when the configured threshold is met;
4. mark it `required` only when the heading is present in 100% of the analyzed records;
5. emit `type: string` plus the exact `source: body.section[...]` directive;
6. detect when distinct exact headings normalize to the same logical field name and fail with all colliding headings listed.

Threshold-level evidence justifies discovering an optional field. It does not justify converting observed absence into a required constraint. A collision requires explicit field names and sources; inference never invents heading-level or numeric suffixes.

`init`, `schema propose`, and analyze/schema-apply paths MUST use the same inference result and MUST serialize the source directive. No path may claim a section inference was handled while silently discarding it.

### New documents and scaffold

`new` and `migrate --scaffold` MUST inspect canonical required source-backed fields rather than branching on `type: section`.

For a required `body.section[...]` field that is absent:

- materialize the exact declared heading in the body;
- use the field `default` as body content when non-empty, otherwise use `<!-- TODO -->`;
- order multiple generated sections lexically by exact heading;
- do not write an empty frontmatter key that would shadow the source;
- validate the resulting document against the same schema contract.

A legitimate corpus with no missing required sections may remain a successful zero-result operation. An invalid legacy declaration, ambiguous source, or schema-resolution failure MUST NOT be reported as a clean no-op.

### Generic transport and inspection

- Stem serialization and `migrate --split` preserve source declarations.
- Schema diffing recognizes source changes as incompatible field-binding changes.
- Query resolves source-backed fields by logical field name through the shared resolver.
- Describe exposes both physical `.stem` provenance and the logical extraction directive without conflating their labels.
- Explain resolves and reports the same effective value and source binding used by validation and query.
- The independent repair proposal `set_section` remains a document-repair operation, not another schema representation.

## Legacy Declaration Migration

Legacy forms are rejected with an actionable diagnostic:

```yaml
# Before
notes:
  type: section
  heading: "## Notes"
```

```yaml
# After
notes:
  type: string
  source: body.section["## Notes"]
```

`heading` and `ordered` are not retained as parallel field metadata. A declaration such as a heading-shaped field key with no explicit heading metadata cannot be migrated by guessing; the diagnostic requires the author to name the intended source.

No new migration command is introduced. The correction is a bounded schema edit whose target form is printed by the diagnostic.

## Portable Error Provenance

Rootline keeps physical paths internally for reading files and tracing rules. Before a validation result enters the public envelope, path-like `source` values are projected relative to the record's governance root.

Rules:

1. The base is the directory governed by the root-most `.stem` with `root: true`, not the process working directory or the `--all` scan subdirectory.
2. JSON paths use `/` separators on every platform.
3. The same record and rule produce the same `source` under single-file and `--all` validation.
4. Symbolic sources such as `schema` and `scope` remain unchanged.
5. Existing result-path semantics are outside this change; only path-like error provenance is normalized.

Because schema discovery bounds governing stems to the governance root, a governing `.stem` source is expected to be representable without escaping that root.

## Compatibility and Migration Impact

This is an observable contract correction:

- Records whose values contradict declared types will begin failing validation.
- Unknown and incomplete type declarations will begin failing stem health.
- Legacy `type: section`, `heading`, and `ordered` declarations will require the printed source-backed migration.
- Legacy `type: bool` declarations will require `type: boolean`; existing YAML booleans remain unchanged.
- Existing `type: integer` declarations and YAML integers remain canonical and become enforceable.
- Single-value enum domains remain strict constraints; legacy `enum:` keys migrate to canonical `values:` without inventing another value.
- Automatically materialized sections will use lexical heading order instead of legacy `ordered` metadata or map iteration.
- Empty headings will satisfy `required` but fail `non_empty` when that explicit rule applies.
- Duplicate headings targeted by one source will become validation errors.
- Consumers expecting absolute `errors[].source` values must resolve the new value against the governance root.
- Valid `string → enum` narrowing will stop failing `type-consistency`.
- A child that changes an inherited source will become a schema conflict.

No automatic coercion is permitted. Converting `"12"` to another representation, flattening a list, inventing a missing value, or guessing a heading would erase the invalid-data signal.

Users inspect the impact with existing commands:

```bash
rootline validate --all <dir> -o json
rootline analyze <dir> -o json
```

They then correct the record or amend the schema. Release notes MUST identify strict type conformance, section-source migration, duplicate-heading rejection, stable source inheritance, and relative error sources as compatibility changes.

## Requirements and Scenarios

### R1 — One value-type vocabulary

Rootline MUST use one supported value-type vocabulary across parsing diagnostics, validation, inheritance, and documentation.

**GIVEN** a `.stem` declares an unknown type
**WHEN** stem health runs
**THEN** validation fails with an actionable schema diagnostic rather than accepting records without enforcing the declaration.

### R2 — Separate value type from source

A Markdown body location MUST be represented as a source on a real value type.

**GIVEN** a field declares `type: string` and `source: body.section["## Summary"]`
**WHEN** the section exists
**THEN** validation, query, describe, and explain observe the same logical string field.

### R3 — Reject legacy section dialects

Rootline MUST NOT accept `type: section`, `heading`, or `ordered` as a second representation.

**GIVEN** a legacy section declaration
**WHEN** stem health runs
**THEN** it fails with the exact canonical `type + source` migration shape.

### R4 — Enforce declared representation

Every present field MUST conform to its supported declared type.

**GIVEN** a field declares `type: list`
**WHEN** a record supplies a YAML string
**THEN** Rootline emits one `rule: "type"` validation error.

### R5 — Preserve presence distinctions

`required`, `non_empty`, and `exists` MUST retain independent semantics.

**GIVEN** a required string field exists with value `""`
**WHEN** no `non_empty` rule applies
**THEN** the field passes the presence constraint.

**GIVEN** the same field has a `non_empty` rule
**WHEN** validation runs
**THEN** Rootline emits a `non_empty` error.

### R6 — Preserve empty section presence

A heading and its body content MUST be represented independently.

**GIVEN** the declared heading exists with no body content
**WHEN** a required source-backed field resolves
**THEN** it is present with value `""`.

### R7 — Reject ambiguous duplicate headings

A body source MUST resolve to at most one section.

**GIVEN** the exact declared heading occurs twice
**WHEN** any consumer resolves the field
**THEN** Rootline emits the same actionable ambiguity error instead of choosing an occurrence.

### R8 — Generate schemas that validate their source corpus

Schema inference MUST preserve observed optionality and emit the canonical source form.

**GIVEN** a heading appears in four of five records
**WHEN** `init` or schema generation runs
**THEN** it emits an optional `type: string` source-backed field and the generated schema does not reject the fifth source record.

### R9 — Materialize required sections through the source contract

Document generation and scaffold MUST consume canonical source metadata.

**GIVEN** a required section source is absent
**WHEN** `new` or `migrate --scaffold` materializes the document
**THEN** the exact heading is added to the body and subsequent validation observes it as present.

**GIVEN** multiple required section sources are absent
**WHEN** one operation materializes them
**THEN** their headings are appended in lexical order.

### R10 — Keep inherited source identity stable

A child field MUST NOT change or remove its inherited source binding.

**GIVEN** a parent binds `summary` to `body.section["## Summary"]`
**WHEN** a child binds `summary` to another section or removes the source
**THEN** stem health reports an incompatible field-binding change.

### R11 — Accept valid type narrowing

Child constraints MUST be allowed to reduce, but never enlarge, the parent value domain.

**GIVEN** a parent declares `type: string`
**AND** a child declares `type: enum` with values
**WHEN** layered resolution and stem health run
**THEN** both accept the child declaration without `type-consistency` or `field-override` diagnostics.

### R12 — Reject widening consistently

Every consumer of the type relation MUST reject the same widening.

**GIVEN** a parent enum and a child string
**WHEN** layered resolution, stem health, and CLI validation inspect the hierarchy
**THEN** they classify the child as an invalid widening using the same contract.

### R13 — Avoid derivative error noise

Rootline MUST stop semantic validation after a representation mismatch for that field.

**GIVEN** an enum field contains a YAML list
**WHEN** validation runs
**THEN** Rootline emits a type error and does not also emit enum-membership suggestions.

### R14 — Emit portable provenance

Path-like validation sources MUST be stable across machines and invocation modes.

**GIVEN** single-file and `--all` validation inspect the same record
**WHEN** the same inherited rule fails
**THEN** both envelopes contain the same governance-root-relative `source` using `/` separators.

### R15 — Preserve symbolic provenance

Source normalization MUST NOT reinterpret non-path identifiers.

**GIVEN** an error source is `schema` or `scope`
**WHEN** the envelope is emitted
**THEN** the source remains unchanged.

### R16 — Enforce active scalar types without coercion

Boolean and integer declarations MUST validate their native YAML representations.

**GIVEN** a field declares `type: boolean`
**WHEN** a record contains the string `"true"`
**THEN** Rootline emits a type error rather than coercing it.

**GIVEN** a field declares `type: integer`
**WHEN** a record contains the YAML integer `3`
**THEN** it passes type conformance.

### R17 — Reject inferred logical-name collisions

Inference MUST NOT silently overwrite or rename distinct headings that normalize to one logical field.

**GIVEN** `## Notes` and `### Notes` both normalize to `notes`
**WHEN** section inference generates a schema candidate
**THEN** it fails with both exact headings and requests explicit field names.

### R18 — Preserve single-value enum constraints

An enum MUST declare at least one value and MAY declare exactly one.

**GIVEN** a field declares `values: [theory]`
**WHEN** a record contains `theory`
**THEN** it passes enum validation.

**WHEN** another record contains `hypothesis`
**THEN** it fails enum validation.

## Error Handling

- Schema declaration and inheritance failures appear in `stem_health` and follow existing strict/error exit rules.
- Record conformance and source-resolution failures appear in `results[].errors[]` with the field severity.
- A failure to resolve or scan the governed corpus remains represented through the existing validate envelope notices.
- Scaffold and schema-application paths MUST propagate resolution failures rather than converting them into successful zero-result operations.
- No failed validation path may emit a successful exit merely because the schema or source could not be interpreted.

## Test Strategy

### Unit field-contract matrix

Table-driven tests cover every supported type, including native YAML booleans and integers, with valid and invalid values, empty and single-value enum domains, incomplete declarations, unknown types, canonical sources, legacy `bool`, and legacy section declarations.

### Monotonic algebra

Focused tests cover:

- identical types and sources;
- `string → enum`;
- enum subset;
- enum extension;
- `enum → string`;
- incompatible unequal types;
- inherited source equality;
- changed and removed child sources.

The existing resolver test and contradictory CLI/stem-health test for #148 are reconciled against this matrix.

### Source resolution parity

Tests prove AST-backed and text-backed consumers agree on:

- exact heading level and text;
- absent heading;
- present empty heading;
- duplicate heading ambiguity;
- frontmatter override precedence.

### Inference and round-trip convergence

Tests prove:

- inferred headings preserve their exact source;
- threshold matches are optional unless present in every record;
- logical-name collisions fail without invented suffixes;
- init-generated schema validates its source corpus;
- schema propose/apply does not discard section inferences;
- serializers and `migrate --split` preserve source declarations;
- schema diff reports incompatible source changes.

### Materialization convergence

Tests exercise complete loops:

- `new → validate`;
- `migrate --scaffold → validate`;
- legitimate scaffold no-op;
- lexical ordering for multiple generated sections;
- malformed and legacy contracts produce non-success diagnostics;
- no empty frontmatter override shadows a generated body section.

### Validation precedence

Tests prove:

- one type error per mismatched field;
- derivative semantic checks do not run after mismatch;
- `required`, `non_empty`, and `exists` remain distinct;
- body-sourced and frontmatter-precedence fields receive the same type enforcement.

### Query and inspection

Tests exercise real command paths rather than manually populated section maps. Query, describe, and explain MUST report the same logical value and source binding.

### CLI provenance

Tests compare single-file and `--all` output, assert governance-root-relative sources, preserve symbolic sources, normalize separators, and reject absolute-path leakage in the v2 envelope.

### Documentation contracts

Living examples for types, body sources, hierarchy, presence, generation, scaffold, and validate output are backed by executable fixtures or captured contract tests where practical.

Defect-pinning tests that accept `type: string + heading`, `type: section`, successful unrecognized scaffold no-ops, or first/last duplicate selection are replaced rather than retained as compatibility tests.

## Surface Inventory

Expected implementation surfaces include:

- `internal/rules/` — shared field contract, record validation, layered compatibility, stem health, describe/explain inputs.
- `internal/extract/` — source resolution with explicit presence and ambiguity.
- `internal/infer/` — exact section inference and canonical schema generation.
- `internal/migrate/` — canonical scaffold, split preservation, and source-aware diffing.
- `cmd/rootline/init.go`, `schema.go`, `new.go`, `set.go`, `query.go`, `describe.go`, `explain.go`, and `validate.go` — producer/consumer convergence and output normalization.
- Focused tests under the corresponding packages.
- Living docs for validation, initialization, migration, query, set, extensibility, and schema hierarchy.
- `.claude/skills/rootline/` — synchronized agent-facing field semantics.
- `CHANGELOG.md` or the repository's release-note surface for compatibility guidance.

Exact files may narrow during implementation, but scope MUST NOT expand beyond contract convergence for #142, #144, #148, #151, and #152 without a new design decision. Hierarchical section selectors remain outside this scope under #190.

## Delivery Slices

1. **Field algebra and source resolution** — central value types, monotonic type/source compatibility, explicit presence, duplicate ambiguity, and focused unit tests.
2. **Schema producers and transport** — inference, init, schema propose/apply, serialization, split, diff, and source-corpus convergence tests.
3. **Document materialization and runtime consumers** — new, scaffold, validate, query, describe, explain, set override behavior, and end-to-end convergence tests.
4. **Portable provenance** — governance-root-relative error sources and cross-mode CLI tests.
5. **Living contract synchronization** — docs, agent skill, legacy migration guidance, compatibility notes, and final captured examples.

Each slice keeps code, tests, and documentation for its contract together. The implementation plan must forecast review workload and split delivery further if any slice exceeds the repository's review budget.

## Historical Records

Completed roadmap and design records remain unchanged, including O10/T001 and O14/T001/T007. Living documentation may state that monotonic semantics are universal and section location is expressed through `source`, but historical records preserve the mechanisms and contradictions recorded at their time.
