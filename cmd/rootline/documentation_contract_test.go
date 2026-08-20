package main

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/rules"
	"gopkg.in/yaml.v3"
)

func TestActiveDocumentationUsesCanonicalDialect(t *testing.T) {
	repo := documentationContractRepoRoot(t)
	for _, path := range activeDocumentationContractFiles() {
		text := string(documentationContractRead(t, repo, path))
		assertNoInlineLegacyDeclarations(t, path, contentOutsideFences(text))
		for _, block := range fencedBlocks(text) {
			switch block.lang {
			case "yaml", "yml":
				assertYAMLBlockHasNoLegacyDeclarations(t, path, block)
			case "json":
				assertJSONBlockHasCanonicalProvenance(t, path, block)
			}
		}
	}
}

func TestHistoricalDocumentationPairsLegacyAndCanonicalForms(t *testing.T) {
	repo := documentationContractRepoRoot(t)
	upgrade := string(documentationContractRead(t, repo, "docs/UPGRADE.md"))
	assertScopeContainsAll(t, "docs/UPGRADE.md canonical migration", sectionFrom(upgrade, "## Canonical Field Migration"), []string{
		"type: section", "heading:", "ordered:", "type: string", "source: body.section[\"## Notes\"]",
		"type: bool", "type: boolean", "enum:", "values:", "Ancestor-qualified selectors remain deferred to #190",
	})

	changelog := string(documentationContractRead(t, repo, "CHANGELOG.md"))
	assertScopeContainsAll(t, "CHANGELOG.md compatibility correction", paragraphFrom(changelog, "Compatibility correction (unreleased)"), []string{
		"type: section", "heading:", "ordered:", "type: string", "source: body.section[\"## Heading\"]",
		"type: bool", "type: boolean", "enum:", "values:", "ancestor-qualified selectors remain deferred to #190",
	})
}

func TestActiveSchemasUseCanonicalDeclarationsAndPassHealth(t *testing.T) {
	repo := documentationContractRepoRoot(t)
	for _, path := range []string{"docs/roadmap", "lines", "paused", "closed", "theories"} {
		result, err := rules.ValidateStemHealth(context.Background(), filepath.Join(repo, path))
		if err != nil {
			t.Fatalf("ValidateStemHealth %s: %v", path, err)
		}
		for _, check := range result.Checks {
			if check.Status == "fail" {
				t.Fatalf("%s active .stem health failed: %+v", path, check)
			}
		}
	}

	roadmapStem := documentationContractSchema(t, repo, "docs/roadmap/.stem")
	assertExact(t, "docs/roadmap/.stem schema", documentationContractMap(t, roadmapStem["schema"], "docs/roadmap/.stem schema"), map[string]any{
		"titulo":  map[string]any{"type": "string", "source": "body.h1"},
		"estado":  map[string]any{"type": "enum", "required": map[string]any{"match": []any{"T*"}}, "match": []any{"O*", "T*"}, "values": []any{"Pending", "Specified", "In Progress", "Completed", "Blocked", "On Hold", "Obsolete"}},
		"tipo":    map[string]any{"type": "enum", "required": map[string]any{"match": []any{"O*", "T*"}}, "match": []any{"O*", "T*"}, "values": []any{"outcome", "task"}},
		"id":      map[string]any{"type": "sequence", "match": map[string]any{"O*": map[string]any{"prefix": "O", "digits": 2}, "T*": map[string]any{"prefix": "T", "digits": 3}}},
		"is_done": map[string]any{"type": "boolean", "severity": "off"},
	})
	assertExact(t, "docs/roadmap derive", roadmapStem["derive"], map[string]any{"is_done": "estado in ['Completed', 'Obsolete']"})
	assertExact(t, "docs/roadmap validate", roadmapStem["validate"], []any{map[string]any{"field": "tipo", "rule": "non_empty"}})

	assertStemSchemaExact(t, repo, "lines/.stem", map[string]any{
		"tipo": map[string]any{"type": "enum", "values": []any{"question", "field-log"}, "required": true}, "estado": map[string]any{"type": "enum", "values": []any{"active", "paused", "closed"}}, "tipo_entendimiento": map[string]any{"type": "enum", "values": []any{"describe", "understand", "explore", "build"}}, "fecha_inicio": map[string]any{"type": "string"}, "linea": map[string]any{"type": "string"}, "ciclos_registrados": map[string]any{"type": "integer"},
	})
	assertStemSchemaExact(t, repo, "paused/.stem", map[string]any{
		"tipo": map[string]any{"type": "enum", "values": []any{"question", "field-log"}, "required": true}, "estado": map[string]any{"type": "enum", "values": []any{"active", "paused", "closed"}}, "tipo_entendimiento": map[string]any{"type": "enum", "values": []any{"describe", "understand", "explore", "build"}}, "fecha_inicio": map[string]any{"type": "string"}, "linea": map[string]any{"type": "string"}, "ciclos_registrados": map[string]any{"type": "integer"},
	})
	assertStemSchemaExact(t, repo, "closed/.stem", map[string]any{
		"tipo": map[string]any{"type": "enum", "values": []any{"question", "field-log", "closure"}, "required": true}, "estado": map[string]any{"type": "enum", "values": []any{"active", "paused", "closed"}}, "tipo_entendimiento": map[string]any{"type": "enum", "values": []any{"describe", "understand", "explore", "build"}}, "fecha_inicio": map[string]any{"type": "string"}, "fecha_cierre": map[string]any{"type": "string"}, "linea": map[string]any{"type": "string"}, "ciclos_registrados": map[string]any{"type": "integer"}, "ciclos_completados": map[string]any{"type": "integer"}, "teoria_emergente": map[string]any{"type": "string"},
	})
	assertStemSchemaExact(t, repo, "theories/.stem", map[string]any{
		"tipo": map[string]any{"type": "enum", "values": []any{"theory"}, "required": true}, "confianza": map[string]any{"type": "enum", "values": []any{"emergent", "developing", "consolidated"}, "required": true}, "fecha": map[string]any{"type": "string", "required": true}, "linea_origen": map[string]any{"type": "string", "required": true},
	})
}

