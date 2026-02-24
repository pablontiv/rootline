// Package index implements file discovery and scanning.
//
// It walks directory trees, respects .stemignore files,
// and delegates extraction to registered extractors.
package index

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

// ScanOption configures optional Scan behavior.
type ScanOption func(*scanConfig)

type scanConfig struct {
	scopeResolver ScopeResolver
}

// WithScopeResolver adds scope filtering to Scan. The resolver is called
// once per directory to obtain the effective StemFile. If the resolver
// returns nil, no .stem governs the directory and all its files are
// excluded. Files that don't match scope.match are also skipped.
func WithScopeResolver(fn func(dir string) *rules.StemFile) ScanOption {
	return func(c *scanConfig) { c.scopeResolver = fn }
}

// Scan walks rootPath recursively, extracting records from files
// matched by the registry. It respects .stemignore files and always
// excludes .git/ directories. Pass WithScopeResolver to filter files
// by their directory's effective scope.match before extraction.
func Scan(ctx context.Context, rootPath string, registry *extract.Registry, opts ...ScanOption) ([]*extract.Record, error) {
	var cfg scanConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, err
	}

	var records []*extract.Record

	// ignoreStack tracks .stemignore patterns per directory depth.
	var ignoreStack []ignoreEntry

	// scopeCache avoids repeated resolver calls for the same directory.
	var scopeCache map[string]*rules.StemFile
	if cfg.scopeResolver != nil {
		scopeCache = make(map[string]*rules.StemFile)
	}

	err = filepath.WalkDir(absRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Check for context cancellation on each directory.
		if d.IsDir() {
			if ctx.Err() != nil {
				return ctx.Err()
			}
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

		// Apply scope filtering if resolver is configured.
		if cfg.scopeResolver != nil {
			dir := filepath.Dir(path)
			stem, cached := scopeCache[dir]
			if !cached {
				stem = cfg.scopeResolver(dir)
				scopeCache[dir] = stem
			}
			// If the resolver returned nil, no .stem governs this
			// directory — exclude the file from scoped results.
			if stem == nil {
				return nil
			}
			if !MatchesScope(path, stem) {
				return nil
			}
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
		// Use path separator to prevent /sub from matching /sub-extra.
		if absPath != entry.dir && !strings.HasPrefix(absPath, entry.dir+string(filepath.Separator)) {
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
