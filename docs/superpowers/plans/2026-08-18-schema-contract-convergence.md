---
estado: Specified
---

# Schema Contract Convergence Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Rootline field types, Markdown body sources, hierarchical compatibility, section materialization, and validation provenance one deterministic executable contract.

**Architecture:** `internal/rules` owns one field contract for declaration validation, source-aware value resolution, type conformance, and monotonic compatibility. `internal/extract` preserves section occurrences and resolves canonical body-source directives; producers and consumers share those interfaces instead of branching on legacy `type: section` metadata. Validation keeps physical paths internally and normalizes public error sources per governing root.

**Tech Stack:** Go 1.26+, standard `testing`, Cobra CLI, `gopkg.in/yaml.v3`, Goldmark AST, Rootline `.stem` schemas.

**Spec:** `docs/superpowers/specs/2026-08-18-schema-contract-convergence-design.md`

## Global Constraints

- Strict TDD is mandatory: RED, GREEN, TRIANGULATE, REFACTOR for every task.
- Supported value types are exactly `string`, `list`, `enum`, `sequence`, `link`, `boolean`, and `integer`.
- `type: bool` migrates to canonical `type: boolean`; no boolean or integer coercion is allowed.
- Enums require one or more values; a one-value enum is a valid literal domain.
- `type: section`, `heading`, and `ordered` are rejected legacy declarations.
- Markdown location is expressed only through `source`; frontmatter remains the explicit override.
- Inherited source bindings cannot change or disappear.
- Duplicate matching headings and inferred logical-name collisions fail explicitly.
- Ancestor-qualified section selectors remain out of scope under #190.
- `required` means presence; `non_empty` and `exists` retain their separate semantics.
- No new CLI command, flag, schema mode, envelope version, dependency, or automatic migration is permitted.
- No implementation commit may exceed 400 authored lines without an explicit review-size decision.
- Keep the unrelated modified `.gitignore` untouched.

---

## File Responsibility Map

### New focused units

- `internal/extract/body_source.go` — canonical directive parsing and occurrence-preserving body resolution.
- `internal/extract/body_source_test.go` — directive, empty-presence, duplicate, and AST/text parity tests.
- `internal/rules/field_source.go` — frontmatter-first effective field resolution.
- `internal/rules/field_source_test.go` — source precedence and explicit presence tests.
- `internal/rules/type_contract.go` — supported declarations and value conformance.
- `internal/rules/type_contract_test.go` — strict type and enum-domain matrix.
- `internal/rules/field_compatibility.go` — monotonic type/source compatibility.
- `internal/rules/field_compatibility_test.go` — cumulative inheritance matrix.
- `internal/rules/section_materialization.go` — canonical required-section candidates.
- `internal/rules/section_materialization_test.go` — defaults, ambiguity, and lexical order.
- `internal/rules/source_normalization.go` — governance-relative public error paths.
- `internal/rules/source_normalization_test.go` — symbolic, escape, and platform path cases.

### Existing owners

- `internal/extract/{body.go,extract.go}` — AST section extraction and record representation.
- `internal/rules/{rules.go,merge.go,resolver.go,stemhealth.go,validate.go,describe.go,explain.go}` — schema model and all field-contract consumers.
- `internal/derive/enrich.go` — query-time source-backed enrichment.
- `internal/infer/{inference.go,report.go,body_sections.go,schema_gen.go,apply.go,delta.go,validation_gaps.go}` — canonical inference and report transport.
- `internal/migrate/{scaffold.go,split.go,diff.go}` — schema transport and document materialization.
- `cmd/rootline/{init.go,schema.go,new.go,migrate.go,query.go,set.go,describe.go,explain.go,validate.go}` — CLI convergence.
- Living docs and `.claude/skills/rootline/` — public and agent-facing contract.

## Delivery Topology

Execute tasks sequentially because later tasks consume interfaces from earlier tasks. Each task is one review slice and one conventional commit. Do not release partial slices: Tasks 1–10 form one compatibility-changing feature chain; Task 11 is the living-contract release boundary.

---

### Task 1: Make body-source resolution explicit

**Files:**
- Create: `internal/extract/body_source.go`
- Create: `internal/extract/body_source_test.go`
- Modify: `internal/extract/body.go`
- Modify: `internal/extract/extract.go`
- Create: `internal/rules/field_source.go`
- Create: `internal/rules/field_source_test.go`
- Modify: `internal/rules/validate.go`
- Modify: `internal/derive/enrich.go`

**Interfaces:**
- Produces:

```go
type BodySourceKind string

const (
    BodySourceH1      BodySourceKind = "h1"
    BodySourceSection BodySourceKind = "section"
)

type BodySource struct {
    Kind    BodySourceKind
    Heading string
}

func ParseBodySource(directive string) (BodySource, error)
func CanonicalSectionSource(exactHeading string) (string, error)
func ResolveBodyValue(record *Record, directive string) (string, bool, error)
```

- Produces for rules consumers:

```go
func ResolveFieldValue(record *extract.Record, name string, field SchemaField) (any, bool, error)
func ResolveEffectiveField(record *extract.Record, effective *StemFile, name string) (any, bool, error)
```

- `extract.Record` gains `BodySections []Section` and reuses the existing ordered `Section` representation as the occurrence-preserving source boundary. Existing `Sections` may remain for compatibility but MUST NOT drive source resolution.

- [ ] **Step 1: Write failing directive and occurrence tests**

```go
func TestResolveBodyValue_PresentEmptySection(t *testing.T) {
    rec := &Record{BodySections: []Section{{Heading: "Notes", Level: 2, Content: ""}}}
    got, present, err := ResolveBodyValue(rec, `body.section["## Notes"]`)
    if err != nil || !present || got != "" {
        t.Fatalf("got value=%q present=%v err=%v", got, present, err)
    }
}

func TestResolveBodyValue_DuplicateSectionIsAmbiguous(t *testing.T) {
    rec := &Record{BodySections: []Section{
        {Heading: "Notes", Level: 2, Content: "first"},
        {Heading: "Notes", Level: 2, Content: "second"},
    }}
    _, _, err := ResolveBodyValue(rec, `body.section["## Notes"]`)
    if err == nil || !strings.Contains(err.Error(), "ambiguous") {
        t.Fatalf("expected ambiguity error, got %v", err)
    }
}
```

- [ ] **Step 2: Run the focused tests and verify RED**

Run:

```bash
go test ./internal/extract ./internal/rules ./internal/derive -run 'Test(ParseBodySource|ResolveBodyValue|ResolveFieldValue|MarkdownExtractor_PreservesDuplicate)' -count=1
```

