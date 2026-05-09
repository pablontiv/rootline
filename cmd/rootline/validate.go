package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pablontiv/rootline/internal/derive"
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
	validateWhere  []string
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
	validateCmd.Flags().StringArrayVar(&validateWhere, "where", nil, "filter expression for --all mode (e.g. \"estado == 'Pending'\")")
	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	if validateStaged {
		return runValidateStaged(cmd)
	}
	if validateAll {
		return runValidateAll(cmd, args)
	}
	if len(args) == 0 {
		return fmt.Errorf("specify file(s) to validate or use --all")
	}
	return runValidateFiles(cmd, args)
}

func runValidateFiles(cmd *cobra.Command, files []string) error {
	ctx := cmd.Context()

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

		// Resolve effective stem (with levels expansion for hierarchical schemas)
		dir := filepath.Dir(absPath)
		effective, err := rules.ResolveForRecord(dir, file)
		if err != nil {
			return fmt.Errorf("resolving .stem for %s: %w", file, err)
		}

		// Structural integrity check: detect multiple YAML documents.
		structErrs := rules.ValidateStructure(content, file)
		var errs []rules.ValidationError

		// Validate
		errs = append(errs, rules.Validate(ctx, record, effective)...)
		errs = append(errs, structErrs...)

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

func runValidateAll(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	scanRoot := "."
	if len(args) > 0 {
		scanRoot = args[0]
	}
	root, err := filepath.Abs(scanRoot)
	if err != nil {
		return err
	}

	// Phase 1: Stem health checks.
	var results []*rules.ValidationResult
	stemHealth, stemErr := rules.ValidateStemHealth(ctx, root)
	if stemErr == nil {
		results = append(results, stemHealthToResults(stemHealth)...)
	}

	// Phase 2: Document validation.
	reg := extract.NewRegistry()
	resolver := func(dir string) *rules.StemFile {
		entries, err := rules.WalkUp(dir)
		if err != nil || len(entries) == 0 {
			return nil
		}
		return rules.MergeStemFiles(entries)
	}

	records, err := index.Scan(ctx, root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		return fmt.Errorf("scanning: %w", err)
	}

	derive.DeriveAllSimple(ctx, records, root)
	derive.EnrichBuiltinsSimple(ctx, records, root)
	derive.AggregateAllSimple(ctx, records, root)

	// Apply --where filter.
	records, err = filterRecords(ctx, records, validateWhere, nil)
	if err != nil {
		return fmt.Errorf("filtering records: %w", err)
	}

	visitedDirs := make(map[string]bool)

	for _, rec := range records {
		absPath := filepath.Join(root, rec.Path)
		dir := filepath.Dir(absPath)
		effective, resolveErr := rules.ResolveForRecord(dir, rec.Path)
		if resolveErr != nil {
			continue
		}
		errs := rules.Validate(ctx, rec, effective)

		results = append(results, rules.NewValidationResult(rec.Path, errs))

		// Track directories for structural validation.
		if !visitedDirs[dir] {
			visitedDirs[dir] = true
		}
	}

	// Structural directory validation.
	for dir := range visitedDirs {
		entries, walkErr := rules.WalkUp(dir)
		if walkErr != nil {
			continue
		}
		effective := rules.MergeStemFiles(entries)
		if effective.Structural.IsEmpty() {
			continue
		}

		structErrs := rules.ValidateDirectory(ctx, dir, effective)
		relDir, _ := filepath.Rel(root, dir)
		if relDir == "" || relDir == "." {
			relDir = ""
		}
		dirPath := relDir + "/"
		results = append(results, rules.NewValidationResult(dirPath, structErrs))
	}

	// Drift detection: group records by parent directory and detect drift
	// for each index file against its direct children.
	var driftWarnings []rules.DriftWarning
	parentChildren := groupByParentDir(records, root)
	for dir, group := range parentChildren {
		if group.parent == nil {
			continue
		}
		entries, walkErr := rules.WalkUp(dir)
		if walkErr != nil || len(entries) == 0 {
			continue
		}
		effective := rules.MergeStemFiles(entries)
		if effective == nil {
			continue
		}
		driftWarnings = append(driftWarnings, rules.DetectDrift(*group.parent, group.children, effective.Schema)...)
	}

	batch := rules.NewBatchValidationResultWithDrift(results, driftWarnings)
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

// stemHealthToResults converts stem-health checks into ValidationResults.
func stemHealthToResults(result *rules.StemHealthResult) []*rules.ValidationResult {
	var results []*rules.ValidationResult
	for _, c := range result.Checks {
		if c.Status == "pass" {
			continue
		}
		severity := "warn"
		if c.Status == "fail" {
			severity = "error"
		}
		path := c.Path
		if path == "" {
			path = ".stem"
		}
		errs := []rules.ValidationError{{
			Rule:     c.Name,
			Field:    c.Field,
			Message:  c.Message,
			Source:   "stem-health",
			Severity: severity,
		}}
		results = append(results, rules.NewValidationResult(path, errs))
	}
	return results
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

	// Render drift warnings section if present.
	if len(batch.DriftWarnings) > 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout())
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Drift Warnings")
		driftHeaders := []string{"Field", "Parent", "Parent Value", "Children Value"}
		var driftRows [][]string
		for _, dw := range batch.DriftWarnings {
			driftRows = append(driftRows, []string{
				dw.Field,
				dw.ParentPath,
				fmt.Sprintf("%v", dw.ParentValue),
				fmt.Sprintf("%v", dw.ChildrenValue),
			})
		}
		renderTable(cmd.OutOrStdout(), driftHeaders, driftRows)
	}

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

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))

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

	current, err := extractFieldPath(obj, splitDotPath(path), path)
	if err != nil {
		return nil, err
	}

	return json.Marshal(current)
}

