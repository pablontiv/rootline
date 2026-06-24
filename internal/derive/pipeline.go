package derive

import (
	"context"
	"path/filepath"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

// StemResolver returns the effective .stem for a record, respecting match-scoped schema fields.
type StemResolver func(dir, recordPath string) *rules.StemFile

// DeriveAll runs derivation on all records, grouping children by parent
// directory. For each record it resolves the effective .stem per-record,
// collects sibling records as children (for aggregation builtins), and evaluates
// derive expressions. Errors are silently skipped (derivation is best-effort).
//
// Note: Resolution is per-record via ResolveForRecord so that match:-scoped
// schema fields apply only to records they match. Derive and Aggregate rules are
// unaffected by match filtering and behave identically per record.
func DeriveAll(ctx context.Context, records []*extract.Record, root string, resolver StemResolver) {
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

	for _, rec := range records {
		// Check for context cancellation between records.
		if ctx.Err() != nil {
			return
		}

		absPath := filepath.Join(root, rec.Path)
		dir := filepath.Dir(absPath)

		eff := resolver(dir, rec.Path)

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
		_, _ = DeriveRecord(ctx, rec, eff, children, WithResolver(recResolver))
	}
}

// DefaultResolver creates a StemResolver that uses ResolveForRecord for per-record resolution.
func DefaultResolver() StemResolver {
	return func(dir, recordPath string) *rules.StemFile {
		stem, err := rules.ResolveForRecord(dir, recordPath)
		if err != nil {
			return nil
		}
		return stem
	}
}

// DeriveAllSimple runs derivation using the default resolver. This is the
// convenience function for CLI commands.
func DeriveAllSimple(ctx context.Context, records []*extract.Record, root string) {
	DeriveAll(ctx, records, root, DefaultResolver())
}

// HasDeriveFields reports whether any .stem in the record tree has derive
// expressions. This avoids the overhead of stem resolution when derivation
// is not configured. For simplicity, this always returns true — the actual
// check happens inside DeriveAll per-directory.
func HasDeriveFields(records []*extract.Record) bool {
	return len(records) > 0
}
