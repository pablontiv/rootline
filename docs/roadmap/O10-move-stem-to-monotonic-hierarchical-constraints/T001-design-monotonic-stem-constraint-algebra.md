---
estado: In Progress
tipo: task
---
# T001: Design monotonic stem constraint algebra

**Outcome**: [O10 Move .stem to monotonic hierarchical constraints](README.md)
**Contribuye a**: CE1 del Outcome.

## Preserva

- INV1: Moving down the directory tree must never silently reduce parent guarantees.
  - Verificar: the design explicitly classifies reductions as invalid or explicit evolution.
- INV3: v2 compatibility or migration behavior is explicit.
  - Verificar: the design names the version/flag/migration strategy.

## Contexto

Research concluded that Rootline is closer to a hierarchical filesystem database than a config cascade. Parent `.stem` files should declare guarantees for descendants; child `.stem` files should add or narrow constraints. Current merge semantics allow destructive replacement, required loosening, array replacement, and nil removal.

## Alcance

**In**:
1. Define the partial order for field type, required, enum values, severity, match/scope, excludes, default, sequence, domain, section metadata, links, derive, aggregate, and structural rules.
2. Decide how child enum values behave: subset/narrowing, extension/widening, or explicit replacement.
3. Decide how defaults are classified: constraint, authoring hint, or generation policy.
4. Decide compatibility strategy: v3, `semantics: monotonic`, project setting, or migration command.
5. Document examples for valid narrowing and invalid destructive override.

**Out**:
- Implementing resolver changes.
- Applying schema evolution operations.

## Estado inicial esperado

- Current `MergeStemFiles` is a YAML cascade: maps merge, arrays/scalars replace, nil removes, schema fields replace wholesale with only severity tightening as an exception.

## Criterios de Aceptación

- A design document contains a matrix of allowed, rejected, and explicit-evolution operations.
- The design answers whether child `string -> enum` narrowing is valid.
- The design states how legacy v2 projects are handled.
- The design identifies tests needed for every constraint category.

## Fuente de verdad

- `internal/rules/merge.go`
- `internal/rules/rules.go`
- `internal/rules/merge_test.go`
- `internal/rules/stemhealth.go`
- `docs/levels.md`
- `README.md`

---

## Diseño: Álgebra de Restricciones Monotónicas

### Objetivo

Establish a formal partial order for `.stem` field properties such that child `.stem` files can only narrow, narrow-with-explicit-evolution, or keep constraints, never silently widen them. This preserves INV1 (parent guarantees flow down) and enables hierarchical refinement at each directory level.

### Partial Order Definition

A **narrowing** operation reduces a constraint scope; a **widening** operation increases it (destructive). The algebra classifies each property:

| Field Property | Type | Narrowing | Widening | Evolution | Notes |
|---|---|---|---|---|---|
| **type** | Constraint | X | X | YES (type: string → enum) | Child can specialize type (string to bounded enum). Requires `semantics: monotonic` flag. |
| **required** | Constraint | YES | **REJECTED** | X | Child can require-match narrower subset of paths. Loosening (required→optional) is rejected at parse time. |
| **values** (enum) | Constraint | YES | **REJECTED** | X | Child can narrow enum to subset. Extension requires explicit list addition with version bump. |
| **default** | Authoring hint | YES | YES | X | Defaults do not impose constraints on valid data. Child can override. No monotonic check needed. |
| **severity** | Constraint | YES (tighter) | **REJECTED** | X | Child can raise warn→error. Lowering is rejected (existing rule). |
| **scope/match** | Constraint | YES | YES | X | Child can narrow match pattern. Widening allowed (child can apply to more records if same constraint). |
| **excludes** | Constraint | YES | YES | X | Child can add excludes or narrow existing excludes. Removal is allowed (no exclusion is a widening that is permitted — fewer exceptions). |
| **prefix, digits** | Constraint | YES | X | X | Child can tighten sequence format (fewer valid values). Cannot loosen. |
| **domain** | Semantic type | X | X | NO | Domain is immutable once declared. Child redefining domain is a type-safety error. |
| **links.allowed** | Constraint | YES | **REJECTED** | X | Child can narrow allowed link types. Extending requires explicit merge semantics. Currently replaces (array semantics). |
| **links.rules** | Constraint | X | X | X | Rules map-merge at key level (existing behavior). Rule tightening (target narrowing) allowed, loosening rejected. |
| **derive** | Generation policy | YES | YES | X | Child can override/redefine derivation. No monotonic check (derive is authoring, not constraint). |
| **aggregate** | Generation policy | YES | YES | X | Child can override aggregation expression. No monotonic check (aggregate is query machinery, not constraint). |
| **structural** | Constraint | YES | **REJECTED** | X | Child can tighten min_children, max_children, require_index. Loosening rejected. |

