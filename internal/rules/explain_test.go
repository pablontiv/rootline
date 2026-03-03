package rules

import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func TestSortedMapKeys(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]any
		want []string
	}{
		{"nil map", nil, nil},
		{"empty map", map[string]any{}, nil},
		{"single key", map[string]any{"a": 1}, []string{"a"}},
		{"sorted order", map[string]any{"b": 2, "a": 1, "c": 3}, []string{"a", "b", "c"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sortedMapKeys(tt.m)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("index %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestNewExplainResult_FrontmatterOnly(t *testing.T) {
	record := &extract.Record{
		Frontmatter: map[string]any{"estado": "Pending", "tipo": "test"},
		Derived:     map[string]any{},
	}
	entries := []StemEntry{
		{Path: "/tmp/.stem"},
	}
	effective := &StemFile{
		Schema: map[string]SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Completed"}, Source: "/tmp/.stem"},
			"tipo":   {Type: "string", Source: "/tmp/.stem"},
		},
	}

	result := NewExplainResult("doc.md", entries, effective, record, nil)

	if result.Version != 1 {
		t.Errorf("version = %d, want 1", result.Version)
	}
	if result.Kind != "rootline/explain" {
		t.Errorf("kind = %q, want rootline/explain", result.Kind)
	}
	if result.Path != "doc.md" {
		t.Errorf("path = %q, want doc.md", result.Path)
	}
	if len(result.StemChain) != 1 || result.StemChain[0] != "/tmp/.stem" {
		t.Errorf("stem_chain = %v, want [/tmp/.stem]", result.StemChain)
	}

	// Should have 2 frontmatter fields (estado, tipo) sorted.
	if len(result.Fields) < 2 {
		t.Fatalf("fields = %d, want >= 2", len(result.Fields))
	}
	if result.Fields[0].Name != "estado" || result.Fields[0].Origin != "frontmatter" {
		t.Errorf("fields[0] = %v, want estado/frontmatter", result.Fields[0])
	}
	if result.Fields[0].Source != "/tmp/.stem" {
		t.Errorf("fields[0].source = %q, want /tmp/.stem", result.Fields[0].Source)
	}
	if len(result.Errors) != 0 {
		t.Errorf("errors = %d, want 0", len(result.Errors))
	}
}

func TestNewExplainResult_NilEffective(t *testing.T) {
	record := &extract.Record{
		Frontmatter: map[string]any{"title": "Hello"},
		Derived:     map[string]any{},
	}

	result := NewExplainResult("doc.md", nil, nil, record, nil)

	if result.Version != 1 {
		t.Errorf("version = %d, want 1", result.Version)
	}
	if len(result.Fields) != 1 {
		t.Fatalf("fields = %d, want 1", len(result.Fields))
	}
	if result.Fields[0].Name != "title" {
		t.Errorf("field name = %q, want title", result.Fields[0].Name)
	}
	// No source when effective is nil.
	if result.Fields[0].Source != "" {
		t.Errorf("source = %q, want empty", result.Fields[0].Source)
	}
}

func TestNewExplainResult_DerivedFields(t *testing.T) {
	record := &extract.Record{
		Frontmatter: map[string]any{"estado": "Pending"},
		Derived:     map[string]any{"slug": "pending"},
	}
	effective := &StemFile{
		Schema: map[string]SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending"}, Source: ".stem"},
		},
		Derive: map[string]any{
			"slug": "slugify(estado)",
		},
	}

	result := NewExplainResult("doc.md", nil, effective, record, nil)

	// Find the slug field.
	var slugField *ExplainField
	for i := range result.Fields {
		if result.Fields[i].Name == "slug" {
			slugField = &result.Fields[i]
			break
		}
	}
	if slugField == nil {
		t.Fatal("slug field not found")
	}
	if slugField.Origin != "derived" {
		t.Errorf("slug origin = %q, want derived", slugField.Origin)
	}
	if slugField.Expression != "slugify(estado)" {
		t.Errorf("slug expression = %q, want slugify(estado)", slugField.Expression)
	}
	if slugField.Value != "pending" {
		t.Errorf("slug value = %v, want pending", slugField.Value)
	}
}

func TestNewExplainResult_AggregateFields(t *testing.T) {
	record := &extract.Record{
		Frontmatter: map[string]any{},
		Derived:     map[string]any{"total": 5},
	}
	effective := &StemFile{
		Schema:    map[string]SchemaField{},
		Aggregate: map[string]any{"total": "count(children)"},
	}

	result := NewExplainResult("README.md", nil, effective, record, nil)

	var totalField *ExplainField
	for i := range result.Fields {
		if result.Fields[i].Name == "total" {
			totalField = &result.Fields[i]
			break
		}
	}
	if totalField == nil {
		t.Fatal("total field not found")
	}
	if totalField.Origin != "aggregate" {
		t.Errorf("total origin = %q, want aggregate", totalField.Origin)
	}
	if totalField.Expression != "count(children)" {
		t.Errorf("total expression = %q, want count(children)", totalField.Expression)
	}
}

func TestNewExplainResult_SchemaFieldMissing(t *testing.T) {
	// When a schema field is not in frontmatter, it appears with origin "schema".
	record := &extract.Record{
		Frontmatter: map[string]any{},
		Derived:     map[string]any{},
	}
	effective := &StemFile{
		Schema: map[string]SchemaField{
			"estado": {Type: "enum", Required: true, Source: ".stem"},
		},
	}

	result := NewExplainResult("doc.md", nil, effective, record, nil)

	if len(result.Fields) != 1 {
		t.Fatalf("fields = %d, want 1", len(result.Fields))
	}
	if result.Fields[0].Origin != "schema" {
		t.Errorf("origin = %q, want schema", result.Fields[0].Origin)
	}
	if result.Fields[0].Value != nil {
		t.Errorf("value = %v, want nil", result.Fields[0].Value)
	}
}

func TestNewExplainResult_SchemaFieldWithDefault(t *testing.T) {
	record := &extract.Record{
		Frontmatter: map[string]any{},
		Derived:     map[string]any{},
	}
	effective := &StemFile{
		Schema: map[string]SchemaField{
			"tipo": {Type: "string", Default: "feature", Source: ".stem"},
		},
	}

	result := NewExplainResult("doc.md", nil, effective, record, nil)

	if len(result.Fields) != 1 {
		t.Fatalf("fields = %d, want 1", len(result.Fields))
	}
	if result.Fields[0].Value != "feature" {
		t.Errorf("value = %v, want feature", result.Fields[0].Value)
	}
}

func TestNewExplainResult_ValidationErrors(t *testing.T) {
	record := &extract.Record{
		Frontmatter: map[string]any{"estado": "Bad"},
		Derived:     map[string]any{},
	}
	effective := &StemFile{
		Schema: map[string]SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending"}, Source: ".stem"},
		},
	}
	valErrs := []ValidationError{
		{Rule: "enum", Field: "estado", Message: "invalid value", Source: ".stem", Severity: "error"},
	}

	result := NewExplainResult("doc.md", nil, effective, record, valErrs)

	if len(result.Errors) != 1 {
		t.Fatalf("errors = %d, want 1", len(result.Errors))
	}
	e := result.Errors[0]
	if e.Rule != "enum" {
		t.Errorf("rule = %q, want enum", e.Rule)
	}
	if e.Field != "estado" {
		t.Errorf("field = %q, want estado", e.Field)
	}
	if e.Severity != "error" {
		t.Errorf("severity = %q, want error", e.Severity)
	}
}

