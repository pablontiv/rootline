# DDL Governance Detectors Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add 4 governance detectors to rootline's `analyze` pipeline that flag DDL-level gaps — missing domains, missing schemas, validation gaps, and naming inconsistencies.

**Architecture:** New detectors follow the existing functional pattern (`func Detect*(inputs) []Inference`) and register as categories in `analyze.go`. Agent-required inferences are gated via `agentRequiredTypes`. The `apply` command gains a pre-phase for `missing_schema` (file creation) and new switch cases for mechanical fixes.

**Tech Stack:** Go 1.25+, standard `testing` package, `gopkg.in/yaml.v3` for YAML node manipulation, `os`/`filepath` for filesystem walking.

**Spec:** `docs/superpowers/specs/2026-03-24-rootline-ddl-governance.md`

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `internal/infer/domain_coverage.go` | Create | Detect fields without `domain` in stem schemas |
| `internal/infer/domain_coverage_test.go` | Create | Unit tests for domain coverage detector |
| `internal/infer/schema_coverage.go` | Create | Detect directories without `.stem` files |
| `internal/infer/schema_coverage_test.go` | Create | Unit tests for schema coverage detector |
| `internal/infer/validation_gaps.go` | Create | Detect fields with insufficient validation rules |
| `internal/infer/validation_gaps_test.go` | Create | Unit tests for validation gaps detector |
| `internal/infer/structural.go` | Modify | Add naming inconsistency detection |
| `internal/infer/structural_test.go` | Modify | Add naming inconsistency tests |
| `internal/infer/delta.go` | Modify | Add `isCovered()` cases for governance types |
| `internal/infer/delta_test.go` | Modify | Add incremental filter tests |
| `internal/infer/apply.go` | Modify | Add apply handlers for `untyped_field`, `sequence_incomplete` |
| `internal/infer/scaffold.go` | Create | `ScaffoldSchema` — creates minimal `.stem` from observed fields |
| `cmd/rootline/apply.go` | Modify | Add `ScaffoldSchema` pre-phase for `missing_schema` |
| `cmd/rootline/analyze.go` | Modify | Register 3 new categories, extend `agentRequiredTypes` |
| `internal/e2e/analyze_test.go` | Modify | Update `runAnalyze` helper with governance categories |
| `internal/e2e/governance_test.go` | Create | E2E test for full governance pipeline |

---

### Task 1: Domain Coverage Detector

**Files:**
- Create: `internal/infer/domain_coverage.go`
- Create: `internal/infer/domain_coverage_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/infer/domain_coverage_test.go
package infer

import (
	"testing"

	"github.com/pablontiv/rootline/internal/rules"
)

func TestDetectMissingDomains_FlagsMissingDomain(t *testing.T) {
	stem := &rules.StemFile{
		Path: "docs/.stem",
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"draft", "active"}, Source: "docs/.stem"},
			"titulo": {Type: "string", Domain: "title", Source: "docs/.stem"},
		},
	}
	got := DetectMissingDomains(stem)
	if len(got) != 1 {
		t.Fatalf("expected 1 inference, got %d", len(got))
	}
	if got[0].Type != "missing_domain" {
		t.Errorf("expected type missing_domain, got %s", got[0].Type)
	}
	if got[0].Field != "estado" {
		t.Errorf("expected field estado, got %s", got[0].Field)
	}
	if got[0].Source != "docs/.stem" {
		t.Errorf("expected source docs/.stem, got %s", got[0].Source)
	}
}

func TestDetectMissingDomains_SkipsSections(t *testing.T) {
	stem := &rules.StemFile{
		Path: "docs/.stem",
		Schema: map[string]rules.SchemaField{
			"contexto": {Type: "section", Heading: "Contexto", Source: "docs/.stem"},
		},
	}
	got := DetectMissingDomains(stem)
	if len(got) != 0 {
		t.Fatalf("expected 0 inferences for section fields, got %d", len(got))
	}
}

func TestDetectMissingDomains_AllHaveDomain(t *testing.T) {
	stem := &rules.StemFile{
		Path: "docs/.stem",
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Domain: "lifecycle_state", Source: "docs/.stem"},
			"titulo": {Type: "string", Domain: "title", Source: "docs/.stem"},
		},
	}
	got := DetectMissingDomains(stem)
	if len(got) != 0 {
		t.Fatalf("expected 0 inferences, got %d", len(got))
	}
}

func TestDetectMissingDomains_NilStem(t *testing.T) {
	got := DetectMissingDomains(nil)
	if len(got) != 0 {
		t.Fatalf("expected 0 inferences for nil stem, got %d", len(got))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/infer/ -run TestDetectMissingDomains -v`
Expected: FAIL — `DetectMissingDomains` undefined

- [ ] **Step 3: Write minimal implementation**

```go
// internal/infer/domain_coverage.go
package infer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pablontiv/rootline/internal/rules"
)

// DetectMissingDomains flags schema fields that lack a domain declaration.
// Fields of type "section" are skipped — they don't carry semantic meaning.
func DetectMissingDomains(stem *rules.StemFile) []Inference {
	if stem == nil || len(stem.Schema) == 0 {
		return nil
	}

	var inferences []Inference

	// Sort field names for deterministic output.
	names := make([]string, 0, len(stem.Schema))
	for name := range stem.Schema {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		sf := stem.Schema[name]
		if sf.Domain != "" {
			continue
		}
		if sf.Type == "section" {
			continue
		}

		msg := fmt.Sprintf("Field %q", name)
		if sf.Type != "" {
			msg += fmt.Sprintf(" (type: %s", sf.Type)
			if len(sf.Values) > 0 {
				msg += fmt.Sprintf(", values: [%s]", strings.Join(sf.Values, ", "))
			}
			msg += ")"
		}
		msg += " has no domain declared"

		inferences = append(inferences, Inference{
			Type:    "missing_domain",
			Source:  sf.Source,
			Field:   name,
			Message: msg,
		})
	}

	return inferences
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/infer/ -run TestDetectMissingDomains -v`
Expected: PASS (all 4 tests)