### Child `string → enum` Narrowing — DECISION: VALID (with flag)

**Question**: Is child narrowing from `string` to `enum` valid?

**Answer**: **YES, VALID** — with explicit opt-in via `semantics: monotonic` flag.

**Rationale**:
- Semantically: `string` accepts any text; `enum` accepts only listed values. Enum is a proper subset.
- Constraint tightening: All valid enum values are valid strings. No existing data invalidated.
- Safety: Requires project-wide flag to avoid accidental breakage in legacy projects.
- Example: Parent declares `status: {type: string}` at Epic level. Feature `.stem` narrows to `{type: enum, values: [Active, Completed]}` for more control.

**Implementation**:
```yaml
version: 2
semantics: monotonic    # Enables type narrowing (string→enum, etc.)
schema:
  status:
    type: enum          # Child narrowing from parent's string
    values: [Active, Completed]
```

Validation: If parent field is `string` and child is `enum`, accept only if `semantics: monotonic` is set in child `.stem`. Emit warning if not set.

### Defaults: Classification as Authoring Hint

**Decision**: Defaults are **authoring hints**, not constraints.

**Rationale**:
- A default does not invalidate documents lacking that field.
- Parent default: `status: {type: enum, values: [...], default: "draft"}` — allows docs with no status.
- Child override: `status: {default: "active"}` — changes authoring UX, not validation.
- No monotonic check required for defaults.

**Implementation**:
- Defaults merge via scalar-replace rule (last writer wins).
- `rootline new` uses effective schema (merged defaults).
- Validation ignores defaults when checking required fields.

### Compatibility Strategy — DECISION: `semantics: monotonic` flag + v2 migration

**Decision**: Introduce optional `semantics: monotonic` flag in v2 `.stem` files. Legacy v2 projects without the flag use permissive merge; projects with `semantics: monotonic` enforce monotonic constraints.

**Rationale**:
- **No version bump**: Stays in v2 (only adds one optional field).
- **Backward compatible**: Existing v2 projects work unchanged.
- **Opt-in safety**: Projects can adopt monotonic semantics per-project.
- **Future path**: Could become default in v3 or enforced via `--strict` flag.

**Implementation**:
1. Parse `semantics: monotonic` from `.stem` root during `ParseStem`.
2. In `MergeStemFiles`, check `semantics` flag from child `.stem`:
   - If flag present: enforce narrowing-only for all inherited fields.
   - If absent: use existing permissive merge.
3. In validation, emit warnings for violations (configurable to errors via `--strict`).
4. `rootline analyze` detector: flag non-monotonic merges as schema-coverage gaps.
5. Migration command: `rootline migrate --to-monotonic` adds flag to all `.stem` files after validating no breaking changes.

### Valid Narrowing Examples

**Example 1: Required narrowing (match-based)**
```yaml
# Parent (root/.stem)
schema:
  cliente:
    type: string
    required: false

# Child (features/.stem)
schema:
  cliente:
    type: string
    required:
      match: "F*"    # Require only for Features
```
Valid: Child narrows required scope. Parent allows omitting cliente for non-F records; child requires it for F records.

**Example 2: Enum narrowing**
```yaml
# Parent (root/.stem)
schema:
  status:
    type: enum
    values: [Pending, Active, Completed, Archived]

# Child (tasks/.stem, with semantics: monotonic)
schema:
  status:
    type: enum
    values: [Pending, Active, Completed]  # Archived not allowed at task level
```
Valid: Child narrows enum subset. All valid task statuses are valid parent statuses.

**Example 3: Sequence tightening**
```yaml
# Parent (root/.stem)
schema:
  id:
    type: sequence
    prefix: "T"
    digits: 2

# Child (tasks/.stem)
schema:
  id:
    type: sequence
    prefix: "T"
    digits: 3    # More precise (T001-T999, not T00-T99)
```
Valid: Child tightens sequence format. More restrictive, but only affects new IDs (not backcompat with existing T00 IDs).

