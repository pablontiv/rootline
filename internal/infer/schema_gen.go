package infer

import (
	"context"
	"fmt"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

// InferOptions controls schema generation behavior.
type InferOptions struct {
	// SectionThreshold is the minimum frequency (0.0-1.0) for a section to be
	// considered required. Default is 0.80 for init mode (prescriptive).
	SectionThreshold float64
	// IncludeStructural enables structural rule detection (require_index, min/max_children).
	IncludeStructural bool
}

// DefaultInferOptions returns options suitable for init command.
func DefaultInferOptions() InferOptions {
	return InferOptions{
		SectionThreshold:  0.80,
		IncludeStructural: true,
	}
}

// GenerateFlatSchema generates a flat .stem schema from observed frontmatter
// without writing any file. Returns a StemFile that combines:
// - inferred schema from frontmatter (Analyze)
// - detected section patterns
// - structural rules
func GenerateFlatSchema(ctx context.Context, dir string, records []*extract.Record, opts InferOptions) (*rules.StemFile, error) {
	if len(records) == 0 {
		return nil, fmt.Errorf("no records to analyze")
	}

	// Analyze frontmatter
	schema := Analyze(records)

	// Detect section patterns
	sectionInferences := DetectSectionPatterns(records, opts.SectionThreshold)
	for _, inf := range sectionInferences {
		fieldName := sectionFieldName(inf.Field)
		heading := "## " + inf.Field
		sf := rules.SchemaField{
			Type:    "section",
			Heading: heading,
		}
		if inf.Type == "required_section" {
			sf.Required = true
		}
		schema.Schema[fieldName] = sf
	}

	// Detect structural rules if enabled
	var structuralRules rules.StructuralRules
	if opts.IncludeStructural {
		structuralRules = generateStructuralRules(dir)
	}

	// Build StemFile
	stem := &rules.StemFile{
		Version: 2,
		Scope: rules.Scope{
			Match: "*.md",
		},
		Schema:     schema.Schema,
		Structural: structuralRules,
	}

	return stem, nil
}

// GenerateHierarchicalSchema detects hierarchical directory patterns and generates
// per-level schema candidates without writing any file. Returns a map of path patterns
// to StemFiles.
func GenerateHierarchicalSchema(ctx context.Context, dir string, records []*extract.Record, opts InferOptions) (map[string]*rules.StemFile, error) {
	if len(records) == 0 {
		return nil, fmt.Errorf("no records to analyze")
	}

	// Analyze hierarchy
	hierarchy := AnalyzeHierarchy(records, dir)
	if !hierarchy.Detected || len(hierarchy.Levels) < 2 {
		return nil, fmt.Errorf("hierarchical pattern not detected (need at least 2 levels)")
	}

	// Generate aggregate expressions for root schema enum fields
	aggregates := generateAggregateExpressions(hierarchy.Root.Schema)

	// Build root StemFile
	rootStem := buildRootStemFile(hierarchy, aggregates, dir, opts)

	// Return root .stem keyed by "."
	result := make(map[string]*rules.StemFile)
	result["."] = rootStem

	return result, nil
}

// buildRootStemFile constructs the root .stem with match-based per-level fields
// and aggregates.
func buildRootStemFile(hierarchy *HierarchyResult, aggregates map[string]string, dir string, opts InferOptions) *rules.StemFile {
	matchSchema := hierarchy.ToMatchSchema()

	// Prepare derive and aggregate maps
	deriveMap := make(map[string]any)
	aggregateMap := make(map[string]any)

	// Convert aggregate expressions to map[string]any
	for k, v := range aggregates {
		aggregateMap[k] = v
	}

	// Detect structural rules if enabled
	var structuralRules rules.StructuralRules
	if opts.IncludeStructural {
		structuralRules = generateStructuralRules(dir)
	}

	return &rules.StemFile{
		Version: 2,
		Scope: rules.Scope{
			Match: "*.md",
		},
		Schema:     matchSchema,
		Derive:     deriveMap,
		Aggregate:  aggregateMap,
		Structural: structuralRules,
	}
}

// generateStructuralRules detects structural constraints from a directory.
func generateStructuralRules(dir string) rules.StructuralRules {
	inferences := DetectStructural(dir)
	if len(inferences) == 0 {
		return rules.StructuralRules{}
	}

	var subdirs rules.SubdirRules
	for _, inf := range inferences {
		if inf.Type != "add_structural_rule" {
			continue
		}
		switch inf.Field {
		case "require_index":
			subdirs.RequireIndex = inf.Value
		case "min_children":
			_, _ = fmt.Sscanf(inf.Value, "%d", &subdirs.MinChildren)
		case "max_children":
			_, _ = fmt.Sscanf(inf.Value, "%d", &subdirs.MaxChildren)
		}
	}

	return rules.StructuralRules{Subdirs: subdirs}
}

// generateAggregateExpressions produces aggregate expressions for all enum fields
// in the root schema. This replicates the logic from migrate.GenerateAggregates.
func generateAggregateExpressions(rootSchema map[string]rules.SchemaField) map[string]string {
	result := make(map[string]string)
	for fieldName, sf := range rootSchema {
		expr := generateAggregateExpr(fieldName, sf)
		if expr != "" {
			result[fieldName] = expr
		}
	}
	return result
}

// generateAggregateExpr produces an aggregate expression for a single enum field.
// The expression uses ternary chains with all()/any() over descendants.
// Returns "" for non-enum fields.
func generateAggregateExpr(fieldName string, sf rules.SchemaField) string {
	if sf.Type != "enum" || len(sf.Values) == 0 {
		return ""
	}

	if len(sf.Values) == 1 {
		return fmt.Sprintf("%q", sf.Values[0])
	}

	classified := classifyEnumValues(sf.Values)

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

// enumClassification groups enum values by their semantic role.
type enumClassification struct {
	terminal []string
	negative []string
	active   []string
	neutral  []string
}

// classifyEnumValues sorts enum values into semantic classes using keyword matching.
// If no keywords match any value, falls back to positional: last = terminal, first = default.
func classifyEnumValues(values []string) enumClassification {
	var c enumClassification

	terminalKW := []string{"completed", "completado", "done", "closed", "obsolete", "obsoleto"}
	negativeKW := []string{"blocked", "bloqueada", "hold", "diferida", "paused"}
	activeKW := []string{"in progress", "en progreso", "active"}

	anyMatched := false
	for _, v := range values {
		lower := strings.ToLower(v)
		switch {
		case containsKeyword(lower, terminalKW):
			c.terminal = append(c.terminal, v)
			anyMatched = true
		case containsKeyword(lower, negativeKW):
			c.negative = append(c.negative, v)
			anyMatched = true
		case containsKeyword(lower, activeKW):
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

// containsKeyword reports whether s contains any of the keywords.
func containsKeyword(s string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}

// sectionFieldName converts a heading text to a .stem field name:
// lowercase with spaces replaced by underscores.
func sectionFieldName(heading string) string {
	return strings.ToLower(strings.ReplaceAll(heading, " ", "_"))
}
