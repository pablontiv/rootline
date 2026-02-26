package proposal

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

const (
	siblingInferThreshold = 0.6  // 60% agreement for missing fields
	outlierThreshold      = 0.75 // 75% agreement to flag outliers
	minSiblingsForInfer   = 2    // minimum siblings with majority value
	minSiblingsForOutlier = 3    // higher bar for corrections
)

// majorityValue returns the most frequent value in a slice of strings.
// Returns (value, count, total). If the slice is empty or there's a tie,
// returns ("", 0, 0) or ("", count, total) respectively.
func majorityValue(values []string) (string, int, int) {
	if len(values) == 0 {
		return "", 0, 0
	}

	counts := make(map[string]int)
	for _, v := range values {
		counts[v]++
	}

	best := ""
	bestCount := 0
	tied := false

	for v, c := range counts {
		if c > bestCount {
			best = v
			bestCount = c
			tied = false
		} else if c == bestCount {
			tied = true
		}
	}

	if tied {
		return "", bestCount, len(values)
	}
	return best, bestCount, len(values)
}

// detectInferFromSiblings finds records with missing required enum fields
// where the majority of siblings in the same directory have the same value.
func detectInferFromSiblings(records []*extract.Record, effective *rules.StemFile, errs map[string][]rules.ValidationError) []Proposal {
	if effective == nil {
		return nil
	}

	// Group records by directory, excluding README.md files.
	dirRecords := make(map[string][]*extract.Record)
	for _, rec := range records {
		if strings.HasSuffix(rec.Path, "README.md") {
			continue
		}
		dir := filepath.Dir(rec.Path)
		dirRecords[dir] = append(dirRecords[dir], rec)
	}

	// Build error lookup: path → set of fields with "required" errors.
	requiredErrs := make(map[string]map[string]bool)
	for path, pathErrs := range errs {
		for _, e := range pathErrs {
			if e.Rule == "required" {
				if requiredErrs[path] == nil {
					requiredErrs[path] = make(map[string]bool)
				}
				requiredErrs[path][e.Field] = true
			}
		}
	}

	var proposals []Proposal

	for _, siblings := range dirRecords {
		// For each enum field in schema, collect values from siblings.
		for fieldName, sf := range effective.Schema {
			if sf.Type != "enum" || len(sf.Values) == 0 {
				continue
			}

			var values []string
			for _, rec := range siblings {
				if v, ok := rec.Frontmatter[fieldName]; ok {
					if s, ok := v.(string); ok && s != "" {
						values = append(values, s)
					}
				}
			}

			majority, count, total := majorityValue(values)
			if majority == "" || total == 0 {
				continue
			}

			ratio := float64(count) / float64(total)
			if ratio < siblingInferThreshold || count < minSiblingsForInfer {
				continue
			}

			// Find siblings with required errors for this field.
			for _, rec := range siblings {
				if requiredErrs[rec.Path] != nil && requiredErrs[rec.Path][fieldName] {
					proposals = append(proposals, Proposal{
						Type:        InferFromSiblings,
						Field:       fieldName,
						Description: fmt.Sprintf("infer %q from %d siblings (%d/%d agree)", majority, total, count, total),
						Paths:       []string{rec.Path},
						Value:       majority,
					})
				}
			}
		}
	}

	return proposals
}

// detectOutlierValue finds records whose enum field value differs from the
// strong majority of siblings in the same directory.
func detectOutlierValue(records []*extract.Record, effective *rules.StemFile, errs map[string][]rules.ValidationError) []Proposal {
	if effective == nil {
		return nil
	}

	// Group records by directory, excluding README.md files.
	dirRecords := make(map[string][]*extract.Record)
	for _, rec := range records {
		if strings.HasSuffix(rec.Path, "README.md") {
			continue
		}
		dir := filepath.Dir(rec.Path)
		dirRecords[dir] = append(dirRecords[dir], rec)
	}

	// Build error lookup: path+field → has any validation error.
	hasError := make(map[string]bool)
	for path, pathErrs := range errs {
		for _, e := range pathErrs {
			hasError[path+"\x00"+e.Field] = true
		}
	}

	var proposals []Proposal

	for _, siblings := range dirRecords {
		for fieldName, sf := range effective.Schema {
			if sf.Type != "enum" || len(sf.Values) == 0 {
				continue
			}

			// Collect values only from siblings that have the field and no errors for it.
			var values []string
			type recValue struct {
				rec   *extract.Record
				value string
			}
			var candidates []recValue

			for _, rec := range siblings {
				v, ok := rec.Frontmatter[fieldName]
				if !ok {
					continue
				}
				s, ok := v.(string)
				if !ok || s == "" {
					continue
				}

				if hasError[rec.Path+"\x00"+fieldName] {
					continue // skip records with validation errors for this field
				}

				values = append(values, s)
				candidates = append(candidates, recValue{rec: rec, value: s})
			}

			majority, count, total := majorityValue(values)
			if majority == "" || total == 0 {
				continue
			}

			ratio := float64(count) / float64(total)
			if ratio < outlierThreshold || count < minSiblingsForOutlier {
				continue
			}

			// Flag candidates whose value differs from majority.
			for _, c := range candidates {
				if c.value != majority {
					proposals = append(proposals, Proposal{
						Type:        CorrectOutlier,
						Field:       fieldName,
						Description: fmt.Sprintf("value %q is an outlier among %d siblings (majority: %q, %d/%d)", c.value, total, majority, count, total),
						Paths:       []string{c.rec.Path},
						From:        c.value,
						To:          majority,
					})
				}
			}
		}
	}

	return proposals
}
