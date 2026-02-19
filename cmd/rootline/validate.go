package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/rules"
	"github.com/spf13/cobra"
)

// ErrValidationFailed is returned when validation finds errors.
// The caller should set exit code 1.
var ErrValidationFailed = errors.New("validation failed")

var validateAll bool

var validateCmd = &cobra.Command{
	Use:   "validate [file...]",
	Short: "Check documents against .stem rules",
	Long:  "Validate one or more documents against the effective schema\ndefined by .stem files in the directory tree.",
	RunE:  runValidate,
}

func init() {
	validateCmd.Flags().BoolVar(&validateAll, "all", false, "validate all files in scope from current directory")
	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
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
		return outputJSON(cmd, results[0], !results[0].Valid)
	}
	batch := rules.NewBatchValidationResult(results)
	return outputJSON(cmd, batch, batch.Summary.Invalid > 0)
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
	return outputJSON(cmd, batch, batch.Summary.Invalid > 0)
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
