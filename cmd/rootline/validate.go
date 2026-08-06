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

	reg := extract.NewASTRegistry()
	var results []*rules.ValidationResult
	linkCache := rules.NewHeadingCache()

	for _, file := range files {
		absPath, err := filepath.Abs(file)
		if err != nil {
			return fmt.Errorf("resolving path %s: %w", file, err)
		}

		// Check the path exists and is a file. A directory otherwise reaches the
		// extractor registry as an unrecognised file type and comes back as
		// "no extractor for <dir>", which describes a missing file type and
		// hides the flag that actually handles directories.
		info, err := os.Stat(absPath)
		if err != nil {
			return fmt.Errorf("file not found: %s", file)
		}
		if info.IsDir() {
			return fmt.Errorf("%s is a directory; use 'rootline validate --all %s'", file, file)
		}

		// Honor the same exclusions `--all` applies. Naming a file explicitly
		// must not smuggle it back into governance, or the pre-commit hook
		// and CI enforce different rules on the same file. The skip is
		// reported rather than silent so the user can tell why.
		if reason := excludedFromGovernance(absPath); reason != "" {
			results = append(results, rules.NewValidationResult(file, []rules.ValidationError{{
				Rule:     "skipped",
				Field:    "",
				Message:  reason,
				Source:   "scope",
				Severity: "warn",
			}}))
			continue
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
		errs = append(errs, rules.CheckLinks(record.Links, effective.Links, absPath, rules.SchemaRoot(absPath), linkCache)...)
		errs = append(errs, rules.ExtractionErrors(record)...)
		errs = append(errs, structErrs...)

		results = append(results, rules.NewValidationResult(file, errs))
	}

	// One file, several files or none: the envelope is the same shape, so a
	// consumer never has to branch on how the command was invoked.
	return emitValidateEnvelope(cmd, rules.NewValidationEnvelope(rules.ValidationEnvelopeInput{Results: results}))
}

