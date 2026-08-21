package derive

import (
	"context"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

// AggregateAll runs hierarchical aggregation on index files.
//
// For each index file (identified by structural.subdirs.require_index,
// default "README.md"), it collects all descendant non-index records and
// evaluates aggregate expressions from the effective .stem.
//
// Processing order is bottom-up (deepest index files first) so that
// child index files have their Derived values computed before parent
// index files read them via the "children" variable.
//
// Results are stored in record.Derived alongside any derive results.
func AggregateAll(ctx context.Context, records []*extract.Record, root string, resolver StemResolver) error {
	if resolver == nil {
		return nil
	}

	resolved, err := resolveBatch(ctx, records, root, resolver)
	if err != nil {
		return err
	}

	// Separate index files from non-index files.
	var indexRecords []*extract.Record
	var nonIndexRecords []*extract.Record

	for _, rec := range records {
		if isIdx, ok := rec.Derived["isIndex"]; ok && isIdx == true {
			indexRecords = append(indexRecords, rec)
		} else {
			nonIndexRecords = append(nonIndexRecords, rec)
		}
	}

	if len(indexRecords) == 0 {
		return nil
	}

	// Sort index records by path depth descending (deepest first → bottom-up).
	sort.Slice(indexRecords, func(i, j int) bool {
		return strings.Count(indexRecords[i].Path, "/") > strings.Count(indexRecords[j].Path, "/")
	})

	for _, idx := range indexRecords {
		// Check for context cancellation between records.
		if err := ctx.Err(); err != nil {
			return err
		}

		eff := resolved[idx]
		if eff == nil || len(eff.Aggregate) == 0 {
			continue
		}

		idxDir := filepath.Dir(idx.Path)

		// When index is at root ("README.md"), filepath.Dir returns "."
		// and prefix "./" won't match any record. Use empty prefix instead
		// so all records are treated as descendants.
		prefix := idxDir + "/"
		if idxDir == "." {
			prefix = ""
		}

		// Collect descendants: all non-index records under this directory.
		var descendants []*extract.Record
		for _, rec := range nonIndexRecords {
			if strings.HasPrefix(rec.Path, prefix) {
				descendants = append(descendants, rec)
			}
		}

		// Collect children: sub-index records (already aggregated by bottom-up order).
		var children []*extract.Record
		for _, other := range indexRecords {
			if other.Path == idx.Path {
				continue
			}
			otherDir := filepath.Dir(other.Path)
			if strings.HasPrefix(otherDir, prefix) {
				children = append(children, other)
			}
		}

		aggregateRecord(idx, eff, descendants, children)
	}
	return nil
}

// aggregateRecord evaluates aggregate expressions for a single index record.
func aggregateRecord(record *extract.Record, effective *rules.StemFile, descendants []*extract.Record, children []*extract.Record) {
	ev := NewEvaluatorWithBuiltins()
	env := BuildEnv(record.Frontmatter)

	// Add descendants (non-index records) as []any.
	descMaps := make([]any, len(descendants))
	for i, d := range descendants {
		descMaps[i] = childToMap(d)
	}
	env["descendants"] = descMaps

	// Add children (sub-index records) with their derived values.
	childMaps := make([]any, len(children))
	for i, c := range children {
		cm := childToMap(c)
		for k, v := range c.Derived {
			cm[k] = v
		}
		childMaps[i] = cm
	}
	env["children"] = childMaps

	// Process aggregate fields in sorted order.
	fields := make([]string, 0, len(effective.Aggregate))
	for k := range effective.Aggregate {
		fields = append(fields, k)
	}
	sort.Strings(fields)

	for _, field := range fields {
		exprVal := effective.Aggregate[field]
		exprStr, ok := exprVal.(string)
		if !ok {
			continue
		}

		compiled, err := ev.Compile(exprStr)
		if err != nil {
			continue
		}

		result, err := ev.Eval(compiled, env)
		if err != nil {
			continue
		}

		if record.Derived == nil {
			record.Derived = make(map[string]any)
		}
		record.Derived[field] = result
	}
}

// AggregateAllSimple runs aggregation using the default resolver.
func AggregateAllSimple(ctx context.Context, records []*extract.Record, root string) error {
	return AggregateAll(ctx, records, root, DefaultResolver())
}
