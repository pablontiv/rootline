package stemyaml

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/rules"
)

func TestStrictSequenceValidationRejectsBeforeDestinationBytes(t *testing.T) {
	maxUint64 := ^uint64(0)
	tests := []struct {
		name       string
		field      rules.SchemaField
		fieldError string
		attrsError string
	}{
		{
			name:       "incomplete declaration",
			field:      rules.SchemaField{Type: "sequence"},
			fieldError: `invalid field declaration: sequence field "id" must declare prefix and positive digits`,
			attrsError: `invalid field declaration: sequence field "field" must declare prefix and positive digits`,
		},
		{
			name:       "nil config",
			field:      rules.SchemaField{Type: "sequence", Prefix: "T", Digits: 2, Match: &rules.FieldMatch{Configs: map[string]any{"T*": nil}}},
			fieldError: `invalid field declaration: sequence field "id" must declare prefix and positive digits`,
			attrsError: `invalid field declaration: sequence field "field" must declare prefix and positive digits`,
		},
		{
			name:       "scalar config",
			field:      rules.SchemaField{Type: "sequence", Prefix: "T", Digits: 2, Match: &rules.FieldMatch{Configs: map[string]any{"T*": "prefix: T"}}},
			fieldError: `invalid field declaration: sequence field "id" must declare prefix and positive digits`,
			attrsError: `invalid field declaration: sequence field "field" must declare prefix and positive digits`,
		},
		{
			name:       "unsupported key",
			field:      rules.SchemaField{Type: "sequence", Match: &rules.FieldMatch{Configs: map[string]any{"T*": map[string]any{"prefix": "T", "digits": 2, "suffix": true}}}},
			fieldError: `invalid field declaration: sequence field "id" must declare prefix and positive digits: match "T*": unsupported sequence config key(s): suffix`,
			attrsError: `invalid field declaration: sequence field "field" must declare prefix and positive digits: match "T*": unsupported sequence config key(s): suffix`,
		},
		{
			name:       "nonpositive digits",
			field:      rules.SchemaField{Type: "sequence", Prefix: "T", Digits: 2, Match: &rules.FieldMatch{Configs: map[string]any{"T*": map[string]any{"prefix": "T", "digits": 0}}}},
			fieldError: `invalid field declaration: sequence field "id" must declare prefix and positive digits: match "T*": digits must be a positive integer`,
			attrsError: `invalid field declaration: sequence field "field" must declare prefix and positive digits: match "T*": digits must be a positive integer`,
		},
		{
			name:       "floating digits",
			field:      rules.SchemaField{Type: "sequence", Prefix: "T", Digits: 2, Match: &rules.FieldMatch{Configs: map[string]any{"T*": map[string]any{"prefix": "T", "digits": 2.0}}}},
			fieldError: `invalid field declaration: sequence field "id" must declare prefix and positive digits: match "T*": digits must be a positive integer`,
			attrsError: `invalid field declaration: sequence field "field" must declare prefix and positive digits: match "T*": digits must be a positive integer`,
		},
		{
			name:       "overflow digits",
			field:      rules.SchemaField{Type: "sequence", Prefix: "T", Digits: 2, Match: &rules.FieldMatch{Configs: map[string]any{"T*": map[string]any{"prefix": "T", "digits": maxUint64}}}},
			fieldError: `invalid field declaration: sequence field "id" must declare prefix and positive digits: match "T*": digits must be a positive integer`,
			attrsError: `invalid field declaration: sequence field "field" must declare prefix and positive digits: match "T*": digits must be a positive integer`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"/field", func(t *testing.T) {
			assertAppendErrorLeavesBuilder(t, tt.fieldError, func(b *strings.Builder) error {
				return AppendSchemaField(b, "id", tt.field)
			})
		})
		t.Run(tt.name+"/attrs", func(t *testing.T) {
			assertAppendErrorLeavesBuilder(t, tt.attrsError, func(b *strings.Builder) error {
				return AppendSchemaFieldAttrs(b, tt.field)
			})
		})
	}
}