- [ ] **Step 5: Commit**

```bash
git add internal/infer/domain_coverage.go internal/infer/domain_coverage_test.go
git commit -m "feat(infer): add domain coverage governance detector"
```

---

### Task 2: Schema Coverage Detector

**Files:**
- Create: `internal/infer/schema_coverage.go`
- Create: `internal/infer/schema_coverage_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/infer/schema_coverage_test.go
package infer

import (
	"os"
	"path/filepath"
	"testing"
)

func setupSchemaCoverageDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	// Create .git marker (WalkUp boundary)
	os.MkdirAll(filepath.Join(root, ".git"), 0o755)

	// Directory WITH .stem and markdown
	governed := filepath.Join(root, "governed")
	os.MkdirAll(governed, 0o755)
	os.WriteFile(filepath.Join(governed, ".stem"), []byte("version: 2\nschema:\n  estado:\n    type: string\n"), 0o644)
	os.WriteFile(filepath.Join(governed, "doc.md"), []byte("---\nestado: ok\n---\n"), 0o644)

	// Directory WITHOUT .stem but WITH markdown
	ungoverned := filepath.Join(root, "ungoverned")
	os.MkdirAll(ungoverned, 0o755)
	os.WriteFile(filepath.Join(ungoverned, "doc1.md"), []byte("---\ntitle: one\n---\n"), 0o644)
	os.WriteFile(filepath.Join(ungoverned, "doc2.md"), []byte("---\ntitle: two\n---\n"), 0o644)

	// Directory without any markdown (should be ignored)
	empty := filepath.Join(root, "assets")
	os.MkdirAll(empty, 0o755)
	os.WriteFile(filepath.Join(empty, "logo.png"), []byte("binary"), 0o644)

	return root
}

func TestDetectMissingSchemata_FindsUngoverned(t *testing.T) {
	root := setupSchemaCoverageDir(t)
	got := DetectMissingSchemata(root)

	var missing []Inference
	for _, inf := range got {
		if inf.Type == "missing_schema" {
			missing = append(missing, inf)
		}
	}
	if len(missing) != 1 {
		t.Fatalf("expected 1 missing_schema inference, got %d", len(missing))
	}
	if missing[0].Source != filepath.Join(root, "ungoverned") {
		t.Errorf("expected source %s, got %s", filepath.Join(root, "ungoverned"), missing[0].Source)
	}
}

func TestDetectMissingSchemata_IgnoresGoverned(t *testing.T) {
	root := setupSchemaCoverageDir(t)
	got := DetectMissingSchemata(root)

	for _, inf := range got {
		if inf.Source == filepath.Join(root, "governed") {
			t.Errorf("governed directory should not produce inferences, got: %s", inf.Message)
		}
	}
}

func TestDetectMissingSchemata_IgnoresNonMarkdown(t *testing.T) {
	root := setupSchemaCoverageDir(t)
	got := DetectMissingSchemata(root)

	for _, inf := range got {
		if inf.Source == filepath.Join(root, "assets") {
			t.Errorf("non-markdown directory should not produce inferences, got: %s", inf.Message)
		}
	}
}

func TestDetectMissingSchemata_ImplicitSchema(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".git"), 0o755)
	// .stem at root level
	os.WriteFile(filepath.Join(root, ".stem"), []byte("version: 2\nschema:\n  x:\n    type: string\n"), 0o644)
	// Deeply nested dir — 3 levels below .stem
	deep := filepath.Join(root, "a", "b", "c")
	os.MkdirAll(deep, 0o755)
	os.WriteFile(filepath.Join(deep, "doc.md"), []byte("---\nx: val\n---\n"), 0o644)

	got := DetectMissingSchemata(root)
	found := false
	for _, inf := range got {
		if inf.Type == "implicit_schema" && inf.Source == deep {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected implicit_schema inference for deeply nested directory")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/infer/ -run TestDetectMissingSchemata -v`
Expected: FAIL — `DetectMissingSchemata` undefined

- [ ] **Step 3: Write minimal implementation**

