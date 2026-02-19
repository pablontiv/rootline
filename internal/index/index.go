// Package index implements file discovery and scanning.
//
// It walks directory trees, respects .stemignore files,
// and delegates extraction to registered extractors.
package index

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
)

// Scan walks rootPath recursively, extracting records from files
// matched by the registry. It respects .stemignore files and always
// excludes .git/ directories.
func Scan(rootPath string, registry *extract.Registry) ([]*extract.Record, error) {
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, err
	}

	var records []*extract.Record

	// ignoreStack tracks .stemignore patterns per directory depth.
	var ignoreStack []ignoreEntry

	err = filepath.WalkDir(absRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, _ := filepath.Rel(absRoot, path)

		// Always skip .git directories.
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}

		// On entering a directory, check for .stemignore.
		if d.IsDir() {
			stemignorePath := filepath.Join(path, ".stemignore")
			if patterns, loadErr := parseStemignore(stemignorePath); loadErr == nil {
				ignoreStack = append(ignoreStack, ignoreEntry{
					dir:      path,
					patterns: patterns,
				})
			}
			return nil
		}

		// Skip .stemignore files themselves.
		if d.Name() == ".stemignore" {
			return nil
		}

		// Check if file is ignored.
		if isIgnored(path, ignoreStack) {
			return nil
		}

		// Check registry for extractor.
		ext := registry.ForFile(path, "")
		if ext == nil {
			return nil
		}

		// Read and extract.
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		record, extractErr := ext.Extract(rel, content)
		if extractErr != nil {
			return extractErr
		}

		records = append(records, record)
		return nil
	})

	return records, err
}

// ignoreEntry holds patterns from a single .stemignore file.
type ignoreEntry struct {
	dir      string
	patterns []string
}

// parseStemignore reads a .stemignore file and returns its patterns.
// Returns error if file doesn't exist (caller skips).
func parseStemignore(path string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var patterns []string
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns, nil
}

// isIgnored checks if a file path matches any active .stemignore patterns.
func isIgnored(absPath string, stack []ignoreEntry) bool {
	name := filepath.Base(absPath)
	for _, entry := range stack {
		// Only apply if file is under the .stemignore's directory.
		if !strings.HasPrefix(absPath, entry.dir) {
			continue
		}
		for _, pattern := range entry.patterns {
			// Match against filename.
			if matched, _ := filepath.Match(pattern, name); matched {
				return true
			}
			// Match against relative path from ignore dir.
			relFromIgnore, _ := filepath.Rel(entry.dir, absPath)
			if matched, _ := filepath.Match(pattern, relFromIgnore); matched {
				return true
			}
		}
	}
	return false
}