### Invalid Destructive Overrides (Rejected)

**Example 1: Loosening required (REJECTED)**
```yaml
# Parent (root/.stem)
schema:
  titulo:
    type: string
    required: true

# Child (features/.stem) — INVALID
schema:
  titulo:
    type: string
    required: false    # ERROR: Loosens parent constraint
```
Validation error: "titulo.required: child cannot loosen parent constraint from true to false"

**Example 2: Extending enum (REJECTED)**
```yaml
# Parent (root/.stem)
schema:
  status:
    type: enum
    values: [Pending, Completed]

# Child (tasks/.stem) — INVALID (without explicit evolution)
schema:
  status:
    type: enum
    values: [Pending, In Progress, Completed]  # ERROR: Added "In Progress"
```
Validation error: "status.values: child cannot extend parent enum; use explicit schema evolution"

**Example 3: Loosening structural constraint (REJECTED)**
```yaml
# Parent (root/.stem)
structural:
  subdirs:
    min_children: 2

# Child (features/.stem) — INVALID
structural:
  subdirs:
    min_children: 1    # ERROR: Loosens parent min_children
```
Validation error: "structural.min_children: child cannot loosen parent constraint from 2 to 1"

### Constraint Validation Matrix (Merge-time checks)

When `semantics: monotonic` is set, at merge-time for each field:

| Category | Check | Action on Violation |
|---|---|---|
| Type change | If parent type ≠ child type: child must be narrowing (string→enum, int→positive-int). | Error + suggest explicit evolution |
| Required | If parent required=true and child required=false: reject. If parent has match pattern and child narrows it: accept. | Error |
| Enum values | If child.values ⊂ parent.values: accept. If child.values ⊄ parent.values or extends: reject. | Error + suggest explicit enum merge syntax |
| Severity | If child.severity < parent.severity (tightness): reject. | Error (existing rule) |
| Match/scope | Child can narrow match pattern. Child can expand if constraint is identical or tighter. | Accept narrowing; warn on expansion if constraint loosens |
| Structural | If child constraint is looser (min→max, require→optional): reject. | Error |
| Domain | If parent has domain X and child redefines to domain Y: reject. | Error |

### Explicit Evolution Path (Future, out of scope for T001)

For future versions, allow explicit evolution via schema annotation:

```yaml
version: 2
semantics: monotonic
schema:
  status:
    type: enum
    values: [Pending, Completed]
    evolution:
      v2.1:
        added: [In Progress]      # v2.1 adds "In Progress"
      v2.2:
        removed: [Pending]        # v2.2 removes "Pending"
```

This is **not implemented in T001** but enables auditable schema changes.

### Legacy v2 Handling

**For existing v2 projects without `semantics: monotonic`**:
- Continue using permissive merge (current behavior).
- `rootline analyze` flags non-monotonic inheritance as schema-coverage gaps (informational).
- Migration: `rootline migrate --to-monotonic` walks all `.stem` files, validates no breaking changes, adds flag.
- If breaking changes detected during migration, halt and report issues for manual resolution.

**For new v2 projects**:
- Encourage `semantics: monotonic` in `.stem` templates (via `rootline init --template`).
- Document as best practice in README.

### Test Scenarios (at least one per constraint category)

Tests must cover:

1. **Type narrowing** (string→enum with flag)
   - `TestMonotonicTypeNarrowing_StringToEnum` — parent string, child enum → pass if semantics: monotonic, fail otherwise
   
2. **Required tightening** (false→true rejected)
   - `TestMonotonicRequiredLoosening_Rejected` — parent required=true, child required=false → error
   - `TestMonotonicRequiredNarrowing_Accepted` — parent required=false, child required: {match: "T*"} → pass
   
3. **Enum narrowing** (subset valid)
   - `TestMonotonicEnumNarrowing_Subset` — parent [A,B,C], child [A,B] → pass
   - `TestMonotonicEnumExtension_Rejected` — parent [A,B], child [A,B,C] → error
   
