package migrate

import (
	"fmt"
	"strings"

	"github.com/pablontiv/rootline/internal/rules"
)

// GenerateAggregateExpr produces an aggregate expression for a single enum field.
// The expression uses ternary chains with all()/any() over descendants.
// Returns "" for non-enum fields.
func GenerateAggregateExpr(fieldName string, sf rules.SchemaField) string {
	if sf.Type != "enum" || len(sf.Values) == 0 {
		return ""
	}

	if len(sf.Values) == 1 {
		return fmt.Sprintf("%q", sf.Values[0])
	}

	classified := classifyValues(sf.Values)

	var lines []string

	// Terminal values use all() — "everything is done".
	for _, v := range classified.terminal {
		lines = append(lines, fmt.Sprintf(
			"all(descendants, {.%s == %q}) ? %q",
			fieldName, v, v,
		))
	}

	// Negative values use any() — high priority.
	for _, v := range classified.negative {
		lines = append(lines, fmt.Sprintf(
			"any(descendants, {.%s == %q}) ? %q",
			fieldName, v, v,
		))
	}

	// Active values use any().
	for _, v := range classified.active {
		lines = append(lines, fmt.Sprintf(
			"any(descendants, {.%s == %q}) ? %q",
			fieldName, v, v,
		))
	}

	// Neutral values use any() — except the last which becomes default.
	for _, v := range classified.neutral {
		lines = append(lines, fmt.Sprintf(
			"any(descendants, {.%s == %q}) ? %q",
			fieldName, v, v,
		))
	}

	// The default is the lowest-priority classified value.
	defaultVal := sf.Values[0]
	switch {
	case len(classified.neutral) > 0:
		defaultVal = classified.neutral[0]
	case len(classified.active) > 0:
		defaultVal = classified.active[0]
	case len(classified.negative) > 0:
		defaultVal = classified.negative[0]
	case len(classified.terminal) > 0:
		defaultVal = classified.terminal[0]
	}

	// Remove the last any() line that matches the default — it becomes the fallback.
	// Find the line that matches the default value.
	defaultLine := fmt.Sprintf(
		"any(descendants, {.%s == %q}) ? %q",
		fieldName, defaultVal, defaultVal,
	)
	filteredLines := make([]string, 0, len(lines))
	for _, l := range lines {
		if l != defaultLine {
			filteredLines = append(filteredLines, l)
		}
	}

	// Build multi-line expression.
	var b strings.Builder
	for i, line := range filteredLines {
		if i > 0 {
			b.WriteString(" :\n")
		}
		b.WriteString(line)
	}
	if len(filteredLines) > 0 {
		b.WriteString(" :\n")
	}
	fmt.Fprintf(&b, "%q", defaultVal)

	return b.String()
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

// classification groups enum values by their semantic role.
type classification struct {
	terminal []string
	negative []string
	active   []string
	neutral  []string
}

// classifyValues sorts enum values into semantic classes using keyword matching.
// If no keywords match any value, falls back to positional: last = terminal, first = default.
func classifyValues(values []string) classification {
	var c classification

	terminalKW := []string{"completed", "completado", "done", "closed", "obsolete", "obsoleto"}
	negativeKW := []string{"blocked", "bloqueada", "hold", "diferida", "paused"}
	activeKW := []string{"in progress", "en progreso", "active"}

	anyMatched := false
	for _, v := range values {
		lower := strings.ToLower(v)
		switch {
		case containsAny(lower, terminalKW):
			c.terminal = append(c.terminal, v)
			anyMatched = true
		case containsAny(lower, negativeKW):
			c.negative = append(c.negative, v)
			anyMatched = true
		case containsAny(lower, activeKW):
			c.active = append(c.active, v)
			anyMatched = true
		default:
			c.neutral = append(c.neutral, v)
		}
	}

	// Fallback: no keywords matched at all → positional.
	if !anyMatched {
		c.terminal = []string{values[len(values)-1]}
		c.neutral = values[:len(values)-1]
	}

	return c
}

// containsAny reports whether s contains any of the keywords.
func containsAny(s string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}
