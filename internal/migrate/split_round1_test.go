package migrate

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/infer"
	"github.com/pablontiv/rootline/internal/rules"
)

func mustFilterRulesSchemaByMatch(t *testing.T, schema map[string]*rules.SchemaField, recordPath string) map[string]*rules.SchemaField {
	t.Helper()
	filtered, err := rules.FilterSchemaByMatch(schema, recordPath)
	if err != nil {
		t.Fatal(err)
	}
	return filtered
}

func TestBuildSplitStems_RoundTripsCanonicalRootAndChildFields(t *testing.T) {
	existing := &rules.StemFile{
		Version: 2,
		Root:    true,
		Schema: map[string]rules.SchemaField{
			"id":           {Type: "sequence", Prefix: "E", Digits: 2},
			"common_notes": {Type: "string", Extract: `body.section["## Common: note #1"]`, Required: true, Excludes: &rules.ExcludeRule{Match: "archive:/** #old"}},
			"epic_code": {
				Type:     "sequence",
				Extract:  `body.h1`,
				Prefix:   "E",
				Digits:   3,
				Severity: "warn",
				Match:    &rules.FieldMatch{Patterns: []string{"E*: active #1", "E[[]*: child: risky"}},
				RequiredMatch: &rules.FieldMatch{Patterns: []string{
					"E*: active #1",
				}},
			},
			"feature_code": {
				Type:    "sequence",
				Extract: `body.section["### Feature: notes #2"]`,
				Match: &rules.FieldMatch{Configs: map[string]any{
					"F*: active #1": map[string]any{"prefix": "F: #not-comment", "digits": 2},
				}},
			},
		},
	}
	hierarchy := makeHierarchy(
		map[string]rules.SchemaField{"common_notes": {Type: "string"}},
		[]infer.LevelSchema{
			{Level: infer.Level{Prefix: "E", Digits: 2, DirPaths: []string{"E01"}}, OnlyHere: map[string]rules.SchemaField{"id": {Type: "sequence", Prefix: "E", Digits: 2}, "epic_code": {Type: "sequence"}}},
			{Level: infer.Level{Prefix: "F", Digits: 2, DirPaths: []string{"E01/F01"}}, OnlyHere: map[string]rules.SchemaField{"id": {Type: "sequence", Prefix: "F", Digits: 2}, "feature_code": {Type: "sequence"}}},
		},
	)

	result, err := BuildSplitStems("/tmp/test", existing, hierarchy)
	if err != nil {
		t.Fatalf("BuildSplitStems returned error: %v", err)
	}
	if len(result.Stems) != 2 {
		t.Fatalf("expected root plus one child stem, got %d: %+v", len(result.Stems), result.Stems)
	}

	root := result.Stems[0]
	if got := strings.Count(root.Content, "epic_code:"); got != 1 {
		t.Fatalf("level-0 field should appear exactly once in root, got %d:\n%s", got, root.Content)
	}
	if strings.Contains(result.Stems[1].Content, "epic_code:") {
		t.Fatalf("level-0 field leaked into child stem:\n%s", result.Stems[1].Content)
	}
	parsedRoot, err := rules.ParseStem(root.Path, []byte(root.Content))
	if err != nil {
		t.Fatalf("root did not production-parse: %v\n%s", err, root.Content)
	}
	epic := parsedRoot.Schema["epic_code"]
	if epic.Extract != `body.h1` || epic.Prefix != "E" || epic.Digits != 3 || epic.Severity != "warn" || epic.RequiredMatch == nil || epic.Match == nil {
		t.Fatalf("level-0 canonical attrs not preserved: %+v\n%s", epic, root.Content)
	}
	common := parsedRoot.Schema["common_notes"]
	if common.Excludes == nil || common.Excludes.Match != "archive:/** #old" || common.Extract != `body.section["## Common: note #1"]` {
		t.Fatalf("root common canonical attrs not preserved: %+v\n%s", common, root.Content)
	}

	child := result.Stems[1]
	if !strings.HasSuffix(child.Path, filepath.Join("E01", ".stem")) {
		t.Fatalf("unexpected child path %s", child.Path)
	}
	parsedChild, err := rules.ParseStem(child.Path, []byte(child.Content))
	if err != nil {
		t.Fatalf("child did not production-parse: %v\n%s", err, child.Content)
	}
	feature := parsedChild.Schema["feature_code"]
	cfg := feature.Match.Configs["F*: active #1"].(map[string]any)
	if feature.Extract != `body.section["### Feature: notes #2"]` || cfg["prefix"] != "F: #not-comment" || cfg["digits"] != 2 {
		t.Fatalf("child canonical attrs not preserved: field=%+v cfg=%#v\n%s", feature, cfg, child.Content)
	}
}

