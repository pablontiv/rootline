---
estado: Specified
---

# Type Representation Repair Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `fix` safely convert native YAML timestamp, boolean, and integer scalars to exact strings when the effective field contract requires `string`, while reporting every unsupported type mismatch.

**Architecture:** Extraction captures exact top-level YAML scalar lexemes and representations from the same `yaml.Node` used to decode frontmatter. Validation carries expected and actual representations through non-serialized fields; proposal analysis emits existing `correct_value` repairs with an optional `from_representation` discriminator or a non-repairable type finding. Both fix paths preserve existing contracts: in-process fixes quote the exact lexeme, stored reports use a representation-aware fail-closed guard only when the discriminator is present, and old reports retain strict matching.

**Tech Stack:** Go 1.26+, `gopkg.in/yaml.v3`, Cobra CLI, standard `encoding/json`, standard `testing`, Rootline's existing extraction, validation, proposal, fix, and atomic-write packages.

**Spec:** `docs/superpowers/specs/2026-08-24-type-representation-repair-design.md`

## Global Constraints

- Work only on branch `fix/type-representation-repair`; never commit directly to `master`.
- Never read from or write to `/Users/Shared/wiki`; all acceptance fixtures live under `t.TempDir()` or `/tmp/rootline-type-repair`.
- Preserve the exact YAML scalar lexeme. Never reconstruct a repair string from `time.Time`, `bool`, or an integer.
- Automatic repair allowlist is exactly `timestamp`, `boolean`, and `integer` when expected representation is `string`.
- Mapping, sequence, null, number, unknown representations, and inverse conversions are findings, never coercions.
- Reuse proposal type `correct_value`; do not add a proposal type or summary counter.
- `from_representation` is optional and type-aware matching runs only when it is non-empty.
- Existing proposal reports without `from_representation` retain strict `reflect.DeepEqual(current, p.From)` behavior.
- `extract.Record` scalar metadata and `ValidationError` machine evidence use `json:"-"`; public record and validate JSON remain unchanged.
- `type_findings` is additive and uses `omitempty`; proposal and fix envelopes remain version 1.
- Type findings do not alter `fix` completeness or exit status; `validate` remains the validity authority.
- Use TDD: observe every focused test fail for the intended reason before writing production code.
- Use Conventional Commits, neutral English identifiers/comments/docs, no attribution trailers.
- Run `git checkout -- docs/roadmap/` after any checkout because the local post-checkout hook may rewrite governed roadmap records.
- Keep `.claude/session-state/` untracked and untouched.

## File Structure

### New focused files

- `internal/extract/scalar_metadata.go` — scalar metadata type, YAML tag mapping, node traversal, and repairable-representation allowlist.
- `internal/extract/scalar_metadata_test.go` — exact lexeme, quoted-string, malformed-YAML, and CRLF extraction coverage.
- `internal/rules/validation_representation_test.go` — structured machine evidence and JSON non-exposure.
- `internal/proposal/type_representation.go` — safe type-repair and unsupported-finding classifier.
- `internal/proposal/type_representation_test.go` — detector matrix and one-proposal-or-one-finding invariant.
- `internal/proposal/from_representation_test.go` — proposal JSON compatibility tests.
- `internal/fix/type_representation_test.go` — stored-report guards, exact quoting, stale report rejection, and legacy compatibility.
- `cmd/rootline/fix_type_representation_test.go` — real command dry-run/apply/findings/idempotency contracts.

### Existing files modified

- `internal/extract/extract.go` — attach scalar metadata while decoding frontmatter.
- `internal/rules/validate.go` — carry expected and actual representations internally.
- `internal/proposal/proposal.go` — proposal/report fields and detector integration.
- `internal/fix/repair.go` — isolate representation-aware report matching.
- `cmd/rootline/fix.go` — propagate and render type findings in both output envelopes.
- `docs/fix.md` — proposal discriminator, safe conversion matrix, findings, and exit semantics.
- `.claude/skills/rootline/ref-validate.md` — agent-facing preview/apply behavior.
- `.claude/skills/rootline/SKILL.md` — concise safety rule for type findings and exact repairs.
- `CLAUDE.md` — architecture and command contract summary.
- `CHANGELOG.md` — unreleased fix entry for #196.

---

### Task 1: Preserve exact frontmatter scalar metadata

**Files:**
- Create: `internal/extract/scalar_metadata.go`
- Create: `internal/extract/scalar_metadata_test.go`
- Modify: `internal/extract/extract.go:29-40,73-112`

**Interfaces:**
- Consumes: `yaml.Node`, `MarkdownExtractor.Extract(path string, content []byte) (*Record, error)`, and the existing leading-frontmatter boundary.
- Produces: `extract.FrontmatterScalar{Lexeme string, Representation string}`; `Record.FrontmatterScalars map[string]FrontmatterScalar` with `json:"-"`; `extract.IsRepairableScalarRepresentation(name string) bool`; package-private `decodeFrontmatter(content string) (map[string]any, map[string]FrontmatterScalar, error)`.

- [ ] **Step 1: Write the failing exact-metadata tests**

Create `internal/extract/scalar_metadata_test.go`:

```go
package extract

import "testing"

func TestMarkdownExtractorPreservesScalarLexemes(t *testing.T) {
	rec, err := (&MarkdownExtractor{}).Extract("a.md", []byte(`---