```go
// internal/infer/schema_coverage.go
package infer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pablontiv/rootline/internal/rules"
)

// DetectMissingSchemata walks directories from scanRoot and flags those
// containing markdown files but governed by no .stem schema.
func DetectMissingSchemata(scanRoot string) []Inference {
	var inferences []Inference
	var dirs []string

	// Collect all directories recursively.
	filepath.WalkDir(scanRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == ".stemignore" || (len(name) > 0 && name[0] == '.') {
				return filepath.SkipDir
			}
			dirs = append(dirs, path)
		}
		return nil
	})

	sort.Strings(dirs)

	for _, dir := range dirs {
		mdCount := countMarkdownFiles(dir)
		if mdCount == 0 {
			continue
		}

		entries, err := rules.WalkUp(dir)
		if err != nil || len(entries) == 0 {
			inferences = append(inferences, Inference{
				Type:    "missing_schema",
				Source:  dir,
				Message: fmt.Sprintf("Directory contains %d markdown file(s) but no .stem schema (checked walk-up to .git root)", mdCount),
			})
			continue
		}

		// Check inheritance distance: count levels between dir and closest stem.
		// WalkUp returns entries in root-to-leaf order: entries[0] is root-most,
		// entries[len-1] is leaf-most (physically closest to the target directory).
		// For inheritance distance, we want the leaf-most stem.
		closest := entries[len(entries)-1].Path
		stemDir := filepath.Dir(closest)
		relPath, _ := filepath.Rel(stemDir, dir)
		depth := strings.Count(relPath, string(os.PathSeparator))
		if depth >= 3 {
			inferences = append(inferences, Inference{
				Type:    "implicit_schema",
				Source:  dir,
				Value:   closest,
				Message: fmt.Sprintf("Directory inherits schema from %s (%d levels up) — consider adding local .stem for explicit governance", closest, depth),
			})
		}
	}

	return inferences
}

func countMarkdownFiles(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			count++
		}
	}
	return count
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/infer/ -run TestDetectMissingSchemata -v`
Expected: PASS (all 4 tests)

- [ ] **Step 5: Commit**

```bash
git add internal/infer/schema_coverage.go internal/infer/schema_coverage_test.go
git commit -m "feat(infer): add schema coverage governance detector"
```

---

### Task 3: Validation Gaps Detector

**Files:**
- Create: `internal/infer/validation_gaps.go`
- Create: `internal/infer/validation_gaps_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/infer/validation_gaps_test.go
package infer

import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestDetectValidationGaps_EnumWithoutValues(t *testing.T) {
	stem := &rules.StemFile{
		Path: "docs/.stem",
		Schema: map[string]rules.SchemaField{
			"prioridad": {Type: "enum", Source: "docs/.stem"},
		},
	}
	got := DetectValidationGaps(stem, nil, nil)
	found := false
	for _, inf := range got {
		if inf.Type == "enum_without_values" && inf.Field == "prioridad" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected enum_without_values inference for 'prioridad'")
	}
}

func TestDetectValidationGaps_EnumWithValues_NoGap(t *testing.T) {
	stem := &rules.StemFile{
		Path: "docs/.stem",
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"draft", "active"}, Source: "docs/.stem"},
		},
	}
	got := DetectValidationGaps(stem, nil, nil)
	for _, inf := range got {
		if inf.Type == "enum_without_values" && inf.Field == "estado" {
			t.Error("should not flag enum with values")
		}
	}
}

func TestDetectValidationGaps_UntypedField(t *testing.T) {
	stem := &rules.StemFile{
		Path: "docs/.stem",
		Schema: map[string]rules.SchemaField{
			"mystery": {Source: "docs/.stem"},
		},
	}
	got := DetectValidationGaps(stem, nil, nil)
	found := false
	for _, inf := range got {
		if inf.Type == "untyped_field" && inf.Field == "mystery" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected untyped_field inference for 'mystery'")
	}
}

func TestDetectValidationGaps_SequenceIncomplete(t *testing.T) {
	stem := &rules.StemFile{
		Path: "docs/.stem",
		Schema: map[string]rules.SchemaField{
			"id": {Type: "sequence", Prefix: "T", Source: "docs/.stem"},
			// Missing Digits
		},
	}
	got := DetectValidationGaps(stem, nil, nil)
	found := false
	for _, inf := range got {
		if inf.Type == "sequence_incomplete" && inf.Field == "id" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected sequence_incomplete inference for 'id'")
	}
}

func TestDetectValidationGaps_RequiredUnderstatement(t *testing.T) {
	stem := &rules.StemFile{
		Path: "docs/.stem",
		Schema: map[string]rules.SchemaField{
			"tipo": {Type: "string", Source: "docs/.stem"},
		},
	}
	// 10 records, 9 have 'tipo' — 90% usage but not required
	records := make([]*extract.Record, 10)
	for i := range records {
		fm := map[string]any{}
		if i < 9 {
			fm["tipo"] = "feature"
		}
		records[i] = &extract.Record{Path: "doc.md", Frontmatter: fm}
	}
	got := DetectValidationGaps(stem, records, nil)
	found := false
	for _, inf := range got {
		if inf.Type == "required_understatement" && inf.Field == "tipo" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected required_understatement inference for 'tipo'")
	}
}

func TestDetectValidationGaps_NilStem(t *testing.T) {
	got := DetectValidationGaps(nil, nil)
	if len(got) != 0 {
		t.Fatalf("expected 0 inferences for nil stem, got %d", len(got))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/infer/ -run TestDetectValidationGaps -v`
Expected: FAIL — `DetectValidationGaps` undefined

- [ ] **Step 3: Write minimal implementation**

