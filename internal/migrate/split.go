package migrate

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pablontiv/rootline/internal/infer"
	"github.com/pablontiv/rootline/internal/rules"
	"gopkg.in/yaml.v3"
)

// StemOutput represents a generated .stem file with its target path and content.
type StemOutput struct {
	Path    string
	Content string
}

// SplitResult holds the generated stems and any auto-generated aggregate field names.
type SplitResult struct {
	Stems         []StemOutput
	GeneratedAggs []string // sorted field names with newly generated aggregates
}

// BuildSplitStems distributes fields from existing .stem across hierarchy levels.
// Derive, aggregate, links, structural, and validate stay at root.
// Auto-generates aggregate expressions for root enum fields not already in existing.Aggregate.
func BuildSplitStems(absTarget string, existing *rules.StemFile, hierarchy *infer.HierarchyResult) SplitResult {
	var files []StemOutput

	// Determine which fields from existing schema belong at root vs per-level.
	rootFields := make(map[string]rules.SchemaField)
	levelFields := make(map[int]map[string]rules.SchemaField)

	for i := range hierarchy.Levels {
		levelFields[i] = make(map[string]rules.SchemaField)
	}

	for name, sf := range existing.Schema {
		if name == "id" {
			continue // handled per-level via sequence detection
		}

		// Check if this field is in root (all levels) or specific levels.
		if _, inRoot := hierarchy.Root.Schema[name]; inRoot {
			rootFields[name] = sf
		} else {
			// Assign to levels that have it in OnlyHere.
			for i, ls := range hierarchy.Levels {
				if _, ok := ls.OnlyHere[name]; ok {
					levelFields[i][name] = sf
				}
			}
		}
	}

	// Generate aggregate expressions for root enum fields without existing aggregate.
	generatedAgg := GenerateAggregates(rootFields, existing.Aggregate)
	aggNames := make([]string, 0, len(generatedAgg))
	for k := range generatedAgg {
		aggNames = append(aggNames, k)
	}
	sort.Strings(aggNames)

	// Build root .stem YAML preserving derive/aggregate/links/structural.
	rootYAML := buildSplitRootYAML(existing, rootFields, hierarchy, generatedAgg)
	files = append(files, StemOutput{
		Path:    filepath.Join(absTarget, ".stem"),
		Content: rootYAML,
	})

	// Build child .stem files.
	for i := 1; i < len(hierarchy.Levels); i++ {
		ls := hierarchy.Levels[i]
		prevLevel := hierarchy.Levels[i-1]

		childYAML := buildSplitChildYAML(&ls, levelFields[i])

		for _, parentDir := range prevLevel.Level.DirPaths {
			files = append(files, StemOutput{
				Path:    filepath.Join(absTarget, parentDir, ".stem"),
				Content: childYAML,
			})
		}
	}

	return SplitResult{Stems: files, GeneratedAggs: aggNames}
}