Expected: compile failures for the new signatures/types and assertion failures for empty/duplicate sections.

- [ ] **Step 3: Implement canonical parsing and resolution**

```go
func ResolveBodyValue(record *Record, directive string) (string, bool, error) {
    source, err := ParseBodySource(directive)
    if err != nil {
        return "", false, err
    }
    switch source.Kind {
    case BodySourceH1:
        return resolveH1(record)
    case BodySourceSection:
        return resolveUniqueSection(record.BodySections, source.Heading)
    default:
        return "", false, fmt.Errorf("unsupported body source %q", directive)
    }
}
```

Implement `ResolveFieldValue` as frontmatter first, then body source. Implement `ResolveEffectiveField` so explicit rules and query enrichment use the schema field contract rather than bypassing it through `Record.EffectiveField`.

- [ ] **Step 4: Run focused tests and verify GREEN**

```bash
go test ./internal/extract ./internal/rules ./internal/derive -run 'Test(ParseBodySource|ResolveBodyValue|ResolveFieldValue|MarkdownExtractor_PreservesDuplicate)' -count=1
```

Expected: PASS.

- [ ] **Step 5: Triangulate AST and text-backed extraction**

Add one fixture with a fenced-code fake heading, an empty real heading, and duplicate real headings. Assert both extraction paths return the same occurrences and ambiguity.

Run:

```bash
go test ./internal/extract ./internal/rules ./internal/derive -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit the slice**

```bash
git add internal/extract internal/rules/field_source.go internal/rules/field_source_test.go internal/rules/validate.go internal/derive/enrich.go
git commit -m "fix(extract): make body source resolution explicit"
```

---

### Task 2: Enforce one field declaration and value-type vocabulary

**Files:**
- Create: `internal/rules/type_contract.go`
- Create: `internal/rules/type_contract_test.go`
- Modify: `internal/rules/rules.go`
- Modify: `internal/rules/stemhealth.go`
- Modify: `internal/infer/validation_gaps.go`

**Interfaces:**

```go
type FieldContractIssue struct {
    Code     string
    Expected string
    Actual   string
    Message  string
}

func IsSupportedFieldType(typeName string) bool
func ValidateFieldDeclaration(name string, field SchemaField) []FieldContractIssue
func ValidateFieldValue(field SchemaField, value any) *FieldContractIssue
```

Supported names: `string`, `list`, `enum`, `sequence`, `link`, `boolean`, `integer`.

`SchemaField.UnmarshalYAML` preserves private declaration metadata needed to distinguish omitted, explicit empty, null, and legacy keys. Do not expose this metadata in JSON.

- [ ] **Step 1: Write the failing declaration matrix**

```go
func TestValidateFieldDeclaration(t *testing.T) {
    tests := []struct {
        name string
        field SchemaField
        wantCode string
    }{
        {"single enum", SchemaField{Type: "enum", Values: []string{"theory"}}, ""},
        {"empty enum", SchemaField{Type: "enum"}, "incomplete-type"},
        {"boolean", SchemaField{Type: "boolean"}, ""},
        {"legacy bool", SchemaField{Type: "bool"}, "legacy-type"},
        {"integer", SchemaField{Type: "integer"}, ""},
        {"unknown", SchemaField{Type: "number"}, "unknown-type"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            issues := ValidateFieldDeclaration("field", tt.field)
            got := ""
            if len(issues) > 0 { got = issues[0].Code }
            if got != tt.wantCode { t.Fatalf("got %q want %q issues=%+v", got, tt.wantCode, issues) }
        })
    }
}
```

Add YAML parsing tests proving `type: section`, `heading`, `ordered`, and `type: bool` retain enough private metadata to print exact migrations. Use this test helper in `type_contract_test.go`:

```go
func mustParseField(t *testing.T, body string) SchemaField {
    t.Helper()
    yamlText := "version: 2\nschema:\n  field:\n" + strings.ReplaceAll(body, "\n", "\n    ")
    stem, err := ParseStem(".stem", []byte(yamlText))
    if err != nil { t.Fatal(err) }
    return stem.Schema["field"]
}
```

- [ ] **Step 2: Verify RED**

```bash
go test ./internal/rules ./internal/infer -run 'Test(ValidateFieldDeclaration|ValidateFieldValue|ValidateStemHealth_Rejects|DetectValidationGaps_Legacy)' -count=1
```

Expected: missing symbols and current stem-health acceptance/failure mismatches.

- [ ] **Step 3: Implement declaration and value contracts**

```go
var supportedFieldTypes = map[string]struct{}{
    "string": {}, "list": {}, "enum": {}, "sequence": {},
    "link": {}, "boolean": {}, "integer": {},
}

func ValidateFieldValue(field SchemaField, value any) *FieldContractIssue {
    if issue := validateRepresentation(field.Type, value); issue != nil {
        return issue
    }
    return validateTypeSemantics(field, value)
}
```

Use stable YAML representation names: `string`, `sequence`, `mapping`, `boolean`, `integer`, `number`, and `null`. `boolean` accepts only Go/YAML `bool`; `integer` accepts integer decoder forms and rejects strings/floats. Enum accepts one or more values.

- [ ] **Step 4: Route declaration failures through stem health**

Reject unknown/incomplete types, unsupported sources, legacy `section` metadata, and `bool`. The legacy diagnostic MUST print:

```yaml
type: boolean
```

or:

```yaml
type: string
source: body.section["## Notes"]
```

as applicable. Remove the special `type: section` skip from validation-gap analysis.

- [ ] **Step 5: Verify GREEN and triangulate no coercion**

```bash
go test ./internal/rules ./internal/infer -run 'Test(ValidateFieldDeclaration|ValidateFieldValue|ValidateStemHealth_Rejects|DetectValidationGaps_Legacy)' -count=1
go test ./internal/rules ./internal/infer -count=1
```

Expected: PASS; strings `"true"` and `"3"` fail boolean/integer conformance.

- [ ] **Step 6: Commit the slice**

```bash
git add internal/rules/type_contract.go internal/rules/type_contract_test.go internal/rules/rules.go internal/rules/stemhealth.go internal/infer/validation_gaps.go
git commit -m "feat(rules): enforce canonical field declarations"
```

---

### Task 3: Integrate strict conformance without collapsing presence rules

**Files:**
- Modify: `internal/rules/validate.go`
- Modify: `internal/rules/validate_test.go`
- Modify: `internal/rules/type_contract.go`
- Test: `internal/rules/type_contract_test.go`

**Interfaces consumed:** `ResolveFieldValue`, `ResolveEffectiveField`, `ValidateFieldValue`.

- [ ] **Step 1: Write RED validation precedence tests**

```go
func TestValidate_PresenceRulesStayIndependent(t *testing.T) {
    rec := &extract.Record{Frontmatter: map[string]any{"title": ""}}
    stem := &StemFile{Schema: map[string]SchemaField{
        "title": {Type: "string", Required: true, Severity: "error"},
    }}
    if got := Validate(context.Background(), rec, stem); len(got) != 0 {
        t.Fatalf("required must accept present empty string: %+v", got)
    }
}

