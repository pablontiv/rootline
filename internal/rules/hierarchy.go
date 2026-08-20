package rules

// ResolveForRecord resolves the effective StemFile for a specific record
// by walking up the directory tree and merging .stem files top-down.
// It applies FilterSchemaByMatch to produce the effective schema for the
// record's path, resolving match-scoped fields and RequiredMatch.
func ResolveForRecord(dir string, recordPath string) (*StemFile, error) {
	entries, err := WalkUp(dir)
	if err != nil {
		return nil, err
	}
	merged := MergeStemFiles(entries)

	// Apply match-based field scoping
	if len(merged.Schema) > 0 {
		// Build pointer map for FilterSchemaByMatch
		ptrSchema := make(map[string]*SchemaField, len(merged.Schema))
		for name, field := range merged.Schema {
			f := field
			ptrSchema[name] = &f
		}

		filtered, err := FilterSchemaByMatch(ptrSchema, recordPath)
		if err != nil {
			return nil, err
		}

		// Write back to value map
		effective := make(map[string]SchemaField, len(filtered))
		for name, field := range filtered {
			effective[name] = *field
		}
		merged.Schema = effective
	}

	return merged, nil
}