func TestLivingDocumentationStatesBehaviorSpecificContract(t *testing.T) {
	repo := documentationContractRepoRoot(t)
	expect := map[string][]string{
		"README.md":                               {"exists` checks presence of an effective field", "source: body.section[\"## Summary\"]", "frontmatter remains an explicit override"},
		"CLAUDE.md":                               {"internal/extract/body_source.go", "internal/rules/field_source.go", "logical `source`", "physical `defined_in`"},
		"docs/init.md":                            {"0.80 threshold", "analyze` default threshold of 0.60", "Headings below the 0.80 threshold are omitted", "source: body.section[\"## Summary\"]"},
		"docs/new.md":                             {"--dry-run", "--force", "# New Endpoint", "tipo:  # [outcome, task]", "Required enum fields without a default are not written as a successful scaffold. `new` renders no invented first value, then prospective validation refuses the command before dry-run output or disk write", "optional enum fields with no default are omitted", "schema.id.next"},
		"docs/migrate.md":                         {"Content Priority", "lexical heading order", "+ ## Summary", "Sections are inserted at the end"},
		"docs/extensibility.md":                   {"without changing the core model", "YAML / JSON / TOML", "source: body.section[\"## Summary\"]"},
		"docs/describe.md":                        {"\"source\": \"body.section", "\"defined_in\": \"docs/.stem\""},
		"docs/explain.md":                         {"\"name\": \"notes\"", "\"source\": \"body.section", "\"defined_in\": \"docs/.stem\""},
		"docs/derivation.md":                      {"origin: derived", "origin: aggregate", "Physical schema provenance is reported as `defined_in`; the logical `source` field is used only for extraction directives"},
		".claude/skills/rootline/SKILL.md":        {"defined_in `.stem`", "source: body.section", "frontmatter override"},
		".claude/skills/rootline/ref-schema.md":   {"optional enum fields with no default are omitted", "Required enum fields without a default cause `rootline new` to refuse before dry-run output or disk write", "does not invent the first allowed value", "| Field | Type | Required | Values | Defined In | Source |"},
		".claude/skills/rootline/ref-query.md":    {"defined_in `.stem`", "source: body.section", "duplicate matching headings", "origin: frontmatter, derived, aggregate, schema"},
		".claude/skills/rootline/ref-validate.md": {"governance-root-relative", "empty section is present", "required means presence"},
		".claude/skills/rootline/ref-advanced.md": {"source: body.section", "lexical heading order"},
	}
	for path, fragments := range expect {
		text := string(documentationContractRead(t, repo, path))
		for _, fragment := range fragments {
			if !strings.Contains(text, fragment) {
				t.Errorf("%s missing behavior contract %q", path, fragment)
			}
		}
	}
	assertDocsExplainOrigins(t, repo)
	assertRefSchemaAuthoredNotesExample(t, repo)
	assertRefQueryUsesExactAggregateOrigin(t, repo)
}