func extractFieldPath(current any, parts []string, fullPath string) (any, error) {
	if len(parts) == 0 {
		return current, nil
	}

	part := parts[0]
	if strings.HasSuffix(part, "[]") {
		key := strings.TrimSuffix(part, "[]")
		m, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("field %q: not an object at %q", fullPath, part)
		}
		val, exists := m[key]
		if !exists {
			return nil, fmt.Errorf("field %q: key %q not found", fullPath, key)
		}
		items, ok := val.([]any)
		if !ok {
			return nil, fmt.Errorf("field %q: %q is not an array", fullPath, key)
		}

		projected := make([]any, 0, len(items))
		for i, item := range items {
			v, err := extractFieldPath(item, parts[1:], fullPath)
			if err != nil {
				return nil, fmt.Errorf("field %q: array %q index %d: %w", fullPath, key, i, err)
			}
			projected = append(projected, v)
		}
		return projected, nil
	}

	m, ok := current.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("field %q: not an object at %q", fullPath, part)
	}
	val, exists := m[part]
	if !exists {
		return nil, fmt.Errorf("field %q: key %q not found", fullPath, part)
	}
	return extractFieldPath(val, parts[1:], fullPath)
}

// parentChildGroup holds an index file and its direct children for drift detection.
type parentChildGroup struct {
	parent   *extract.Record
	children []extract.Record
}

// groupByParentDir groups records by their parent directory.
// For each directory, identifies the index file (README.md) as parent
// and other files as children.
func groupByParentDir(records []*extract.Record, root string) map[string]*parentChildGroup {
	groups := make(map[string]*parentChildGroup)

	for _, rec := range records {
		absPath := filepath.Join(root, rec.Path)
		dir := filepath.Dir(absPath)

		if groups[dir] == nil {
			groups[dir] = &parentChildGroup{}
		}

		if filepath.Base(rec.Path) == "README.md" {
			groups[dir].parent = rec
		} else {
			groups[dir].children = append(groups[dir].children, *rec)
		}
	}

	return groups
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
