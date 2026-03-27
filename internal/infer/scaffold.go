package infer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
)

// ScaffoldSchema creates a minimal version-2 .stem file at dirPath by
// reading markdown files and collecting their frontmatter fields.
func ScaffoldSchema(dirPath string) error {
	reg := extract.NewRegistry()

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return err
	}

	fields := make(map[string]bool)
	found := false
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		fpath := filepath.Join(dirPath, e.Name())
		ext := reg.ForFile(fpath, "")
		if ext == nil {
			continue
		}
		content, readErr := os.ReadFile(fpath)
		if readErr != nil {
			continue
		}
		rec, extractErr := ext.Extract(fpath, content)
		if extractErr != nil || rec == nil {
			continue
		}
		found = true
		for key := range rec.Frontmatter {
			fields[key] = true
		}
	}

	if !found {
		return fmt.Errorf("no markdown records found in %s", dirPath)
	}

	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)

	var b strings.Builder
	b.WriteString("version: 2\nschema:\n")
	for _, name := range names {
		fmt.Fprintf(&b, "  %s:\n    type: string\n", name)
	}

	stemPath := filepath.Join(dirPath, ".stem")
	return os.WriteFile(stemPath, []byte(b.String()), 0o644)
}
