// Package doctor provides diagnostic checks for .stem configuration health.
package doctor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pablontiv/rootline/internal/rules"
)

// Result is the versioned output for doctor.
type Result struct {
	Version int     `json:"version"`
	Kind    string  `json:"kind"`
	Checks  []Check `json:"checks"`
	Summary Summary `json:"summary"`
}

// Check represents a single diagnostic check result.
type Check struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "pass", "fail", "warn"
	Message string `json:"message,omitempty"`
	Path    string `json:"path,omitempty"`
}

// Summary holds aggregate counts.
type Summary struct {
	Total    int `json:"total"`
	Passed   int `json:"passed"`
	Warnings int `json:"warnings"`
	Failed   int `json:"failed"`
}

// RunChecks executes all 7 diagnostic checks against .stem files under absRoot.
func RunChecks(ctx context.Context, absRoot string) (*Result, error) {
	var checks []Check

	// Find all .stem files
	var stemFiles []string
	err := filepath.Walk(absRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if info.IsDir() {
			if info.Name() == ".git" || info.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Name() == ".stem" {
			stemFiles = append(stemFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking %s: %w", absRoot, err)
	}

	if len(stemFiles) == 0 {
		checks = append(checks, Check{
			Name:    "stem-files-exist",
			Status:  "warn",
			Message: "no .stem files found",
		})
	}

	// Check 1: Parse validity
	parsedStems := make(map[string]*rules.StemFile)
	for _, sf := range stemFiles {
		relPath, _ := filepath.Rel(absRoot, sf)
		stem, parseErr := rules.ParseStemFile(sf)
		if parseErr != nil {
			checks = append(checks, Check{
				Name:    "yaml-valid",
				Status:  "fail",
				Message: fmt.Sprintf("invalid YAML: %v", parseErr),
				Path:    relPath,
			})
		} else {
			checks = append(checks, Check{
				Name:   "yaml-valid",
				Status: "pass",
				Path:   relPath,
			})
			parsedStems[sf] = stem
		}
	}

	// Check for context cancellation between checks.
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Check 2: Orphan scopes (scope.match doesn't match any file in directory)
	for sf, stem := range parsedStems {
		if stem.Scope.Match == "" {
			continue
		}
		relPath, _ := filepath.Rel(absRoot, sf)
		dir := filepath.Dir(sf)
		entries, readErr := os.ReadDir(dir)
		if readErr != nil {
			continue
		}
		hasMatch := false
		for _, e := range entries {
			if e.IsDir() || e.Name() == ".stem" {
				continue
			}
			matched, _ := filepath.Match(stem.Scope.Match, e.Name())
			if matched {
				hasMatch = true
				break
			}
		}
		if !hasMatch {
			checks = append(checks, Check{
				Name:    "scope-match",
				Status:  "warn",
				Message: fmt.Sprintf("scope.match %q matches no files in directory", stem.Scope.Match),
				Path:    relPath,
			})
		}
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Check 3: Inheritance consistency (child doesn't change type of inherited field)
	for sf, stem := range parsedStems {
		relPath, _ := filepath.Rel(absRoot, sf)
		dir := filepath.Dir(sf)
		parentEntries, walkErr := rules.WalkUp(dir)
		if walkErr != nil || len(parentEntries) < 2 {
			continue
		}
		// Merge parent entries (excluding last = self)
		parentMerged := rules.MergeStemFiles(parentEntries[:len(parentEntries)-1])
		if parentMerged == nil {
			continue
		}
		for fieldName, childField := range stem.Schema {
			if parentField, exists := parentMerged.Schema[fieldName]; exists {
				if childField.Type != "" && parentField.Type != "" && childField.Type != parentField.Type {
					checks = append(checks, Check{
						Name:    "type-consistency",
						Status:  "fail",
						Message: fmt.Sprintf("field %q changes type from %q to %q (inherited from parent)", fieldName, parentField.Type, childField.Type),
						Path:    relPath,
					})
				}
			}
		}
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Check 4: Enum fields have at least 2 values
	for sf, stem := range parsedStems {
		relPath, _ := filepath.Rel(absRoot, sf)
		for fieldName, field := range stem.Schema {
			if field.Type == "enum" && len(field.Values) < 2 {
				checks = append(checks, Check{
					Name:    "enum-values",
					Status:  "warn",
					Message: fmt.Sprintf("enum field %q has %d value(s), expected at least 2", fieldName, len(field.Values)),
					Path:    relPath,
				})
			}
		}
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Check 5: Validate rules reference existing schema fields
	for sf, stem := range parsedStems {
		relPath, _ := filepath.Rel(absRoot, sf)
		// Get effective schema (merged with parents)
		dir := filepath.Dir(sf)
		entries, walkErr := rules.WalkUp(dir)
		if walkErr != nil {
			continue
		}
		effective := rules.MergeStemFiles(entries)
		if effective == nil {
			continue
		}
		for _, rule := range stem.Validate {
			if rule.Field != "" {
				if _, exists := effective.Schema[rule.Field]; !exists {
					checks = append(checks, Check{
						Name:    "rule-field-exists",
						Status:  "warn",
						Message: fmt.Sprintf("validation rule references field %q not in schema", rule.Field),
						Path:    relPath,
					})
				}
			}
		}
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Check 6: Child redefines parent field (informative warning)
	for sf, stem := range parsedStems {
		relPath, _ := filepath.Rel(absRoot, sf)
		dir := filepath.Dir(sf)
		parentEntries, walkErr := rules.WalkUp(dir)
		if walkErr != nil || len(parentEntries) < 2 {
			continue
		}
		parentMerged := rules.MergeStemFiles(parentEntries[:len(parentEntries)-1])
		if parentMerged == nil {
			continue
		}
		for fieldName := range stem.Schema {
			if _, exists := parentMerged.Schema[fieldName]; exists {
				checks = append(checks, Check{
					Name:    "field-override",
					Status:  "warn",
					Message: fmt.Sprintf("field %q overrides parent definition", fieldName),
					Path:    relPath,
				})
			}
		}
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Check 7: Aggregated required fields (required + aggregate on same field)
	for sf, stem := range parsedStems {
		relPath, _ := filepath.Rel(absRoot, sf)
		for fieldName, field := range stem.Schema {
			if !field.Required {
				continue
			}
			if _, hasAggregate := stem.Aggregate[fieldName]; hasAggregate {
				checks = append(checks, Check{
					Name:   "aggregated-required",
					Status: "warn",
					Message: fmt.Sprintf(
						"field %q is required but also has an aggregate expression; required is auto-skipped on index files — consider removing required or using excludes",
						fieldName,
					),
					Path: relPath,
				})
			}
		}
	}

	// Build summary
	summary := Summary{Total: len(checks)}
	for _, c := range checks {
		switch c.Status {
		case "pass":
			summary.Passed++
		case "warn":
			summary.Warnings++
		case "fail":
			summary.Failed++
		}
	}

	result := &Result{
		Version: 1,
		Kind:    "rootline/doctor",
		Checks:  checks,
		Summary: summary,
	}

	return result, nil
}
