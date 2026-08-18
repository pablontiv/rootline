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

	"github.com/pablontiv/picokit/fuzzy"
	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/fsx"
	"github.com/pablontiv/rootline/internal/proposal"
	"github.com/pablontiv/rootline/internal/rules"
	"gopkg.in/yaml.v3"
)

// newFileMode is the mode every write in this package hands to
// fsx.WriteFileAtomic. It is a CREATE-NEW default and nothing more: when the
// target already exists, fsx keeps the mode the target already has and ignores
// this value entirely.
//
// It is named rather than repeated as a literal 0o644 because the bare literal
// reads like an instruction to set the mode, which is what it used to be. This
// package once carried its own atomic writer that applied the caller's mode
// unconditionally, so every rewrite widened a 0600 governed document to 0644.
// Rewriting a record's frontmatter is a content operation and must not change
// who may read the file.
const newFileMode = 0o644

// ApplyProposals applies repair-surface proposals from a report to the filesystem.
// Schema-surface proposals (extend_enum, add_aggregate, remove_stem_field) are
// already separated into SchemaSuggestions by the caller and are not processed here.
// The report's Proposals slice is updated to reflect only the proposals that were
// actually applied.
// Declined proposals are returned, not dropped: a silent skip reads as "nothing
// needed doing", the opposite of the truth.
func ApplyProposals(_ context.Context, report *proposal.Report, root string, records []*extract.Record, fillMissing bool) ([]proposal.Proposal, error) {
	// Build record map for quick lookup.
	recordMap := make(map[string]*extract.Record)
	for _, rec := range records {
		recordMap[rec.Path] = rec
	}

	// Apply only data-level proposals (pre-separated from schema proposals).
	var applied []proposal.Proposal
	var skipped []proposal.Proposal

	for _, p := range report.Proposals {
		// A value the engine picked because the schema declared none is not
		// data — writing it would erase the very signal the missing field was
		// raising. It is proposed for review, not applied, unless asked for.
		if p.Type == proposal.AddField && proposal.IsEngineChosen(p.ValueSource) && !fillMissing {
			skipped = append(skipped, p)
			continue
		}
		switch p.Type {
		case proposal.MigrateValue:
			if err := applyMigrateValue(p, root, recordMap); err != nil {
				return nil, fmt.Errorf("migrate_value %s: %w", p.Paths[0], err)
			}
			applied = append(applied, p)
		case proposal.CorrectValue:
			if err := applyCorrectValue(p, root, recordMap); err != nil {
				return nil, fmt.Errorf("correct_value %s: %w", p.Paths[0], err)
			}
			applied = append(applied, p)
		case proposal.ExtractBody, proposal.InferFromChildren, proposal.AddField, proposal.InferFromSiblings, proposal.SetField:
			if err := applySetField(p, root, recordMap); err != nil {
				return nil, fmt.Errorf("%s %s: %w", p.Type, p.Paths[0], err)
			}
			applied = append(applied, p)
		case proposal.SetSection:
			// fix --all builds proposals from scanned record paths, which are
			// already root-relative; absolute input is tolerated but still confined.
			if err := applySetSection(p, root, recordMap, PolicyAcceptAbsolute); err != nil {
				return nil, fmt.Errorf("set_section %s: %w", p.Heading, err)
			}
			applied = append(applied, p)
		case proposal.CorrectOutlier:
			if err := applyCorrectValue(p, root, recordMap); err != nil {
				return nil, fmt.Errorf("correct_outlier %s: %w", p.Paths[0], err)
			}
			applied = append(applied, p)
		case proposal.PropagateAggregate:
			if err := applyCorrectValue(p, root, recordMap); err != nil {
				return nil, fmt.Errorf("propagate_aggregate %s: %w", p.Paths[0], err)
			}
			applied = append(applied, p)
		case proposal.CorrectLink:
			if err := applyCorrectLink(p, root); err != nil {
				return nil, fmt.Errorf("correct_link %s: %w", p.Paths[0], err)
			}
			applied = append(applied, p)
		}
	}

	// Update report to reflect only applied proposals.
	report.Proposals = applied

	return skipped, nil
}

