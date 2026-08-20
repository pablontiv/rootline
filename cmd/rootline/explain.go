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

	// A directory otherwise reaches the extractor registry as an unrecognised
	// file type and comes back as "no extractor for <dir>". explain traces one
	// document and has no --all to offer, so the message names the argument
	// shape it does accept.
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("file not found: %s", file)
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory; use 'rootline explain <file>'", file)
	}

	// Extract record with body structure because source-backed schema fields
	// resolve through the same canonical body-aware contract as query.
	reg := extract.NewASTRegistry()
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
	markExplainBuiltinIsIndex(record, effective)

	// Run aggregation for index files.
	if effective != nil && len(effective.Aggregate) > 0 && rules.IsIndexFile(absPath, effective) {
		scanDir := filepath.Dir(absPath)
		reg2 := extract.NewASTRegistry()
		siblings, scanErr := index.Scan(ctx, scanDir, reg2)
		if scanErr == nil && len(siblings) > 0 {
			derive.DeriveAllSimple(ctx, siblings, scanDir)
			markExplainSiblingIndexes(siblings, scanDir)
			aggregateInputs := explainAggregateInputRecords(siblings, scanDir, absPath)
			if err := derive.EnrichBuiltins(ctx, aggregateInputs, scanDir, derive.DefaultResolver()); err != nil {
				return fmt.Errorf("enriching aggregate input records: %w", err)
			}
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

	// Build explain result before validation so source-backed target fields are
	// resolved first by the authoritative explain builder.
	result, err := rules.NewExplainResult(file, entries, effective, record, nil)
	if err != nil {
		return fmt.Errorf("resolving explain fields: %w", err)
	}

	// Run validation only after a complete explain result exists, then attach its
	// diagnostics before rendering any output.
	if effective != nil {
		for _, ve := range rules.Validate(ctx, record, effective) {
			result.Errors = append(result.Errors, rules.ExplainError(ve))
		}
	}

	if outputFormat == "table" {
		return renderExplainTable(cmd, result)
	}
	return outputJSON(cmd, result, false)
}

func markExplainBuiltinIsIndex(record *extract.Record, effective *rules.StemFile) {
	if record == nil {
		return
	}
	if record.Derived == nil {
		record.Derived = make(map[string]any)
	}
	record.Derived["isIndex"] = rules.IsIndexFile(record.Path, effective)
}

func markExplainSiblingIndexes(records []*extract.Record, root string) {
	resolver := derive.DefaultResolver()
	for _, record := range records {
		absPath := filepath.Join(root, record.Path)
		markExplainBuiltinIsIndex(record, resolver(filepath.Dir(absPath), record.Path))
	}
}

func explainAggregateInputRecords(records []*extract.Record, root, targetAbsPath string) []*extract.Record {
	inputs := make([]*extract.Record, 0, len(records))
	for _, record := range records {
		if filepath.Clean(filepath.Join(root, record.Path)) == filepath.Clean(targetAbsPath) {
			continue
		}
		inputs = append(inputs, record)
	}
	return inputs
}

func renderExplainTable(cmd *cobra.Command, r *rules.ExplainResult) error {
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "File: %s\n", r.Path)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Stem chain: %v\n\n", r.StemChain)

	// Fields table.
	headers := []string{"Field", "Value", "Origin", "Source", "Defined In", "Expression"}
	var rows [][]string
	for _, f := range r.Fields {
		valStr := fmt.Sprintf("%v", f.Value)
		if f.Value == nil {
			valStr = "<nil>"
		}
		rows = append(rows, []string{f.Name, valStr, f.Origin, f.Source, f.DefinedIn, f.Expression})
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
