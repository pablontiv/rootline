// Package fix implements the fix engine for rootline.
//
// It applies proposal-based fixes to records: extending enums in .stem files,
// migrating values, correcting typos, setting missing fields, correcting links,
// and rewriting frontmatter in markdown files.
package fix

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/proposal"
	"github.com/pablontiv/rootline/internal/rules"
	"gopkg.in/yaml.v3"
)

// ApplyProposals applies all proposals from a report to the filesystem.
// It modifies .stem files for extend_enum proposals and record files for
// data-level proposals. The report's Proposals slice is updated to reflect
// only the proposals that were actually applied.
func ApplyProposals(_ context.Context, report *proposal.Report, root string, records []*extract.Record) error {
	// Build record map for quick lookup.
	recordMap := make(map[string]*extract.Record)
	for _, rec := range records {
		recordMap[rec.Path] = rec
	}

	// Apply extend_enum proposals first (they modify .stem).
	for _, p := range report.Proposals {
		if p.Type != proposal.ExtendEnum {
			continue
		}
		if err := applyExtendEnum(p, root); err != nil {
			return fmt.Errorf("extend_enum %s: %w", p.Field, err)
		}
	}

	// Re-read stem after extend_enum to get updated enum values.
	var freshStem *rules.StemFile
	freshEntries, freshErr := rules.WalkUp(root)
	if freshErr == nil {
		freshStem = rules.MergeStemFiles(freshEntries)
	}

	// Apply data-level proposals (modify frontmatter/body of individual files).
	// Track which proposals are actually applied so reporting is accurate.
	var applied []proposal.Proposal
	for _, p := range report.Proposals {
		switch p.Type {
		case proposal.ExtendEnum:
			applied = append(applied, p) // already applied above
			continue
		case proposal.MigrateValue:
			if err := applyMigrateValue(p, root, recordMap); err != nil {
				return fmt.Errorf("migrate_value %s: %w", p.Paths[0], err)
			}
			applied = append(applied, p)
		case proposal.CorrectValue:
			// Skip if extend_enum made the original value valid.
			if freshStem != nil {
				if sf, ok := freshStem.Schema[p.Field]; ok && sf.Type == "enum" {
					nowValid := false
					for _, v := range sf.Values {
						if v == p.From {
							nowValid = true
							break
						}
					}
					if nowValid {
						continue
					}
				}
			}
			if err := applyCorrectValue(p, root, recordMap); err != nil {
				return fmt.Errorf("correct_value %s: %w", p.Paths[0], err)
			}
			applied = append(applied, p)
		case proposal.ExtractBody, proposal.InferFromChildren, proposal.AddField:
			if err := applySetField(p, root, recordMap); err != nil {
				return fmt.Errorf("%s %s: %w", p.Type, p.Paths[0], err)
			}
			applied = append(applied, p)
		case proposal.CorrectLink:
			if err := applyCorrectLink(p, root); err != nil {
				return fmt.Errorf("correct_link %s: %w", p.Paths[0], err)
			}
			applied = append(applied, p)
		case proposal.AddAggregate:
			if len(p.Paths) > 0 {
				if err := addAggregateToStem(p.Paths[0], p.Field, p.AggregateExpr); err != nil {
					return fmt.Errorf("add_aggregate %s: %w", p.Field, err)
				}
			}
			applied = append(applied, p)
		}
	}

	// Update report to reflect only applied proposals.
	report.Proposals = applied
	return nil
}

// ApplyFixes modifies the record's frontmatter in-place based on validation errors.
func ApplyFixes(_ context.Context, record *extract.Record, effective *rules.StemFile, errs []rules.ValidationError) (added []string, corrected []string) {
	if effective == nil {
		return
	}

	for _, e := range errs {
		field := e.Field
		sf, exists := effective.Schema[field]
		if !exists {
			continue
		}

		switch e.Rule {
		case "required":
			// Add missing required field with default or first enum value
			value := sf.Default
			if value == "" && len(sf.Values) > 0 {
				value = sf.Values[0]
			}
			record.Frontmatter[field] = value
			added = append(added, fmt.Sprintf("%s=%q", field, value))

		case "enum":
			// Correct invalid enum value to closest match
			if currentVal, ok := record.Frontmatter[field]; ok {
				current := fmt.Sprintf("%v", currentVal)
				closest := ClosestMatch(current, sf.Values)
				if closest != "" {
					record.Frontmatter[field] = closest
					corrected = append(corrected, fmt.Sprintf("%s: %q -> %q", field, current, closest))
				}
			}
		}
	}
	return
}

