package derive

import (
	"path/filepath"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

// StemResolver returns the effective .stem for a directory.
type StemResolver func(dir string) *rules.StemFile

// DeriveAll runs derivation on all records, grouping children by parent
// directory. For each record it resolves the effective .stem, collects
// sibling records as children (for aggregation builtins), and evaluates
// derive expressions. Errors are silently skipped (derivation is best-effort).
func DeriveAll(records []*extract.Record, root string, resolver StemResolver) {
	if resolver == nil {
		return
	}

	// Group records by parent directory for children lookup.
	byDir := make(map[string][]*extract.Record)
	for _, rec := range records {
		dir := filepath.Dir(rec.Path)
		byDir[dir] = append(byDir[dir], rec)
	}

	// Build a record resolver for linked field injection.
	recResolver := NewMapResolver(records)

	// Cache effective stems per directory.
	stemCache := make(map[string]*rules.StemFile)

	for _, rec := range records {
		absPath := filepath.Join(root, rec.Path)
		dir := filepath.Dir(absPath)

		eff, ok := stemCache[dir]
		if !ok {
			eff = resolver(dir)
			stemCache[dir] = eff
		}

		if eff == nil || len(eff.Derive) == 0 {
			continue
		}

		// Children: sibling records in the same parent directory.
		parentDir := filepath.Dir(rec.Path)
		siblings := byDir[parentDir]
		var children []*extract.Record
		for _, s := range siblings {
			if s.Path != rec.Path {
				children = append(children, s)
			}
		}

		// Best-effort: ignore derivation errors.
		_, _ = DeriveRecord(rec, eff, children, WithResolver(recResolver))
	}
}

// DefaultResolver creates a StemResolver that uses WalkUp + MergeStemFiles.
func DefaultResolver() StemResolver {
	return func(dir string) *rules.StemFile {
		entries, err := rules.WalkUp(dir)
		if err != nil {
			return nil
		}
		return rules.MergeStemFiles(entries)
	}
}

// DeriveAllSimple runs derivation using the default resolver. This is the
// convenience function for CLI commands.
func DeriveAllSimple(records []*extract.Record, root string) {
	DeriveAll(records, root, DefaultResolver())
}

// HasDeriveFields reports whether any .stem in the record tree has derive
// expressions. This avoids the overhead of stem resolution when derivation
// is not configured. For simplicity, this always returns true — the actual
// check happens inside DeriveAll per-directory.
func HasDeriveFields(records []*extract.Record) bool {
	return len(records) > 0
}