func TestAppendSchemaFieldAttrsWritesCanonicalAttributesTransactionally(t *testing.T) {
	field := rules.SchemaField{
		Type:     "sequence",
		Required: true,
		Values:   []string{"Todo", "Done"},
		Default:  "Todo",
		Severity: "warn",
		Extract:  "body.h1",
		Prefix:   "ID-",
		Digits:   3,
		Excludes: &rules.ExcludeRule{Match: "archive/**"},
		Match: &rules.FieldMatch{Configs: map[string]any{
			"ID*": map[string]any{},
		}},
	}

	const sentinel = "prefix already written\n"
	var b strings.Builder
	b.WriteString(sentinel)

	if err := AppendSchemaFieldAttrs(&b, field); err != nil {
		t.Fatalf("AppendSchemaFieldAttrs returned error: %v", err)
	}
	want := sentinel + "    type: sequence\n" +
		"    required: true\n" +
		"    values: [Todo, Done]\n" +
		"    default: Todo\n" +
		"    severity: warn\n" +
		"    source: body.h1\n" +
		"    prefix: ID-\n" +
		"    digits: 3\n" +
		"    excludes:\n" +
		"      match: archive/**\n" +
		"    match:\n" +
		"      \"ID*\": {}\n"
	if b.String() != want {
		t.Fatalf("canonical attrs output mismatch:\ngot:\n%swant:\n%s", b.String(), want)
	}

	stem, err := rules.ParseStem("attrs.stem", []byte("version: 2\nschema:\n  id:\n"+strings.TrimPrefix(b.String(), sentinel)))
	if err != nil {
		t.Fatalf("canonical attrs did not parse: %v\n%s", err, b.String())
	}
	got := stem.Schema["id"]
	if !got.Required || got.Type != field.Type || got.Default != field.Default || got.Severity != field.Severity || got.Extract != field.Extract || got.Prefix != field.Prefix || got.Digits != field.Digits {
		t.Fatalf("parsed attrs not preserved: got %+v", got)
	}
	if got.Excludes == nil || got.Excludes.Match != field.Excludes.Match {
		t.Fatalf("parsed excludes not preserved: got %#v", got.Excludes)
	}
	if cfg, ok := got.Match.Configs["ID*"].(map[string]any); !ok || len(cfg) != 0 {
		t.Fatalf("empty match config was not preserved as empty map: %#v", got.Match.Configs["ID*"])
	}
}

func TestAppendSchemaFieldPreservesAcceptedNativeIntegerConfigDigits(t *testing.T) {
	tests := []struct {
		pattern string
		prefix  string
		digits  any
		want    int
	}{
		{pattern: "I*", prefix: "I", digits: int(2), want: 2},
		{pattern: "U*", prefix: "U", digits: uint(3), want: 3},
		{pattern: "L*", prefix: "L", digits: uint64(4), want: 4},
	}

	configs := make(map[string]any, len(tests))
	for _, tt := range tests {
		configs[tt.pattern] = map[string]any{"prefix": tt.prefix, "digits": tt.digits}
	}
	field := rules.SchemaField{Type: "sequence", Match: &rules.FieldMatch{Configs: configs}}

	var b strings.Builder
	if err := AppendSchemaField(&b, "id", field); err != nil {
		t.Fatalf("AppendSchemaField returned error: %v", err)
	}
	out := "version: 2\nschema:\n" + b.String()
	stem, err := rules.ParseStem("native-digits.stem", []byte(out))
	if err != nil {
		t.Fatalf("serialized native digits did not parse: %v\n%s", err, out)
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%T", tt.digits), func(t *testing.T) {
			cfg := stem.Schema["id"].Match.Configs[tt.pattern].(map[string]any)
			if cfg["prefix"] != tt.prefix || cfg["digits"] != tt.want {
				t.Fatalf("config %s not preserved: %#v", tt.pattern, cfg)
			}
			if !strings.Contains(b.String(), fmt.Sprintf("      digits: %d\n", tt.want)) {
				t.Fatalf("serialized digits value %d missing from output:\n%s", tt.want, b.String())
			}
		})
	}
}

