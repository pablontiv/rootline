package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/pablontiv/rootline/internal/derive"
	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/rules"
	"github.com/spf13/cobra"
)

var explainCmd = &cobra.Command{
	Use:   "explain <file>",
	Short: "Trace why a document has a given state",
	Long:  "Show the .stem rules and derivation chain that produced\nthe current state of a document. Every computed field is traceable.",
	Args:  cobra.ExactArgs(1),
	RunE:  runExplain,
}

func init() {
	rootCmd.AddCommand(explainCmd)
}


func runExplain(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	file := args[0]
	absPath, err := filepath.Abs(file)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	if _, err := os.Stat(absPath); err != nil {
		return fmt.Errorf("file not found: %s", file)
	}

	// Extract record.
	reg := extract.NewRegistry()
	ext := reg.ForFile(absPath, "")
	if ext == nil {
		return fmt.Errorf("no extractor for %s", file)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", file, err)
	}

	record, err := ext.Extract(file, content)
	if err != nil {
		return fmt.Errorf("extracting %s: %w", file, err)
	}

	// Resolve effective stem.
	dir := filepath.Dir(absPath)
	entries, err := rules.WalkUp(dir)
	if err != nil {
		return fmt.Errorf("resolving .stem for %s: %w", file, err)
	}
	effective := rules.MergeStemFiles(entries)

	// Run derivation.
	if effective != nil && len(effective.Derive) > 0 {
		_, _ = derive.DeriveRecord(ctx, record, effective, nil)
	}

	// Run aggregation for index files.
	if effective != nil && len(effective.Aggregate) > 0 && isExplainIndexFile(absPath) {
		scanDir := filepath.Dir(absPath)
		reg2 := extract.NewRegistry()
		siblings, scanErr := index.Scan(ctx, scanDir, reg2)
		if scanErr == nil && len(siblings) > 0 {
			derive.DeriveAllSimple(ctx, siblings, scanDir)
			derive.AggregateAllSimple(ctx, siblings, scanDir)
			// Copy derived values from the scanned version of this record.
			for _, s := range siblings {
				sAbs := filepath.Join(scanDir, s.Path)
				if sAbs == absPath {
					record.Derived = s.Derived
					break
				}
			}
		}
	}

	// Run validation.
	var valErrs []rules.ValidationError
	if effective != nil {
		valErrs = rules.Validate(ctx, record, effective)
	}

	// Build explain result.
	result := rules.NewExplainResult(file, entries, effective, record, valErrs)

	if outputFormat == "table" {
		return renderExplainTable(cmd, result)
	}
	return outputJSON(cmd, result, false)
}

func renderExplainTable(cmd *cobra.Command, r *rules.ExplainResult) error {
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "File: %s\n", r.Path)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Stem chain: %v\n\n", r.StemChain)

	// Fields table.
	headers := []string{"Field", "Value", "Origin", "Source", "Expression"}
	var rows [][]string
	for _, f := range r.Fields {
		valStr := fmt.Sprintf("%v", f.Value)
		if f.Value == nil {
			valStr = "<nil>"
		}
		rows = append(rows, []string{f.Name, valStr, f.Origin, f.Source, f.Expression})
	}
	renderTable(cmd.OutOrStdout(), headers, rows)

	// Errors.
	if len(r.Errors) > 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nValidation Errors:")
		errHeaders := []string{"Rule", "Field", "Message", "Severity", "Source"}
		var errRows [][]string
		for _, e := range r.Errors {
			errRows = append(errRows, []string{e.Rule, e.Field, e.Message, e.Severity, e.Source})
		}
		renderTable(cmd.OutOrStdout(), errHeaders, errRows)
	}

	return nil
}

func sortedMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func isExplainIndexFile(absPath string) bool {
	return filepath.Base(absPath) == "README.md"
}
