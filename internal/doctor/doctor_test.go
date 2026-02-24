package doctor

import (
	"os"
	"path/filepath"
	"testing"
)

func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestRunChecks_ValidStems(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
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
	mustWriteFile(t, filepath.Join(dir, "test.md"), []byte("---\nestado: draft\n---\n# Test\n"))

	result, err := RunChecks(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Summary.Failed > 0 {
		t.Errorf("expected no failures, got %d", result.Summary.Failed)
	}
	if result.Version != 1 {
		t.Errorf("expected version 1, got %d", result.Version)
	}
	if result.Kind != "rootline/doctor" {
		t.Errorf("expected kind rootline/doctor, got %s", result.Kind)
	}
}

func TestRunChecks_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("{{invalid yaml"))

	result, err := RunChecks(dir)
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

func TestRunChecks_OrphanScope(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
scope:
  match: "*.xyz"
`))

	result, err := RunChecks(dir)
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

func TestRunChecks_ScopeMatchWithMatchingFile(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
scope:
  match: "*.md"
`))
	mustWriteFile(t, filepath.Join(dir, "readme.md"), []byte("# Hello"))

	result, err := RunChecks(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, c := range result.Checks {
		if c.Name == "scope-match" {
			t.Error("unexpected scope-match check when files match")
		}
	}
}

func TestRunChecks_EnumValues(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
schema:
  estado:
    type: enum
    values: [OnlyOne]
`))

	result, err := RunChecks(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, c := range result.Checks {
		if c.Name == "enum-values" && c.Status == "warn" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected enum-values warning")
	}
}

func TestRunChecks_EnumWithTwoValues_NoWarning(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
schema:
  estado:
    type: enum
    values: [A, B]
`))

	result, err := RunChecks(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, c := range result.Checks {
		if c.Name == "enum-values" {
			t.Error("unexpected enum-values warning for 2-value enum")
		}
	}
}

func TestRunChecks_TypeConsistency(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
schema:
  estado:
    type: string
`))
	sub := filepath.Join(dir, "sub")
	mustWriteFile(t, filepath.Join(sub, ".stem"), []byte(`version: 1
schema:
  estado:
    type: enum
    values: [A, B]
`))

	result, err := RunChecks(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, c := range result.Checks {
		if c.Name == "type-consistency" && c.Status == "fail" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected type-consistency failure")
	}
}

func TestRunChecks_RuleFieldExists(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
schema:
  estado:
    type: string
validate:
  - rule: non_empty
    field: nonexistent_field
    severity: error
`))

	result, err := RunChecks(dir)
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

func TestRunChecks_FieldOverride(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
schema:
  estado:
    type: string
`))
	sub := filepath.Join(dir, "sub")
	mustWriteFile(t, filepath.Join(sub, ".stem"), []byte(`version: 1
schema:
  estado:
    type: string
    required: true
`))

	result, err := RunChecks(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, c := range result.Checks {
		if c.Name == "field-override" && c.Status == "warn" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected field-override warning")
	}
}

func TestRunChecks_AggregatedRequired(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Completado]
aggregate:
  estado: "len(filter(descendants, {.estado == 'Completado'})) == len(descendants) ? 'Completado' : 'Pending'"
`))

	result, err := RunChecks(dir)
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

func TestRunChecks_RequiredWithoutAggregate_NoWarning(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Completado]
`))

	result, err := RunChecks(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, c := range result.Checks {
		if c.Name == "aggregated-required" {
			t.Error("unexpected aggregated-required warning")
		}
	}
}

func TestRunChecks_AggregateWithoutRequired_NoWarning(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
schema:
  estado:
    type: enum
    values: [Pending, Completado]
aggregate:
  estado: "some_expr"
`))

	result, err := RunChecks(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, c := range result.Checks {
		if c.Name == "aggregated-required" {
			t.Error("unexpected aggregated-required warning")
		}
	}
}

func TestRunChecks_NoStems(t *testing.T) {
	dir := t.TempDir()

	result, err := RunChecks(dir)
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

func TestRunChecks_SummaryCount(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
schema:
  estado:
    type: string
`))

	result, err := RunChecks(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Summary.Total != len(result.Checks) {
		t.Errorf("summary total %d != len(checks) %d", result.Summary.Total, len(result.Checks))
	}
	sum := result.Summary.Passed + result.Summary.Warnings + result.Summary.Failed
	if sum != result.Summary.Total {
		t.Errorf("summary counts don't add up: %d+%d+%d != %d",
			result.Summary.Passed, result.Summary.Warnings, result.Summary.Failed, result.Summary.Total)
	}
}

func TestRunChecks_SkipsGitDir(t *testing.T) {
	dir := t.TempDir()
	// Put a .stem inside .git — it should be skipped
	gitDir := filepath.Join(dir, ".git")
	mustWriteFile(t, filepath.Join(gitDir, ".stem"), []byte(`version: 1`))
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
schema:
  x:
    type: string
`))

	result, err := RunChecks(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should only find the root .stem, not .git/.stem
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

func TestRunChecks_EnumZeroValues(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
schema:
  estado:
    type: enum
`))

	result, err := RunChecks(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, c := range result.Checks {
		if c.Name == "enum-values" && c.Status == "warn" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected enum-values warning for 0-value enum")
	}
}

func TestRunChecks_MultipleStems(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
schema:
  titulo:
    type: string
`))
	sub := filepath.Join(dir, "sub")
	mustWriteFile(t, filepath.Join(sub, ".stem"), []byte(`version: 1
schema:
  prioridad:
    type: string
`))

	result, err := RunChecks(dir)
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
