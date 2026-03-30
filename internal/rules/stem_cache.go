package rules

import (
	"os"
	"path/filepath"
	"slices"
)

// StemCache caches parsed .stem files by directory path, avoiding repeated
// filesystem walks and YAML parsing within a single request scope.
type StemCache struct {
	parsed map[string]*stemCacheEntry
	hits   int
}

type stemCacheEntry struct {
	stem *StemFile
	ok   bool
}

// NewStemCache creates a new request-scoped StemCache.
func NewStemCache() *StemCache {
	return &StemCache{parsed: make(map[string]*stemCacheEntry)}
}

// Hits returns the number of cache hits (directory lookups served from cache).
func (c *StemCache) Hits() int {
	return c.hits
}

// WalkUp walks from targetPath up to the repository root (.git boundary),
// collecting and parsing .stem files along the way. Results are cached per
// directory so repeated calls for files in the same tree skip disk I/O.
// Returns entries ordered root-to-leaf (ready for top-down merge).
func (c *StemCache) WalkUp(targetPath string) ([]StemEntry, error) {
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return nil, err
	}

	dir := absTarget
	if info, err := os.Stat(absTarget); err == nil && !info.IsDir() {
		dir = filepath.Dir(absTarget)
	}

	var entries []StemEntry
	for {
		entry, err := c.loadStem(dir)
		if err != nil {
			return nil, err
		}
		if entry != nil {
			entries = append(entries, *entry)
		}

		gitPath := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			break
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, nil
		}
		dir = parent
	}

	slices.Reverse(entries)
	return entries, nil
}

func (c *StemCache) loadStem(dir string) (*StemEntry, error) {
	if cached, ok := c.parsed[dir]; ok {
		c.hits++
		if !cached.ok {
			return nil, nil
		}
		return &StemEntry{Path: filepath.Join(dir, stemFileName), Stem: cached.stem}, nil
	}

	stemPath := filepath.Join(dir, stemFileName)
	content, err := os.ReadFile(stemPath)
	if err != nil {
		c.parsed[dir] = &stemCacheEntry{ok: false}
		return nil, nil
	}

	stem, err := ParseStem(stemPath, content)
	if err != nil {
		return nil, err
	}

	c.parsed[dir] = &stemCacheEntry{stem: stem, ok: true}
	return &StemEntry{Path: stemPath, Stem: stem}, nil
}
