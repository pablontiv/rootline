package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// A file target must reach the same verdict as --all on the same tree. The
// preflight lets validate through so it can report the failure in its own
// envelope; if only the --all path re-arms that check, naming a file smuggles
// an undeclared boundary past governance and exits 0.
func TestValidateSingleFileUndeclaredBoundaryFails(t *testing.T) {
	dir := t.TempDir()
	stem := "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  titulo: {type: string, required: true}\n"
	if err := os.WriteFile(filepath.Join(dir, ".stem"), []byte(stem), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("---\ntitulo: Doc\n---\n\n# Doc\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldCwd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	cmd := rootCmd
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"validate", "a.md", "-o", "json"})
	runErr := cmd.Execute()

	var env struct {
		Notices []struct {
			Code     string `json:"code"`
			Severity string `json:"severity"`
		} `json:"notices"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("stdout must carry the envelope, got %q: %v", stdout.String(), err)
	}

	found := false
	for _, n := range env.Notices {
		if n.Code == "schema_resolution_failed" && n.Severity == "error" {
			found = true
		}
	}
	if !found {
		t.Errorf("notices = %+v, want a schema_resolution_failed error notice", env.Notices)
	}
	if runErr == nil {
		t.Error("validate <file> exited 0 against an undeclared boundary; --all exits 1 on the same tree")
	}
}