func TestValidate_TypeMismatchStopsSemanticChecks(t *testing.T) {
    rec := &extract.Record{Frontmatter: map[string]any{"status": []any{"draft"}}}
    stem := &StemFile{Schema: map[string]SchemaField{
        "status": {Type: "enum", Values: []string{"draft", "done"}, Severity: "error"},
    }}
    got := Validate(context.Background(), rec, stem)
    if len(got) != 1 || got[0].Rule != "type" || got[0].Suggestion != "" {
        t.Fatalf("expected one type error without enum suggestion, got %+v", got)
    }
}
```

Use this explicit representation table in `TestValidate_TypeConformanceMatrix`:

```go
tests := []struct {
    name string
    field SchemaField
    value any
    valid bool
}{
    {"string", SchemaField{Type: "string"}, "text", true},
    {"string rejects integer", SchemaField{Type: "string"}, 3, false},
    {"list", SchemaField{Type: "list"}, []any{"a"}, true},
    {"single enum", SchemaField{Type: "enum", Values: []string{"theory"}}, "theory", true},
    {"single enum rejects other", SchemaField{Type: "enum", Values: []string{"theory"}}, "hypothesis", false},
    {"sequence", SchemaField{Type: "sequence", Prefix: "T", Digits: 3}, "T007", true},
    {"link", SchemaField{Type: "link"}, "[[target]]", true},
    {"boolean", SchemaField{Type: "boolean"}, true, true},
    {"boolean rejects string", SchemaField{Type: "boolean"}, "true", false},
    {"integer", SchemaField{Type: "integer"}, 3, true},
    {"integer rejects float", SchemaField{Type: "integer"}, 3.5, false},
}
```

- [ ] **Step 2: Verify RED**

```bash
go test ./internal/rules -run 'TestValidate_(PresenceRulesStayIndependent|TypeConformanceMatrix|TypeMismatchStopsSemanticChecks|BodySourceUsesSameTypeContract)' -count=1
```

Expected: current validation accepts representation mismatches and may emit derivative errors.

- [ ] **Step 3: Implement ordered validation phases**

```go
value, present, resolveErr := ResolveFieldValue(record, name, field)
if resolveErr != nil {
    errs = append(errs, sourceResolutionError(name, field, resolveErr))
    continue
}
if field.Required && !present {
    errs = append(errs, requiredError(name, field, record))
    continue
}
if !present {
    continue
}
if issue := ValidateFieldValue(field, value); issue != nil {
    errs = append(errs, validationErrorFromContract(name, field, issue))
    continue
}
```

Route `non_empty`, `exists`, `requires`, and explicit enum rules through `ResolveEffectiveField`. Keep exact-empty-string behavior unchanged.

- [ ] **Step 4: Verify GREEN**

```bash
go test ./internal/rules -run 'TestValidate_(PresenceRulesStayIndependent|TypeConformanceMatrix|TypeMismatchStopsSemanticChecks|BodySourceUsesSameTypeContract)' -count=1
go test ./internal/rules -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit the slice**

```bash
git add internal/rules/validate.go internal/rules/validate_test.go internal/rules/type_contract.go internal/rules/type_contract_test.go
git commit -m "feat(rules): enforce declared field value types"
```

---

### Task 4: Make hierarchical compatibility universally monotonic

**Files:**
- Create: `internal/rules/field_compatibility.go`
- Create: `internal/rules/field_compatibility_test.go`
- Modify: `internal/rules/merge.go`
- Modify: `internal/rules/merge_test.go`
- Modify: `internal/rules/resolver.go`
- Modify: `internal/rules/resolver_test.go`
- Modify: `internal/rules/stemhealth.go`
- Modify: `internal/rules/stemhealth_test.go`
- Modify: `cmd/rootline/validate_test.go`

**Interfaces:**

```go
type FieldCompatibilityIssue struct {
    Constraint string
    Operation  string
    Value      any
    Message    string
}

func CheckFieldCompatibility(parent, child SchemaField) []FieldCompatibilityIssue
func ResolveLayered(path, root string) (*LayeredResolution, error)
```

- [ ] **Step 1: Write RED compatibility tests**

```go
func TestFieldCompatibility_SourceMatrix(t *testing.T) {
    parent := SchemaField{Type: "string", Extract: `body.section["## Summary"]`}
    tests := []struct{name string; child SchemaField; wantIssue bool}{
        {"omitted inherits", SchemaField{Type: "string"}, false},
        {"same source", SchemaField{Type: "string", Extract: parent.Extract}, false},
        {"changed", SchemaField{Type: "string", Extract: `body.section["## Context"]`}, true},
        {"removed", mustParseField(t, "type: string\nsource: null\n"), true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := CheckFieldCompatibility(parent, tt.child)
            if (len(got) > 0) != tt.wantIssue {
                t.Fatalf("issues=%+v wantIssue=%v", got, tt.wantIssue)
            }
        })
    }
}
```

Add cumulative three-layer coverage: root source A, middle omits, leaf changes to B must fail.

- [ ] **Step 2: Verify RED**

```bash
go test ./internal/rules ./cmd/rootline -run 'Test(FieldCompatibility|MergeSchemaFields_InheritsOmittedSource|ResolveLayered_|ValidateStemHealth_StringToEnum|ValidateStemHealth_EnumToString)' -count=1
```

Expected: non-monotonic mode still exists, adjacent-only checks miss cumulative conflicts, and stem health rejects valid narrowing.

- [ ] **Step 3: Implement cumulative compatibility**

Merge omitted child sources from the cumulative parent. Remove the `monotonic bool` parameter. For each child layer, compare against the cumulative merged ancestor before applying the child.

```go
func ResolveLayered(path, root string) (*LayeredResolution, error) {
    base, err := Resolve(path, root)
    if err != nil { return nil, err }
    return resolveMonotonicLayers(base)
}
```

- [ ] **Step 4: Replace generic health checks**

Use `FieldCompatibilityIssue` for `type-consistency`, source conflicts, required loosening, enum extension, and severity loosening. Suppress `field-override` for valid narrowing.

