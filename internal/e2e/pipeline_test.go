// Package e2e contains integration tests that exercise the full
// pipeline across packages: rules → index → extract → validate.
package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/query"
	"github.com/pablontiv/rootline/internal/rules"
)

// setupProject creates a temporary directory tree simulating a real
// rootline project. It creates a .git marker, files, and returns
// the root path.
func setupProject(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()

	// Create .git marker so WalkUp has a boundary.
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	for relPath, content := range files {
		absPath := filepath.Join(root, relPath)
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// buildScopeResolver returns a ScopeResolver that uses WalkUp + Merge
// to compute the effective stem for each directory.
func buildScopeResolver() index.ScopeResolver {
	return func(dir string) *rules.StemFile {
		entries, err := rules.WalkUp(dir)
		if err != nil {
			return nil
		}
		return rules.MergeStemFiles(entries)
	}
}

// recordPaths extracts sorted paths from records for easy assertion.
func recordPaths(records []*extract.Record) []string {
	paths := make([]string, len(records))
	for i, r := range records {
		paths[i] = r.Path
	}
	sort.Strings(paths)
	return paths
}

// TestPipeline_ScanExtractWithStemScope tests the full pipeline:
// .stem files define scope → scanner filters by scope → extractor
// produces records with correct frontmatter.
func TestPipeline_ScanExtractWithStemScope(t *testing.T) {
	root := setupProject(t, map[string]string{
		// Root .stem: scope only *.md files
		".stem": "version: 1\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n    required: true\n",

		// Markdown files at root
		"readme.md":  "---\ntitle: Root Readme\n---\n# Welcome",
		"notes.md":   "---\ntitle: Notes\n---\n# Notes",
		"config.yml": "key: value",

		// Subdirectory with its own .stem narrowing scope
		"docs/.stem":        "scope:\n  match: \"*.md\"\nschema:\n  status:\n    type: string\n",
		"docs/guide.md":     "---\ntitle: Guide\nstatus: draft\n---\n# Guide",
		"docs/api.md":       "---\ntitle: API Reference\nstatus: published\n---\n# API",
		"docs/changelog.md": "---\ntitle: Changelog\n---\n# Changes",
	})

	reg := extract.NewRegistry()
	resolver := buildScopeResolver()

	records, err := index.Scan(root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	// All .md files should be found (scope is *.md at both levels).
	paths := recordPaths(records)
	expected := []string{
		"docs/api.md",
		"docs/changelog.md",
		"docs/guide.md",
		"notes.md",
		"readme.md",
	}
	if len(paths) != len(expected) {
		t.Fatalf("got paths %v, want %v", paths, expected)
	}
	for i, p := range expected {
		if paths[i] != p {
			t.Errorf("path[%d] = %q, want %q", i, paths[i], p)
		}
	}
}

// TestPipeline_ScopeNarrowsInSubdirectory tests that a child .stem
// can narrow the scope so only specific files are extracted.
func TestPipeline_ScopeNarrowsInSubdirectory(t *testing.T) {
	root := setupProject(t, map[string]string{
		// Root: match all markdown
		".stem": "version: 1\nscope:\n  match: \"*.md\"\n",

		"readme.md": "---\ntitle: Root\n---\n",

		// tasks/ narrows scope to T*.md only
		"tasks/.stem":       "scope:\n  match: \"T*.md\"\n",
		"tasks/T001-foo.md": "---\ntitle: Task 1\n---\n",
		"tasks/T002-bar.md": "---\ntitle: Task 2\n---\n",
		"tasks/README.md":   "---\ntitle: Tasks Index\n---\n",
	})

	reg := extract.NewRegistry()
	resolver := buildScopeResolver()

	records, err := index.Scan(root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	paths := recordPaths(records)
	// Root: readme.md matches *.md
	// tasks/: only T001-foo.md and T002-bar.md match T*.md
	// tasks/README.md does NOT match T*.md
	expected := []string{
		"readme.md",
		"tasks/T001-foo.md",
		"tasks/T002-bar.md",
	}
	if len(paths) != len(expected) {
		t.Fatalf("got paths %v, want %v", paths, expected)
	}
	for i, p := range expected {
		if paths[i] != p {
			t.Errorf("path[%d] = %q, want %q", i, paths[i], p)
		}
	}
}

// TestPipeline_StemignoreAndScopeCombined tests that .stemignore
// exclusions and scope filtering work together correctly.
func TestPipeline_StemignoreAndScopeCombined(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem":       "version: 1\nscope:\n  match: \"*.md\"\n",
		".stemignore": "draft-*.md\n",

		"published.md":  "---\ntitle: Published\n---\n",
		"draft-idea.md": "---\ntitle: Draft\n---\n",
		"notes.txt":     "plain text",
	})

	reg := extract.NewRegistry()
	resolver := buildScopeResolver()

	records, err := index.Scan(root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	// published.md: passes scope (*.md) and not in .stemignore
	// draft-idea.md: passes scope but excluded by .stemignore
	// notes.txt: no extractor registered
	if len(records) != 1 {
		paths := recordPaths(records)
		t.Fatalf("got %d records %v, want 1 [published.md]", len(records), paths)
	}
	if records[0].Frontmatter["title"] != "Published" {
		t.Errorf("title = %v, want Published", records[0].Frontmatter["title"])
	}
}

// TestPipeline_StemMergeInheritance tests that schema fields from
// parent .stem files are inherited by child directories via merge.
func TestPipeline_StemMergeInheritance(t *testing.T) {
	root := setupProject(t, map[string]string{
		// Root defines base schema
		".stem": "version: 1\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n    required: true\n  owner:\n    type: string\n",

		// Child adds status field, inherits title + owner
		"docs/.stem": "schema:\n  status:\n    type: string\n    values:\n      - draft\n      - published\n",

		"docs/guide.md": "---\ntitle: Guide\nstatus: draft\nowner: team-a\n---\n# Guide",
	})

	// Verify stem merge works correctly for docs/ directory
	entries, err := rules.WalkUp(filepath.Join(root, "docs"))
	if err != nil {
		t.Fatalf("WalkUp error: %v", err)
	}

	effective := rules.MergeStemFiles(entries)

	// Should have all 3 fields: title (root), owner (root), status (child)
	if len(effective.Schema) != 3 {
		t.Fatalf("schema has %d fields, want 3: %v", len(effective.Schema), effective.Schema)
	}
	if _, ok := effective.Schema["title"]; !ok {
		t.Error("missing inherited 'title' field")
	}
	if _, ok := effective.Schema["owner"]; !ok {
		t.Error("missing inherited 'owner' field")
	}
	if sf, ok := effective.Schema["status"]; !ok {
		t.Error("missing child 'status' field")
	} else if len(sf.Values) != 2 {
		t.Errorf("status.values = %v, want [draft published]", sf.Values)
	}

	// Verify scope inherited from root
	if effective.Scope.Match != "*.md" {
		t.Errorf("scope.match = %q, want *.md", effective.Scope.Match)
	}

	// Verify extraction still works with the merged scope
	reg := extract.NewRegistry()
	resolver := buildScopeResolver()

	records, err := index.Scan(root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}
	rec := records[0]
	if rec.Frontmatter["title"] != "Guide" {
		t.Errorf("title = %v, want Guide", rec.Frontmatter["title"])
	}
	if rec.Frontmatter["status"] != "draft" {
		t.Errorf("status = %v, want draft", rec.Frontmatter["status"])
	}
}

// TestPipeline_NoStemMatchesEverything tests that without any .stem
// files, the scanner extracts all files with registered extractors.
func TestPipeline_NoStemMatchesEverything(t *testing.T) {
	root := setupProject(t, map[string]string{
		"a.md": "---\ntitle: A\n---\n",
		"b.md": "---\ntitle: B\n---\n",
	})

	reg := extract.NewRegistry()
	resolver := buildScopeResolver()

	records, err := index.Scan(root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	// No .stem = no scope = match all. Both .md files should be extracted.
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}
}

// TestPipeline_DeepNesting tests a 3-level deep directory with stem
// inheritance at each level.
func TestPipeline_DeepNesting(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem":                   "version: 1\nscope:\n  match: \"*.md\"\nschema:\n  project:\n    type: string\n",
		"epics/.stem":             "schema:\n  epic:\n    type: string\n",
		"epics/features/.stem":    "schema:\n  feature:\n    type: string\n",
		"epics/features/task.md":  "---\ntitle: Task\nproject: rootline\nepic: E01\nfeature: F01\n---\n# Task",
		"epics/features/notes.md": "---\ntitle: Notes\n---\n# Notes",
	})

	// Check full merge at deepest level
	entries, err := rules.WalkUp(filepath.Join(root, "epics", "features"))
	if err != nil {
		t.Fatalf("WalkUp error: %v", err)
	}

	effective := rules.MergeStemFiles(entries)

	// Should have project (root) + epic (epics/) + feature (features/)
	if len(effective.Schema) != 3 {
		t.Fatalf("schema has %d fields, want 3", len(effective.Schema))
	}

	// Scan should find both .md files
	reg := extract.NewRegistry()
	resolver := buildScopeResolver()

	records, err := index.Scan(root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	paths := recordPaths(records)
	expected := []string{
		"epics/features/notes.md",
		"epics/features/task.md",
	}
	if len(paths) != len(expected) {
		t.Fatalf("got paths %v, want %v", paths, expected)
	}
	for i, p := range expected {
		if paths[i] != p {
			t.Errorf("path[%d] = %q, want %q", i, paths[i], p)
		}
	}

	// Verify frontmatter extraction is correct
	for _, rec := range records {
		if rec.Path == "epics/features/task.md" {
			if rec.Frontmatter["project"] != "rootline" {
				t.Errorf("project = %v, want rootline", rec.Frontmatter["project"])
			}
			if rec.Frontmatter["epic"] != "E01" {
				t.Errorf("epic = %v, want E01", rec.Frontmatter["epic"])
			}
		}
	}
}

// TestPipeline_FrontmatterExtractionIntegrity verifies that extracted
// records preserve all frontmatter types correctly through the pipeline.
func TestPipeline_FrontmatterExtractionIntegrity(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": "version: 1\nscope:\n  match: \"*.md\"\n",
		"doc.md": "---\ntitle: Test Doc\nstatus: draft\npriority: 1\ntags:\n  - go\n  - testing\n---\n# Body\n\nParagraph.",
	})

	reg := extract.NewRegistry()
	resolver := buildScopeResolver()

	records, err := index.Scan(root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}

	rec := records[0]

	// String fields
	if rec.Frontmatter["title"] != "Test Doc" {
		t.Errorf("title = %v", rec.Frontmatter["title"])
	}
	if rec.Frontmatter["status"] != "draft" {
		t.Errorf("status = %v", rec.Frontmatter["status"])
	}

	// Integer field
	if rec.Frontmatter["priority"] != 1 {
		t.Errorf("priority = %v (type %T), want 1", rec.Frontmatter["priority"], rec.Frontmatter["priority"])
	}

	// Array field
	tags, ok := rec.Frontmatter["tags"].([]any)
	if !ok {
		t.Fatalf("tags type = %T, want []any", rec.Frontmatter["tags"])
	}
	if len(tags) != 2 || tags[0] != "go" || tags[1] != "testing" {
		t.Errorf("tags = %v, want [go testing]", tags)
	}

	// Body
	if rec.Body != "# Body\n\nParagraph." {
		t.Errorf("body = %q", rec.Body)
	}

	// Record type
	if rec.Type != "markdown" {
		t.Errorf("type = %q, want markdown", rec.Type)
	}
}