```go
// internal/infer/validation_gaps.go
package infer

import (
	"fmt"
	"sort"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

const requiredUsageThreshold = 0.80

// DetectValidationGaps checks schema fields for insufficient validation rules.
// Emits distinct inference types: enum_without_values, untyped_field,
// sequence_incomplete, required_understatement.
//
// priorInferences contains results from data-inference detectors that ran
// before this governance detector. Used for deduplication: we skip fields
// already covered by enum_values or required_field inferences.
func DetectValidationGaps(stem *rules.StemFile, records []*extract.Record, priorInferences []Inference) []Inference {
	if stem == nil || len(stem.Schema) == 0 {
		return nil
	}

	// Build deduplication index: fields already flagged by data detectors.
	coveredByEnum := make(map[string]bool)    // fields with enum_values inferences
	coveredByRequired := make(map[string]bool) // fields with required_field inferences
	for _, inf := range priorInferences {
		switch inf.Type {
		case "enum_values":
			coveredByEnum[inf.Field] = true
		case "required_field":
			coveredByRequired[inf.Field] = true
		}
	}

	var inferences []Inference

	names := make([]string, 0, len(stem.Schema))
	for name := range stem.Schema {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		sf := stem.Schema[name]

		// Enum without values — skip if data detector already inferred values.
		if sf.Type == "enum" && len(sf.Values) == 0 && !coveredByEnum[name] {
			inferences = append(inferences, Inference{
				Type:    "enum_without_values",
				Source:  sf.Source,
				Field:   name,
				Message: fmt.Sprintf("Field %q is declared as enum but has no values list — cannot validate", name),
			})
		}

		// Untyped field (no type and no domain).
		if sf.Type == "" && sf.Domain == "" {
			inferences = append(inferences, Inference{
				Type:    "untyped_field",
				Source:  sf.Source,
				Field:   name,
				Message: fmt.Sprintf("Field %q has no type and no domain — rootline cannot validate it", name),
			})
		}

		// Sequence incomplete.
		if sf.Type == "sequence" && (sf.Prefix == "" || sf.Digits == 0) {
			missing := ""
			if sf.Prefix == "" && sf.Digits == 0 {
				missing = "prefix and digits"
			} else if sf.Prefix == "" {
				missing = "prefix"
			} else {
				missing = "digits"
			}
			inferences = append(inferences, Inference{
				Type:    "sequence_incomplete",
				Source:  sf.Source,
				Field:   name,
				Message: fmt.Sprintf("Field %q is type sequence but missing %s", name, missing),
			})
		}
	}

	// Required understatement — needs records.
	if len(records) >= 3 {
		total := len(records)
		for _, name := range names {
			sf := stem.Schema[name]
			if sf.Required || coveredByRequired[name] {
				continue
			}
			if sf.Type == "section" {
				continue
			}
			count := 0
			for _, rec := range records {
				if _, ok := rec.Frontmatter[name]; ok {
					count++
				}
			}
			ratio := float64(count) / float64(total)
			if ratio >= requiredUsageThreshold {
				inferences = append(inferences, Inference{
					Type:    "required_understatement",
					Source:  sf.Source,
					Field:   name,
					Message: fmt.Sprintf("Field %q is used in %d/%d records (%.0f%%) but not declared required", name, count, total, ratio*100),
				})
			}
		}
	}

	return inferences
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/infer/ -run TestDetectValidationGaps -v`
Expected: PASS (all 6 tests)

- [ ] **Step 5: Commit**

```bash
git add internal/infer/validation_gaps.go internal/infer/validation_gaps_test.go
git commit -m "feat(infer): add validation gaps governance detector"
```

---

### Task 4: Structural Hygiene — Naming Inconsistency

**Files:**
- Modify: `internal/infer/structural.go`
- Modify: `internal/infer/structural_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/infer/structural_test.go`. First, add `"fmt"` to the import block (existing imports are `"os"`, `"path/filepath"`, `"testing"`):

```go
import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)
```

Then add the tests:

```go
func TestDetectStructural_NamingInconsistency(t *testing.T) {
	root := t.TempDir()
	// Dominant pattern: E##-slug (8 of 10)
	for i := 1; i <= 8; i++ {
		name := fmt.Sprintf("E%02d-feature-%d", i, i)
		mkDir(t, root, []string{name}, nil)
	}
	// Outliers (2 of 10)
	mkDir(t, root, []string{"notes"}, nil)
	mkDir(t, root, []string{"archive"}, nil)

	got := DetectStructural(root)
	found := false
	for _, inf := range got {
		if inf.Type == "naming_inconsistency" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected naming_inconsistency inference for outliers")
	}
}

func TestDetectStructural_NoNamingInconsistency_AllMatch(t *testing.T) {
	root := t.TempDir()
	for i := 1; i <= 5; i++ {
		name := fmt.Sprintf("E%02d-feature-%d", i, i)
		mkDir(t, root, []string{name}, nil)
	}
	got := DetectStructural(root)
	for _, inf := range got {
		if inf.Type == "naming_inconsistency" {
			t.Error("should not flag when all children match")
		}
	}
}

func TestDetectStructural_NoNamingInconsistency_NoPattern(t *testing.T) {
	root := t.TempDir()
	// No dominant pattern — all different
	mkDir(t, root, []string{"alpha", "beta", "gamma", "delta"}, nil)
	got := DetectStructural(root)
	for _, inf := range got {
		if inf.Type == "naming_inconsistency" {
			t.Error("should not flag when no dominant pattern exists")
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/infer/ -run TestDetectStructural_Naming -v`
Expected: FAIL — no `naming_inconsistency` type emitted

- [ ] **Step 3: Implement naming inconsistency detection**

Add to `internal/infer/structural.go`, inside `DetectStructural` before the return, or as a helper called from it. The pattern detection uses regex to identify `E##-*`, `F##-*`, `S###-*`, `T###-*` and similar prefix patterns:

```go
// Add to structural.go — naming inconsistency detection.
// detectNamingInconsistencies checks for mixed naming conventions
// among children of a directory. If >= 70% match a dominant pattern
// but outliers exist, flag them.

import "regexp"

var namingPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^[A-Z]\d{2,3}-.+$`),  // E01-foo, F02-bar, S001-baz, T001-qux
	regexp.MustCompile(`^\d{2,3}-.+$`),         // 01-foo, 001-bar
}

