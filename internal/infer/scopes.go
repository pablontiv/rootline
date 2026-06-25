package infer

import (
	"path/filepath"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

// StemResolver returns the effective (merged) stem for a directory and the path
// of the closest (leaf-most) .stem governing it — the scope group key.
// Returns (nil, "") when no .stem governs the directory.
type StemResolver func(dir string) (*rules.StemFile, string)

// ScopeGroup is the set of records governed by one effective .stem.
type ScopeGroup struct {
	Key     string
	Stem    *rules.StemFile
	Records []*extract.Record
}

// GroupByScope buckets records by the closest .stem governing them, preserving
// first-appearance order of scope keys.
func GroupByScope(records []*extract.Record, root string, resolve StemResolver) []ScopeGroup {
	var order []string
	byKey := make(map[string]*ScopeGroup)
	for _, rec := range records {
		dir := filepath.Dir(filepath.Join(root, rec.Path))
		stem, key := resolve(dir)
		g, ok := byKey[key]
		if !ok {
			g = &ScopeGroup{Key: key, Stem: stem}
			byKey[key] = g
			order = append(order, key)
		}
		g.Records = append(g.Records, rec)
	}
	groups := make([]ScopeGroup, 0, len(order))
	for _, k := range order {
		groups = append(groups, *byKey[k])
	}
	return groups
}

// DefaultStemResolver resolves each directory's effective stem via WalkUp +
// MergeStemFiles, caching per directory to avoid re-walking per record. The
// group key is the leaf-most .stem path.
//
// The cache is unsynchronized and is intended for sequential use within a
// single command's detector loop (e.g. analyze, schema propose), not for
// concurrent callers from multiple goroutines.
func DefaultStemResolver() StemResolver {
	type entry struct {
		stem *rules.StemFile
		key  string
	}
	cache := make(map[string]entry)
	return func(dir string) (*rules.StemFile, string) {
		if e, ok := cache[dir]; ok {
			return e.stem, e.key
		}
		var e entry
		entries, err := rules.WalkUp(dir)
		if err == nil && len(entries) > 0 {
			e.stem = rules.MergeStemFiles(entries)
			e.key = entries[len(entries)-1].Path
		}
		cache[dir] = e
		return e.stem, e.key
	}
}
