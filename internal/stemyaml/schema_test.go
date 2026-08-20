package stemyaml

import (
	"strings"
	"testing"

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

func TestAppendSchemaField_RoundTripsCanonicalAttributes(t *testing.T) {
	field := rules.SchemaField{
		Type:     "sequence",
		Required: true,
		Values:   []string{"Todo", "Done: archived #1"},
		Default:  "Todo: triage #1",
		Severity: "warn",
		Extract:  `body.section["## Notes: risk #1"]`,
		Prefix:   "E",
		Digits:   3,
		Excludes: &rules.ExcludeRule{Match: "archive:/** #old"},
		Match: &rules.FieldMatch{Patterns: []string{
			"E*: active #1",
			"F[[]*: child: risky",
		}},
		RequiredMatch: &rules.FieldMatch{Patterns: []string{"E*: required #1"}},
	}

	var b strings.Builder
	if err := AppendSchemaField(&b, "state", field); err != nil {
		t.Fatalf("AppendSchemaField returned error: %v", err)
	}
	out := "version: 2\nschema:\n" + b.String()
	if strings.Contains(out, "required: true") {
		t.Fatalf("conditional required was serialized as unconditional bool:\n%s", out)
	}

	parsed, err := rules.ParseStem("roundtrip.stem", []byte(out))
	if err != nil {
		t.Fatalf("serialized stem did not parse: %v\n%s", err, out)
	}
	got := parsed.Schema["state"]
	if got.Type != field.Type || got.Default != field.Default || got.Severity != field.Severity || got.Extract != field.Extract || got.Prefix != field.Prefix || got.Digits != field.Digits {
		t.Fatalf("scalar attributes not preserved: got %+v", got)
	}
	if len(got.Values) != len(field.Values) || got.Values[1] != field.Values[1] {
		t.Fatalf("values not preserved: got %#v", got.Values)
	}
	if got.Excludes == nil || got.Excludes.Match != field.Excludes.Match {
		t.Fatalf("excludes not preserved: got %#v", got.Excludes)
	}
	if got.RequiredMatch == nil || len(got.RequiredMatch.Patterns) != 1 || got.RequiredMatch.Patterns[0] != field.RequiredMatch.Patterns[0] {
		t.Fatalf("required match not preserved: got %#v", got.RequiredMatch)
	}
	if got.Match == nil || len(got.Match.Patterns) != 2 || got.Match.Patterns[0] != field.Match.Patterns[0] || got.Match.Patterns[1] != field.Match.Patterns[1] {
		t.Fatalf("match patterns not preserved: got %#v", got.Match)
	}
}

func TestAppendSchemaField_RoundTripsYAMLSafeMatchConfigs(t *testing.T) {
	field := rules.SchemaField{
		Type: "sequence",
		Match: &rules.FieldMatch{Configs: map[string]any{
			"E*: active #1":       map[string]any{"prefix": "E: #not-comment", "digits": 2},
			"F[[]*: child: risky": map[string]any{"prefix": "F", "digits": 3},
		}},
	}

	var b strings.Builder
	if err := AppendSchemaField(&b, "id", field); err != nil {
		t.Fatalf("AppendSchemaField returned error: %v", err)
	}
	out := "version: 2\nschema:\n" + b.String()
	parsed, err := rules.ParseStem("configs.stem", []byte(out))
	if err != nil {
		t.Fatalf("serialized config match did not parse: %v\n%s", err, out)
	}
	cfg := parsed.Schema["id"].Match.Configs["E*: active #1"].(map[string]any)
	if cfg["prefix"] != "E: #not-comment" || cfg["digits"] != 2 {
		t.Fatalf("config not preserved: %#v\n%s", cfg, out)
	}
}

func TestAppendSchemaField_RoundTripsEmptyMatchSemantics(t *testing.T) {
	const emptyMatchStem = "version: 2\nschema:\n  never:\n    type: string\n    match: []\n  sometimes_required:\n    type: string\n    required:\n      match: []\n"
	stem, err := rules.ParseStem("empty-match.stem", []byte(emptyMatchStem))
	if err != nil {
		t.Fatalf("fixture did not parse: %v", err)
	}

	var b strings.Builder
	if err := AppendSchemaField(&b, "never", stem.Schema["never"]); err != nil {
		t.Fatalf("AppendSchemaField never returned error: %v", err)
	}
	if err := AppendSchemaField(&b, "sometimes_required", stem.Schema["sometimes_required"]); err != nil {
		t.Fatalf("AppendSchemaField sometimes_required returned error: %v", err)
	}
	out := "version: 2\nschema:\n" + b.String()
	if !strings.Contains(out, "    match: []\n") {
		t.Fatalf("empty field match was not serialized exactly:\n%s", out)
	}
	if !strings.Contains(out, "    required:\n      match: []\n") {
		t.Fatalf("empty required match was not serialized exactly:\n%s", out)
	}
	if got := strings.Count(out, "match: []\n"); got != 2 {
		t.Fatalf("nil absent match was not distinguished from the two explicit empty matches, got %d:\n%s", got, out)
	}

	parsed, err := rules.ParseStem("roundtrip-empty-match.stem", []byte(out))
	if err != nil {
		t.Fatalf("serialized empty match stem did not parse: %v\n%s", err, out)
	}
	never := parsed.Schema["never"]
	sometimesRequired := parsed.Schema["sometimes_required"]
	filtered := mustFilterRulesSchemaByMatch(t, map[string]*rules.SchemaField{
		"never":              &never,
		"sometimes_required": &sometimesRequired,
	}, "E01/file.md")
	if _, ok := filtered["never"]; ok {
		t.Fatalf("empty match widened to an applying field: %#v", filtered["never"])
	}
	if got := filtered["sometimes_required"]; got == nil || got.Required {
		t.Fatalf("empty required match did not remain conditional no-match: %#v", got)
	}
}

func TestAppendSchemaField_RejectsUnsupportedMatchConfigShapes(t *testing.T) {
	tests := []struct {
		name string
		cfg  any
	}{
		{name: "nil config", cfg: nil},
		{name: "scalar config", cfg: "prefix: E"},
		{name: "unsupported key", cfg: map[string]any{"prefix": "E", "unknown": true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := rules.SchemaField{Type: "sequence", Match: &rules.FieldMatch{Configs: map[string]any{"E*": tt.cfg}}}
			var b strings.Builder
			if err := AppendSchemaField(&b, "id", field); err == nil {
				t.Fatalf("expected unsupported config error, got nil and output:\n%s", b.String())
			}
		})
	}
}

func TestAppendSchemaField_LeavesDestinationUnchangedOnErrors(t *testing.T) {
	tests := []struct {
		name  string
		field rules.SchemaField
		attrs bool
	}{
		{name: "nil config", field: rules.SchemaField{Type: "sequence", Match: &rules.FieldMatch{Configs: map[string]any{"E*": nil}}}},
		{name: "scalar config", field: rules.SchemaField{Type: "sequence", Match: &rules.FieldMatch{Configs: map[string]any{"E*": "prefix: E"}}}},
		{name: "unsupported config key", field: rules.SchemaField{Type: "sequence", Match: &rules.FieldMatch{Configs: map[string]any{"E*": map[string]any{"prefix": "E", "unknown": true}}}}},
		{name: "render error", field: rules.SchemaField{Type: "string", Default: "line one\nline two"}},
		{name: "attrs bypass", field: rules.SchemaField{Type: "sequence", Match: &rules.FieldMatch{Configs: map[string]any{"E*": "prefix: E"}}}, attrs: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const sentinel = "preseed sentinel\n"
			var b strings.Builder
			b.WriteString(sentinel)
			err := AppendSchemaField(&b, "field", tt.field)
			if tt.attrs {
				err = AppendSchemaFieldAttrs(&b, tt.field)
			}
			if err == nil || b.String() != sentinel {
				t.Fatalf("got err=%v output=%q, want error and unchanged %q", err, b.String(), sentinel)
			}
		})
	}
}