// RewriteFrontmatter rebuilds a markdown file with updated frontmatter.
func RewriteFrontmatter(original string, fm map[string]any) string {
	// Find frontmatter boundaries
	if !strings.HasPrefix(original, "---\n") {
		// No frontmatter — prepend it
		var b strings.Builder
		b.WriteString("---\n")
		WriteFrontmatterFields(&b, fm)
		b.WriteString("---\n")
		b.WriteString(original)
		return b.String()
	}

	endIdx := strings.Index(original[4:], "\n---\n")
	if endIdx == -1 {
		return original
	}
	body := original[4+endIdx+5:]

	var b strings.Builder
	b.WriteString("---\n")
	WriteFrontmatterFields(&b, fm)
	b.WriteString("---\n")
	b.WriteString(body)
	return b.String()
}

// WriteFrontmatterFields writes frontmatter fields to a builder in sorted order.
func WriteFrontmatterFields(b *strings.Builder, fm map[string]any) {
	keys := make([]string, 0, len(fm))
	for k := range fm {
		keys = append(keys, k)
	}
	// Sort for deterministic output
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	for _, k := range keys {
		valBytes, err := yaml.Marshal(fm[k])
		if err != nil {
			fmt.Fprintf(b, "%s: %q\n", k, fmt.Sprintf("%v", fm[k]))
			continue
		}
		val := strings.TrimSuffix(string(valBytes), "\n")
		if strings.Contains(val, "\n") {
			fmt.Fprintf(b, "%s:\n", k)
			for _, line := range strings.Split(val, "\n") {
				fmt.Fprintf(b, "  %s\n", line)
			}
		} else {
			fmt.Fprintf(b, "%s: %s\n", k, val)
		}
	}
}

// ClosestMatch finds the closest string by Levenshtein distance.
func ClosestMatch(s string, candidates []string) string {
	if len(candidates) == 0 {
		return ""
	}

	best := candidates[0]
	bestDist := levenshtein(strings.ToLower(s), strings.ToLower(best))

	for _, c := range candidates[1:] {
		d := levenshtein(strings.ToLower(s), strings.ToLower(c))
		if d < bestDist {
			bestDist = d
			best = c
		}
	}
	return best
}

// InsertWikiLinksBeforeHeading inserts wiki-links before the first ## heading.
func InsertWikiLinksBeforeHeading(content string, links []string) string {
	linkBlock := strings.Join(links, "\n") + "\n\n"
	// Find the first ## heading.
	idx := strings.Index(content, "\n## ")
	if idx >= 0 {
		return content[:idx+1] + linkBlock + content[idx+1:]
	}
	// If no ## heading, append at end.
	return content + "\n" + linkBlock
}

// levenshtein computes the edit distance between two strings.
func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	prev := make([]int, lb+1)
	curr := make([]int, lb+1)

	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(curr[j-1]+1, min(prev[j]+1, prev[j-1]+cost))
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

func applyExtendEnum(p proposal.Proposal, root string) error {
	// Find the .stem file that contains this field.
	stemPath := filepath.Join(root, ".stem")
	// Walk up from root to find the nearest .stem with this field.
	entries, err := rules.WalkUp(root)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.Stem != nil {
			if sf, ok := e.Stem.Schema[p.Field]; ok && sf.Type == "enum" {
				stemPath = e.Path
				break
			}
		}
	}

	content, err := os.ReadFile(stemPath)
	if err != nil {
		return err
	}

	// Parse YAML, add value to the enum's values list, rewrite.
	var doc yaml.Node
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return fmt.Errorf("parsing %s: %w", stemPath, err)
	}

	if len(doc.Content) == 0 {
		return fmt.Errorf("empty .stem document")
	}

	rootNode := doc.Content[0]
	if err := addEnumValueToNode(rootNode, p.Field, p.Value); err != nil {
		return err
	}

	out, err := yaml.Marshal(&doc)
	if err != nil {
		return err
	}
	return os.WriteFile(stemPath, out, 0644)
}

