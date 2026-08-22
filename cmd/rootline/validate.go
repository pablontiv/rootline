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
	commandScopeRoot, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	return runValidateFilesInScope(cmd, files, commandScopeRoot)
}

func runValidateFilesInScope(cmd *cobra.Command, files []string, commandScopeRoot string) error {
	ctx := cmd.Context()

	var results []*rules.ValidationResult
	var notices []rules.Notice
	linkCache := rules.NewHeadingCache()

	// The preflight lets validate through on an undeclared boundary so the
	// failure lands in the envelope. Naming a file must not be a way around
	// that, so each target's own chain is checked here; one notice per chain,
	// since several files usually share one.
	boundaryChecked := make(map[string]bool)

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
		if reason := excludedFromGovernance(absPath, commandScopeRoot); reason != "" {
			results = append(results, rules.NewValidationResult(file, []rules.ValidationError{{
				Rule:     "skipped",
				Field:    "",
				Message:  reason,
				Source:   "scope",
				Severity: "warn",
			}}))
			continue
		}

		if dir := filepath.Dir(absPath); !boundaryChecked[dir] {
			boundaryChecked[dir] = true
			if n := undeclaredBoundaryNotice(dir); n != nil {
				notices = append(notices, *n)
			}
		}

		recCtx, err := resolveValidationRecordContext(absPath, file)
		if err != nil {
			results, notices = appendSkippedSchemaResolutionResult(results, notices, file, fmt.Sprintf("resolving .stem for %s: %v", file, err))
			continue
		}

		result, err := validateProspectiveRecord(ctx, prospectiveRecordValidationInput{
			Path:      file,
			AbsPath:   absPath,
			ReadFile:  true,
			Effective: recCtx.effective,
			LinkCache: linkCache,
		})
		if err != nil {
			return err
		}
		results, notices = appendNormalizedValidationResult(results, notices, result, file, absPath, recCtx.governanceRoot)
	}

	// One file, several files or none: the envelope is the same shape, so a
	// consumer never has to branch on how the command was invoked.
	return emitValidateEnvelope(cmd, rules.NewValidationEnvelope(rules.ValidationEnvelopeInput{Results: results, Notices: notices}))
}

// emitValidateEnvelope writes the envelope in the requested format and maps it
// to the process exit code. Every validate path ends here.
func emitValidateEnvelope(cmd *cobra.Command, batch *rules.BatchValidationResult) error {
	if outputFormat == "table" {
		return renderValidateTable(cmd, batch)
	}
	return outputJSON(cmd, batch, validateHasFailure(batch))
}

type validationRecordContext struct {
	effective      *rules.StemFile
	governanceRoot string
}

func resolveValidationRecordContext(absPath, recordPath string) (validationRecordContext, error) {
	dir := filepath.Dir(absPath)
	entries, err := rules.WalkUp(dir)
	if err != nil {
		return validationRecordContext{}, err
	}
	effective, err := rules.ResolveForRecord(dir, recordPath)
	if err != nil {
		return validationRecordContext{}, err
	}
	return validationRecordContext{effective: effective, governanceRoot: filepath.Dir(entries[0].Path)}, nil
}

func normalizeValidationResultSources(result *rules.ValidationResult, recordPath, absPath, governanceRoot string) (*rules.ValidationResult, error) {
	errs, err := normalizeDocumentValidationErrors(result.Errors, recordPath, absPath, governanceRoot)
	if err != nil {
		return nil, err
	}
	warnings, err := normalizeDocumentValidationErrors(result.Warnings, recordPath, absPath, governanceRoot)
	if err != nil {
		return nil, err
	}
	out := *result
	out.Errors = errs
	out.Warnings = warnings
	return &out, nil
}

func normalizeDocumentValidationErrors(errs []rules.ValidationError, recordPath, absPath, governanceRoot string) ([]rules.ValidationError, error) {
	if len(errs) == 0 {
		return errs, nil
	}
	return rules.NormalizeValidationSources(canonicalizeRecordOwnedSources(errs, recordPath, absPath), governanceRoot)
}

func canonicalizeRecordOwnedSources(errs []rules.ValidationError, recordPath, absPath string) []rules.ValidationError {
	out := append([]rules.ValidationError(nil), errs...)
	recordSource := comparableRecordOwnedSource(recordPath)
	for i := range out {
		if comparableRecordOwnedSource(out[i].Source) == recordSource {
			out[i].Source = absPath
		}
	}
	return out
}