const namingConsistencyThreshold = 0.70

func detectNamingInconsistencies(dir string, children []os.DirEntry) []Inference {
	if len(children) < 3 {
		return nil
	}

	// Count matches per pattern.
	type patternResult struct {
		pattern  *regexp.Regexp
		matches  []string
		outliers []string
	}

	var best *patternResult
	for _, pat := range namingPatterns {
		pr := &patternResult{pattern: pat}
		for _, c := range children {
			if !c.IsDir() || (len(c.Name()) > 0 && c.Name()[0] == '.') {
				continue
			}
			if pat.MatchString(c.Name()) {
				pr.matches = append(pr.matches, c.Name())
			} else {
				pr.outliers = append(pr.outliers, c.Name())
			}
		}
		total := len(pr.matches) + len(pr.outliers)
		if total < 3 {
			continue
		}
		ratio := float64(len(pr.matches)) / float64(total)
		if ratio >= namingConsistencyThreshold && len(pr.outliers) > 0 {
			if best == nil || len(pr.matches) > len(best.matches) {
				best = pr
			}
		}
	}

	if best == nil {
		return nil
	}

	total := len(best.matches) + len(best.outliers)
	return []Inference{{
		Type:    "naming_inconsistency",
		Source:  dir,
		Value:   best.pattern.String(),
		Message: fmt.Sprintf("%d/%d children match pattern; outliers: %v", len(best.matches), total, best.outliers),
	}}
}
```

Call `detectNamingInconsistencies` from within `DetectStructural` using `filepath.WalkDir` for full-depth recursion. **Important:** Place this code BEFORE the existing early return `if len(subdirs) < structuralMinSampleSize { return nil }` (line ~38 in structural.go), so naming checks run even when the root has <3 subdirectories — deeper directories may still have naming inconsistencies:

```go
// Inside DetectStructural, right after reading entries from os.ReadDir:
// Full-depth naming consistency check (runs independently of min sample size).
var namingInfs []Inference
filepath.WalkDir(scanRoot, func(path string, d os.DirEntry, err error) error {
	if err != nil || !d.IsDir() {
		return nil
	}
	name := d.Name()
	if len(name) > 0 && name[0] == '.' {
		return filepath.SkipDir
	}
	children, readErr := os.ReadDir(path)
	if readErr != nil {
		return nil
	}
	namingInfs = append(namingInfs, detectNamingInconsistencies(path, children)...)
	return nil
})

// Append naming results to the main result slice. If the existing logic
// returns nil early (< 3 subdirs), naming results are still returned.
// Change the early return to: if len(subdirs) < ... { return namingInfs }
// instead of { return nil }.
```

Update the early return at line ~38 from `return nil` to `return namingInfs` so naming checks are always returned:

```go
if len(subdirs) < structuralMinSampleSize {
	return namingInfs // was: return nil
}
```

And at the end of `DetectStructural`, before the final `return result`, append naming results:

```go
result = append(result, namingInfs...)
return result
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/infer/ -run TestDetectStructural_Naming -v`
Expected: PASS (all 3 tests)

- [ ] **Step 5: Run all structural tests to check for regressions**

Run: `go test ./internal/infer/ -run TestDetectStructural -v`
Expected: All existing + new tests PASS

- [ ] **Step 6: Commit**

```bash
git add internal/infer/structural.go internal/infer/structural_test.go
git commit -m "feat(infer): add naming inconsistency detection to structural detector"
```

---

### Task 5: Register Governance Detectors in analyze.go

**Files:**
- Modify: `cmd/rootline/analyze.go:35-38` (agentRequiredTypes)
- Modify: `cmd/rootline/analyze.go:87-127` (categories slice)

- [ ] **Step 1: Add agent-required types**

In `cmd/rootline/analyze.go`, extend the `agentRequiredTypes` map (around line 35):

```go
var agentRequiredTypes = map[string]bool{
	"informal_dependency_candidate": true,
	"unverified_traceability":       true,
	// Governance detectors
	"missing_domain":          true,
	"implicit_schema":         true,
	"naming_inconsistency":    true,
	"enum_without_values":     true,
	"required_understatement": true,
}
```

- [ ] **Step 2: Add governance categories to the categories slice**

After the existing `{"structural", ...}` entry (around line 127), add:

```go
{"domain_coverage", "Domain Coverage", func() []infer.Inference {
	return infer.DetectMissingDomains(stem)
}},
{"schema_coverage", "Schema Coverage", func() []infer.Inference {
	return infer.DetectMissingSchemata(root)
}},
{"validation_gaps", "Validation Gaps", func() []infer.Inference {
	// Collect prior inferences from data detectors for deduplication.
	var prior []infer.Inference
	for _, cat := range report.Categories {
		for _, inf := range cat.Inferences {
			prior = append(prior, infer.Inference{
				Type: inf.Type, Field: inf.Field, Value: inf.Value,
			})
		}
	}
	return infer.DetectValidationGaps(stem, records, prior)
}},
```

Note: `stem` is the merged `StemFile` already resolved in `runAnalyze`. If the variable is named differently, use whatever the existing code provides. The `stem` variable is used for `--incremental` filtering (around line 131-137); the same stem is the input for domain coverage and validation gaps.

- [ ] **Step 3: Build and verify**

Run: `go build ./cmd/rootline/`
Expected: Builds without errors

- [ ] **Step 4: Smoke test**

Run: `go run ./cmd/rootline/ analyze --help`
Expected: Help output — no changes to flags/interface

- [ ] **Step 5: Commit**

```bash
git add cmd/rootline/analyze.go
git commit -m "feat(infer): register governance detectors in analyze pipeline"
```

---

### Task 6: Incremental Filtering for Governance Types

**Files:**
- Modify: `internal/infer/delta.go:24-53` (isCovered switch)
- Modify: `internal/infer/delta_test.go`

- [ ] **Step 1: Write failing tests**

Add to `internal/infer/delta_test.go` (or create if doesn't exist):

```go
func TestIsCovered_MissingDomain(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Domain: "lifecycle_state"},
		},
	}
	covered := isCovered(Inference{Type: "missing_domain", Field: "estado"}, stem)
	if !covered {
		t.Error("missing_domain should be covered when field has domain")
	}

	notCovered := isCovered(Inference{Type: "missing_domain", Field: "titulo"}, stem)
	if notCovered {
		t.Error("missing_domain should not be covered for unknown field")
	}
}

