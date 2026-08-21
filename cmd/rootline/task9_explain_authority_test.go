package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTask9ExplainAggregateTargetDuplicateSourceIsBuilderOwned(t *testing.T) {
	dir := t.TempDir()
	writeTask9ExplainAuthorityStem(t, dir)
	mustWriteFile(t, filepath.Join(dir, "README.md"), []byte("# Root\n\n## Status\n\nfirst\n\n## Status\n\nsecond\n"), 0o644)
	mustMkdir(t, filepath.Join(dir, "child"))
	mustWriteFile(t, filepath.Join(dir, "child", "README.md"), []byte("# Child\n\n## Status\n\ndone\n"), 0o644)
	declareTestBoundary(t, dir)

	stdout, err := runTask9CmdStdoutOnly(t, "explain", filepath.Join(dir, "README.md"))
	assertTask9ExplainBuilderOwnedAmbiguity(t, stdout, err)
}

func TestTask9ExplainAggregateNonTargetDuplicateSourceIsEnrichmentOwned(t *testing.T) {
	dir := t.TempDir()
	writeTask9ExplainAuthorityStem(t, dir)
	mustWriteFile(t, filepath.Join(dir, "README.md"), []byte("# Root\n"), 0o644)
	mustMkdir(t, filepath.Join(dir, "child"))
	mustWriteFile(t, filepath.Join(dir, "child", "README.md"), []byte("# Child\n\n## Status\n\nfirst\n\n## Status\n\nsecond\n"), 0o644)
	declareTestBoundary(t, dir)

	stdout, err := runTask9CmdStdoutOnly(t, "explain", filepath.Join(dir, "README.md"))
	if err == nil {
		t.Fatalf("explain succeeded unexpectedly; stdout=%q", stdout)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("explain emitted partial output on non-target ambiguity: %q", stdout)
	}
	if !strings.Contains(err.Error(), "enriching aggregate input records") || !strings.Contains(err.Error(), "ambiguous body section source") {
		t.Fatalf("explain err=%v, want aggregate-input enrichment-owned ambiguity", err)
	}
	if strings.Contains(err.Error(), "resolving explain fields") {
		t.Fatalf("explain err=%v, want non-target enrichment to fail before builder", err)
	}
}

func TestTask9ExplainRelativeNestedTargetDuplicateSourceIsBuilderOwned(t *testing.T) {
	dir := t.TempDir()
	writeTask9ExplainAuthorityStem(t, dir)
	nested := filepath.Join(dir, "nested")
	mustMkdir(t, nested)
	mustWriteFile(t, filepath.Join(nested, "README.md"), []byte("# Nested\n\n## Status\n\nfirst\n\n## Status\n\nsecond\n"), 0o644)
	mustMkdir(t, filepath.Join(nested, "child"))
	mustWriteFile(t, filepath.Join(nested, "child", "README.md"), []byte("# Child\n\n## Status\n\ndone\n"), 0o644)
	declareTestBoundary(t, dir)
	mustChdir(t, dir)

	stdout, err := runTask9CmdStdoutOnly(t, "explain", filepath.Join("nested", "..", "nested", "README.md"))
	assertTask9ExplainBuilderOwnedAmbiguity(t, stdout, err)
}

func writeTask9ExplainAuthorityStem(t *testing.T, dir string) {
	t.Helper()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
root: true
schema:
  status:
    type: string
    source: body.section["## Status"]
aggregate:
  done_children: 'len(filter(children, {.status == "done"}))'
`), 0o644)
}

func assertTask9ExplainBuilderOwnedAmbiguity(t *testing.T, stdout string, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("explain succeeded unexpectedly; stdout=%q", stdout)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("explain emitted partial output on target ambiguity: %q", stdout)
	}
	if !strings.Contains(err.Error(), "resolving explain fields") || !strings.Contains(err.Error(), "ambiguous body section source") {
		t.Fatalf("explain err=%v, want builder-owned source ambiguity", err)
	}
	if strings.Contains(err.Error(), "enriching aggregate input records") {
		t.Fatalf("explain err=%v, want target excluded from aggregate-input enrichment", err)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
}
