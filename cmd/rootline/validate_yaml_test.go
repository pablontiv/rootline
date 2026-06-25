package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFile is a tiny helper for these fixtures.
func writeYAMLFixture(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// malformed: title has an unquoted internal colon → yaml.v3 rejects it.
const malformedDoc = "---\ntitle: Foo: Bar\n---\n# Body\n"

func TestValidate_MalformedYAML_SingleFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeYAMLFixture(t, dir, ".stem", "version: 2\nschema:\n  title:\n    type: string\n    required: true\n")
	f := writeYAMLFixture(t, dir, "broken.md", malformedDoc)

	out, err := runCmd(t, "validate", f)
	if err == nil {
		t.Fatalf("expected validate to fail (exit error) on malformed YAML, got nil; out=%s", out)
	}
	if !strings.Contains(out, "malformed_yaml") {
		t.Errorf("expected malformed_yaml in output, got: %s", out)
	}
}

func TestValidate_MalformedYAML_All(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeYAMLFixture(t, dir, ".stem", "version: 2\nschema:\n  title:\n    type: string\n    required: true\n")
	writeYAMLFixture(t, dir, "broken.md", malformedDoc)

	out, err := runCmd(t, "validate", "--all", dir)
	if err == nil {
		t.Fatalf("expected validate --all to fail on malformed YAML, got nil; out=%s", out)
	}
	if !strings.Contains(out, "malformed_yaml") {
		t.Errorf("expected malformed_yaml in --all output, got: %s", out)
	}
}

func TestValidate_MalformedYAML_ReportsEverything(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	// estado is required; the malformed doc omits it entirely.
	writeYAMLFixture(t, dir, ".stem", "version: 2\nschema:\n  title:\n    type: string\n    required: true\n  estado:\n    type: string\n    required: true\n")
	f := writeYAMLFixture(t, dir, "broken.md", malformedDoc)

	out, err := runCmd(t, "validate", f)
	if err == nil {
		t.Fatalf("expected failure, got nil; out=%s", out)
	}
	if !strings.Contains(out, "malformed_yaml") {
		t.Errorf("expected malformed_yaml in output, got: %s", out)
	}
	// "report everything": the schema error for the missing required field also appears.
	if !strings.Contains(out, "estado") {
		t.Errorf("expected the missing-required schema error for 'estado' to also appear, got: %s", out)
	}
}

func TestValidate_ValidYAML_NoMalformedError(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeYAMLFixture(t, dir, ".stem", "version: 2\nschema:\n  title:\n    type: string\n    required: true\n")
	f := writeYAMLFixture(t, dir, "ok.md", "---\ntitle: \"Foo: Bar\"\n---\n# Body\n")

	out, err := runCmd(t, "validate", f)
	if err != nil {
		t.Fatalf("expected valid file to pass, got error; out=%s", out)
	}
	if strings.Contains(out, "malformed_yaml") {
		t.Errorf("did not expect malformed_yaml for a valid (quoted) title, got: %s", out)
	}
}
