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
			fmt.Fprintf(cmd.OutOrStdout(), "%s: no errors to fix\n", file)
			continue
		}

		// Apply fixes to frontmatter
		added, corrected := applyFixes(record, effective, errs)

		if fixDryRun {
			for _, a := range added {
				fmt.Fprintf(cmd.OutOrStdout(), "%s: would add %s\n", file, a)
			}
			for _, c := range corrected {
				fmt.Fprintf(cmd.OutOrStdout(), "%s: would correct %s\n", file, c)
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
			fmt.Fprintf(cmd.OutOrStdout(), "%s: added %s\n", file, a)
		}
		for _, c := range corrected {
			fmt.Fprintf(cmd.OutOrStdout(), "%s: corrected %s\n", file, c)
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\nFixed: %d fields added, %d values corrected\n", totalAdded, totalCorrected)
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
		if err != nil {
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
		// Use the first effective stem (they should all be equivalent for the root).
		var effective *rules.StemFile
		for _, s := range effectiveStems {
			effective = s
			break
		}

		report := proposal.Analyze(records, effective, allErrs)

		if outputFormat == "table" {
			return renderProposalTable(cmd, report)
		}
		return outputJSON(cmd, report, false)
	}

	// Apply mode: apply fixes directly.
	var results []*FixResult

	for _, rec := range records {
		errs, hasErrs := allErrs[rec.Path]
		if !hasErrs {
			results = append(results, &FixResult{
				Path:    rec.Path,
				Fixed:   false,
				Changes: []string{},
			})
			continue
		}

		effective := effectiveStems[rec.Path]
		added, corrected := applyFixes(rec, effective, errs)

		var changes []string
		for _, a := range added {
			changes = append(changes, "add "+a)
		}
		for _, c := range corrected {
			changes = append(changes, "correct "+c)
		}

		absPath := filepath.Join(root, rec.Path)
		content, readErr := os.ReadFile(absPath)
		if readErr != nil {
			return fmt.Errorf("reading %s: %w", rec.Path, readErr)
		}

		newContent := rewriteFrontmatter(string(content), rec.Frontmatter)
		if err := os.WriteFile(absPath, []byte(newContent), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", rec.Path, err)
		}

		results = append(results, &FixResult{
			Path:            rec.Path,
			Fixed:           true,
			FieldsAdded:     len(added),
			ValuesCorrected: len(corrected),
			Changes:         changes,
		})
	}

	batch := newBatchFixResult(results)

	if outputFormat == "table" {
		return renderFixTable(cmd, batch)
	}
	return outputJSON(cmd, batch, false)
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
		fmt.Fprintf(b, "%s: %v\n", k, fm[k])
	}
}