func TestAppendSchemaFieldAndAttrsLeaveDestinationUnchangedOnSerializerErrors(t *testing.T) {
	tests := []struct {
		name  string
		field rules.SchemaField
		want  string
	}{
		{
			name:  "values multiline scalar",
			field: rules.SchemaField{Type: "string", Values: []string{"line one\nline two"}},
			want:  `serializing values: multiline scalar "line one\nline two" is unsupported`,
		},
		{
			name:  "source multiline scalar",
			field: rules.SchemaField{Type: "string", Extract: "body.h1\nbody.h2"},
			want:  `serializing source: multiline scalar "body.h1\nbody.h2" is unsupported`,
		},
		{
			name:  "prefix multiline scalar",
			field: rules.SchemaField{Type: "string", Prefix: "A\nB"},
			want:  `serializing prefix: multiline scalar "A\nB" is unsupported`,
		},
		{
			name:  "excludes multiline scalar",
			field: rules.SchemaField{Type: "string", Excludes: &rules.ExcludeRule{Match: "old\nnew"}},
			want:  `serializing excludes: multiline scalar "old\nnew" is unsupported`,
		},
		{
			name:  "required match malformed config",
			field: rules.SchemaField{Type: "string", RequiredMatch: &rules.FieldMatch{Configs: map[string]any{"A*": nil}}},
			want:  `serializing required match: pattern "A*" config must be a map`,
		},
		{
			name:  "match malformed config",
			field: rules.SchemaField{Type: "string", Match: &rules.FieldMatch{Configs: map[string]any{"A*": "prefix: A"}}},
			want:  `serializing match: pattern "A*" config must be a map`,
		},
		{
			name:  "match unsupported config key",
			field: rules.SchemaField{Type: "string", Match: &rules.FieldMatch{Configs: map[string]any{"A*": map[string]any{"prefix": "A", "suffix": true}}}},
			want:  `serializing match: pattern "A*" config key "suffix" is unsupported`,
		},
		{
			name:  "match prefix not string",
			field: rules.SchemaField{Type: "string", Match: &rules.FieldMatch{Configs: map[string]any{"A*": map[string]any{"prefix": 7}}}},
			want:  `serializing match: pattern "A*" prefix must be a string`,
		},
		{
			name:  "match prefix multiline scalar",
			field: rules.SchemaField{Type: "string", Match: &rules.FieldMatch{Configs: map[string]any{"A*": map[string]any{"prefix": "A\nB"}}}},
			want:  `serializing match: pattern "A*" prefix: multiline scalar "A\nB" is unsupported`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"/field", func(t *testing.T) {
			assertAppendErrorLeavesBuilder(t, tt.want, func(b *strings.Builder) error {
				return AppendSchemaField(b, "status", tt.field)
			})
		})
		t.Run(tt.name+"/attrs", func(t *testing.T) {
			assertAppendErrorLeavesBuilder(t, tt.want, func(b *strings.Builder) error {
				return AppendSchemaFieldAttrs(b, tt.field)
			})
		})
	}
}

func assertAppendErrorLeavesBuilder(t *testing.T, wantErr string, run func(*strings.Builder) error) {
	t.Helper()
	const sentinel = "preexisting bytes\n"
	var b strings.Builder
	b.WriteString(sentinel)

	err := run(&b)
	if err == nil {
		t.Fatalf("got nil error and output %q, want %q", b.String(), wantErr)
	}
	if err.Error() != wantErr {
		t.Fatalf("error mismatch:\ngot:  %q\nwant: %q", err.Error(), wantErr)
	}
	if b.String() != sentinel {
		t.Fatalf("destination changed on error: got %q, want %q", b.String(), sentinel)
	}
}
