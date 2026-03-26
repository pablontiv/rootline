package rules

import (
	"path/filepath"
	"strings"
)

// DomainDef defines a well-known semantic domain with its base type
// and required attributes.
type DomainDef struct {
	Name          string
	BaseType      string
	RequiredAttrs []string
}

// coreDomains is the registry of well-known semantic domains.
var coreDomains = map[string]*DomainDef{
	"lifecycle_state": {Name: "lifecycle_state", BaseType: "enum", RequiredAttrs: []string{"values"}},
	"record_type":     {Name: "record_type", BaseType: "enum", RequiredAttrs: []string{"values"}},
	"identifier":      {Name: "identifier", BaseType: "sequence", RequiredAttrs: []string{"prefix", "digits"}},
	"title":           {Name: "title", BaseType: "string"},
	"created_date":    {Name: "created_date", BaseType: "string"},
	"due_date":        {Name: "due_date", BaseType: "string"},
	"owner":           {Name: "owner", BaseType: "string"},
	"parent_ref":      {Name: "parent_ref", BaseType: "string"},
	"priority":        {Name: "priority", BaseType: "enum", RequiredAttrs: []string{"values"}},
	"description":     {Name: "description", BaseType: "string"},
	"confidence":      {Name: "confidence", BaseType: "enum", RequiredAttrs: []string{"values"}},
	"source":          {Name: "source", BaseType: "string"},
}

// LookupDomain returns the DomainDef for a well-known domain name,
// or nil if the domain is custom (contains "/") or unknown.
func LookupDomain(name string) *DomainDef {
	if strings.Contains(name, "/") {
		return nil
	}
	return coreDomains[name]
}

// IsCustomDomain reports whether the domain name uses the namespace
// convention (contains "/"), indicating a user-defined domain.
func IsCustomDomain(name string) bool {
	return strings.Contains(name, "/")
}

// FindFieldByDomain returns the field name with the given domain in
// the effective schema for a record path. When multiple fields share
// the same domain (with different match patterns), it returns the one
// matching the deepest path component (closest to the file).
func FindFieldByDomain(schema map[string]SchemaField, domain string, recordPath string) (string, bool) {
	// Build pointer map for FilterSchemaByMatch
	ptrSchema := make(map[string]*SchemaField, len(schema))
	for name, field := range schema {
		f := field
		ptrSchema[name] = &f
	}

	filtered := FilterSchemaByMatch(ptrSchema, recordPath)
	components := pathComponents(recordPath)

	bestName := ""
	bestDepth := -1

	for name, field := range filtered {
		if field.Domain != domain {
			continue
		}
		if field.Match == nil {
			// No match constraint — global field, depth = -1 (lowest priority)
			if bestName == "" {
				bestName = name
				bestDepth = -1
			}
			continue
		}
		// Find the deepest matching component
		depth := matchDepth(field.Match, components)
		if depth > bestDepth {
			bestDepth = depth
			bestName = name
		}
	}

	return bestName, bestName != ""
}

// matchDepth returns the index of the deepest path component that
// matches any pattern in the FieldMatch. Returns -1 if no match.
func matchDepth(fm *FieldMatch, components []string) int {
	best := -1
	for i := len(components) - 1; i >= 0; i-- {
		for _, pattern := range fm.Patterns {
			if matched, _ := filepath.Match(pattern, components[i]); matched {
				if i > best {
					best = i
				}
			}
		}
	}
	return best
}

// DomainAliases builds a map from domain name to field name for all
// fields with domains in the effective schema for a record path.
func DomainAliases(schema map[string]SchemaField, recordPath string) map[string]string {
	// Build pointer map for FilterSchemaByMatch
	ptrSchema := make(map[string]*SchemaField, len(schema))
	for name, field := range schema {
		f := field
		ptrSchema[name] = &f
	}

	filtered := FilterSchemaByMatch(ptrSchema, recordPath)
	aliases := make(map[string]string)
	for name, field := range filtered {
		if field.Domain != "" {
			aliases[field.Domain] = name
		}
	}
	return aliases
}

// ResolveDomainType sets the Type field on schema entries based on their
// domain's base type, when the type is not explicitly declared.
// Called during ParseStem after initial parsing.
func ResolveDomainType(schema map[string]SchemaField) map[string]SchemaField {
	for name, field := range schema {
		if field.Domain == "" || field.Type != "" {
			continue
		}
		def := LookupDomain(field.Domain)
		if def != nil {
			field.Type = def.BaseType
			schema[name] = field
		}
	}
	return schema
}
