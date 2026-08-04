package rules

import "testing"

// ValidateStructure inspects the leading frontmatter block only. Everything in
// the Markdown body — thematic breaks included — is ordinary content.
func TestValidateStructure_SingleDocument(t *testing.T) {
	cases := map[string]string{
		"no frontmatter":          "# Plain markdown\n",
		"plain frontmatter":       "---\nestado: Pending\n---\n# Body\n",
		"CRLF frontmatter":        "---\r\nfoo: 1\r\n---\r\nbody\r\n",
		"empty frontmatter":       "---\n---\nbody\n",
		"one thematic break":      "---\nfoo: 1\n---\nbody1\n---\nbody2\n",
		"two thematic breaks":     "---\nfoo: 1\n---\nbody1\n---\nbar: 2\n---\nbody2\n",
		"breaks in a fence":       "---\nfoo: 1\n---\n\n```\n---\n---\n```\n",
		"breaks in a tilde fence": "---\nfoo: 1\n---\n\n~~~\n---\n---\n~~~\n",
		"asterisk breaks":         "---\nfoo: 1\n---\n***\nbody\n***\n",
		"underscore breaks":       "---\nfoo: 1\n---\n___\nbody\n___\n",
		"unterminated block":      "---\nfoo: 1\n\nbody\n",
	}

	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			if got := ValidateStructure([]byte(content), "test.md"); got != nil {
				t.Errorf("expected no structural error, got %v", got)
			}
		})
	}
}

// A genuine multi-document frontmatter region is still rejected: the extractor
// would silently keep only the first document.
func TestValidateStructure_MultipleDocuments(t *testing.T) {
	content := []byte("---\nfoo: 1\n...\nbar: 2\n---\nbody\n")
	got := ValidateStructure(content, "test.md")
	if len(got) != 1 {
		t.Fatalf("expected 1 error, got %d (%v)", len(got), got)
	}
	if got[0].Rule != "multiple_yaml_documents" {
		t.Errorf("expected rule multiple_yaml_documents, got %s", got[0].Rule)
	}
	if got[0].Severity != "error" {
		t.Errorf("expected severity error, got %s", got[0].Severity)
	}
}
