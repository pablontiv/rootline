package derive

import (
	"context"
	"path/filepath"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

// EnrichBuiltins populates built-in derived fields on all records.
// These are system-computed fields prefixed with "_" to avoid collision
// with user frontmatter. Currently computes:
//   - isIndex: true if the record is a directory index file
//
// Must run AFTER DeriveAll (so Derived map exists) and BEFORE AggregateAll
// (which uses isIndex for index/non-index classification).
func EnrichBuiltins(ctx context.Context, records []*extract.Record, root string, resolver StemResolver) {
	if resolver == nil {
		return
	}

	stemCache := make(map[string]*rules.StemFile)

	for _, rec := range records {
		if ctx.Err() != nil {
			return
		}

		absPath := filepath.Join(root, rec.Path)
		dir := filepath.Dir(absPath)

		eff, ok := stemCache[dir]
		if !ok {
			eff = resolver(dir)
			stemCache[dir] = eff
		}

		if rec.Derived == nil {
			rec.Derived = make(map[string]any)
		}
		rec.Derived["isIndex"] = rules.IsIndexFile(rec.Path, eff)
	}
}

// EnrichBuiltinsSimple runs EnrichBuiltins using the default resolver.
func EnrichBuiltinsSimple(ctx context.Context, records []*extract.Record, root string) {
	EnrichBuiltins(ctx, records, root, DefaultResolver())
}