func addEnumValueToNode(node *yaml.Node, field, value string) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("expected mapping node")
	}

	// Find "schema" key.
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == "schema" {
			schemaNode := node.Content[i+1]
			if schemaNode.Kind != yaml.MappingNode {
				continue
			}
			// Find the field within schema.
			for j := 0; j+1 < len(schemaNode.Content); j += 2 {
				if schemaNode.Content[j].Value == field {
					fieldNode := schemaNode.Content[j+1]
					if fieldNode.Kind != yaml.MappingNode {
						continue
					}
					// Find "values" within the field.
					for k := 0; k+1 < len(fieldNode.Content); k += 2 {
						if fieldNode.Content[k].Value == "values" {
							valuesNode := fieldNode.Content[k+1]
							if valuesNode.Kind != yaml.SequenceNode {
								continue
							}
							// Add the new value.
							valuesNode.Content = append(valuesNode.Content, &yaml.Node{
								Kind:  yaml.ScalarNode,
								Tag:   "!!str",
								Value: value,
							})
							return nil
						}
					}
				}
			}
		}
	}
	return fmt.Errorf("field %q not found in schema", field)
}

func applyMigrateValue(p proposal.Proposal, root string, recordMap map[string]*extract.Record) error {
	for _, path := range p.Paths {
		rec, ok := recordMap[path]
		if !ok {
			continue
		}
		rec.Frontmatter[p.Field] = p.To

		absPath := filepath.Join(root, path)
		content, err := os.ReadFile(absPath)
		if err != nil {
			return err
		}

		newContent := RewriteFrontmatter(string(content), rec.Frontmatter)

		// Insert wiki-links before first heading if any.
		if len(p.WikiLinks) > 0 {
			newContent = InsertWikiLinksBeforeHeading(newContent, p.WikiLinks)
		}

		if err := os.WriteFile(absPath, []byte(newContent), 0644); err != nil {
			return err
		}
	}
	return nil
}

func applyCorrectValue(p proposal.Proposal, root string, recordMap map[string]*extract.Record) error {
	for _, path := range p.Paths {
		rec, ok := recordMap[path]
		if !ok {
			continue
		}
		rec.Frontmatter[p.Field] = p.To
		if err := rewriteRecordFile(root, path, rec.Frontmatter); err != nil {
			return err
		}
	}
	return nil
}

func applySetField(p proposal.Proposal, root string, recordMap map[string]*extract.Record) error {
	value := p.Value
	if value == "" {
		value = p.To
	}
	for _, path := range p.Paths {
		rec, ok := recordMap[path]
		if !ok {
			continue
		}
		rec.Frontmatter[p.Field] = value
		if err := rewriteRecordFile(root, path, rec.Frontmatter); err != nil {
			return err
		}
	}
	return nil
}

func rewriteRecordFile(root, path string, fm map[string]any) error {
	absPath := filepath.Join(root, path)
	content, err := os.ReadFile(absPath)
	if err != nil {
		return err
	}
	newContent := RewriteFrontmatter(string(content), fm)
	return os.WriteFile(absPath, []byte(newContent), 0644)
}

func applyCorrectLink(p proposal.Proposal, root string) error {
	for _, path := range p.Paths {
		absPath := filepath.Join(root, path)
		content, err := os.ReadFile(absPath)
		if err != nil {
			return err
		}
		newContent := strings.Replace(string(content), p.From, p.To, 1)
		if err := os.WriteFile(absPath, []byte(newContent), 0644); err != nil {
			return err
		}
	}
	return nil
}

// addAggregateToStem adds an aggregate expression to a .stem file using YAML AST.
func addAggregateToStem(stemPath, fieldName, expr string) error {
	content, err := os.ReadFile(stemPath)
	if err != nil {
		return err
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return fmt.Errorf("parsing %s: %w", stemPath, err)
	}

	if len(doc.Content) == 0 {
		return fmt.Errorf("empty .stem document")
	}

	rootNode := doc.Content[0]
	if rootNode.Kind != yaml.MappingNode {
		return fmt.Errorf("expected mapping node in .stem")
	}

	// Find existing "aggregate" key.
	var aggNode *yaml.Node
	for i := 0; i+1 < len(rootNode.Content); i += 2 {
		if rootNode.Content[i].Value == "aggregate" {
			aggNode = rootNode.Content[i+1]
			break
		}
	}

	if aggNode == nil {
		// Create new aggregate section.
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "aggregate"}
		aggNode = &yaml.Node{Kind: yaml.MappingNode}
		rootNode.Content = append(rootNode.Content, keyNode, aggNode)
	}

	// Add field to aggregate mapping.
	fieldKeyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: fieldName}
	fieldValNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: expr, Style: yaml.LiteralStyle}
	aggNode.Content = append(aggNode.Content, fieldKeyNode, fieldValNode)

	out, err := yaml.Marshal(&doc)
	if err != nil {
		return err
	}
	return os.WriteFile(stemPath, out, 0644)
}
