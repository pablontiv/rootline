package derive

import (
	"context"
	"path/filepath"
	"strings"

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
func EnrichBuiltins(ctx context.Context, records []*extract.Record, root string, resolver StemResolver) {
	if resolver == nil {
		return
	}

	for _, rec := range records {
		if ctx.Err() != nil {
			return
		}

		absPath := filepath.Join(root, rec.Path)
		dir := filepath.Dir(absPath)

		eff := resolver(dir, rec.Path)

		if rec.Derived == nil {
			rec.Derived = make(map[string]any)
		}
		rec.Derived["isIndex"] = rules.IsIndexFile(rec.Path, eff)

		// Extract source-derived fields from schema
		if eff != nil && eff.Schema != nil {
			for name, field := range eff.Schema {
				if field.Extract == "" {
					continue
				}

				var value string
				if field.Extract == "body.h1" {
					value = extract.ExtractBodyH1(rec.Body)
				} else if strings.HasPrefix(field.Extract, "body.section[") {
					// Parse body.section["## Heading"] format
					start := strings.Index(field.Extract, "[\"")
					end := strings.LastIndex(field.Extract, "\"]")
					if start >= 0 && end > start {
						heading := field.Extract[start+2 : end]
						value = extract.ExtractBodySection(rec.Body, heading)
					}
				}

				if value != "" {
					rec.Derived[name] = value
				}
			}
		}
	}
}

// EnrichBuiltinsSimple runs EnrichBuiltins using the default resolver.
func EnrichBuiltinsSimple(ctx context.Context, records []*extract.Record, root string) {
	EnrichBuiltins(ctx, records, root, DefaultResolver())
}
