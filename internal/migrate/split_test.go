package migrate

import (
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/infer"
	"github.com/pablontiv/rootline/internal/rules"
)

// makeHierarchy builds a minimal HierarchyResult for testing split logic.
func makeHierarchy(rootSchema map[string]rules.SchemaField, levels []infer.LevelSchema) *infer.HierarchyResult {
	return &infer.HierarchyResult{
		Root: &infer.InferredSchema{
			Schema: rootSchema,
		},
		Levels:   levels,
		Detected: true,
	}
}

func TestBuildSplitStems_BasicTwoLevels(t *testing.T) {
	existing := &rules.StemFile{
		Version: 1,
		Scope:   rules.Scope{Match: "*.md"},
		Schema: map[string]rules.SchemaField{
			"id":     {Type: "sequence", Prefix: "E", Digits: 2},
			"estado": {Type: "enum", Values: []string{"Pending", "Done"}, Required: true},
			"tipo":   {Type: "enum", Values: []string{"feature", "historia"}},
		},
	}

	hierarchy := makeHierarchy(
		map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Done"}, Required: true},
		},
		[]infer.LevelSchema{
			{
				Level: infer.Level{
					Prefix: "E", Digits: 2, Depth: 0,
					DirPaths: []string{"E01-infra", "E02-platform"},
				},
				OnlyHere: map[string]rules.SchemaField{
					"id": {Type: "sequence", Prefix: "E", Digits: 2},
				},
			},
			{
				Level: infer.Level{
					Prefix: "F", Digits: 2, Depth: 1,
					DirPaths: []string{"E01-infra/F01-net", "E01-infra/F02-store"},
				},
				OnlyHere: map[string]rules.SchemaField{
					"id":   {Type: "sequence", Prefix: "F", Digits: 2},
					"tipo": {Type: "enum", Values: []string{"feature", "historia"}},
				},
			},
		},
	)

	files := BuildSplitStems("/tmp/test", existing, hierarchy)

	if len(files) == 0 {
		t.Fatal("expected at least one stem file")
	}

	// First file should be the root .stem.
	root := files[0]
	if root.Path != "/tmp/test/.stem" {
		t.Errorf("root path = %s, want /tmp/test/.stem", root.Path)
	}
	if !strings.Contains(root.Content, "prefix: E") {
		t.Errorf("root should contain prefix: E, got:\n%s", root.Content)
	}
	if !strings.Contains(root.Content, "estado:") {
		t.Errorf("root should contain estado, got:\n%s", root.Content)
	}
	if !strings.Contains(root.Content, "version: 1") {
		t.Errorf("root should contain version: 1, got:\n%s", root.Content)
	}
	if !strings.Contains(root.Content, `match: "*.md"`) {
		t.Errorf("root should preserve scope match, got:\n%s", root.Content)
	}

	// Child stems should be placed in E01-infra and E02-platform.
	childPaths := make(map[string]string)
	for _, f := range files[1:] {
		childPaths[f.Path] = f.Content
	}
	for _, dir := range []string{"E01-infra", "E02-platform"} {
		path := "/tmp/test/" + dir + "/.stem"
		content, ok := childPaths[path]
		if !ok {
			t.Errorf("missing child .stem for %s", dir)
			continue
		}
		if !strings.Contains(content, "prefix: F") {
			t.Errorf("child %s should contain prefix: F, got:\n%s", dir, content)
		}
		if !strings.Contains(content, "tipo:") {
			t.Errorf("child %s should contain tipo, got:\n%s", dir, content)
		}
	}
}

