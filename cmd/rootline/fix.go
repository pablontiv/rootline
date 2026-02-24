package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/proposal"
	"github.com/pablontiv/rootline/internal/rules"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// FixResult represents the fix outcome for a single file.
type FixResult struct {
	Path            string   `json:"path"`
	Fixed           bool     `json:"fixed"`
	FieldsAdded     int      `json:"fields_added"`
	ValuesCorrected int      `json:"values_corrected"`
	Changes         []string `json:"changes"`
}

// BatchFixResult is the versioned JSON output for multi-file fix.
type BatchFixResult struct {
	Version int          `json:"version"`
	Kind    string       `json:"kind"`
	Results []*FixResult `json:"results"`
	Summary FixSummary   `json:"summary"`
}

// FixSummary holds aggregate counts for batch fix.
type FixSummary struct {
	Total   int `json:"total"`
	Fixed   int `json:"fixed"`
	Skipped int `json:"skipped"`
}

var (
	fixDryRun bool
	fixAll    bool
)

var fixCmd = &cobra.Command{
	Use:   "fix [file...]",
	Short: "Auto-repair validation errors",
	Long:  "Fix validation errors by adding missing required fields\nand correcting invalid enum values.",
	Args: func(cmd *cobra.Command, args []string) error {
		if fixAll {
			return nil
		}
		if len(args) == 0 {
			return fmt.Errorf("specify file(s) to fix or use --all")
		}
		return nil
	},
	RunE: runFix,
}

func init() {
	fixCmd.Flags().BoolVar(&fixDryRun, "dry-run", false, "show proposed changes without modifying files")
	fixCmd.Flags().BoolVar(&fixAll, "all", false, "fix all files in scope from current directory")
	rootCmd.AddCommand(fixCmd)
}

