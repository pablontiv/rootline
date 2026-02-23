package migrate

import (
	"fmt"
	"sort"

	"github.com/pablontiv/rootline/internal/extract"
)

// Classify enriches a DiffResult with AffectedFiles counts by scanning
// records to determine how many files are impacted by each change.
// It also re-sorts changes so breaking changes appear first.
func Classify(result *DiffResult, records []*extract.Record) {
	for i := range result.Changes {
		result.Changes[i].AffectedFiles = countAffected(result.Changes[i], records)
	}

	// Sort: breaking first, then by field name, then by kind.
	sort.SliceStable(result.Changes, func(i, j int) bool {
		if result.Changes[i].Breaking != result.Changes[j].Breaking {
			return result.Changes[i].Breaking
		}
		if result.Changes[i].Field != result.Changes[j].Field {
			return result.Changes[i].Field < result.Changes[j].Field
		}
		return result.Changes[i].Kind < result.Changes[j].Kind
	})
}

// countAffected returns how many records are impacted by a given change.
func countAffected(c Change, records []*extract.Record) int {
	count := 0
	for _, r := range records {
		if isAffected(c, r) {
			count++
		}
	}
	return count
}

// isAffected checks whether a single record is affected by a change.
func isAffected(c Change, r *extract.Record) bool {
	switch c.Kind {
	case FieldRemoved:
		// Files that have this field are affected.
		_, exists := r.Frontmatter[c.Field]
		return exists

	case EnumValueRemoved:
		// Files using the removed enum value are affected.
		val, exists := r.Frontmatter[c.Field]
		if !exists {
			return false
		}
		return fmt.Sprintf("%v", val) == c.Before

	case TypeChanged:
		// Files that have this field are affected (their values may become invalid).
		_, exists := r.Frontmatter[c.Field]
		return exists

	case RequiredAdded:
		// Files WITHOUT this field are affected (they now violate the required constraint).
		_, exists := r.Frontmatter[c.Field]
		return !exists

	case RuleRemoved:
		// All files are potentially affected when a validation rule is removed.
		// Conservative: count all records.
		return true

	case RuleAdded:
		// New rules could affect all existing records.
		return true

	case FieldAdded, EnumValueAdded, RequiredRemoved, DefaultChanged, SeverityChanged:
		// Non-breaking changes: no files are negatively impacted.
		return false

	default:
		return false
	}
}
