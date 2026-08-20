package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pablontiv/rootline/internal/rules"
	"github.com/spf13/cobra"
)

var describeCmd = &cobra.Command{
	Use:   "describe <path>",
	Short: "Show effective schema for a directory",
	Long:  "Display the merged .stem schema that applies to documents\nin the given directory, showing inherited and local rules.",
	Args:  cobra.ExactArgs(1),
	RunE:  runDescribe,
}

func init() {
	rootCmd.AddCommand(describeCmd)
}

// noSchemaGovernsError restates ErrNoSchemaFound as the condition the user hit
// rather than the step that was running, while staying the same error to code
// that tests for it with errors.Is.
type noSchemaGovernsError struct{ cause error }

func (e *noSchemaGovernsError) Error() string {
	return "no schema governs this path; run 'rootline init <path>' to create one"
}

func (e *noSchemaGovernsError) Unwrap() error { return e.cause }

// describeSchemaError turns a discovery failure into the answer describe owes
// the user.
//
// describe prints a resolved schema, so it must still fail when there is none —
// but "discovering .stem files: ..." named the internal step instead of the
// condition, which reads as a fault in rootline rather than an answer about the
// tree. A real IO or parse failure is a different answer and keeps its cause.
func describeSchemaError(err error) error {
	if rules.IsRealSchemaError(err) {
		return fmt.Errorf("resolving .stem: %w", err)
	}
	return &noSchemaGovernsError{cause: err}
}

func runDescribe(cmd *cobra.Command, args []string) error {
	targetPath, err := filepath.Abs(args[0])
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	var entries []rules.StemEntry
	var effective *rules.StemFile

	info, statErr := os.Stat(targetPath)
	if statErr == nil && !info.IsDir() {
		// File target: resolve with levels expansion so level-specific
		// schema fields (e.g. tipo at task level) appear in output.
		dir := filepath.Dir(targetPath)
		entries, err = rules.WalkUp(dir)
		if err != nil {
			return describeSchemaError(err)
		}
		effective, err = rules.ResolveForRecord(dir, args[0])
		if err != nil {
			return fmt.Errorf("resolving .stem for record: %w", err)
		}
	} else {
		// Directory (or non-existent path): use existing merge without levels.
		entries, err = rules.WalkUp(targetPath)
		if err != nil {
			return describeSchemaError(err)
		}
		effective = rules.MergeStemFiles(entries)
	}

	// Use relative path for display
	relPath := args[0]

	result, err := rules.NewDescribeResult(relPath, entries, effective)
	if err != nil {
		return fmt.Errorf("describing .stem: %w", err)
	}

	// Add hint when no schema is found
	if effective == nil || len(effective.Schema) == 0 {
		result.Hints = append(result.Hints, "No .stem schema found. Run 'rootline init <path>' to infer schema from existing files.")
	}

	if outputFormat == "table" {
		return renderDescribeTable(cmd, result)
	}
	return outputJSON(cmd, result, false)
}

func renderDescribeTable(cmd *cobra.Command, r *rules.DescribeResult) error {
	headers := []string{"Field", "Type", "Required", "Values", "Source", "Defined In"}
	var rows [][]string

	keys := make([]string, 0, len(r.Schema))
	for k := range r.Schema {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		f := r.Schema[k]
		req := "no"
		if f.Required {
			req = "yes"
		}
		vals := ""
		if len(f.Values) > 0 {
			vals = strings.Join(f.Values, ", ")
		}
		typeStr := f.Type
		rows = append(rows, []string{k, typeStr, req, vals, f.Extract, f.Source})
	}

	renderTable(cmd.OutOrStdout(), headers, rows)

	if len(r.Hints) > 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout())
		for _, h := range r.Hints {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Hint: %s\n", h)
		}
	}

	return nil
}
