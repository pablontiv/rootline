package migrate

import (
	"os"
	"path/filepath"
	"reflect"
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

func mustBuildSplitStems(t *testing.T, absTarget string, existing *rules.StemFile, hierarchy *infer.HierarchyResult) SplitResult {
	t.Helper()
	result, err := BuildSplitStems(absTarget, existing, hierarchy)
	if err != nil {
		t.Fatalf("BuildSplitStems returned error: %v", err)
	}
	return result
}

func TestBuildSplitStems_BasicTwoLevels(t *testing.T) {
	existing := &rules.StemFile{
		Version: 2,
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

	result := mustBuildSplitStems(t, "/tmp/test", existing, hierarchy)

	if len(result.Stems) == 0 {
		t.Fatal("expected at least one stem file")
	}

	// First file should be the root .stem.
	root := result.Stems[0]
	if root.Path != "/tmp/test/.stem" {
		t.Errorf("root path = %s, want /tmp/test/.stem", root.Path)
	}
	if !strings.Contains(root.Content, "prefix: E") {
		t.Errorf("root should contain prefix: E, got:\n%s", root.Content)
	}
	if !strings.Contains(root.Content, "estado:") {
		t.Errorf("root should contain estado, got:\n%s", root.Content)
	}
	if !strings.Contains(root.Content, "version: 2") {
		t.Errorf("root should contain version: 2, got:\n%s", root.Content)
	}
	if !strings.Contains(root.Content, `match: "*.md"`) {
		t.Errorf("root should preserve scope match, got:\n%s", root.Content)
	}

	// Child stems should be placed in E01-infra and E02-platform.
	childPaths := make(map[string]string)
	for _, f := range result.Stems[1:] {
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

func TestBuildSplitStems_PreservesFieldSource(t *testing.T) {
	existing := &rules.StemFile{
		Version: 2,
		Scope:   rules.Scope{Match: "*.md"},
		Schema: map[string]rules.SchemaField{
			"id":    {Type: "sequence", Prefix: "E", Digits: 2},
			"notes": {Type: "string", Extract: `body.section["## Notes: risks #1"]`, Required: true, Default: "TODO", Severity: "warn"},
			"title": {Type: "string", Extract: `body.h1`, Match: &rules.FieldMatch{Patterns: []string{"E*"}}},
		},
	}

	hierarchy := makeHierarchy(
		map[string]rules.SchemaField{
			"notes": {Type: "string"},
		},
		[]infer.LevelSchema{
			{
				Level:    infer.Level{Prefix: "E", Digits: 2, DirPaths: []string{"E01"}},
				OnlyHere: map[string]rules.SchemaField{"id": {Type: "sequence", Prefix: "E", Digits: 2}},
			},
			{
				Level:    infer.Level{Prefix: "F", Digits: 2, DirPaths: []string{"E01/F01"}},
				OnlyHere: map[string]rules.SchemaField{"id": {Type: "sequence", Prefix: "F", Digits: 2}, "title": {Type: "string"}},
			},
		},
	)

	result := mustBuildSplitStems(t, "/tmp/test", existing, hierarchy)
	if len(result.Stems) < 2 {
		t.Fatalf("expected root and child stems, got %d", len(result.Stems))
	}

	root := result.Stems[0]
	if !strings.Contains(root.Content, `source: 'body.section["## Notes: risks #1"]'`) {
		t.Fatalf("source lost from root stem:\n%s", root.Content)
	}
	if !strings.Contains(root.Content, "required: true") || !strings.Contains(root.Content, "default: TODO") || !strings.Contains(root.Content, "severity: warn") {
		t.Fatalf("root field attrs were not preserved:\n%s", root.Content)
	}
	parsedRoot, err := rules.ParseStem(root.Path, []byte(root.Content))
	if err != nil {
		t.Fatalf("root stem did not parse: %v\n%s", err, root.Content)
	}
	if got := parsedRoot.Schema["notes"].Extract; got != `body.section["## Notes: risks #1"]` {
		t.Fatalf("root source = %q", got)
	}

	var child StemOutput
	for _, sf := range result.Stems[1:] {
		if strings.HasSuffix(sf.Path, "E01/.stem") {
			child = sf
		}
	}
	if child.Content == "" {
		t.Fatalf("missing child stem in %+v", result.Stems)
	}
	if !strings.Contains(child.Content, "source: body.h1") || !strings.Contains(child.Content, `match: "E*"`) {
		t.Fatalf("child source/match attrs were not preserved:\n%s", child.Content)
	}
	parsedChild, err := rules.ParseStem(child.Path, []byte(child.Content))
	if err != nil {
		t.Fatalf("child stem did not parse: %v\n%s", err, child.Content)
	}
	if got := parsedChild.Schema["title"].Extract; got != `body.h1` {
		t.Fatalf("child source = %q", got)
	}
}

func TestBuildSplitStems_PreservesStructural(t *testing.T) {
	existing := &rules.StemFile{
		Version: 2,
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

	result := mustBuildSplitStems(t, "/tmp/test", existing, hierarchy)
	root := result.Stems[0].Content

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
		Version: 2,
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

	result := mustBuildSplitStems(t, "/tmp/test", existing, hierarchy)
	root := result.Stems[0].Content

	if !strings.Contains(root, "links:") {
		t.Errorf("root should preserve links section, got:\n%s", root)
	}
	parsed, err := rules.ParseStem(result.Stems[0].Path, []byte(root))
	if err != nil {
		t.Fatalf("root did not parse: %v\n%s", err, root)
	}
	if strings.Join(parsed.Links.Allowed, ",") != "blocks,reference" {
		t.Errorf("root should preserve allowed links, got: %+v", parsed.Links)
	}
}

func TestBuildSplitStems_RoundTripsRootGovernance(t *testing.T) {
	tests := []struct {
		name       string
		governance string
	}{
		{
			name: "all structural fields",
			governance: `structural:
  subdirs:
    require_index: "README: governance #1.md"
    min_children: 2
    max_children: 7
    severity: warn
`,
		},
		{name: "max children only", governance: "structural:\n  subdirs:\n    max_children: 3\n"},
		{name: "severity only", governance: "structural:\n  subdirs:\n    severity: off\n"},
		{
			name: "all links fields",
			governance: `links:
  allowed: ["blocks: #1", reference]
  styles: [wikilink, markdown]
  checks:
    resolve: false
    anchors: true
    encoding: true
    cycles: true
  basename_fallback: true
  "blocks: #1":
    target: "../tasks: active #1/*.md"
    field: "blocked_by: #1"
    value_field: "id: #1"
`,
		},
		{name: "styles without allowed", governance: "links:\n  styles: [markdown]\n"},
		{name: "fallback without allowed", governance: "links:\n  basename_fallback: true\n"},
		{name: "rule without allowed", governance: "links:\n  parent:\n    target: '../*.md'\n    field: parent\n    value_field: id\n"},
		{name: "explicit empty allowed", governance: "links:\n  allowed: []\n  styles: [markdown]\n  checks:\n    resolve: false\n"},
		{
			name: "all links fields with empty allowed",
			governance: `links:
  allowed: []
  styles: [wikilink, markdown]
  checks:
    resolve: false
    anchors: true
    encoding: true
    cycles: true
  basename_fallback: true
  parent:
    target: "../*.md"
    field: parent
    value_field: id
`,
		},
		{name: "empty allowed only", governance: "links:\n  allowed: []\n"},
		{name: "empty lists", governance: "links:\n  allowed: []\n  styles: []\n"},
		{name: "empty checks", governance: "links:\n  checks: {}\n"},
		{name: "checks without resolve", governance: "links:\n  checks:\n    anchors: true\n"},
		{name: "explicit true resolve", governance: "links:\n  checks:\n    resolve: true\n"},
		{
			name: "explicit false checks",
			governance: `links:
  checks:
    resolve: false
    anchors: false
    encoding: false
    cycles: false
  basename_fallback: false
`,
		},
	}
	hierarchy := makeHierarchy(
		map[string]rules.SchemaField{"title": {Type: "string"}},
		[]infer.LevelSchema{
			{Level: infer.Level{Prefix: "E", Digits: 2, DirPaths: []string{"E01", "E02"}}, OnlyHere: map[string]rules.SchemaField{"id": {Type: "sequence", Prefix: "E", Digits: 2}}},
			{Level: infer.Level{Prefix: "F", Digits: 2, DirPaths: []string{"E01/F01", "E02/F01"}}, OnlyHere: map[string]rules.SchemaField{"id": {Type: "sequence", Prefix: "F", Digits: 2}}},
		},
	)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			existing, err := rules.ParseStem("source.stem", []byte(`version: 2
root: true
scope:
  match: "*.md"
schema:
  title:
    type: string
derive:
  title: 'title'
aggregate:
  title: '"root"'
validate:
  - rule: non_empty
    field: title
`+tt.governance))
			if err != nil {
				t.Fatalf("fixture did not production-parse: %v", err)
			}
			result := mustBuildSplitStems(t, "/tmp/test", existing, hierarchy)
			if len(result.Stems) != 3 {
				t.Fatalf("expected root and two children, got %d", len(result.Stems))
			}
			root := result.Stems[0]
			parsed, err := rules.ParseStem(root.Path, []byte(root.Content))
			if err != nil {
				t.Fatalf("root did not production-parse: %v\n%s", err, root.Content)
			}
			if parsed.Structural != existing.Structural || !reflect.DeepEqual(parsed.Links, existing.Links) {
				t.Fatalf("root governance changed:\nstructural: got %+v, want %+v\nlinks: got %+v, want %+v\n%s",
					parsed.Structural, existing.Structural, parsed.Links, existing.Links, root.Content)
			}
			if parsed.Version != 2 || !parsed.Root || parsed.Scope != existing.Scope {
				t.Fatalf("root version/boundary/scope changed: %+v", parsed)
			}
			for _, child := range result.Stems[1:] {
				parsedChild, err := rules.ParseStem(child.Path, []byte(child.Content))
				if err != nil {
					t.Fatalf("child did not production-parse: %v\n%s", err, child.Content)
				}
				if parsedChild.Version != 2 || parsedChild.Root || parsedChild.Structural != (rules.StructuralRules{}) ||
					!reflect.DeepEqual(parsedChild.Links, rules.LinkSchema{}) || len(parsedChild.Derive) != 0 ||
					len(parsedChild.Aggregate) != 0 || len(parsedChild.Validate) != 0 {
					t.Fatalf("root governance leaked into child: %+v\n%s", parsedChild, child.Content)
				}
				for _, section := range []string{"root:", "structural:", "links:", "derive:", "aggregate:", "validate:"} {
					if strings.Contains(child.Content, section) {
						t.Fatalf("%s leaked into child:\n%s", section, child.Content)
					}
				}
			}
		})
	}
}

func TestBuildSplitStems_PreservesDeriveAndAggregate(t *testing.T) {
	existing := &rules.StemFile{
		Version: 2,
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

	result := mustBuildSplitStems(t, "/tmp/test", existing, hierarchy)
	root := result.Stems[0].Content

	if !strings.Contains(root, "derive:") {
		t.Errorf("root should preserve derive section, got:\n%s", root)
	}
	if !strings.Contains(root, "aggregate:") {
		t.Errorf("root should preserve aggregate section, got:\n%s", root)
	}
}

func TestBuildSplitStems_PreservesValidate(t *testing.T) {
	existing := &rules.StemFile{
		Version: 2,
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

	result := mustBuildSplitStems(t, "/tmp/test", existing, hierarchy)
	root := result.Stems[0].Content

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
		Version: 2,
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

	result := mustBuildSplitStems(t, "/tmp/test", existing, hierarchy)

	// Find child stem.
	var childContent string
	for _, f := range result.Stems {
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
		Version: 2,
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

	result := mustBuildSplitStems(t, "/tmp/test", existing, hierarchy)
	root := result.Stems[0].Content

	if strings.Contains(root, "scope:") {
		t.Errorf("root should not have scope when match is empty, got:\n%s", root)
	}
}

func TestBuildSplitStems_ChildVersionEmitted(t *testing.T) {
	existing := &rules.StemFile{
		Version: 2,
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
					DirPaths: []string{"E01-infra"},
				},
				OnlyHere: map[string]rules.SchemaField{
					"id": {Type: "sequence", Prefix: "E", Digits: 2},
				},
			},
			{
				Level: infer.Level{
					Prefix: "F", Digits: 2, Depth: 1,
					DirPaths: []string{"E01-infra/F01-net"},
				},
				OnlyHere: map[string]rules.SchemaField{
					"id":   {Type: "sequence", Prefix: "F", Digits: 2},
					"tipo": {Type: "enum", Values: []string{"feature", "historia"}},
				},
			},
		},
	)

	result := mustBuildSplitStems(t, "/proj", existing, hierarchy)
	if len(result.Stems) < 2 {
		t.Fatalf("expected root plus at least one child stem, got %d", len(result.Stems))
	}

	for _, sf := range result.Stems {
		parsed, err := rules.ParseStem(sf.Path, []byte(sf.Content))
		if err != nil {
			t.Fatalf("ParseStem(%s) failed: %v\ncontent:\n%s", sf.Path, err, sf.Content)
		}
		if parsed.Version != 2 {
			t.Errorf("%s: expected Version 2, got %d\ncontent:\n%s", sf.Path, parsed.Version, sf.Content)
		}
		if !strings.HasPrefix(sf.Content, "version: 2\n") {
			t.Errorf("%s: expected version on first line, got:\n%s", sf.Path, sf.Content)
		}
	}
}

func TestBuildSplitStems_RootPreservation(t *testing.T) {
	tests := []struct {
		name     string
		root     bool
		wantRoot bool
	}{
		{name: "root marker present", root: true, wantRoot: true},
		{name: "root marker absent", root: false, wantRoot: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			existing := &rules.StemFile{
				Version: 2,
				Root:    tt.root,
				Scope:   rules.Scope{Match: "*.md"},
				Schema: map[string]rules.SchemaField{
					"id":     {Type: "sequence", Prefix: "E", Digits: 2},
					"estado": {Type: "enum", Values: []string{"Pending", "Done"}, Required: true},
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
							DirPaths: []string{"E01-infra"},
						},
						OnlyHere: map[string]rules.SchemaField{
							"id": {Type: "sequence", Prefix: "E", Digits: 2},
						},
					},
					{
						Level: infer.Level{
							Prefix: "F", Digits: 2, Depth: 1,
							DirPaths: []string{"E01-infra/F01-net"},
						},
						OnlyHere: map[string]rules.SchemaField{
							"id": {Type: "sequence", Prefix: "F", Digits: 2},
						},
					},
				},
			)

			result := mustBuildSplitStems(t, "/proj", existing, hierarchy)
			rootStem := result.Stems[0]

			parsed, err := rules.ParseStem(rootStem.Path, []byte(rootStem.Content))
			if err != nil {
				t.Fatalf("ParseStem failed: %v\ncontent:\n%s", err, rootStem.Content)
			}
			if parsed.Root != tt.wantRoot {
				t.Errorf("expected Root %v, got %v\ncontent:\n%s", tt.wantRoot, parsed.Root, rootStem.Content)
			}
			if tt.wantRoot && !strings.Contains(rootStem.Content, "root: true\n") {
				t.Errorf("expected literal 'root: true' in root stem, got:\n%s", rootStem.Content)
			}
			if !tt.wantRoot && strings.Contains(rootStem.Content, "root:") {
				t.Errorf("expected no root key when input root is false, got:\n%s", rootStem.Content)
			}

			// Child stems never carry the root marker.
			for _, sf := range result.Stems[1:] {
				if strings.Contains(sf.Content, "root:") {
					t.Errorf("child %s must not carry root marker, got:\n%s", sf.Path, sf.Content)
				}
			}
		})
	}
}

