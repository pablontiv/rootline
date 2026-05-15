package rules

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
	filtered := FilterSchemaByMatch(ptrSchema, path)

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

		filtered := FilterSchemaByMatch(ptrSchema, path)

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
// violations when monotonic=true.
type LayeredResolution struct {
	*Resolution                   // Embed existing resolution
	Layers      []LayerConstraint // All constraints from all stems, root-to-leaf
	Conflicts   []LayerConstraint // Constraints that violate monotonic rules
}

// ResolveLayered returns a LayeredResolution that tracks constraints across
// the .stem chain and validates monotonic narrowing if enabled.
//
// In non-monotonic mode (monotonic=false), behaves like Resolve but populates
// Layers for provenance tracking. Conflicts is empty.
//
// In monotonic mode (monotonic=true), validates each child constraint against
// parent constraints. Populates Conflicts for violations:
//   - Type widening (e.g., enum → string)
//   - Required loosening (true → false)
//   - Enum extension without explicit narrowing to subset
//   - Severity loosening (error → warn)
//   - Structural loosening (min_children reduction)
//   - Domain redefinition
//   - etc.
func ResolveLayered(path string, root string, monotonic bool) (*LayeredResolution, error) {
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

	// Collect all constraints from all stems.
	for i, entry := range baseRes.Chain {
		stem := entry.Stem
		for fieldName, field := range stem.Schema {
			// Record type constraint
			lr.Layers = append(lr.Layers, LayerConstraint{
				StemPath:  entry.Path,
				Field:     fieldName + ".type",
				Value:     field.Type,
				Operation: "define",
			})

			// Record required constraint
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

			// Record enum values constraint
			if len(field.Values) > 0 {
				lr.Layers = append(lr.Layers, LayerConstraint{
					StemPath:  entry.Path,
					Field:     fieldName + ".values",
					Value:     field.Values,
					Operation: "define",
				})
			}

			// Record severity constraint
			if field.Severity != "" {
				lr.Layers = append(lr.Layers, LayerConstraint{
					StemPath:  entry.Path,
					Field:     fieldName + ".severity",
					Value:     field.Severity,
					Operation: "define",
				})
			}
		}

		// Validate monotonic constraints if enabled and not the first (root) level.
		if monotonic && i > 0 {
			validateMonotonicConstraints(baseRes.Chain[i-1].Stem, stem, entry.Path, lr)
		}
	}

	// Validate structural constraints if monotonic.
	if monotonic && len(baseRes.Chain) > 1 {
		validateMonotonicStructural(baseRes.Chain, lr)
	}

	return lr, nil
}

// validateMonotonicConstraints checks that child constraints do not violate parent constraints.
func validateMonotonicConstraints(parentStem, childStem *StemFile, childPath string, lr *LayeredResolution) {
	for fieldName, childField := range childStem.Schema {
		parentField, exists := parentStem.Schema[fieldName]
		if !exists {
			// New field in child — always allowed
			continue
		}

		// Type constraint: child can narrow (string→enum) but not widen
		if childField.Type != parentField.Type {
			if !isValidTypeNarrowing(parentField.Type, childField.Type) {
				lr.Conflicts = append(lr.Conflicts, LayerConstraint{
					StemPath:  childPath,
					Field:     fieldName + ".type",
					Value:     childField.Type + " (parent: " + parentField.Type + ")",
					Operation: "conflict",
				})
			}
		}

		// Required constraint: child cannot loosen
		if parentField.Required && !childField.Required && childField.RequiredMatch == nil {
			lr.Conflicts = append(lr.Conflicts, LayerConstraint{
				StemPath:  childPath,
				Field:     fieldName + ".required",
				Value:     "false (parent: true)",
				Operation: "conflict",
			})
		}

		// Enum values: child must be a subset (narrowing)
		if parentField.Type == "enum" && childField.Type == "enum" {
			if len(parentField.Values) > 0 && len(childField.Values) > 0 {
				if !isSubset(childField.Values, parentField.Values) {
					// Extension or incompatible set — this is a conflict
					added := findAdded(childField.Values, parentField.Values)
					if len(added) > 0 {
						lr.Conflicts = append(lr.Conflicts, LayerConstraint{
							StemPath:  childPath,
							Field:     fieldName + ".values",
							Value:     added,
							Operation: "extension",
						})
					}
				}
			}
		}

		// Severity constraint: child can tighten (warn→error) but not loosen
		if childField.Severity != "" && parentField.Severity != "" {
			parentSev := severityOrder[parentField.Severity]
			childSev := severityOrder[childField.Severity]
			if childSev < parentSev {
				lr.Conflicts = append(lr.Conflicts, LayerConstraint{
					StemPath:  childPath,
					Field:     fieldName + ".severity",
					Value:     childField.Severity + " (parent: " + parentField.Severity + ")",
					Operation: "conflict",
				})
			}
		}

	}
}

// validateMonotonicStructural checks structural constraint monotonicity.
func validateMonotonicStructural(chain []StemEntry, lr *LayeredResolution) {
	for i := 1; i < len(chain); i++ {
		parentStem := chain[i-1].Stem
		childStem := chain[i].Stem

		// min_children: child can increase (tighten) but not decrease (loosen)
		if parentStem.Structural.Subdirs.MinChildren > 0 && childStem.Structural.Subdirs.MinChildren > 0 {
			if childStem.Structural.Subdirs.MinChildren < parentStem.Structural.Subdirs.MinChildren {
				lr.Conflicts = append(lr.Conflicts, LayerConstraint{
					StemPath:  chain[i].Path,
					Field:     "structural.subdirs.min_children",
					Value:     childStem.Structural.Subdirs.MinChildren,
					Operation: "conflict",
				})
			}
		}

		// max_children: child can decrease (tighten) but not increase (loosen)
		if parentStem.Structural.Subdirs.MaxChildren > 0 && childStem.Structural.Subdirs.MaxChildren > 0 {
			if childStem.Structural.Subdirs.MaxChildren > parentStem.Structural.Subdirs.MaxChildren {
				lr.Conflicts = append(lr.Conflicts, LayerConstraint{
					StemPath:  chain[i].Path,
					Field:     "structural.subdirs.max_children",
					Value:     childStem.Structural.Subdirs.MaxChildren,
					Operation: "conflict",
				})
			}
		}
	}
}

// isValidTypeNarrowing returns true if childType is a valid narrowing of parentType.
// Valid narrowings:
//   - string → enum (restricts to enum values)
//   - int → positive-int (future)
func isValidTypeNarrowing(parentType, childType string) bool {
	if parentType == childType {
		return true
	}
	if parentType == "string" && childType == "enum" {
		return true
	}
	// Add more narrowing rules here as needed
	return false
}

// isSubset returns true if child is a subset of parent (all child values in parent).
func isSubset(child, parent []string) bool {
	parentMap := make(map[string]bool)
	for _, v := range parent {
		parentMap[v] = true
	}
	for _, v := range child {
		if !parentMap[v] {
			return false
		}
	}
	return true
}

// findAdded returns values in child that are not in parent.
func findAdded(child, parent []string) []string {
	parentMap := make(map[string]bool)
	for _, v := range parent {
		parentMap[v] = true
	}
	var added []string
	for _, v := range child {
		if !parentMap[v] {
			added = append(added, v)
		}
	}
	return added
}
