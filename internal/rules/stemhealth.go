package rules

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Check 8: Aggregate formula coverage (formula references all enum values)
	quotedStringRe := regexp.MustCompile(`"([^"]*)"`)
	for sf, stem := range parsedStems {
		relPath, _ := filepath.Rel(absRoot, sf)
		// Build effective schema for this stem.
		dir := filepath.Dir(sf)
		entries, walkErr := WalkUp(dir)
		if walkErr != nil {
			continue
		}
		effective := MergeStemFiles(entries)
		if effective == nil {
			continue
		}

		for fieldName, exprAny := range stem.Aggregate {
			expr, ok := exprAny.(string)
			if !ok {
				continue
			}
			sf, exists := effective.Schema[fieldName]
			if !exists || sf.Type != "enum" || len(sf.Values) == 0 {
				continue
			}

			// Extract all quoted strings from the expression.
			matches := quotedStringRe.FindAllStringSubmatch(expr, -1)
			quotedValues := make(map[string]bool)
			for _, m := range matches {
				quotedValues[m[1]] = true
			}

			// Check each enum value is referenced.
			var missing []string
			for _, v := range sf.Values {
				if !quotedValues[v] {
					missing = append(missing, v)
				}
			}
			if len(missing) > 0 {
				checks = append(checks, StemHealthCheck{
					Name:    "aggregate-formula-coverage",
					Status:  "warn",
					Message: fmt.Sprintf("aggregate formula for %q does not reference enum value(s): %s", fieldName, strings.Join(missing, ", ")),
					Path:    relPath,
					Field:   fieldName,
				})
			}
		}
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Check 9: Domain type compatibility (core domain vs declared type)
	for sf, stem := range parsedStems {
		relPath, _ := filepath.Rel(absRoot, sf)
		for fieldName, field := range stem.Schema {
			if field.Domain == "" {
				continue
			}
			def := LookupDomain(field.Domain)
			if def == nil {
				// Custom domain (with "/") — require explicit type
				if IsCustomDomain(field.Domain) && field.Type == "" {
					checks = append(checks, StemHealthCheck{
						Name:    "domain-custom-no-type",
						Status:  "fail",
						Message: fmt.Sprintf("custom domain %q on field %q requires an explicit type", field.Domain, fieldName),
						Path:    relPath,
						Field:   fieldName,
					})
				}
				continue
			}
			// Core domain: check type compatibility (only if type was explicitly declared in this stem,
			// not inferred — we detect this by checking the raw content)
			if field.Type != "" && field.Type != def.BaseType {
				checks = append(checks, StemHealthCheck{
					Name:    "domain-type-compat",
					Status:  "warn",
					Message: fmt.Sprintf("field %q has domain %q (base type %q) but declared type %q", fieldName, field.Domain, def.BaseType, field.Type),
					Path:    relPath,
					Field:   fieldName,
				})
			}
			// Check required attrs
			for _, attr := range def.RequiredAttrs {
				if !fieldHasAttr(field, attr) {
					checks = append(checks, StemHealthCheck{
						Name:    "domain-missing-attrs",
						Status:  "warn",
						Message: fmt.Sprintf("field %q with domain %q is missing required attribute %q", fieldName, field.Domain, attr),
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

	// Check 10: Domain uniqueness within effective scope
	for sf, stem := range parsedStems {
		relPath, _ := filepath.Rel(absRoot, sf)
		// Collect fields with domains in this stem
		byDomain := make(map[string][]domainEntry)
		for fieldName, field := range stem.Schema {
			if field.Domain != "" {
				byDomain[field.Domain] = append(byDomain[field.Domain], domainEntry{fieldName, field})
			}
		}
		for domain, entries := range byDomain {
			if len(entries) < 2 {
				continue
			}
			// Check if match patterns overlap
			if matchPatternsOverlap(entries) {
				names := make([]string, len(entries))
				for i, e := range entries {
					names[i] = e.fieldName
				}
				checks = append(checks, StemHealthCheck{
					Name:    "domain-duplicate-scope",
					Status:  "fail",
					Message: fmt.Sprintf("domain %q assigned to multiple fields with overlapping scope: %s", domain, strings.Join(names, ", ")),
					Path:    relPath,
				})
			}
		}
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	return &StemHealthResult{Checks: checks}, nil
}

// StemHealthToResults converts stem-health checks into ValidationResults.
func StemHealthToResults(result *StemHealthResult) []*ValidationResult {
	var results []*ValidationResult
	for _, c := range result.Checks {
		if c.Status == "pass" {
			continue
		}
		severity := "warn"
		if c.Status == "fail" {
			severity = "error"
		}
		path := c.Path
		if path == "" {
			path = ".stem"
		}
		errs := []ValidationError{{
			Rule:     c.Name,
			Field:    c.Field,
			Message:  c.Message,
			Source:   "stem-health",
			Severity: severity,
		}}
		results = append(results, NewValidationResult(path, errs))
	}
	return results
}

// fieldHasAttr checks if a SchemaField has a given attribute set.
func fieldHasAttr(f SchemaField, attr string) bool {
	switch attr {
	case "values":
		return len(f.Values) > 0
	case "prefix":
		return f.Prefix != ""
	case "digits":
		return f.Digits > 0
	default:
		return false
	}
}

// domainEntry pairs a field name with its SchemaField for domain checks.
type domainEntry struct {
	fieldName string
	field     SchemaField
}

// matchPatternsOverlap checks if any entries have overlapping match patterns.
// Two fields overlap if: both have no match (global), or both share the same
// match pattern(s). Conservative: distinct patterns are treated as non-overlapping.
func matchPatternsOverlap(entries []domainEntry) bool {
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			a, b := entries[i].field, entries[j].field
			if a.Match == nil && b.Match == nil {
				return true // both global
			}
			if a.Match == nil || b.Match == nil {
				return true // one global, one scoped — global overlaps everything
			}
			// Both have match patterns — check for shared patterns
			for _, pa := range a.Match.Patterns {
				for _, pb := range b.Match.Patterns {
					if pa == pb {
						return true
					}
				}
			}
		}
	}
	return false
}
