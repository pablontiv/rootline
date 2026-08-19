package rules

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func mustWriteStemTestFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestValidateStemHealth_ValidStems(t *testing.T) {
	dir := t.TempDir()
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: string
    required: true
  tipo:
    type: enum
    values: [a, b]
`))
	mustWriteStemTestFile(t, filepath.Join(dir, "test.md"), []byte("---\nestado: draft\n---\n# Test\n"))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, c := range result.Checks {
		if c.Status == "fail" {
			t.Errorf("unexpected failure: %s - %s", c.Name, c.Message)
		}
	}
}

func TestValidateStemHealth_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte("{{invalid yaml"))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, c := range result.Checks {
		if c.Name == "yaml-valid" && c.Status == "fail" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected yaml-valid fail check")
	}
}

func TestValidateStemHealth_OrphanScope(t *testing.T) {
	dir := t.TempDir()
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
scope:
  match: "*.xyz"
`))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, c := range result.Checks {
		if c.Name == "scope-match" && c.Status == "warn" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected scope-match warning")
	}
}

func TestValidateStemHealth_ScopeMatchWithMatchingFile(t *testing.T) {
	dir := t.TempDir()
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
scope:
  match: "*.md"
`))
	mustWriteStemTestFile(t, filepath.Join(dir, "readme.md"), []byte("# Hello"))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, c := range result.Checks {
		if c.Name == "scope-match" {
			t.Error("unexpected scope-match check when files match")
		}
	}
}

func TestValidateStemHealth_SingleValueEnumHasNoWarning(t *testing.T) {
	dir := t.TempDir()
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
schema:
  estado:
    type: enum
    values: [OnlyOne]
`))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, c := range result.Checks {
		if c.Name == "enum-values" || c.Name == "field-declaration" {
			t.Errorf("unexpected enum declaration warning for one-value enum: %+v", c)
		}
	}
}

func TestValidateStemHealth_EnumWithTwoValues_NoWarning(t *testing.T) {
	dir := t.TempDir()
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
schema:
  estado:
    type: enum
    values: [A, B]
`))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, c := range result.Checks {
		if c.Name == "enum-values" {
			t.Error("unexpected enum-values warning for 2-value enum")
		}
	}
}

func TestValidateStemHealth_TypeConsistency(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
schema:
  estado:
    type: string
`))
	sub := filepath.Join(dir, "sub")
	mustWriteStemTestFile(t, filepath.Join(sub, ".stem"), []byte(`version: 2
schema:
  estado:
    type: enum
    values: [A, B]
`))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, c := range result.Checks {
		if c.Name == "type-consistency" && c.Status == "fail" {
			found = true
			if c.Field != "estado" {
				t.Errorf("expected field 'estado', got %q", c.Field)
			}
			break
		}
	}
	if !found {
		t.Error("expected type-consistency failure")
	}
}

func TestValidateStemHealth_RuleFieldExists(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
schema:
  estado:
    type: string
validate:
  - rule: non_empty
    field: nonexistent_field
    severity: error
`))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, c := range result.Checks {
		if c.Name == "rule-field-exists" && c.Status == "warn" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected rule-field-exists warning")
	}
}

func TestValidateStemHealth_FieldOverride(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
schema:
  estado:
    type: string
`))
	sub := filepath.Join(dir, "sub")
	mustWriteStemTestFile(t, filepath.Join(sub, ".stem"), []byte(`version: 2
schema:
  estado:
    type: string
    required: true
`))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, c := range result.Checks {
		if c.Name == "field-override" && c.Status == "warn" {
			found = true
			if c.Field != "estado" {
				t.Errorf("expected field 'estado', got %q", c.Field)
			}
			break
		}
	}
	if !found {
		t.Error("expected field-override warning")
	}
}

func TestValidateStemHealth_AggregatedRequired(t *testing.T) {
	dir := t.TempDir()
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Completado]
aggregate:
  estado: "len(filter(descendants, {.estado == 'Completado'})) == len(descendants) ? 'Completado' : 'Pending'"
