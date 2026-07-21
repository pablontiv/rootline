// Package index implements file discovery and scanning.
//
// It walks directory trees, respects .stemignore files,
// and delegates extraction to registered extractors.
package index

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

// fileEntry holds paths collected during Phase 1 discovery.
type fileEntry struct {
	absPath string
	relPath string
}

// ScanOption configures optional Scan behavior.
type ScanOption func(*scanConfig)

type scanConfig struct {
	scopeResolver   ScopeResolver
	allowUngoverned bool
}

// AllowUngoverned lets a scan read documents in a tree where no .stem resolves.
//
// It exists for the bootstrap commands only. `schema propose` and `analyze`
// derive a schema FROM documents that do not have one yet, so demanding a
// resolved schema before reading them would make them useless for their only
// purpose — you cannot require a schema as the precondition for creating one.
//
// It tolerates a MISSING schema, never a broken one: hard read and parse errors
// still abort the scan. Governed commands must not use it, because succeeding
// over zero governed records is exactly the false green this change removes.
func AllowUngoverned() ScanOption {
	return func(c *scanConfig) { c.allowUngoverned = true }
}

// WithScopeResolver adds scope filtering to Scan. The resolver is called
// once per directory to obtain the effective StemFile. If the resolver
// returns an error:
// - a hard error aborts the scan
// - ErrNoSchemaFound skips that directory and continues
// If the resolver returns nil with no error, all files in that directory
// are excluded. Files that don't match scope.match are also skipped.
func WithScopeResolver(fn func(dir string) (*rules.StemFile, error)) ScanOption {
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

	// Phase 1: Sequential discovery — walk the tree collecting file paths.
	var files []fileEntry

	var ignoreStack []ignoreEntry

	var scopeCache map[string]*rules.StemFile
	var scopeErrors map[string]error
	var hitErrNoSchemaFound bool
	if cfg.scopeResolver != nil {
		scopeCache = make(map[string]*rules.StemFile)
		scopeErrors = make(map[string]error)
	}

	err = filepath.WalkDir(absRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if ctx.Err() != nil {
				return ctx.Err()
			}
		}

		rel, _ := filepath.Rel(absRoot, path)

		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}

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

		if d.Name() == ".stemignore" {
			return nil
		}

		if isIgnored(path, ignoreStack) {
			return nil
		}

		if cfg.scopeResolver != nil {
			dir := filepath.Dir(path)
			stem, cached := scopeCache[dir]
			var resolveErr error
			if !cached {
				stem, resolveErr = cfg.scopeResolver(dir)
				scopeCache[dir] = stem
				scopeErrors[dir] = resolveErr
			} else {
				resolveErr = scopeErrors[dir]
			}

			// Hard error from resolver aborts the scan.
			if resolveErr != nil && !errors.Is(resolveErr, rules.ErrNoSchemaFound) {
				return resolveErr
			}

			// ErrNoSchemaFound for this directory means skip it, and mark that we hit it.
			// Bootstrap scans instead read the file with no schema applied, since
			// deriving a schema requires reading documents that lack one.
			ungoverned := errors.Is(resolveErr, rules.ErrNoSchemaFound)
			if ungoverned {
				hitErrNoSchemaFound = true
				if !cfg.allowUngoverned {
					return nil
				}
			}

			// An ungoverned bootstrap scan has no scope to filter by, so every
			// readable file is in scope.
			if !ungoverned {
				if stem == nil {
					return nil
				}
				if !MatchesScope(path, stem) {
					return nil
				}
			}
		}

		if registry.ForFile(path, "") == nil {
			return nil
		}

		files = append(files, fileEntry{absPath: path, relPath: rel})
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		// Report ErrNoSchemaFound only when a directory actually failed to
		// resolve. An empty result because there was simply nothing to scan is
		// a legitimate success. Bootstrap scans never report it: reading
		// schemaless documents is their purpose.
		if hitErrNoSchemaFound && !cfg.allowUngoverned {
			return nil, rules.ErrNoSchemaFound
		}
		return nil, nil
	}

	// Phase 2: Parallel extraction via worker pool.
	type result struct {
		record *extract.Record
		err    error
	}

	numWorkers := runtime.NumCPU()
	if numWorkers > len(files) {
		numWorkers = len(files)
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	results := make([]result, len(files))
	var wg sync.WaitGroup
	work := make(chan int, len(files))

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range work {
				if ctx.Err() != nil {
					results[i] = result{err: ctx.Err()}
					continue
				}
				f := files[i]
				content, readErr := os.ReadFile(f.absPath)
				if readErr != nil {
					results[i] = result{err: readErr}
					continue
				}
				ext := registry.ForFile(f.absPath, "")
				rec, extractErr := ext.Extract(f.relPath, content)
				if extractErr != nil {
					results[i] = result{err: extractErr}
					continue
				}
				results[i] = result{record: rec}
			}
		}()
	}

	for i := range files {
		work <- i
	}
	close(work)
	wg.Wait()

	// Collect results preserving discovery order.
	records := make([]*extract.Record, 0, len(files))
	for _, r := range results {
		if r.err != nil {
			return nil, r.err
		}
		if r.record != nil {
			records = append(records, r.record)
		}
	}

	return records, nil
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
