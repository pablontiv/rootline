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
	tests := []struct {
		name        string
		parentField string
		childField  string
		message     string
	}{
		{
			name:        "enum to string",
			parentField: "type: enum\n    values: [A, B]\n",
			childField:  "type: string\n",
			message:     `type changes from "enum" to "string"`,
		},
		{
			name:        "string to list",
			parentField: "type: string\n",
			childField:  "type: list\n",
			message:     `type changes from "string" to "list"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
				t.Fatal(err)
			}
			mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\nschema:\n  estado:\n    "+tt.parentField))
			sub := filepath.Join(dir, "sub")
			mustWriteStemTestFile(t, filepath.Join(sub, ".stem"), []byte("version: 2\nschema:\n  estado:\n    "+tt.childField))

			result, err := ValidateStemHealth(context.Background(), dir)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			assertStemDiagnostics(t, diagnosticsForField(StemHealthDiagnostics(result), "estado"), []StemHealthDiagnostic{{
				Path:     "sub/.stem",
				Check:    "type-consistency",
				Field:    "estado",
				Severity: "error",
				Message:  tt.message,
			}})
		})
	}
}

func TestValidateStemHealth_StringToEnumNarrowingHasNoCompatibilityNoise(t *testing.T) {
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
	assertNoStemHealthCheck(t, result, "type-consistency", "estado")
	assertNoStemHealthCheck(t, result, "field-override", "estado")
	assertNoStemHealthCheck(t, result, "monotonic-violations", "estado")
}

func TestValidateStemHealth_EnumSubsetNarrowingHasNoOverrideNoise(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
schema:
  estado:
    type: enum
    values: [A, B, C]
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
	assertNoStemHealthCheck(t, result, "field-override", "estado")
	assertNoStemHealthCheck(t, result, "monotonic-violations", "estado")
}

func TestValidateStemHealth_SourceIncompatibilityFails(t *testing.T) {
	tests := []struct{ name, parent, child string }{
		{"changed", "source: body.section[\"## Summary\"]\n", "source: body.section[\"## Context\"]\n"},
		{"removed", "source: body.section[\"## Summary\"]\n", "source: null\n"},
		{"added", "", "source: body.h1\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
				t.Fatal(err)
			}
			mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\nschema:\n  summary:\n    type: string\n    "+tt.parent))
			sub := filepath.Join(dir, "sub")
			mustWriteStemTestFile(t, filepath.Join(sub, ".stem"), []byte("version: 2\nschema:\n  summary:\n    type: string\n    "+tt.child))

			result, err := ValidateStemHealth(context.Background(), dir)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			found := false
			for _, c := range result.Checks {
				if c.Name == "monotonic-violations" && c.Status == "fail" && c.Field == "summary" {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected source compatibility failure, got %+v", result.Checks)
			}
		})
	}
}

func TestValidateStemHealth_MonotonicConflictOwnedByDeclaringStemOnly(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
schema:
  title:
    type: string
    required: true
`))
	middle := filepath.Join(dir, "middle")
	mustWriteStemTestFile(t, filepath.Join(middle, ".stem"), []byte(`version: 2
schema:
  title:
    type: string
    required: false
`))
	leaf := filepath.Join(middle, "leaf")
	mustWriteStemTestFile(t, filepath.Join(leaf, ".stem"), []byte(`version: 2
schema:
  other:
    type: string
`))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got []StemHealthDiagnostic
	for _, diag := range StemHealthDiagnostics(result) {
		if diag.Check == "monotonic-violations" && diag.Field == "title" {
			got = append(got, diag)
		}
	}
	assertStemDiagnostics(t, got, []StemHealthDiagnostic{{
		Path:     "middle/.stem",
		Check:    "monotonic-violations",
		Field:    "title",
		Severity: "error",
		Message:  `field "title" loosens required: false`,
	}})
}

