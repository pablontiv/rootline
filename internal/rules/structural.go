package rules

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// ValidateDirectory checks structural rules for the given directory against
// the effective .stem file. It validates directory structure (not file contents).
func ValidateDirectory(_ context.Context, dir string, stem *StemFile) []ValidationError {
	if stem == nil || stem.Structural.IsEmpty() {
		return nil
	}

	rules := stem.Structural.Subdirs
	severity := rules.Severity
	if severity == "" {
		severity = "error"
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return []ValidationError{{
			Rule:     "structural",
			Field:    "directory",
			Message:  fmt.Sprintf("cannot read directory: %v", err),
			Source:   stem.Path,
			Severity: severity,
		}}
	}

	// Collect subdirectories.
	var subdirs []string
	for _, e := range entries {
		if e.IsDir() && !isHiddenEntry(e.Name()) {
			subdirs = append(subdirs, e.Name())
		}
	}

	var errs []ValidationError

	// Check min_children.
	if rules.MinChildren > 0 && len(subdirs) < rules.MinChildren {
		errs = append(errs, ValidationError{
			Rule:     "min_children",
			Field:    "directory",
			Message:  fmt.Sprintf("directory %q has %d subdirectories, minimum is %d", dir, len(subdirs), rules.MinChildren),
			Source:   stem.Path,
			Severity: severity,
		})
	}

	// Check max_children.
	if rules.MaxChildren > 0 && len(subdirs) > rules.MaxChildren {
		errs = append(errs, ValidationError{
			Rule:     "max_children",
			Field:    "directory",
			Message:  fmt.Sprintf("directory %q has %d subdirectories, maximum is %d", dir, len(subdirs), rules.MaxChildren),
			Source:   stem.Path,
			Severity: severity,
		})
	}

	// Check require_index.
	if rules.RequireIndex != "" {
		for _, subdir := range subdirs {
			indexPath := filepath.Join(dir, subdir, rules.RequireIndex)
			if _, err := os.Stat(indexPath); os.IsNotExist(err) {
				errs = append(errs, ValidationError{
					Rule:     "require_index",
					Field:    "directory",
					Message:  fmt.Sprintf("subdirectory %q is missing required index file %q", subdir, rules.RequireIndex),
					Source:   stem.Path,
					Severity: severity,
				})
			}
		}
	}

	return errs
}

// isHiddenEntry returns true for dot-prefixed directory names.
func isHiddenEntry(name string) bool {
	return len(name) > 0 && name[0] == '.'
}
