package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/rules"
)

func TestInitDryRun(t *testing.T) {
	dir := setupTestDir(t) // from commands_test.go
	out, err := runCmd(t, "init", dir, "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "version:") {
		t.Errorf("expected version: in output, got: %s", out)
	}
	if !strings.Contains(out, "schema:") {
		t.Errorf("expected schema: in output, got: %s", out)
	}
	if !strings.Contains(out, "estado:") {
		t.Errorf("expected estado field inferred, got: %s", out)
	}
}

func TestInitWritesFile(t *testing.T) {
	dir := t.TempDir()
	// Create markdown files (no .stem)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\nestado: draft\n---\n# A\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\nestado: done\n---\n# B\n"), 0644)

	out, err := runCmd(t, "init", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Created") {
		t.Errorf("expected 'Created' message, got: %s", out)
	}

	// Verify .stem was written
	stemPath := filepath.Join(dir, ".stem")
	content, err := os.ReadFile(stemPath)
	if err != nil {
		t.Fatalf("expected .stem file to exist: %v", err)
	}
	if !strings.Contains(string(content), "version:") {
		t.Errorf("expected valid .stem content, got: %s", string(content))
	}
}

func TestInitExistingStem(t *testing.T) {
	dir := setupTestDir(t) // has .stem already
	_, err := runCmd(t, "init", dir)
	if err == nil {
		t.Fatal("expected error for existing .stem")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}

func TestInitForce(t *testing.T) {
	dir := setupTestDir(t) // has .stem
	out, err := runCmd(t, "init", dir, "--force")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Created") {
		t.Errorf("expected 'Created' message, got: %s", out)
	}
}

func TestInitNoFiles(t *testing.T) {
	dir := t.TempDir()
	_, err := runCmd(t, "init", dir)
	if err == nil {
		t.Fatal("expected error for empty directory")
	}
	if !strings.Contains(err.Error(), "no markdown files") {
		t.Errorf("expected 'no markdown files' error, got: %v", err)
	}
}

func TestInitMixedContentWarning(t *testing.T) {
	dir := t.TempDir()
	// Create 3 files with frontmatter and 2 without (40% without > 20% threshold)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\nestado: draft\n---\n# A\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\nestado: done\n---\n# B\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "c.md"), []byte("---\nestado: review\n---\n# C\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "readme.md"), []byte("# No frontmatter\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "notes.md"), []byte("# Also no frontmatter\n"), 0644)

	out, err := runCmd(t, "init", dir, "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Warning") {
		t.Errorf("expected Warning for mixed content, got: %s", out)
	}
	if !strings.Contains(out, "2 of 5") {
		t.Errorf("expected '2 of 5' in warning, got: %s", out)
	}
	// Schema should still be generated
	if !strings.Contains(out, "version:") {
		t.Errorf("expected schema generated despite warning, got: %s", out)
	}
}

func TestInitCleanContentNoWarning(t *testing.T) {
	dir := t.TempDir()
	// All files have frontmatter
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\nestado: draft\n---\n# A\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\nestado: done\n---\n# B\n"), 0644)

	out, err := runCmd(t, "init", dir, "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, "Warning") {
		t.Errorf("expected no warning when all files have frontmatter, got: %s", out)
	}
}

func TestStemSerializer_PreservesSource(t *testing.T) {
	ordered := 1
	stem := &rules.StemFile{
		Version: 2,
		Root:    true,
		Schema: map[string]rules.SchemaField{
			"h1":     {Type: "string", Extract: `body.h1`},
			"notes":  {Type: "string", Extract: `body.section["## Notes: risks #1"]`, Required: true},
			"legacy": {Type: "string", Heading: "## Legacy", Ordered: &ordered},
		},
	}

	out, err := stemFileToYAML(stem, t.TempDir())
	if err != nil {
		t.Fatalf("stemFileToYAML: %v", err)
	}
	if !strings.Contains(out, "source: body.h1") {
		t.Fatalf("body.h1 source not serialized:\n%s", out)
	}
	if !strings.Contains(out, `source: 'body.section["## Notes: risks #1"]'`) {
		t.Fatalf("hazardous section source not safely serialized:\n%s", out)
	}
	if strings.Contains(out, "heading:") || strings.Contains(out, "ordered:") {
		t.Fatalf("canonical serializer emitted legacy keys:\n%s", out)
	}

	parsed, err := rules.ParseStem(".stem", []byte(out))
	if err != nil {
		t.Fatalf("serialized stem did not parse: %v\n%s", err, out)
	}
	if got := parsed.Schema["h1"].Extract; got != `body.h1` {
		t.Fatalf("h1 source=%q", got)
	}
	if got := parsed.Schema["notes"].Extract; got != `body.section["## Notes: risks #1"]` {
		t.Fatalf("notes source=%q", got)
	}
	if got := parsed.Schema["legacy"].Extract; got != "" {
		t.Fatalf("legacy source invented as %q", got)
	}
}