func TestValidateStemHealth_DefaultErrorSeveritySurvivesInvalidWarningDescendants(t *testing.T) {
	tests := []struct {
		name       string
		leafSchema string
		wantPaths  []string
	}{
		{
			name: "leaf explicit warning owns second loosening",
			leafSchema: `schema:
  estado:
    type: string
    severity: warn
`,
			wantPaths: []string{"middle/.stem", "middle/leaf/.stem"},
		},
		{
			name: "leaf omission inherits cumulative error without duplicate",
			leafSchema: `schema:
  estado:
    type: string
`,
			wantPaths: []string{"middle/.stem"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
				t.Fatal(err)
			}
			mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
schema:
  estado:
    type: string
`))
			middle := filepath.Join(dir, "middle")
			mustWriteStemTestFile(t, filepath.Join(middle, ".stem"), []byte(`version: 2
schema:
  estado:
    type: string
    severity: warn
`))
			leaf := filepath.Join(middle, "leaf")
			mustWriteStemTestFile(t, filepath.Join(leaf, ".stem"), []byte("version: 2\n"+tt.leafSchema))

			result, err := ValidateStemHealth(context.Background(), dir)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var got []StemHealthDiagnostic
			for _, diag := range StemHealthDiagnostics(result) {
				if diag.Check == "monotonic-violations" && diag.Field == "estado" {
					got = append(got, diag)
				}
			}
			if len(got) != len(tt.wantPaths) {
				t.Fatalf("severity monotonic diagnostic count = %d, want %d: %+v", len(got), len(tt.wantPaths), got)
			}
			for i, wantPath := range tt.wantPaths {
				if got[i].Path != wantPath {
					t.Fatalf("severity conflict[%d] path = %q, want %q: %+v", i, got[i].Path, wantPath, got)
				}
			}

			lr, err := ResolveLayered(filepath.Join(leaf, "doc.md"), dir)
			if err != nil {
				t.Fatalf("ResolveLayered() error: %v", err)
			}
			if got := lr.EffectiveSchema["estado"].Severity; got != "error" {
				t.Fatalf("effective severity = %q, want cumulative default error", got)
			}
		})
	}
}

func assertNoStemHealthCheck(t *testing.T, result *StemHealthResult, name, field string) {
	t.Helper()
	for _, c := range result.Checks {
		if c.Name == name && c.Field == field {
			t.Fatalf("unexpected %s check for %s: %+v", name, field, c)
		}
	}
}

func diagnosticsForField(diags []StemHealthDiagnostic, field string) []StemHealthDiagnostic {
	var out []StemHealthDiagnostic
	for _, diag := range diags {
		if diag.Field == field {
			out = append(out, diag)
		}
	}
	return out
}

func assertStemDiagnostics(t *testing.T, got, want []StemHealthDiagnostic) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("diagnostic count = %d, want %d\ngot:  %+v\nwant: %+v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("diagnostic[%d] = %+v, want %+v", i, got[i], want[i])
		}
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

func TestValidateStemHealth_NullSchemaFieldClassifiedAsSchemaValid(t *testing.T) {
	dir := t.TempDir()
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\nroot: true\n"))
	mustWriteStemTestFile(t, filepath.Join(dir, "child", ".stem"), []byte("version: 2\nschema:\n  removed: null\n"))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got *StemHealthDiagnostic
	for _, diag := range StemHealthDiagnostics(result) {
		if diag.Path == "child/.stem" {
			d := diag
			got = &d
			break
		}
	}
	if got == nil {
		t.Fatalf("expected diagnostic for child/.stem, got %+v", StemHealthDiagnostics(result))
	}
	if got.Check != "schema-valid" {
		t.Fatalf("check = %q, want %q", got.Check, "schema-valid")
	}
	if got.Field != "removed" {
		t.Fatalf("field = %q, want %q", got.Field, "removed")
	}
	if !strings.Contains(got.Message, `schema field "removed" is null`) {
		t.Fatalf("message = %q, want null-field reason", got.Message)
	}
	if strings.Contains(got.Message, dir) {
		t.Fatalf("message leaks absolute path %q: %q", dir, got.Message)
	}
}

func TestValidateStemHealth_MalformedYAMLRemainsYamlValid(t *testing.T) {
	dir := t.TempDir()
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte("version: [broken\n"))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got *StemHealthDiagnostic
	for _, diag := range StemHealthDiagnostics(result) {
		if diag.Path == ".stem" {
			d := diag
			got = &d
			break
		}
	}
	if got == nil {
		t.Fatalf("expected diagnostic for .stem, got %+v", StemHealthDiagnostics(result))
	}
	if got.Check != "yaml-valid" {
		t.Fatalf("check = %q, want %q", got.Check, "yaml-valid")
	}
	if got.Field != "" {
		t.Fatalf("field = %q, want empty", got.Field)
	}
}

func TestValidateStemHealth_V1RejectedAsSchemaValidWithoutField(t *testing.T) {
	dir := t.TempDir()
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 1
schema:
  titulo:
    type: string
`))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got *StemHealthDiagnostic
	for _, diag := range StemHealthDiagnostics(result) {
		if diag.Path == ".stem" {
			d := diag
			got = &d
			break
		}
	}
	if got == nil {
		t.Fatalf("expected diagnostic for .stem, got %+v", StemHealthDiagnostics(result))
	}
	if got.Check != "schema-valid" {
		t.Fatalf("check = %q, want %q", got.Check, "schema-valid")
	}
	if got.Field != "" {
		t.Fatalf("field = %q, want empty", got.Field)
	}
	if !strings.Contains(got.Message, "stem version 1 is no longer supported") {
		t.Fatalf("message = %q, want unsupported-version reason", got.Message)
	}
	if strings.Contains(got.Message, dir) {
		t.Fatalf("message leaks absolute path %q: %q", dir, got.Message)
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

func TestValidateStemHealth_MonotonicViolation_EnumExtension(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
schema:
  estado:
    type: enum
    values: [draft, active]
`))
	sub := filepath.Join(dir, "sub")
	mustWriteStemTestFile(t, filepath.Join(sub, ".stem"), []byte(`version: 2
schema:
  estado:
    type: enum
    values: [draft, active, completed]
`))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertStemDiagnostics(t, diagnosticsForField(StemHealthDiagnostics(result), "estado"), []StemHealthDiagnostic{{
		Path:     "sub/.stem",
		Check:    "monotonic-violations",
		Field:    "estado",
		Severity: "error",
		Message:  `field "estado": enum extended with disallowed value(s): [completed]`,
	}})
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

func TestValidateStemHealth_UnknownLinkRuleKeys(t *testing.T) {
	dir := t.TempDir()
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
links:
  blocks:
    target: "../tasks/*.md"
    targte: "../tasks/*.md"
    field: blocked_by
    valeu_field: blocked_by
    value_field: blocked_by
`))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := map[string]StemHealthCheck{}
	for _, c := range result.Checks {
		if c.Name == "unknown-link-rule-keys" {
			found[c.Field] = c
		}
		if c.Name == "unknown-link-rule-keys" && (c.Field == "target" || c.Field == "field" || c.Field == "value_field") {
			t.Fatalf("valid link rule key unexpectedly reported as unknown: %+v", c)
		}
	}

	if len(found) != 2 {
		t.Fatalf("unknown-link-rule-keys count = %d, want 2; checks=%+v", len(found), result.Checks)
	}

	targte, ok := found["targte"]
	if !ok {
		t.Fatalf("missing unknown-link-rule-keys diagnostic for targte: %+v", found)
	}
	if targte.Status != "warn" {
		t.Errorf("targte status = %q, want warn", targte.Status)
	}
	if !strings.Contains(targte.Message, `links.blocks`) {
		t.Errorf("targte message = %q, want links.blocks context", targte.Message)
	}
	if !strings.Contains(targte.Message, `did you mean "target"?`) {
		t.Errorf("targte message = %q, want target suggestion", targte.Message)
	}

	if got := found["valeu_field"].Message; !strings.Contains(got, `links.blocks`) {
		t.Errorf("valeu_field message = %q, want links.blocks context", got)
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

// TestValidateStemHealth_NullFieldRemovalE2E tests that schema: {field: null} removes fields
// from the effective schema without triggering false stem-health errors. This is the
// end-to-end validation test for issue #183, using the same fixture as the issue report.
func TestValidateStemHealth_NullFieldRemovalE2E(t *testing.T) {
	// Reproduce the issue's fixture:
	// Root .stem: titulo and removed both required:true
	// Child .stem: removes 'removed' field with null
	// Child record: has only titulo

	dir := t.TempDir()

	// Root .stem
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
root: true
scope:
  match: "*.md"
schema:
  titulo:
    type: string
    required: true
  removed:
    type: string
    required: true
`))

	// Parent record (satisfies both fields)
	mustWriteStemTestFile(t, filepath.Join(dir, "p.md"), []byte(`---
titulo: Hello World
removed: x
---

# Parent
`))

	// Child directory with .stem that removes 'removed'
	mustWriteStemTestFile(t, filepath.Join(dir, "child", ".stem"), []byte(`version: 2
scope:
  match: "*.md"
schema:
  removed: null
`))

	// Child record (has only titulo - valid because 'removed' is removed by null)
	mustWriteStemTestFile(t, filepath.Join(dir, "child", "c.md"), []byte(`---
titulo: Hello World
---

# Child
`))

	// Run validate --all (stem health check)
	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify no stem-health errors for the null field removal
	var fieldDeclErrors []string
	var monoViolationErrors []string
	var typeConsistencyErrors []string

	for _, c := range result.Checks {
		if c.Status != "fail" {
			continue
		}
		if c.Name == "field-declaration" && c.Field == "removed" {
			fieldDeclErrors = append(fieldDeclErrors, c.Message)
		}
		if c.Name == "monotonic-violations" && c.Field == "removed" {
			monoViolationErrors = append(monoViolationErrors, c.Message)
		}
		if c.Name == "type-consistency" && c.Field == "removed" {
			typeConsistencyErrors = append(typeConsistencyErrors, c.Message)
		}
	}

	if len(fieldDeclErrors) > 0 {
		t.Errorf("field-declaration errors for removed field (should be none): %v", fieldDeclErrors)
	}
	if len(monoViolationErrors) > 0 {
		t.Errorf("monotonic-violations errors for removed field (should be none): %v", monoViolationErrors)
	}
	if len(typeConsistencyErrors) > 0 {
		t.Errorf("type-consistency errors for removed field (should be none): %v", typeConsistencyErrors)
	}
}

func TestValidateStemHealth_EmptyMapSchemaFieldReportsOnlyFieldDeclaration(t *testing.T) {
	dir := t.TempDir()

	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
root: true
schema:
  removed:
    type: string
    required: true
`))

	mustWriteStemTestFile(t, filepath.Join(dir, "child", ".stem"), []byte(`version: 2
schema:
  removed: {}
`))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var removedFailures []StemHealthCheck
	for _, check := range result.Checks {
		if check.Status == "fail" && check.Field == "removed" {
			removedFailures = append(removedFailures, check)
		}
	}

	if len(removedFailures) != 1 {
		t.Fatalf("failures for field removed = %d, want 1: %+v", len(removedFailures), removedFailures)
	}
	if removedFailures[0].Name != "field-declaration" {
		t.Fatalf("failure check = %q, want field-declaration", removedFailures[0].Name)
	}
	if !strings.Contains(removedFailures[0].Message, "incomplete-type") {
		t.Fatalf("failure message = %q, want incomplete-type", removedFailures[0].Message)
	}

	assertNoStemHealthCheck(t, result, "type-consistency", "removed")
	assertNoStemHealthCheck(t, result, "monotonic-violations", "removed")
}

// TestValidateStemHealth_TypeWideningStillDetected is an adversarial control:
// verify that real type widening (not null removal) still produces errors.
func TestValidateStemHealth_TypeWideningStillDetected(t *testing.T) {
	// Parent declares titulo: string, child widens to list
	dir := t.TempDir()

	// Root .stem
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
root: true
scope:
  match: "*.md"
schema:
  titulo:
    type: string
    required: true
`))

	// Child directory with .stem that widens tipo
	mustWriteStemTestFile(t, filepath.Join(dir, "child", ".stem"), []byte(`version: 2
scope:
  match: "*.md"
schema:
  titulo:
    type: list
    required: true
`))

	// Run validate --all (stem health check)
	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify type-consistency or monotonic-violations errors ARE reported for real widening
	var foundTypeOrMonoError bool
	for _, c := range result.Checks {
		if c.Status == "fail" && c.Field == "titulo" &&
			(c.Name == "type-consistency" || c.Name == "monotonic-violations") {
			foundTypeOrMonoError = true
			break
		}
	}

	if !foundTypeOrMonoError {
		t.Error("expected type-consistency or monotonic-violations error for real type widening, but got none")
	}
}