date: 2026-06-22
timestamp: 2026-06-22T00:00:00Z
boolean: TRUE
octal: 042
signed: +42
quoted: "042"
---
# Probe
`))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	want := map[string]FrontmatterScalar{
		"date":      {Lexeme: "2026-06-22", Representation: "timestamp"},
		"timestamp": {Lexeme: "2026-06-22T00:00:00Z", Representation: "timestamp"},
		"boolean":   {Lexeme: "TRUE", Representation: "boolean"},
		"octal":     {Lexeme: "042", Representation: "integer"},
		"signed":    {Lexeme: "+42", Representation: "integer"},
	}
	if len(rec.FrontmatterScalars) != len(want) {
		t.Fatalf("FrontmatterScalars = %#v, want %#v", rec.FrontmatterScalars, want)
	}
	for field, expected := range want {
		if got := rec.FrontmatterScalars[field]; got != expected {
			t.Errorf("%s metadata = %#v, want %#v", field, got, expected)
		}
	}
	if _, ok := rec.FrontmatterScalars["quoted"]; ok {
		t.Error("quoted string must not be marked as a native scalar repair candidate")
	}
}

func TestMarkdownExtractorMalformedYAMLPublishesNoScalarEvidence(t *testing.T) {
	rec, err := (&MarkdownExtractor{}).Extract("a.md", []byte("---\ndate: [broken\n---\nbody\n"))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(rec.Errors) == 0 {
		t.Fatal("expected malformed YAML extraction error")
	}
	if len(rec.FrontmatterScalars) != 0 {
		t.Fatalf("malformed YAML published repair evidence: %#v", rec.FrontmatterScalars)
	}
}

func TestMarkdownExtractorScalarMetadataSupportsCRLF(t *testing.T) {
	rec, err := (&MarkdownExtractor{}).Extract("a.md", []byte("---\r\ndate: 2026-06-22\r\n---\r\nbody\r\n"))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	got := rec.FrontmatterScalars["date"]
	if got.Lexeme != "2026-06-22" || got.Representation != "timestamp" {
		t.Fatalf("date metadata = %#v", got)
	}
}

func TestMarkdownExtractorEmptyFrontmatterRemainsValid(t *testing.T) {
	rec, err := (&MarkdownExtractor{}).Extract("a.md", []byte("---\n---\n# Empty\n"))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(rec.Errors) != 0 || len(rec.Frontmatter) != 0 || len(rec.FrontmatterScalars) != 0 {
		t.Fatalf("empty frontmatter changed contract: %#v", rec)
	}
}

func TestIsRepairableScalarRepresentation(t *testing.T) {
	for _, name := range []string{"timestamp", "boolean", "integer"} {
		if !IsRepairableScalarRepresentation(name) {
			t.Errorf("%q must be repairable", name)
		}
	}
	for _, name := range []string{"", "string", "number", "mapping", "sequence", "null"} {
		if IsRepairableScalarRepresentation(name) {
			t.Errorf("%q must not be repairable", name)
		}
	}
}
```

- [ ] **Step 2: Run the extraction tests and observe the missing interface**

Run:

```bash
go test ./internal/extract -run 'ScalarMetadata|ScalarLexemes|RepairableScalar|EmptyFrontmatter' -v
```

Expected: FAIL to compile because `FrontmatterScalar`, `Record.FrontmatterScalars`, and `IsRepairableScalarRepresentation` do not exist.

- [ ] **Step 3: Add the scalar metadata model and node decoder**

Create `internal/extract/scalar_metadata.go`:

```go
package extract

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// FrontmatterScalar preserves the authored YAML text and the native
// representation resolved from it. It is repair evidence, not public record data.
type FrontmatterScalar struct {
	Lexeme         string
	Representation string
}

// IsRepairableScalarRepresentation reports whether quoting the exact scalar
// lexeme is an approved representation-only repair for a string field.
func IsRepairableScalarRepresentation(name string) bool {
	switch name {
	case "timestamp", "boolean", "integer":
		return true
	default:
		return false
	}
}

func representationForYAMLTag(tag string) (string, bool) {
	switch tag {
	case "!!timestamp":
		return "timestamp", true
	case "!!bool":
		return "boolean", true
	case "!!int":
		return "integer", true
	default:
		return "", false
	}
}

func decodeFrontmatter(content string) (map[string]any, map[string]FrontmatterScalar, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		return nil, nil, err
	}

	frontmatter := make(map[string]any)
	if err := doc.Decode(&frontmatter); err != nil {
		return nil, nil, err
	}

	scalars, err := collectFrontmatterScalars(&doc)
	if err != nil {
		return nil, nil, err
	}
	return frontmatter, scalars, nil
}

func collectFrontmatterScalars(doc *yaml.Node) (map[string]FrontmatterScalar, error) {
	scalars := make(map[string]FrontmatterScalar)
	if len(doc.Content) == 0 {
		return scalars, nil
	}
	if doc.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("frontmatter must be a YAML mapping")
	}
	mapping := doc.Content[0]
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		key, value := mapping.Content[i], mapping.Content[i+1]
		if value.Kind != yaml.ScalarNode {
			continue
		}
		representation, ok := representationForYAMLTag(value.Tag)
		if !ok {
			continue
		}
		scalars[key.Value] = FrontmatterScalar{
			Lexeme:         value.Value,
			Representation: representation,
		}
	}
	return scalars, nil
}
```

- [ ] **Step 4: Wire metadata into `Record` and `MarkdownExtractor.Extract`**

Add to `Record` in `internal/extract/extract.go`:

```go
	FrontmatterScalars map[string]FrontmatterScalar `json:"-"`
```

Replace the direct `yaml.Unmarshal` block with:

```go
	frontmatter, scalars, err := decodeFrontmatter(fmContent)
	if err != nil {
		record.Errors = append(record.Errors, ExtractionError{
			Line:    1,
			Message: fmt.Sprintf("malformed YAML frontmatter: %v", err),
		})
		record.Frontmatter = fallbackParseFrontmatter(fmContent)
		record.FrontmatterScalars = nil
	} else {
		record.Frontmatter = frontmatter
		record.FrontmatterScalars = scalars
	}
```

Remove the now-unused `gopkg.in/yaml.v3` import from `extract.go`; `scalar_metadata.go` owns it.

- [ ] **Step 5: Run focused and package tests**

Run:

```bash
gofmt -w internal/extract/extract.go internal/extract/scalar_metadata.go internal/extract/scalar_metadata_test.go
go test ./internal/extract -run 'ScalarMetadata|ScalarLexemes|RepairableScalar|EmptyFrontmatter|Extract_' -v
go test ./internal/extract -race
```

Expected: PASS. Existing empty-frontmatter, duplicate-key, malformed-frontmatter, body, and link extraction tests remain green.

- [ ] **Step 6: Commit extraction evidence**