// buildSplitRootYAML generates the root .stem preserving non-schema sections.
func buildSplitRootYAML(existing *rules.StemFile, rootFields map[string]rules.SchemaField, hierarchy *infer.HierarchyResult, generatedAgg map[string]string) string {
	var b strings.Builder

	b.WriteString("version: 1\n")
	if existing.Scope.Match != "" {
		fmt.Fprintf(&b, "scope:\n  match: %q\n", existing.Scope.Match)
	}

	b.WriteString("\nschema:\n")

	// Add first level's sequence id.
	if len(hierarchy.Levels) > 0 {
		if idField, ok := hierarchy.Levels[0].OnlyHere["id"]; ok {
			b.WriteString("  id:\n")
			fmt.Fprintf(&b, "    type: %s\n", idField.Type)
			if idField.Prefix != "" {
				fmt.Fprintf(&b, "    prefix: %s\n", idField.Prefix)
			}
			if idField.Digits > 0 {
				fmt.Fprintf(&b, "    digits: %d\n", idField.Digits)
			}
		}
	}

	// Add root fields sorted.
	keys := make([]string, 0, len(rootFields))
	for k := range rootFields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, name := range keys {
		sf := rootFields[name]
		fmt.Fprintf(&b, "  %s:\n", name)
		fmt.Fprintf(&b, "    type: %s\n", sf.Type)
		if sf.Required {
			b.WriteString("    required: true\n")
		}
		if len(sf.Values) > 0 {
			fmt.Fprintf(&b, "    values: [%s]\n", strings.Join(sf.Values, ", "))
		}
		if sf.Severity != "" {
			fmt.Fprintf(&b, "    severity: %s\n", sf.Severity)
		}
	}

	// Preserve structural.
	if existing.Structural.Subdirs.RequireIndex != "" || existing.Structural.Subdirs.MinChildren > 0 {
		b.WriteString("\nstructural:\n  subdirs:\n")
		if existing.Structural.Subdirs.RequireIndex != "" {
			fmt.Fprintf(&b, "    require_index: %s\n", existing.Structural.Subdirs.RequireIndex)
		}
		if existing.Structural.Subdirs.MinChildren > 0 {
			fmt.Fprintf(&b, "    min_children: %d\n", existing.Structural.Subdirs.MinChildren)
		}
		if existing.Structural.Subdirs.Severity != "" {
			fmt.Fprintf(&b, "    severity: %s\n", existing.Structural.Subdirs.Severity)
		}
	}

	// Preserve links.
	if len(existing.Links.Allowed) > 0 {
		fmt.Fprintf(&b, "\nlinks:\n  allowed: [%s]\n", strings.Join(existing.Links.Allowed, ", "))
		for name, rule := range existing.Links.Rules {
			fmt.Fprintf(&b, "  %s:\n", name)
			if rule.Target != "" {
				fmt.Fprintf(&b, "    target: %q\n", rule.Target)
			}
			if rule.Field != "" {
				fmt.Fprintf(&b, "    field: %s\n", rule.Field)
			}
		}
	}

	// Preserve derive.
	if len(existing.Derive) > 0 {
		b.WriteString("\nderive:\n")
		for name, expr := range existing.Derive {
			exprStr := fmt.Sprintf("%v", expr)
			if strings.Contains(exprStr, "\n") {
				fmt.Fprintf(&b, "  %s: |\n    %s\n", name, strings.ReplaceAll(strings.TrimSpace(exprStr), "\n", "\n    "))
			} else {
				fmt.Fprintf(&b, "  %s: %q\n", name, exprStr)
			}
		}
	}

	// Preserve existing aggregate + emit generated ones.
	if len(existing.Aggregate) > 0 || len(generatedAgg) > 0 {
		b.WriteString("\naggregate:\n")
		for name, expr := range existing.Aggregate {
			exprStr := fmt.Sprintf("%v", expr)
			if strings.Contains(exprStr, "\n") {
				fmt.Fprintf(&b, "  %s: |\n    %s\n", name, strings.ReplaceAll(strings.TrimSpace(exprStr), "\n", "\n    "))
			} else {
				fmt.Fprintf(&b, "  %s: %q\n", name, exprStr)
			}
		}
		// Emit auto-generated aggregates (already excludes existing ones).
		aggKeys := make([]string, 0, len(generatedAgg))
		for k := range generatedAgg {
			aggKeys = append(aggKeys, k)
		}
		sort.Strings(aggKeys)
		for _, name := range aggKeys {
			fmt.Fprintf(&b, "  %s: |\n", name)
			for _, line := range strings.Split(generatedAgg[name], "\n") {
				fmt.Fprintf(&b, "    %s\n", line)
			}
		}
	}

	// Preserve validate.
	if len(existing.Validate) > 0 {
		b.WriteString("\nvalidate:\n")
		for _, v := range existing.Validate {
			entry := map[string]any{"rule": v.Rule}
			if v.Field != "" {
				entry["field"] = v.Field
			}
			if v.Severity != "" {
				entry["severity"] = v.Severity
			}
			if len(v.If) > 0 {
				entry["if"] = v.If
			}
			if len(v.Then) > 0 {
				entry["then"] = v.Then
			}
			bytes, _ := yaml.Marshal([]any{entry})
			indented := "  " + strings.ReplaceAll(strings.TrimSpace(string(bytes)), "\n", "\n  ")
			b.WriteString(indented + "\n")
		}
	}

	return b.String()
}

// buildSplitChildYAML generates a child .stem with level-specific overrides.
func buildSplitChildYAML(ls *infer.LevelSchema, extraFields map[string]rules.SchemaField) string {
	var b strings.Builder
	b.WriteString("schema:\n")

	// Sequence id.
	if idField, ok := ls.OnlyHere["id"]; ok {
		b.WriteString("  id:\n")
		fmt.Fprintf(&b, "    type: %s\n", idField.Type)
		if idField.Prefix != "" {
			fmt.Fprintf(&b, "    prefix: %s\n", idField.Prefix)
		}
		if idField.Digits > 0 {
			fmt.Fprintf(&b, "    digits: %d\n", idField.Digits)
		}
	}

	// Level-specific fields from existing schema.
	keys := make([]string, 0, len(extraFields))
	for k := range extraFields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, name := range keys {
		sf := extraFields[name]
		fmt.Fprintf(&b, "  %s:\n", name)
		fmt.Fprintf(&b, "    type: %s\n", sf.Type)
		if sf.Required {
			b.WriteString("    required: true\n")
		}
		if len(sf.Values) > 0 {
			fmt.Fprintf(&b, "    values: [%s]\n", strings.Join(sf.Values, ", "))
		}
		if sf.Severity != "" {
			fmt.Fprintf(&b, "    severity: %s\n", sf.Severity)
		}
	}

	return b.String()
}
