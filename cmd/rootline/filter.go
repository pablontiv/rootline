package main

import (
	"context"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/query"
)

// filterRecords applies where expressions to records, returning only those that match.
// Multiple wheres are combined with AND. Empty wheres returns all records (passthrough).
func filterRecords(records []*extract.Record, wheres []string) ([]*extract.Record, error) {
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
	program, err := query.CompileWhere(whereExpr)
	if err != nil {
		return nil, err
	}

	var filtered []*extract.Record
	for _, rec := range records {
		match, err := query.MatchRecord(context.TODO(), program, rec)
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