func comparableRecordOwnedSource(source string) string {
	return strings.ReplaceAll(filepath.ToSlash(source), `\\`, "/")
}

func appendNormalizedValidationResult(results []*rules.ValidationResult, notices []rules.Notice, result *rules.ValidationResult, recordPath, absPath, governanceRoot string) ([]*rules.ValidationResult, []rules.Notice) {
	normalized, err := normalizeValidationResultSources(result, recordPath, absPath, governanceRoot)
	if err != nil {
		return appendSkippedSchemaResolutionResult(results, notices, recordPath, fmt.Sprintf("normalizing validation sources for %s: %v", recordPath, err))
	}
	return append(results, normalized), notices
}

func appendSkippedSchemaResolutionResult(results []*rules.ValidationResult, notices []rules.Notice, path, message string) ([]*rules.ValidationResult, []rules.Notice) {
	return append(results, skippedValidationResult(path)), appendSchemaResolutionNotice(notices, message)
}

func emitValidateSchemaResolutionEnvelope(cmd *cobra.Command, records []*extract.Record, results []*rules.ValidationResult, health []rules.StemHealthDiagnostic, notices []rules.Notice, message string) error {
	for _, rec := range records {
		results, notices = appendSkippedSchemaResolutionResult(results, notices, rec.Path, fmt.Sprintf("%s for %s", message, rec.Path))
	}
	return emitValidateEnvelope(cmd, rules.NewValidationEnvelope(rules.ValidationEnvelopeInput{
		Results:    results,
		StemHealth: health,
		Notices:    notices,
	}))
}

func emitValidateRunLevelSchemaEnvelope(cmd *cobra.Command, records []*extract.Record, results []*rules.ValidationResult, health []rules.StemHealthDiagnostic, notices []rules.Notice, message string) error {
	notices = appendSchemaResolutionNotice(notices, message)
	for _, rec := range records {
		results = append(results, skippedValidationResult(rec.Path))
	}
	return emitValidateEnvelope(cmd, rules.NewValidationEnvelope(rules.ValidationEnvelopeInput{
		Results:    results,
		StemHealth: health,
		Notices:    notices,
	}))
}

// undeclaredBoundaryNotice reports the notice for a chain that reaches the
// filesystem root without a declared boundary, or nil when one is declared.
//
// The boundary preflight lets `validate` through precisely so it can report
// this in its own envelope instead of a bare Go error. That makes the check
// this function performs the ONLY thing standing between a file target and a
// chain the preflight already judged unsafe, so every validate entry point
// must call it. It lives here, shared, because the first attempt re-armed the
// check on the `--all` path alone and a file target silently exited 0.
func undeclaredBoundaryNotice(dir string) *rules.Notice {
	entries, err := rules.WalkUp(dir)
	if err != nil && errors.Is(err, rules.ErrNoSchemaFound) {
		// WalkUp found nothing above; look down the subtree, as the preflight does.
		entries, _ = rules.DownwardScan(dir)
	}
	if len(entries) == 0 || !rules.ChainHasNoDeclaredBoundary(entries) {
		return nil
	}
	return &rules.Notice{
		Severity: rules.SeverityError,
		Code:     "schema_resolution_failed",
		Message:  fmt.Sprintf("undeclared governance boundary: add 'root: true' to %s/.stem", filepath.Dir(entries[len(entries)-1].Path)),
	}
}

func appendSchemaResolutionNotice(notices []rules.Notice, message string) []rules.Notice {
	return append(notices, rules.Notice{
		Severity: rules.SeverityError,
		Code:     "schema_resolution_failed",
		Message:  message,
	})
}

func skippedValidationResult(path string) *rules.ValidationResult {
	return rules.NewValidationResult(path, []rules.ValidationError{{
		Rule:     "skipped",
		Field:    "",
		Message:  "validation incomplete because schema resolution failed; see notices",
		Source:   "schema",
		Severity: rules.SeverityError,
	}})
}