```bash
git add internal/extract/extract.go internal/extract/scalar_metadata.go internal/extract/scalar_metadata_test.go
git commit -m "feat(extract): preserve scalar representation lexemes"
```

---

### Task 2: Carry structured type evidence without changing validate JSON

**Files:**
- Modify: `internal/rules/validate.go:16-56`
- Create: `internal/rules/validation_representation_test.go`

**Interfaces:**
- Consumes: `FieldContractIssue.Expected`, `FieldContractIssue.Actual`, and `validationErrorFromContract`.
- Produces: `ValidationError.ExpectedRepresentation string` and `ValidationError.ActualRepresentation string`, both `json:"-"`.

- [ ] **Step 1: Write the failing structured-evidence tests**

Create `internal/rules/validation_representation_test.go`:

```go
package rules

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidationErrorCarriesTypeRepresentationsInternally(t *testing.T) {
	err := validationErrorFromContract("ingested", SchemaField{Type: "string"}, &FieldContractIssue{
		Code:     "type-mismatch",
		Expected: "string",
		Actual:   "timestamp",
	})
	if err.ExpectedRepresentation != "string" {
		t.Errorf("ExpectedRepresentation = %q", err.ExpectedRepresentation)
	}
	if err.ActualRepresentation != "timestamp" {
		t.Errorf("ActualRepresentation = %q", err.ActualRepresentation)
	}
}

func TestValidationErrorRepresentationEvidenceIsNotJSON(t *testing.T) {
	err := ValidationError{
		Rule:                   "type",
		Field:                  "ingested",
		Message:                "human text",
		ExpectedRepresentation: "string",
		ActualRepresentation:   "timestamp",
	}
	data, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("Marshal: %v", marshalErr)
	}
	for _, forbidden := range []string{"expected_representation", "actual_representation", "timestamp"} {
		if strings.Contains(string(data), forbidden) {
			t.Errorf("internal evidence leaked into JSON: %s", data)
		}
	}
}
```

- [ ] **Step 2: Run the rules tests and observe the missing fields**

Run:

```bash
go test ./internal/rules -run 'ValidationError.*Representation' -v
```

Expected: FAIL to compile because the two `ValidationError` fields do not exist.

- [ ] **Step 3: Add and populate the internal fields**

Extend `ValidationError` in `internal/rules/validate.go`:

```go
	// ExpectedRepresentation and ActualRepresentation carry machine evidence
	// from the field contract to internal consumers. Public validate envelopes
	// keep their existing versioned shape.
	ExpectedRepresentation string `json:"-"`
	ActualRepresentation   string `json:"-"`
```

Populate them in `validationErrorFromContract`:

```go
		ExpectedRepresentation: issue.Expected,
		ActualRepresentation:   issue.Actual,
```

Keep the existing public `Rule`, `Field`, `Message`, `Source`, `Severity`, and `Suggestion` behavior unchanged for enum, link, and sequence remapping.

- [ ] **Step 4: Run focused and package tests**

```bash
gofmt -w internal/rules/validate.go internal/rules/validation_representation_test.go
go test ./internal/rules -run 'ValidationError.*Representation|Validate_Type|Timestamp' -v
go test ./internal/rules -race
```

Expected: PASS, including existing validation JSON and timestamp diagnostic tests.

- [ ] **Step 5: Commit structured validation evidence**

```bash
git add internal/rules/validate.go internal/rules/validation_representation_test.go
git commit -m "refactor(rules): retain internal type representation evidence"
```

---

### Task 3: Generate exact type repairs and non-repairable findings

**Files:**
- Modify: `internal/proposal/proposal.go:50-101,125-210`
- Create: `internal/proposal/type_representation.go`
- Create: `internal/proposal/type_representation_test.go`
- Create: `internal/proposal/from_representation_test.go`

**Interfaces:**
- Consumes: `extract.Record.FrontmatterScalars`, `extract.IsRepairableScalarRepresentation`, and structured fields on `rules.ValidationError` from Tasks 1–2.
- Produces: `Proposal.FromRepresentation string` with `json:"from_representation,omitempty"`; `TypeFinding`; `Report.TypeFindings`; `detectTypeRepresentationRepairs(records []*extract.Record, errs map[string][]rules.ValidationError) ([]Proposal, []TypeFinding)`.

- [ ] **Step 1: Write failing proposal contract tests**

Create `internal/proposal/from_representation_test.go`:

```go
package proposal

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestProposalFromRepresentationRoundTrips(t *testing.T) {
	input := Proposal{
		Type: CorrectValue, Field: "ingested", Paths: []string{"a.md"},
		From: "2026-06-22", To: "2026-06-22", FromRepresentation: "timestamp",
	}
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(data), `"from_representation":"timestamp"`) {
		t.Fatalf("missing discriminator: %s", data)
	}
	var output Proposal
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if output.FromRepresentation != "timestamp" {
		t.Fatalf("FromRepresentation = %q", output.FromRepresentation)
	}
}

func TestProposalFromRepresentationOmittedWhenEmpty(t *testing.T) {
	data, err := json.Marshal(Proposal{Type: CorrectValue, Field: "estado", From: "Pendng", To: "Pending"})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if strings.Contains(string(data), "from_representation") {
		t.Fatalf("legacy proposal shape changed: %s", data)
	}
}
```

- [ ] **Step 2: Write the failing detector matrix**

Create `internal/proposal/type_representation_test.go` with helpers and the core invariant:

```go
package proposal

import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

func typeError(field, expected, actual, message string) rules.ValidationError {
	return rules.ValidationError{
		Rule:                   "type",
		Field:                  field,
		Message:                message,
		ExpectedRepresentation: expected,
		ActualRepresentation:   actual,
	}
}

func TestDetectTypeRepresentationRepairsPreservesExactLexemes(t *testing.T) {
	record := &extract.Record{
		Path: "a.md",
		FrontmatterScalars: map[string]extract.FrontmatterScalar{
			"date":    {Lexeme: "2026-06-22T00:00:00Z", Representation: "timestamp"},
			"boolean": {Lexeme: "TRUE", Representation: "boolean"},
			"integer": {Lexeme: "042", Representation: "integer"},
		},
	}
	errs := map[string][]rules.ValidationError{"a.md": {
		typeError("date", "string", "timestamp", "message wording is irrelevant"),
		typeError("boolean", "string", "boolean", "another message"),
		typeError("integer", "string", "integer", "changed prose"),
	}}

	proposals, findings := detectTypeRepresentationRepairs([]*extract.Record{record}, errs)
	if len(findings) != 0 || len(proposals) != 3 {
		t.Fatalf("proposals=%#v findings=%#v", proposals, findings)
	}
	want := map[string]string{"date": "2026-06-22T00:00:00Z", "boolean": "TRUE", "integer": "042"}
	for _, p := range proposals {
		if p.Type != CorrectValue || p.From != want[p.Field] || p.To != want[p.Field] {
			t.Errorf("proposal = %#v", p)
		}
		if p.FromRepresentation != record.FrontmatterScalars[p.Field].Representation {
			t.Errorf("representation = %q for %s", p.FromRepresentation, p.Field)
		}
	}
}

func TestEveryTypeErrorBecomesOneRepairOrOneFinding(t *testing.T) {
	record := &extract.Record{
		Path: "a.md",
		FrontmatterScalars: map[string]extract.FrontmatterScalar{
			"safe":     {Lexeme: "+42", Representation: "integer"},
			"mismatch": {Lexeme: "TRUE", Representation: "boolean"},
		},
	}
	errs := map[string][]rules.ValidationError{"a.md": {
		typeError("safe", "string", "integer", "safe"),
		typeError("mapping", "string", "mapping", "mapping"),
		typeError("sequence", "string", "sequence", "sequence"),
		typeError("null", "string", "null", "null"),
		typeError("number", "string", "number", "number"),
		typeError("inverse", "boolean", "string", "inverse"),
		typeError("mismatch", "string", "integer", "metadata disagrees"),
	}}

	proposals, findings := detectTypeRepresentationRepairs([]*extract.Record{record}, errs)
	if len(proposals) != 1 || len(findings) != 6 {
		t.Fatalf("proposals=%d findings=%d", len(proposals), len(findings))
	}
	if len(proposals)+len(findings) != len(errs["a.md"]) {
		t.Fatal("a type error was duplicated or dropped")
	}
}
```

- [ ] **Step 3: Run tests and observe missing proposal interfaces**

```bash
go test ./internal/proposal -run 'FromRepresentation|TypeRepresentation|EveryTypeError' -v
```

Expected: FAIL to compile because the proposal field, finding type, report field, and detector are absent.

- [ ] **Step 4: Extend proposal and report contracts**

In `internal/proposal/proposal.go`, add after `ValueSource`:

```go
	// FromRepresentation names the YAML representation expected on disk for a
	// type-mismatch repair. From carries the exact scalar lexeme; this optional
	// discriminator isolates representation-aware matching from legacy repairs.
	FromRepresentation string `json:"from_representation,omitempty"`
```

Add beside `LinkFinding`:

```go
// TypeFinding reports a representation error fix cannot repair safely.
type TypeFinding struct {
	Path                 string `json:"path"`
	Field                string `json:"field"`
	Message              string `json:"message"`
	ActualRepresentation string `json:"actual_representation,omitempty"`
}
```

Add to `Report`:

```go
	TypeFindings []TypeFinding `json:"type_findings,omitempty"`
```

- [ ] **Step 5: Implement the detector in a focused file**

Create `internal/proposal/type_representation.go`:

```go
package proposal

import (
	"fmt"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

func detectTypeRepresentationRepairs(records []*extract.Record, errs map[string][]rules.ValidationError) ([]Proposal, []TypeFinding) {
	recordsByPath := make(map[string]*extract.Record, len(records))
	for _, record := range records {
		recordsByPath[record.Path] = record
	}

	var proposals []Proposal
	var findings []TypeFinding
	for path, pathErrs := range errs {
		for _, validationErr := range pathErrs {
			if validationErr.Rule != "type" {
				continue
			}

			record := recordsByPath[path]
			scalar, hasScalar := extract.FrontmatterScalar{}, false
			if record != nil {
				scalar, hasScalar = record.FrontmatterScalars[validationErr.Field]
			}
			repairable := validationErr.ExpectedRepresentation == "string" &&
				extract.IsRepairableScalarRepresentation(validationErr.ActualRepresentation) &&
				hasScalar && scalar.Representation == validationErr.ActualRepresentation

			if !repairable {
				findings = append(findings, TypeFinding{
					Path:                 path,
					Field:                validationErr.Field,
					Message:              validationErr.Message,
					ActualRepresentation: validationErr.ActualRepresentation,
				})
				continue
			}

			proposals = append(proposals, Proposal{
				Type:               CorrectValue,
				Field:              validationErr.Field,
				Description:        fmt.Sprintf("quote %s value %q as an exact string", scalar.Representation, scalar.Lexeme),
				Paths:              []string{path},
				From:               scalar.Lexeme,
				To:                 scalar.Lexeme,
				FromRepresentation: scalar.Representation,
			})
		}
	}
	return proposals, findings
}
```

- [ ] **Step 6: Integrate detection into `Analyze` and summary accounting**

In `Analyze`, call the detector once after existing schema-dependent detectors:

```go
	typeRepairs, typeFindings := detectTypeRepresentationRepairs(records, errs)
	proposals = append(proposals, typeRepairs...)
```

Return the findings in the report:

```go
		TypeFindings: typeFindings,
```

Do not add a switch case or summary field: the existing `CorrectValue` case counts safe type repairs.

- [ ] **Step 7: Run detector, contract, and regression tests**

```bash
gofmt -w internal/proposal/proposal.go internal/proposal/type_representation.go internal/proposal/type_representation_test.go internal/proposal/from_representation_test.go
go test ./internal/proposal -run 'FromRepresentation|TypeRepresentation|EveryTypeError|CorrectValue|MigrateValue' -v
go test ./internal/proposal -race
```

Expected: PASS. Existing enum typo and migration priority behavior remains unchanged.

- [ ] **Step 8: Commit proposal generation and findings**

```bash
git add internal/proposal/proposal.go internal/proposal/type_representation.go internal/proposal/type_representation_test.go internal/proposal/from_representation_test.go
git commit -m "feat(proposal): detect safe type representation repairs"
```