// TestPipeline_ValidateAfterExtraction exercises the full pipeline:
// scan → extract → resolve effective stem → validate.
// Simulates a real project with PRD documents.
func TestPipeline_ValidateAfterExtraction(t *testing.T) {
	root := setupProject(t, map[string]string{
		// Root .stem: Fecha required for all docs
		".stem": `version: 1
scope:
  match: "*.md"
schema:
  Fecha:
    type: string
    required: true
`,
		// PRD .stem: adds Estado enum + conditional requires
		"prd/.stem": `schema:
  Estado:
    type: enum
    values:
      - Pending
      - Completado
    required: true
  Tipo:
    type: enum
    values:
      - servicio-docker
      - modulo-sistema
    required: true
validate:
  - rule: requires
    if: { Estado: Completado }
    then: { fields: [Fecha] }
  - rule: non_empty
    field: Tipo
`,
		// Valid PRD document
		"prd/valid.md": "---\nFecha: \"2026-01-15\"\nEstado: Pending\nTipo: servicio-docker\n---\n# Valid PRD",

		// Invalid PRD: bad Estado enum value
		"prd/bad-enum.md": "---\nFecha: \"2026-01-15\"\nEstado: Invalid\nTipo: modulo-sistema\n---\n# Bad Enum",

		// Invalid PRD: missing required Fecha
		"prd/missing-required.md": "---\nEstado: Pending\nTipo: servicio-docker\n---\n# Missing Required",

		// Invalid PRD: Estado=Completado but no Fecha (requires rule fires)
		"prd/requires-fail.md": "---\nEstado: Completado\nTipo: modulo-sistema\n---\n# Requires Fail",
	})

	reg := extract.NewRegistry()
	resolver := buildScopeResolver()

	records, err := index.Scan(root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	// Resolve effective stem for prd/ directory
	entries, err := rules.WalkUp(filepath.Join(root, "prd"))
	if err != nil {
		t.Fatalf("WalkUp error: %v", err)
	}
	effective := rules.MergeStemFiles(entries)

	// Validate each record
	results := make(map[string][]rules.ValidationError)
	for _, rec := range records {
		errs := rules.Validate(rec, effective)
		results[rec.Path] = errs
	}

	// valid.md: should have 0 errors
	if errs := results["prd/valid.md"]; len(errs) != 0 {
		t.Errorf("valid.md: got %d errors, want 0: %v", len(errs), errs)
	}

	// bad-enum.md: Estado=Invalid should fail enum check
	if errs := results["prd/bad-enum.md"]; len(errs) == 0 {
		t.Error("bad-enum.md: expected enum error for Estado=Invalid")
	} else {
		foundEnum := false
		for _, e := range errs {
			if e.Rule == "enum" && e.Field == "Estado" {
				foundEnum = true
			}
		}
		if !foundEnum {
			t.Errorf("bad-enum.md: expected enum error for Estado, got: %v", errs)
		}
	}

	// missing-required.md: Fecha missing
	if errs := results["prd/missing-required.md"]; len(errs) == 0 {
		t.Error("missing-required.md: expected required error for Fecha")
	} else {
		foundRequired := false
		for _, e := range errs {
			if e.Rule == "required" && e.Field == "Fecha" {
				foundRequired = true
			}
		}
		if !foundRequired {
			t.Errorf("missing-required.md: expected required error for Fecha, got: %v", errs)
		}
	}

	// requires-fail.md: Estado=Completado without Fecha triggers requires rule
	if errs := results["prd/requires-fail.md"]; len(errs) == 0 {
		t.Error("requires-fail.md: expected requires error for Fecha")
	} else {
		foundRequires := false
		foundRequired := false
		for _, e := range errs {
			if e.Rule == "requires" && e.Field == "Fecha" {
				foundRequires = true
			}
			if e.Rule == "required" && e.Field == "Fecha" {
				foundRequired = true
			}
		}
		if !foundRequires {
			t.Errorf("requires-fail.md: expected requires error, got: %v", errs)
		}
		if !foundRequired {
			t.Errorf("requires-fail.md: expected required error for Fecha, got: %v", errs)
		}
	}
}

// TestPipeline_BatchValidationOutput exercises the full pipeline through
// to JSON output: scan → extract → validate → BatchValidationResult.
func TestPipeline_BatchValidationOutput(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": `version: 1
scope:
  match: "*.md"
schema:
  title:
    type: string
    required: true
  status:
    type: enum
    values: [draft, published]
`,
		"valid.md":   "---\ntitle: Good Doc\nstatus: draft\n---\n# OK",
		"invalid.md": "---\nstatus: unknown\n---\n# Bad",
	})

	reg := extract.NewRegistry()
	resolver := buildScopeResolver()

	records, err := index.Scan(root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	entries, err := rules.WalkUp(root)
	if err != nil {
		t.Fatalf("WalkUp error: %v", err)
	}
	effective := rules.MergeStemFiles(entries)

	// Build ValidationResults and batch
	var validationResults []*rules.ValidationResult
	for _, rec := range records {
		errs := rules.Validate(rec, effective)
		validationResults = append(validationResults, rules.NewValidationResult(rec.Path, errs))
	}

	batch := rules.NewBatchValidationResult(validationResults)

	// Verify batch structure
	if batch.Summary.Total != 2 {
		t.Fatalf("total = %d, want 2", batch.Summary.Total)
	}
	if batch.Summary.Valid != 1 {
		t.Errorf("valid = %d, want 1", batch.Summary.Valid)
	}
	if batch.Summary.Invalid != 1 {
		t.Errorf("invalid = %d, want 1", batch.Summary.Invalid)
	}

	// Verify JSON roundtrip
	data, err := batch.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}
	if parsed["version"].(float64) != 1 {
		t.Errorf("version = %v", parsed["version"])
	}
	if parsed["kind"] != "rootline/validate-batch" {
		t.Errorf("kind = %v", parsed["kind"])
	}
}

// TestPipeline_DescribeEffectiveSchema exercises the describe pipeline:
// WalkUp → Merge → DescribeResult with source tracking and applies.
func TestPipeline_DescribeEffectiveSchema(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": `version: 1
scope:
  match: "*.md"
schema:
  Fecha:
    type: string
    required: true
`,
		"prd/.stem": `schema:
  Estado:
    type: enum
    values:
      - Pending
      - Completado
    required: true
validate:
  - rule: requires
    if: { Estado: Completado }
    then: { fields: [Fecha] }
`,
	})

	// Walk-up from prd/
	entries, err := rules.WalkUp(filepath.Join(root, "prd"))
	if err != nil {
		t.Fatalf("WalkUp error: %v", err)
	}
	effective := rules.MergeStemFiles(entries)
	result := rules.NewDescribeResult("prd/", entries, effective)

	// Verify contract
	if result.Version != 1 {
		t.Errorf("version = %d", result.Version)
	}
	if result.Kind != "rootline/describe" {
		t.Errorf("kind = %q", result.Kind)
	}

	// Applies: root/.stem then prd/.stem
	if len(result.Applies) != 2 {
		t.Fatalf("applies has %d entries, want 2", len(result.Applies))
	}

	// Schema: Fecha (from root) + Estado (from prd)
	if len(result.Schema) != 2 {
		t.Fatalf("schema has %d fields, want 2", len(result.Schema))
	}

	fecha, ok := result.Schema["Fecha"]
	if !ok {
		t.Fatal("missing Fecha in schema")
	}
	if !fecha.Required {
		t.Error("Fecha should be required")
	}

	estado, ok := result.Schema["Estado"]
	if !ok {
		t.Fatal("missing Estado in schema")
	}
	if len(estado.Values) != 2 {
		t.Errorf("Estado.values = %v", estado.Values)
	}

	// Source tracking
	if fecha.Source == "" {
		t.Error("Fecha.source should be set")
	}
	if estado.Source == "" {
		t.Error("Estado.source should be set")
	}
	if fecha.Source == estado.Source {
		t.Error("Fecha and Estado should come from different .stem files")
	}

	// Validate rules preserved
	if len(result.Validate) != 1 {
		t.Fatalf("validate has %d rules, want 1", len(result.Validate))
	}
	if result.Validate[0].Rule != "requires" {
		t.Errorf("validate[0].rule = %q", result.Validate[0].Rule)
	}

	// Scope inherited from root
	if result.Scope.Match != "*.md" {
		t.Errorf("scope.match = %q", result.Scope.Match)
	}

	// JSON roundtrip
	data, err := result.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON error: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}
	if parsed["kind"] != "rootline/describe" {
		t.Errorf("JSON kind = %v", parsed["kind"])
	}
}

// TestPipeline_QueryAfterScanExtract exercises the full pipeline:
// scan → extract → query with operators.
func TestPipeline_QueryAfterScanExtract(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": "version: 1\nscope:\n  match: \"*.md\"\n",
		"tasks/T001.md": "---\ntitle: Deploy Redis\nestado: Pending\ntipo: servicio-docker\n---\n# Deploy",
		"tasks/T002.md": "---\ntitle: Auth Module\nestado: Completado\ntipo: modulo-sistema\n---\n# Auth",
		"tasks/T003.md": "---\ntitle: LXC Setup\nestado: Pending\ntipo: lxc\n---\n# LXC\n\nMigration needed.",
	})

	reg := extract.NewRegistry()
	resolver := buildScopeResolver()

	records, err := index.Scan(root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	// Query: estado eq Pending
	result, err := query.Execute(records, &query.Query{
		Where: &query.Condition{Op: query.OpEq, Field: "estado", Value: "Pending"},
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	qr := result.(*query.QueryResult)
	if qr.Meta.Count != 2 {
		t.Errorf("eq Pending: count = %d, want 2", qr.Meta.Count)
	}

	// Query: tipo in [lxc, vm]
	result2, _ := query.Execute(records, &query.Query{
		Where: &query.Condition{Op: query.OpIn, Field: "tipo", Values: []string{"lxc", "vm"}},
	})
	qr2 := result2.(*query.QueryResult)
	if qr2.Meta.Count != 1 {
		t.Errorf("in [lxc,vm]: count = %d, want 1", qr2.Meta.Count)
	}

	// Query: body contains "Migration"
	result3, _ := query.Execute(records, &query.Query{
		Where: &query.Condition{Op: query.OpContains, Field: "body", Value: "Migration"},
	})
	qr3 := result3.(*query.QueryResult)
	if qr3.Meta.Count != 1 {
		t.Errorf("body contains Migration: count = %d, want 1", qr3.Meta.Count)
	}

	// Count query
	result4, _ := query.Execute(records, &query.Query{
		Where: &query.Condition{Op: query.OpEq, Field: "estado", Value: "Completado"},
		Count: true,
	})
	cr := result4.(*query.CountResult)
	if cr.Count != 1 {
		t.Errorf("count Completado: %d, want 1", cr.Count)
	}

	// And query: tipo eq servicio-docker AND estado eq Pending
	result5, _ := query.Execute(records, &query.Query{
		Where: &query.Condition{
			Op: query.OpAnd,
			Children: []query.Condition{
				{Op: query.OpEq, Field: "tipo", Value: "servicio-docker"},
				{Op: query.OpEq, Field: "estado", Value: "Pending"},
			},
		},
	})
	qr5 := result5.(*query.QueryResult)
	if qr5.Meta.Count != 1 {
		t.Errorf("and(tipo=servicio-docker, estado=Pending): count = %d, want 1", qr5.Meta.Count)
	}
}