func TestBuildSplitStems_RoundTripsEmptyMatchLists(t *testing.T) {
	existing, err := rules.ParseStem("source.stem", []byte(`version: 2
schema:
  never:
    type: string
    match: []
  sometimes_required:
    type: string
    required:
      match: []
`))
	if err != nil {
		t.Fatalf("fixture did not parse: %v", err)
	}
	hierarchy := makeHierarchy(
		map[string]rules.SchemaField{"never": {Type: "string"}, "sometimes_required": {Type: "string"}},
		[]infer.LevelSchema{{Level: infer.Level{Prefix: "E", Digits: 2, DirPaths: []string{"E01"}}, OnlyHere: map[string]rules.SchemaField{"id": {Type: "sequence", Prefix: "E", Digits: 2}}}},
	)

	result, err := BuildSplitStems("/tmp/test", existing, hierarchy)
	if err != nil {
		t.Fatalf("BuildSplitStems returned error: %v", err)
	}
	root := result.Stems[0]
	if !strings.Contains(root.Content, "    match: []\n") || !strings.Contains(root.Content, "    required:\n      match: []\n") {
		t.Fatalf("split root did not preserve empty match forms:\n%s", root.Content)
	}
	parsed, err := rules.ParseStem(root.Path, []byte(root.Content))
	if err != nil {
		t.Fatalf("split root did not production-parse: %v\n%s", err, root.Content)
	}
	never := parsed.Schema["never"]
	sometimesRequired := parsed.Schema["sometimes_required"]
	filtered := mustFilterRulesSchemaByMatch(t, map[string]*rules.SchemaField{
		"never":              &never,
		"sometimes_required": &sometimesRequired,
	}, filepath.Join("E01", "README.md"))
	if _, ok := filtered["never"]; ok {
		t.Fatalf("empty match widened during split: %#v", filtered["never"])
	}
	if got := filtered["sometimes_required"]; got == nil || got.Required {
		t.Fatalf("empty required match did not remain conditional after split: %#v", got)
	}
}

func TestBuildSplitStems_RejectsUnsupportedMatchConfigBeforeReturningStems(t *testing.T) {
	existing := &rules.StemFile{
		Version: 2,
		Schema: map[string]rules.SchemaField{
			"code": {Type: "sequence", Match: &rules.FieldMatch{Configs: map[string]any{"E*": "prefix: E"}}},
		},
	}
	hierarchy := makeHierarchy(map[string]rules.SchemaField{"code": {Type: "sequence"}}, []infer.LevelSchema{{Level: infer.Level{Prefix: "E", Digits: 2, DirPaths: []string{"E01"}}, OnlyHere: map[string]rules.SchemaField{"id": {Type: "sequence"}}}})

	result, err := BuildSplitStems("/tmp/test", existing, hierarchy)
	if err == nil {
		t.Fatalf("expected unsupported config error, got nil and result %+v", result)
	}
	if len(result.Stems) != 0 {
		t.Fatalf("expected transactional failure with no stems, got %+v", result.Stems)
	}
}
