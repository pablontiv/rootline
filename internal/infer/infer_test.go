package infer

import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func makeRecords(fields ...map[string]any) []*extract.Record {
	var records []*extract.Record
	for i, f := range fields {
		records = append(records, &extract.Record{
			Path:        "doc" + string(rune('1'+i)) + ".md",
			Frontmatter: f,
		})
	}
	return records
}

func TestAnalyzeEmpty(t *testing.T) {
	result := Analyze(nil)
	if len(result.Fields) != 0 {
		t.Errorf("expected empty fields, got %d", len(result.Fields))
	}
	if len(result.Schema) != 0 {
		t.Errorf("expected empty schema, got %d", len(result.Schema))
	}
}

func TestAnalyzeRequired(t *testing.T) {
	// 9 out of 10 records have "estado" -> should be required (>80%)
	var records []*extract.Record
	for i := 0; i < 10; i++ {
		fm := map[string]any{}
		if i < 9 {
			fm["estado"] = "Pending"
		}
		records = append(records, &extract.Record{Path: "doc.md", Frontmatter: fm})
	}

	result := Analyze(records)
	sf, exists := result.Schema["estado"]
	if !exists {
		t.Fatal("expected estado in schema")
	}
	if !sf.Required {
		t.Error("expected estado to be required (9/10 = 90%)")
	}
}

func TestAnalyzeNotRequired(t *testing.T) {
	// 5 out of 10 records have "optional" -> not required (50% <= 80%)
	var records []*extract.Record
	for i := 0; i < 10; i++ {
		fm := map[string]any{}
		if i < 5 {
			fm["optional"] = "yes"
		}
		records = append(records, &extract.Record{Path: "doc.md", Frontmatter: fm})
	}

	result := Analyze(records)
	sf := result.Schema["optional"]
	if sf.Required {
		t.Error("expected optional to NOT be required (5/10 = 50%)")
	}
}

func TestAnalyzeEnum(t *testing.T) {
	// 10 records with "estado" having 3 unique values -> enum
	records := make([]*extract.Record, 10)
	vals := []string{"Pending", "Completado", "Draft"}
	for i := 0; i < 10; i++ {
		records[i] = &extract.Record{
			Path:        "doc.md",
			Frontmatter: map[string]any{"estado": vals[i%3]},
		}
	}

	result := Analyze(records)
	sf := result.Schema["estado"]
	if sf.Type != "enum" {
		t.Errorf("expected enum type, got %s", sf.Type)
	}
	if len(sf.Values) != 3 {
		t.Errorf("expected 3 enum values, got %d", len(sf.Values))
	}
}

func TestAnalyzeString(t *testing.T) {
	// Field with many unique values -> string
	records := make([]*extract.Record, 30)
	for i := 0; i < 30; i++ {
		records[i] = &extract.Record{
			Path:        "doc.md",
			Frontmatter: map[string]any{"title": "Unique Title " + string(rune('A'+i))},
		}
	}

	result := Analyze(records)
	sf := result.Schema["title"]
	if sf.Type != "string" {
		t.Errorf("expected string type, got %s", sf.Type)
	}
}

func TestAnalyzeList(t *testing.T) {
	records := makeRecords(
		map[string]any{"tags": []any{"go", "cli"}},
		map[string]any{"tags": []any{"go", "mcp"}},
	)

	result := Analyze(records)
	sf := result.Schema["tags"]
	if sf.Type != "list" {
		t.Errorf("expected list type, got %s", sf.Type)
	}
	stats := result.Fields["tags"]
	if !stats.HasList {
		t.Error("expected HasList=true")
	}
}
