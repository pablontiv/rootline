---
estado: Specified
---

# Type Representation Repair Design

**Date:** 2026-08-24
**Status:** Approved design
**Issue:** #196

## Purpose

Give `fix` a safe data-migration path for YAML scalar representations that violate a governed `type: string` field, while preserving the exact authored scalar text and reporting every unsupported `rule: type` error instead of silently returning an empty proposal set.

Strict field contracts correctly reject an unquoted YAML date, boolean, or integer when the schema requires a string. The current proposal engine ignores the entire `rule: type` surface, so `validate` can fail while `fix --all --dry-run` emits zero proposals and exits successfully. On the corpus that surfaced #196, this leaves 1,069 unquoted dates without an automated migration path.

## Decisions

1. Automatic representation repair applies only when the effective field contract expects `string` and the source value is a YAML timestamp, boolean, or integer scalar.
2. The repair preserves the scalar's exact YAML lexeme. It does not reconstruct text from `time.Time`, `bool`, or integer values.
3. Mapping, sequence, null, number, unknown representations, and inverse conversions are reported as non-repairable type findings and are never coerced.
4. Type repairs reuse the existing `correct_value` proposal type.
5. `Proposal` gains an optional `from_representation` discriminator. No sentinel value and no new proposal type are introduced.
6. Type-aware matching runs only when `from_representation` is populated. Existing reports retain their current strict string comparison.
7. Extraction owns YAML syntax evidence, proposal analysis owns repair classification, and fix owns guarded mutation.
8. Structured expected and actual representations travel internally on `ValidationError` through fields excluded from JSON. Human-readable error messages are never parsed by the repair path.
9. Unsupported type findings are visible in dry-run and applied output but do not change `fix` exit semantics. `validate` remains the authority on corpus validity.
10. Existing version 1 envelopes receive additive `omitempty` fields; no envelope version is incremented.

## Scope

### In scope

- Exact frontmatter scalar lexeme and representation capture.
- Internal structured type evidence on validation errors.
- Safe timestamp, boolean, and integer scalar-to-string proposals.
- Type-aware, fail-closed guards for stored reports.
- Automatic application through normal `fix --all`.
- Non-repairable type findings in JSON and table output.
- Backward compatibility for proposal reports without `from_representation`.
- Unit, contract, CLI, and end-to-end regression coverage.
- Living documentation and agent-skill synchronization required by repository gates.

### Out of scope

- Mapping or sequence serialization into strings.
- Null replacement or missing-value invention.
- Floating-point lexical normalization.
- String-to-boolean, string-to-integer, or any other inverse coercion.
- A new proposal type or summary counter.
- Parsing `ValidationError.Message`.
- Changing validation type semantics.
- Making type findings fail `fix` or redefining exit 0 as corpus validity.
- Writing to or migrating any external consumer repository.

## Architecture

### Exact scalar metadata in extraction

`extract.Record` gains a non-serialized map containing syntax evidence for frontmatter scalar fields:

```go
type FrontmatterScalar struct {
    Lexeme         string
    Representation string
}

type Record struct {
    // Existing fields omitted.
    FrontmatterScalars map[string]FrontmatterScalar `json:"-"`
}
```

The Markdown extractor parses the leading frontmatter into a `yaml.Node`, decodes the existing `Frontmatter` map from that same document, and walks the top-level mapping to capture supported scalar metadata. The canonical representation mapping is:

| YAML tag | Representation |
|---|---|
| `!!timestamp` | `timestamp` |
| `!!bool` | `boolean` |
| `!!int` | `integer` |

`yaml.Node.Value` is the authoritative lexeme. It preserves distinctions such as `2026-06-22` versus `2026-06-22T00:00:00Z`, `TRUE` versus `true`, and `042` versus `42`.

Quoted values have YAML tag `!!str` and require no representation repair. Nested mappings and sequences are not flattened into this map. If normal YAML decoding fails and extraction uses its malformed-frontmatter fallback, no scalar repair metadata is published; incomplete evidence cannot authorize an automatic write.

### Structured internal validation evidence

`rules.ValidationError` gains two machine-only fields:

```go
ExpectedRepresentation string `json:"-"`
ActualRepresentation   string `json:"-"`
```

`validationErrorFromContract` populates them from `FieldContractIssue.Expected` and `FieldContractIssue.Actual`. They remain absent from the public validate envelope.

This avoids two invalid dependencies:

- parsing human-readable error prose; and
- consulting the single "richest" effective stem passed to `proposal.Analyze`, which may not be the stem that governed a record under `scope.match`.

The detector trusts the per-record validation result and cross-checks its actual representation against the record's extracted scalar metadata.

### Proposal contract

`proposal.Proposal` gains:

```go
// FromRepresentation names the YAML representation of the value the repair
// expects to find on disk, when that value is not a string. It is set only on
// type-mismatch repairs whose exact scalar lexeme is carried in From.
FromRepresentation string `json:"from_representation,omitempty"`
```

A safe type repair uses the existing `CorrectValue` type:

```json
{
  "type": "correct_value",
  "field": "ingested",
  "paths": ["notes/a.md"],
  "from": "2026-06-22",
  "to": "2026-06-22",
  "from_representation": "timestamp"
}
```

`From` is always the real lexeme, never a placeholder such as `<timestamp>`. `To` is the same plain string value, never pre-quoted. `RewriteFrontmatter` is responsible for encoding it as a YAML string.

Reusing `CorrectValue` preserves proposal taxonomy, surface classification, filtering, summary counters, and repair command handling. The representation field explains why identical textual `from` and `to` values still describe a real change.

### Type repair detector

The detector processes every `rule: type` validation error and classifies it exactly once.

A repair proposal is emitted only when all conditions hold:

1. `ExpectedRepresentation == "string"`;
2. `ActualRepresentation` is `timestamp`, `boolean`, or `integer`;
3. the record and field have scalar metadata;
4. metadata representation equals `ActualRepresentation`; and
5. metadata lexeme is non-empty where the YAML representation requires content.

Every other type error becomes a `TypeFinding` rather than disappearing.

The detector must not inspect `ValidationError.Message`, infer expected type from a global stem, or synthesize a value from the parsed Go scalar.

### Non-repairable findings

Proposal and applied-fix envelopes gain an optional collection:

```go
type TypeFinding struct {
    Path                 string `json:"path"`
    Field                string `json:"field"`
    Message              string `json:"message"`
    ActualRepresentation string `json:"actual_representation,omitempty"`
}
```

`Report.TypeFindings` and `BatchFixResult.TypeFindings` use `json:"type_findings,omitempty"`. Table output labels them as reported but not repaired, matching the established `link_findings` presentation pattern.

Findings do not set `complete: false` and do not cause a non-zero exit. A successful `fix` exit means the requested repair operation completed, not that all records satisfy every validation rule. Callers that need a validity verdict run `validate`.

### Guarded application

The in-process `fix --all` path applies generated safe proposals through the existing `CorrectValue` handler. It replaces the parsed scalar with the exact lexeme as a Go string, after which `RewriteFrontmatter` emits an explicitly quoted `!!str` node.

`repair apply` treats report files as untrusted and re-extracts every target. Its guard branches as follows:

- `from_representation` absent: retain the existing `reflect.DeepEqual(current, p.From)` behavior unchanged;
- `from_representation` present and supported: require exact equality of both current scalar lexeme and representation;
- marker unknown, metadata absent, field absent, lexeme changed, or representation changed: reject the proposal without writing.

The guard never falls back from a failed type-aware comparison to the legacy comparison. This is fail-closed behavior.

After a successful write, the existing per-file post-validation and rollback contract for `repair apply` remains authoritative.

## Data Flow

```text
Markdown bytes
  -> locate leading frontmatter
  -> parse one yaml.Node document
  -> decode Frontmatter map
  -> capture exact top-level scalar lexemes and representations
  -> resolve the record's effective schema
  -> validate field representation
       -> public ValidationError fields
       -> internal ExpectedRepresentation / ActualRepresentation
  -> proposal detector cross-checks error + scalar metadata
       safe scalar -> CorrectValue + FromRepresentation
       otherwise   -> TypeFinding
  -> dry-run emits proposals and findings
  -> normal fix replaces parsed scalar with exact lexeme string
  -> RewriteFrontmatter emits quoted !!str
  -> subsequent extraction sees representation string
  -> subsequent fix emits no proposal
```

For a stored report, `repair apply` repeats extraction before the write and compares the current lexeme and representation against the report. The generated report therefore cannot overwrite a semantically or textually changed field.

## Compatibility

