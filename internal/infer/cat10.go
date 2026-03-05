package infer

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
)

// crossRefRe matches hierarchical path references like E02/F04, E13/F02/S001, etc.
var crossRefRe = regexp.MustCompile(`\b([EFST]\d{2,4}(?:/[EFST]\d{2,4})*)\b`)

// DetectCrossReferences implements Category 10: cross-epic path reference extraction.
// It scans record bodies for hierarchical path patterns and validates they exist
// relative to rootDir.
func DetectCrossReferences(records []*extract.Record, rootDir string) []Inference {
	var inferences []Inference

	for _, rec := range records {
		if rec.Body == "" {
			continue
		}

		matches := crossRefRe.FindAllString(rec.Body, -1)
		seen := make(map[string]bool)

		for _, ref := range matches {
			if seen[ref] {
				continue
			}
			seen[ref] = true

			// Check if the path exists as a directory under rootDir.
			// Match against directory names that start with the ref components.
			resolved := resolveHierarchicalPath(rootDir, ref)

			if resolved != "" {
				inferences = append(inferences, Inference{
					Category: 10,
					Type:     "cross_reference",
					Source:   rec.Path,
					Field:    "body",
					Value:    ref,
					Message:  fmt.Sprintf("cross-reference %q resolves to %s", ref, resolved),
				})
			} else {
				inferences = append(inferences, Inference{
					Category: 10,
					Type:     "broken_cross_reference",
					Source:   rec.Path,
					Field:    "body",
					Value:    ref,
					Message:  fmt.Sprintf("cross-reference %q does not match any path under %s", ref, rootDir),
				})
			}
		}
	}

	return inferences
}

// resolveHierarchicalPath attempts to find a directory matching a hierarchical
// reference like "E02/F04" by scanning for directories whose names start with
// each path component (e.g., E02-*/F04-*).
func resolveHierarchicalPath(rootDir, ref string) string {
	parts := strings.Split(ref, "/")

	current := rootDir
	for _, part := range parts {
		entries, err := os.ReadDir(current)
		if err != nil {
			return ""
		}
		found := ""
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			if len(name) >= len(part) && name[:len(part)] == part && (len(name) == len(part) || name[len(part)] == '-') {
				found = filepath.Join(current, name)
				break
			}
		}
		if found == "" {
			return ""
		}
		current = found
	}

	return current
}