func TestNewExplainResult_MultipleStemChain(t *testing.T) {
	record := &extract.Record{
		Frontmatter: map[string]any{"estado": "Pending"},
		Derived:     map[string]any{},
	}
	entries := []StemEntry{
		{Path: "/project/.stem"},
		{Path: "/project/docs/.stem"},
	}
	effective := &StemFile{
		Schema: map[string]SchemaField{
			"estado": {Type: "enum", Source: "/project/docs/.stem"},
		},
	}

	result := NewExplainResult("docs/task.md", entries, effective, record, nil)

	if len(result.StemChain) != 2 {
		t.Fatalf("stem_chain = %d, want 2", len(result.StemChain))
	}
	if result.StemChain[0] != "/project/.stem" {
		t.Errorf("stem_chain[0] = %q, want /project/.stem", result.StemChain[0])
	}
	if result.StemChain[1] != "/project/docs/.stem" {
		t.Errorf("stem_chain[1] = %q, want /project/docs/.stem", result.StemChain[1])
	}
}

func TestNewExplainResult_DeriveNonStringExpression(t *testing.T) {
	// When derive expression is not a string (e.g., a map or int), expression should be empty.
	record := &extract.Record{
		Frontmatter: map[string]any{},
		Derived:     map[string]any{"computed": 42},
	}
	effective := &StemFile{
		Schema: map[string]SchemaField{},
		Derive: map[string]any{
			"computed": 42, // non-string expression
		},
	}

	result := NewExplainResult("doc.md", nil, effective, record, nil)

	var field *ExplainField
	for i := range result.Fields {
		if result.Fields[i].Name == "computed" {
			field = &result.Fields[i]
			break
		}
	}
	if field == nil {
		t.Fatal("computed field not found")
	}
	if field.Expression != "" {
		t.Errorf("expression = %q, want empty (non-string derive)", field.Expression)
	}
}