func resilientValidateAllScopeResolver() func(dir string) (*rules.StemFile, error) {
	return func(dir string) (*rules.StemFile, error) {
		entries, err := rules.WalkUp(dir)
		if err != nil {
			if errors.Is(err, rules.ErrNoSchemaFound) {
				return nil, err
			}
			return &rules.StemFile{}, nil
		}
		if len(entries) == 0 {
			return nil, rules.ErrNoSchemaFound
		}
		return rules.MergeStemFiles(entries), nil
	}
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
	resolver := resilientValidateAllScopeResolver()

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

	// Phase 1.5: the chain may have been collected from outside the project.
	if n := undeclaredBoundaryNotice(root); n != nil {
		notices = append(notices, *n)
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

	if err := derive.DeriveAllSimple(ctx, records, root); err != nil {
		return emitValidateSchemaResolutionEnvelope(cmd, records, results, health, notices, fmt.Sprintf("deriving records: %v", err))
	}
	if err := derive.EnrichBuiltinsSimple(ctx, records, root); err != nil {
		return emitValidateRunLevelSchemaEnvelope(cmd, records, results, health, notices, fmt.Sprintf("enriching records: %v", err))
	}
	if err := derive.AggregateAllSimple(ctx, records, root); err != nil {
		return emitValidateSchemaResolutionEnvelope(cmd, records, results, health, notices, fmt.Sprintf("aggregating records: %v", err))
	}

	// Apply --where filter.
	records, err = filterRecords(ctx, records, validateWhere, knownWhereFields(records, root), cmd.ErrOrStderr())
	if err != nil {
		return fmt.Errorf("filtering records: %w", err)
	}

	visitedDirs := make(map[string]bool)
	linkCache := rules.NewHeadingCache()

	for _, rec := range records {
		absPath := filepath.Join(root, rec.Path)
		dir := filepath.Dir(absPath)
		recCtx, resolveErr := resolveValidationRecordContext(absPath, rec.Path)
		if resolveErr != nil {
			results, notices = appendSkippedSchemaResolutionResult(results, notices, rec.Path, fmt.Sprintf("resolving .stem for %s: %v", rec.Path, resolveErr))
			continue
		}
		errs := rules.Validate(ctx, rec, recCtx.effective)
		errs = append(errs, rules.CheckLinks(rec.Links, recCtx.effective.Links, absPath, root, linkCache)...)
		errs = append(errs, rules.ExtractionErrors(rec)...)
		if content, readErr := os.ReadFile(absPath); readErr == nil {
			errs = append(errs, rules.ValidateStructure(content, rec.Path)...)
		}
		results, notices = appendNormalizedValidationResult(results, notices, rules.NewValidationResult(rec.Path, errs), rec.Path, absPath, recCtx.governanceRoot)

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
	return validationBatchHasFailure(batch, validateStrict)
}

func validationBatchHasFailure(batch *rules.BatchValidationResult, strict bool) bool {
	if batch.Summary.Invalid > 0 ||
		batch.Summary.StructuralErrorsCount > 0 ||
		batch.Summary.StemHealthErrorsCount > 0 ||
		batch.HasErrorNotice() {
		return true
	}
	if !strict {
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

	// Apply --field extraction. The flag is documented "(repeatable)" and
	// registered as a StringSliceVar, but only fieldPath[0] was read, so
	// `--field a --field b` returned only a and the result depended purely on
	// argument order. Several paths now yield a JSON array in flag order; a
	// single path keeps emitting the bare value, so no existing caller moves.
	if paths := effectiveFieldPaths(); len(paths) > 0 {
		extracted, err := extractFields(data, paths)
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
//
// The single-path form, kept for callers that render their own JSON instead of
// going through outputJSON. Repeatable --field reaches those callers only once
// they adopt extractFields.
func extractField(data []byte, path string) ([]byte, error) {
	return extractFields(data, []string{path})
}

// extractFields navigates a JSON structure once per requested path.
//
// One path marshals to the bare value it resolves to, which is what every
// existing caller pipes into jq or a shell variable. Several marshal to a JSON
// array in flag order — the only shape that keeps N results distinguishable
// without inventing keys the envelope never had. The paths are resolved against
// the same document, so a failure on the third one fails the whole run rather
// than emitting a partial array.
func extractFields(data []byte, paths []string) ([]byte, error) {
	var obj any
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, fmt.Errorf("parsing JSON for field extraction: %w", err)
	}

	values := make([]any, 0, len(paths))
	for _, path := range paths {
		value, err := extractFieldPath(obj, splitDotPath(path), path)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}

	if len(values) == 1 {
		return json.Marshal(values[0])
	}
	return json.Marshal(values)
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
