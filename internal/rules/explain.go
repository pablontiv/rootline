package rules

import (
	"sort"

	"github.com/pablontiv/rootline/internal/extract"
)

// ExplainResult is the versioned JSON output for the explain command.
type ExplainResult struct {
	Version    int               `json:"version"`
	Kind       string            `json:"kind"`
	Path       string            `json:"path"`
	StemChain  []string          `json:"stem_chain"`
	Layers     []string          `json:"layers,omitempty"`
	Provenance map[string]string `json:"provenance,omitempty"`
	Fields     []ExplainField    `json:"fields"`
	Errors     []ExplainError    `json:"errors,omitempty"`
}

// ExplainField traces a single field's value and origin.
type ExplainField struct {
	Name       string `json:"name"`
	Value      any    `json:"value"`
	Origin     string `json:"origin"`
	Source     string `json:"source,omitempty"`
	DefinedIn  string `json:"defined_in,omitempty"`
	Expression string `json:"expression,omitempty"`
}

// ExplainError traces a validation error to its source rule.
type ExplainError struct {
	Rule                   string `json:"rule"`
	Field                  string `json:"field"`
	Message                string `json:"message"`
	Source                 string `json:"source"`
	Severity               string `json:"severity"`
	Suggestion             string `json:"suggestion,omitempty"`
	ExpectedRepresentation string `json:"-"`
	ActualRepresentation   string `json:"-"`
}

// NewExplainResult builds an ExplainResult from walk-up entries, the effective
// StemFile, and validation errors for a given path. Source-resolution failures
// are returned before callers render a partial inspection payload.
func NewExplainResult(
	path string,
	entries []StemEntry,
	effective *StemFile,
	record *extract.Record,
	valErrs []ValidationError,
) (*ExplainResult, error) {
	chain := make([]string, len(entries))
	for i, e := range entries {
		chain[i] = e.Path
	}

	// Build layers (same as chain for now, but semantically distinct)
	layers := make([]string, len(entries))
	for i, e := range entries {
		layers[i] = e.Path
	}

	// Build provenance map: field name → stem path that defined it
	provenance := make(map[string]string)
	for _, entry := range entries {
		if entry.Stem != nil {
			for name := range entry.Stem.Schema {
				// Last writer wins — later entries (closer to leaf) override earlier ones.
				provenance[name] = entry.Path
			}
		}
	}

	var fields []ExplainField
	for _, name := range explainFieldNames(record, effective) {
		field := ExplainField{Name: name, Origin: explainOriginFromRecord(record, name)}
		if record != nil {
			if value, ok := record.EffectiveField(name); ok {
				field.Value = value
			}
		}
		if effective != nil {
			if sf, ok := effective.Schema[name]; ok {
				value, valueOK, err := ResolveEffectiveField(record, effective, name)
				if err != nil {
					return nil, err
				}
				if valueOK {
					field.Value = value
					if field.Origin == "schema" && sf.Extract != "" {
						field.Origin = "derived"
					}
				} else if sf.Default != "" {
					field.Value = sf.Default
				}
				field.Source = sf.Extract
				field.DefinedIn = sf.Source
			}
			if exprVal, ok := effective.Derive[name]; ok {
				if exprStr, ok := exprVal.(string); ok {
					field.Expression = exprStr
				}
			} else if exprVal, ok := effective.Aggregate[name]; ok {
				field.Origin = "aggregate"
				if exprStr, ok := exprVal.(string); ok {
					field.Expression = exprStr
				}
			}
		}
		fields = append(fields, field)
	}

	// Validation errors.
	var explainErrs []ExplainError
	for _, ve := range valErrs {
		explainErrs = append(explainErrs, ExplainError(ve))
	}

	return &ExplainResult{
		Version:    1,
		Kind:       "rootline/explain",
		Path:       path,
		StemChain:  chain,
		Layers:     layers,
		Provenance: provenance,
		Fields:     fields,
		Errors:     explainErrs,
	}, nil
}

func explainFieldNames(record *extract.Record, effective *StemFile) []string {
	seen := make(map[string]bool)
	if record != nil {
		for name := range record.Frontmatter {
			seen[name] = true
		}
		for name := range record.Derived {
			seen[name] = true
		}
	}
	if effective != nil {
		for name := range effective.Schema {
			seen[name] = true
		}
		for name := range effective.Derive {
			seen[name] = true
		}
		for name := range effective.Aggregate {
			seen[name] = true
		}
	}
	keys := make([]string, 0, len(seen))
	for name := range seen {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	return keys
}

func explainOriginFromRecord(record *extract.Record, name string) string {
	if record != nil {
		if _, ok := record.Frontmatter[name]; ok {
			return "frontmatter"
		}
		if _, ok := record.Derived[name]; ok {
			return "derived"
		}
	}
	return "schema"
}

func sortedMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