func TestBuildSplitStems_PreservesStructural(t *testing.T) {
	existing := &rules.StemFile{
		Version: 1,
		Schema: map[string]rules.SchemaField{
			"id":     {Type: "sequence", Prefix: "E", Digits: 2},
			"estado": {Type: "string"},
		},
		Structural: rules.StructuralRules{
			Subdirs: rules.SubdirRules{
				RequireIndex: "README.md",
				MinChildren:  2,
				Severity:     "warn",
			},
		},
	}

	hierarchy := makeHierarchy(
		map[string]rules.SchemaField{
			"estado": {Type: "string"},
		},
		[]infer.LevelSchema{
			{
				Level:    infer.Level{Prefix: "E", Digits: 2, DirPaths: []string{"E01"}},
				OnlyHere: map[string]rules.SchemaField{"id": {Type: "sequence", Prefix: "E", Digits: 2}},
			},
		},
	)

	files := BuildSplitStems("/tmp/test", existing, hierarchy)
	root := files[0].Content

	if !strings.Contains(root, "structural:") {
		t.Errorf("root should preserve structural section, got:\n%s", root)
	}
	if !strings.Contains(root, "require_index: README.md") {
		t.Errorf("root should preserve require_index, got:\n%s", root)
	}
	if !strings.Contains(root, "min_children: 2") {
		t.Errorf("root should preserve min_children, got:\n%s", root)
	}
	if !strings.Contains(root, "severity: warn") {
		t.Errorf("root should preserve severity, got:\n%s", root)
	}
}

func TestBuildSplitStems_PreservesLinks(t *testing.T) {
	existing := &rules.StemFile{
		Version: 1,
		Schema: map[string]rules.SchemaField{
			"id":     {Type: "sequence", Prefix: "E", Digits: 2},
			"estado": {Type: "string"},
		},
		Links: rules.LinkSchema{
			Allowed: []string{"blocks", "reference"},
			Rules: map[string]rules.LinkRule{
				"blocks": {Target: `^T\d{3}-`, Field: "blocked_by"},
			},
		},
	}

	hierarchy := makeHierarchy(
		map[string]rules.SchemaField{"estado": {Type: "string"}},
		[]infer.LevelSchema{
			{
				Level:    infer.Level{Prefix: "E", Digits: 2, DirPaths: []string{"E01"}},
				OnlyHere: map[string]rules.SchemaField{"id": {Type: "sequence", Prefix: "E", Digits: 2}},
			},
		},
	)

	files := BuildSplitStems("/tmp/test", existing, hierarchy)
	root := files[0].Content

	if !strings.Contains(root, "links:") {
		t.Errorf("root should preserve links section, got:\n%s", root)
	}
	if !strings.Contains(root, "allowed: [blocks, reference]") {
		t.Errorf("root should preserve allowed links, got:\n%s", root)
	}
}

func TestBuildSplitStems_PreservesDeriveAndAggregate(t *testing.T) {
	existing := &rules.StemFile{
		Version: 1,
		Schema: map[string]rules.SchemaField{
			"id":     {Type: "sequence", Prefix: "E", Digits: 2},
			"estado": {Type: "string"},
		},
		Derive: map[string]any{
			"estado": `hold != nil ? "On Hold" : estado`,
		},
		Aggregate: map[string]any{
			"estado": `all(descendants, {.estado == "Completed"}) ? "Completed" : "Pending"`,
		},
	}

	hierarchy := makeHierarchy(
		map[string]rules.SchemaField{"estado": {Type: "string"}},
		[]infer.LevelSchema{
			{
				Level:    infer.Level{Prefix: "E", Digits: 2, DirPaths: []string{"E01"}},
				OnlyHere: map[string]rules.SchemaField{"id": {Type: "sequence", Prefix: "E", Digits: 2}},
			},
		},
	)

	files := BuildSplitStems("/tmp/test", existing, hierarchy)
	root := files[0].Content

	if !strings.Contains(root, "derive:") {
		t.Errorf("root should preserve derive section, got:\n%s", root)
	}
	if !strings.Contains(root, "aggregate:") {
		t.Errorf("root should preserve aggregate section, got:\n%s", root)
	}
}

