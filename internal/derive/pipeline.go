package derive

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

// StemResolver returns the effective .stem for a record, respecting match-scoped schema fields.
type StemResolver func(dir, recordPath string) (*rules.StemFile, error)

// DeriveAll runs derivation on all records, grouping children by parent
// directory. For each record it resolves the effective .stem per-record,
// collects sibling records as children (for aggregation builtins), and evaluates
// derive expressions. Errors are silently skipped (derivation is best-effort).
//
// Note: Resolution is per-record via ResolveForRecord so that match:-scoped
// schema fields apply only to records they match. Derive and Aggregate rules are
// unaffected by match filtering and behave identically per record.
func DeriveAll(ctx context.Context, records []*extract.Record, root string, resolver StemResolver) error {
	if resolver == nil {
		return nil
	}

	resolved, err := resolveBatch(ctx, records, root, resolver)
	if err != nil {
		return err
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
		if err := ctx.Err(); err != nil {
			return err
		}

		eff := resolved[rec]
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

		// Best-effort: ignore expression evaluation errors after schema resolution succeeded.
		_, _ = DeriveRecord(ctx, rec, eff, children, WithResolver(recResolver))
	}
	return nil
}

// DefaultResolver creates a StemResolver that uses ResolveForRecord for per-record resolution.
func DefaultResolver() StemResolver {
	return func(dir, recordPath string) (*rules.StemFile, error) {
		stem, err := rules.ResolveForRecord(dir, recordPath)
		if errors.Is(err, rules.ErrNoSchemaFound) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		return stem, nil
	}
}

// DeriveAllSimple runs derivation using the default resolver. This is the
// convenience function for CLI commands.
func DeriveAllSimple(ctx context.Context, records []*extract.Record, root string) error {
	return DeriveAll(ctx, records, root, DefaultResolver())
}

func resolveBatch(ctx context.Context, records []*extract.Record, root string, resolver StemResolver) (map[*extract.Record]*rules.StemFile, error) {
	resolved := make(map[*extract.Record]*rules.StemFile, len(records))
	for _, rec := range records {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		absPath := filepath.Join(root, rec.Path)
		dir := filepath.Dir(absPath)
		eff, err := resolver(dir, rec.Path)
		if err != nil {
			return nil, fmt.Errorf("resolve schema for %s: %w", rec.Path, err)
		}
		resolved[rec] = eff
	}
	return resolved, nil
}

// HasDeriveFields reports whether any .stem in the record tree has derive
// expressions. This avoids the overhead of stem resolution when derivation
// is not configured. For simplicity, this always returns true — the actual
// check happens inside DeriveAll per-directory.
func HasDeriveFields(records []*extract.Record) bool {
	return len(records) > 0
}