`))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, c := range result.Checks {
		if c.Name == "aggregated-required" && c.Status == "warn" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected aggregated-required warning")
	}
}

func TestValidateStemHealth_RequiredWithoutAggregate_NoWarning(t *testing.T) {
	dir := t.TempDir()
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Completado]
`))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, c := range result.Checks {
		if c.Name == "aggregated-required" {
			t.Error("unexpected aggregated-required warning")
		}
	}
}

func TestValidateStemHealth_AggregateWithoutRequired_NoWarning(t *testing.T) {
	dir := t.TempDir()
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
schema:
  estado:
    type: enum
    values: [Pending, Completado]
aggregate:
  estado: "some_expr"
`))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, c := range result.Checks {
		if c.Name == "aggregated-required" {
			t.Error("unexpected aggregated-required warning")
		}
	}
}

func TestValidateStemHealth_NoStems(t *testing.T) {
	dir := t.TempDir()

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, c := range result.Checks {
		if c.Name == "stem-files-exist" && c.Status == "warn" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected stem-files-exist warning")
	}
}

func TestValidateStemHealth_SkipsGitDir(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	mustWriteStemTestFile(t, filepath.Join(gitDir, ".stem"), []byte(`version: 2`))
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
schema:
  x:
    type: string
`))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stemCount := 0
	for _, c := range result.Checks {
		if c.Name == "yaml-valid" {
			stemCount++
		}
	}
	if stemCount != 1 {
		t.Errorf("expected 1 yaml-valid check, got %d", stemCount)
	}
}

func TestValidateStemHealth_EnumZeroValues(t *testing.T) {
	dir := t.TempDir()
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
schema:
  estado:
    type: enum
`))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, c := range result.Checks {
		if c.Name == "field-declaration" && c.Status == "fail" && c.Field == "estado" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected field-declaration failure for 0-value enum")
	}
}

func TestValidateStemHealth_V1RejectedAtParse(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
schema:
  titulo:
    type: string
`))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// v1 stems now fail at parse time (yaml-valid check)
	found := false
	for _, c := range result.Checks {
		if c.Name == "yaml-valid" && c.Status == "fail" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected yaml-valid fail check for v1 stem")
	}
}

func TestStemHealth_FormulaCompleteness_Complete(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
schema:
  estado:
    type: enum
    values: [Pending, Completed, Blocked]
aggregate:
  estado: |
    all(descendants, {.estado == "Completed"}) ? "Completed" :
    any(descendants, {.estado == "Blocked"}) ? "Blocked" : "Pending"
`))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, c := range result.Checks {
		if c.Name == "aggregate-formula-coverage" {
			t.Errorf("unexpected formula-coverage warning for complete formula: %s", c.Message)
		}
	}
}

func TestStemHealth_FormulaCompleteness_Incomplete(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
schema:
  estado:
    type: enum
    values: [Pending, Completed, Obsolete]
aggregate:
  estado: |
    all(descendants, {.estado == "Completed"}) ? "Completed" : "Pending"
`))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, c := range result.Checks {
		if c.Name == "aggregate-formula-coverage" && c.Status == "warn" {
			found = true
			if c.Field != "estado" {
				t.Errorf("expected field 'estado', got %q", c.Field)
			}
			break
		}
	}
	if !found {
		t.Error("expected aggregate-formula-coverage warning for missing Obsolete")
	}
}

func TestStemHealth_FormulaCompleteness_NonEnumField(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
schema:
  score:
    type: string
aggregate:
  score: "sum(descendants, {.score})"
`))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, c := range result.Checks {
		if c.Name == "aggregate-formula-coverage" {
			t.Errorf("unexpected formula-coverage check for non-enum field: %s", c.Message)
		}
	}
}

func TestStemHealth_FormulaCompleteness_NoAggregate(t *testing.T) {
	dir := t.TempDir()
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
schema:
  estado:
    type: enum
    values: [Pending, Completed]
`))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, c := range result.Checks {
		if c.Name == "aggregate-formula-coverage" {
			t.Errorf("unexpected formula-coverage check for stem without aggregate: %s", c.Message)
		}
	}
}

