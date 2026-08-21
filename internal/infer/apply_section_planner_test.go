package infer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/rules"
)

func TestApplySchemaInferences_RejectsBatchSectionSourceConflictsDeterministically(t *testing.T) {
	orders := [][]ReportInference{
		{{Type: "required_section", Field: "notes", SourceDirective: `body.section["## A"]`}, {Type: "optional_section", Field: "notes", SourceDirective: `body.section["## B"]`}},
		{{Type: "optional_section", Field: "notes", SourceDirective: `body.section["## B"]`}, {Type: "required_section", Field: "notes", SourceDirective: `body.section["## A"]`}},
	}
	var firstErr string
	for i, inferences := range orders {
		stemPath := writeApplyStem(t, "version: 2\nschema: {}\n")
		errText := applyExpectErrorNoChange(t, stemPath, inferences)
		if i == 0 {
			firstErr = errText
		} else if errText != firstErr {
			t.Fatalf("errors differ by order:\n%s\n%s", firstErr, errText)
		}
	}
}

func TestApplySchemaInferences_RejectsConflictingSectionBatchSourcesWithoutExistingComparison(t *testing.T) {
	orders := [][]ReportInference{
		{{Type: "required_section", Field: "notes", SourceDirective: `body.section["## A"]`}, {Type: "optional_section", Field: "notes", SourceDirective: `body.section["## B"]`}},
		{{Type: "optional_section", Field: "notes", SourceDirective: `body.section["## B"]`}, {Type: "required_section", Field: "notes", SourceDirective: `body.section["## A"]`}},
	}
	const stem = "version: 2\nschema:\n  notes:\n    type: string\n    source: 'body.section[\"## A\"]'\n"
	const wantErr = `section inference for "notes" has conflicting sources: body.section["## A"], body.section["## B"]`
	var firstErr string
	for i, inferences := range orders {
		stemPath := writeApplyStem(t, stem)
		errText := applyExpectErrorNoChange(t, stemPath, inferences)
		if errText != wantErr {
			t.Fatalf("error = %q, want %q", errText, wantErr)
		}
		if strings.Contains(errText, "existing schema") {
			t.Fatalf("conflicting batch compared retained source to existing schema: %q", errText)
		}
		if i == 0 {
			firstErr = errText
		} else if errText != firstErr {
			t.Fatalf("errors differ by order:\n%s\n%s", firstErr, errText)
		}
	}
}

func TestApplySchemaInferences_RejectsThreeSectionBatchSourcesWithOneFullSetDiagnostic(t *testing.T) {
	permutations := [][]ReportInference{
		{{Type: "optional_section", Field: "notes", SourceDirective: `body.section["## A"]`}, {Type: "required_section", Field: "notes", SourceDirective: `body.section["## B"]`}, {Type: "optional_section", Field: "notes", SourceDirective: `body.section["## C"]`}},
		{{Type: "optional_section", Field: "notes", SourceDirective: `body.section["## C"]`}, {Type: "required_section", Field: "notes", SourceDirective: `body.section["## B"]`}, {Type: "optional_section", Field: "notes", SourceDirective: `body.section["## A"]`}},
		{{Type: "required_section", Field: "notes", SourceDirective: `body.section["## B"]`}, {Type: "optional_section", Field: "notes", SourceDirective: `body.section["## A"]`}, {Type: "optional_section", Field: "notes", SourceDirective: `body.section["## C"]`}},
	}
	const wantErr = `section inference for "notes" has conflicting sources: body.section["## A"], body.section["## B"], body.section["## C"]`
	var firstErr string
	for i, inferences := range permutations {
		stemPath := writeApplyStem(t, "version: 2\nschema: {}\n")
		errText := applyExpectErrorNoChange(t, stemPath, inferences)
		if errText != wantErr {
			t.Fatalf("error = %q, want %q", errText, wantErr)
		}
		if got := strings.Count(errText, `section inference for "notes" has conflicting sources`); got != 1 {
			t.Fatalf("diagnostic count = %d in %q, want exactly one", got, errText)
		}
		if i == 0 {
			firstErr = errText
		} else if errText != firstErr {
			t.Fatalf("errors differ by permutation:\n%s\n%s", firstErr, errText)
		}
	}
}

func TestApplySchemaInferences_RejectsSameFieldSectionAndNonsectionIntents(t *testing.T) {
	for _, nonsection := range []ReportInference{
		{Type: "field_type", Field: "notes", Value: "integer"},
		{Type: "required_field", Field: "notes"},
		{Type: "constant_field", Field: "notes", Value: "draft"},
		{Type: "enum_values", Field: "notes", Value: "[a b]"},
		{Type: "sequence_incomplete", Field: "notes", Value: "RL-:3"},
	} {
		var firstErr string
		for i, inferences := range [][]ReportInference{
			{nonsection, {Type: "optional_section", Field: "notes", SourceDirective: `body.section["## Notes"]`}},
			{{Type: "optional_section", Field: "notes", SourceDirective: `body.section["## Notes"]`}, nonsection},
		} {
			stemPath := writeApplyStem(t, "version: 2\nschema: {}\n")
			errText := applyExpectErrorNoChange(t, stemPath, inferences)
			if i == 0 {
				firstErr = errText
			} else if errText != firstErr {
				t.Fatalf("%s errors differ by order: %q vs %q", nonsection.Type, firstErr, errText)
			}
		}
	}
}

func TestApplySchemaInferences_SameSourceSectionBatchConvergesRequiredBothOrders(t *testing.T) {
	var firstStem string
	for i, inferences := range [][]ReportInference{
		{{Type: "optional_section", Field: "notes", SourceDirective: `body.section["## Notes"]`}, {Type: "required_section", Field: "notes", SourceDirective: `body.section["## Notes"]`}},
		{{Type: "required_section", Field: "notes", SourceDirective: `body.section["## Notes"]`}, {Type: "optional_section", Field: "notes", SourceDirective: `body.section["## Notes"]`}},
	} {
		stemPath := writeApplyStem(t, "version: 2\nschema: {}\n")
		if _, err := ApplySchemaInferences(stemPath, inferences, false); err != nil {
			t.Fatalf("same-source batch rejected: %v", err)
		}
		field := readApplyStem(t, stemPath).Schema["notes"]
		if !field.Required || field.Extract != `body.section["## Notes"]` || field.Type != "string" {
			t.Fatalf("same-source batch produced %+v", field)
		}
		if stem := string(mustReadApplyFile(t, stemPath)); i == 0 {
			firstStem = stem
		} else if stem != firstStem {
			t.Fatalf("same-source schemas differ by order:\n%s\n%s", firstStem, stem)
		}
	}
}

func writeApplyStem(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, ".stem")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readApplyStem(t *testing.T, path string) *rules.StemFile {
	t.Helper()
	stem, err := rules.ParseStem(path, mustReadApplyFile(t, path))
	if err != nil {
		t.Fatalf("parse stem: %v", err)
	}
	return stem
}

func applyExpectErrorNoChange(t *testing.T, stemPath string, inferences []ReportInference) string {
	t.Helper()
	before := string(mustReadApplyFile(t, stemPath))
	_, err := ApplySchemaInferences(stemPath, inferences, false)
	if err == nil {
		t.Fatal("expected apply error")
	}
	if after := string(mustReadApplyFile(t, stemPath)); after != before {
		t.Fatalf("rejected apply mutated stem:\nbefore:\n%s\nafter:\n%s", before, after)
	}
	return err.Error()
}

func mustReadApplyFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