// ApplyFixes modifies the record's frontmatter in-place based on validation errors.
func ApplyFixes(_ context.Context, record *extract.Record, effective *rules.StemFile, errs []rules.ValidationError, fillMissing bool) (added []string, corrected []string, skipped []string) {
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
			// This path consumes no proposals, so provenance is read straight
			// off the schema: anything other than a declared default: is the
			// engine inventing a value.
			value, source := proposal.AddFieldValue(sf)
			if proposal.IsEngineChosen(source) && !fillMissing {
				skipped = append(skipped, fmt.Sprintf("%s (no default declared; pass --fill-missing to write %q)", field, value))
				continue
			}
			record.Frontmatter[field] = value
			added = append(added, fmt.Sprintf("%s=%q", field, value))

		case "enum":
			// Correct invalid enum value to closest match
			if currentVal, ok := record.Frontmatter[field]; ok {
				current := fmt.Sprintf("%v", currentVal)
				closest := fuzzy.Match(current, sf.Values)
				if closest != "" && closest != current {
					record.Frontmatter[field] = closest
					corrected = append(corrected, fmt.Sprintf("%s: %q -> %q", field, current, closest))
				}
			}
		}
	}
	return
}

// RewriteFrontmatter rebuilds a markdown file with updated frontmatter.
//
// Existing keys keep their original position, comments and scalar quoting style,
// so a one-field mutation yields a one-field diff. Keys the map introduces are
// appended after the existing ones; keys it omits are removed.
//
// The preserved guarantee is bounded by what a yaml.Node round-trip provides:
// yaml.v3 re-emits from the node tree rather than the source bytes, so
// inter-token whitespace and nested indentation are normalized. Frontmatter that
// does not parse as a YAML mapping falls back to the sorted, comment-less
// rebuild this function used previously.
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
	fmText := original[4 : 4+endIdx+1]
	body := original[4+endIdx+5:]

	if rendered, ok := renderPreservedFrontmatter(fmText, fm); ok {
		return "---\n" + rendered + "---\n" + body
	}

	var b strings.Builder
	b.WriteString("---\n")
	WriteFrontmatterFields(&b, fm)
	b.WriteString("---\n")
	b.WriteString(body)
	return b.String()
}

// WriteFrontmatterFields writes frontmatter fields to a builder in sorted order.
// It is the fallback path for frontmatter that RewriteFrontmatter cannot parse
// as a YAML mapping, and the writer used when a document has no frontmatter yet.
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

// ClosestMatch finds the closest string by Levenshtein distance within an adaptive threshold.
// Returns "" if no candidate is close enough.
func ClosestMatch(s string, candidates []string) string {
	return fuzzy.Match(s, candidates)
}

// InsertWikiLinksBeforeHeading inserts missing standalone wiki-link lines before the first ## heading.
func InsertWikiLinksBeforeHeading(content string, links []string) string {
	missing := missingStandaloneLinks(content, links)
	if len(missing) == 0 {
		return content
	}

	linkBlock := strings.Join(missing, "\n") + "\n\n"
	idx := strings.Index(content, "\n## ")
	if idx >= 0 {
		return content[:idx+1] + linkBlock + content[idx+1:]
	}
	return content + "\n" + linkBlock
}

func missingStandaloneLinks(content string, links []string) []string {
	requested := make([]string, 0, len(links))
	seenRequest := make(map[string]bool, len(links))
	for _, link := range links {
		if seenRequest[link] {
			continue
		}
		seenRequest[link] = true
		requested = append(requested, link)
	}

	existing := make(map[string]bool)
	for _, line := range strings.Split(content, "\n") {
		existing[strings.TrimSpace(line)] = true
	}

	missing := make([]string, 0, len(requested))
	for _, link := range requested {
		if existing[link] {
			continue
		}
		missing = append(missing, link)
	}
	return missing
}