func TestValidateStemHealth_MultipleStems(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
schema:
  titulo:
    type: string
`))
	sub := filepath.Join(dir, "sub")
	mustWriteStemTestFile(t, filepath.Join(sub, ".stem"), []byte(`version: 2
schema:
  prioridad:
    type: string
`))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	yamlChecks := 0
	for _, c := range result.Checks {
		if c.Name == "yaml-valid" && c.Status == "pass" {
			yamlChecks++
		}
	}
	if yamlChecks != 2 {
		t.Errorf("expected 2 yaml-valid pass checks, got %d", yamlChecks)
	}
}

func TestValidateStemHealth_MonotonicViolation_TypeWidening(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	// Parent: estado is enum
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
schema:
  estado:
    type: enum
    values: [draft, active, completed]
`))
	// Child: estado is widened to string (violation!)
	sub := filepath.Join(dir, "sub")
	mustWriteStemTestFile(t, filepath.Join(sub, ".stem"), []byte(`version: 2
schema:
  estado:
    type: string
`))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, c := range result.Checks {
		if c.Name == "monotonic-violations" && c.Status == "fail" {
			found = true
			if c.Field != "estado" {
				t.Errorf("expected field 'estado', got %q", c.Field)
			}
			break
		}
	}
	if !found {
		t.Error("expected monotonic-violations error for type widening (enum → string)")
	}
}

func TestValidateStemHealth_MonotonicNarrowing_AllowedNoViolation(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	// Parent: estado is string
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
schema:
  estado:
    type: string
`))
	// Child: estado is narrowed to enum (allowed!)
	sub := filepath.Join(dir, "sub")
	mustWriteStemTestFile(t, filepath.Join(sub, ".stem"), []byte(`version: 2
schema:
  estado:
    type: enum
    values: [draft, active]
`))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, c := range result.Checks {
		if c.Name == "monotonic-violations" && c.Field == "estado" {
			t.Errorf("unexpected monotonic-violations error for valid narrowing (string → enum): %s", c.Message)
		}
	}
}

func TestValidateStemHealth_MonotonicViolation_BatchOutput(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	// Parent: required: true
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
schema:
  titulo:
    type: string
    required: true
`))
	// Child: required: false (violation!)
	sub := filepath.Join(dir, "sub")
	mustWriteStemTestFile(t, filepath.Join(sub, ".stem"), []byte(`version: 2
schema:
  titulo:
    type: string
    required: false
`))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, c := range result.Checks {
		if c.Name == "monotonic-violations" && c.Status == "fail" {
			found = true
			if c.Field != "titulo" {
				t.Errorf("expected field 'titulo', got %q", c.Field)
			}
			break
		}
	}
	if !found {
		t.Error("expected monotonic-violations error in batch output for required loosening")
	}
}

func TestValidateStemHealth_UnknownCheckKeys(t *testing.T) {
	dir := t.TempDir()
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
links:
  checks:
    cicles: true
`))
	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var found *StemHealthCheck
	for i, c := range result.Checks {
		if c.Name == "unknown-check-keys" {
			found = &result.Checks[i]
			break
		}
	}
	if found == nil {
		t.Fatal("expected unknown-check-keys check")
	}
	if found.Status != "warn" {
		t.Errorf("status = %q, want warn", found.Status)
	}
	if found.Field != "cicles" {
		t.Errorf("field = %q, want cicles", found.Field)
	}
	if !strings.Contains(found.Message, `did you mean "cycles"?`) {
		t.Errorf("message = %q, want cycles suggestion", found.Message)
	}
}

func TestValidateStemHealth_KnownCheckKeysNoWarn(t *testing.T) {
	dir := t.TempDir()
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
links:
  checks:
    resolve: true
    cycles: true
`))
	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, c := range result.Checks {
		if c.Name == "unknown-check-keys" {
			t.Errorf("unexpected unknown-check-keys check: %+v", c)
		}
	}
}

