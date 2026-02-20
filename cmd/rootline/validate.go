package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/rules"
	"github.com/spf13/cobra"
)

// ErrValidationFailed is returned when validation finds errors.
// The caller should set exit code 1.
var ErrValidationFailed = errors.New("validation failed")

var (
	validateAll    bool
	validateStrict bool
	validateStaged bool
)

var validateCmd = &cobra.Command{
	Use:   "validate [file...]",
	Short: "Check documents against .stem rules",
	Long:  "Validate one or more documents against the effective schema\ndefined by .stem files in the directory tree.",
	RunE:  runValidate,
}

func init() {
	validateCmd.Flags().BoolVar(&validateAll, "all", false, "validate all files in scope from current directory")
	validateCmd.Flags().BoolVar(&validateStrict, "strict", false, "treat warnings as errors (exit code 1)")
	validateCmd.Flags().BoolVar(&validateStaged, "staged", false, "validate only files in git staging area")
	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	if validateStaged {
		return runValidateStaged(cmd)
	}
	if validateAll {
		return runValidateAll(cmd)
	}
	if len(args) == 0 {
		return fmt.Errorf("specify file(s) to validate or use --all")
	}
	return runValidateFiles(cmd, args)
}

func runValidateFiles(cmd *cobra.Command, files []string) error {
	reg := extract.NewRegistry()
	var results []*rules.ValidationResult

	for _, file := range files {
		absPath, err := filepath.Abs(file)
		if err != nil {
			return fmt.Errorf("resolving path %s: %w", file, err)
		}

		// Check file exists
		if _, err := os.Stat(absPath); err != nil {
			return fmt.Errorf("file not found: %s", file)
		}

		// Check extractor exists
		ext := reg.ForFile(absPath, "")
		if ext == nil {
			return fmt.Errorf("no extractor for %s", file)
		}

		// Read and extract
		content, err := os.ReadFile(absPath)
		if err != nil {
			return fmt.Errorf("reading %s: %w", file, err)
		}

		record, err := ext.Extract(file, content)
		if err != nil {
			return fmt.Errorf("extracting %s: %w", file, err)
		}

		// Resolve effective stem
		dir := filepath.Dir(absPath)
		entries, err := rules.WalkUp(dir)
		if err != nil {
			return fmt.Errorf("resolving .stem for %s: %w", file, err)
		}
		effective := rules.MergeStemFiles(entries)

		// Validate
		errs := rules.Validate(record, effective)
		results = append(results, rules.NewValidationResult(file, errs))
	}

	// Single file → single result; multiple → batch
	if len(results) == 1 {
		hasErr := validateHasFailure(rules.NewBatchValidationResult(results))
		if outputFormat == "table" {
			return renderValidateTable(cmd, rules.NewBatchValidationResult(results))
		}
		return outputJSON(cmd, results[0], hasErr)
	}
	batch := rules.NewBatchValidationResult(results)
	if outputFormat == "table" {
		return renderValidateTable(cmd, batch)
	}
	return outputJSON(cmd, batch, validateHasFailure(batch))
}

func runValidateAll(cmd *cobra.Command) error {
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

	var results []*rules.ValidationResult
	for _, rec := range records {
		absPath := filepath.Join(root, rec.Path)
		dir := filepath.Dir(absPath)
		entries, walkErr := rules.WalkUp(dir)
		if walkErr != nil {
			continue
		}
		effective := rules.MergeStemFiles(entries)
		errs := rules.Validate(rec, effective)
		results = append(results, rules.NewValidationResult(rec.Path, errs))
	}

	batch := rules.NewBatchValidationResult(results)
	if outputFormat == "table" {
		return renderValidateTable(cmd, batch)
	}
	return outputJSON(cmd, batch, validateHasFailure(batch))
}

func runValidateStaged(cmd *cobra.Command) error {
	files, err := getStagedFiles()
	if err != nil {
		return err
	}

	if len(files) == 0 {
		// No staged markdown files — nothing to validate
		return nil
	}

	return runValidateFiles(cmd, files)
}

// getStagedFiles returns markdown files in the git staging area.
func getStagedFiles() ([]string, error) {
	out, err := exec.Command("git", "diff", "--cached", "--name-only", "--diff-filter=ACM").Output()
	if err != nil {
		return nil, fmt.Errorf("git diff --cached: %w", err)
	}

	var mdFiles []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		if strings.HasSuffix(line, ".md") {
			mdFiles = append(mdFiles, line)
		}
	}
	return mdFiles, nil
}

// validateHasFailure returns true if the batch has errors,
// or if --strict and has warnings.
func validateHasFailure(batch *rules.BatchValidationResult) bool {
	if batch.Summary.Invalid > 0 {
		return true
	}
	if validateStrict && batch.Summary.WarningsCount > 0 {
		return true
	}
	return false
}

func renderValidateTable(cmd *cobra.Command, batch *rules.BatchValidationResult) error {
	headers := []string{"File", "Valid", "Errors"}
	var rows [][]string
	for _, r := range batch.Results {
		valid := "yes"
		if !r.Valid {
			valid = "no"
		}
		errMsgs := make([]string, len(r.Errors))
		for i, e := range r.Errors {
			errMsgs[i] = e.Message
		}
		errStr := ""
		if len(errMsgs) > 0 {
			errStr = strings.Join(errMsgs, "; ")
		}
		rows = append(rows, []string{r.Path, valid, errStr})
	}
	renderTable(cmd.OutOrStdout(), headers, rows)
	if batch.Summary.Invalid > 0 {
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
		return ErrValidationFailed
	}
	return nil
}

// outputJSON marshals v to JSON, applies --field extraction if set,
// writes to stdout, and returns an error sentinel for exit code 1.
func outputJSON(cmd *cobra.Command, v any, hasErrors bool) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}

	// Apply --field extraction
	if len(fieldPath) > 0 {
		extracted, err := extractField(data, fieldPath[0])
		if err != nil {
			return err
		}
		data = extracted
	}

	fmt.Fprintln(cmd.OutOrStdout(), string(data))

	if hasErrors {
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
		return ErrValidationFailed
	}
	return nil
}

// extractField navigates a JSON structure by dot-separated path.
func extractField(data []byte, path string) ([]byte, error) {
	var obj any
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, fmt.Errorf("parsing JSON for field extraction: %w", err)
	}

	current := obj
	parts := splitDotPath(path)
	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("field %q: not an object at %q", path, part)
		}
		val, exists := m[part]
		if !exists {
			return nil, fmt.Errorf("field %q: key %q not found", path, part)
		}
		current = val
	}

	return json.Marshal(current)
}

// splitDotPath splits a dot-separated path into parts.
func splitDotPath(path string) []string {
	var parts []string
	current := ""
	for _, ch := range path {
		if ch == '.' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