---

### Task 4: Apply stored type repairs with a fail-closed guard

**Files:**
- Modify: `internal/fix/repair.go:409-455`
- Create: `internal/fix/type_representation_test.go`

**Interfaces:**
- Consumes: `Proposal.FromRepresentation`, `extract.Record.FrontmatterScalars`, and `extract.IsRepairableScalarRepresentation`.
- Produces: package-private `proposalSourceMatches(record *extract.Record, current any, p *proposal.Proposal) bool`; unchanged `ApplyRepair` result contract.

- [ ] **Step 1: Write failing acceptance and rejection tests**

Create `internal/fix/type_representation_test.go`:

```go
package fix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/proposal"
)

func writeTypeRepairFixture(t *testing.T, valueLine string) string {
	t.Helper()
	dir := t.TempDir()
	stem := "version: 2\nroot: true\nscope:\n  match: \"*.md\"\nschema:\n  value:\n    type: string\n"
	if err := os.WriteFile(filepath.Join(dir, ".stem"), []byte(stem), 0o644); err != nil {
		t.Fatal(err)
	}
	content := "---\nvalue: " + valueLine + "\n---\n# Probe\n"
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func typeRepairProposal(from, representation string) proposal.Proposal {
	return proposal.Proposal{
		Type: proposal.CorrectValue, Field: "value", Paths: []string{"a.md"},
		From: from, To: from, FromRepresentation: representation,
	}
}

func TestApplyRepairQuotesExactScalarLexeme(t *testing.T) {
	dir := writeTypeRepairFixture(t, "042")
	result, err := ApplyRepair([]proposal.Proposal{typeRepairProposal("042", "integer")}, false, dir, false)
	if err != nil {
		t.Fatalf("ApplyRepair: %v", err)
	}
	if len(result.Changed) != 1 || len(result.Rejected) != 0 || len(result.RolledBack) != 0 {
		t.Fatalf("result = %#v", result)
	}
	content, err := os.ReadFile(filepath.Join(dir, "a.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), `value: "042"`) {
		t.Fatalf("exact lexeme was not quoted once:\n%s", content)
	}
	info, err := os.Stat(filepath.Join(dir, "a.md"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 600", info.Mode().Perm())
	}
}

func TestApplyRepairTypeGuardRejectsStaleOrUnknownEvidence(t *testing.T) {
	cases := []struct {
		name           string
		valueLine      string
		from           string
		representation string
	}{
		{name: "changed lexeme", valueLine: "+42", from: "42", representation: "integer"},
		{name: "changed representation", valueLine: "true", from: "true", representation: "integer"},
		{name: "unknown marker", valueLine: "42", from: "42", representation: "number"},
		{name: "quoted string", valueLine: `"42"`, from: "42", representation: "integer"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := writeTypeRepairFixture(t, tc.valueLine)
			before, err := os.ReadFile(filepath.Join(dir, "a.md"))
			if err != nil {
				t.Fatal(err)
			}
			result, err := ApplyRepair([]proposal.Proposal{typeRepairProposal(tc.from, tc.representation)}, false, dir, false)
			if err != nil {
				t.Fatalf("ApplyRepair: %v", err)
			}
			if len(result.Rejected) != 1 || len(result.Changed) != 0 {
				t.Fatalf("result = %#v", result)
			}
			after, err := os.ReadFile(filepath.Join(dir, "a.md"))
			if err != nil {
				t.Fatal(err)
			}
			if string(after) != string(before) {
				t.Fatal("rejected proposal modified the file")
			}
		})
	}
}

func TestApplyRepairLegacyCorrectValueRemainsStrict(t *testing.T) {
	dir := writeTypeRepairFixture(t, "zed")
	legacy := proposal.Proposal{
		Type: proposal.CorrectValue, Field: "value", Paths: []string{"a.md"},
		From: "alice", To: "bob",
	}
	result, err := ApplyRepair([]proposal.Proposal{legacy}, false, dir, false)
	if err != nil {
		t.Fatalf("ApplyRepair: %v", err)
	}
	if len(result.Rejected) != 1 || len(result.Changed) != 0 {
		t.Fatalf("legacy strict guard changed: %#v", result)
	}
}
```

- [ ] **Step 2: Run tests and observe type proposals rejected by the legacy guard**

```bash
go test ./internal/fix -run 'ApplyRepair.*(ExactScalar|TypeGuard|LegacyCorrect)' -v
```

Expected: `TestApplyRepairQuotesExactScalarLexeme` FAIL because `time/int/bool current` does not strictly equal string `From`; rejection tests and legacy control establish the required boundaries.

- [ ] **Step 3: Add the isolated source-matching helper**

Add beside `proposalValueMatches` in `internal/fix/repair.go`:

```go
func proposalSourceMatches(record *extract.Record, current any, p *proposal.Proposal) bool {
	if p.FromRepresentation == "" {
		return proposalValueMatches(current, p.From)
	}
	if !extract.IsRepairableScalarRepresentation(p.FromRepresentation) {
		return false
	}
	scalar, ok := record.FrontmatterScalars[p.Field]
	return ok &&
		scalar.Representation == p.FromRepresentation &&
		scalar.Lexeme == p.From
}
```

This helper has no fallback from a populated but invalid discriminator to legacy matching.

- [ ] **Step 4: Use the helper only in the stale-report guard**

In `applyRepairCorrectValue`, preserve the existing already-applied check first, then replace only the `From` mismatch branch:

```go
		case !exists || !proposalSourceMatches(tgt.record, current, p):
			result.Rejected = append(result.Rejected,
				fmt.Sprintf("%s in %s is not expected value %q", p.Field, path, p.From))
			continue
```

Do not change `proposalValueMatches`, `applyRepairAddField`, or any proposal type other than `CorrectValue`/`MigrateValue` reaching this handler. `MigrateValue` never carries `FromRepresentation`, so it remains on the legacy branch.

- [ ] **Step 5: Add idempotency and timestamp/boolean cases**

Extend the new test file:

```go
func TestApplyRepairTypeRepresentationIsIdempotent(t *testing.T) {
	cases := []struct {
		line, from, representation, quoted string
	}{
		{"2026-06-22T00:00:00Z", "2026-06-22T00:00:00Z", "timestamp", `"2026-06-22T00:00:00Z"`},
		{"TRUE", "TRUE", "boolean", `"TRUE"`},
		{"+42", "+42", "integer", `"+42"`},
	}
	for _, tc := range cases {
		t.Run(tc.representation+"/"+tc.from, func(t *testing.T) {
			dir := writeTypeRepairFixture(t, tc.line)
			p := typeRepairProposal(tc.from, tc.representation)
			first, err := ApplyRepair([]proposal.Proposal{p}, false, dir, false)
			if err != nil || len(first.Changed) != 1 {
				t.Fatalf("first apply: result=%#v err=%v", first, err)
			}
			firstBytes, err := os.ReadFile(filepath.Join(dir, "a.md"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(firstBytes), "value: "+tc.quoted) {
				t.Fatalf("exact quote missing:\n%s", firstBytes)
			}
			second, err := ApplyRepair([]proposal.Proposal{p}, false, dir, false)
			if err != nil || len(second.Changed) != 0 || len(second.Skipped) != 1 {
				t.Fatalf("second apply: result=%#v err=%v", second, err)
			}
			secondBytes, err := os.ReadFile(filepath.Join(dir, "a.md"))
			if err != nil {
				t.Fatal(err)
			}
			if string(secondBytes) != string(firstBytes) {
				t.Fatal("second apply changed bytes")
			}
		})
	}
}
```

- [ ] **Step 6: Run fix package tests with race detection**

```bash
gofmt -w internal/fix/repair.go internal/fix/type_representation_test.go
go test ./internal/fix -run 'ApplyRepair.*(ExactScalar|TypeGuard|LegacyCorrect|TypeRepresentation)' -v
go test ./internal/fix -race
```

Expected: PASS, including issue #178 stale-report controls, rollback, atomicity, and file-mode tests.

- [ ] **Step 7: Commit the fail-closed application path**

```bash
git add internal/fix/repair.go internal/fix/type_representation_test.go
git commit -m "fix(repair): guard typed scalar repairs by exact lexeme"
```

---

### Task 5: Expose findings and prove the complete CLI workflow

**Files:**
- Modify: `cmd/rootline/fix.go:29-47,258-325,524-610`
- Create: `cmd/rootline/fix_type_representation_test.go`

**Interfaces:**
- Consumes: `proposal.Report.TypeFindings` and existing `renderLinkFindings`/batch propagation patterns.
- Produces: `BatchFixResult.TypeFindings []proposal.TypeFinding`; `renderTypeFindings(cmd *cobra.Command, findings []proposal.TypeFinding)`; JSON key `type_findings`; unchanged exit semantics.

- [ ] **Step 1: Write failing command tests for preview, apply, findings, and idempotency**

Create `cmd/rootline/fix_type_representation_test.go`:

```go
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTypeRepresentationFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	stem := `version: 2
root: true
scope:
  match: "*.md"
schema:
  date:
    type: string
  boolean:
    type: string
  integer:
    type: string
  object:
    type: string
`
	doc := `---
date: 2026-06-22T00:00:00Z
boolean: TRUE
integer: 042
object:
  nested: value
---
# Probe
`
	if err := os.WriteFile(filepath.Join(dir, ".stem"), []byte(stem), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte(doc), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestFixAllDryRunReportsSafeTypeRepairsAndUnsupportedFindings(t *testing.T) {
	dir := setupTypeRepresentationFixture(t)
	before, err := os.ReadFile(filepath.Join(dir, "a.md"))
	if err != nil {
		t.Fatal(err)
	}
	out, err := runCmd(t, "fix", "--all", dir, "--dry-run")
	if err != nil {
		t.Fatalf("dry-run: %v\n%s", err, out)
	}
	var report struct {
		Proposals    []map[string]any `json:"proposals"`
		TypeFindings []map[string]any `json:"type_findings"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("JSON: %v\n%s", err, out)
	}
	if len(report.Proposals) != 3 || len(report.TypeFindings) != 1 {
		t.Fatalf("proposals=%d findings=%d output=%s", len(report.Proposals), len(report.TypeFindings), out)
	}
	if !strings.Contains(out, `"from_representation":"timestamp"`) ||
		!strings.Contains(out, `"from_representation":"boolean"`) ||
		!strings.Contains(out, `"from_representation":"integer"`) {
		t.Fatalf("missing typed proposal evidence: %s", out)
	}
	after, err := os.ReadFile(filepath.Join(dir, "a.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("dry-run modified the document")
	}
}