4. **Severity tightening** (warn→error valid)
   - `TestMonotonicSeverityTightening` — parent severity=warn, child severity=error → pass
   - `TestMonotonicSeverityLoosening_Rejected` — parent severity=error, child severity=warn → error (existing behavior)
   
5. **Scope narrowing** (match pattern)
   - `TestMonotonicScopeNarrowing` — parent match: "*.md", child match: "T*.md" → pass
   
6. **Structural tightening** (min/max)
   - `TestMonotonicStructuralTightening` — parent min_children=1, child min_children=2 → pass
   - `TestMonotonicStructuralLoosening_Rejected` — parent min_children=2, child min_children=1 → error
   
7. **Default override** (no constraint check)
   - `TestDefaultOverride_AllowedAnytime` — parent default="draft", child default="active" → pass (no check)
   
8. **Derive/Aggregate override** (no constraint check)
   - `TestDeriveOverride_AllowedAnytime` — parent derive: {slug: "..."}, child derive: {slug: "..."} → pass (no check)
   
9. **Domain immutability**
   - `TestDomainImmutable_Rejected` — parent domain: lifecycle_state, child domain: record_type → error
   
10. **Three-level hierarchy**
    - `TestMonotonicThreeLevelHierarchy` — gp→parent→child with cumulative narrowing → pass if all narrowing, fail if any widening

### Stem Health Diagnostic (new check 11)

Add to `stemhealth.go`:

```go
// Check 11: Monotonic narrowing violations (if semantics: monotonic)
for sf, stem := range parsedStems {
    if stem.Semantics != "monotonic" {
        continue
    }
    relPath, _ := filepath.Rel(absRoot, sf)
    dir := filepath.Dir(sf)
    parentEntries, walkErr := WalkUp(dir)
    if walkErr != nil || len(parentEntries) < 2 {
        continue
    }
    parentMerged := MergeStemFiles(parentEntries[:len(parentEntries)-1])
    if parentMerged == nil {
        continue
    }
    
    // For each field in child, check monotonic constraints
    for fieldName, childField := range stem.Schema {
        if parentField, exists := parentMerged.Schema[fieldName]; exists {
            // Check type narrowing
            if childField.Type != parentField.Type {
                if !isValidNarrowing(parentField.Type, childField.Type) {
                    checks = append(checks, StemHealthCheck{
                        Name: "monotonic-type-narrowing",
                        Status: "fail",
                        Message: fmt.Sprintf("field %q: invalid type change from %q to %q", fieldName, parentField.Type, childField.Type),
                        Path: relPath,
                        Field: fieldName,
                    })
                }
            }
            // Check enum narrowing
            if parentField.Type == "enum" && childField.Type == "enum" {
                if !isSubset(childField.Values, parentField.Values) {
                    checks = append(checks, StemHealthCheck{
                        Name: "monotonic-enum-narrowing",
                        Status: "fail",
                        Message: fmt.Sprintf("field %q: enum not a subset of parent %v", fieldName, parentField.Values),
                        Path: relPath,
                        Field: fieldName,
                    })
                }
            }
            // Check required loosening
            if parentField.Required && !childField.Required {
                checks = append(checks, StemHealthCheck{
                    Name: "monotonic-required-loosening",
                    Status: "fail",
                    Message: fmt.Sprintf("field %q: cannot loosen required constraint", fieldName),
                    Path: relPath,
                    Field: fieldName,
                })
            }
            // ... more checks
        }
    }
}
```

### Implementation Checklist (Out of scope for T001, informs roadmap)

- [ ] Add `semantics` field to `StemFile` struct
- [ ] Parse `semantics: monotonic` in `ParseStem`
- [ ] Update `MergeStemFiles` to check semantics flag and enforce narrowing-only
- [ ] Add validation checks in `stemhealth.go` (check 11)
- [ ] Helper functions: `isValidNarrowing(parentType, childType)`, `isSubset(child, parent)`, `isMonotonicMatch(parent, child)`
- [ ] Update merge tests: 10+ test cases per category
- [ ] Migration command: `rootline migrate --to-monotonic`
- [ ] `rootline analyze` detector: flag non-monotonic schemas
- [ ] CLI flag: `rootline validate --strict` enforces monotonic checks even without flag
- [ ] Documentation: schema design guide, monotonic best practices
- [ ] Template updates: `rootline init --template` includes `semantics: monotonic`
