package rules

import (
	"context"
	"os"
	"path/filepath"
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
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
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
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
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
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
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

func TestValidateStemHealth_EnumValues(t *testing.T) {
	dir := t.TempDir()
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
schema:
  estado:
    type: enum
    values: [OnlyOne]
`))

	result, err := ValidateStemHealth(context.Background(), dir)
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

func TestValidateStemHealth_EnumWithTwoValues_NoWarning(t *testing.T) {
	dir := t.TempDir()
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
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
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
schema:
  estado:
    type: string
`))
	sub := filepath.Join(dir, "sub")
	mustWriteStemTestFile(t, filepath.Join(sub, ".stem"), []byte(`version: 1
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
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
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
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
schema:
  estado:
    type: string
`))
	sub := filepath.Join(dir, "sub")
	mustWriteStemTestFile(t, filepath.Join(sub, ".stem"), []byte(`version: 1
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
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
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
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
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
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
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
	mustWriteStemTestFile(t, filepath.Join(gitDir, ".stem"), []byte(`version: 1`))
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
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
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
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
		if c.Name == "enum-values" && c.Status == "warn" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected enum-values warning for 0-value enum")
	}
}

func TestValidateStemHealth_LevelsChildrenValid(t *testing.T) {
	dir := t.TempDir()
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
levels:
  epic:
    match: "E*"
    children: [feature]
  feature:
    match: "F*"
    children: [nonexistent]
`))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, c := range result.Checks {
		if c.Name == "levels-children-valid" && c.Status == "fail" {
			found = true
			if c.Field != "levels.feature.children" {
				t.Errorf("expected field 'levels.feature.children', got %q", c.Field)
			}
			break
		}
	}
	if !found {
		t.Error("expected levels-children-valid failure for nonexistent child")
	}
}

func TestValidateStemHealth_LevelsChildrenValid_AllExist(t *testing.T) {
	dir := t.TempDir()
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
levels:
  epic:
    match: "E*"
    children: [feature]
  feature:
    match: "F*"
    children: [story]
  story:
    match: "S*"
`))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, c := range result.Checks {
		if c.Name == "levels-children-valid" {
			t.Errorf("unexpected levels-children-valid check: %s", c.Message)
		}
	}
}

func TestValidateStemHealth_LevelsNoCycles(t *testing.T) {
	dir := t.TempDir()
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
levels:
  epic:
    match: "E*"
    children: [feature]
  feature:
    match: "F*"
    children: [epic]
`))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, c := range result.Checks {
		if c.Name == "levels-no-cycles" && c.Status == "fail" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected levels-no-cycles failure for circular children")
	}
}

func TestValidateStemHealth_LevelsNoCycles_Valid(t *testing.T) {
	dir := t.TempDir()
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
levels:
  epic:
    match: "E*"
    children: [feature]
  feature:
    match: "F*"
    children: [story]
  story:
    match: "S*"
`))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, c := range result.Checks {
		if c.Name == "levels-no-cycles" {
			t.Errorf("unexpected levels-no-cycles check: %s", c.Message)
		}
	}
}

func TestValidateStemHealth_NoLevels_SkipsChecks(t *testing.T) {
	dir := t.TempDir()
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
schema:
  estado:
    type: string
`))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, c := range result.Checks {
		if c.Name == "levels-children-valid" || c.Name == "levels-no-cycles" {
			t.Errorf("unexpected levels check %q when no levels defined", c.Name)
		}
	}
}

func TestValidateStemHealth_MultipleStems(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
schema:
  titulo:
    type: string
`))
	sub := filepath.Join(dir, "sub")
	mustWriteStemTestFile(t, filepath.Join(sub, ".stem"), []byte(`version: 1
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
