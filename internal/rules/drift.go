package rules

import (
	"fmt"
	"path/filepath"

	"github.com/pablontiv/rootline/internal/extract"
)

// DriftWarning represents a detected inconsistency between a parent record
// and its children for a shared field.
type DriftWarning struct {
	Field         string   `json:"field"`
	ParentValue   any      `json:"parent_value"`
	ChildrenValue any      `json:"children_value"`
	ParentPath    string   `json:"parent_path"`
	ChildPaths    []string `json:"child_paths"`
}

// DetectDrift compares shared fields between a parent record and its children.
// For each field in schema where Match is nil (applies everywhere):
// if all children agree on a value that differs from the parent's value,
// a DriftWarning is returned.
func DetectDrift(parent extract.Record, children []extract.Record, schema map[string]SchemaField) []DriftWarning {
	if len(children) == 0 {
		return nil
	}

	var warnings []DriftWarning

	for fieldName, field := range schema {
		// Only check fields without match restriction (applies everywhere)
		if field.Match != nil {
			continue
		}

		parentVal, parentHas := parent.Frontmatter[fieldName]
		if !parentHas {
			continue
		}

		// Collect children values for this field
		var childVals []any
		var childPaths []string
		for _, child := range children {
			val, has := child.Frontmatter[fieldName]
			if has {
				childVals = append(childVals, val)
				childPaths = append(childPaths, child.Path)
			}
		}

		if len(childVals) == 0 {
			continue
		}

		// Check if all children agree on the same value
		consensus, unanimous := unanimousValue(childVals)
		if !unanimous {
			continue
		}

		// Compare with parent
		if !valuesEqual(parentVal, consensus) {
			warnings = append(warnings, DriftWarning{
				Field:         fieldName,
				ParentValue:   parentVal,
				ChildrenValue: consensus,
				ParentPath:    parent.Path,
				ChildPaths:    childPaths,
			})
		}
	}

	return warnings
}

// unanimousValue returns the consensus value and true if all values are equal.
func unanimousValue(vals []any) (any, bool) {
	if len(vals) == 0 {
		return nil, false
	}
	first := vals[0]
	for _, v := range vals[1:] {
		if !valuesEqual(first, v) {
			return nil, false
		}
	}
	return first, true
}

// ParentChildGroup holds an index file and its direct children for drift detection.
type ParentChildGroup struct {
	Parent   *extract.Record
	Children []extract.Record
}

// GroupByParentDir groups records by their parent directory.
// For each directory, identifies the index file (README.md) as parent
// and other files as children.
func GroupByParentDir(records []*extract.Record, root string) map[string]*ParentChildGroup {
	groups := make(map[string]*ParentChildGroup)

	for _, rec := range records {
		absPath := filepath.Join(root, rec.Path)
		dir := filepath.Dir(absPath)

		if groups[dir] == nil {
			groups[dir] = &ParentChildGroup{}
		}

		if filepath.Base(rec.Path) == "README.md" {
			groups[dir].Parent = rec
		} else {
			groups[dir].Children = append(groups[dir].Children, *rec)
		}
	}

	return groups
}

// valuesEqual compares two frontmatter values using string representation.
func valuesEqual(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	// Use fmt.Sprint for consistent comparison of YAML-parsed values
	// (int vs string, etc.)
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}
