package e2e

import (
	"context"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/query"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestDomain_TypeInference(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": `version: 2
schema:
  estado:
    domain: lifecycle_state
    values: [draft, active, closed]
  titulo:
    domain: title
`,
		"doc.md": "---\nestado: active\ntitulo: Test Doc\n---\n# Test\n",
	})

	entries, err := rules.WalkUp(root)
	if err != nil {
		t.Fatalf("WalkUp: %v", err)
	}
	effective := rules.MergeStemFiles(entries)
	if effective == nil {
		t.Fatal("no effective stem")
	}

	// Type should be inferred from domain
	estado := effective.Schema["estado"]
	if estado.Type != "enum" {
		t.Errorf("estado.Type = %q, want %q (inferred from lifecycle_state)", estado.Type, "enum")
	}
	if estado.Domain != "lifecycle_state" {
		t.Errorf("estado.Domain = %q, want %q", estado.Domain, "lifecycle_state")
	}

	titulo := effective.Schema["titulo"]
	if titulo.Type != "string" {
		t.Errorf("titulo.Type = %q, want %q (inferred from title)", titulo.Type, "string")
	}
}

func TestDomain_AliasQuery(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": `version: 2
schema:
  estado:
    domain: lifecycle_state
    values: [draft, active, closed]
`,
		"a.md": "---\nestado: active\n---\n# A\n",
		"b.md": "---\nestado: draft\n---\n# B\n",
		"c.md": "---\nestado: closed\n---\n# C\n",
	})

	ctx := context.Background()
	reg := extract.NewRegistry()
	records, err := index.Scan(ctx, root, reg, index.WithScopeResolver(buildScopeResolver()))
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	// Build domain aliases
	entries, err := rules.WalkUp(root)
	if err != nil {
		t.Fatalf("WalkUp: %v", err)
	}
	effective := rules.MergeStemFiles(entries)
	aliases := rules.DomainAliases(effective.Schema, root)

	// Query using domain alias instead of field name
	prog, err := query.CompileWhere("lifecycle_state == 'active'")
	if err != nil {
		t.Fatalf("CompileWhere: %v", err)
	}

	var matched []*extract.Record
	for _, rec := range records {
		m, err := query.MatchRecord(ctx, prog, rec, aliases)
		if err != nil {
			t.Fatalf("MatchRecord: %v", err)
		}
		if m {
			matched = append(matched, rec)
		}
	}

	if len(matched) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matched))
	}
}

func TestDomain_FindFieldByDomain_ScopedMatch(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": `version: 2
schema:
  estado_epic:
    domain: lifecycle_state
    type: enum
    values: [draft, active]
    match: "E*"
  estado_task:
    domain: lifecycle_state
    type: enum
    values: [pending, done]
    match: "T*"
  titulo:
    domain: title
    type: string
`,
		"E01-test/README.md":    "---\nestado_epic: active\ntitulo: Epic\n---\n# Epic\n",
		"E01-test/T001-task.md": "---\nestado_task: pending\ntitulo: Task\n---\n# Task\n",
	})

	entries, err := rules.WalkUp(root)
	if err != nil {
		t.Fatalf("WalkUp: %v", err)
	}
	effective := rules.MergeStemFiles(entries)

	// For an epic-level record, lifecycle_state should resolve to estado_epic
	name, found := rules.FindFieldByDomain(effective.Schema, "lifecycle_state", "E01-test/README.md")
	if !found || name != "estado_epic" {
		t.Errorf("FindFieldByDomain(lifecycle_state, E01) = (%q, %v), want (estado_epic, true)", name, found)
	}

	// For a task-level record, lifecycle_state should resolve to estado_task
	name, found = rules.FindFieldByDomain(effective.Schema, "lifecycle_state", "E01-test/T001-task.md")
	if !found || name != "estado_task" {
		t.Errorf("FindFieldByDomain(lifecycle_state, T001) = (%q, %v), want (estado_task, true)", name, found)
	}
}

func TestDomain_BackwardCompatibility(t *testing.T) {
	// Project WITHOUT any domain annotations — should work identically
	root := setupProject(t, map[string]string{
		".stem": `version: 2
schema:
  estado:
    type: enum
    values: [draft, active, closed]
    required: true
  titulo:
    type: string
`,
		"a.md": "---\nestado: active\ntitulo: Test\n---\n# A\n",
		"b.md": "---\nestado: draft\ntitulo: Test2\n---\n# B\n",
	})

	ctx := context.Background()
	reg := extract.NewRegistry()
	records, err := index.Scan(ctx, root, reg, index.WithScopeResolver(buildScopeResolver()))
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}

	// Standard query with field name should still work
	prog, err := query.CompileWhere("estado == 'active'")
	if err != nil {
		t.Fatalf("CompileWhere: %v", err)
	}

	var matched int
	for _, rec := range records {
		m, err := query.MatchRecord(ctx, prog, rec, nil)
		if err != nil {
			t.Fatalf("MatchRecord: %v", err)
		}
		if m {
			matched++
		}
	}

	if matched != 1 {
		t.Errorf("expected 1 match for estado=='active', got %d", matched)
	}

	// Validation should still work
	entries, err := rules.WalkUp(root)
	if err != nil {
		t.Fatalf("WalkUp: %v", err)
	}
	effective := rules.MergeStemFiles(entries)
	if effective == nil {
		t.Fatal("no effective stem")
	}

	// No domain should be set
	if effective.Schema["estado"].Domain != "" {
		t.Errorf("estado.Domain = %q, want empty", effective.Schema["estado"].Domain)
	}
}

func TestDomain_StemHealthValidation(t *testing.T) {
	root := setupProject(t, map[string]string{
		// Custom domain without explicit type — should fail stem-health
		".stem": `version: 2
schema:
  velocity:
    domain: acme/sprint_velocity
`,
	})

	result, err := rules.ValidateStemHealth(context.Background(), root)
	if err != nil {
		t.Fatalf("ValidateStemHealth: %v", err)
	}

	found := false
	for _, c := range result.Checks {
		if c.Name == "domain-custom-no-type" && c.Status == "fail" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected domain-custom-no-type error for custom domain without type")
	}
}