- [ ] **Step 5: Verify GREEN**

```bash
go test ./internal/rules ./cmd/rootline -run 'Test(FieldCompatibility|MergeSchemaFields_InheritsOmittedSource|ResolveLayered_|ValidateStemHealth_StringToEnum|ValidateStemHealth_EnumToString)' -count=1
go test ./internal/rules ./cmd/rootline -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit the slice**

```bash
git add internal/rules/field_compatibility.go internal/rules/field_compatibility_test.go internal/rules/merge.go internal/rules/merge_test.go internal/rules/resolver.go internal/rules/resolver_test.go internal/rules/stemhealth.go internal/rules/stemhealth_test.go cmd/rootline/validate_test.go
git commit -m "fix(rules): converge monotonic field compatibility"
```

---

### Task 5: Emit exact canonical section inference

**Files:**
- Modify: `internal/infer/inference.go`
- Modify: `internal/infer/body_sections.go`
- Modify: `internal/infer/body_sections_test.go`
- Modify: `internal/infer/schema_gen.go`
- Modify: `internal/infer/schema_gen_test.go`
- Modify: `cmd/rootline/init.go`
- Modify: `cmd/rootline/init_test.go`

**Interfaces:**

```go
type Inference struct {
    Type            string `json:"type"`
    Source          string `json:"source"`
    Field           string `json:"field,omitempty"`
    Value           string `json:"value,omitempty"`
    Message         string `json:"message"`
    SourceDirective string `json:"source_directive,omitempty"`
}

func DetectSectionPatterns(records []*extract.Record, threshold float64) ([]Inference, error)
```

- [ ] **Step 1: Write RED inference tests**

```go
func TestDetectSectionPatterns_ThresholdCandidateOptionalUntilUniversal(t *testing.T) {
    records := []*extract.Record{
        makeRecord("## Notes\nA\n"), makeRecord("## Notes\nB\n"),
        makeRecord("## Notes\nC\n"), makeRecord("## Notes\nD\n"),
        makeRecord("# No notes\n"),
    }
    got, err := DetectSectionPatterns(records, 0.8)
    if err != nil { t.Fatal(err) }
    for _, inf := range got {
        if inf.Field == "notes" {
            if inf.Type != "optional_section" || inf.SourceDirective != `body.section["## Notes"]` {
                t.Fatalf("unexpected inference: %+v", inf)
            }
            return
        }
    }
    t.Fatalf("notes inference missing: %+v", got)
}

func TestDetectSectionPatterns_NameCollisionFails(t *testing.T) {
    records := []*extract.Record{makeRecord("## Notes\nA\n\n### Notes\nB\n")}
    _, err := DetectSectionPatterns(records, 0.5)
    if err == nil || !strings.Contains(err.Error(), "## Notes") || !strings.Contains(err.Error(), "### Notes") {
        t.Fatalf("expected both colliding headings, got %v", err)
    }
}
```

- [ ] **Step 2: Verify RED**

```bash
go test ./internal/infer ./cmd/rootline -run 'Test(DetectSectionPatterns_|Generate.*SectionSource|Init.*SectionSource|InitGeneratedSchema)' -count=1
```

Expected: current detector drops level, excludes empty records, marks threshold matches required, and silently collides.

- [ ] **Step 3: Implement exact inference**

Use every record as denominator. Count heading presence independently from content. Construct exact headings from `Level` plus `Heading`. Mark required only when `count == len(records)`.

```go
exact := strings.Repeat("#", section.Level) + " " + section.Heading
source, err := extract.CanonicalSectionSource(exact)
```

Before returning, group candidates by logical field name; if more than one exact source maps to one name, return an error listing every source.

- [ ] **Step 4: Serialize canonical fields in init**

Generate:

```yaml
notes:
  type: string
  source: body.section["## Notes"]
```

Never serialize `heading` or reconstruct every heading as H2.

- [ ] **Step 5: Verify GREEN and source-corpus convergence**

```bash
go test ./internal/infer ./cmd/rootline -run 'Test(DetectSectionPatterns_|Generate.*SectionSource|Init.*SectionSource|InitGeneratedSchema)' -count=1
go test ./internal/infer ./cmd/rootline -count=1
```

Expected: PASS; a four-of-five heading is optional and the generated schema validates all five inputs.

- [ ] **Step 6: Commit the slice**

```bash
git add internal/infer/inference.go internal/infer/body_sections.go internal/infer/body_sections_test.go internal/infer/schema_gen.go internal/infer/schema_gen_test.go cmd/rootline/init.go cmd/rootline/init_test.go
git commit -m "fix(infer): emit canonical optional section fields"
```

---

### Task 6: Preserve section sources through analyze and schema apply

**Files:**
- Modify: `internal/infer/report.go`
- Modify: `internal/infer/report_test.go`
- Modify: `internal/infer/apply.go`
- Modify: `internal/infer/apply_test.go`
- Modify: `internal/infer/delta.go`
- Modify: `internal/infer/delta_test.go`
- Modify: `cmd/rootline/analyze.go`
- Modify: `cmd/rootline/schema.go`
- Modify: `cmd/rootline/schema_test.go`
- Modify: `internal/e2e/schema_apply_e2e_test.go`

**Interfaces consumed:** `Inference.SourceDirective` from Task 5.

- [ ] **Step 1: Write RED report and apply tests**

```go
func TestApplySchemaInferences_AddsCanonicalSectionField(t *testing.T) {
    inf := Inference{Type: "required_section", Field: "notes", SourceDirective: `body.section["## Notes"]`}
    stem := &rules.StemFile{Schema: map[string]rules.SchemaField{}}
    got, err := ApplySchemaInferences(stem, []Inference{inf})
    if err != nil { t.Fatal(err) }
    field := got.Schema["notes"]
    if field.Type != "string" || field.Extract != inf.SourceDirective || !field.Required {
        t.Fatalf("unexpected field: %+v", field)
    }
}
```

Add these exact assertions:

```go
func TestAnalyzeReport_SortsBySourceDirective(t *testing.T) {
    report := NewAnalyzeReport([]Inference{
        {Type: "optional_section", Field: "notes", SourceDirective: `body.section["### Notes"]`},
        {Type: "optional_section", Field: "notes", SourceDirective: `body.section["## Notes"]`},
    })
    if report.Inferences[0].SourceDirective != `body.section["## Notes"]` {
        t.Fatalf("non-deterministic report: %+v", report.Inferences)
    }
}
```

In `TestApplySchemaInferences_RejectsConflictingSectionSource`, start with `notes` bound to `## Notes`, apply an inference bound to `## Context`, and assert a non-nil conflict error. In the command-level test, run schema propose, apply its report, then validate the original corpus and assert `summary.invalid == 0`.