func assertDocsExplainOrigins(t *testing.T, repo string) {
	t.Helper()
	foundExplain := false
	found := map[string]string{}
	for _, value := range jsonFenceValues(t, repo, "docs/explain.md") {
		obj, ok := value.(map[string]any)
		if !ok || obj["kind"] != "rootline/explain" {
			continue
		}
		foundExplain = true
		for _, field := range []string{"notes", "total"} {
			if origin, ok := fieldOrigin(obj, field); ok {
				found[field] = origin
			}
		}
	}
	if !foundExplain {
		t.Fatalf("docs/explain.md missing rootline/explain JSON payload")
	}
	for field, origin := range map[string]string{"notes": "derived", "total": "aggregate"} {
		got, ok := found[field]
		if !ok {
			t.Fatalf("docs/explain.md missing explain field %q with origin", field)
		}
		if got != origin {
			t.Fatalf("docs/explain.md %s origin = %q, want %q", field, got, origin)
		}
	}
}

func assertRefSchemaAuthoredNotesExample(t *testing.T, repo string) {
	t.Helper()
	text := string(documentationContractRead(t, repo, ".claude/skills/rootline/ref-schema.md"))
	for _, block := range fencedBlocks(text) {
		if block.lang != "yaml" && block.lang != "yml" {
			continue
		}
		var value map[string]any
		if err := yaml.Unmarshal([]byte(block.body), &value); err != nil {
			continue
		}
		notes, ok := value["notes"].(map[string]any)
		if !ok {
			continue
		}
		want := map[string]any{"type": "string", "required": true, "source": "body.section[\"## Notes\"]"}
		if reflect.DeepEqual(notes, want) {
			return
		}
		t.Fatalf("ref-schema notes authoring example = %#v, want %#v", notes, want)
	}
	t.Fatalf("ref-schema missing authored .stem YAML example for required notes source")
}

func assertRefQueryUsesExactAggregateOrigin(t *testing.T, repo string) {
	t.Helper()
	text := string(documentationContractRead(t, repo, ".claude/skills/rootline/ref-query.md"))
	if strings.Contains(text, "origin: frontmatter, derived, aggregated, schema") {
		t.Fatalf("ref-query uses stale explain origin spelling aggregated")
	}
}

func jsonFenceValues(t *testing.T, repo, path string) []any {
	t.Helper()
	var values []any
	for _, block := range fencedBlocks(string(documentationContractRead(t, repo, path))) {
		if block.lang == "json" {
			if value := assertJSONBlockHasCanonicalProvenance(t, path, block); value != nil {
				values = append(values, value)
			}
		}
	}
	return values
}

func fieldOrigin(obj map[string]any, name string) (string, bool) {
	fields, ok := obj["fields"].([]any)
	if !ok {
		return "", false
	}
	for _, item := range fields {
		field, ok := item.(map[string]any)
		if !ok || field["name"] != name {
			continue
		}
		origin, ok := field["origin"].(string)
		return origin, ok
	}
	return "", false
}

func assertStemSchemaExact(t *testing.T, repo, path string, want map[string]any) {
	t.Helper()
	schema := documentationContractMap(t, documentationContractSchema(t, repo, path)["schema"], path+" schema")
	assertExact(t, path+" schema", schema, want)
}

func assertExact(t *testing.T, name string, got, want any) {
	t.Helper()
	if !documentationExactEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
}

func assertScopeContainsAll(t *testing.T, name, scope string, fragments []string) {
	t.Helper()
	for _, fragment := range fragments {
		if !strings.Contains(scope, fragment) {
			t.Errorf("%s missing paired fragment %q", name, fragment)
		}
	}
}

func sectionFrom(text, heading string) string {
	start := strings.Index(text, heading)
	if start < 0 {
		return ""
	}
	section := text[start+len(heading):]
	if end := strings.Index(section, "\n## "); end >= 0 {
		section = section[:end]
	}
	return section
}

func paragraphFrom(text, needle string) string {
	start := strings.Index(text, needle)
	if start < 0 {
		return ""
	}
	paragraph := text[start:]
	if end := strings.Index(paragraph, "\n\n"); end >= 0 {
		paragraph = paragraph[:end]
	}
	return paragraph
}
