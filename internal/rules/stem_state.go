package rules

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// StemState captures the prospective .stem files and filesystem inventory for a
// scan rooted at Root. Stems above Root are retained as resolution context, but
// Root remains the ownership boundary for evaluated diagnostics.
type StemState struct {
	Root        string
	Stems       map[string]*StemFile
	ParseErrors map[string]error
	Entries     map[string]StemStateEntry
	// Evaluated marks explicit candidate owners outside Root. Discovery keeps
	// external ancestors context-only; overlays add write targets here so the
	// evaluator checks the exact prospective target chain without owning every
	// untouched external ancestor.
	Evaluated map[string]bool
}

// StemStateEntry records the state-local inventory for a discovered path.
type StemStateEntry struct {
	IsDir   bool
	Ignored bool
}

// DiscoverStemState walks root once, preserving both parseable .stem files and
// malformed .stem parse errors. Ancestor .stem files outside root are retained
// as schema-resolution context.
func DiscoverStemState(ctx context.Context, root string) (*StemState, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	absRoot = filepath.Clean(absRoot)

	state := &StemState{
		Root:        absRoot,
		Stems:       make(map[string]*StemFile),
		ParseErrors: make(map[string]error),
		Entries:     make(map[string]StemStateEntry),
	}
	var ignores []stemIgnoreScope

	scanRoot, err := os.OpenRoot(absRoot)
	if err != nil {
		return nil, fmt.Errorf("opening stem state root %s: %w", absRoot, err)
	}

	walkErr := filepath.Walk(absRoot, func(path string, info os.FileInfo, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return walkErr
		}

		path = filepath.Clean(path)
		if info.IsDir() && path != absRoot && shouldSkipStemStateDir(info.Name()) {
			return filepath.SkipDir
		}

		if info.IsDir() {
			state.Entries[path] = StemStateEntry{IsDir: true}
			if patterns, loadErr := parseStemIgnore(filepath.Join(path, ".stemignore")); loadErr == nil {
				ignores = append(ignores, stemIgnoreScope{dir: path, patterns: patterns})
			}
			return nil
		}

		state.Entries[path] = StemStateEntry{Ignored: stemIsIgnored(path, ignores)}

		if info.Name() == stemFileName {
			content, err := readStemStateFileThroughRoot(scanRoot, absRoot, path)
			if err != nil {
				return err
			}
			addStemToState(state, path, content)
		}

		return nil
	})
	closeErr := scanRoot.Close()
	if walkErr != nil && closeErr != nil {
		return nil, errors.Join(walkErr, fmt.Errorf("closing stem state root %s: %w", absRoot, closeErr))
	}
	if walkErr != nil {
		return nil, walkErr
	}
	if closeErr != nil {
		return nil, fmt.Errorf("closing stem state root %s: %w", absRoot, closeErr)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := collectExternalStemState(ctx, state, filepath.Dir(absRoot)); err != nil {
		return nil, err
	}

	return state, nil
}