- [ ] **Step 2: Verify RED**

```bash
go test ./internal/infer ./internal/e2e ./cmd/rootline -run 'Test(AnalyzeReport_PreservesSection|ApplySchemaInferences_.*Section|IsCovered_Section|SchemaPropose.*Section)' -count=1
```

Expected: section inference is discarded or loses source metadata.

- [ ] **Step 3: Implement transport**

Preserve `SourceDirective` in report DTOs and sorting. Route `required_section` and `optional_section` into schema apply; optional sets `Required=false`. Reject a conflicting existing source instead of replacing it.

- [ ] **Step 4: Verify GREEN**

```bash
go test ./internal/infer ./internal/e2e ./cmd/rootline -run 'Test(AnalyzeReport_PreservesSection|ApplySchemaInferences_.*Section|IsCovered_Section|SchemaPropose.*Section)' -count=1
go test ./internal/infer ./internal/e2e ./cmd/rootline -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit the slice**

```bash
git add internal/infer/report.go internal/infer/report_test.go internal/infer/apply.go internal/infer/apply_test.go internal/infer/delta.go internal/infer/delta_test.go cmd/rootline/analyze.go cmd/rootline/schema.go cmd/rootline/schema_test.go internal/e2e/schema_apply_e2e_test.go
git commit -m "fix(schema): preserve section source in propose and apply"
```

---

### Task 7: Preserve source bindings through schema transport

**Files:**
- Modify: `internal/migrate/split.go`
- Modify: `internal/migrate/split_test.go`
- Modify: `internal/migrate/diff.go`
- Modify: `internal/migrate/diff_test.go`
- Modify: `cmd/rootline/init.go`

- [ ] **Step 1: Write RED split and diff tests**

```go
// Add this field and assertion to the existing BasicTwoLevels fixture.
existing.Schema["notes"] = rules.SchemaField{Type: "string", Extract: `body.section["## Notes"]`}
result := BuildSplitStems("/tmp/test", existing, hierarchy)
if !strings.Contains(result.Stems[0].Content, `source: body.section["## Notes"]`) {
    t.Fatalf("source lost from root stem:\n%s", result.Stems[0].Content)
}

func TestDiff_SourceChangedIsBreaking(t *testing.T) {
    before := &rules.StemFile{Schema: map[string]rules.SchemaField{
        "notes": {Type: "string", Extract: `body.section["## Notes"]`},
    }}
    after := &rules.StemFile{Schema: map[string]rules.SchemaField{
        "notes": {Type: "string", Extract: `body.section["## Context"]`},
    }}
    got := Diff(".stem", before, after)
    if got.BreakingCount != 1 || got.Changes[0].Kind != SourceChanged || !got.Changes[0].Breaking {
        t.Fatalf("unexpected diff: %+v", got)
    }
}
```

- [ ] **Step 2: Verify RED**

```bash
go test ./internal/migrate ./cmd/rootline -run 'Test(BuildSplitStems_PreservesFieldSource|Diff_SourceChanged|StemSerializer_PreservesSource)' -count=1
```

Expected: source is omitted and diff ignores the binding change.

- [ ] **Step 3: Implement one schema-field serializer and source-aware diff**

Use one private serializer for init/split output. Serialize logical `source` and never legacy `heading`/`ordered`. Classify changed or removed source as breaking.

- [ ] **Step 4: Verify GREEN**

```bash
go test ./internal/migrate ./cmd/rootline -run 'Test(BuildSplitStems_PreservesFieldSource|Diff_SourceChanged|StemSerializer_PreservesSource)' -count=1
go test ./internal/migrate ./cmd/rootline -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit the slice**

```bash
git add internal/migrate/split.go internal/migrate/split_test.go internal/migrate/diff.go internal/migrate/diff_test.go cmd/rootline/init.go
git commit -m "fix(migrate): preserve source bindings across schema transport"
```

---

### Task 8: Converge new and scaffold on one materialization contract

**Files:**
- Create: `internal/rules/section_materialization.go`
- Create: `internal/rules/section_materialization_test.go`
- Modify: `cmd/rootline/new.go`
- Modify: `cmd/rootline/new_test.go`
- Create: `cmd/rootline/new_source_test.go`
- Modify: `internal/migrate/scaffold.go`
- Modify: `internal/migrate/scaffold_test.go`
- Modify: `cmd/rootline/migrate.go`
- Modify: `cmd/rootline/migrate_scaffold_test.go`

**Interfaces:**

```go
type SectionMaterialization struct {
    Field   string
    Heading string
    Content string
}

func RequiredSectionMaterializations(record *extract.Record, effective *StemFile) ([]SectionMaterialization, error)
```

- [ ] **Step 1: Write RED shared materialization tests**

```go
func TestRequiredSectionMaterializations_LexicalOrder(t *testing.T) {
    stem := &StemFile{Schema: map[string]SchemaField{
        "zeta": {Type: "string", Required: true, Extract: `body.section["## Zeta"]`},
        "alpha": {Type: "string", Required: true, Extract: `body.section["## Alpha"]`},
    }}
    rec := &extract.Record{Frontmatter: map[string]any{}, BodySections: nil}
    got, err := RequiredSectionMaterializations(rec, stem)
    if err != nil { t.Fatal(err) }
    if len(got) != 2 || got[0].Heading != "## Alpha" || got[1].Heading != "## Zeta" {
        t.Fatalf("not lexical: %+v", got)
    }
}
```

Add this table beside the lexical-order test:

```go
tests := []struct {
    name string
    field SchemaField
    record *extract.Record
    wantCount int
    wantContent string
    wantErr bool
}{
    {"default", SchemaField{Type: "string", Required: true, Extract: `body.section["## Notes"]`, Default: "seed"}, emptyRecord(), 1, "seed", false},
    {"placeholder", SchemaField{Type: "string", Required: true, Extract: `body.section["## Notes"]`}, emptyRecord(), 1, "<!-- TODO -->", false},
    {"frontmatter override", SchemaField{Type: "string", Required: true, Extract: `body.section["## Notes"]`}, recordWithFrontmatter("notes", "override"), 0, "", false},
    {"empty section present", SchemaField{Type: "string", Required: true, Extract: `body.section["## Notes"]`}, recordWithSection("## Notes", ""), 0, "", false},
    {"optional", SchemaField{Type: "string", Required: false, Extract: `body.section["## Notes"]`}, emptyRecord(), 0, "", false},
    {"duplicate", SchemaField{Type: "string", Required: true, Extract: `body.section["## Notes"]`}, recordWithDuplicateSection("## Notes"), 0, "", true},
}
```

