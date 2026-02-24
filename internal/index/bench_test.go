package index

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func BenchmarkScan(b *testing.B) {
	root := b.TempDir()

	// Create .stem with schema.
	stemContent := "version: 1\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: string\n"
	if err := os.WriteFile(filepath.Join(root, ".stem"), []byte(stemContent), 0600); err != nil {
		b.Fatal(err)
	}

	// Create 100 markdown files.
	for i := 0; i < 100; i++ {
		content := fmt.Sprintf("---\nestado: value%d\n---\n# Doc %d\n", i, i)
		name := fmt.Sprintf("doc%03d.md", i)
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0600); err != nil {
			b.Fatal(err)
		}
	}

	reg := extract.NewRegistry()

	b.ResetTimer()
	for b.Loop() {
		_, err := Scan(context.Background(), root, reg)
		if err != nil {
			b.Fatal(err)
		}
	}
}