// Overlay returns a cloned state with path parsed from content. The receiver is
// never mutated; invalid overlay content returns a contextual error and no clone.
func (s *StemState) Overlay(path string, content []byte) (*StemState, error) {
	if s == nil {
		return nil, fmt.Errorf("overlaying %s: nil stem state", path)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	absPath = filepath.Clean(absPath)

	stem, err := ParseStem(absPath, content)
	if err != nil {
		return nil, fmt.Errorf("overlaying %s: %w", absPath, err)
	}

	clone := s.clone()
	clone.Stems[absPath] = stem
	delete(clone.ParseErrors, absPath)
	clone.Entries[absPath] = StemStateEntry{IsDir: false}
	clone.Entries[filepath.Dir(absPath)] = StemStateEntry{IsDir: true}
	if clone.Evaluated == nil {
		clone.Evaluated = make(map[string]bool)
	}
	clone.Evaluated[absPath] = true
	clone.markCandidateChainEvaluated(absPath)
	return clone, nil
}

// Chain returns parsed .stem entries governing path, ordered root-to-leaf. The
// lookup is state-local: it walks through discovered maps, including ancestor
// context outside Root, and stops after the closest root marker.
func (s *StemState) Chain(path string) []StemEntry {
	if s == nil {
		return nil
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = filepath.Clean(path)
	} else {
		absPath = filepath.Clean(absPath)
	}

	dir := filepath.Dir(absPath)
	if entry, ok := s.Entries[absPath]; ok && entry.IsDir {
		dir = absPath
	}

	var entries []StemEntry
	for {
		stemPath := filepath.Join(dir, stemFileName)
		if stem, ok := s.Stems[stemPath]; ok {
			entries = append(entries, StemEntry{Path: stemPath, Stem: stem})
			if stem.Root {
				break
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
	return entries
}

// EvaluatedStemPaths returns scan-owned .stem paths in deterministic order.
func (s *StemState) EvaluatedStemPaths() []string {
	if s == nil {
		return nil
	}

	seen := make(map[string]struct{})
	for path := range s.Stems {
		if isStemStatePathAtOrBelow(s.Root, path) {
			seen[path] = struct{}{}
		}
	}
	for path := range s.ParseErrors {
		if isStemStatePathAtOrBelow(s.Root, path) {
			seen[path] = struct{}{}
		}
	}
	for path, evaluated := range s.Evaluated {
		if evaluated {
			seen[filepath.Clean(path)] = struct{}{}
		}
	}

	paths := make([]string, 0, len(seen))
	for path := range seen {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

// EffectiveScopeMatches reports which .stem declaration owns at least one
// matching file in the state inventory. The closest non-empty scope.match in a
// file's governing chain owns that match, so inheritance, overrides, and nested
// root boundaries follow the same resolution semantics as record discovery.
func (s *StemState) EffectiveScopeMatches() map[string]bool {
	matches := make(map[string]bool)
	if s == nil {
		return matches
	}

	for path, entry := range s.Entries {
		if entry.IsDir || entry.Ignored || filepath.Base(path) == stemFileName {
			continue
		}

		owner := ""
		pattern := ""
		for _, stemEntry := range s.Chain(path) {
			if stemEntry.Stem != nil && stemEntry.Stem.Scope.Match != "" {
				owner = stemEntry.Path
				pattern = stemEntry.Stem.Scope.Match
			}
		}
		if owner == "" {
			continue
		}

		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err == nil && matched {
			matches[owner] = true
		}
	}
	return matches
}

func (s *StemState) clone() *StemState {
	clone := &StemState{
		Root:        s.Root,
		Stems:       make(map[string]*StemFile, len(s.Stems)),
		ParseErrors: make(map[string]error, len(s.ParseErrors)),
		Entries:     make(map[string]StemStateEntry, len(s.Entries)),
		Evaluated:   make(map[string]bool, len(s.Evaluated)),
	}
	for path, stem := range s.Stems {
		clone.Stems[path] = stem
	}
	for path, err := range s.ParseErrors {
		clone.ParseErrors[path] = err
	}
	for path, entry := range s.Entries {
		clone.Entries[path] = entry
	}
	for path, evaluated := range s.Evaluated {
		clone.Evaluated[path] = evaluated
	}
	return clone
}

func collectExternalStemState(ctx context.Context, state *StemState, startDir string) error {
	for dir := filepath.Clean(startDir); ; dir = filepath.Dir(dir) {
		if err := ctx.Err(); err != nil {
			return err
		}

		stemPath := filepath.Join(dir, stemFileName)
		content, err := os.ReadFile(stemPath)
		switch {
		case err == nil:
			stem, parseErr := ParseStem(stemPath, content)
			if parseErr != nil {
				state.ParseErrors[stemPath] = parseErr
				delete(state.Stems, stemPath)
			} else {
				state.Stems[stemPath] = stem
				delete(state.ParseErrors, stemPath)
				if stem.Root {
					return nil
				}
			}
		case errors.Is(err, fs.ErrNotExist):
		default:
			return fmt.Errorf("reading external stem %s: %w", stemPath, err)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return nil
		}
	}
}

func (s *StemState) markCandidateChainEvaluated(path string) {
	for dir := filepath.Dir(path); ; dir = filepath.Dir(dir) {
		stemPath := filepath.Join(dir, stemFileName)
		if _, malformed := s.ParseErrors[stemPath]; malformed {
			s.Evaluated[stemPath] = true
		}
		if stem := s.Stems[stemPath]; stem != nil && stem.Root {
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
	}
}

func addStemToState(state *StemState, path string, content []byte) {
	path = filepath.Clean(path)
	stem, err := ParseStem(path, content)
	if err != nil {
		state.ParseErrors[path] = err
		delete(state.Stems, path)
		return
	}
	state.Stems[path] = stem
	delete(state.ParseErrors, path)
}

func readStemStateFileThroughRoot(root *os.Root, absRoot, path string) (content []byte, err error) {
	path = filepath.Clean(path)
	rel, err := filepath.Rel(absRoot, path)
	if err != nil {
		return nil, fmt.Errorf("resolving %s relative to stem state root %s: %w", path, absRoot, err)
	}
	if startsWithParentTraversal(rel) {
		return nil, fmt.Errorf("reading %s: path escapes stem state root %s", path, absRoot)
	}

	file, err := root.Open(rel)
	if err != nil {
		return nil, fmt.Errorf("reading %s through stem state root %s: %w", path, absRoot, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("closing %s through stem state root %s: %w", path, absRoot, closeErr)
		}
	}()

	content, err = io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("reading %s through stem state root %s: %w", path, absRoot, err)
	}
	return content, nil
}

func shouldSkipStemStateDir(name string) bool {
	return name == ".git" || name == "node_modules"
}

func isStemStatePathAtOrBelow(root, path string) bool {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	if path == root {
		return true
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != ".." && rel != "." && !startsWithParentTraversal(rel)
}

func startsWithParentTraversal(rel string) bool {
	return rel == ".." || len(rel) > 3 && rel[:3] == ".."+string(filepath.Separator)
}
