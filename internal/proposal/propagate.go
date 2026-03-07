package proposal

import (
	"fmt"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

// DetectPropagateAggregate finds index files where a frontmatter field
// differs from its aggregate-computed value and emits PropagateAggregate
// proposals to correct the stored value.
//
// Before emitting a proposal, it performs a formula completeness pre-check:
// every distinct descendant value for the field must appear as a quoted
// string in the aggregate expression. If any value is missing, the formula
// may produce incorrect results, so propagation is skipped for that field.
func DetectPropagateAggregate(records []*extract.Record, effective *rules.StemFile) []Proposal {
	if effective == nil || len(effective.Aggregate) == 0 {
		return nil
	}

	// Build map of records by path for descendant lookup.
	recordMap := make(map[string]*extract.Record)
	for _, rec := range records {
		recordMap[rec.Path] = rec
	}

	var proposals []Proposal

	for _, rec := range records {
		// Only index files (README.md) can have aggregate values.
		if !strings.HasSuffix(rec.Path, "README.md") {
			continue
		}

		dir := strings.TrimSuffix(rec.Path, "README.md")

		for field, exprAny := range effective.Aggregate {
			expr, ok := exprAny.(string)
			if !ok {
				continue
			}

			// Get computed value from derive pipeline.
			derivedVal, hasDerived := rec.Derived[field]
			if !hasDerived {
				continue
			}

			// Get stored value from frontmatter.
			storedVal, hasStored := rec.Frontmatter[field]
			if !hasStored {
				continue
			}

			derivedStr := fmt.Sprintf("%v", derivedVal)
			storedStr := fmt.Sprintf("%v", storedVal)

			// Skip if values already match.
			if derivedStr == storedStr {
				continue
			}

			// Formula pre-check: collect distinct descendant values for this field.
			descendantValues := collectDescendantValues(field, dir, rec.Path, records)
			if !formulaCoversValues(expr, descendantValues) {
				continue
			}

			proposals = append(proposals, Proposal{
				Type:        PropagateAggregate,
				Field:       field,
				Description: fmt.Sprintf("aggregate computes %q but stored value is %q", derivedStr, storedStr),
				Paths:       []string{rec.Path},
				From:        storedStr,
				To:          derivedStr,
			})
		}
	}

	return proposals
}

// collectDescendantValues gathers all distinct values for a field from
// descendant records (children in the same directory, excluding the index itself).
func collectDescendantValues(field, dir, indexPath string, records []*extract.Record) map[string]bool {
	values := make(map[string]bool)
	for _, rec := range records {
		if rec.Path == indexPath {
			continue
		}
		if !strings.HasPrefix(rec.Path, dir) {
			continue
		}
		if v, ok := rec.Frontmatter[field]; ok {
			values[fmt.Sprintf("%v", v)] = true
		}
	}
	return values
}

// formulaCoversValues checks that every descendant value appears as a quoted
// string in the aggregate expression. If any value is absent, the formula
// may not handle it correctly.
func formulaCoversValues(expr string, values map[string]bool) bool {
	for v := range values {
		// Check for the value in double quotes (as used in expr-lang expressions).
		if !strings.Contains(expr, `"`+v+`"`) {
			return false
		}
	}
	return true
}
