package main

import (
	"context"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/query"
	"github.com/pablontiv/rootline/internal/rules"
)

// whereEnvBuiltins are the names query.BuildEnv injects on every record
// regardless of schema or frontmatter. They are legal in a where expression
// even on a corpus where no document declares anything.
var whereEnvBuiltins = []string{"path", "body", "type", "sections"}

// knownWhereFields is the set of names a where expression or a sort key may
// legally reference: the builtins, every key any record actually carries, and
// every field the effective schema declares — including derive: and aggregate:
// names, which a record only carries once the pipeline has populated them.
//
// The union is deliberately generous. A false "unknown field" on a name that
// does work is worse than the silence this replaces: it teaches callers to
// ignore the warning.
//
// It returns nil for a corpus with neither records nor schema. There is
// nothing to check against there, and every name would look wrong.
func knownWhereFields(records []*extract.Record, absRoot string) []string {
	fields := make(map[string]bool)

	for _, rec := range records {
		for k := range rec.Frontmatter {
			fields[k] = true
		}
		for k := range rec.Derived {
			fields[k] = true
		}
	}

	if entries, err := rules.WalkUp(absRoot); err == nil && len(entries) > 0 {
		merged := rules.MergeStemFiles(entries)
		for k := range merged.Schema {
			fields[k] = true
		}
		for k := range merged.Derive {
			fields[k] = true
		}
		for k := range merged.Aggregate {
			fields[k] = true
		}
	}

	if len(fields) == 0 {
		return nil
	}
	for _, b := range whereEnvBuiltins {
		fields[b] = true
	}
	return slices.Sorted(maps.Keys(fields))
}

// filterRecords applies where expressions to records, returning only those that match.
// Multiple wheres are combined with AND. Empty wheres returns all records (passthrough).
// If knownFields and warn are both non-nil, unknown field names in the
// expression are reported on warn.
func filterRecords(ctx context.Context, records []*extract.Record, wheres []string, knownFields []string, warn io.Writer) ([]*extract.Record, error) {
	// Filter empty strings — StringArrayVar may produce [""] on cobra re-execution.
	var cleaned []string
	for _, w := range wheres {
		if w != "" {
			cleaned = append(cleaned, w)
		}
	}
	if len(cleaned) == 0 {
		return records, nil
	}

	whereExpr := strings.Join(cleaned, " && ")

	if len(knownFields) > 0 && warn != nil {
		for _, w := range query.CheckFieldNames(whereExpr, knownFields) {
			_, _ = fmt.Fprintf(warn, "warning: %s\n", w.Message)
		}
	}

	program, err := query.CompileWhere(whereExpr)
	if err != nil {
		return nil, err
	}

	var filtered []*extract.Record
	for _, rec := range records {
		match, err := query.MatchRecord(ctx, program, rec, nil)
		if err != nil {
			return nil, err
		}
		if match {
			filtered = append(filtered, rec)
		}
	}

	if filtered == nil {
		filtered = []*extract.Record{}
	}
	return filtered, nil
}
