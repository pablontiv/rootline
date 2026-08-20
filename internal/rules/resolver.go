package rules

import "slices"

// Resolution holds the complete resolution context for a file path:
// the stem chain (root-to-leaf), the effective merged schema, and per-field provenance.
type Resolution struct {
	// Path is the absolute path to the record being resolved.
	Path string

	// Chain is the ordered list of StemFiles from root to leaf that apply to this path.
	Chain []StemEntry

	// EffectiveSchema is the merged schema from all stems in the chain,
	// with match filtering applied for the specific record path.
	EffectiveSchema map[string]SchemaField

	// EffectiveStem is the complete merged StemFile (before match filtering).
	// Useful for accessing Validate, Derive, Aggregate, Links, etc.
	EffectiveStem *StemFile

	// Provenance maps field names to the path of the .stem file that defined them.
	// Fields introduced at multiple levels show the closest (leaf-most) source.
	Provenance map[string]string
}

// StemChain returns the stem files from root to leaf that apply to the given path.
// The path may be a directory or file; the returned entries are guaranteed to be
// ordered root-to-leaf (top-down merge order).
//
// This is a convenience wrapper around WalkUp that returns StemEntry objects
// for transparent integration with existing code.
func StemChain(path string, root string) ([]StemEntry, error) {
	// If path is relative, assume it's relative to root; otherwise use as-is.
	// For now, we just delegate to WalkUp which handles both cases.
	return WalkUp(path)
}

// EffectiveSchema returns the merged schema for a path, with match-based field
// filtering applied. This is the schema that actually applies to the record.
//
// Fields without Match constraints apply everywhere. Fields with Match constraints
// only apply when the record's path matches one of the patterns. RequiredMatch
// is resolved based on the specific record path.
func EffectiveSchema(path string, root string) (map[string]SchemaField, error) {
	entries, err := WalkUp(path)
	if err != nil {
		return nil, err
	}

	// Merge all stems top-down to get the raw schema.
	merged := MergeStemFiles(entries)

	// Build pointer map for FilterSchemaByMatch.
	if len(merged.Schema) == 0 {
		return make(map[string]SchemaField), nil
	}

	ptrSchema := make(map[string]*SchemaField, len(merged.Schema))
	for name, field := range merged.Schema {
		f := field
		ptrSchema[name] = &f
	}

	// Apply match-based field scoping for the specific record path.
	filtered, err := FilterSchemaByMatch(ptrSchema, path)
	if err != nil {
		return nil, err
	}

	// Convert back to value map.
	effective := make(map[string]SchemaField, len(filtered))
	for name, field := range filtered {
		effective[name] = *field
	}

	return effective, nil
}

// Resolve returns the complete resolution context for a file path:
// the stem chain, the effective merged schema, and field provenance.
//
// Provenance tracks which .stem file defined each field, enabling callers
// to understand the inheritance hierarchy and identify where schema changes
// need to be applied.
//
// This is the new central resolver API that replaces ad-hoc combinations
// of WalkUp + MergeStemFiles + FilterSchemaByMatch throughout the codebase.
func Resolve(path string, root string) (*Resolution, error) {
	entries, err := WalkUp(path)
	if err != nil {
		return nil, err
	}

	// Merge all stems top-down to get the raw and effective stems.
	mergedRaw := MergeStemFiles(entries)

	// Build the effective schema (with match filtering).
	effective := make(map[string]SchemaField)
	if len(mergedRaw.Schema) > 0 {
		ptrSchema := make(map[string]*SchemaField, len(mergedRaw.Schema))
		for name, field := range mergedRaw.Schema {
			f := field
			ptrSchema[name] = &f
		}

		filtered, err := FilterSchemaByMatch(ptrSchema, path)
		if err != nil {
			return nil, err
		}

		// Convert back to value map.
		for name, field := range filtered {
			effective[name] = *field
		}
	}

	// Build provenance map: field name → closest (leaf-most) .stem path that defined it.
	provenance := make(map[string]string)
	for _, entry := range entries {
		for name := range entry.Stem.Schema {
			// Last writer wins — later entries (closer to leaf) override earlier ones.
			provenance[name] = entry.Path
		}
	}

	return &Resolution{
		Path:            path,
		Chain:           entries,
		EffectiveSchema: effective,
		EffectiveStem:   mergedRaw,
		Provenance:      provenance,
	}, nil
}

// ClosestStem returns the leaf-most (closest) StemEntry in the chain.
// This is the .stem file most specific to the record path.
// Returns nil if the chain is empty.
func (r *Resolution) ClosestStem() *StemEntry {
	if len(r.Chain) == 0 {
		return nil
	}
	return &r.Chain[len(r.Chain)-1]
}

// RootMostStem returns the root-most (furthest) StemEntry in the chain.
// This is the .stem file at the repository root or near it.
// Returns nil if the chain is empty.
func (r *Resolution) RootMostStem() *StemEntry {
	if len(r.Chain) == 0 {
		return nil
	}
	return &r.Chain[0]
}

// LayerConstraint represents a single constraint from a specific .stem file layer.
type LayerConstraint struct {
	StemPath  string      // Path to the .stem file that defined this constraint
	Field     string      // Field name
	Value     interface{} // The constraint value (e.g., enum values, type, required status)
	Operation string      // "define", "narrow", "conflict", "destructive", "extension"
}