Define these test-only constructors in `section_materialization_test.go`:

```go
func emptyRecord() *extract.Record {
    return &extract.Record{Frontmatter: map[string]any{}}
}

func recordWithFrontmatter(name string, value any) *extract.Record {
    return &extract.Record{Frontmatter: map[string]any{name: value}}
}

func recordWithSection(exactHeading, content string) *extract.Record {
    source, err := extract.ParseBodySource(`body.section["` + exactHeading + `"]`)
    if err != nil { panic(err) }
    return &extract.Record{Frontmatter: map[string]any{}, BodySections: []extract.Section{{Heading: strings.TrimLeft(source.Heading, "# "), Level: strings.Count(source.Heading, "#"), Content: content}}}
}

func recordWithDuplicateSection(exactHeading string) *extract.Record {
    rec := recordWithSection(exactHeading, "first")
    rec.BodySections = append(rec.BodySections, rec.BodySections[0])
    rec.BodySections[1].Content = "second"
    return rec
}
```

- [ ] **Step 2: Verify RED**

```bash
go test ./internal/rules ./internal/migrate ./cmd/rootline -run 'Test(RequiredSectionMaterializations|New.*SectionSource|ScaffoldRequiredSectionSource|MigrateScaffold)' -count=1
```

Expected: `new`/scaffold still branch on `type: section`, lexical order is absent, and scaffold swallows resolution failures.

- [ ] **Step 3: Implement shared candidates**

```go
func RequiredSectionMaterializations(record *extract.Record, effective *StemFile) ([]SectionMaterialization, error) {
    var out []SectionMaterialization
    for name, field := range effective.Schema {
        if issues := ValidateFieldDeclaration(name, field); len(issues) > 0 {
            return nil, fmt.Errorf("field %q: %s", name, issues[0].Message)
        }
        if !field.Required || field.Extract == "" {
            continue
        }
        source, err := extract.ParseBodySource(field.Extract)
        if err != nil { return nil, err }
        if source.Kind != extract.BodySourceSection { continue }
        _, present, err := ResolveFieldValue(record, name, field)
        if err != nil { return nil, err }
        if present { continue }
        content := field.Default
        if content == "" { content = "<!-- TODO -->" }
        out = append(out, SectionMaterialization{Field: name, Heading: source.Heading, Content: content})
    }
    sort.Slice(out, func(i, j int) bool { return out[i].Heading < out[j].Heading })
    return out, nil
}
```

Do not use map iteration as output order.

- [ ] **Step 4: Integrate `new` and scaffold**

`new` writes body sections without an empty frontmatter shadow and validates generated bytes before writing. Scaffold validates prospective bytes before the existing atomic write. Both propagate ambiguity, legacy declaration, and schema-resolution errors.

- [ ] **Step 5: Verify GREEN and complete loops**

```bash
go test ./internal/rules ./internal/migrate ./cmd/rootline -run 'Test(RequiredSectionMaterializations|New.*SectionSource|ScaffoldRequiredSectionSource|MigrateScaffold)' -count=1
go test ./internal/rules ./internal/migrate ./cmd/rootline -count=1
```

Expected: PASS, including `new → validate` and `scaffold → validate`.

- [ ] **Step 6: Commit the slice**

```bash
git add internal/rules/section_materialization.go internal/rules/section_materialization_test.go cmd/rootline/new.go cmd/rootline/new_test.go cmd/rootline/new_source_test.go internal/migrate/scaffold.go internal/migrate/scaffold_test.go cmd/rootline/migrate.go cmd/rootline/migrate_scaffold_test.go
git commit -m "fix(scaffold): materialize canonical required section sources"
```

---

### Task 9: Converge query, set, describe, and explain

**Files:**
- Modify: `internal/derive/enrich.go`
- Modify: `internal/derive/enrich_test.go`
- Modify: `cmd/rootline/query.go`
- Create: `cmd/rootline/query_source_test.go`
- Modify: `cmd/rootline/set.go`
- Modify: `cmd/rootline/set_test.go`
- Modify: `internal/rules/describe.go`
- Modify: `internal/rules/describe_test.go`
- Modify: `internal/rules/explain.go`
- Modify: `internal/rules/explain_test.go`
- Modify: `cmd/rootline/describe.go`
- Modify: `cmd/rootline/explain.go`
- Create: `cmd/rootline/inspection_source_test.go`

**Public output contract:** logical extraction uses JSON key `source`; physical `.stem` provenance uses `defined_in` or the existing top-level provenance map. Document this compatibility change in Task 11.

- [ ] **Step 1: Write RED runtime convergence tests**

```go
func TestQuerySourceBackedField_FrontmatterOverride(t *testing.T) {
    dir := setupTestDir(t)
    mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\nroot: true\nschema:\n  notes:\n    type: string\n    source: body.section[\"## Notes\"]\n"), 0o644)
    mustWriteFile(t, filepath.Join(dir, "doc.md"), []byte("---\nnotes: override\n---\n## Notes\nbody\n"), 0o644)
    out, err := runCmd(t, "query", "--from", dir, "--select", "notes")
    if err != nil { t.Fatal(err) }
    if !strings.Contains(out, `"notes":"override"`) { t.Fatalf("output=%s", out) }
}

func TestExplainCmd_ReportsSameSourceBackedValue(t *testing.T) {
    dir := setupTestDir(t)
    mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\nroot: true\nschema:\n  notes:\n    type: string\n    source: body.section[\"## Notes\"]\n"), 0o644)
    path := filepath.Join(dir, "doc.md")
    mustWriteFile(t, path, []byte("## Notes\nbody value\n"), 0o644)
    out, err := runCmd(t, "explain", path)
    if err != nil { t.Fatal(err) }
    if !strings.Contains(out, "body value") || !strings.Contains(out, `body.section`) {
        t.Fatalf("output=%s", out)
    }
}
```

Add these concrete command assertions:

```go
func TestQueryDuplicateSourceFails(t *testing.T) {
    dir := setupTestDir(t)
    mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\nroot: true\nschema:\n  notes:\n    type: string\n    source: body.section[\"## Notes\"]\n"), 0o644)
    mustWriteFile(t, filepath.Join(dir, "doc.md"), []byte("## Notes\nfirst\n\n## Notes\nsecond\n"), 0o644)
    _, err := runCmd(t, "query", "--from", dir, "--select", "notes")
    if err == nil || !strings.Contains(err.Error(), "ambiguous") {
        t.Fatalf("expected ambiguity error, got %v", err)
    }
}

func TestSetSourceBackedFieldWritesFrontmatterOverride(t *testing.T) {
    dir := setupTestDir(t)
    mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\nroot: true\nschema:\n  notes:\n    type: string\n    source: body.section[\"## Notes\"]\n"), 0o644)
    path := filepath.Join(dir, "doc.md")
    body := "## Notes\nbody value\n"
    mustWriteFile(t, path, []byte(body), 0o644)
    if _, err := runCmd(t, "set", path, "notes=override"); err != nil { t.Fatal(err) }
    data, err := os.ReadFile(path)
    if err != nil { t.Fatal(err) }
    if !strings.HasSuffix(string(data), body) || strings.Count(string(data), "## Notes") != 1 {
        t.Fatalf("body changed:\n%s", data)
    }
}

func TestDescribeCmd_SeparatesSourceAndDefinition(t *testing.T) {
    dir := setupTestDir(t)
    mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\nroot: true\nschema:\n  notes:\n    type: string\n    source: body.section[\"## Notes\"]\n"), 0o644)
    path := filepath.Join(dir, "doc.md")
    mustWriteFile(t, path, []byte("## Notes\nbody\n"), 0o644)
    out, err := runCmd(t, "describe", path)
    if err != nil { t.Fatal(err) }
    var payload map[string]any
    if err := json.Unmarshal([]byte(out), &payload); err != nil { t.Fatal(err) }
    schema := payload["schema"].(map[string]any)
    notes := schema["notes"].(map[string]any)
    if notes["source"] != `body.section["## Notes"]` || !strings.HasSuffix(notes["defined_in"].(string), ".stem") {
        t.Fatalf("notes=%+v", notes)
    }
}
```

- [ ] **Step 2: Verify RED**

```bash
go test ./internal/derive ./internal/rules ./cmd/rootline -run 'Test(EnrichBuiltins_.*Source|QuerySourceBacked|QueryDuplicateSource|SetSourceBacked|Describe.*Source|Explain.*Source)' -count=1
```

Expected: query precedence, ambiguity propagation, and explain/describe output disagree.

- [ ] **Step 3: Integrate the shared resolver**

Use `ResolveEffectiveField` in enrichment and inspection. Propagate ambiguity instead of emitting partial rows. Remove `sf.Type == "section"` from `set`; set always writes a frontmatter override and leaves the body unchanged.

- [ ] **Step 4: Separate public labels**

Change `SchemaField` JSON projection so the logical directive is `source` and physical declaration location is `defined_in`. Do not rename internal Go fields during this slice unless required for correctness.

- [ ] **Step 5: Verify GREEN**

```bash
go test ./internal/derive ./internal/rules ./cmd/rootline -run 'Test(EnrichBuiltins_.*Source|QuerySourceBacked|QueryDuplicateSource|SetSourceBacked|Describe.*Source|Explain.*Source)' -count=1
go test ./internal/derive ./internal/rules ./cmd/rootline -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit the slice**

```bash
git add internal/derive/enrich.go internal/derive/enrich_test.go internal/rules/describe.go internal/rules/describe_test.go internal/rules/explain.go internal/rules/explain_test.go cmd/rootline/query.go cmd/rootline/query_source_test.go cmd/rootline/set.go cmd/rootline/set_test.go cmd/rootline/describe.go cmd/rootline/explain.go cmd/rootline/inspection_source_test.go
git commit -m "fix(runtime): converge source-backed field consumers"
```

---

### Task 10: Emit governance-relative validation provenance

**Files:**
- Create: `internal/rules/source_normalization.go`
- Create: `internal/rules/source_normalization_test.go`
- Modify: `cmd/rootline/validate.go`
- Create: `cmd/rootline/validate_provenance_test.go`

**Interfaces:**

```go
func NormalizeValidationSources(errs []ValidationError, governanceRoot string) ([]ValidationError, error)
```

Normalization is per record because nested `root: true` boundaries may coexist in one `--all` run.

- [ ] **Step 1: Write RED normalization tests**

```go
func TestNormalizeValidationSources_GovernanceRelative(t *testing.T) {
    root := t.TempDir()
    source := filepath.Join(root, "docs", ".stem")
    got, err := NormalizeValidationSources([]ValidationError{{Source: source}}, root)
    if err != nil { t.Fatal(err) }
    if got[0].Source != "docs/.stem" { t.Fatalf("got %q", got[0].Source) }
}

func TestNormalizeValidationSources_PreservesSymbolic(t *testing.T) {
    for _, source := range []string{"schema", "scope", "links.checks"} {
        got, err := NormalizeValidationSources([]ValidationError{{Source: source}}, t.TempDir())
        if err != nil || got[0].Source != source { t.Fatalf("source=%q got=%+v err=%v", source, got, err) }
    }
}
```

Add these exact helpers and cases:

```go
func normalizePublicPath(path string) string {
    return strings.ReplaceAll(filepath.ToSlash(path), `\\`, "/")
}

func TestNormalizeValidationSources_RejectsOutsideRoot(t *testing.T) {
    root := t.TempDir()
    outside := filepath.Join(filepath.Dir(root), "outside", ".stem")
    _, err := NormalizeValidationSources([]ValidationError{{Source: outside}}, root)
    if err == nil { t.Fatal("expected outside-root error") }
}

