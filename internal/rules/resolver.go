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
