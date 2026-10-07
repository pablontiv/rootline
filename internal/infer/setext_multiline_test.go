package infer

import (
	"context"
	"reflect"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

const multilineSetextSource = `body.section["## First\nSecond #"]`

func TestDetectSectionPatterns_MultilineSetext(t *testing.T) {
	record := makeRecord("First\nSecond #\n---\n\nContent\n")
	sections := extract.ExtractSections(record.AST, []byte(record.Body))
	if len(sections) != 1 || len(sections[0].Path) != 1 {
		t.Fatalf("sections = %+v, want one section path", sections)
	}
	wantPath := extract.SectionPath{{Level: 2, Text: "First\nSecond #"}}
	if !reflect.DeepEqual(sections[0].Path, wantPath) {
		t.Fatalf("section path = %+v, want %+v", sections[0].Path, wantPath)
	}

	inferences, err := DetectSectionPatterns([]*extract.Record{record}, 1)
	if err != nil {
		t.Fatalf("DetectSectionPatterns rejected a valid multiline Setext heading: %v", err)
	}
	inf, ok := findSectionInference(inferences, "first_second")
	if !ok || inf.Type != "required_section" || inf.SourceDirective != multilineSetextSource {
		t.Fatalf("inference = %+v, present = %v", inf, ok)
	}

	parsed, err := extract.ParseBodySource(inf.SourceDirective)
	if err != nil {
		t.Fatalf("ParseBodySource rejected the inferred selector: %v", err)
	}
	if !reflect.DeepEqual(extract.SectionPath(parsed.Selector), sections[0].Path) {
		t.Fatalf("parsed selector = %+v, want section path %+v", parsed.Selector, sections[0].Path)
	}
	value, present, err := extract.ResolveBodyValue(record, inf.SourceDirective)
	if err != nil || !present || value != "Content" {
		t.Fatalf("resolved value = %q, present = %v, error = %v", value, present, err)
	}
}

func TestGenerateFlatSchema_MultilineSetext(t *testing.T) {
	record := makeRecord("First\nSecond #\n---\n\nContent\n")
	opts := DefaultInferOptions()
	opts.IncludeStructural = false
	opts.SectionThreshold = 1
	stem, err := GenerateFlatSchema(context.Background(), ".", []*extract.Record{record}, opts)
	if err != nil {
		t.Fatalf("GenerateFlatSchema rejected a valid multiline Setext heading: %v", err)
	}
	field, ok := stem.Schema["first_second"]
	if !ok || field.Type != "string" || !field.Required || field.Extract != multilineSetextSource {
		t.Fatalf("generated field = %+v, present = %v", field, ok)
	}
	if errs := rules.Validate(context.Background(), record, stem); len(errs) != 0 {
		t.Fatalf("generated schema rejected its source record: %+v", errs)
	}
}

func TestApplySchemaInferences_MultilineSetext(t *testing.T) {
	record := makeRecord("First\nSecond #\n---\n\nContent\n")
	inferences, err := DetectSectionPatterns([]*extract.Record{record}, 1)
	if err != nil {
		t.Fatalf("DetectSectionPatterns returned an error: %v", err)
	}
	inf, ok := findSectionInference(inferences, "first_second")
	if !ok {
		t.Fatalf("multiline Setext inference is missing: %+v", inferences)
	}

	stemPath := writeApplyStem(t, "version: 2\nschema: {}\n")
	result, err := ApplySchemaInferences(stemPath, []ReportInference{{
		Type:            inf.Type,
		Field:           inf.Field,
		SourceDirective: inf.SourceDirective,
	}}, false)
	if err != nil {
		t.Fatalf("ApplySchemaInferences rejected the inferred selector: %v", err)
	}
	if len(result.Applied) != 1 {
		t.Fatalf("applied actions = %v, want one action", result.Applied)
	}
	field := readApplyStem(t, stemPath).Schema["first_second"]
	if field.Type != "string" || !field.Required || field.Extract != multilineSetextSource {
		t.Fatalf("applied field = %+v", field)
	}
	value, present, err := extract.ResolveBodyValue(record, field.Extract)
	if err != nil || !present || value != "Content" {
		t.Fatalf("applied selector resolved value = %q, present = %v, error = %v", value, present, err)
	}
}