func TestFixAllAppliesExactTypeRepairsAndRemainsIdempotent(t *testing.T) {
	dir := setupTypeRepresentationFixture(t)
	firstOut, err := runCmd(t, "fix", "--all", dir)
	if err != nil {
		t.Fatalf("first fix: %v\n%s", err, firstOut)
	}
	firstBytes, err := os.ReadFile(filepath.Join(dir, "a.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, quoted := range []string{`date: "2026-06-22T00:00:00Z"`, `boolean: "TRUE"`, `integer: "042"`} {
		if !strings.Contains(string(firstBytes), quoted) {
			t.Errorf("missing %s:\n%s", quoted, firstBytes)
		}
	}
	if !strings.Contains(firstOut, "type_findings") || !strings.Contains(firstOut, "mapping") {
		t.Fatalf("applied run hid unsupported finding: %s", firstOut)
	}

	secondOut, err := runCmd(t, "fix", "--all", dir)
	if err != nil {
		t.Fatalf("second fix: %v\n%s", err, secondOut)
	}
	secondBytes, err := os.ReadFile(filepath.Join(dir, "a.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(secondBytes) != string(firstBytes) {
		t.Fatal("second fix was not idempotent")
	}
	if strings.Contains(secondOut, "from_representation") {
		t.Fatalf("second run proposed already-applied scalar repairs: %s", secondOut)
	}
	if !strings.Contains(secondOut, "type_findings") {
		t.Fatalf("unsupported mapping finding disappeared: %s", secondOut)
	}
}
```

Add the table-output test through the existing command helper:

```go
func TestFixAllTypeFindingsTable(t *testing.T) {
	dir := setupTypeRepresentationFixture(t)
	out, err := runCmd(t, "fix", "--all", dir, "--dry-run", "--output", "table")
	if err != nil {
		t.Fatalf("table dry-run: %v\n%s", err, out)
	}
	for _, want := range []string{
		"Type findings: 1 (reported, not repaired)",
		"a.md", "object", "mapping",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("table output missing %q:\n%s", want, out)
		}
	}
}
```

- [ ] **Step 2: Run the command tests and observe missing output plumbing**

```bash
go test ./cmd/rootline -run 'FixAll.*TypeRepair|FixAll.*TypeRepresentation|RenderTypeFindings' -v
```

Expected: detector-backed proposal assertions pass once Task 3 is present, while applied output and table assertions FAIL because `BatchFixResult` does not yet carry or render type findings.

- [ ] **Step 3: Add type findings to applied batch output**

Extend `BatchFixResult`:

```go
	// TypeFindings reports representation mismatches fix cannot repair safely.
	// They are observations, not failed writes; validate owns corpus validity.
	TypeFindings []proposal.TypeFinding `json:"type_findings,omitempty"`
```

After each `newBatchFixResultWithSuggestions` call in the unresolved and normal apply branches, assign:

```go
	batch.TypeFindings = report.TypeFindings
```

Dry-run JSON already serializes `proposal.Report`; do not build a second findings collection in `cmd/rootline`.

- [ ] **Step 4: Render findings in both table paths**

Add:

```go
func renderTypeFindings(cmd *cobra.Command, findings []proposal.TypeFinding) {
	if len(findings) == 0 {
		return
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout())
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Type findings: %d (reported, not repaired)\n", len(findings))
	for _, finding := range findings {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s: %s: %s (%s)\n",
			finding.Path, finding.Field, finding.Message, finding.ActualRepresentation)
	}
}
```

Call `renderTypeFindings(cmd, report.TypeFindings)` after `renderProposalTable` in dry-run table output, and `renderTypeFindings(cmd, batch.TypeFindings)` beside `renderLinkFindings` in applied table output.

Do not include findings in `fixBatchFailed`; exit status remains unchanged.

- [ ] **Step 5: Add an explicit exit-contract test**

Add to the command test file:

```go
func TestFixTypeFindingsDoNotReplaceValidateExitContract(t *testing.T) {
	dir := t.TempDir()
	stem := "version: 2\nroot: true\nscope:\n  match: \"*.md\"\nschema:\n  object:\n    type: string\n"
	doc := "---\nobject:\n  nested: value\n---\n# Probe\n"
	if err := os.WriteFile(filepath.Join(dir, ".stem"), []byte(stem), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"fix", "--all", dir, "--dry-run"},
		{"fix", "--all", dir},
	} {
		out, err := runCmd(t, args...)
		if err != nil {
			t.Fatalf("%v returned failure: %v\n%s", args, err, out)
		}
		if !strings.Contains(out, "type_findings") {
			t.Fatalf("%v hid finding: %s", args, out)
		}
	}
	out, err := runCmd(t, "validate", "--all", dir)
	if err == nil {
		t.Fatalf("validate unexpectedly accepted mapping-to-string mismatch: %s", out)
	}
}
```

This pins the distinction between successful repair execution and corpus validity.

- [ ] **Step 6: Run command, repair, and end-to-end packages**

```bash
gofmt -w cmd/rootline/fix.go cmd/rootline/fix_type_representation_test.go
go test ./cmd/rootline -run 'FixAll.*(Type|Finding)|RenderTypeFindings' -v
go test ./cmd/rootline ./internal/proposal ./internal/fix ./internal/e2e -race
```

Expected: PASS. Existing `link_findings`, proposal table, batch envelope, output-field, and repair command tests remain green.

- [ ] **Step 7: Commit CLI contracts and integration coverage**

```bash
git add cmd/rootline/fix.go cmd/rootline/fix_type_representation_test.go
git commit -m "fix(cli): surface and apply type representation repairs"
```

---

### Task 6: Synchronize living contracts and verify the compiled artifact

**Files:**
- Modify: `docs/fix.md:6-65,70-160,214-240`
- Modify: `.claude/skills/rootline/ref-validate.md:172-242`
- Modify: `.claude/skills/rootline/SKILL.md:130-142`
- Modify: `CLAUDE.md:31-45`
- Modify: `CHANGELOG.md:7-20,70-100`

**Interfaces:**
- Consumes: final JSON shapes and behavior from Tasks 1–5.
- Produces: documented `from_representation`, `type_findings`, safe allowlist, exact-lexeme guarantee, old-report compatibility, and exit semantics.

- [ ] **Step 1: Update `docs/fix.md` with the executable contract**

Add a **Type representation repairs** section after **Link findings** containing this matrix:

| Expected | Actual YAML representation | Result |
|---|---|---|
| string | timestamp | exact lexeme quoted automatically |
| string | boolean | exact lexeme quoted automatically |
| string | integer | exact lexeme quoted automatically |
| string | mapping, sequence, null, number | `type_findings`, no mutation |
| boolean/integer/other | string or another representation | `type_findings`, no coercion |

State explicitly:

- `from` and `to` can be textually equal because the representation changes.
- `from_representation` is set only for type repairs and omitted for historical proposal shapes.
- `repair apply` requires exact lexeme and representation matches; unknown markers fail closed.
- Findings are reported in JSON/table but do not change exit 0; callers run `validate` for a validity verdict.
- The YAML writer adds quoting; reports never embed quote characters in `to`.

Extend the Proposal Struct table with `from`, `to`, `value_source`, and `from_representation`; correct the existing sentence so `heading`/`mode` apply to `set_section`, not `set_field`.

- [ ] **Step 2: Update the Rootline agent skill**

In `.claude/skills/rootline/ref-validate.md`, extend Batch Preview and Batch Apply examples with optional `type_findings`. Add this operational rule:

```markdown
A `correct_value` proposal with `from_representation` is a representation-only repair.
Its `from` and `to` preserve the same exact scalar text; applying it quotes the YAML value.
Only timestamp, boolean and integer to string are automatic. Treat `type_findings` as
unresolved validation defects and run `validate` after repair even though findings do not
change the `fix` exit code.
```

In `.claude/skills/rootline/SKILL.md`, add a concise sibling paragraph beside `link_findings` with the same safety boundary. Do not repeat the full matrix in the short skill file; link to `ref-validate.md`.

- [ ] **Step 3: Update maintainer architecture and changelog**

In `CLAUDE.md`:

- extend the `internal/extract` bullet with exact scalar lexeme metadata;
- extend the `internal/proposal` bullet with type-representation repairs/findings;
- extend the `internal/fix` and `cmd/rootline` bullets with the isolated guard and applied output behavior.

Under `CHANGELOG.md` → Unreleased → Fixed, add:

```markdown
- `fix` now detects native YAML timestamp, boolean and integer values in governed string fields, preserves their exact scalar text while quoting them, and reports unsupported `rule: type` mismatches under `type_findings`. Stored repair reports carry optional `from_representation` evidence and reject stale lexeme or representation changes without weakening historical `correct_value` guards. Fix findings remain informational; `validate` remains the corpus-validity command. Closes #196.
```

- [ ] **Step 4: Run documentation and repository gates**

```bash
gofmt -l .
just check
just test
just coverage-check
rootline validate docs/superpowers/plans/2026-08-24-type-representation-repair.md -o json
rootline validate docs/superpowers/specs/2026-08-24-type-representation-repair-design.md -o json
rootline validate docs/adr/0002-preservar-lexemas-en-reparaciones-de-tipo.md -o json
git diff --check
```

Expected: no gofmt output; all Go/lint/race/coverage gates green; all three governed documents valid; no whitespace errors.

- [ ] **Step 5: Build and run the isolated acceptance fixture**

```bash
set -euo pipefail
rm -rf /tmp/rootline-type-repair
mkdir -p /tmp/rootline-type-repair
bin=/tmp/rootline-type-repair/rootline
go build -o "$bin" ./cmd/rootline
cat > /tmp/rootline-type-repair/.stem <<'STEM'
version: 2
root: true
scope:
  match: "*.md"