// emitValidateEnvelope writes the envelope in the requested format and maps it
// to the process exit code. Every validate path ends here.
func emitValidateEnvelope(cmd *cobra.Command, batch *rules.BatchValidationResult) error {
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

	// Phase 1: Stem health checks. These describe `.stem` files, never records,
	// so they travel in their own collection and never reach summary.total.
	var results []*rules.ValidationResult
	var notices []rules.Notice
	stemHealth, stemErr := rules.ValidateStemHealth(ctx, root)
	if stemErr != nil {
		notices = append(notices, rules.Notice{
			Severity: rules.SeverityWarn,
			Code:     "stem_health_unavailable",
			Message:  stemErr.Error(),
		})
	}
	health := rules.StemHealthDiagnostics(stemHealth)

	// Phase 2: Document validation.
	reg := extract.NewASTRegistry()
	resolver := stemScopeResolver()

	records, err := index.Scan(ctx, root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		// A corpus that cannot be scanned is exactly the case Phase 1 explains:
		// no .stem anywhere, or one that does not parse. Returning the raw error
		// discarded every diagnostic and wrote a non-JSON line to a consumer
		// that asked for JSON, so the failure is reported inside the envelope.
		notices = append(notices, rules.Notice{
			Severity: rules.SeverityError,
			Code:     "scan_failed",
			Message:  fmt.Sprintf("scanning: %v", err),
		})
		return emitValidateEnvelope(cmd, rules.NewValidationEnvelope(rules.ValidationEnvelopeInput{
			StemHealth: health,
			Notices:    notices,
		}))
	}

	if len(records) == 0 {
		// A path that was renamed or emptied used to report total: 1, valid: 1
		// — the stem-files-exist pseudo-record — and a CI gate read it as green.
		notices = append(notices, rules.Notice{
			Severity: rules.SeverityWarn,
			Code:     "no_records",
			Message:  "no records found in scope",
		})
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
	linkCache := rules.NewHeadingCache()

	for _, rec := range records {
		absPath := filepath.Join(root, rec.Path)
		dir := filepath.Dir(absPath)
		effective, resolveErr := rules.ResolveForRecord(dir, rec.Path)
		if resolveErr != nil {
			continue
		}
		errs := rules.Validate(ctx, rec, effective)
		errs = append(errs, rules.CheckLinks(rec.Links, effective.Links, absPath, root, linkCache)...)
		errs = append(errs, rules.ExtractionErrors(rec)...)
		if content, readErr := os.ReadFile(absPath); readErr == nil {
			errs = append(errs, rules.ValidateStructure(content, rec.Path)...)
		}

		results = append(results, rules.NewValidationResult(rec.Path, errs))

		// Track directories for structural validation.
		if !visitedDirs[dir] {
			visitedDirs[dir] = true
		}
	}

	// Structural directory validation. A directory is not a record either, so
	// its verdict travels beside the documents rather than among them.
	var structural []*rules.ValidationResult
	for dir := range visitedDirs {
		entries, walkErr := rules.WalkUp(dir)
		if walkErr != nil {
			notices = append(notices, rules.Notice{
				Severity: rules.SeverityError,
				Code:     "schema_resolution_failed",
				Message:  fmt.Sprintf("resolving schema for structural validation in %s: %v", dir, walkErr),
			})
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
		structural = append(structural, rules.NewValidationResult(dirPath, structErrs))
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
		if walkErr != nil {
			notices = append(notices, rules.Notice{
				Severity: rules.SeverityError,
				Code:     "schema_resolution_failed",
				Message:  fmt.Sprintf("resolving schema for drift detection in %s: %v", dir, walkErr),
			})
			continue
		}
		if len(entries) == 0 {
			continue
		}
		effective := rules.MergeStemFiles(entries)
		if effective == nil {
			continue
		}
		driftWarnings = append(driftWarnings, rules.DetectDrift(*group.parent, group.children, effective.Schema)...)
	}

	return emitValidateEnvelope(cmd, rules.NewValidationEnvelope(rules.ValidationEnvelopeInput{
		Results:       results,
		Structural:    structural,
		StemHealth:    health,
		DriftWarnings: driftWarnings,
		Notices:       notices,
	}))
}

func runValidateStaged(cmd *cobra.Command) error {
	files, err := getStagedFiles()
	if err != nil {
		return err
	}

	// An empty staging area is a corpus of size zero, not an absence of output.
	// Writing nothing broke `rootline validate --staged | jq -e '.summary.invalid == 0'`
	// in the pre-commit hook this flag exists for.
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

// validateHasFailure reports whether the run should exit non-zero.
//
// Errors fail on every axis — an invalid record, a directory that breaks its
// structural rules, a `.stem` that does not parse, a run-level failure such as
// an unscannable corpus. Splitting the populations changed where a verdict is
// reported, never whether it counts. Warnings fail only under --strict, and
// info never fails: a nested root marker is a supported configuration, and
// promoting it to a warning broke CI runs that had no way to suppress it.
func validateHasFailure(batch *rules.BatchValidationResult) bool {
	if batch.Summary.Invalid > 0 ||
		batch.Summary.StructuralErrorsCount > 0 ||
		batch.Summary.StemHealthErrorsCount > 0 ||
		batch.HasErrorNotice() {
		return true
	}
	if !validateStrict {
		return false
	}
	if batch.Summary.WarningsCount > 0 ||
		batch.Summary.StructuralWarningsCount > 0 ||
		batch.Summary.StemHealthWarningsCount > 0 {
		return true
	}
	for _, n := range batch.Notices {
		if n.Severity == rules.SeverityWarn {
			return true
		}
	}
	return false
}

func renderValidateTable(cmd *cobra.Command, batch *rules.BatchValidationResult) error {
	headers := []string{"File", "Valid", "Errors"}
	var rows [][]string
	// Directories render in the same table as documents — they are both
	// verdicts a reader scans top to bottom — while staying separate in JSON.
	for _, r := range append(append([]*rules.ValidationResult{}, batch.Results...), batch.Structural...) {
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

	// Stem health and notices get their own sections for the same reason they
	// get their own JSON keys: a schema diagnostic is not a record verdict.
	if len(batch.StemHealth) > 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout())
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Stem Health")
		var healthRows [][]string
		for _, d := range batch.StemHealth {
			healthRows = append(healthRows, []string{d.Path, d.Check, d.Severity, d.Message})
		}
		renderTable(cmd.OutOrStdout(), []string{"Stem", "Check", "Severity", "Message"}, healthRows)
	}

	if len(batch.Notices) > 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout())
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Notices")
		var noticeRows [][]string
		for _, n := range batch.Notices {
			noticeRows = append(noticeRows, []string{n.Severity, n.Code, n.Message})
		}
		renderTable(cmd.OutOrStdout(), []string{"Severity", "Code", "Message"}, noticeRows)
	}

	if validateHasFailure(batch) {
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
	return strings.FieldsFunc(path, func(r rune) bool {
		return r == '.'
	})
}
