package infer

import (
	"testing"

	"github.com/pablontiv/rootline/internal/rules"
)

func TestDetectMissingDomains_FlagsMissingDomain(t *testing.T) {
	stem := &rules.StemFile{
		Path: "docs/.stem",
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"draft", "active"}, Source: "docs/.stem"},
			"titulo": {Type: "string", Domain: "title", Source: "docs/.stem"},
		},
	}
	got := DetectMissingDomains(stem)
	if len(got) != 1 {
		t.Fatalf("expected 1 inference, got %d", len(got))
	}
	if got[0].Type != "missing_domain" {
		t.Errorf("expected type missing_domain, got %s", got[0].Type)
	}
	if got[0].Field != "estado" {
		t.Errorf("expected field estado, got %s", got[0].Field)
	}
	if got[0].Source != "docs/.stem" {
		t.Errorf("expected source docs/.stem, got %s", got[0].Source)
	}
}

func TestDetectMissingDomains_SkipsSections(t *testing.T) {
	stem := &rules.StemFile{
		Path: "docs/.stem",
		Schema: map[string]rules.SchemaField{
			"contexto": {Type: "section", Heading: "Contexto", Source: "docs/.stem"},
		},
	}
	got := DetectMissingDomains(stem)
	if len(got) != 0 {
		t.Fatalf("expected 0 inferences for section fields, got %d", len(got))
	}
}

func TestDetectMissingDomains_AllHaveDomain(t *testing.T) {
	stem := &rules.StemFile{
		Path: "docs/.stem",
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Domain: "lifecycle_state", Source: "docs/.stem"},
			"titulo": {Type: "string", Domain: "title", Source: "docs/.stem"},
		},
	}
	got := DetectMissingDomains(stem)
	if len(got) != 0 {
		t.Fatalf("expected 0 inferences, got %d", len(got))
	}
}

func TestDetectMissingDomains_NilStem(t *testing.T) {
	got := DetectMissingDomains(nil)
	if len(got) != 0 {
		t.Fatalf("expected 0 inferences for nil stem, got %d", len(got))
	}
}
