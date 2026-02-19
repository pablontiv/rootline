package rules

// MergeStemFiles merges an ordered list of StemEntries (root-to-leaf)
// into a single effective StemFile. Merge behavior is type-driven:
//   - map + map → recursive key-level merge
//   - array + array → child replaces
//   - scalar + scalar → child replaces
//   - any + nil → key removed
func MergeStemFiles(entries []StemEntry) *StemFile {
	result := &StemFile{}
	if len(entries) == 0 {
		return result
	}

	for _, entry := range entries {
		s := entry.Stem
		path := entry.Path

		// Version: last writer wins.
		if s.Version != 0 {
			result.Version = s.Version
		}

		// Scope: last writer wins (scalar-like).
		if s.Scope.Match != "" {
			result.Scope = s.Scope
		}

		// Schema: map merge with source tracking.
		result.Schema = mergeSchemaFields(result.Schema, s.Schema, path)

		// Validate: array → child replaces entirely.
		if s.Validate != nil {
			result.Validate = s.Validate
		}

		// Derive, State, Links: generic type-driven merge.
		result.Derive = mergeAnyMap(result.Derive, s.Derive)
		result.State = mergeAnyMap(result.State, s.State)
		result.Links = mergeAnyMap(result.Links, s.Links)

		// Track which .stem files contributed.
		result.Path = path
	}

	return result
}

// mergeSchemaFields merges two schema maps with source tracking.
func mergeSchemaFields(parent, child map[string]SchemaField, source string) map[string]SchemaField {
	if len(child) == 0 {
		return parent
	}
	if len(parent) == 0 {
		// Tag all child fields with source.
		out := make(map[string]SchemaField, len(child))
		for k, v := range child {
			v.Source = source
			out[k] = v
		}
		return out
	}

	out := make(map[string]SchemaField, len(parent)+len(child))
	for k, v := range parent {
		out[k] = v
	}
	for k, v := range child {
		v.Source = source
		out[k] = v
	}
	return out
}

// mergeAnyMap performs a type-driven merge of two map[string]any values.
// - Maps merge recursively at the key level.
// - nil values in child remove the key.
// - All other types (arrays, scalars): child replaces.
func mergeAnyMap(parent, child map[string]any) map[string]any {
	if len(child) == 0 {
		return parent
	}
	if len(parent) == 0 {
		return child
	}

	out := make(map[string]any, len(parent)+len(child))
	for k, v := range parent {
		out[k] = v
	}
	for k, v := range child {
		if v == nil {
			delete(out, k)
			continue
		}
		childMap, childIsMap := toStringMap(v)
		parentVal, parentExists := out[k]
		if parentExists {
			parentMap, parentIsMap := toStringMap(parentVal)
			if childIsMap && parentIsMap {
				out[k] = mergeAnyMapGeneric(parentMap, childMap)
				continue
			}
		}
		// Array, scalar, or parent not a map: child replaces.
		out[k] = v
	}
	return out
}

// mergeAnyMapGeneric recursively merges two map[string]any values.
func mergeAnyMapGeneric(parent, child map[string]any) map[string]any {
	out := make(map[string]any, len(parent)+len(child))
	for k, v := range parent {
		out[k] = v
	}
	for k, v := range child {
		if v == nil {
			delete(out, k)
			continue
		}
		childMap, childIsMap := toStringMap(v)
		parentVal, parentExists := out[k]
		if parentExists {
			parentMap, parentIsMap := toStringMap(parentVal)
			if childIsMap && parentIsMap {
				out[k] = mergeAnyMapGeneric(parentMap, childMap)
				continue
			}
		}
		out[k] = v
	}
	return out
}

// toStringMap attempts to convert an any value to map[string]any.
// YAML v3 unmarshals maps as map[string]interface{}.
func toStringMap(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	return m, ok
}
