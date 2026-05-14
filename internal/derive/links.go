// Linked Field Injection
//
// When a .stem file defines link rules with a Field, the derive engine
// resolves wiki-links on each record and injects the referenced field
// values into the expression environment before evaluation.
//
// # Injected Variables
//
// For each link type with a Field defined in the stem's links.rules,
// a variable named after the Field is injected into the env map.
// The variable is a []any (slice of any) containing the string values of the
// ValueField frontmatter field from each resolved target record.
//
// Example .stem configuration (with explicit value_field):
//
//	links:
//	  rules:
//	    blocks:
//	      target: "*.md"
//	      field: blocked_by
//	      value_field: estado
//
// Example .stem configuration (backward compatible, defaults to "estado"):
//
//	links:
//	  rules:
//	    blocks:
//	      target: "*.md"
//	      field: blocked_by
//
// Given a record with [[blocks:B.md]] and [[blocks:C.md]], where B has
// estado=Completed and C has estado=Pending, the env will contain:
//
//	blocked_by = []any{"Completed", "Pending"}
//
// If no links of that type exist (or none resolve), the variable is not set
// in the env (i.e., it is nil in expr-lang expressions). If value_field is
// not specified, it defaults to "estado" for backward compatibility.
//
// # Usage in Derive Expressions
//
// The injected variables can be used with expr-lang builtins:
//
//	all(blocked_by, {# == "Completed"})  # true if all blockers are done
//	any(blocked_by, {# == "Blocked"})    # true if any blocker is blocked
//	len(blocked_by)                       # number of resolved links
//	blocked_by == nil                     # true if no links of this type
//	blocked_by[0]                         # first linked value
package derive

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

// RecordResolver resolves a record path to its Record.
// Returns nil if the path does not match any known record.
type RecordResolver interface {
	Resolve(path string) *extract.Record
}

// MapResolver is a RecordResolver backed by a map of path to Record.
type MapResolver struct {
	records  map[string]*extract.Record
	basename map[string][]*extract.Record
}

// NewMapResolver builds a MapResolver from a slice of records.
// It indexes records by exact path and by basename (with and without .md)
// for fallback resolution, mirroring the graph package's basename logic.
func NewMapResolver(records []*extract.Record) *MapResolver {
	m := &MapResolver{
		records:  make(map[string]*extract.Record, len(records)),
		basename: make(map[string][]*extract.Record, len(records)),
	}
	for _, r := range records {
		m.records[r.Path] = r
		base := filepath.Base(r.Path)
		m.basename[base] = append(m.basename[base], r)
		noExt := strings.TrimSuffix(base, ".md")
		if noExt != base {
			m.basename[noExt] = append(m.basename[noExt], r)
		}
	}
	return m
}

// Resolve returns the record at the given path, or nil if not found.
// It tries exact path match first, then falls back to basename lookup
// (only if the basename is unambiguous).
func (m *MapResolver) Resolve(path string) *extract.Record {
	if r, ok := m.records[path]; ok {
		return r
	}
	if matches, ok := m.basename[path]; ok && len(matches) == 1 {
		return matches[0]
	}
	return nil
}

// InjectLinkedFields resolves wiki-links on a record using the link schema
// from the effective .stem, and injects the referenced field values into env.
//
// For each link on the record, it looks up the link type in the stem's
// LinkSchema.Rules. If a rule defines a Field, the linked record is resolved
// and its ValueField frontmatter value is extracted and accumulated into a
// string slice keyed by rule.Field in the env map. If ValueField is not
// specified, it defaults to "estado" for backward compatibility.
//
// Links to non-existent targets are silently skipped.
func InjectLinkedFields(env map[string]any, record *extract.Record, stem *rules.StemFile, resolver RecordResolver) {
	if resolver == nil || stem == nil || len(record.Links) == 0 || stem.Links.Rules == nil {
		return
	}

	fieldValues := make(map[string][]any)

	for _, link := range record.Links {
		rule, ok := stem.Links.Rules[link.Type]
		if !ok || rule.Field == "" {
			continue
		}

		// Resolve the target path relative to the source, then look it up.
		resolved := resolveTarget(record.Path, link.Target)
		target := resolver.Resolve(resolved)
		if target == nil {
			// Try the raw target as well (for basename-only references).
			target = resolver.Resolve(link.Target)
		}
		if target == nil {
			continue
		}

		// Extract the target's ValueField value.
		// If ValueField is not specified, default to "estado" for backward compatibility.
		valueField := rule.ValueField
		if valueField == "" {
			valueField = "estado"
		}

		val, ok := target.EffectiveField(valueField)
		if ok {
			fieldValues[rule.Field] = append(fieldValues[rule.Field], fmt.Sprintf("%v", val))
		}
	}

	for field, values := range fieldValues {
		env[field] = values
	}
}

// resolveTarget resolves a link target relative to the source record's directory.
// If the target contains a path separator or "..", it's resolved relative to
// the source's directory. Otherwise it's returned as-is.
func resolveTarget(sourcePath, target string) string {
	if strings.Contains(target, "/") || strings.Contains(target, "..") {
		dir := filepath.Dir(sourcePath)
		return filepath.Clean(filepath.Join(dir, target))
	}
	return target
}
