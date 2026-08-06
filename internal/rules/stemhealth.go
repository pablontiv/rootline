package rules

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/pablontiv/picokit/fuzzy"
)

// Severity levels shared by validation errors, stem-health diagnostics and
// run-level notices, so one vocabulary describes all three.
const (
	SeverityError = "error"
	SeverityWarn  = "warn"
	SeverityInfo  = "info"
)

// StemHealthCheck represents a single stem-health diagnostic result.
type StemHealthCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "pass", "fail", "warn", "info"
	Message string `json:"message,omitempty"`
	Path    string `json:"path,omitempty"` // relative to absRoot
	Field   string `json:"field,omitempty"`
}

// StemHealthResult holds all stem-health diagnostic checks.
type StemHealthResult struct {
	Checks []StemHealthCheck
}

// StemHealthDiagnostic is a stem-health finding as it appears in the validate
// envelope: a `.stem` file, not a record, so it is reported on its own axis.
type StemHealthDiagnostic struct {
	Path     string `json:"path"`
	Check    string `json:"check"`
	Field    string `json:"field,omitempty"`
	Severity string `json:"severity"` // "error", "warn" or "info"
	Message  string `json:"message,omitempty"`
}

// StemHealthDiagnostics converts raw checks into reportable diagnostics,
// dropping the passing ones.
//
// The status→severity mapping is total: "fail" is an error, "info" stays info
// (a nested root marker is a supported configuration, so it must not fail
// --strict), and anything else is a warning. An earlier mapper handled only
// "pass" and "fail", which silently promoted "info" to a warning.
func StemHealthDiagnostics(result *StemHealthResult) []StemHealthDiagnostic {
	if result == nil {
		return nil
	}
	var diags []StemHealthDiagnostic
	for _, c := range result.Checks {
		if c.Status == "pass" {
			continue
		}
		severity := SeverityWarn
		switch c.Status {
		case "fail":
			severity = SeverityError
		case SeverityInfo:
			severity = SeverityInfo
		}
		path := c.Path
		if path == "" {
			// stem-files-exist fires when no .stem exists anywhere, so it has
			// no path of its own to report.
			path = stemFileName
		}
		diags = append(diags, StemHealthDiagnostic{
			Path:     path,
			Check:    c.Name,
			Field:    c.Field,
			Severity: severity,
			Message:  c.Message,
		})
	}
	return diags
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

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Check 11: Monotonic constraint violations
	for sf := range parsedStems {
		relPath, _ := filepath.Rel(absRoot, sf)
		dir := filepath.Dir(sf)

		// Resolve layered constraints with monotonic=true for this directory
		lr, resolveErr := ResolveLayered(dir, absRoot, true)
		if resolveErr != nil {
			continue
		}

		// For each conflict, emit a diagnostic error
		for _, conflict := range lr.Conflicts {
			fieldName, msg := monotonicViolation(conflict)

			checks = append(checks, StemHealthCheck{
				Name:    "monotonic-violations",
				Status:  "fail",
				Message: msg,
				Path:    relPath,
				Field:   fieldName,
			})
		}
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Check 11: unknown keys under links.checks (silently inert otherwise)
	for sf, stem := range parsedStems {
		relPath, _ := filepath.Rel(absRoot, sf)
		for _, key := range stem.Links.UnknownCheckKeys {
			msg := fmt.Sprintf("unknown key %q in links.checks", key)
			if match := fuzzy.Match(key, knownCheckKeys); match != "" {
				msg += fmt.Sprintf(" (did you mean %q?)", match)
			}
			checks = append(checks, StemHealthCheck{
				Name:    "unknown-check-keys",
				Status:  "warn",
				Message: msg,
				Path:    relPath,
				Field:   key,
			})
		}
	}

	// Check 12: nested-root-marker (INFO level)
	// Detect when a directory declares root: true and has an ancestor also with root: true
	for sf, stem := range parsedStems {
		if !stem.Root {
			continue // Only check stems with root: true
		}

		relPath, _ := filepath.Rel(absRoot, sf)
		dir := filepath.Dir(sf)

		// Manually walk up the directory tree (without stopping at markers) to find
		// ancestor directories that contain .stem files with root: true
		current := dir
		foundAncestorRoot := false
		var ancestorRootPath string

		for {
			parent := filepath.Dir(current)
			if parent == current {
				// Reached filesystem root
				break
			}

			// Check if parent contains a .stem with root: true
			parentStemPath := filepath.Join(parent, stemFileName)
			if parentStem, exists := parsedStems[parentStemPath]; exists && parentStem.Root {
				foundAncestorRoot = true
				ancestorRootPath = parentStemPath
				break
			}

			current = parent
		}

		if foundAncestorRoot {
			ancestorRelPath, _ := filepath.Rel(absRoot, ancestorRootPath)
			checks = append(checks, StemHealthCheck{
				Name:    "nested-root-marker",
				Status:  "info",
				Message: fmt.Sprintf("%s declares root: true, so records below it do not inherit %s", relPath, ancestorRelPath),
				Path:    relPath,
			})
		}
	}

	return &StemHealthResult{Checks: checks}, nil
}

// monotonicConstraintSuffixes are the schema-level constraints the resolver
// appends to a field name when it records a conflict. Anything else — a
// structural path like "structural.subdirs.min_children" — is reported whole.
var monotonicConstraintSuffixes = []string{"type", "required", "severity", "values"}

// monotonicField splits a conflict path into the name a reader recognises and
// the constraint that was loosened.
//
// "estado.required" answers ("estado", "required"): the reader looks for the
// field they wrote. "structural.subdirs.min_children" answers itself with an
// empty constraint, because truncating it at the first dot rendered both
// subdir bounds as the single field "structural" and made them
// indistinguishable in the report.
func monotonicField(path string) (field, constraint string) {
	dotIdx := strings.LastIndex(path, ".")
	if dotIdx <= 0 {
		return path, ""
	}
	suffix := path[dotIdx+1:]
	if slices.Contains(monotonicConstraintSuffixes, suffix) {
		return path[:dotIdx], suffix
	}
	return path, ""
}

// monotonicViolation renders a resolver conflict as a field name and a message
// that names the category it belongs to.
//
// The resolver reports five distinct loosenings — type widening, required
// loosening, severity loosening, enum extension and structural loosening — but
// four of them share Operation "conflict". Discriminating on Operation alone
// labelled all four "type change"; the constraint suffix is what tells them
// apart.
func monotonicViolation(conflict LayerConstraint) (field, message string) {
	field, constraint := monotonicField(conflict.Field)

	if conflict.Operation == "extension" {
		return field, fmt.Sprintf("field %q: enum extended with disallowed value(s): %v", field, conflict.Value)
	}
	if conflict.Operation != "conflict" {
		return field, fmt.Sprintf("field %q: %s violation at %s", field, conflict.Operation, conflict.Field)
	}

	switch constraint {
	case "type":
		return field, fmt.Sprintf("field %q widens type: %v", field, conflict.Value)
	case "required":
		return field, fmt.Sprintf("field %q loosens required: %v", field, conflict.Value)
	case "severity":
		return field, fmt.Sprintf("field %q loosens severity: %v", field, conflict.Value)
	case "values":
		return field, fmt.Sprintf("field %q narrows enum incompatibly: %v", field, conflict.Value)
	}
	return field, fmt.Sprintf("constraint %q loosens the parent constraint: %v", field, conflict.Value)
}