- `ValidationError` JSON is byte-shape compatible because its new evidence fields use `json:"-"`.
- `extract.Record` JSON is unchanged because scalar metadata uses `json:"-"`.
- `Proposal.from_representation` is optional and additive.
- Old proposal reports omit the field and retain strict legacy matching.
- `rootline/proposals` and `rootline/fix-batch` remain version 1 with additive `type_findings` fields.
- `correct_value` remains a repair-surface proposal; `Surface()` does not change.
- Summary counts include safe type repairs under the existing `correct_value` counter.
- The visible `from` and `to` strings may be equal; descriptions and table output must name the representation change clearly.

## Error Handling

- YAML node decoding failure follows the existing extraction error/fallback path and cannot authorize a repair.
- Unsupported or incomplete representation evidence becomes a finding.
- Unknown `from_representation` values in external reports are rejected.
- Changed target fields are rejected, even if the new parsed Go value is semantically equivalent.
- Filesystem read, containment, atomic write, and post-validation failures retain existing repair classifications.
- A safe proposal that cannot be applied does not silently become a finding during `repair apply`; it is explicitly rejected because the stored report no longer matches disk.

## Testing Strategy

### Extraction

- Preserve exact lexemes for a date, full timestamp, `TRUE`, `042`, and `+42`.
- Distinguish quoted strings from native scalar representations.
- Preserve current frontmatter map decoding and body extraction behavior.
- Publish no scalar metadata from malformed-YAML fallback extraction.
- Cover LF and CRLF frontmatter.

### Validation

- Populate internal expected and actual representations for type mismatch errors.
- Leave unrelated validation errors without misleading type evidence.
- Assert JSON serialization does not expose the internal fields.
- Cover fields governed by different `scope.match` rules.

### Proposal analysis

- Generate safe `CorrectValue` proposals for timestamp, boolean, and integer to string.
- Assert `From`, `To`, and `FromRepresentation` exactly.
- Prove detector behavior is independent of human error-message wording.
- Emit findings for mapping, sequence, null, number, unknown metadata, and inverse conversions.
- Assert every `rule: type` error becomes one proposal or one finding, never both or neither.
- Retain existing detector priority and summary behavior.

### Fix and repair

- `fix --all --dry-run` emits safe proposals and non-repairable findings without writes.
- Normal `fix --all` quotes each safe lexeme exactly once.
- Comments, key order, file mode, and body remain preserved.
- A second execution emits no proposal and performs no write.
- `repair apply` accepts exact lexeme and representation matches.
- It rejects changed lexeme, changed representation, unknown marker, missing metadata, and missing fields.
- Existing reports without `from_representation` retain strict matching.
- Post-validation rollback remains active.

### CLI and contracts

- JSON and table output cover empty and populated `type_findings`.
- Findings appear in both dry-run and applied runs.
- Findings do not alter `fix` exit status or completeness.
- Proposal JSON round-trips `from_representation` and omits it when empty.
- A compiled-binary fixture reproduces the original failure before the change and validates cleanly after safe repair.

### Repository gates

```bash
go test ./internal/extract ./internal/rules ./internal/proposal ./internal/fix ./cmd/rootline -race
just check
just test
just coverage-check
```

Documentation, agent skill, and governed records are validated with the repository's normal Rootline checks. Verification uses synthetic fixtures outside the repository and never writes to a consumer corpus.

## Acceptance Criteria

1. An unquoted YAML date in a governed string field produces a safe `correct_value` proposal instead of an empty report.
2. Native YAML timestamp, boolean, and integer scalars are rewritten as strings with their exact original lexemes.
3. Quoting is performed by the YAML writer and never stored as embedded quote characters.
4. A second `fix` run is a no-op.
5. A stored report cannot overwrite a field whose lexeme or representation changed after report generation.
6. Reports without `from_representation` behave exactly as before.
7. Every unsupported `rule: type` error appears in `type_findings`.
8. Type findings do not change the existing `fix` completion or exit contract.
9. Detection does not parse human error text or consult an unrelated global effective stem.
10. Public validate and record JSON contracts do not expose internal syntax evidence.
11. All focused, repository, race, lint, and coverage gates pass.
12. Verification evidence comes from a compiled Rootline binary and isolated fixtures, not from an external consumer repository.

## Delivery Boundary

This design is one implementation plan but should be delivered as reviewable commits on `fix/type-representation-repair`:

1. extraction metadata and internal validation evidence;
2. detector, proposal contract, and non-repairable findings;
3. guarded fix/repair application and end-to-end tests;
4. CLI output, living documentation, and agent-skill synchronization.

No implementation slice may weaken existing `correct_value` guards or broaden coercion beyond the approved allowlist.