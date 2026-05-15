package migrate

import (
	"fmt"

	"github.com/pablontiv/rootline/internal/rules"
)

// GenerateAggregateExpr generates a field-agnostic aggregate expression.
// Without semantic keyword matching, we can only construct simple positional expressions.
// Returns "" for non-enum fields.
// This is a simplified version that does not infer semantic meaning from value names.
func GenerateAggregateExpr(fieldName string, sf rules.SchemaField) string {
	if sf.Type != "enum" || len(sf.Values) == 0 {
		return ""
	}

	if len(sf.Values) == 1 {
		return fmt.Sprintf("%q", sf.Values[0])
	}

	// Without semantic classification, we use the first value as default
	// and build a simple expression. Proper aggregate logic must be
	// explicitly configured in .stem files by the user.
	defaultVal := sf.Values[0]

	// Return a simple positional fallback (no semantic inference)
	return fmt.Sprintf("%q", defaultVal)
}

// GenerateAggregates produces aggregate expressions for all enum fields in a schema
// that don't already have an existing aggregate.
func GenerateAggregates(rootSchema map[string]rules.SchemaField, existingAgg map[string]any) map[string]string {
	result := make(map[string]string)
	for fieldName, sf := range rootSchema {
		if existingAgg != nil {
			if _, exists := existingAgg[fieldName]; exists {
				continue
			}
		}
		expr := GenerateAggregateExpr(fieldName, sf)
		if expr != "" {
			result[fieldName] = expr
		}
	}
	return result
}
