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

// ExpandLevels generates virtual StemEntries from a StemFile's Levels map
// for a given record's relative path. Each path component is matched against
// level patterns; matching levels produce a StemEntry with that level's
// schema and validate fields. Entries are returned in path order (shallowest
// first) for correct merge ordering.
func ExpandLevels(stem *StemFile, recordRelPath string) []StemEntry {
	if stem == nil || len(stem.Levels) == 0 {
		return nil
	}

	parts := strings.Split(filepath.ToSlash(recordRelPath), "/")
	var entries []StemEntry

	for _, part := range parts {
		levelName := matchLevel(stem.Levels, part)
		if levelName == "" {
			continue
		}

		level := stem.Levels[levelName]
		virtual := &StemFile{
			Path:   fmt.Sprintf("<levels:%s>", levelName),
			Schema: level.Schema,
		}
		if level.Validate != nil {
			virtual.Validate = level.Validate
		}

		entries = append(entries, StemEntry{
			Path: virtual.Path,
			Stem: virtual,
		})
	}

	return entries
}

// ResolveForRecord resolves the effective StemFile for a specific record
// by walking up the directory tree, merging .stem files, and expanding
// levels if present. Virtual level entries are appended after real entries
// so they override the base schema with per-level specifics.
func ResolveForRecord(dir string, recordPath string) (*StemFile, error) {
	entries, err := WalkUp(dir)
	if err != nil {
		return nil, err
	}
	merged := MergeStemFiles(entries)
	if merged.Levels != nil {
		virtualEntries := ExpandLevels(merged, recordPath)
		entries = append(entries, virtualEntries...)
		merged = MergeStemFiles(entries)
	}
	return merged, nil
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
