package main

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDocumentationContractHelpersRejectAdversarialFixtures(t *testing.T) {
	t.Run("malformed recognized YAML fence", func(t *testing.T) {
		if _, err := parseDocumentationYAML("schema: [broken"); err == nil {
			t.Fatal("malformed YAML fence parsed successfully")
		}
	})

	t.Run("legacy type applies only to type values", func(t *testing.T) {
		var node yaml.Node
		if err := yaml.Unmarshal([]byte("example: section\ntype: string\n"), &node); err != nil {
			t.Fatal(err)
		}
		if isLegacyTypeScalar("example", node.Content[0].Content[1].Value) {
			t.Fatal("ordinary scalar section was treated as a legacy type")
		}
		if !isLegacyTypeScalar("type", "section") || !isLegacyTypeScalar("type", "bool") || isLegacyTypeScalar("type", "string") {
			t.Fatal("legacy type classifier does not recognize exactly the retired type values")
		}
	})

	t.Run("physical stem source is rejected only for field objects", func(t *testing.T) {
		if !isPhysicalStemFieldSource(map[string]any{"type": "string"}, "docs/.stem") {
			t.Fatal("physical .stem source on a field object was not detected")
		}
		if isPhysicalStemFieldSource(map[string]any{"message": "diagnostic", "severity": "error", "rule": "required"}, "docs/.stem") {
			t.Fatal("diagnostic source was incorrectly treated as field provenance")
		}
	})

	t.Run("missing explain field and origin are not accepted", func(t *testing.T) {
		payload := map[string]any{"fields": []any{map[string]any{"name": "notes"}}}
		if _, ok := fieldOrigin(payload, "notes"); ok {
			t.Fatal("field without origin was accepted")
		}
		if _, ok := fieldOrigin(payload, "total"); ok {
			t.Fatal("missing explain field was accepted")
		}
	})

	t.Run("extra roadmap field breaks exact schema comparison", func(t *testing.T) {
		want := map[string]any{"id": map[string]any{"type": "sequence"}}
		got := map[string]any{"id": map[string]any{"type": "sequence"}, "extra": map[string]any{"type": "string"}}
		if documentationExactEqual(got, want) {
			t.Fatal("extra roadmap field was accepted by exact comparison")
		}
	})
}
