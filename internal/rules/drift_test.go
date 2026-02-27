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

func TestDriftMatchExclusion(t *testing.T) {
	t.Run("match-scoped field present at both levels still excluded", func(t *testing.T) {
		// Edge case: both parent and children have "tipo" with different values,
		// but tipo has match restriction — drift should NOT be reported
		parent := extract.Record{
			Path:        "project/F01-feat/README.md",
			Frontmatter: map[string]any{"tipo": "feature"},
		}
		children := []extract.Record{
			{Path: "project/F01-feat/T001-task.md", Frontmatter: map[string]any{"tipo": "task"}},
			{Path: "project/F01-feat/T002-task.md", Frontmatter: map[string]any{"tipo": "task"}},
		}
		schema := map[string]SchemaField{
			"tipo": {Type: "enum", Match: &FieldMatch{Patterns: []string{"F*", "T*"}}},
		}

		warnings := DetectDrift(parent, children, schema)
		if len(warnings) != 0 {
			t.Errorf("got %d warnings, want 0 (match-scoped field at both levels)", len(warnings))
		}
	})

	t.Run("match-scoped field missing at parent excluded", func(t *testing.T) {
		// Parent doesn't have the field, children do — no drift (match-scoped)
		parent := extract.Record{
			Path:        "project/E01-epic/README.md",
			Frontmatter: map[string]any{"estado": "In Progress"},
		}
		children := []extract.Record{
			{Path: "project/E01-epic/F01-feat/README.md", Frontmatter: map[string]any{"estado": "In Progress", "tipo": "feature"}},
		}
		schema := map[string]SchemaField{
			"estado": {Type: "enum"}, // no match = checked
			"tipo":   {Type: "enum", Match: &FieldMatch{Patterns: []string{"F*", "T*"}}},
		}

		warnings := DetectDrift(parent, children, schema)
		// Only check that tipo produces no warning (estado matches so no warning either)
		for _, w := range warnings {
			if w.Field == "tipo" {
				t.Error("tipo should not produce drift warning (match-scoped)")
			}
		}
	})

	t.Run("map-form match also excluded", func(t *testing.T) {
		parent := extract.Record{
			Path:        "project/README.md",
			Frontmatter: map[string]any{"id": "E01"},
		}
		children := []extract.Record{
			{Path: "project/F01-feat/README.md", Frontmatter: map[string]any{"id": "F01"}},
		}
		schema := map[string]SchemaField{
			"id": {Type: "sequence", Match: &FieldMatch{Configs: map[string]any{
				"E*": map[string]any{"prefix": "E", "digits": 2},
				"F*": map[string]any{"prefix": "F", "digits": 2},
			}}},
		}

		warnings := DetectDrift(parent, children, schema)
		if len(warnings) != 0 {
			t.Errorf("got %d warnings, want 0 (map-form match excluded)", len(warnings))
		}
	})

	t.Run("field without match produces drift normally", func(t *testing.T) {
		parent := extract.Record{
			Path:        "project/README.md",
			Frontmatter: map[string]any{"estado": "In Progress"},
		}
		children := []extract.Record{
			{Path: "project/task1.md", Frontmatter: map[string]any{"estado": "Completed"}},
			{Path: "project/task2.md", Frontmatter: map[string]any{"estado": "Completed"}},
		}
		schema := map[string]SchemaField{
			"estado": {Type: "enum"}, // no match — should detect drift
		}

		warnings := DetectDrift(parent, children, schema)
		if len(warnings) != 1 {
			t.Errorf("got %d warnings, want 1 (no-match field should drift)", len(warnings))
		}
	})
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

func TestDetectDrift_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		parent       extract.Record
		children     []extract.Record
		schema       map[string]SchemaField
		wantWarnings int
		wantFields   []string
	}{
		{
			name: "single child matches parent — no warning",
			parent: extract.Record{
				Path:        "project/README.md",
				Frontmatter: map[string]any{"estado": "Pending"},
			},
			children: []extract.Record{
				{Path: "project/task1.md", Frontmatter: map[string]any{"estado": "Pending"}},
			},
			schema:       map[string]SchemaField{"estado": {Type: "enum"}},
			wantWarnings: 0,
		},
		{
			name: "single child differs from parent — warning",
			parent: extract.Record{
				Path:        "project/README.md",
				Frontmatter: map[string]any{"estado": "In Progress"},
			},
			children: []extract.Record{
				{Path: "project/task1.md", Frontmatter: map[string]any{"estado": "Completed"}},
			},
			schema:       map[string]SchemaField{"estado": {Type: "enum"}},
			wantWarnings: 1,
			wantFields:   []string{"estado"},
		},
		{
			name: "non-enum string field drift detected",
			parent: extract.Record{
				Path:        "project/README.md",
				Frontmatter: map[string]any{"titulo": "Old Title"},
			},
			children: []extract.Record{
				{Path: "project/doc1.md", Frontmatter: map[string]any{"titulo": "New Title"}},
				{Path: "project/doc2.md", Frontmatter: map[string]any{"titulo": "New Title"}},
			},
			schema:       map[string]SchemaField{"titulo": {Type: "string"}},
			wantWarnings: 1,
			wantFields:   []string{"titulo"},
		},
		{
			name: "multiple fields with drift — multiple warnings",
			parent: extract.Record{
				Path:        "project/README.md",
				Frontmatter: map[string]any{"estado": "Pending", "prioridad": "low"},
			},
			children: []extract.Record{
				{Path: "project/t1.md", Frontmatter: map[string]any{"estado": "Completed", "prioridad": "high"}},
				{Path: "project/t2.md", Frontmatter: map[string]any{"estado": "Completed", "prioridad": "high"}},
			},
			schema: map[string]SchemaField{
				"estado":    {Type: "enum"},
				"prioridad": {Type: "enum"},
			},
			wantWarnings: 2,
			wantFields:   []string{"estado", "prioridad"},
		},
		{
			name: "some children missing field — only present children count",
			parent: extract.Record{
				Path:        "project/README.md",
				Frontmatter: map[string]any{"estado": "Pending"},
			},
			children: []extract.Record{
				{Path: "project/t1.md", Frontmatter: map[string]any{"estado": "Completed"}},
				{Path: "project/t2.md", Frontmatter: map[string]any{}},
			},
			schema:       map[string]SchemaField{"estado": {Type: "enum"}},
			wantWarnings: 1,
			wantFields:   []string{"estado"},
		},
		{
			name: "integer vs string comparison works",
			parent: extract.Record{
				Path:        "project/README.md",
				Frontmatter: map[string]any{"version": 1},
			},
			children: []extract.Record{
				{Path: "project/t1.md", Frontmatter: map[string]any{"version": 2}},
			},
			schema:       map[string]SchemaField{"version": {Type: "number"}},
			wantWarnings: 1,
			wantFields:   []string{"version"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := DetectDrift(tt.parent, tt.children, tt.schema)
			if len(warnings) != tt.wantWarnings {
				t.Errorf("got %d warnings, want %d: %+v", len(warnings), tt.wantWarnings, warnings)
			}
			if tt.wantFields != nil {
				gotFields := make(map[string]bool)
				for _, w := range warnings {
					gotFields[w.Field] = true
				}
				for _, f := range tt.wantFields {
					if !gotFields[f] {
						t.Errorf("missing expected warning for field %q", f)
					}
				}
			}
		})
	}
}
