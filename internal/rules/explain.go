package rules

import (
	"sort"

	"github.com/pablontiv/rootline/internal/extract"
)

// ExplainResult is the versioned JSON output for the explain command and MCP tool.
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
	Expression string `json:"expression,omitempty"`
}

// ExplainError traces a validation error to its source rule.
type ExplainError struct {
	Rule       string `json:"rule"`
	Field      string `json:"field"`
	Message    string `json:"message"`
	Source     string `json:"source"`
	Severity   string `json:"severity"`
	Suggestion string `json:"suggestion,omitempty"`
}

// NewExplainResult builds an ExplainResult from walk-up entries,
// the effective StemFile, and validation errors for a given path.
func NewExplainResult(
	path string,
	entries []StemEntry,
	effective *StemFile,
	record *extract.Record,
	valErrs []ValidationError,
) *ExplainResult {
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
	if entries != nil {
		for _, entry := range entries {
			if entry.Stem != nil {
				for name := range entry.Stem.Schema {
					// Last writer wins — later entries (closer to leaf) override earlier ones.
					provenance[name] = entry.Path
				}
			}
		}
	}

	var fields []ExplainField

	// Frontmatter fields.
	fmKeys := sortedMapKeys(record.Frontmatter)
	for _, k := range fmKeys {
		f := ExplainField{
			Name:   k,
			Value:  record.Frontmatter[k],
			Origin: "frontmatter",
		}
		// Check if this field is defined in schema.
		if effective != nil {
			if sf, ok := effective.Schema[k]; ok {
				f.Source = sf.Source
			}
		}
		fields = append(fields, f)
	}

	// Schema fields not present in frontmatter (missing required, defaults).
	if effective != nil {
		for name, sf := range effective.Schema {
			if _, exists := record.Frontmatter[name]; exists {
				continue
			}
			f := ExplainField{
				Name:   name,
				Value:  nil,
				Origin: "schema",
				Source: sf.Source,
			}
			if sf.Default != "" {
				f.Value = sf.Default
			}
			fields = append(fields, f)
		}
	}

	// Derived fields (from derive and aggregate).
	if effective != nil {
		derivedKeys := sortedMapKeys(record.Derived)
		for _, k := range derivedKeys {
			f := ExplainField{
				Name:   k,
				Value:  record.Derived[k],
				Origin: "derived",
			}
			if exprVal, ok := effective.Derive[k]; ok {
				if exprStr, ok := exprVal.(string); ok {
					f.Expression = exprStr
				}
			} else if exprVal, ok := effective.Aggregate[k]; ok {
				f.Origin = "aggregate"
				if exprStr, ok := exprVal.(string); ok {
					f.Expression = exprStr
				}
			}
			fields = append(fields, f)
		}
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
	}
}

func sortedMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