// Slice 2: Nested-root-marker health check
func TestValidateStemHealth_NestedRootMarker(t *testing.T) {
	dir := t.TempDir()

	// Create /repo/.stem with root: true
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
root: true
scope:
  match: "*.md"
`))

	// Create /repo/docs/.stem also with root: true
	docsDir := filepath.Join(dir, "docs")
	mustWriteStemTestFile(t, filepath.Join(docsDir, ".stem"), []byte(`version: 2
root: true
scope:
  match: "*.md"
`))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find the nested-root-marker check
	var found *StemHealthCheck
	for i, c := range result.Checks {
		if c.Name == "nested-root-marker" {
			found = &result.Checks[i]
			break
		}
	}

	if found == nil {
		t.Fatal("expected nested-root-marker check")
	}

	if found.Status != "info" {
		t.Errorf("status = %q, want info", found.Status)
	}

	// Should report the nested marker directory
	if !strings.Contains(found.Path, "docs") {
		t.Errorf("path = %q, expected to contain 'docs'", found.Path)
	}

	// Message should mention root marker and inheritance narrowing
	if !strings.Contains(found.Message, "root: true") || !strings.Contains(found.Message, "do not inherit") {
		t.Errorf("message = %q, want to contain 'root: true' and 'do not inherit'", found.Message)
	}
}

// TestStemHealthDocumentationDrift asserts that the stem-health check names
// documented in docs/validate.md match the Name: literals emitted by
// ValidateStemHealth in stemhealth.go.
//
// The code is the authoritative source: names are extracted from stemhealth.go
// with the Go AST rather than a regex so the test survives formatting changes,
// and without exporting a names list from production code.
//
// Documentation format contract (docs/validate.md): the section starts on a
// line containing "**Stem Health**", each check is a list item whose first
// backtick-wrapped token is the check name, and the section ends at the next
// numbered list item.
func TestStemHealthDocumentationDrift(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed; cannot locate test file")
	}
	testDir := filepath.Dir(thisFile)

	codeNames := stemHealthNamesFromSource(t, filepath.Join(testDir, "stemhealth.go"))
	if len(codeNames) == 0 {
		t.Fatal("no check names extracted from stemhealth.go; AST parsing may have failed")
	}

	repoRoot := filepath.Dir(filepath.Dir(testDir))
	docNames := stemHealthNamesFromDoc(t, filepath.Join(repoRoot, "docs", "validate.md"))

	for name := range codeNames {
		if !docNames[name] {
			t.Errorf("check %q exists in stemhealth.go but is not documented in docs/validate.md", name)
		}
	}
	for name := range docNames {
		if !codeNames[name] {
			t.Errorf("check %q is documented in docs/validate.md but does not exist in stemhealth.go", name)
		}
	}
	if len(codeNames) != len(docNames) {
		t.Errorf("check count mismatch: stemhealth.go has %d, docs/validate.md lists %d", len(codeNames), len(docNames))
	}
}

// stemHealthNamesFromSource returns the distinct Name: string literals assigned
// inside composite literals in the given Go source file.
func stemHealthNamesFromSource(t *testing.T, path string) map[string]bool {
	t.Helper()

	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}

	names := make(map[string]bool)
	ast.Inspect(astFile, func(node ast.Node) bool {
		lit, ok := node.(*ast.CompositeLit)
		if !ok {
			return true
		}
		for _, elt := range lit.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok || key.Name != "Name" {
				continue
			}
			val, ok := kv.Value.(*ast.BasicLit)
			if !ok || val.Kind != token.STRING {
				continue
			}
			name, err := strconv.Unquote(val.Value)
			if err != nil {
				t.Fatalf("unquoting %s: %v", val.Value, err)
			}
			names[name] = true
		}
		return true
	})
	return names
}

// stemHealthNamesFromDoc returns the check names listed in the Stem Health
// section of the given markdown file.
func stemHealthNamesFromDoc(t *testing.T, path string) map[string]bool {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	itemRe := regexp.MustCompile("^\\s*-\\s+`([^`]+)`")
	nextItemRe := regexp.MustCompile(`^\s*\d+\.\s`)

	names := make(map[string]bool)
	inSection := false
	for _, line := range strings.Split(string(content), "\n") {
		if strings.Contains(line, "**Stem Health**") {
			inSection = true
			continue
		}
		if !inSection {
			continue
		}
		if nextItemRe.MatchString(line) {
			break
		}
		if m := itemRe.FindStringSubmatch(line); m != nil {
			names[m[1]] = true
		}
	}

	if len(names) == 0 {
		t.Fatalf("no check names parsed from %s; the Stem Health section format may have changed", path)
	}
	return names
}
