package extract

import "testing"

func FuzzExtract(f *testing.F) {
	// Seed corpus: representative markdown patterns.
	seeds := []string{
		"---\nestado: Pending\ntipo: test\n---\n# Title\n",
		"---\n---\n# Empty frontmatter\n",
		"---\ninvalid yaml: [\n---\n# Bad YAML\n",
		"# No frontmatter at all\nJust body text.\n",
		"",
		"---\nkey: value with: colons\nnested:\n  sub: item\n---\n",
		"---\nestado: Completado\n---",
		"\xef\xbb\xbf---\nbom: true\n---\n# BOM file\n",
		"---\ntitle: \"quoted \\\"value\\\"\"\n---\n# Quoted\n",
		"---\nlist:\n  - one\n  - two\n---\n# List frontmatter\n",
	}

	for _, s := range seeds {
		f.Add(s)
	}

	ext := &MarkdownExtractor{}
	f.Fuzz(func(t *testing.T, data string) {
		rec, err := ext.Extract("fuzz.md", []byte(data))
		if err != nil {
			return // errors are OK, panics are not
		}
		if rec == nil {
			t.Fatal("Extract returned nil record without error")
		}
	})
}