func TestBuildSplitStems_PreservesValidate(t *testing.T) {
	existing := &rules.StemFile{
		Version: 1,
		Schema: map[string]rules.SchemaField{
			"id":     {Type: "sequence", Prefix: "E", Digits: 2},
			"estado": {Type: "string"},
		},
		Validate: []rules.ValidationRule{
			{Field: "estado", Rule: "non_empty"},
			{Rule: "requires", If: map[string]any{"tipo": "ci-cd"}, Then: map[string]any{"fields": []any{"ejecutable_en"}}},
		},
	}

	hierarchy := makeHierarchy(
		map[string]rules.SchemaField{"estado": {Type: "string"}},
		[]infer.LevelSchema{
			{
				Level:    infer.Level{Prefix: "E", Digits: 2, DirPaths: []string{"E01"}},
				OnlyHere: map[string]rules.SchemaField{"id": {Type: "sequence", Prefix: "E", Digits: 2}},
			},
		},
	)

	files := BuildSplitStems("/tmp/test", existing, hierarchy)
	root := files[0].Content

	if !strings.Contains(root, "validate:") {
		t.Errorf("root should preserve validate section, got:\n%s", root)
	}
	if !strings.Contains(root, "non_empty") {
		t.Errorf("root should preserve non_empty rule, got:\n%s", root)
	}
	if !strings.Contains(root, "requires") {
		t.Errorf("root should preserve requires rule, got:\n%s", root)
	}
}

func TestBuildSplitStems_ChildYAMLHasOnlyLevelFields(t *testing.T) {
	existing := &rules.StemFile{
		Version: 1,
		Schema: map[string]rules.SchemaField{
			"id":     {Type: "sequence", Prefix: "E", Digits: 2},
			"estado": {Type: "string", Required: true},
			"tipo":   {Type: "enum", Values: []string{"feature"}, Severity: "warn"},
		},
	}

	hierarchy := makeHierarchy(
		map[string]rules.SchemaField{"estado": {Type: "string", Required: true}},
		[]infer.LevelSchema{
			{
				Level:    infer.Level{Prefix: "E", Digits: 2, DirPaths: []string{"E01"}},
				OnlyHere: map[string]rules.SchemaField{"id": {Type: "sequence", Prefix: "E", Digits: 2}},
			},
			{
				Level:    infer.Level{Prefix: "F", Digits: 2, DirPaths: []string{"E01/F01"}},
				OnlyHere: map[string]rules.SchemaField{"id": {Type: "sequence", Prefix: "F", Digits: 2}, "tipo": {Type: "enum", Values: []string{"feature"}}},
			},
		},
	)

	files := BuildSplitStems("/tmp/test", existing, hierarchy)

	// Find child stem.
	var childContent string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "E01/.stem") {
			childContent = f.Content
			break
		}
	}
	if childContent == "" {
		t.Fatal("expected child .stem at E01/.stem")
	}

	if !strings.Contains(childContent, "prefix: F") {
		t.Errorf("child should have prefix: F, got:\n%s", childContent)
	}
	if !strings.Contains(childContent, "tipo:") {
		t.Errorf("child should have tipo field, got:\n%s", childContent)
	}
	// Child should NOT have estado (that's in root).
	if strings.Contains(childContent, "estado:") {
		t.Errorf("child should NOT have estado (root field), got:\n%s", childContent)
	}
	// Child should NOT have derive/aggregate/structural/links.
	for _, section := range []string{"derive:", "aggregate:", "structural:", "links:", "validate:"} {
		if strings.Contains(childContent, section) {
			t.Errorf("child should NOT have %s, got:\n%s", section, childContent)
		}
	}
}

func TestBuildSplitStems_NoScopeOmitted(t *testing.T) {
	existing := &rules.StemFile{
		Version: 1,
		Schema: map[string]rules.SchemaField{
			"id":     {Type: "sequence", Prefix: "E", Digits: 2},
			"estado": {Type: "string"},
		},
	}

	hierarchy := makeHierarchy(
		map[string]rules.SchemaField{"estado": {Type: "string"}},
		[]infer.LevelSchema{
			{
				Level:    infer.Level{Prefix: "E", Digits: 2, DirPaths: []string{"E01"}},
				OnlyHere: map[string]rules.SchemaField{"id": {Type: "sequence", Prefix: "E", Digits: 2}},
			},
		},
	)

	files := BuildSplitStems("/tmp/test", existing, hierarchy)
	root := files[0].Content

	if strings.Contains(root, "scope:") {
		t.Errorf("root should not have scope when match is empty, got:\n%s", root)
	}
}

// Ensure the extract.Record import is used (needed for infer.InferredSchema).
var _ = (*extract.Record)(nil)