schema:
  date:
    type: string
  boolean:
    type: string
  integer:
    type: string
STEM
cat > /tmp/rootline-type-repair/a.md <<'DOC'
---
date: 2026-06-22T00:00:00Z
boolean: TRUE
integer: 042
---
# Probe
DOC

set +e
"$bin" validate --all /tmp/rootline-type-repair -o json > /tmp/rootline-type-repair/before.json
before_exit=$?
set -e
test "$before_exit" -eq 1

"$bin" fix --all /tmp/rootline-type-repair --dry-run -o json > /tmp/rootline-type-repair/report.json
jq -e '.summary.correct_value == 3' /tmp/rootline-type-repair/report.json
jq -e '[.proposals[].from_representation] | sort == ["boolean","integer","timestamp"]' /tmp/rootline-type-repair/report.json

"$bin" repair apply --report /tmp/rootline-type-repair/report.json -o json > /tmp/rootline-type-repair/applied.json
jq -e '.complete == true and (.changed | length) == 3' /tmp/rootline-type-repair/applied.json
rg -n '^date: "2026-06-22T00:00:00Z"$|^boolean: "TRUE"$|^integer: "042"$' /tmp/rootline-type-repair/a.md

"$bin" validate --all /tmp/rootline-type-repair -o json > /tmp/rootline-type-repair/after.json
jq -e '.summary.invalid == 0 and .summary.valid == 1' /tmp/rootline-type-repair/after.json
cp /tmp/rootline-type-repair/a.md /tmp/rootline-type-repair/first-pass.md
"$bin" fix --all /tmp/rootline-type-repair -o json > /tmp/rootline-type-repair/second.json
cmp /tmp/rootline-type-repair/first-pass.md /tmp/rootline-type-repair/a.md
jq -e '.summary.fixed == 0' /tmp/rootline-type-repair/second.json
```

Expected: initial validate exit 1; report has three typed repairs; repair apply writes three exact quoted strings; validate passes; second fix leaves bytes unchanged and reports zero fixed records.

- [ ] **Step 6: Run stale-report acceptance**

Recreate the fixture and report, then change the date while independently quoting the boolean and integer before `repair apply`. This leaves no accepted write on the still-invalid date file, so the test isolates the stale-report guard rather than the existing post-validation rollback contract:

```bash
perl -0pi -e 's/2026-06-22T00:00:00Z/2026-06-23T00:00:00Z/; s/boolean: TRUE/boolean: "TRUE"/; s/integer: 042/integer: "042"/' /tmp/rootline-type-repair/a.md
"$bin" repair apply --report /tmp/rootline-type-repair/report.json -o json > /tmp/rootline-type-repair/stale.json
jq -e '.complete == true and (.changed | length) == 0 and (.skipped | length) == 2 and (.rejected | length) == 1' /tmp/rootline-type-repair/stale.json
rg -n '^date: 2026-06-23T00:00:00Z$|^boolean: "TRUE"$|^integer: "042"$' /tmp/rootline-type-repair/a.md
```

Expected: the changed date remains untouched and rejected; already-correct boolean and integer values are skipped; rejection alone preserves exit 0 and `complete: true` under the existing repair contract.

- [ ] **Step 7: Commit living documentation**

```bash
git add docs/fix.md .claude/skills/rootline/ref-validate.md .claude/skills/rootline/SKILL.md CLAUDE.md CHANGELOG.md
git commit -m "docs: document exact type representation repairs"
```

- [ ] **Step 8: Final branch verification**

```bash
just check && just test && just coverage-check
git diff --check master...HEAD
git status --short --branch
git log --oneline master..HEAD
```

Expected: every gate green; only the pre-existing untracked `.claude/session-state/` remains; the branch contains the design commit plus focused extraction, validation, proposal, repair, CLI, and documentation commits.

Clean the acceptance fixture only after capturing results:

```bash
rm -rf /tmp/rootline-type-repair
```
