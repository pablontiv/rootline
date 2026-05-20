package rules

import "testing"

func TestValidateStructure_NoFrontmatter(t *testing.T) {
	got := ValidateStructure([]byte("# Plain markdown\n"), "test.md")
	if got != nil {
		t.Errorf("expected nil for non-frontmatter file, got %v", got)
	}
}

func TestValidateStructure_SingleDocument(t *testing.T) {
	content := []byte("---\nestado: Pending\n---\n# Body\n")
	got := ValidateStructure(content, "test.md")
	if got != nil {
		t.Errorf("expected nil for single document, got %v", got)
	}
}

func TestValidateStructure_MultipleDocuments(t *testing.T) {
	content := []byte("---\nfoo: 1\n---\nbody1\n---\nbar: 2\n---\nbody2\n")
	got := ValidateStructure(content, "test.md")
	if len(got) != 1 {
		t.Fatalf("expected 1 error, got %d", len(got))
	}
	if got[0].Rule != "multiple_yaml_documents" {
		t.Errorf("expected rule multiple_yaml_documents, got %s", got[0].Rule)
	}
	if got[0].Severity != "error" {
		t.Errorf("expected severity error, got %s", got[0].Severity)
	}
}

func TestValidateStructure_CRLFFrontmatter(t *testing.T) {
	content := []byte("---\r\nfoo: 1\r\n---\r\nbody\r\n")
	got := ValidateStructure(content, "test.md")
	if got != nil {
		t.Errorf("expected nil for valid CRLF frontmatter, got %v", got)
	}
}
