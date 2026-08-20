package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestValidateProducerSourcesExtractionMalformedSingleAndAll(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":  "version: 2\nroot: true\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n",
		"bad.md": "---\ntitle: [broken\n---\n# Bad\n",
	})
	mustChdir(t, root)

	stdout, err := executeValidate(t, "bad.md", "-o", "json")
	if err != ErrValidationFailed {
		t.Fatalf("single err = %v, want ErrValidationFailed\nstdout=%s", err, stdout)
	}
	if got := validationErrorSourcesByRule(t, firstResult(t, stdout)); !reflect.DeepEqual(got, map[string][]string{"malformed_yaml": {"bad.md"}}) {
		t.Fatalf("single sources = %#v", got)
	}

	stdout, err = executeValidate(t, "--all", "-o", "json")
	if err != ErrValidationFailed {
		t.Fatalf("all err = %v, want ErrValidationFailed\nstdout=%s", err, stdout)
	}
	env := decodeEnvelope(t, stdout)
	byPath := validationResultsByPath(t, env)
	if got := validationErrorSourcesByRule(t, byPath["bad.md"]); !reflect.DeepEqual(got, map[string][]string{"malformed_yaml": {"bad.md"}}) {
		t.Fatalf("all sources = %#v", got)
	}
}

func TestValidateProducerSourcesStructureRecordOwned(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":  "version: 2\nroot: true\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n",
		"bad.md": "---\ntitle: Bad\n...\nextra: doc\n---\n# Bad\n",
	})
	subdir := filepath.Join(root, "subdir")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	mustChdir(t, subdir)

	stdout, err := executeValidate(t, "../bad.md", "-o", "json")
	if err != ErrValidationFailed {
		t.Fatalf("err = %v, want ErrValidationFailed\nstdout=%s", err, stdout)
	}
	row := firstResult(t, stdout)
	if row["path"] != "../bad.md" {
		t.Fatalf("path = %v, want invocation spelling", row["path"])
	}
	if got := validationErrorSourcesByRule(t, row); !reflect.DeepEqual(got, map[string][]string{"multiple_yaml_documents": {"bad.md"}}) {
		t.Fatalf("sources = %#v", got)
	}
}

func TestValidateProducerSourcesPhysicalLinkSchemaStem(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem": `version: 2
root: true
scope:
  match: "*.md"
links:
  allowed: [blocks]
`,
		"doc.md":    "---\ntitle: Doc\n---\n# Doc\n\n[[target]]\n",
		"target.md": "---\ntitle: Target\n---\n# Target\n",
	})
	mustChdir(t, root)

	stdout, err := executeValidate(t, "doc.md", "-o", "json")
	if err != ErrValidationFailed {
		t.Fatalf("err = %v, want ErrValidationFailed\nstdout=%s", err, stdout)
	}
	got := validationErrorSourcesByRule(t, firstResult(t, stdout))
	if !reflect.DeepEqual(got, map[string][]string{"link_type": {".stem"}}) {
		t.Fatalf("sources = %#v, want only physical link_type .stem diagnostic", got)
	}
}

func TestValidateProducerSourcesLinksChecksSymbolicErrorAndWarning(t *testing.T) {
	t.Run("encoding error", func(t *testing.T) {
		root := setupValidateProject(t, map[string]string{
			".stem": `version: 2
root: true
scope:
  match: "*.md"
links:
  checks:
    resolve: false
    encoding: true
`,
			"doc.md": "---\ntitle: Doc\n---\n# Doc\n\n[[bad target]]\n",
		})
		mustChdir(t, root)

		stdout, err := executeValidate(t, "doc.md", "-o", "json")
		if err != ErrValidationFailed {
			t.Fatalf("err = %v, want ErrValidationFailed\nstdout=%s", err, stdout)
		}
		got := validationErrorSourcesByRule(t, firstResult(t, stdout))
		if !reflect.DeepEqual(got, map[string][]string{"link_encoding": {"links.checks"}}) {
			t.Fatalf("sources = %#v", got)
		}
	})

	t.Run("basename fallback warning", func(t *testing.T) {
		root := setupValidateProject(t, map[string]string{
			".stem": `version: 2
root: true
scope:
  match: "*.md"
links:
  basename_fallback: true
  checks:
    resolve: true
`,
			"doc.md": "---\ntitle: Doc\n---\n# Doc\n\n[[missing-bare]]\n",
		})
		mustChdir(t, root)

		stdout, err := executeValidate(t, "doc.md", "-o", "json")
		if err != nil {
			t.Fatalf("err = %v, want warning-only success\nstdout=%s", err, stdout)
		}
		got := validationWarningSourcesByRule(t, firstResult(t, stdout))
		if !reflect.DeepEqual(got, map[string][]string{"link_unverifiable": {"links.checks"}}) {
			t.Fatalf("warning sources = %#v", got)
		}
	})
}
