package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func TestCollectLinkCheckIssuesReturnsInvalidGovernanceError(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".stem"), []byte(`version: 2
root: true
schema:
  id:
    type: sequence
    match:
      "T*": {prefix: T, digits: 2.0}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	issues, err := collectLinkCheckIssues([]*extract.Record{{
		Path:  "T001-task.md",
		Links: []extract.Link{{Target: "missing.md", Style: extract.StyleMarkdown}},
	}}, root)
	if err == nil || !strings.Contains(err.Error(), "digits") {
		t.Fatalf("issues=%#v error=%v, want strict sequence resolution cause", issues, err)
	}
}
