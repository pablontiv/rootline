package rules

import "testing"

func TestFieldCompatibility_TypeAlgebra(t *testing.T) {
	tests := []struct {
		name           string
		parent, child  SchemaField
		wantConstraint string
	}{
		{"same type", SchemaField{Type: "string"}, SchemaField{Type: "string"}, ""},
		{"string to enum narrows", SchemaField{Type: "string"}, SchemaField{Type: "enum", Values: []string{"draft"}}, ""},
		{"enum subset narrows", SchemaField{Type: "enum", Values: []string{"draft", "done"}}, SchemaField{Type: "enum", Values: []string{"draft"}}, ""},
		{"enum extension widens", SchemaField{Type: "enum", Values: []string{"draft"}}, SchemaField{Type: "enum", Values: []string{"draft", "done"}}, "values"},
		{"enum to string widens", SchemaField{Type: "enum", Values: []string{"draft"}}, SchemaField{Type: "string"}, "type"},
		{"unequal types conflict", SchemaField{Type: "string"}, SchemaField{Type: "list"}, "type"},
		{"required loosening", SchemaField{Type: "string", Required: true}, SchemaField{Type: "string"}, "required"},
		{"severity loosening", SchemaField{Type: "string", Severity: "error"}, SchemaField{Type: "string", Severity: "warn"}, "severity"},
		{"severity tightening", SchemaField{Type: "string", Severity: "warn"}, SchemaField{Type: "string", Severity: "error"}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := CheckFieldCompatibility(tt.parent, tt.child)
			if tt.wantConstraint == "" {
				if len(issues) != 0 {
					t.Fatalf("issues=%+v, want none", issues)
				}
				return
			}
			if len(issues) == 0 {
				t.Fatalf("got no issues, want %q", tt.wantConstraint)
			}
			if issues[0].Constraint != tt.wantConstraint {
				t.Fatalf("constraint=%q issues=%+v, want %q", issues[0].Constraint, issues, tt.wantConstraint)
			}
			if issues[0].Operation == "field-override" {
				t.Fatalf("generic override noise is not a compatibility issue: %+v", issues[0])
			}
		})
	}
}

func TestFieldCompatibility_SourceMatrix(t *testing.T) {
	parent := SchemaField{Type: "string", Extract: `body.section["## Summary"]`}
	tests := []struct {
		name           string
		child          SchemaField
		wantConstraint string
	}{
		{"omitted inherits", SchemaField{Type: "string"}, ""},
		{"same source", SchemaField{Type: "string", Extract: parent.Extract}, ""},
		{"changed", SchemaField{Type: "string", Extract: `body.section["## Context"]`}, "source"},
		{"empty removed", mustParseField(t, "type: string\nsource: \"\"\n"), "source"},
		{"null removed", mustParseField(t, "type: string\nsource: null\n"), "source"},
		{"added source changes stable origin", SchemaField{Type: "string", Extract: `body.h1`}, "source"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := CheckFieldCompatibility(parent, tt.child)
			if tt.wantConstraint == "" {
				if len(issues) != 0 {
					t.Fatalf("issues=%+v, want none", issues)
				}
				return
			}
			if len(issues) == 0 || issues[0].Constraint != tt.wantConstraint {
				t.Fatalf("issues=%+v, want first constraint %q", issues, tt.wantConstraint)
			}
		})
	}
}

func TestFieldCompatibility_RejectsAddingSourceToStableFrontmatterField(t *testing.T) {
	issues := CheckFieldCompatibility(
		SchemaField{Type: "string"},
		SchemaField{Type: "string", Extract: `body.h1`},
	)
	if len(issues) == 0 || issues[0].Constraint != "source" {
		t.Fatalf("issues=%+v, want source compatibility issue", issues)
	}
}
