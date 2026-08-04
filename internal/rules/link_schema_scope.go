package rules

import (
	"path/filepath"

	"github.com/pablontiv/rootline/internal/extract"
)

// FilterLinksByTypedRules drops links whose type has no rule in the record's
// own effective schema.
//
// Typed-rule filtering used to read the invocation root's stem chain once, so
// a rule declared in a subdirectory was ignored whenever the command ran from
// a parent — the same command already resolved link styles per record, so it
// mixed two different schema-resolution granularities. Both are per record now.
//
// Filtering applies only where typed rules are actually declared: a schema
// carrying only styles or checks must not suppress links, or a styles-only
// repository ends up with an empty graph.
func FilterLinksByTypedRules(records []*extract.Record, root string) {
	for _, rec := range records {
		if len(rec.Links) == 0 {
			continue
		}
		effective := effectiveFor(rec, root)
		if effective == nil || len(effective.Links.Rules) == 0 {
			continue
		}
		filtered := rec.Links[:0]
		for _, link := range rec.Links {
			if _, ok := effective.Links.Rules[link.Type]; ok {
				filtered = append(filtered, link)
			}
		}
		rec.Links = filtered
	}
}

// CycleFailureScope reports, per record path, whether that record's effective
// schema opts into failing on link cycles.
//
// Cycle hardening declared in a subdirectory used to evaporate the moment CI
// ran the command from the repository root, because the opt-in was read from
// the chain above the scan root rather than from the records being scanned.
// Returning the decision per record lets a cycle fail when any node in it is
// governed by a schema that asked for it, without failing cycles elsewhere in
// a tree that never opted in.
func CycleFailureScope(records []*extract.Record, root string) map[string]bool {
	scope := make(map[string]bool, len(records))
	for _, rec := range records {
		effective := effectiveFor(rec, root)
		scope[rec.Path] = effective != nil &&
			effective.Links.Checks != nil &&
			effective.Links.Checks.Cycles
	}
	return scope
}

// effectiveFor resolves a record's effective schema, or nil when none governs
// it. A record without a schema is ungoverned, not an error: graph and query
// both describe trees that may carry no .stem at all.
func effectiveFor(rec *extract.Record, root string) *StemFile {
	dir := filepath.Dir(filepath.Join(root, rec.Path))
	effective, err := ResolveForRecord(dir, rec.Path)
	if err != nil {
		return nil
	}
	return effective
}