func TestNormalizePublicPath_WindowsSeparators(t *testing.T) {
    if got := normalizePublicPath(`docs\\nested\\.stem`); got != "docs/nested/.stem" {
        t.Fatalf("got %q", got)
    }
}
```

- [ ] **Step 2: Verify RED**

```bash
go test ./internal/rules ./cmd/rootline -run 'Test(NormalizeValidationSources|ValidateErrorSources)' -count=1
```

Expected: missing helper and absolute source leakage.

- [ ] **Step 3: Implement fail-closed normalization**

```go
func NormalizeValidationSources(errs []ValidationError, root string) ([]ValidationError, error) {
    out := slices.Clone(errs)
    for i := range out {
        if isSymbolicValidationSource(out[i].Source) { continue }
        rel, err := filepath.Rel(root, out[i].Source)
        if err != nil || escapesRoot(rel) { return nil, fmt.Errorf("validation source outside governance root") }
        out[i].Source = filepath.ToSlash(rel)
    }
    return out, nil
}
```

Resolve the governance root from the root-most applicable `.stem`, not the scan subdirectory or process CWD.

- [ ] **Step 4: Verify cross-mode parity**

```bash
go test ./internal/rules ./cmd/rootline -run 'Test(NormalizeValidationSources|ValidateErrorSources)' -count=1
go test ./internal/rules ./cmd/rootline -count=1
```

Expected: single-file and `--all` output use the same relative source; no `\` or absolute path remains.

- [ ] **Step 5: Commit the slice**

```bash
git add internal/rules/source_normalization.go internal/rules/source_normalization_test.go cmd/rootline/validate.go cmd/rootline/validate_provenance_test.go
git commit -m "fix(validate): emit governance-relative error sources"
```

---

### Task 11: Migrate active schemas and synchronize living contracts

**Files:**
- Modify: `docs/roadmap/.stem`
- Modify: `lines/.stem`
- Modify: `paused/.stem`
- Modify: `closed/.stem`
- Modify: `theories/.stem`
- Modify: `README.md`
- Modify: `CLAUDE.md`
- Modify: `docs/{analyze,init,validate,migrate,new,query,describe,explain,set,extensibility,levels,UPGRADE}.md`
- Modify: `CHANGELOG.md`
- Modify: `.claude/skills/rootline/{SKILL.md,ref-schema.md,ref-validate.md,ref-query.md,ref-advanced.md}`
- Create: `cmd/rootline/documentation_contract_test.go`

**Exact schema migrations:**

```yaml
# docs/roadmap/.stem
is_done:
  type: boolean
  severity: off
```

```yaml
# lines/.stem, paused/.stem, closed/.stem, theories/.stem
# Replace every legacy `enum:` key with `values:`.
# Keep `type: integer` and native YAML integer data unchanged.
# Keep `values: [theory]` as a valid single-value enum.
```

- [ ] **Step 1: Write RED living-contract tests**

```go
func TestLivingSchemaDocumentationUsesCanonicalSourceDialect(t *testing.T) {
    forbidden := []string{"type: section", "heading:", "ordered:", "type: bool", "enum:"}
    assertLivingContractFilesExclude(t, livingSchemaContractFiles(), forbidden)
}
```

Use this explicit positive table in the same test:

```go
required := map[string][]string{
    "docs/validate.md": {"required means presence", "non_empty", "governance root"},
    "docs/init.md": {`source: body.section[`, "optional"},
    "docs/extensibility.md": {"type: boolean", "type: integer", "values: [theory]"},
    "docs/UPGRADE.md": {"type: section", "type: string", "type: bool", "type: boolean"},
    ".claude/skills/rootline/SKILL.md": {`source: body.section[`, "#190"},
}
for path, fragments := range required {
    data, err := os.ReadFile(path)
    if err != nil { t.Fatal(err) }
    for _, fragment := range fragments {
        if !bytes.Contains(data, []byte(fragment)) {
            t.Errorf("%s missing %q", path, fragment)
        }
    }
}
```

- [ ] **Step 2: Verify RED**

```bash
go test ./cmd/rootline ./internal/rules -run 'Test(LivingSchemaDocumentation|StemHealthDocumentationDrift)' -count=1
```

Expected: active schemas and living docs still contain legacy dialects.

- [ ] **Step 3: Apply exact active-schema migrations**

Edit only schema declarations. Do not rewrite numeric/boolean record data. Validate each governed tree with its existing root marker; if a tree is intentionally separate, run the smallest valid `rootline validate --all <tree> -o json` command.

- [ ] **Step 4: Synchronize living docs and skills**

Document:

- strict types, including boolean/integer and single-value enums;
- `type + source` section fields;
- empty-present section behavior;
- duplicate and inferred-name collision errors;
- stable source inheritance;
- `set` as frontmatter override;
- lexical scaffold order;
- governance-relative validation sources;
- manual legacy migration examples;
- hierarchical selectors deferred to #190.

Do not edit O10/O14 roadmap history or prior design records.

- [ ] **Step 5: Verify focused documentation and schema contracts**

```bash
go test ./cmd/rootline ./internal/rules -run 'Test(LivingSchemaDocumentation|StemHealthDocumentationDrift)' -count=1
rootline validate docs/superpowers/specs/2026-08-18-schema-contract-convergence-design.md -o json
rootline validate docs/superpowers/plans/2026-08-18-schema-contract-convergence.md -o json
```

Expected: PASS with no legacy living-contract matches.

- [ ] **Step 6: Run final repository verification**

```bash
just check
just test
just coverage-check
just validate
```

Expected:

- formatting, lint, and build pass;
- `go test ./... -race` passes;
- every package meets its configured 85% coverage floor;
- governed roadmap validation passes.

- [ ] **Step 7: Inspect the complete change before commit**

```bash
git diff --check
git status --short
git diff --stat
```

Expected: no unrelated `.gitignore` change staged or modified by this task; every changed path belongs to the approved spec.

- [ ] **Step 8: Commit the release-boundary slice**

```bash
git add docs/roadmap/.stem lines/.stem paused/.stem closed/.stem theories/.stem README.md CLAUDE.md docs .claude/skills/rootline cmd/rootline/documentation_contract_test.go
git commit -m "docs: synchronize canonical field contract guidance"
```

---

## Spec Coverage Matrix

| Requirement | Tasks |
|---|---|
| R1 one value-type vocabulary | 2, 3, 11 |
| R2 type/source separation | 1, 5, 9, 11 |
| R3 reject legacy section dialect | 2, 8, 11 |
| R4 enforce representation | 2, 3 |
| R5 independent presence rules | 1, 3 |
| R6 empty section presence | 1, 3, 8 |
| R7 duplicate ambiguity | 1, 8, 9 |
| R8 inference validates corpus | 5, 6 |
| R9 canonical materialization | 8 |
| R10 stable inherited source | 4 |
| R11 valid narrowing | 4 |
| R12 consistent widening rejection | 4 |
| R13 no derivative noise | 3 |
| R14 portable provenance | 10 |
| R15 symbolic provenance | 10 |
| R16 boolean/integer without coercion | 2, 3, 11 |
| R17 inferred-name collision rejection | 5, 11 |
| R18 single-value enum | 2, 3, 11 |

## Review Workload Forecast

- Estimated authored implementation: 3,300–4,300 lines across production, tests, active schema migrations, and living docs.
- 400-line budget risk: **High** for the complete change, controlled by eleven sequential review slices.
- Chained PRs recommended: **Yes** if work is published before the whole chain is ready.
- Each task is one commit and one fresh review gate; split a task again if its authored diff reaches 400 lines.
- #190 remains independent and may proceed on its own timeline without blocking this chain.
