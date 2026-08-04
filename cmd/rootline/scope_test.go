package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeScopeFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		".stem":    "version: 2\nroot: true\nscope:\n  match: \"T*.md\"\nschema:\n  titulo:\n    type: string\n",
		"T001.md":  "---\ntitulo: T1\n---\n\nSee [[zzz-missing]].\n",
		"other.md": "---\nnope: x\n---\n\nSee [[qqq-missing]].\n",
	}
	for rel, content := range files {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// validate --all honored scope.match while graph did not, so a file the schema
// declared out of governance still became a node and could fail graph --check
// (issue #62 sub-defect 5).
func TestGraphHonorsScopeMatch(t *testing.T) {
	dir := writeScopeFixture(t)
	out, _ := runCmd(t, "graph", dir, "--check")
	if strings.Contains(out, "qqq-missing") {
		t.Errorf("out-of-scope other.md contributed an edge to graph:\n%s", out)
	}
	if !strings.Contains(out, "zzz-missing") {
		t.Errorf("in-scope T001.md should still report its broken link:\n%s", out)
	}
}

// query scanned without scope filtering too.
func TestQueryHonorsScopeMatch(t *testing.T) {
	dir := writeScopeFixture(t)
	out, err := runCmd(t, "query", dir, "--select", "path")
	if err != nil {
		t.Fatalf("query failed: %v\n%s", err, out)
	}
	if strings.Contains(out, "other.md") {
		t.Errorf("out-of-scope other.md appeared in query results:\n%s", out)
	}
}

// validate --all skipped an out-of-scope file while validate <file> validated
// it anyway, so the pre-commit hook and CI enforced different rules on the
// same file (issue #62 sub-defect 11). Naming a file explicitly must not
// smuggle it back into governance — but the skip has to be visible, not
// silent, or the user cannot tell why nothing happened.
func TestValidateNamedFileHonorsScope(t *testing.T) {
	dir := writeScopeFixture(t)
	out, err := runCmd(t, "validate", filepath.Join(dir, "other.md"))
	if err != nil {
		t.Errorf("out-of-scope file should not fail validation, got %v\n%s", err, out)
	}
	if !strings.Contains(out, "out of scope") && !strings.Contains(out, "skipped") {
		t.Errorf("skip must be reported, not silent:\n%s", out)
	}
	if strings.Contains(out, "required field") {
		t.Errorf("out-of-scope file was validated anyway:\n%s", out)
	}
}

func TestValidateNamedFileHonorsStemignore(t *testing.T) {
	dir := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".stem", "version: 2\nroot: true\nscope:\n  match: \"*.md\"\nschema:\n  titulo:\n    type: string\n    required: true\n")
	write(".stemignore", "skipme.md\n")
	write("skipme.md", "---\nzzz: x\n---\n\n# S\n")

	out, err := runCmd(t, "validate", filepath.Join(dir, "skipme.md"))
	if err != nil {
		t.Errorf("ignored file should not fail validation, got %v\n%s", err, out)
	}
	if strings.Contains(out, "required field") {
		t.Errorf("ignored file was validated anyway:\n%s", out)
	}
}