// ApplySchemaProposals applies schema-surface proposals (extend_enum, remove_stem_field)
// that were separated from data proposals. It is the counterpart to ApplyProposals.
func ApplySchemaProposals(_ context.Context, report *proposal.Report, root string) error {
	all := make([]proposal.Proposal, 0, len(report.Proposals)+len(report.SchemaSuggestions))
	all = append(all, report.Proposals...)
	all = append(all, report.SchemaSuggestions...)
	for _, p := range all {
		if p.Type == proposal.ExtendEnum {
			if err := applyExtendEnum(p, root); err != nil {
				return fmt.Errorf("extend_enum %s: %w", p.Field, err)
			}
		}
	}
	return nil
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
	return fsx.WriteFileAtomic(stemPath, out, newFileMode)
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

		if err := fsx.WriteFileAtomic(absPath, []byte(newContent), newFileMode); err != nil {
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
	return fsx.WriteFileAtomic(absPath, []byte(newContent), newFileMode)
}

// applySetSection rewrites a named section of each target document. Both the
// repair-apply path and the fix --all path reach it, so it validates containment
// itself rather than trusting the caller to have done so; policy carries the
// caller's trust model for absolute paths.
func applySetSection(p proposal.Proposal, root string, _ map[string]*extract.Record, policy ContainmentPolicy) error {
	return applySetSectionWithWrite(p, root, policy, true)
}

func applySetSectionWithWrite(p proposal.Proposal, root string, policy ContainmentPolicy, write bool) error {
	for _, relPath := range p.Paths {
		absPath, err := ContainPath(root, relPath, policy)
		if err != nil {
			return err
		}

		data, err := os.ReadFile(absPath)
		if err != nil {
			return err
		}
		content, err := transformSetSection(string(data), p, relPath)
		if err != nil {
			return err
		}
		if !write {
			continue
		}
		if err := fsx.WriteFileAtomic(absPath, []byte(content), newFileMode); err != nil {
			return err
		}
	}
	return nil
}

func transformSetSection(content string, p proposal.Proposal, relPath string) (string, error) {
	heading := p.Heading
	newValue := strings.TrimRight(p.Value, "\n")

	headingEnd := -1
	searchFor := "\n" + heading + "\n"
	idx := strings.Index(content, searchFor)
	if idx >= 0 {
		headingEnd = idx + len(searchFor)
	} else if strings.HasPrefix(content, heading+"\n") {
		headingEnd = len(heading) + 1
	}

	if headingEnd == -1 {
		if p.Mode == "create" {
			if !strings.HasSuffix(content, "\n") {
				content += "\n"
			}
			return content + "\n" + heading + "\n\n" + newValue + "\n", nil
		}
		return "", fmt.Errorf("section %q not found in %s", heading, relPath)
	}

	headingLevel := 0
	for _, ch := range heading {
		if ch == '#' {
			headingLevel++
		} else {
			break
		}
	}

	sectionEnd := len(content)
	remaining := content[headingEnd:]
	offset := headingEnd
	for len(remaining) > 0 {
		lineEnd := strings.Index(remaining, "\n")
		var line string
		var advance int
		if lineEnd == -1 {
			line = remaining
			advance = len(remaining)
			remaining = ""
		} else {
			line = remaining[:lineEnd]
			advance = lineEnd + 1
			remaining = remaining[lineEnd+1:]
		}

		if len(line) > 0 && line[0] == '#' {
			level := 0
			for _, ch := range line {
				if ch == '#' {
					level++
				} else {
					break
				}
			}
			if level > 0 && level <= headingLevel {
				sectionEnd = offset
				break
			}
		}
		offset += advance
	}

	switch p.Mode {
	case "replace", "":
		return content[:headingEnd] + "\n" + newValue + "\n" + content[sectionEnd:], nil
	case "append", "create":
		existingBody := strings.TrimRight(content[headingEnd:sectionEnd], "\n\r ")
		return content[:headingEnd] + existingBody + "\n\n" + newValue + "\n" + content[sectionEnd:], nil
	default:
		return "", fmt.Errorf("unknown mode %q for set_section", p.Mode)
	}
}

func applyCorrectLink(p proposal.Proposal, root string) error {
	for _, path := range p.Paths {
		absPath := filepath.Join(root, path)
		content, err := os.ReadFile(absPath)
		if err != nil {
			return err
		}
		newContent := strings.Replace(string(content), p.From, p.To, 1)
		if err := fsx.WriteFileAtomic(absPath, []byte(newContent), newFileMode); err != nil {
			return err
		}
	}
	return nil
}

// removeStemSchemaField removes a field from the schema section of a .stem file.
// If the schema section becomes empty after removal, it is also removed.
func removeStemSchemaField(stemPath, fieldName string) error {
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

	// Find "schema" key in root mapping.
	for i := 0; i+1 < len(rootNode.Content); i += 2 {
		if rootNode.Content[i].Value != "schema" {
			continue
		}
		schemaNode := rootNode.Content[i+1]
		if schemaNode.Kind != yaml.MappingNode {
			continue
		}

		// Find and remove the field from schema.
		for j := 0; j+1 < len(schemaNode.Content); j += 2 {
			if schemaNode.Content[j].Value == fieldName {
				// Remove key-value pair (2 nodes).
				schemaNode.Content = append(schemaNode.Content[:j], schemaNode.Content[j+2:]...)

				// If schema is now empty, remove it from root.
				if len(schemaNode.Content) == 0 {
					rootNode.Content = append(rootNode.Content[:i], rootNode.Content[i+2:]...)
				}

				out, marshalErr := yaml.Marshal(&doc)
				if marshalErr != nil {
					return marshalErr
				}
				return fsx.WriteFileAtomic(stemPath, out, newFileMode)
			}
		}

		return fmt.Errorf("field %q not found in schema of %s", fieldName, stemPath)
	}

	return fmt.Errorf("no schema section in %s", stemPath)
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
	return fsx.WriteFileAtomic(stemPath, out, newFileMode)
}
