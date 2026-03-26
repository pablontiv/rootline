package rules

import "testing"

func TestLookupDomain_Core(t *testing.T) {
	tests := []struct {
		name     string
		wantType string
	}{
		{"lifecycle_state", "enum"},
		{"record_type", "enum"},
		{"identifier", "sequence"},
		{"title", "string"},
		{"created_date", "string"},
		{"priority", "enum"},
		{"confidence", "enum"},
	}
	for _, tt := range tests {
		def := LookupDomain(tt.name)
		if def == nil {
			t.Errorf("LookupDomain(%q) = nil, want non-nil", tt.name)
			continue
		}
		if def.BaseType != tt.wantType {
			t.Errorf("LookupDomain(%q).BaseType = %q, want %q", tt.name, def.BaseType, tt.wantType)
		}
	}
}

func TestLookupDomain_Custom(t *testing.T) {
	def := LookupDomain("acme/sprint_velocity")
	if def != nil {
		t.Errorf("LookupDomain(custom) = %v, want nil", def)
	}
}

func TestLookupDomain_Unknown(t *testing.T) {
	def := LookupDomain("nonexistent_domain")
	if def != nil {
		t.Errorf("LookupDomain(unknown) = %v, want nil", def)
	}
}

func TestIsCustomDomain(t *testing.T) {
	if !IsCustomDomain("acme/velocity") {
		t.Error("IsCustomDomain(acme/velocity) = false, want true")
	}
	if IsCustomDomain("lifecycle_state") {
		t.Error("IsCustomDomain(lifecycle_state) = true, want false")
	}
}

func TestResolveDomainType(t *testing.T) {
	schema := map[string]SchemaField{
		"estado": {
			Domain: "lifecycle_state",
			Values: []string{"draft", "active"},
		},
		"titulo": {
			Domain: "title",
		},
		"id": {
			Domain: "identifier",
			Prefix: "T",
			Digits: 3,
		},
		"explicit_type": {
			Domain: "lifecycle_state",
			Type:   "string", // explicit type should NOT be overridden
			Values: []string{"a", "b"},
		},
		"custom": {
			Domain: "acme/sprint",
			// no type — custom domain, should stay empty
		},
		"no_domain": {
			Type: "enum",
		},
	}

	result := ResolveDomainType(schema)

	if result["estado"].Type != "enum" {
		t.Errorf("estado.Type = %q, want %q", result["estado"].Type, "enum")
	}
	if result["titulo"].Type != "string" {
		t.Errorf("titulo.Type = %q, want %q", result["titulo"].Type, "string")
	}
	if result["id"].Type != "sequence" {
		t.Errorf("id.Type = %q, want %q", result["id"].Type, "sequence")
	}
	if result["explicit_type"].Type != "string" {
		t.Errorf("explicit_type.Type = %q, want %q (should preserve explicit)", result["explicit_type"].Type, "string")
	}
	if result["custom"].Type != "" {
		t.Errorf("custom.Type = %q, want empty (custom domain, no inference)", result["custom"].Type)
	}
	if result["no_domain"].Type != "enum" {
		t.Errorf("no_domain.Type = %q, want %q", result["no_domain"].Type, "enum")
	}
}

func TestResolveDomainType_InParseStem(t *testing.T) {
	content := []byte(`
version: 2
schema:
  estado:
    domain: lifecycle_state
    values: [draft, active, closed]
  titulo:
    domain: title
`)
	stem, err := ParseStem("test/.stem", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stem.Schema["estado"].Type != "enum" {
		t.Errorf("estado.Type = %q, want %q (inferred from lifecycle_state)", stem.Schema["estado"].Type, "enum")
	}
	if stem.Schema["titulo"].Type != "string" {
		t.Errorf("titulo.Type = %q, want %q (inferred from title)", stem.Schema["titulo"].Type, "string")
	}
}

func TestFindFieldByDomain(t *testing.T) {
	schema := map[string]SchemaField{
		"estado_epic": {
			Domain: "lifecycle_state",
			Type:   "enum",
			Match:  &FieldMatch{Patterns: []string{"E*"}},
		},
		"estado_feature": {
			Domain: "lifecycle_state",
			Type:   "enum",
			Match:  &FieldMatch{Patterns: []string{"F*"}},
		},
		"titulo": {
			Domain: "title",
			Type:   "string",
		},
	}

	// Epic scope should find estado_epic
	name, found := FindFieldByDomain(schema, "lifecycle_state", "docs/epics/E01-infra/README.md")
	if !found || name != "estado_epic" {
		t.Errorf("FindFieldByDomain(lifecycle_state, E01) = (%q, %v), want (estado_epic, true)", name, found)
	}

	// Feature scope should find estado_feature
	name, found = FindFieldByDomain(schema, "lifecycle_state", "docs/epics/E01-infra/F01-auth/README.md")
	if !found || name != "estado_feature" {
		t.Errorf("FindFieldByDomain(lifecycle_state, F01) = (%q, %v), want (estado_feature, true)", name, found)
	}

	// Title has no match, applies everywhere
	name, found = FindFieldByDomain(schema, "title", "docs/epics/E01-infra/README.md")
	if !found || name != "titulo" {
		t.Errorf("FindFieldByDomain(title, E01) = (%q, %v), want (titulo, true)", name, found)
	}

	// Nonexistent domain
	_, found = FindFieldByDomain(schema, "nonexistent", "docs/epics/E01-infra/README.md")
	if found {
		t.Error("FindFieldByDomain(nonexistent) = true, want false")
	}
}

func TestDomainAliases(t *testing.T) {
	schema := map[string]SchemaField{
		"estado": {
			Domain: "lifecycle_state",
			Type:   "enum",
		},
		"tipo": {
			Domain: "record_type",
			Type:   "enum",
		},
		"titulo": {
			Type: "string",
			// no domain
		},
	}

	aliases := DomainAliases(schema, "docs/README.md")

	if aliases["lifecycle_state"] != "estado" {
		t.Errorf("aliases[lifecycle_state] = %q, want %q", aliases["lifecycle_state"], "estado")
	}
	if aliases["record_type"] != "tipo" {
		t.Errorf("aliases[record_type] = %q, want %q", aliases["record_type"], "tipo")
	}
	if _, exists := aliases["title"]; exists {
		t.Error("aliases[title] should not exist (titulo has no domain)")
	}
	if len(aliases) != 2 {
		t.Errorf("len(aliases) = %d, want 2", len(aliases))
	}
}
