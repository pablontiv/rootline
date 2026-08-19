package infer

import (
	"context"
	"fmt"
	"sort"
	"strconv"
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
	sectionInferences, err := DetectSectionPatterns(records, opts.SectionThreshold)
	if err != nil {
		return nil, err
	}
	if err := addSectionInferenceFields(schema.Schema, sectionInferences, "frontmatter"); err != nil {
		return nil, err
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
	sectionInferences, err := DetectSectionPatterns(records, opts.SectionThreshold)
	if err != nil {
		return nil, err
	}
	if err := addSectionInferenceFields(rootStem.Schema, sectionInferences, "hierarchy"); err != nil {
		return nil, err
	}

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

// generateAggregateExpr generates a field-agnostic aggregate expression.
// Without semantic keyword matching, we can only construct simple positional expressions.
// Returns "" for non-enum fields.
// This is a simplified version that does not infer semantic meaning from value names.
func generateAggregateExpr(fieldName string, sf rules.SchemaField) string {
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

func addSectionInferenceFields(schema map[string]rules.SchemaField, inferences []Inference, existingOrigin string) error {
	if strings.TrimSpace(existingOrigin) == "" {
		existingOrigin = "schema"
	}
	var collisions []string
	for _, inf := range inferences {
		if inf.Type != "required_section" && inf.Type != "optional_section" {
			continue
		}
		if inf.Field == "" {
			return fmt.Errorf("section inference missing field")
		}
		if inf.SourceDirective == "" {
			return fmt.Errorf("section inference for %q missing source directive", inf.Field)
		}
		if _, exists := schema[inf.Field]; exists {
			collisions = append(collisions, fmt.Sprintf("field %q from %s collides with body section source %s", inf.Field, existingOrigin, strconv.Quote(inf.SourceDirective)))
		}
	}
	if len(collisions) > 0 {
		sort.Strings(collisions)
		return fmt.Errorf("section field composition collision: %s", strings.Join(collisions, "; "))
	}

	for _, inf := range inferences {
		if inf.Type != "required_section" && inf.Type != "optional_section" {
			continue
		}
		schema[inf.Field] = rules.SchemaField{
			Type:     "string",
			Required: inf.Type == "required_section",
			Extract:  inf.SourceDirective,
		}
	}
	return nil
}
