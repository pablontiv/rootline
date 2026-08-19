package infer

import "testing"

func TestApplySchemaInferences_AddsCanonicalSectionField(t *testing.T) {
	for _, tt := range []struct {
		name     string
		infType  string
		required bool
	}{
		{name: "required", infType: "required_section", required: true},
		{name: "optional", infType: "optional_section", required: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			stemPath := writeApplyStem(t, "version: 2\nschema: {}\n")
			source := `body.section["## Notes"]`

			result, err := ApplySchemaInferences(stemPath, []ReportInference{{Type: tt.infType, Field: "notes", SourceDirective: source}}, false)
			if err != nil {
				t.Fatalf("apply error: %v", err)
			}
			if len(result.Applied) != 1 {
				t.Fatalf("applied = %v, want one section field action", result.Applied)
			}

			field := readApplyStem(t, stemPath).Schema["notes"]
			if field.Type != "string" || field.Extract != source || field.Required != tt.required {
				t.Fatalf("unexpected field: %+v", field)
			}
		})
	}
}

func TestApplySchemaInferences_SectionSameSourceMergesMonotonically(t *testing.T) {
	source := `body.section["## Notes"]`
	stemPath := writeApplyStem(t, "version: 2\nschema:\n  notes:\n    type: string\n    source: 'body.section[\"## Notes\"]'\n    required: false\n    default: keep\n    severity: warn\n")

	result, err := ApplySchemaInferences(stemPath, []ReportInference{{Type: "required_section", Field: "notes", SourceDirective: source}}, false)
	if err != nil {
		t.Fatalf("tighten required: %v", err)
	}
	if len(result.Applied) != 1 {
		t.Fatalf("applied = %v, want required tightening", result.Applied)
	}
	field := readApplyStem(t, stemPath).Schema["notes"]
	if !field.Required || field.Type != "string" || field.Extract != source || field.Default != "keep" || field.Severity != "warn" {
		t.Fatalf("tightening did not preserve compatible constraints: %+v", field)
	}

	result, err = ApplySchemaInferences(stemPath, []ReportInference{{Type: "optional_section", Field: "notes", SourceDirective: source}}, false)
	if err != nil {
		t.Fatalf("optional merge: %v", err)
	}
	if len(result.Applied) != 0 {
		t.Fatalf("optional inference loosened/applied unexpectedly: %v", result.Applied)
	}
	if field := readApplyStem(t, stemPath).Schema["notes"]; !field.Required {
		t.Fatalf("optional inference loosened required field: %+v", field)
	}
}

func TestApplySchemaInferences_RejectsConflictingSectionSource(t *testing.T) {
	cases := []struct {
		name string
		stem string
	}{
		{name: "different source", stem: "version: 2\nschema:\n  notes:\n    type: string\n    source: 'body.section[\"## Notes\"]'\n"},
		{name: "frontmatter or missing source", stem: "version: 2\nschema:\n  notes:\n    type: string\n"},
		{name: "legacy heading only", stem: "version: 2\nschema:\n  notes:\n    heading: Notes\n"},
		{name: "legacy section type", stem: "version: 2\nschema:\n  notes:\n    type: section\n    source: 'body.section[\"## Notes\"]'\n"},
		{name: "matching source incompatible type", stem: "version: 2\nschema:\n  notes:\n    type: integer\n    source: 'body.section[\"## Context\"]'\n"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			stemPath := writeApplyStem(t, tt.stem)
			before := string(mustReadApplyFile(t, stemPath))
			_, err := ApplySchemaInferences(stemPath, []ReportInference{{Type: "required_section", Field: "notes", SourceDirective: `body.section["## Context"]`}}, false)
			if err == nil {
				t.Fatal("expected section origin conflict")
			}
			if after := string(mustReadApplyFile(t, stemPath)); after != before {
				t.Fatalf("conflict mutated stem:\nbefore:\n%s\nafter:\n%s", before, after)
			}
		})
	}
}

func TestApplySchemaInferences_RejectsMissingSectionSourceBeforeAnyMutation(t *testing.T) {
	stemPath := writeApplyStem(t, "version: 2\nschema:\n  estado:\n    type: string\n")
	applyExpectErrorNoChange(t, stemPath, []ReportInference{
		{Type: "field_type", Field: "owner", Value: "string"},
		{Type: "required_section", Field: "notes"},
	})
}

func TestApplySchemaInferences_RejectsInvalidSectionSourcesBeforeWrite(t *testing.T) {
	for _, source := range []string{`body.h1`, `body.title`, `body.section["## Notes"`, `body.section["Notes"]`} {
		t.Run(source, func(t *testing.T) {
			stemPath := writeApplyStem(t, "version: 2\nschema: {}\n")
			applyExpectErrorNoChange(t, stemPath, []ReportInference{{Type: "required_section", Field: "notes", SourceDirective: source}})
		})
	}
}

func TestApplySchemaInferences_CanonicalizesParseableSectionSource(t *testing.T) {
	stemPath := writeApplyStem(t, "version: 2\nschema: {}\n")
	_, err := ApplySchemaInferences(stemPath, []ReportInference{{Type: "optional_section", Field: "notes", SourceDirective: "body.section[`## Notes`]"}}, false)
	if err != nil {
		t.Fatalf("apply parseable source: %v", err)
	}
	if got, want := readApplyStem(t, stemPath).Schema["notes"].Extract, `body.section["## Notes"]`; got != want {
		t.Fatalf("source = %q, want canonical %q", got, want)
	}
}
