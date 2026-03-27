package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/query"
)

// filterRecords applies where expressions to records, returning only those that match.
// Multiple wheres are combined with AND. Empty wheres returns all records (passthrough).
// If knownFields is non-nil, unknown field names in the expression emit warnings to stderr.
func filterRecords(ctx context.Context, records []*extract.Record, wheres []string, knownFields []string) ([]*extract.Record, error) {
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

	if len(knownFields) > 0 {
		warnings := query.CheckFieldNames(whereExpr, knownFields)
		for _, w := range warnings {
			fmt.Fprintf(os.Stderr, "warning: %s\n", w.Message)
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
