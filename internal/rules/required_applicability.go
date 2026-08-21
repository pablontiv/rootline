package rules

import (
	"path/filepath"

	"github.com/pablontiv/rootline/internal/extract"
)

func requiredCheckApplies(record *extract.Record, effective *StemFile, name string, field SchemaField) bool {
	if field.Severity == "off" {
		return false
	}
	recordPath := ""
	if record != nil {
		recordPath = record.Path
	}
	if effective != nil {
		if _, hasAggregate := effective.Aggregate[name]; hasAggregate && IsIndexFile(recordPath, effective) {
			return false
		}
	}
	if field.Excludes != nil && field.Excludes.Match != "" {
		if matched, _ := filepath.Match(field.Excludes.Match, recordPath); matched {
			return false
		}
	}
	return true
}
