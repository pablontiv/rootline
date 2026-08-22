package main

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestStats_RespectsScope(t *testing.T) {
	root := t.TempDir()

	// Create three markdown files
	mustWriteFile(t, filepath.Join(root, "gov-a.md"), []byte("---\ntitle: Governed\n---\n# Governed\n"), 0o644)
	mustWriteFile(t, filepath.Join(root, "other.md"), []byte("---\ntitle: Other\n---\n# Other\n"), 0o644)
	mustWriteFile(t, filepath.Join(root, "third.md"), []byte("---\ntitle: Third\n---\n# Third\n"), 0o644)

	// Create .stem with scope.match limiting to gov-*.md
	stemContent := `version: 2
root: true
scope:
  match: "gov-*.md"
schema:
  title:
    type: string
    required: true
`
	mustWriteFile(t, filepath.Join(root, ".stem"), []byte(stemContent), 0o644)

	// Run stats against the root directory
	out, err := runCmd(t, "stats", "--from", root)
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}

	// Parse result
	var result StatsResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("failed to parse stats JSON: %v\nGot: %s", err, out)
	}

	// Should count only gov-a.md (1 record), not all three
	if result.Total != 1 {
		t.Errorf("stats.total = %d, want 1 (only gov-a.md should be counted due to scope.match), got all %d files", result.Total, result.Total)
	}
}
