package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkWalkUp(b *testing.B) {
	// Create a 10-level deep directory tree with .stem at each level.
	root := b.TempDir()
	dir := root

	// Create .git at root to set boundary.
	if err := os.Mkdir(filepath.Join(root, ".git"), 0750); err != nil {
		b.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		dir = filepath.Join(dir, fmt.Sprintf("level%d", i))
		if err := os.Mkdir(dir, 0750); err != nil {
			b.Fatal(err)
		}
		stemPath := filepath.Join(dir, ".stem")
		content := fmt.Sprintf("version: 2\nschema:\n  field%d:\n    type: string\n", i)
		if err := os.WriteFile(stemPath, []byte(content), 0600); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for b.Loop() {
		_, err := WalkUp(dir)
		if err != nil {
			b.Fatal(err)
		}
	}
}
