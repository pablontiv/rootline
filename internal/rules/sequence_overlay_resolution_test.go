package rules

import "testing"

func TestValidateFieldDeclarationSequenceEffectiveOverlayAcceptsResolvableConfigs(t *testing.T) {
	tests := []struct {
		name       string
		fieldYAML  string
		recordPath string
		wantPrefix string
		wantDigits int
	}{
		{
			name: "complete global sequence declaration",
			fieldYAML: `type: sequence
prefix: ID-
digits: 3
`,
			recordPath: "anything.md",
			wantPrefix: "ID-",
			wantDigits: 3,
		},
		{
			name: "complete per pattern declarations without global fallback",
			fieldYAML: `type: sequence
match:
  "O*": {prefix: O, digits: 2}
  "T*": {prefix: T, digits: 3}
`,
			recordPath: "O01-objective/T001-task.md",
			wantPrefix: "T",
			wantDigits: 3,
		},
		{
			name: "complete global fallback with partial pattern override",
			fieldYAML: `type: sequence
prefix: O
digits: 2
match:
  "T*": {prefix: T}
`,
			recordPath: "O01-objective/T001-task.md",
			wantPrefix: "T",
			wantDigits: 2,
		},
		{
			name: "match scoped partial global completed by every pattern config",
			fieldYAML: `type: sequence
prefix: O
match:
  "O*": {digits: 2}
  "T*": {prefix: T, digits: 3}
`,
			recordPath: "O01-objective/T001-task.md",
			wantPrefix: "T",
			wantDigits: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := mustParseField(t, tt.fieldYAML)
			if issues := ValidateFieldDeclaration("id", field); len(issues) > 0 {
				t.Fatalf("got declaration issues=%+v, want none", issues)
			}
			resolved, ok := resolveSequenceForPath(t, field, tt.recordPath)
			if !ok {
				t.Fatalf("field did not apply to %s", tt.recordPath)
			}
			if resolved.Prefix != tt.wantPrefix || resolved.Digits != tt.wantDigits {
				t.Fatalf("resolved prefix=%q digits=%d, want prefix=%q digits=%d", resolved.Prefix, resolved.Digits, tt.wantPrefix, tt.wantDigits)
			}
		})
	}
}

func resolveSequenceForPath(t *testing.T, field SchemaField, recordPath string) (SchemaField, bool) {
	t.Helper()
	filtered := mustFilterSchemaByMatch(t, map[string]*SchemaField{"id": &field}, recordPath)
	resolved, ok := filtered["id"]
	if !ok {
		return SchemaField{}, false
	}
	return *resolved, true
}