func TestBuildSplitStems_NestedAncestorBoundary(t *testing.T) {
	// Ancestor tree that must NOT be reachable from inside the project.
	ancestor := t.TempDir()
	if err := os.WriteFile(filepath.Join(ancestor, ".stem"),
		[]byte("version: 2\nschema:\n  ancestor_field:\n    type: string\n"), 0o644); err != nil {
		t.Fatalf("writing ancestor .stem: %v", err)
	}

	project := filepath.Join(ancestor, "project")
	nested := filepath.Join(project, "E01-infra", "F01-net")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("creating nested dirs: %v", err)
	}

	existing := &rules.StemFile{
		Version: 2,
		Root:    true,
		Scope:   rules.Scope{Match: "*.md"},
		Schema: map[string]rules.SchemaField{
			"id":     {Type: "sequence", Prefix: "E", Digits: 2},
			"estado": {Type: "enum", Values: []string{"Pending", "Done"}, Required: true},
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
					DirPaths: []string{"E01-infra"},
				},
				OnlyHere: map[string]rules.SchemaField{
					"id": {Type: "sequence", Prefix: "E", Digits: 2},
				},
			},
			{
				Level: infer.Level{
					Prefix: "F", Digits: 2, Depth: 1,
					DirPaths: []string{"E01-infra/F01-net"},
				},
				OnlyHere: map[string]rules.SchemaField{
					"id": {Type: "sequence", Prefix: "F", Digits: 2},
				},
			},
			{
				Level: infer.Level{
					Prefix: "S", Digits: 3, Depth: 2,
					DirPaths: []string{"E01-infra/F01-net/S001-dns"},
				},
				OnlyHere: map[string]rules.SchemaField{
					"id": {Type: "sequence", Prefix: "S", Digits: 3},
				},
			},
		},
	)

	result := mustBuildSplitStems(t, project, existing, hierarchy)
	for _, sf := range result.Stems {
		if err := os.MkdirAll(filepath.Dir(sf.Path), 0o755); err != nil {
			t.Fatalf("creating %s: %v", filepath.Dir(sf.Path), err)
		}
		if err := os.WriteFile(sf.Path, []byte(sf.Content), 0o644); err != nil {
			t.Fatalf("writing %s: %v", sf.Path, err)
		}
	}

	record := filepath.Join(nested, "README.md")
	if err := os.WriteFile(record, []byte("---\nestado: Pending\n---\n# F01\n"), 0o644); err != nil {
		t.Fatalf("writing record: %v", err)
	}

	entries, err := rules.WalkUp(record)
	if err != nil {
		t.Fatalf("WalkUp failed: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one stem entry")
	}

	ancestorStem := filepath.Join(ancestor, ".stem")
	for _, e := range entries {
		if e.Path == ancestorStem {
			t.Errorf("walk-up escaped the project root and included %s", e.Path)
		}
		rel, relErr := filepath.Rel(project, e.Path)
		if relErr != nil || strings.HasPrefix(rel, "..") {
			t.Errorf("stem %s lies outside the project root %s", e.Path, project)
		}
	}

	// The outermost entry must be the project root stem carrying the marker.
	if entries[0].Path != filepath.Join(project, ".stem") {
		t.Errorf("expected project root .stem first, got %s", entries[0].Path)
	}
	if !entries[0].Stem.Root {
		t.Error("expected project root .stem to carry root: true after split")
	}
}

// Ensure the extract.Record import is used (needed for infer.InferredSchema).
var _ = (*extract.Record)(nil)
