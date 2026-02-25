package rules

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// StemHealthCheck represents a single stem-health diagnostic result.
type StemHealthCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "pass", "fail", "warn"
	Message string `json:"message,omitempty"`
	Path    string `json:"path,omitempty"` // relative to absRoot
	Field   string `json:"field,omitempty"`
}

// StemHealthResult holds all stem-health diagnostic checks.
type StemHealthResult struct {
	Checks []StemHealthCheck
}

// ValidateStemHealth runs all stem-health diagnostic checks against .stem files
// under absRoot and returns the results.
func ValidateStemHealth(ctx context.Context, absRoot string) (*StemHealthResult, error) {
	var checks []StemHealthCheck

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
		if info.Name() == stemFileName {
			stemFiles = append(stemFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking %s: %w", absRoot, err)
	}

	if len(stemFiles) == 0 {
		checks = append(checks, StemHealthCheck{
			Name:    "stem-files-exist",
			Status:  "warn",
			Message: "no .stem files found",
		})
	}

	// Check 1: Parse validity
	parsedStems := make(map[string]*StemFile)
	for _, sf := range stemFiles {
		relPath, _ := filepath.Rel(absRoot, sf)
		stem, parseErr := ParseStemFile(sf)
		if parseErr != nil {
			checks = append(checks, StemHealthCheck{
				Name:    "yaml-valid",
				Status:  "fail",
				Message: fmt.Sprintf("invalid YAML: %v", parseErr),
				Path:    relPath,
			})
		} else {
			checks = append(checks, StemHealthCheck{
				Name:   "yaml-valid",
				Status: "pass",
				Path:   relPath,
			})
			parsedStems[sf] = stem
		}
	}

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
			if e.IsDir() || e.Name() == stemFileName {
				continue
			}
			matched, _ := filepath.Match(stem.Scope.Match, e.Name())
			if matched {
				hasMatch = true
				break
			}
		}
		if !hasMatch {
			checks = append(checks, StemHealthCheck{
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
		parentEntries, walkErr := WalkUp(dir)
		if walkErr != nil || len(parentEntries) < 2 {
			continue
		}
		parentMerged := MergeStemFiles(parentEntries[:len(parentEntries)-1])
		if parentMerged == nil {
			continue
		}
		for fieldName, childField := range stem.Schema {
			if parentField, exists := parentMerged.Schema[fieldName]; exists {
				if childField.Type != "" && parentField.Type != "" && childField.Type != parentField.Type {
					checks = append(checks, StemHealthCheck{
						Name:    "type-consistency",
						Status:  "fail",
						Message: fmt.Sprintf("field %q changes type from %q to %q (inherited from parent)", fieldName, parentField.Type, childField.Type),
						Path:    relPath,
						Field:   fieldName,
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
				checks = append(checks, StemHealthCheck{
					Name:    "enum-values",
					Status:  "warn",
					Message: fmt.Sprintf("enum field %q has %d value(s), expected at least 2", fieldName, len(field.Values)),
					Path:    relPath,
					Field:   fieldName,
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
		dir := filepath.Dir(sf)
		entries, walkErr := WalkUp(dir)
		if walkErr != nil {
			continue
		}
		effective := MergeStemFiles(entries)
		if effective == nil {
			continue
		}
		for _, rule := range stem.Validate {
			if rule.Field != "" {
				if _, exists := effective.Schema[rule.Field]; !exists {
					checks = append(checks, StemHealthCheck{
						Name:    "rule-field-exists",
						Status:  "warn",
						Message: fmt.Sprintf("validation rule references field %q not in schema", rule.Field),
						Path:    relPath,
						Field:   rule.Field,
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
		parentEntries, walkErr := WalkUp(dir)
		if walkErr != nil || len(parentEntries) < 2 {
			continue
		}
		parentMerged := MergeStemFiles(parentEntries[:len(parentEntries)-1])
		if parentMerged == nil {
			continue
		}
		for fieldName := range stem.Schema {
			if _, exists := parentMerged.Schema[fieldName]; exists {
				checks = append(checks, StemHealthCheck{
					Name:    "field-override",
					Status:  "warn",
					Message: fmt.Sprintf("field %q overrides parent definition", fieldName),
					Path:    relPath,
					Field:   fieldName,
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
				checks = append(checks, StemHealthCheck{
					Name:   "aggregated-required",
					Status: "warn",
					Message: fmt.Sprintf(
						"field %q is required but also has an aggregate expression; required is auto-skipped on index files — consider removing required or using excludes",
						fieldName,
					),
					Path:  relPath,
					Field: fieldName,
				})
			}
		}
	}

	// Check 8: Levels children reference valid level names
	for sf, stem := range parsedStems {
		if len(stem.Levels) == 0 {
			continue
		}
		relPath, _ := filepath.Rel(absRoot, sf)
		for levelName, level := range stem.Levels {
			for _, child := range level.Children {
				if _, exists := stem.Levels[child]; !exists {
					checks = append(checks, StemHealthCheck{
						Name:    "levels-children-valid",
						Status:  "fail",
						Message: fmt.Sprintf("level %q references child %q which does not exist in levels", levelName, child),
						Path:    relPath,
						Field:   "levels." + levelName + ".children",
					})
				}
			}
		}
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Check 9: Levels children graph has no cycles
	for sf, stem := range parsedStems {
		if len(stem.Levels) == 0 {
			continue
		}
		relPath, _ := filepath.Rel(absRoot, sf)
		if cycle := detectLevelsCycle(stem.Levels); cycle != "" {
			checks = append(checks, StemHealthCheck{
				Name:    "levels-no-cycles",
				Status:  "fail",
				Message: fmt.Sprintf("cycle detected in levels children: %s", cycle),
				Path:    relPath,
				Field:   "levels",
			})
		}
	}

	return &StemHealthResult{Checks: checks}, nil
}

// detectLevelsCycle checks for cycles in the levels children graph using DFS.
// Returns a string describing the cycle path, or "" if no cycle exists.
func detectLevelsCycle(levels map[string]*HierarchyLevel) string {
	const (
		white = 0 // unvisited
		gray  = 1 // in current path
		black = 2 // fully processed
	)

	color := make(map[string]int)
	parent := make(map[string]string)

	var dfs func(node string) string
	dfs = func(node string) string {
		color[node] = gray
		level, exists := levels[node]
		if !exists {
			color[node] = black
			return ""
		}
		for _, child := range level.Children {
			if _, exists := levels[child]; !exists {
				continue // skip non-existent (handled by check 8)
			}
			if color[child] == gray {
				// Build cycle path
				path := child + " -> " + node
				cur := node
				for cur != child {
					cur = parent[cur]
					path = cur + " -> " + path
				}
				return path
			}
			if color[child] == white {
				parent[child] = node
				if cycle := dfs(child); cycle != "" {
					return cycle
				}
			}
		}
		color[node] = black
		return ""
	}

	for name := range levels {
		if color[name] == white {
			if cycle := dfs(name); cycle != "" {
				return cycle
			}
		}
	}
	return ""
}
