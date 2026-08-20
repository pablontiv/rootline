package derive

import (
	"context"
	"fmt"
	"sort"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

// EnrichBuiltins populates built-in derived fields on all records.
// These are system-computed fields prefixed with "_" to avoid collision
// with user frontmatter. Currently computes:
//   - isIndex: true if the record is a directory index file
//   - source-derived fields: extracts values from record body based on schema Extract directives
//
// Must run AFTER DeriveAll (so Derived map exists) and BEFORE AggregateAll
// (which uses isIndex for index/non-index classification).
func EnrichBuiltins(ctx context.Context, records []*extract.Record, root string, resolver StemResolver) error {
	if resolver == nil {
		return nil
	}

	resolved, err := resolveBatch(ctx, records, root, resolver)
	if err != nil {
		return err
	}

	type stagedRecord struct {
		record  *extract.Record
		derived map[string]any
	}
	staged := make([]stagedRecord, 0, len(records))

	for _, rec := range records {
		if err := ctx.Err(); err != nil {
			return err
		}

		eff := resolved[rec]
		derived := cloneDerived(rec.Derived)
		derived["isIndex"] = rules.IsIndexFile(rec.Path, eff)

		// Extract source-derived fields from schema through the canonical
		// effective-field resolver. Sorting makes the first failure stable even
		// when several source-backed fields are ambiguous in the same record.
		for _, name := range sourceBackedFieldNames(eff) {
			value, ok, err := rules.ResolveEffectiveField(rec, eff, name)
			if err != nil {
				return fmt.Errorf("resolving source field %q for %s: %w", name, rec.Path, err)
			}
			if !ok {
				continue
			}
			derived[name] = value
		}
		staged = append(staged, stagedRecord{record: rec, derived: derived})
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	for _, s := range staged {
		s.record.Derived = s.derived
	}
	return nil
}

func cloneDerived(in map[string]any) map[string]any {
	out := make(map[string]any, len(in)+1)
	for k, v := range in {
		out[k] = v
	}
	return out
}

func sourceBackedFieldNames(eff *rules.StemFile) []string {
	if eff == nil || eff.Schema == nil {
		return nil
	}
	names := make([]string, 0, len(eff.Schema))
	for name, field := range eff.Schema {
		if field.Extract != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// EnrichBuiltinsSimple runs EnrichBuiltins using the default resolver.
func EnrichBuiltinsSimple(ctx context.Context, records []*extract.Record, root string) error {
	return EnrichBuiltins(ctx, records, root, DefaultResolver())
}
