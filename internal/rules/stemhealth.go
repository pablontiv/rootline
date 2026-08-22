package rules

import (
	"context"
	"fmt"
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
	state, err := DiscoverStemState(ctx, absRoot)
	if err != nil {
		return nil, err
	}
	return EvaluateStemState(ctx, state)
}

// EvaluateStemState runs all stem-health diagnostic checks against an explicit
// StemState snapshot. It does not read from the filesystem; discovery, parse
// errors, matching files and ancestor chains are all supplied by state.
func EvaluateStemState(ctx context.Context, state *StemState) (*StemHealthResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if state == nil {
		return nil, fmt.Errorf("evaluating stem health: nil stem state")
	}

	absRoot := filepath.Clean(state.Root)
	var checks []StemHealthCheck

	stemPaths := state.EvaluatedStemPaths()
	if len(stemPaths) == 0 {
		checks = append(checks, StemHealthCheck{
			Name:    "stem-files-exist",
			Status:  "warn",
			Message: "no .stem files found",
		})
	}

	// Check 1: Parse validity
	parsedStems := make(map[string]*StemFile)
	for _, sf := range stemPaths {
		relPath := stemHealthRelPath(absRoot, sf)
		if parseErr := state.ParseErrors[sf]; parseErr != nil {
			checks = append(checks, StemHealthCheck{
				Name:   "yaml-valid",
				Status: "fail",
				// Not always a YAML syntax error: an unsupported version and a
				// null schema field are refused by the same read. The wrapped
				// error names the file and the reason, so asserting "invalid
				// YAML" here would misdescribe both.
				Message: parseErr.Error(),
				Path:    relPath,
			})
			continue
		}
		stem := state.Stems[sf]
		if stem == nil {
			continue
		}
		checks = append(checks, StemHealthCheck{
			Name:   "yaml-valid",
			Status: "pass",
			Path:   relPath,
		})
		parsedStems[sf] = stem
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Check 2: Orphan scopes (scope.match doesn't match any file in directory)
	for _, sf := range sortedStemPaths(parsedStems) {
		stem := parsedStems[sf]
		if stem.Scope.Match == "" {
			continue
		}
		hasMatch := false
		for _, match := range state.MatchingFiles(filepath.Dir(sf), stem.Scope.Match) {
			if filepath.Base(match) == stemFileName {
				continue
			}
			hasMatch = true
			break
		}
		if !hasMatch {
			checks = append(checks, StemHealthCheck{
				Name:    "scope-match",
				Status:  "warn",
				Message: fmt.Sprintf("scope.match %q matches no files in directory", stem.Scope.Match),
				Path:    stemHealthRelPath(absRoot, sf),
			})
		}
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Check 3: Inheritance type consistency through the shared compatibility algebra.
	for _, sf := range sortedStemPaths(parsedStems) {
		stem := parsedStems[sf]
		parentEntries := entriesBeforeStem(state.Chain(filepath.Dir(sf)), sf)
		if len(parentEntries) == 0 {
			continue
		}
		parentMerged := MergeStemFiles(parentEntries)
		if parentMerged == nil {
			continue
		}
		for _, fieldName := range sortedSchemaFieldNames(stem.Schema) {
			childField := stem.Schema[fieldName]
			parentField, exists := parentMerged.Schema[fieldName]
			if !exists {
				continue
			}
			for _, issue := range CheckFieldCompatibility(parentField, childField) {
				if issue.Constraint != "type" {
					continue
				}
				checks = append(checks, StemHealthCheck{
					Name:    "type-consistency",
					Status:  "fail",
					Message: issue.Message,
					Path:    stemHealthRelPath(absRoot, sf),
					Field:   fieldName,
				})
			}
		}
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Check 4: Schema field declarations use the canonical type/source contract.
	checks = append(checks, fieldDeclarationChecks(absRoot, parsedStems)...)

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Check 5: Validate rules reference existing schema fields
	for _, sf := range sortedStemPaths(parsedStems) {
		stem := parsedStems[sf]
		effective := MergeStemFiles(state.Chain(filepath.Dir(sf)))
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
						Path:    stemHealthRelPath(absRoot, sf),
						Field:   rule.Field,
					})
				}
			}
		}
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Check 6: Child redefines parent field (informative warning). Valid
	// type/value narrowings are accepted without generic override noise.
	for _, sf := range sortedStemPaths(parsedStems) {
		stem := parsedStems[sf]
		parentEntries := entriesBeforeStem(state.Chain(filepath.Dir(sf)), sf)
		if len(parentEntries) == 0 {
			continue
		}
		parentMerged := MergeStemFiles(parentEntries)
		if parentMerged == nil {
			continue
		}
		for _, fieldName := range sortedSchemaFieldNames(stem.Schema) {
			childField := stem.Schema[fieldName]
			parentField, exists := parentMerged.Schema[fieldName]
			if !exists || validNarrowingOverride(parentField, childField) || len(CheckFieldCompatibility(parentField, childField)) > 0 {
				continue
			}
			checks = append(checks, StemHealthCheck{
				Name:    "field-override",
				Status:  "warn",
				Message: fmt.Sprintf("field %q overrides parent definition", fieldName),
				Path:    stemHealthRelPath(absRoot, sf),
				Field:   fieldName,
			})
		}
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Check 7: Aggregated required fields (required + aggregate on same field)
	for _, sf := range sortedStemPaths(parsedStems) {
		stem := parsedStems[sf]
		for _, fieldName := range sortedSchemaFieldNames(stem.Schema) {
			field := stem.Schema[fieldName]
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
					Path:  stemHealthRelPath(absRoot, sf),
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
	for _, sf := range sortedStemPaths(parsedStems) {
		stem := parsedStems[sf]
		effective := MergeStemFiles(state.Chain(filepath.Dir(sf)))
		if effective == nil {
			continue
		}

		for _, fieldName := range sortedAggregateFieldNames(stem.Aggregate) {
			exprAny := stem.Aggregate[fieldName]
			expr, ok := exprAny.(string)
			if !ok {
				continue
			}
			schemaField, exists := effective.Schema[fieldName]
			if !exists || schemaField.Type != "enum" || len(schemaField.Values) == 0 {
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
			for _, v := range schemaField.Values {
				if !quotedValues[v] {
					missing = append(missing, v)
				}
			}
			if len(missing) > 0 {
				checks = append(checks, StemHealthCheck{
					Name:    "aggregate-formula-coverage",
					Status:  "warn",
					Message: fmt.Sprintf("aggregate formula for %q does not reference enum value(s): %s", fieldName, strings.Join(missing, ", ")),
					Path:    stemHealthRelPath(absRoot, sf),
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
	emittedMonotonic := make(map[layerConflictIdentity]struct{})
	for _, sf := range sortedStemPaths(parsedStems) {
		lr, resolveErr := resolveLayeredFromEntries(filepath.Dir(sf), state.Chain(filepath.Dir(sf)))
		if resolveErr != nil {
			continue
		}

		// For each conflict, emit a diagnostic error only from the .stem that
		// owns the violation. Descendant resolutions include ancestor conflicts
		// in their chain, but those must not be re-reported at descendant paths.
		for _, conflict := range lr.Conflicts {
			if conflict.StemPath != sf || monotonicConflictIsType(conflict) {
				continue
			}
			identity := layerConflictIdentity{StemPath: conflict.StemPath, Field: conflict.Field, Operation: conflict.Operation}
			if _, exists := emittedMonotonic[identity]; exists {
				continue
			}
			emittedMonotonic[identity] = struct{}{}

			fieldName, msg := monotonicViolation(conflict)

			checks = append(checks, StemHealthCheck{
				Name:    "monotonic-violations",
				Status:  "fail",
				Message: msg,
				Path:    stemHealthRelPath(absRoot, conflict.StemPath),
				Field:   fieldName,
			})
		}
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Check 11: unknown keys under links.checks (silently inert otherwise)
	for _, sf := range sortedStemPaths(parsedStems) {
		stem := parsedStems[sf]
		for _, key := range stem.Links.UnknownCheckKeys {
			msg := fmt.Sprintf("unknown key %q in links.checks", key)
			if match := fuzzy.Match(key, knownCheckKeys); match != "" {
				msg += fmt.Sprintf(" (did you mean %q?)", match)
			}
			checks = append(checks, StemHealthCheck{
				Name:    "unknown-check-keys",
				Status:  "warn",
				Message: msg,
				Path:    stemHealthRelPath(absRoot, sf),
				Field:   key,
			})
		}
	}

	// Check 12: nested-root-marker (INFO level)
	// Detect when a directory declares root: true and has an ancestor also with root: true
	for _, sf := range sortedStemPaths(parsedStems) {
		stem := parsedStems[sf]
		if !stem.Root {
			continue // Only check stems with root: true
		}

		relPath := stemHealthRelPath(absRoot, sf)
		dir := filepath.Dir(sf)

		// Manually walk up the state path hierarchy (without stopping at markers) to find
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
			if parentStem, exists := state.Stems[parentStemPath]; exists && parentStem.Root {
				foundAncestorRoot = true
				ancestorRootPath = parentStemPath
				break
			}

			current = parent
		}

		if foundAncestorRoot {
			ancestorRelPath := stemHealthRelPath(absRoot, ancestorRootPath)
			checks = append(checks, StemHealthCheck{
				Name:    "nested-root-marker",
				Status:  "info",
				Message: fmt.Sprintf("%s declares root: true, so records below it do not inherit %s", relPath, ancestorRelPath),
				Path:    relPath,
			})
		}
	}

	sortStemHealthChecks(checks)
	return &StemHealthResult{Checks: checks}, nil
}

func stemHealthRelPath(absRoot, path string) string {
	relPath, err := filepath.Rel(absRoot, path)
	if err != nil {
		return path
	}
	return relPath
}

func entriesBeforeStem(entries []StemEntry, stemPath string) []StemEntry {
	for i, entry := range entries {
		if entry.Path == stemPath {
			return entries[:i]
		}
	}
	return nil
}

func sortedAggregateFieldNames(aggregate map[string]any) []string {
	fieldNames := make([]string, 0, len(aggregate))
	for fieldName := range aggregate {
		fieldNames = append(fieldNames, fieldName)
	}
	slices.Sort(fieldNames)
	return fieldNames
}

func sortStemHealthChecks(checks []StemHealthCheck) {
	slices.SortFunc(checks, func(a, b StemHealthCheck) int {
		if c := strings.Compare(a.Path, b.Path); c != 0 {
			return c
		}
		if c := strings.Compare(a.Name, b.Name); c != 0 {
			return c
		}
		if c := strings.Compare(a.Field, b.Field); c != 0 {
			return c
		}
		if c := strings.Compare(a.Status, b.Status); c != 0 {
			return c
		}
		return strings.Compare(a.Message, b.Message)
	})
}

func fieldDeclarationChecks(absRoot string, parsedStems map[string]*StemFile) []StemHealthCheck {
	stemPaths := sortedStemPaths(parsedStems)

	var checks []StemHealthCheck
	for _, stemPath := range stemPaths {
		stem := parsedStems[stemPath]
		relPath, _ := filepath.Rel(absRoot, stemPath)
		for _, fieldName := range sortedSchemaFieldNames(stem.Schema) {
			field := stem.Schema[fieldName]
			for _, contractIssue := range ValidateFieldDeclaration(fieldName, field) {
				checks = append(checks, StemHealthCheck{
					Name:    "field-declaration",
					Status:  "fail",
					Message: fmt.Sprintf("%s: %s", contractIssue.Code, contractIssue.Message),
					Path:    relPath,
					Field:   fieldName,
				})
			}
		}
	}
	return checks
}

func sortedStemPaths(parsedStems map[string]*StemFile) []string {
	stemPaths := make([]string, 0, len(parsedStems))
	for stemPath := range parsedStems {
		stemPaths = append(stemPaths, stemPath)
	}
	slices.Sort(stemPaths)
	return stemPaths
}

func validNarrowingOverride(parentField, childField SchemaField) bool {
	if len(CheckFieldCompatibility(parentField, childField)) != 0 {
		return false
	}
	if parentField.Type == "string" && childField.Type == "enum" {
		return true
	}
	return parentField.Type == "enum" && childField.Type == "enum" && len(childField.Values) > 0 && len(parentField.Values) > 0
}

type layerConflictIdentity struct {
	StemPath  string
	Field     string
	Operation string
}

func monotonicConflictIsType(conflict LayerConstraint) bool {
	_, constraint := monotonicField(conflict.Field)
	return constraint == "type"
}

// monotonicConstraintSuffixes are the schema-level constraints the resolver
// appends to a field name when it records a conflict. Anything else — a
// structural path like "structural.subdirs.min_children" — is reported whole.
var monotonicConstraintSuffixes = []string{"type", "source", "required", "severity", "values"}

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
	case "source":
		return field, fmt.Sprintf("field %q changes source: %v", field, conflict.Value)
	case "required":
		return field, fmt.Sprintf("field %q loosens required: %v", field, conflict.Value)
	case "severity":
		return field, fmt.Sprintf("field %q loosens severity: %v", field, conflict.Value)
	case "values":
		return field, fmt.Sprintf("field %q narrows enum incompatibly: %v", field, conflict.Value)
	}
	return field, fmt.Sprintf("constraint %q loosens the parent constraint: %v", field, conflict.Value)
}
