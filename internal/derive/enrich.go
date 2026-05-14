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

		// Extract source-derived fields from schema
		if eff != nil && eff.Schema != nil {
			for name, field := range eff.Schema {
				if field.Extract == "" {
					continue
				}

				var value string
				if field.Extract == "body.h1" {
					value = extractBodyH1(rec.Body)
				} else if strings.HasPrefix(field.Extract, "body.section[") {
					// Parse body.section["## Heading"] format
					start := strings.Index(field.Extract, "[\"")
					end := strings.LastIndex(field.Extract, "\"]")
					if start >= 0 && end > start {
						heading := field.Extract[start+2 : end]
						value = extractBodySection(rec.Body, heading)
					}
				}

				if value != "" {
					rec.Derived[name] = value
				}
			}
		}
	}
}

// extractBodyH1 returns the text of the first H1 heading in the body,
// stripping the "# " prefix. Returns empty string if no H1 is found.
func extractBodyH1(body string) string {
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}

// extractBodySection extracts the content under a specific heading in the body.
// heading should be the full heading line (e.g., "## Heading").
// Returns empty string if the section is not found.
func extractBodySection(body string, heading string) string {
	lines := strings.Split(body, "\n")
	var result []string
	found := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == heading {
			found = true
			continue
		}

		if found {
			// Stop at the next heading (any level)
			if strings.HasPrefix(trimmed, "#") && trimmed != "" {
				break
			}
			// Skip the heading line itself, collect content
			if trimmed != "" {
				result = append(result, line)
			}
		}
	}

	if len(result) > 0 {
		return strings.TrimSpace(strings.Join(result, "\n"))
	}
	return ""
}

// EnrichBuiltinsSimple runs EnrichBuiltins using the default resolver.
func EnrichBuiltinsSimple(ctx context.Context, records []*extract.Record, root string) {
	EnrichBuiltins(ctx, records, root, DefaultResolver())
}
