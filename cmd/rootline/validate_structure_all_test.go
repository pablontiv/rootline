package main

import (
	"path/filepath"
	"strings"
	"testing"
)

const structureStem = "version: 2\nschema:\n  foo:\n    type: string\n"

// Regression suite for issue #83: only the leading "---"-delimited block is
// frontmatter, so thematic breaks and fenced code blocks in the body must never
// be counted as YAML documents.
func TestValidate_BodyThematicBreaksAreContent(t *testing.T) {
	cases := map[string]string{
		"issue #83 reproduction":         "---\nfoo: bar\n---\n\nA.\n\n---\n\nB.\n\n---\n\nC.\n",
		"breaks inside a backtick fence": "---\nfoo: bar\n---\n\n```\n---\n---\n```\n",
		"breaks inside a tilde fence":    "---\nfoo: bar\n---\n\n~~~\n---\n---\n~~~\n",
		"asterisk breaks":                "---\nfoo: bar\n---\n\n***\n\nB.\n\n***\n",
		"underscore breaks":              "---\nfoo: bar\n---\n\n___\n\nB.\n\n___\n",
	}

	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			root := setupValidateProject(t, map[string]string{
				".stem":  structureStem,
				"doc.md": body,
			})

			out, err := executeValidate(t, filepath.Join(root, "doc.md"))
			if err != nil {
				t.Fatalf("expected valid document, got error %v; out=%s", err, out)
			}
			if !strings.Contains(out, `"valid":true`) {
				t.Errorf("expected valid:true, got: %s", out)
			}
		})
	}
}

// A genuine multi-document frontmatter region is still rejected.
func TestValidateAll_MultipleYAMLDocuments(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":    structureStem,
		"multi.md": "---\nfoo: 1\n...\nbar: 2\n---\nbody\n",
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
		".stem": structureStem,
		"ok.md": "---\nfoo: bar\n---\n# Body\n\n---\n\nMore.\n",
	})

	out, _ := executeValidate(t, "--all", root)
	if strings.Contains(out, "multiple_yaml_documents") {
		t.Errorf("did not expect multiple_yaml_documents for a single-document file, got: %s", out)
	}
}

// An unterminated leading block must still produce a clear error, never a
// silent pass — including when the body holds a fence containing "---".
func TestValidate_UnterminatedFrontmatter(t *testing.T) {
	cases := map[string]string{
		"plain":             "---\nfoo: bar\n\nbody\n",
		"fence in body":     "---\nfoo: bar\n\n```\n---\n```\n",
		"no closing at all": "---\nfoo: bar\n",
	}

	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			root := setupValidateProject(t, map[string]string{
				".stem":       structureStem,
				"unclosed.md": content,
			})

			out, err := executeValidate(t, filepath.Join(root, "unclosed.md"))
			if err == nil {
				t.Fatalf("expected an error for unterminated frontmatter, got nil; out=%s", out)
			}
			if !strings.Contains(out, "unclosed frontmatter delimiter") {
				t.Errorf("expected 'unclosed frontmatter delimiter', got: %s", out)
			}
		})
	}
}

// A document with no frontmatter that opens with a "---" thematic break keeps
// the Jekyll/Hugo convention: the leading block is read as frontmatter. Pinned
// here so the behaviour cannot drift silently.
func TestValidate_LeadingThematicBreakIsFrontmatter(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":   structureStem,
		"rule.md": "---\n\nA.\n\n---\n\nB.\n",
	})

	out, err := executeValidate(t, filepath.Join(root, "rule.md"))
	if err == nil {
		t.Fatalf("expected an error, got nil; out=%s", out)
	}
	if !strings.Contains(out, "malformed_yaml") {
		t.Errorf("expected malformed_yaml, got: %s", out)
	}
	if strings.Contains(out, "multiple_yaml_documents") {
		t.Errorf("did not expect multiple_yaml_documents, got: %s", out)
	}
}
