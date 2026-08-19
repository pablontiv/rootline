package infer

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
)

// DetectSectionPatterns analyzes heading structure across records in a directory.
// Every record contributes to the denominator; empty bodies count as records where
// headings are absent. Headings present in at least threshold records are inferred;
// universal headings are required, while non-universal threshold matches are optional.
func DetectSectionPatterns(records []*extract.Record, threshold float64) ([]Inference, error) {
	if math.IsNaN(threshold) || math.IsInf(threshold, 0) || threshold < 0 || threshold > 1 {
		return nil, fmt.Errorf("section threshold must be finite and in [0,1], got %v", threshold)
	}
	if len(records) == 0 {
		return nil, nil
	}

	total := len(records)
	headingCount := make(map[string]int)
	headingExact := make(map[string]string)

	for _, rec := range records {
		if rec == nil || rec.Body == "" {
			continue
		}
		seen := make(map[string]bool)
		for _, sec := range sectionsForInference(rec) {
			if sec.Level <= 0 || sec.Heading == "" {
				continue
			}
			exact := strings.Repeat("#", sec.Level) + " " + sec.Heading
			source, err := extract.CanonicalSectionSource(exact)
			if err != nil {
				return nil, err
			}
			if seen[source] {
				continue
			}
			headingCount[source]++
			headingExact[source] = exact
			seen[source] = true
		}
	}

	allHeadings := make([]sectionCandidate, 0, len(headingCount))
	for source, count := range headingCount {
		freq := float64(count) / float64(total)
		exact := headingExact[source]
		field := sectionFieldName(strings.TrimSpace(exact[strings.IndexByte(exact, ' ')+1:]))
		candidateType := "optional_section"
		if count == total {
			candidateType = "required_section"
		}
		allHeadings = append(allHeadings, sectionCandidate{
			field:    field,
			exact:    exact,
			source:   source,
			count:    count,
			freq:     freq,
			typeName: candidateType,
		})
	}

	sort.Slice(allHeadings, func(i, j int) bool {
		if allHeadings[i].field != allHeadings[j].field {
			return allHeadings[i].field < allHeadings[j].field
		}
		return allHeadings[i].source < allHeadings[j].source
	})

	if err := rejectSectionFieldCollisions(allHeadings); err != nil {
		return nil, err
	}

	candidates := make([]sectionCandidate, 0, len(allHeadings))
	for _, candidate := range allHeadings {
		if candidate.freq >= threshold {
			candidates = append(candidates, candidate)
		}
	}

	inferences := make([]Inference, 0, len(candidates))
	for _, candidate := range candidates {
		freqStr := fmt.Sprintf("%.2f", candidate.freq)
		inferences = append(inferences, Inference{
			Type:            candidate.typeName,
			Field:           candidate.field,
			Value:           freqStr,
			Message:         fmt.Sprintf("section %q appears in %d/%d records (%.0f%%) — %s", candidate.exact, candidate.count, total, candidate.freq*100, strings.TrimSuffix(candidate.typeName, "_section")),
			SourceDirective: candidate.source,
		})
	}

	return inferences, nil
}

type sectionCandidate struct {
	field    string
	exact    string
	source   string
	count    int
	freq     float64
	typeName string
}

func sectionsForInference(rec *extract.Record) []extract.Section {
	if len(rec.BodySections) > 0 {
		return rec.BodySections
	}
	if rec.AST != nil {
		return extract.ExtractSections(rec.AST, []byte(rec.Body))
	}
	return extract.ExtractSectionsFromText(rec.Body)
}

func rejectSectionFieldCollisions(candidates []sectionCandidate) error {
	var collisions []string
	for i := 0; i < len(candidates); {
		j := i + 1
		for j < len(candidates) && candidates[j].field == candidates[i].field {
			j++
		}
		if j-i > 1 {
			headings := make([]string, 0, j-i)
			for _, candidate := range candidates[i:j] {
				headings = append(headings, candidate.exact)
			}
			collisions = append(collisions, fmt.Sprintf("%s: %s", candidates[i].field, strings.Join(headings, ", ")))
		}
		i = j
	}
	if len(collisions) > 0 {
		return fmt.Errorf("section field name collision: %s", strings.Join(collisions, "; "))
	}
	return nil
}