func TestIsCovered_EnumWithoutValues(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"draft", "active"}},
		},
	}
	covered := isCovered(Inference{Type: "enum_without_values", Field: "estado"}, stem)
	if !covered {
		t.Error("enum_without_values should be covered when field has values")
	}
}

func TestIsCovered_UntypedField(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"titulo": {Type: "string"},
		},
	}
	covered := isCovered(Inference{Type: "untyped_field", Field: "titulo"}, stem)
	if !covered {
		t.Error("untyped_field should be covered when field has type")
	}
}

func TestIsCovered_SequenceIncomplete(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"id": {Type: "sequence", Prefix: "T", Digits: 3},
		},
	}
	covered := isCovered(Inference{Type: "sequence_incomplete", Field: "id"}, stem)
	if !covered {
		t.Error("sequence_incomplete should be covered when prefix and digits set")
	}
}

func TestIsCovered_RequiredUnderstatement(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"tipo": {Type: "string", Required: true},
		},
	}
	covered := isCovered(Inference{Type: "required_understatement", Field: "tipo"}, stem)
	if !covered {
		t.Error("required_understatement should be covered when field is required")
	}
}
```

Note: `missing_schema`, `implicit_schema`, and `naming_inconsistency` do NOT need `isCovered` cases. `missing_schema`/`implicit_schema` are filtered at detection time (the detector checks WalkUp and won't emit if a .stem exists). `naming_inconsistency` always runs fresh against filesystem state. Only field-level governance types need `isCovered`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/infer/ -run TestIsCovered_ -v`
Expected: FAIL — new types not handled in switch

- [ ] **Step 3: Add switch cases to isCovered**

In `internal/infer/delta.go`, inside the `isCovered` function switch, add after the existing cases:

```go
case "missing_domain":
	if sf, ok := stem.Schema[inf.Field]; ok && sf.Domain != "" {
		return true
	}

case "enum_without_values":
	if sf, ok := stem.Schema[inf.Field]; ok && sf.Type == "enum" && len(sf.Values) > 0 {
		return true
	}

case "untyped_field":
	if sf, ok := stem.Schema[inf.Field]; ok && (sf.Type != "" || sf.Domain != "") {
		return true
	}

case "sequence_incomplete":
	if sf, ok := stem.Schema[inf.Field]; ok && sf.Prefix != "" && sf.Digits > 0 {
		return true
	}

case "required_understatement":
	if sf, ok := stem.Schema[inf.Field]; ok && sf.Required {
		return true
	}

// Note: missing_schema and implicit_schema use Source (directory path),
// not Field. They cannot be filtered by stem schema lookup since the
// whole point is that no stem exists. These types are NOT added to
// isCovered — they are filtered at the pipeline level in analyze.go
// via the existing --incremental logic: if a .stem file now exists
// at the directory, the schema_coverage detector won't emit the
// inference in the first place (it checks WalkUp at detection time).
// So no isCovered case is needed — the detector itself handles it.

// naming_inconsistency is also not filtered here — structural checks
// always run fresh against current filesystem state.
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/infer/ -run TestIsCovered -v`
Expected: PASS (all new + existing tests)

- [ ] **Step 5: Commit**

```bash
git add internal/infer/delta.go internal/infer/delta_test.go
git commit -m "feat(infer): add incremental filtering for governance inference types"
```

---

### Task 7: Apply Handlers — Schema Fixes and ScaffoldSchema

**Files:**
- Modify: `internal/infer/apply.go:56-80` (switch cases)
- Modify: `cmd/rootline/apply.go:55-63` (pre-phase for ScaffoldSchema)

- [ ] **Step 1: Write failing test for apply handlers**

Add to existing `internal/infer/apply_test.go` (or create if needed):

```go
func TestApplySchemaInferences_UntypedField(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	os.WriteFile(stemPath, []byte("version: 2\nschema:\n  mystery:\n    required: true\n"), 0o644)

	result, err := ApplySchemaInferences(stemPath, []ReportInference{
		{Type: "untyped_field", Field: "mystery", Value: "string"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Applied) == 0 {
		t.Fatal("expected applied changes")
	}

	// Read back and verify type was added.
	data, _ := os.ReadFile(stemPath)
	if !strings.Contains(string(data), "type: string") {
		t.Errorf("expected type: string in stem, got:\n%s", data)
	}
}

func TestApplySchemaInferences_SequenceIncomplete(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	os.WriteFile(stemPath, []byte("version: 2\nschema:\n  id:\n    type: sequence\n"), 0o644)

	result, err := ApplySchemaInferences(stemPath, []ReportInference{
		{Type: "sequence_incomplete", Field: "id", Value: "T:3"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Applied) == 0 {
		t.Fatal("expected applied changes")
	}

	data, _ := os.ReadFile(stemPath)
	content := string(data)
	if !strings.Contains(content, "prefix:") || !strings.Contains(content, "digits:") {
		t.Errorf("expected prefix and digits in stem, got:\n%s", content)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/infer/ -run TestApplySchemaInferences_ -v`
