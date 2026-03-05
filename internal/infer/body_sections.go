package infer

import (
	"fmt"

	"github.com/pablontiv/rootline/internal/extract"
)

// DetectSectionPatterns analyzes heading structure across records in a directory.
// Headings present in ≥80% of records are "required_section".
// Headings present in <20% of records are "optional_section".
// Records without an AST (empty body) are excluded from the analysis.
func DetectSectionPatterns(records []*extract.Record) []Inference {
	// Filter to records that have a parsed AST.
	var valid []*extract.Record
	for _, rec := range records {
		if rec.AST != nil && rec.Body != "" {
			valid = append(valid, rec)
		}
	}

	if len(valid) == 0 {
		return nil
	}

	total := len(valid)

	// Count how many records contain each heading.
	headingCount := make(map[string]int)

	for _, rec := range valid {
		source := []byte(rec.Body)
		sections := extract.ExtractSections(rec.AST, source)
		seen := make(map[string]bool)
		for _, sec := range sections {
			if sec.Heading == "" {
				continue
			}
			if !seen[sec.Heading] {
				headingCount[sec.Heading]++
				seen[sec.Heading] = true
			}
		}
	}

	var inferences []Inference

	for heading, count := range headingCount {
		freq := float64(count) / float64(total)
		freqStr := fmt.Sprintf("%.2f", freq)

		switch {
		case freq >= 0.8:
			inferences = append(inferences, Inference{
				Type:    "required_section",
				Field:   heading,
				Value:   freqStr,
				Message: fmt.Sprintf("section %q appears in %d/%d records (%.0f%%) — required", heading, count, total, freq*100),
			})
		case freq < 0.2:
			inferences = append(inferences, Inference{
				Type:    "optional_section",
				Field:   heading,
				Value:   freqStr,
				Message: fmt.Sprintf("section %q appears in %d/%d records (%.0f%%) — optional", heading, count, total, freq*100),
			})
		}
	}

	return inferences
}
