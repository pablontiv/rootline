package main

import (
	"strings"
	"testing"
)

// Two YAML documents in one file → ValidateStructure flags multiple_yaml_documents.
const multiDocFile = "---\nfoo: 1\n---\nbody1\n---\nbar: 2\n---\nbody2\n"

func TestValidateAll_MultipleYAMLDocuments(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":    "version: 2\nschema:\n  foo:\n    type: string\n",
		"multi.md": multiDocFile,
	})

	out, err := executeValidate(t, "--all", root)
	if err == nil {
		t.Fatalf("expected validate --all to fail on multiple YAML documents, got nil; out=%s", out)
	}
	if !strings.Contains(out, "multiple_yaml_documents") {
		t.Errorf("expected multiple_yaml_documents in --all output, got: %s", out)
	}
}

func TestValidateAll_SingleDocument_NoStructuralError(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem": "version: 2\nschema:\n  foo:\n    type: string\n",
		"ok.md": "---\nfoo: bar\n---\n# Body\n",
	})

	out, _ := executeValidate(t, "--all", root)
	if strings.Contains(out, "multiple_yaml_documents") {
		t.Errorf("did not expect multiple_yaml_documents for a single-document file, got: %s", out)
	}
}
