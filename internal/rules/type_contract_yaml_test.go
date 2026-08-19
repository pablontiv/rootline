package rules

import "testing"

func TestSchemaFieldYAMLMetadata_TypePresenceStates(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantActual string
		wantMeta   schemaFieldDeclarationMetadata
	}{
		{
			name:       "omitted type",
			body:       "source: body.h1\n",
			wantActual: "omitted",
			wantMeta:   schemaFieldDeclarationMetadata{},
		},
		{
			name:       "explicit empty type",
			body:       "type: \"\"\n",
			wantActual: "empty",
			wantMeta:   schemaFieldDeclarationMetadata{TypePresent: true},
		},
		{
			name:       "null type",
			body:       "type: null\n",
			wantActual: "null",
			wantMeta:   schemaFieldDeclarationMetadata{TypePresent: true, TypeNull: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := mustParseField(t, tt.body)
			if field.declaration.TypePresent != tt.wantMeta.TypePresent || field.declaration.TypeNull != tt.wantMeta.TypeNull {
				t.Fatalf("metadata=%+v want %+v", field.declaration, tt.wantMeta)
			}
			issues := ValidateFieldDeclaration("field", field)
			if len(issues) != 1 || issues[0].Code != "incomplete-type" || issues[0].Actual != tt.wantActual {
				t.Fatalf("issues=%+v, want incomplete-type actual %q", issues, tt.wantActual)
			}
		})
	}
}

func TestSchemaFieldYAMLMetadata_SourcePresenceStates(t *testing.T) {
	tests := []struct {
		name              string
		body              string
		wantSourcePresent bool
		wantSourceNull    bool
	}{
		{"omitted source", "type: string\n", false, false},
		{"explicit empty source", "type: string\nsource: \"\"\n", true, false},
		{"null source", "type: string\nsource: null\n", true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := mustParseField(t, tt.body)
			if field.declaration.SourcePresent != tt.wantSourcePresent || field.declaration.SourceNull != tt.wantSourceNull {
				t.Fatalf("metadata=%+v, want source present=%v null=%v", field.declaration, tt.wantSourcePresent, tt.wantSourceNull)
			}
			if issues := ValidateFieldDeclaration("field", field); len(issues) != 0 {
				t.Fatalf("source presence metadata should not make a valid string declaration fail: %+v", issues)
			}
		})
	}
}

func TestSchemaFieldYAMLMetadata_LegacyKeyPresenceStates(t *testing.T) {
	tests := []struct {
		name               string
		body               string
		wantHeadingPresent bool
		wantOrderedPresent bool
	}{
		{"heading value", "type: string\nheading: \"## Notes\"\n", true, false},
		{"heading null", "type: string\nheading: null\n", true, false},
		{"ordered value", "type: string\nordered: 1\n", false, true},
		{"ordered null", "type: string\nordered: null\n", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := mustParseField(t, tt.body)
			if field.declaration.HeadingPresent != tt.wantHeadingPresent || field.declaration.OrderedPresent != tt.wantOrderedPresent {
				t.Fatalf("metadata=%+v, want heading=%v ordered=%v", field.declaration, tt.wantHeadingPresent, tt.wantOrderedPresent)
			}
			if issues := ValidateFieldDeclaration("field", field); len(issues) != 1 || issues[0].Code != "legacy-type" {
				t.Fatalf("issues=%+v, want legacy-type", issues)
			}
		})
	}
}
