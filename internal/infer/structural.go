package infer

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
)

const (
	structuralMinSampleSize    = 3
	structuralIndexThreshold   = 0.90
	structuralChildPaddingDown = 0.8
	structuralChildPaddingUp   = 1.2
)

// DetectStructural analyzes directory structure under scanRoot.
// Returns []Inference with type "add_structural_rule".
func DetectStructural(scanRoot string) []Inference {
	entries, err := os.ReadDir(scanRoot)
	if err != nil {
		return nil
	}

	// Collect non-hidden subdirectories.
	var subdirs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if len(name) > 0 && name[0] == '.' {
			continue
		}
		subdirs = append(subdirs, name)
	}

	if len(subdirs) < structuralMinSampleSize {
		return nil
	}

	var result []Inference

	// Check README.md presence in each subdir.
	withIndex := 0
	for _, sd := range subdirs {
		indexPath := filepath.Join(scanRoot, sd, "README.md")
		if _, err := os.Stat(indexPath); err == nil {
			withIndex++
		}
	}

	ratio := float64(withIndex) / float64(len(subdirs))
	if ratio >= structuralIndexThreshold {
		result = append(result, Inference{
			Type:    "add_structural_rule",
			Field:   "require_index",
			Value:   "README.md",
			Message: fmt.Sprintf("README.md found in %d/%d subdirectories (%.0f%%)", withIndex, len(subdirs), ratio*100),
		})
	}

	// Check child dir counts for each subdir that has children.
	var childCounts []int
	for _, sd := range subdirs {
		subdirPath := filepath.Join(scanRoot, sd)
		subEntries, err := os.ReadDir(subdirPath)
		if err != nil {
			continue
		}

		count := 0
		for _, se := range subEntries {
			if se.IsDir() {
				name := se.Name()
				if len(name) > 0 && name[0] != '.' {
					count++
				}
			}
		}

		if count > 0 {
			childCounts = append(childCounts, count)
		}
	}

	if len(childCounts) >= structuralMinSampleSize {
		minCount := childCounts[0]
		maxCount := childCounts[0]
		for _, c := range childCounts[1:] {
			if c < minCount {
				minCount = c
			}
			if c > maxCount {
				maxCount = c
			}
		}

		minChildren := int(math.Floor(float64(minCount) * structuralChildPaddingDown))
		if minChildren < 1 {
			minChildren = 1
		}
		maxChildren := int(math.Ceil(float64(maxCount) * structuralChildPaddingUp))

		result = append(result, Inference{
			Type:    "add_structural_rule",
			Field:   "min_children",
			Value:   fmt.Sprintf("%d", minChildren),
			Message: fmt.Sprintf("min_children inferred from %d parents with children (min observed: %d, padded: %d)", len(childCounts), minCount, minChildren),
		})
		result = append(result, Inference{
			Type:    "add_structural_rule",
			Field:   "max_children",
			Value:   fmt.Sprintf("%d", maxChildren),
			Message: fmt.Sprintf("max_children inferred from %d parents with children (max observed: %d, padded: %d)", len(childCounts), maxCount, maxChildren),
		})
	}

	return result
}
