package rules

import "github.com/pablontiv/rootline/internal/extract"

// ResolveFieldValue resolves a schema field using frontmatter-first precedence.
func ResolveFieldValue(record *extract.Record, name string, field SchemaField) (any, bool, error) {
	if record != nil && record.Frontmatter != nil {
		if v, ok := record.Frontmatter[name]; ok {
			return v, true, nil
		}
	}
	if field.Extract == "" {
		return nil, false, nil
	}
	v, ok, err := extract.ResolveBodyValue(record, field.Extract)
	if err != nil || !ok {
		return nil, false, err
	}
	return v, true, nil
}

// ResolveEffectiveField resolves a field using the schema contract when source-backed.
func ResolveEffectiveField(record *extract.Record, effective *StemFile, name string) (any, bool, error) {
	if effective != nil {
		if field, ok := effective.Schema[name]; ok && field.Extract != "" {
			return ResolveFieldValue(record, name, field)
		}
	}
	if record == nil {
		return nil, false, nil
	}
	v, ok := record.EffectiveField(name)
	return v, ok, nil
}
