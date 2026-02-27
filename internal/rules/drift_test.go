package rules

import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func TestDetectDrift_UnanimousMismatch(t *testing.T) {
	parent := extract.Record{
		Path:        "project/README.md",
		Frontmatter: map[string]any{"estado": "In Progress"},
	}
	children := []extract.Record{
		{Path: "project/task1.md", Frontmatter: map[string]any{"estado": "Completed"}},
		{Path: "project/task2.md", Frontmatter: map[string]any{"estado": "Completed"}},
	}
	schema := map[string]SchemaField{
		"estado": {Type: "enum"},
	}

	warnings := DetectDrift(parent, children, schema)
	if len(warnings) != 1 {
		t.Fatalf("got %d warnings, want 1", len(warnings))
	}
	w := warnings[0]
	if w.Field != "estado" {
		t.Errorf("field = %q, want estado", w.Field)
	}
	if w.ParentValue != "In Progress" {
		t.Errorf("parent_value = %v, want In Progress", w.ParentValue)
	}
	if w.ChildrenValue != "Completed" {
		t.Errorf("children_value = %v, want Completed", w.ChildrenValue)
	}
	if len(w.ChildPaths) != 2 {
		t.Errorf("child_paths len = %d, want 2", len(w.ChildPaths))
	}
}

func TestDetectDrift_MixedChildren(t *testing.T) {
	parent := extract.Record{
		Path:        "project/README.md",
		Frontmatter: map[string]any{"estado": "In Progress"},
	}
	children := []extract.Record{
		{Path: "project/task1.md", Frontmatter: map[string]any{"estado": "Completed"}},
		{Path: "project/task2.md", Frontmatter: map[string]any{"estado": "Pending"}},
	}
	schema := map[string]SchemaField{
		"estado": {Type: "enum"},
	}

	warnings := DetectDrift(parent, children, schema)
	if len(warnings) != 0 {
		t.Errorf("got %d warnings, want 0 (mixed children)", len(warnings))
	}
}

func TestDetectDrift_ParentMissingField(t *testing.T) {
	parent := extract.Record{
		Path:        "project/README.md",
		Frontmatter: map[string]any{},
	}
	children := []extract.Record{
		{Path: "project/task1.md", Frontmatter: map[string]any{"estado": "Completed"}},
	}
	schema := map[string]SchemaField{
		"estado": {Type: "enum"},
	}

	warnings := DetectDrift(parent, children, schema)
	if len(warnings) != 0 {
		t.Errorf("got %d warnings, want 0 (parent missing field)", len(warnings))
	}
}

func TestDetectDrift_EmptyChildren(t *testing.T) {
	parent := extract.Record{
		Path:        "project/README.md",
		Frontmatter: map[string]any{"estado": "In Progress"},
	}
	schema := map[string]SchemaField{
		"estado": {Type: "enum"},
	}

	warnings := DetectDrift(parent, nil, schema)
	if len(warnings) != 0 {
		t.Errorf("got %d warnings, want 0 (no children)", len(warnings))
	}
}

func TestDetectDrift_MatchedFieldSkipped(t *testing.T) {
	parent := extract.Record{
		Path:        "project/README.md",
		Frontmatter: map[string]any{"tipo": "feature"},
	}
	children := []extract.Record{
		{Path: "project/task1.md", Frontmatter: map[string]any{"tipo": "task"}},
		{Path: "project/task2.md", Frontmatter: map[string]any{"tipo": "task"}},
	}
	schema := map[string]SchemaField{
		"tipo": {Type: "enum", Match: &FieldMatch{Patterns: []string{"T*"}}},
	}

	warnings := DetectDrift(parent, children, schema)
	if len(warnings) != 0 {
		t.Errorf("got %d warnings, want 0 (field has match restriction)", len(warnings))
	}
}

func TestDetectDrift_NoChildrenHaveField(t *testing.T) {
	parent := extract.Record{
		Path:        "project/README.md",
		Frontmatter: map[string]any{"estado": "Pending"},
	}
	children := []extract.Record{
		{Path: "project/task1.md", Frontmatter: map[string]any{"other": "value"}},
	}
	schema := map[string]SchemaField{
		"estado": {Type: "enum"},
	}

	warnings := DetectDrift(parent, children, schema)
	if len(warnings) != 0 {
		t.Errorf("got %d warnings, want 0 (no children have field)", len(warnings))
	}
}

func TestDetectDrift_ValuesMatch(t *testing.T) {
	parent := extract.Record{
		Path:        "project/README.md",
		Frontmatter: map[string]any{"estado": "Completed"},
	}
	children := []extract.Record{
		{Path: "project/task1.md", Frontmatter: map[string]any{"estado": "Completed"}},
		{Path: "project/task2.md", Frontmatter: map[string]any{"estado": "Completed"}},
	}
	schema := map[string]SchemaField{
		"estado": {Type: "enum"},
	}

	warnings := DetectDrift(parent, children, schema)
	if len(warnings) != 0 {
		t.Errorf("got %d warnings, want 0 (values match)", len(warnings))
	}
}
