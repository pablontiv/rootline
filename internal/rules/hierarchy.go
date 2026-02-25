package rules

import (
	"fmt"
	"path/filepath"
	"strings"
)

// CheckNesting validates that a record's path respects the hierarchy defined
// by levels. Each path component must be a child level allowed by its parent.
// Returns nil if levels is nil or empty (transparent skip).
func CheckNesting(levels map[string]*HierarchyLevel, recordRelPath string) []ValidationError {
	if len(levels) == 0 {
		return nil
	}

	parts := strings.Split(filepath.ToSlash(recordRelPath), "/")
	if len(parts) == 0 {
		return nil
	}

	var errs []ValidationError
	var parentLevelName string

	for _, part := range parts {
		levelName := matchLevel(levels, part)
		if levelName == "" {
			// Component doesn't match any level — skip (unknown component).
			parentLevelName = ""
			continue
		}

		if parentLevelName != "" {
			parentLevel := levels[parentLevelName]
			if !containsString(parentLevel.Children, levelName) {
				errs = append(errs, ValidationError{
					Rule:     "nesting",
					Field:    "",
					Message:  fmt.Sprintf("level %q is not an allowed child of %q (allowed: %v)", levelName, parentLevelName, parentLevel.Children),
					Severity: "error",
				})
			}
		}

		parentLevelName = levelName
	}

	return errs
}

// matchLevel returns the level name whose Match pattern matches the given
// path component, or "" if no level matches.
func matchLevel(levels map[string]*HierarchyLevel, component string) string {
	for name, level := range levels {
		matched, err := filepath.Match(level.Match, component)
		if err == nil && matched {
			return name
		}
	}
	return ""
}

// containsString checks if a string slice contains a given value.
func containsString(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