Expected: FAIL — new types not handled

- [ ] **Step 3: Add switch cases to ApplySchemaInferences**

In `internal/infer/apply.go`, inside the switch on `inf.Type` (around line 56-80), add:

```go
case "untyped_field":
	if applyFieldTypeNode(&doc, stem, inf) {
		result.Applied = append(result.Applied, fmt.Sprintf("type: %s → %s", inf.Field, inf.Value))
		modified = true
	}

case "sequence_incomplete":
	if applySequenceCompleteNode(&doc, stem, inf) {
		result.Applied = append(result.Applied, fmt.Sprintf("sequence: %s completed", inf.Field))
		modified = true
	}
```

Add `"strconv"` to the import block of `internal/infer/apply.go` (existing imports: `"fmt"`, `"os"`, `"path/filepath"`, `"strings"`, `rules`, `yaml`).

Write `applySequenceCompleteNode` — a helper that adds `prefix` and `digits` to a sequence field's YAML node. The inference's `Value` field encodes prefix and digits as `"PREFIX:DIGITS"` (e.g., `"T:3"`):

```go
func applySequenceCompleteNode(doc *yaml.Node, stem *rules.StemFile, inf ReportInference) bool {
	parts := strings.SplitN(inf.Value, ":", 2)
	if len(parts) != 2 {
		return false
	}
	prefix := parts[0]
	digits, err := strconv.Atoi(parts[1])
	if err != nil {
		return false
	}

	fieldNode := findSchemaFieldNode(doc, inf.Field)
	if fieldNode == nil {
		return false
	}

	// Use existing setFieldProperty helper (apply.go:304-319).
	// Note: setFieldProperty returns void — it always succeeds if
	// fieldNode is a valid MappingNode (guaranteed by findSchemaFieldNode).
	setFieldProperty(fieldNode, "prefix", prefix)
	setFieldProperty(fieldNode, "digits", strconv.Itoa(digits))
	return true
}
```

Note: `findSchemaFieldNode`, `hasKey`, and `appendKeyValue` are helpers that may already exist or need to be written. Follow the pattern of existing `applyEnumExtensionNode` which navigates the YAML node tree.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/infer/ -run TestApplySchemaInferences_ -v`
Expected: PASS

- [ ] **Step 5: Add ScaffoldSchema pre-phase to cmd/rootline/apply.go**

In `cmd/rootline/apply.go`, before the `WalkUp` call (around line 55), add a pre-phase that processes `missing_schema` inferences:

```go
// Pre-phase: scaffold .stem for directories that have none.
for _, cat := range report.Categories {
	for _, inf := range cat.Inferences {
		if inf.Type != "missing_schema" || inf.RequiresAgent {
			continue
		}
		if err := infer.ScaffoldSchema(inf.Source); err != nil {
			fmt.Fprintf(os.Stderr, "scaffold %s: %v\n", inf.Source, err)
		} else {
			fmt.Fprintf(os.Stdout, "scaffolded %s/.stem\n", inf.Source)
		}
	}
}
```

Write `ScaffoldSchema` in `internal/infer/scaffold.go`:

```go
// internal/infer/scaffold.go
package infer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
)

// ScaffoldSchema creates a minimal version-2 .stem file at dirPath by
// reading markdown files directly and collecting their frontmatter fields.
// Does not use index.Scan (which requires context and scope resolver) —
// instead extracts frontmatter directly using the extract registry.
func ScaffoldSchema(dirPath string) error {
	reg := extract.NewRegistry()

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return err
	}

	fields := make(map[string]bool)
	found := false
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		fpath := filepath.Join(dirPath, e.Name())
		// Registry.ForFile returns an Extractor for the file type.
		// Extractor.Extract takes (path, content) and returns a Record.
		// Pattern from index/index.go:118-128.
		ext := reg.ForFile(fpath, "")
		if ext == nil {
			continue
		}
		content, readErr := os.ReadFile(fpath)
		if readErr != nil {
			continue
		}
		rec, extractErr := ext.Extract(fpath, content)
		if extractErr != nil || rec == nil {
			continue
		}
		found = true
		for key := range rec.Frontmatter {
			fields[key] = true
		}
	}

	if !found {
		return fmt.Errorf("no markdown records found in %s", dirPath)
	}

	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)

	var b strings.Builder
	b.WriteString("version: 2\nschema:\n")
	for _, name := range names {
		b.WriteString(fmt.Sprintf("  %s:\n    type: string\n", name))
	}

	stemPath := filepath.Join(dirPath, ".stem")
	return os.WriteFile(stemPath, []byte(b.String()), 0o644)
}
```

The extraction pattern follows `index/index.go:118-128`: `reg.ForFile()` → `ext.Extract(path, content)`. The key contract is: read markdown files in directory → union their frontmatter keys → write `.stem` with those fields as `type: string`.

- [ ] **Step 6: Run all apply tests**

Run: `go test ./internal/infer/ -run TestApply -v && go build ./cmd/rootline/`
Expected: All PASS, build succeeds

- [ ] **Step 7: Commit**

```bash
git add internal/infer/apply.go internal/infer/scaffold.go cmd/rootline/apply.go
git commit -m "feat(infer): add apply handlers for governance inferences and ScaffoldSchema"
```

---

### Task 8: E2E Test — Full Governance Pipeline

**Files:**
- Create: `internal/e2e/governance_test.go`

- [ ] **Step 1: Update `runAnalyze` helper in `internal/e2e/analyze_test.go`**

The existing `runAnalyze` helper only registers the original 10+ categories. Add governance categories to the helper so E2E tests can see them. After the existing `report.AddCategory` calls, add:

```go
// Governance detectors
report.AddCategory("domain_coverage", "Domain Coverage", infer.DetectMissingDomains(stem), agentTypes)
report.AddCategory("schema_coverage", "Schema Coverage", infer.DetectMissingSchemata(root), agentTypes)