// LayeredResolution extends Resolution with monotonic constraint tracking.
// It tracks all constraints from all .stem files in the chain and identifies
// violations.
type LayeredResolution struct {
	*Resolution                   // Embed existing resolution
	Layers      []LayerConstraint // All constraints from all stems, root-to-leaf
	Conflicts   []LayerConstraint // Constraints that violate monotonic rules
}

// ResolveLayered returns a LayeredResolution that tracks constraints across
// the .stem chain and validates monotonic narrowing. Conflicts records
// violations such as type widening, required loosening, enum extension,
// severity loosening, and structural loosening.
func ResolveLayered(path string, root string) (*LayeredResolution, error) {
	// First, get the base resolution.
	baseRes, err := Resolve(path, root)
	if err != nil {
		return nil, err
	}

	lr := &LayeredResolution{
		Resolution: baseRes,
		Layers:     make([]LayerConstraint, 0),
		Conflicts:  make([]LayerConstraint, 0),
	}

	if len(baseRes.Chain) == 0 {
		return lr, nil
	}

	// Collect all constraints from all stems and compare every child against
	// the cumulative merged ancestor before applying that child.
	var cumulative *StemFile
	for i, entry := range baseRes.Chain {
		stem := entry.Stem
		appendLayerConstraints(lr, entry)

		if i > 0 {
			validateMonotonicConstraints(cumulative, stem, entry.Path, lr)
			validateMonotonicStructural(cumulative, stem, entry.Path, lr)
		}
		cumulative = MergeStemFiles(baseRes.Chain[:i+1])
	}

	return lr, nil
}

func appendLayerConstraints(lr *LayeredResolution, entry StemEntry) {
	for _, fieldName := range sortedSchemaFieldNames(entry.Stem.Schema) {
		field := entry.Stem.Schema[fieldName]
		lr.Layers = append(lr.Layers, LayerConstraint{
			StemPath:  entry.Path,
			Field:     fieldName + ".type",
			Value:     field.Type,
			Operation: "define",
		})
		if field.Required || field.RequiredMatch != nil {
			var reqValue interface{} = field.Required
			if field.RequiredMatch != nil {
				reqValue = field.RequiredMatch
			}
			lr.Layers = append(lr.Layers, LayerConstraint{
				StemPath:  entry.Path,
				Field:     fieldName + ".required",
				Value:     reqValue,
				Operation: "define",
			})
		}
		if len(field.Values) > 0 {
			lr.Layers = append(lr.Layers, LayerConstraint{
				StemPath:  entry.Path,
				Field:     fieldName + ".values",
				Value:     field.Values,
				Operation: "define",
			})
		}
		if field.Severity != "" {
			lr.Layers = append(lr.Layers, LayerConstraint{
				StemPath:  entry.Path,
				Field:     fieldName + ".severity",
				Value:     field.Severity,
				Operation: "define",
			})
		}
	}
}

// validateMonotonicConstraints checks that child constraints do not violate parent constraints.
func validateMonotonicConstraints(parentStem, childStem *StemFile, childPath string, lr *LayeredResolution) {
	if parentStem == nil || childStem == nil {
		return
	}
	for _, fieldName := range sortedSchemaFieldNames(childStem.Schema) {
		childField := childStem.Schema[fieldName]
		parentField, exists := parentStem.Schema[fieldName]
		if !exists {
			continue
		}
		for _, issue := range CheckFieldCompatibility(parentField, childField) {
			lr.Conflicts = append(lr.Conflicts, LayerConstraint{
				StemPath:  childPath,
				Field:     fieldName + "." + issue.Constraint,
				Value:     compatibilityLayerValue(issue),
				Operation: compatibilityLayerOperation(issue),
			})
		}
	}
}

func compatibilityLayerOperation(issue FieldCompatibilityIssue) string {
	if issue.Constraint == "values" && issue.Operation == "extension" {
		return "extension"
	}
	return "conflict"
}

func compatibilityLayerValue(issue FieldCompatibilityIssue) any {
	if issue.Constraint == "source" {
		return issue.Message
	}
	return issue.Value
}

func sortedSchemaFieldNames(schema map[string]SchemaField) []string {
	names := make([]string, 0, len(schema))
	for name := range schema {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// validateMonotonicStructural checks structural constraint monotonicity.
func validateMonotonicStructural(parentStem, childStem *StemFile, childPath string, lr *LayeredResolution) {
	if parentStem == nil || childStem == nil {
		return
	}
	if parentStem.Structural.Subdirs.MinChildren > 0 && childStem.Structural.Subdirs.MinChildren > 0 && childStem.Structural.Subdirs.MinChildren < parentStem.Structural.Subdirs.MinChildren {
		lr.Conflicts = append(lr.Conflicts, LayerConstraint{
			StemPath:  childPath,
			Field:     "structural.subdirs.min_children",
			Value:     childStem.Structural.Subdirs.MinChildren,
			Operation: "conflict",
		})
	}
	if parentStem.Structural.Subdirs.MaxChildren > 0 && childStem.Structural.Subdirs.MaxChildren > 0 && childStem.Structural.Subdirs.MaxChildren > parentStem.Structural.Subdirs.MaxChildren {
		lr.Conflicts = append(lr.Conflicts, LayerConstraint{
			StemPath:  childPath,
			Field:     "structural.subdirs.max_children",
			Value:     childStem.Structural.Subdirs.MaxChildren,
			Operation: "conflict",
		})
	}
}