func TestInitSectionSourceCanonicalDryRunRoundTrips(t *testing.T) {
	tests := []struct {
		name     string
		heading  string
		field    string
		wantLine string
		wantSrc  string
	}{
		{"h1", "# Overview", "overview", `    source: body.section["# Overview"]`, `body.section["# Overview"]`},
		{"h2", "## Notes", "notes", `    source: body.section["## Notes"]`, `body.section["## Notes"]`},
		{"h3", "### Deep Dive", "deep_dive", `    source: body.section["### Deep Dive"]`, `body.section["### Deep Dive"]`},
		{"colon", "## Has: colon", "has_colon", `    source: 'body.section["## Has: colon"]'`, `body.section["## Has: colon"]`},
		{"comment", "## Hash # marker", "hash_marker", `    source: 'body.section["## Hash # marker"]'`, `body.section["## Hash # marker"]`},
		{"braces", "## Brace {value}", "brace_value", `    source: body.section["## Brace {value}"]`, `body.section["## Brace {value}"]`},
		{"brackets", "## Bracket [value]", "bracket_value", `    source: body.section["## Bracket [value]"]`, `body.section["## Bracket [value]"]`},
		{"quotes", `## Quote "double"`, "quote_double", `    source: body.section["## Quote \"double\""]`, `body.section["## Quote \"double\""]`},
		{"backslash", `## Backslash \ path`, "backslash_path", `    source: body.section["## Backslash \\ path"]`, `body.section["## Backslash \\ path"]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for i := 0; i < 4; i++ {
				body := fmt.Sprintf("---\ntitle: doc-%d\n---\n%s\nBody\n", i, tt.heading)
				mustWriteFile(t, filepath.Join(dir, fmt.Sprintf("with-section-%d.md", i)), []byte(body), 0644)
			}
			mustWriteFile(t, filepath.Join(dir, "without-section.md"), []byte("---\ntitle: other\n---\n# Different\n"), 0644)

			out, err := runCmd(t, "init", dir, "--dry-run")
			if err != nil {
				t.Fatalf("unexpected error: %v\noutput:\n%s", err, out)
			}
			if !strings.Contains(out, tt.wantLine) {
				t.Fatalf("init output source bytes mismatch; want line %q in:\n%s", tt.wantLine, out)
			}
			stem, err := rules.ParseStem(filepath.Join(dir, ".stem"), []byte(out))
			if err != nil {
				t.Fatalf("dry-run output did not parse with production parser: %v\n%s", err, out)
			}
			field := stem.Schema[tt.field]
			if field.Type != "string" || field.Required || field.Extract != tt.wantSrc || field.Heading != "" {
				t.Fatalf("parsed field = %+v, want optional string source %q without heading", field, tt.wantSrc)
			}
			if strings.Contains(out, "heading:") {
				t.Fatalf("init output should not serialize legacy heading:\n%s", out)
			}
		})
	}
}

func TestInitSectionSourceFieldCollisionLeavesNoPartialOutput(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 2; i++ {
		mustWriteFile(t, filepath.Join(dir, fmt.Sprintf("notes-%d.md", i)), []byte("---\nnotes:\n  - keep\n---\n## Notes\nBody\n"), 0644)
	}

	out, err := runCmd(t, "init", dir)
	if err == nil || !strings.Contains(err.Error(), `field "notes"`) || !strings.Contains(err.Error(), "frontmatter") || !strings.Contains(err.Error(), "body section") {
		t.Fatalf("expected frontmatter/body section collision error, got out=%q err=%v", out, err)
	}
	if strings.Contains(out, "version: 2") || strings.Contains(out, "Created") {
		t.Fatalf("expected no partial schema output, got %q", out)
	}
	if _, statErr := os.Stat(filepath.Join(dir, ".stem")); !os.IsNotExist(statErr) {
		t.Fatalf("expected no partial .stem, stat err=%v", statErr)
	}
}

func TestInitSectionSourceErrorLeavesNoPartialOutput(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "collision.md"), []byte("## Notes\nA\n\n### Notes\nB\n"), 0644)

	out, err := runCmd(t, "init", dir)
	if err == nil || !strings.Contains(err.Error(), "section field name collision") {
		t.Fatalf("expected collision error, got out=%q err=%v", out, err)
	}
	if strings.Contains(out, "version: 2") || strings.Contains(out, "Created") {
		t.Fatalf("expected no partial schema output, got %q", out)
	}
	if _, statErr := os.Stat(filepath.Join(dir, ".stem")); !os.IsNotExist(statErr) {
		t.Fatalf("expected no partial .stem, stat err=%v", statErr)
	}
}

func TestInitMixedBelowThresholdNoWarning(t *testing.T) {
	dir := t.TempDir()
	// 1 of 10 files without frontmatter = 10% < 20% threshold
	for i := 0; i < 9; i++ {
		mustWriteFile(t, filepath.Join(dir, fmt.Sprintf("f%d.md", i)),
			[]byte(fmt.Sprintf("---\nestado: s%d\n---\n# F%d\n", i, i)), 0644)
	}
	mustWriteFile(t, filepath.Join(dir, "readme.md"), []byte("# No frontmatter\n"), 0644)

	out, err := runCmd(t, "init", dir, "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, "Warning") {
		t.Errorf("expected no warning at 10%% ratio, got: %s", out)
	}
}

func TestInitTemplateInvalidRef(t *testing.T) {
	dir := t.TempDir()
	_, err := runCmd(t, "init", dir, "--template", "invalid")
	if err == nil {
		t.Fatal("expected error for invalid template ref")
	}
	if !strings.Contains(err.Error(), "invalid template ref") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInitTemplateDryRun(t *testing.T) {
	// Test that --template with a nonexistent repo returns an error.
	dir := t.TempDir()
	_, err := runCmd(t, "init", dir, "--template", "nonexistent-xyz/repo-xyz-12345")
	if err == nil {
		t.Fatal("expected error for nonexistent remote repo")
	}
}

func TestInitStructuralInference(t *testing.T) {
	dir := t.TempDir()
	// Create 4 subdirectories all with README.md to trigger structural inference.
	for _, name := range []string{"A", "B", "C", "D"} {
		subdir := filepath.Join(dir, name)
		_ = os.MkdirAll(subdir, 0755)
		mustWriteFile(t, filepath.Join(subdir, "README.md"),
			[]byte("---\nestado: draft\n---\n# "+name+"\n"), 0644)
	}

	out, err := runCmd(t, "init", dir, "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "structural:") {
		t.Errorf("expected structural: section in output, got: %s", out)
	}
	if !strings.Contains(out, "require_index:") {
		t.Errorf("expected require_index in output, got: %s", out)
	}
}

func TestInitFlatFallback(t *testing.T) {
	dir := t.TempDir()

	// Create flat directory without naming patterns.
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\nestado: draft\n---\n# A\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\nestado: done\n---\n# B\n"), 0644)

	out, err := runCmd(t, "init", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should fall back to flat mode.
	if !strings.Contains(out, "Created") {
		t.Errorf("expected 'Created' message, got: %s", out)
	}
	if strings.Contains(out, "levels detected") {
		t.Errorf("expected flat mode (no levels), got: %s", out)
	}

	// Single .stem file should exist.
	content, err := os.ReadFile(filepath.Join(dir, ".stem"))
	if err != nil {
		t.Fatalf("expected .stem file: %v", err)
	}
	if !strings.Contains(string(content), "version:") {
		t.Errorf("expected version:, got: %s", string(content))
	}
}

func TestInitAutoHierarchy(t *testing.T) {
	dir := t.TempDir()

	// Create a 2-level hierarchy: E##-*/F##-*
	for _, epic := range []string{"E01-infra", "E02-platform"} {
		epicDir := filepath.Join(dir, epic)
		_ = os.MkdirAll(epicDir, 0755)
		mustWriteFile(t, filepath.Join(epicDir, "README.md"),
			[]byte("---\nestado: Pending\n---\n# "+epic+"\n"), 0644)

		for _, feat := range []string{"F01-net", "F02-store"} {
			featDir := filepath.Join(epicDir, feat)
			_ = os.MkdirAll(featDir, 0755)
			mustWriteFile(t, filepath.Join(featDir, "README.md"),
				[]byte("---\nestado: Pending\ntipo: feature\n---\n# "+feat+"\n"), 0644)
		}
	}

	out, err := runCmd(t, "init", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should create single .stem with levels.
	if !strings.Contains(out, "levels") {
		t.Errorf("expected hierarchical output message, got: %s", out)
	}

	// Root .stem should exist with match-based schema (v2 format).
	rootStem := filepath.Join(dir, ".stem")
	content, err := os.ReadFile(rootStem)
	if err != nil {
		t.Fatalf("expected root .stem: %v", err)
	}
	s := string(content)
	if !strings.Contains(s, "version: 2") {
		t.Errorf("expected version: 2 in root .stem, got: %s", s)
	}
	if !strings.Contains(s, "schema:") {
		t.Errorf("expected schema: section in root .stem, got: %s", s)
	}
	if !strings.Contains(s, "match:") {
		t.Errorf("expected match: annotations in root .stem, got: %s", s)
	}
	if !strings.Contains(s, "estado:") {
		t.Errorf("expected estado field in root .stem, got: %s", s)
	}

	// No child .stem files should be created.
	childStem := filepath.Join(dir, "E01-infra", ".stem")
	if _, err := os.Stat(childStem); err == nil {
		t.Error("expected no child .stem in E01-infra/ (match-based schema is in root .stem)")
	}
}

func TestInitAutoHierarchyDryRun(t *testing.T) {
	dir := t.TempDir()

	for _, epic := range []string{"E01-a", "E02-b"} {
		epicDir := filepath.Join(dir, epic)
		_ = os.MkdirAll(epicDir, 0755)
		mustWriteFile(t, filepath.Join(epicDir, "README.md"),
			[]byte("---\nestado: Pending\n---\n# Epic\n"), 0644)

		for _, feat := range []string{"F01-x", "F02-y"} {
			featDir := filepath.Join(epicDir, feat)
			_ = os.MkdirAll(featDir, 0755)
			mustWriteFile(t, filepath.Join(featDir, "README.md"),
				[]byte("---\nestado: Pending\n---\n# Feature\n"), 0644)
		}
	}

	out, err := runCmd(t, "init", dir, "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should show single .stem with match-based schema (v2 format).
	if !strings.Contains(out, "# --- ") {
		t.Errorf("expected file separator in dry-run output, got: %s", out)
	}
	if !strings.Contains(out, "version: 2") {
		t.Errorf("expected version: 2 in dry-run output, got: %s", out)
	}
	if !strings.Contains(out, "schema:") {
		t.Errorf("expected schema: section in dry-run output, got: %s", out)
	}
	if !strings.Contains(out, "match:") {
		t.Errorf("expected match: annotations in dry-run output, got: %s", out)
	}

	// No files should be written.
	if _, err := os.Stat(filepath.Join(dir, ".stem")); err == nil {
		t.Error("expected no .stem file written in dry-run mode")
	}
}

func TestInitAutoHierarchy_GeneratesAggregate(t *testing.T) {
	dir := t.TempDir()

	estados := []string{"Pending", "In Progress", "Completed"}
	for i, epic := range []string{"E01-infra", "E02-platform"} {
		epicDir := filepath.Join(dir, epic)
		_ = os.MkdirAll(epicDir, 0755)
		mustWriteFile(t, filepath.Join(epicDir, "README.md"),
			[]byte("---\nestado: "+estados[i]+"\n---\n# "+epic+"\n"), 0644)

		for j, feat := range []string{"F01-net", "F02-store"} {
			featDir := filepath.Join(epicDir, feat)
			_ = os.MkdirAll(featDir, 0755)
			mustWriteFile(t, filepath.Join(featDir, "README.md"),
				[]byte("---\nestado: "+estados[j]+"\n---\n# "+feat+"\n"), 0644)
		}
	}

	out, err := runCmd(t, "init", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Note: auto-generated aggregate for 'estado'") {
		t.Errorf("expected aggregate note in output, got: %s", out)
	}

	rootStem := filepath.Join(dir, ".stem")
	content, err := os.ReadFile(rootStem)
	if err != nil {
		t.Fatalf("expected root .stem: %v", err)
	}
	s := string(content)
	if !strings.Contains(s, "aggregate:") {
		t.Errorf("expected aggregate: section in root .stem, got:\n%s", s)
	}
	if !strings.Contains(s, "estado:") {
		t.Errorf("expected estado field in aggregate section, got:\n%s", s)
	}
	// Field-agnostic: uses first value as default
	if !strings.Contains(s, `"In Progress"`) && !strings.Contains(s, `"Pending"`) {
		t.Errorf("expected first value as default, got:\n%s", s)
	}
}

func TestInitEmitsSectionFields(t *testing.T) {
	dir := t.TempDir()

	// 5 documents: all have "## Context", 4/5 have "## Notes"
	docs := []string{
		"---\ntitle: doc1\n---\n## Context\ncontent\n## Notes\nnotes here\n",
		"---\ntitle: doc2\n---\n## Context\ncontent\n## Notes\nnotes here\n",
		"---\ntitle: doc3\n---\n## Context\ncontent\n## Notes\nnotes here\n",
		"---\ntitle: doc4\n---\n## Context\ncontent\n## Notes\nnotes here\n",
		"---\ntitle: doc5\n---\n## Context\ncontent\n",
	}
	for i, content := range docs {
		mustWriteFile(t, filepath.Join(dir, fmt.Sprintf("doc%d.md", i+1)), []byte(content), 0644)
	}

	out, err := runCmd(t, "init", dir, "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// "## Context" appears in 5/5 = 100% → required_section at 0.80 threshold
	if !strings.Contains(out, "type: string") {
		t.Errorf("expected 'type: string' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "context:") {
		t.Errorf("expected 'context:' field name in output, got:\n%s", out)
	}
	if !strings.Contains(out, `source: body.section["## Context"]`) {
		t.Errorf("expected canonical Context source in output, got:\n%s", out)
	}
	if !strings.Contains(out, "required: true") {
		t.Errorf("expected 'required: true' for context section, got:\n%s", out)
	}

	// "## Notes" appears in 4/5 = 80% → exactly at 0.80 threshold → optional_section
	if !strings.Contains(out, "notes:") {
		t.Errorf("expected 'notes:' field name in output, got:\n%s", out)
	}
	if !strings.Contains(out, `source: body.section["## Notes"]`) {
		t.Errorf("expected canonical Notes source in output, got:\n%s", out)
	}
	if strings.Contains(out, "notes:\n    type: string\n    required: true") {
		t.Errorf("expected Notes to remain optional, got:\n%s", out)
	}
	if strings.Contains(out, "heading:") {
		t.Errorf("expected no legacy heading serialization, got:\n%s", out)
	}
}

func TestInitSectionFieldsWrittenToFile(t *testing.T) {
	dir := t.TempDir()

	// 5 documents all with "## Context"
	for i := 1; i <= 5; i++ {
		content := fmt.Sprintf("---\ntitle: doc%d\n---\n## Context\ncontent here\n", i)
		mustWriteFile(t, filepath.Join(dir, fmt.Sprintf("doc%d.md", i)), []byte(content), 0644)
	}

	_, err := runCmd(t, "init", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stemPath := filepath.Join(dir, ".stem")
	content, err := os.ReadFile(stemPath)
	if err != nil {
		t.Fatalf("expected .stem file: %v", err)
	}
	s := string(content)

	if !strings.Contains(s, "type: string") {
		t.Errorf("expected 'type: string' in .stem, got:\n%s", s)
	}
	if !strings.Contains(s, "context:") {
		t.Errorf("expected 'context:' field in .stem, got:\n%s", s)
	}
	if !strings.Contains(s, `source: body.section["## Context"]`) {
		t.Errorf("expected canonical source in .stem, got:\n%s", s)
	}
	if strings.Contains(s, "heading:") {
		t.Errorf("expected no legacy heading in .stem, got:\n%s", s)
	}
}

func TestInitSectionThresholdStrict(t *testing.T) {
	dir := t.TempDir()

	// 5 documents: "## Rare" appears in only 3/5 = 60% → below 0.80, should NOT appear
	docs := []string{
		"---\ntitle: doc1\n---\n## Rare\ncontent\n",
		"---\ntitle: doc2\n---\n## Rare\ncontent\n",
		"---\ntitle: doc3\n---\n## Rare\ncontent\n",
		"---\ntitle: doc4\n---\n# Just title\n",
		"---\ntitle: doc5\n---\n# Just title\n",
	}
	for i, content := range docs {
		mustWriteFile(t, filepath.Join(dir, fmt.Sprintf("doc%d.md", i+1)), []byte(content), 0644)
	}

	out, err := runCmd(t, "init", dir, "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// At 0.80 threshold, 3/5 = 60% should NOT be emitted as a section field
	if strings.Contains(out, "rare:") {
		t.Errorf("expected 'rare:' NOT in output at 0.80 threshold (60%%), got:\n%s", out)
	}
}

func TestInitAutoHierarchy_NoAggregateForNonEnum(t *testing.T) {
	dir := t.TempDir()

	for _, epic := range []string{"E01-a", "E02-b"} {
		epicDir := filepath.Join(dir, epic)
		_ = os.MkdirAll(epicDir, 0755)
		mustWriteFile(t, filepath.Join(epicDir, "README.md"),
			[]byte("---\ntitle: something\n---\n# "+epic+"\n"), 0644)

		for _, feat := range []string{"F01-x", "F02-y"} {
			featDir := filepath.Join(epicDir, feat)
			_ = os.MkdirAll(featDir, 0755)
			mustWriteFile(t, filepath.Join(featDir, "README.md"),
				[]byte("---\ntitle: other\n---\n# "+feat+"\n"), 0644)
		}
	}

	out, err := runCmd(t, "init", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(out, "Note: auto-generated aggregate") {
		t.Errorf("expected no aggregate note for non-enum fields, got: %s", out)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".stem"))
	if err != nil {
		t.Fatalf("expected root .stem: %v", err)
	}
	if strings.Contains(string(content), "aggregate:") {
		t.Errorf("expected no aggregate section for non-enum fields, got:\n%s", string(content))
	}
}

func TestInitMarkerOutput(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\nestado: draft\n---\n# A\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\nestado: done\n---\n# B\n"), 0644)

	out, err := runCmd(t, "init", dir, "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(out, "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got: %s", out)
	}

	// Version should be on first line
	if !strings.HasPrefix(lines[0], "version:") {
		t.Errorf("expected version on line 1, got: %s", lines[0])
	}

	// Root should be on second line
	if !strings.HasPrefix(lines[1], "root:") || !strings.Contains(lines[1], "true") {
		t.Errorf("expected 'root: true' on line 2, got: %s", lines[1])
	}

	// Also verify it's in the output
	if !strings.Contains(out, "root: true") {
		t.Errorf("expected 'root: true' in output, got: %s", out)
	}
}

func TestInitHierarchicalMarkerRootOnly(t *testing.T) {
	dir := t.TempDir()

	// Create a 2-level hierarchy
	for _, epic := range []string{"E01-infra", "E02-platform"} {
		epicDir := filepath.Join(dir, epic)
		_ = os.MkdirAll(epicDir, 0755)
		mustWriteFile(t, filepath.Join(epicDir, "README.md"),
			[]byte("---\nestado: Pending\n---\n# "+epic+"\n"), 0644)

		for _, feat := range []string{"F01-net", "F02-store"} {
			featDir := filepath.Join(epicDir, feat)
			_ = os.MkdirAll(featDir, 0755)
			mustWriteFile(t, filepath.Join(featDir, "README.md"),
				[]byte("---\nestado: Pending\ntipo: feature\n---\n# "+feat+"\n"), 0644)
		}
	}

	out, err := runCmd(t, "init", dir, "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find the first .stem in the output (the root-level one)
	if !strings.Contains(out, "root: true") {
		t.Errorf("expected 'root: true' in hierarchical init output, got: %s", out)
	}

	// Root marker should appear exactly once in the output
	count := strings.Count(out, "root: true")
	if count != 1 {
		t.Errorf("expected 'root: true' to appear exactly once (only at root level), got %d times in: %s", count, out)
	}
}