var priorInfs []infer.Inference
for _, cat := range report.Categories {
	for _, ri := range cat.Inferences {
		priorInfs = append(priorInfs, infer.Inference{Type: ri.Type, Field: ri.Field, Value: ri.Value})
	}
}
report.AddCategory("validation_gaps", "Validation Gaps", infer.DetectValidationGaps(stem, records, priorInfs), agentTypes)
```

Where `stem` is the merged stem from the resolver and `agentTypes` includes the governance types. Update the `agentTypes` map in the helper to include:
```go
"missing_domain": true, "implicit_schema": true, "naming_inconsistency": true,
"enum_without_values": true, "required_understatement": true,
```

- [ ] **Step 2: Write E2E test**

```go
// internal/e2e/governance_test.go
package e2e

import (
	"testing"

	"github.com/pablontiv/rootline/internal/infer"
)

func TestGovernance_AnalyzePipeline(t *testing.T) {
	root := setupProject(t, map[string]string{
		// Governed directory with .stem but missing domains
		".stem": "version: 2\nschema:\n  estado:\n    type: enum\n    values: [draft, active, closed]\n  tipo:\n    type: string\n",
		"doc1.md": "---\nestado: draft\ntipo: feature\n---\n# Doc 1\n",
		"doc2.md": "---\nestado: active\ntipo: feature\n---\n# Doc 2\n",
		"doc3.md": "---\nestado: closed\ntipo: feature\n---\n# Doc 3\n",
		// Ungoverned subdirectory (no .stem, markdown files)
		"ungoverned/note1.md": "---\ntitle: Note 1\n---\n# Note\n",
		"ungoverned/note2.md": "---\ntitle: Note 2\n---\n# Note\n",
	})

	report := runAnalyze(t, root)

	// Verify domain_coverage category exists.
	domainCat := findCategory(report, "domain_coverage")
	if domainCat == nil {
		t.Fatal("expected domain_coverage category in report")
	}
	// Should flag estado (no domain) and tipo (no domain)
	if domainCat.InferenceCount < 2 {
		t.Errorf("expected at least 2 missing_domain inferences, got %d", domainCat.InferenceCount)
	}

	// Verify schema_coverage category exists.
	schemaCat := findCategory(report, "schema_coverage")
	if schemaCat == nil {
		t.Fatal("expected schema_coverage category in report")
	}
	// Should flag ungoverned/ directory.
	found := false
	for _, inf := range schemaCat.Inferences {
		if inf.Type == "missing_schema" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected missing_schema inference for ungoverned/ directory")
	}

	// Verify agent gating.
	for _, inf := range domainCat.Inferences {
		if !inf.RequiresAgent {
			t.Errorf("missing_domain should be agent-required, got RequiresAgent=false for %s", inf.Field)
		}
	}
}

func findCategory(report *infer.AnalyzeReport, id string) *infer.CategoryResult {
	for i := range report.Categories {
		if report.Categories[i].ID == id {
			return &report.Categories[i]
		}
	}
	return nil
}
```

Note: This test reuses the existing `setupProject` and `runAnalyze` helpers from `internal/e2e/`. The `runAnalyze` helper must be updated to include the governance categories in its detector list — or the test calls the detectors directly. Adapt based on how `runAnalyze` is structured.

- [ ] **Step 2: Run E2E test**

Run: `go test ./internal/e2e/ -run TestGovernance -v`
Expected: PASS

- [ ] **Step 3: Run full test suite**

Run: `go test ./... -race`
Expected: All tests PASS, no regressions

- [ ] **Step 4: Commit**

```bash
git add internal/e2e/governance_test.go
git commit -m "test(e2e): add governance pipeline integration test"
```

---

### Task 9: Final Verification

- [ ] **Step 1: Run lint and build**

Run: `just check`
Expected: PASS — no lint errors, build succeeds

- [ ] **Step 2: Run full test suite with race detector**

Run: `just test`
Expected: All tests PASS

- [ ] **Step 3: Manual smoke test**

Run: `go run ./cmd/rootline/ analyze docs/epics/ --output json | jq '.categories[] | select(.id | startswith("domain_") or startswith("schema_") or startswith("validation_"))'`
Expected: Governance categories appear with real inferences for docs/epics/

- [ ] **Step 4: Verify incremental filtering**

Run: `go run ./cmd/rootline/ analyze --incremental docs/epics/ --output json | jq '.categories[] | select(.id | startswith("domain_") or startswith("schema_") or startswith("validation_")) | .inference_count'`
Expected: Counts reflect filtered (uncovered) inferences only

- [ ] **Step 5: Final commit (if any cleanup needed)**

Stage only specific files that were modified during cleanup — do not use `git add -A`. Then:

```bash
git commit -m "chore: cleanup governance detector implementation"
```
