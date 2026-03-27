package infer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pablontiv/rootline/internal/rules"
)

// DetectMissingSchemata walks directories from scanRoot and flags those
// containing markdown files but governed by no .stem schema.
func DetectMissingSchemata(scanRoot string) []Inference {
	var inferences []Inference
	var dirs []string

	if err := filepath.WalkDir(scanRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == ".stemignore" || (len(name) > 0 && name[0] == '.') {
				return filepath.SkipDir
			}
			dirs = append(dirs, path)
		}
		return nil
	}); err != nil {
		return inferences
	}

	sort.Strings(dirs)

	for _, dir := range dirs {
		mdCount := countMarkdownFiles(dir)
		if mdCount == 0 {
			continue
		}

		entries, err := rules.WalkUp(dir)
		if err != nil || len(entries) == 0 {
			inferences = append(inferences, Inference{
				Type:    "missing_schema",
				Source:  dir,
				Message: fmt.Sprintf("Directory contains %d markdown file(s) but no .stem schema (checked walk-up to .git root)", mdCount),
			})
			continue
		}

		// WalkUp returns entries in root-to-leaf order: entries[0] is root-most,
		// entries[len-1] is leaf-most (physically closest to the target directory).
		closest := entries[len(entries)-1].Path
		stemDir := filepath.Dir(closest)
		relPath, _ := filepath.Rel(stemDir, dir)
		depth := strings.Count(relPath, string(os.PathSeparator))
		if depth >= 2 {
			inferences = append(inferences, Inference{
				Type:    "implicit_schema",
				Source:  dir,
				Value:   closest,
				Message: fmt.Sprintf("Directory inherits schema from %s (%d levels up) — consider adding local .stem for explicit governance", closest, depth),
			})
		}
	}

	return inferences
}

func countMarkdownFiles(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			count++
		}
	}
	return count
}