func runFix(cmd *cobra.Command, args []string) error {
	if fixAll {
		return runFixAll(cmd)
	}

	reg := extract.NewRegistry()
	totalAdded := 0
	totalCorrected := 0

	for _, file := range args {
		absPath, err := filepath.Abs(file)
		if err != nil {
			return fmt.Errorf("resolving path %s: %w", file, err)
		}

		content, err := os.ReadFile(absPath)
		if err != nil {
			return fmt.Errorf("reading %s: %w", file, err)
		}

		ext := reg.ForFile(absPath, "")
		if ext == nil {
			return fmt.Errorf("no extractor for %s", file)
		}

		record, err := ext.Extract(file, content)
		if err != nil {
			return fmt.Errorf("extracting %s: %w", file, err)
		}

		// Resolve effective schema
		dir := filepath.Dir(absPath)
		entries, err := rules.WalkUp(dir)
		if err != nil {
			return fmt.Errorf("resolving .stem for %s: %w", file, err)
		}
		effective := rules.MergeStemFiles(entries)

		// Validate
		errs := rules.Validate(record, effective)
		if len(errs) == 0 {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s: no errors to fix\n", file)
			continue
		}

		// Apply fixes to frontmatter
		added, corrected := applyFixes(record, effective, errs)

		if fixDryRun {
			for _, a := range added {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s: would add %s\n", file, a)
			}
			for _, c := range corrected {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s: would correct %s\n", file, c)
			}
			totalAdded += len(added)
			totalCorrected += len(corrected)
			continue
		}

		// Rewrite file
		newContent := rewriteFrontmatter(string(content), record.Frontmatter)
		if err := os.WriteFile(absPath, []byte(newContent), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", file, err)
		}

		totalAdded += len(added)
		totalCorrected += len(corrected)

		for _, a := range added {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s: added %s\n", file, a)
		}
		for _, c := range corrected {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s: corrected %s\n", file, c)
		}
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nFixed: %d fields added, %d values corrected\n", totalAdded, totalCorrected)
	return nil
}

func runFixAll(cmd *cobra.Command) error {
	root, err := filepath.Abs(".")
	if err != nil {
		return err
	}

	reg := extract.NewRegistry()
	resolver := func(dir string) *rules.StemFile {
		entries, err := rules.WalkUp(dir)
		if err != nil || len(entries) == 0 {
			return nil
		}
		return rules.MergeStemFiles(entries)
	}

	records, err := index.Scan(root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		return fmt.Errorf("scanning: %w", err)
	}

	// First pass: collect all records, their effective stems, and errors.
	allErrs := make(map[string][]rules.ValidationError)
	effectiveStems := make(map[string]*rules.StemFile)

	for _, rec := range records {
		absPath := filepath.Join(root, rec.Path)
		dir := filepath.Dir(absPath)
		entries, walkErr := rules.WalkUp(dir)
		if walkErr != nil {
			continue
		}
		effective := rules.MergeStemFiles(entries)
		effectiveStems[rec.Path] = effective

		errs := rules.Validate(rec, effective)
		if len(errs) > 0 {
			allErrs[rec.Path] = errs
		}
	}

	// In dry-run mode, use proposal engine for richer output.
	if fixDryRun {
		// Use the effective stem with the richest schema (most fields defined).
		var effective *rules.StemFile
		for _, s := range effectiveStems {
			if effective == nil || len(s.Schema) > len(effective.Schema) {
				effective = s
			}
		}

		report := proposal.Analyze(records, effective, allErrs)

		if outputFormat == "table" {
			return renderProposalTable(cmd, report)
		}
		return outputJSON(cmd, report, false)
	}

	// Apply mode: use proposal engine internally, output BatchFixResult for compatibility.
	var effective *rules.StemFile
	for _, s := range effectiveStems {
		if effective == nil || len(s.Schema) > len(effective.Schema) {
			effective = s
		}
	}

	report := proposal.Analyze(records, effective, allErrs)

	// Apply proposals if any.
	if len(report.Proposals) > 0 {
		if err := applyProposals(report, root, records); err != nil {
			return err
		}
	}

	// Convert proposals to BatchFixResult for output compatibility.
	results := proposalsToFixResults(report, records)
	batch := newBatchFixResult(results)

	if outputFormat == "table" {
		return renderFixTable(cmd, batch)
	}
	return outputJSON(cmd, batch, false)
}

func proposalsToFixResults(report *proposal.Report, records []*extract.Record) []*FixResult {
	// Group proposals by path.
	pathProposals := make(map[string][]proposal.Proposal)
	for _, p := range report.Proposals {
		for _, path := range p.Paths {
			pathProposals[path] = append(pathProposals[path], p)
		}
	}

	var results []*FixResult
	for _, rec := range records {
		proposals, hasProposals := pathProposals[rec.Path]
		if !hasProposals {
			results = append(results, &FixResult{
				Path: rec.Path, Fixed: false, Changes: []string{},
			})
			continue
		}

		var changes []string
		fieldsAdded := 0
		valuesCorrected := 0
		for _, p := range proposals {
			switch p.Type {
			case proposal.AddField, proposal.ExtractBody, proposal.InferFromChildren:
				fieldsAdded++
				changes = append(changes, fmt.Sprintf("add %s=%q", p.Field, p.Value))
			case proposal.CorrectValue, proposal.MigrateValue, proposal.CorrectLink:
				valuesCorrected++
				changes = append(changes, fmt.Sprintf("correct %s: %q -> %q", p.Field, p.From, p.To))
			case proposal.ExtendEnum:
				changes = append(changes, fmt.Sprintf("extend enum %s += %q", p.Field, p.Value))
			}
		}

		results = append(results, &FixResult{
			Path:            rec.Path,
			Fixed:           true,
			FieldsAdded:     fieldsAdded,
			ValuesCorrected: valuesCorrected,
			Changes:         changes,
		})
	}
	return results
}

func applyProposals(report *proposal.Report, root string, records []*extract.Record) error {
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
		}
	}

	// Update report to reflect only applied proposals.
	report.Proposals = applied
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

	root_node := doc.Content[0]
	if err := addEnumValueToNode(root_node, p.Field, p.Value); err != nil {
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

		newContent := rewriteFrontmatter(string(content), rec.Frontmatter)

		// Insert wiki-links before first heading if any.
		if len(p.WikiLinks) > 0 {
			newContent = insertWikiLinksBeforeHeading(newContent, p.WikiLinks)
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
	newContent := rewriteFrontmatter(string(content), fm)
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

func insertWikiLinksBeforeHeading(content string, links []string) string {
	linkBlock := strings.Join(links, "\n") + "\n\n"
	// Find the first ## heading.
	idx := strings.Index(content, "\n## ")
	if idx >= 0 {
		return content[:idx+1] + linkBlock + content[idx+1:]
	}
	// If no ## heading, append at end.
	return content + "\n" + linkBlock
}

func renderProposalTable(cmd *cobra.Command, report *proposal.Report) error {
	headers := []string{"Type", "Field", "Description", "Files"}
	var rows [][]string
	for _, p := range report.Proposals {
		files := strings.Join(p.Paths, ", ")
		rows = append(rows, []string{string(p.Type), p.Field, p.Description, files})
	}
	if len(rows) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No proposals generated (0 fixable errors)")
		return nil
	}
	renderTable(cmd.OutOrStdout(), headers, rows)
	return nil
}

func newBatchFixResult(results []*FixResult) *BatchFixResult {
	summary := FixSummary{Total: len(results)}
	for _, r := range results {
		if r.Fixed || r.FieldsAdded > 0 || r.ValuesCorrected > 0 {
			summary.Fixed++
		} else {
			summary.Skipped++
		}
	}
	return &BatchFixResult{
		Version: 1,
		Kind:    "rootline/fix-batch",
		Results: results,
		Summary: summary,
	}
}

func renderFixTable(cmd *cobra.Command, batch *BatchFixResult) error {
	headers := []string{"File", "Fixed", "Changes"}
	var rows [][]string
	for _, r := range batch.Results {
		fixed := "no"
		if r.Fixed {
			fixed = "yes"
		}
		changesStr := ""
		if len(r.Changes) > 0 {
			changesStr = strings.Join(r.Changes, "; ")
		}
		rows = append(rows, []string{r.Path, fixed, changesStr})
	}
	renderTable(cmd.OutOrStdout(), headers, rows)
	return nil
}

// applyFixes modifies the record's frontmatter in-place based on validation errors.
func applyFixes(record *extract.Record, effective *rules.StemFile, errs []rules.ValidationError) (added []string, corrected []string) {
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
				closest := closestMatch(current, sf.Values)
				if closest != "" {
					record.Frontmatter[field] = closest
					corrected = append(corrected, fmt.Sprintf("%s: %q -> %q", field, current, closest))
				}
			}
		}
	}
	return
}

// closestMatch finds the closest string by Levenshtein distance.
func closestMatch(s string, candidates []string) string {
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

// rewriteFrontmatter rebuilds a markdown file with updated frontmatter.
func rewriteFrontmatter(original string, fm map[string]any) string {
	// Find frontmatter boundaries
	if !strings.HasPrefix(original, "---\n") {
		// No frontmatter — prepend it
		var b strings.Builder
		b.WriteString("---\n")
		writeFrontmatterFields(&b, fm)
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
	writeFrontmatterFields(&b, fm)
	b.WriteString("---\n")
	b.WriteString(body)
	return b.String()
}

func writeFrontmatterFields(b *strings.Builder, fm map[string]any) {
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
