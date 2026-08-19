package rules

// MergeStemFiles merges an ordered list of StemEntries (root-to-leaf)
// into a single effective StemFile. Merge behavior is type-driven:
//   - map + map → recursive key-level merge
//   - array + array → child replaces (except validate, which accumulates)
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

		// Validate: accumulate rules from all levels.
		if s.Validate != nil {
			result.Validate = append(result.Validate, s.Validate...)
		}

		// Derive, Aggregate: generic type-driven merge.
		result.Derive = mergeAnyMap(result.Derive, s.Derive)
		result.Aggregate = mergeAnyMap(result.Aggregate, s.Aggregate)

		// Links: typed merge (Allowed replaces, Rules map-merges).
		result.Links = mergeLinkSchema(result.Links, s.Links)

		// Structural: child replaces if non-empty.
		if !s.Structural.IsEmpty() {
			result.Structural = s.Structural
		}

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
		if parentField, exists := out[k]; exists {
			v = mergeFieldSource(parentField, v)
			v = mergeFieldSeverity(parentField, v)
		}
		out[k] = v
	}
	return out
}

// mergeFieldSource carries an inherited extraction source through child fields
// that omit source. Explicit empty or null source declarations are preserved as
// removals so compatibility checks can reject them when inherited.
func mergeFieldSource(parent, child SchemaField) SchemaField {
	if child.Extract == "" && !child.declaration.SourcePresent {
		child.Extract = parent.Extract
	}
	return child
}

// mergeFieldSeverity applies the tighten-only rule for severity:
// child can tighten (warn -> error) but not loosen (error -> warn).
// An unspecified severity ("") defaults to "error" — the absence of
// an explicit severity is not a loosening intent.
func mergeFieldSeverity(parent, child SchemaField) SchemaField {
	parentSeverity, parentOK := normalizedSeverity(parent.Severity)
	childSeverity, childOK := normalizedSeverity(child.Severity)
	if !parentOK || !childOK {
		return child
	}
	if severityOrder[childSeverity] < severityOrder[parentSeverity] {
		// Child is trying to loosen — keep parent's stricter severity.
		child.Severity = parentSeverity
	} else {
		child.Severity = childSeverity
	}
	return child
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

// mergeLinkSchema merges two LinkSchema values.
// Allowed, Styles, Checks (arrays/struct) → child replaces. Rules (map) → key-level merge.
func mergeLinkSchema(parent, child LinkSchema) LinkSchema {
	if child.IsEmpty() {
		return parent
	}
	if parent.IsEmpty() {
		return child
	}

	result := LinkSchema{}

	// BasenameFallback: a child that opts in wins; otherwise inherit. It is a
	// bool, so "not set" and "set false" are indistinguishable — a child
	// cannot opt back out of a parent's fallback, matching how the other
	// scalar link settings inherit.
	result.BasenameFallback = parent.BasenameFallback || child.BasenameFallback

	// Allowed: child replaces (array semantics).
	if child.Allowed != nil {
		result.Allowed = child.Allowed
	} else {
		result.Allowed = parent.Allowed
	}

	// Styles: child replaces (array semantics).
	if child.Styles != nil {
		result.Styles = child.Styles
	} else {
		result.Styles = parent.Styles
	}

	// Checks: child replaces when declared (struct semantics), except that an
	// undeclared resolve inherits rather than reverting to the default.
	//
	// resolve is tri-state and defaults to on, so wholesale replacement would
	// let a child declaring only `encoding: true` silently switch a parent's
	// `resolve: false` back on. Opting out of a check has to survive being
	// inherited, or the opt-out is worthless in any nested tree.
	if child.Checks != nil {
		merged := *child.Checks
		if merged.Resolve == nil && parent.Checks != nil {
			merged.Resolve = parent.Checks.Resolve
		}
		result.Checks = &merged
	} else {
		result.Checks = parent.Checks
	}

	// Rules: map merge at key level.
	result.Rules = make(map[string]LinkRule, len(parent.Rules)+len(child.Rules))
	for k, v := range parent.Rules {
		result.Rules[k] = v
	}
	for k, v := range child.Rules {
		result.Rules[k] = v
	}

	return result
}
