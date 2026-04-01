package validation

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/pablontiv/rootline/internal/derive"
	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/query"
	"github.com/pablontiv/rootline/internal/rules"
)

// ValidateAll runs the full validation pipeline against the given root directory.
// It includes stem health checks, document scanning, derivation, aggregation,
// filtering via where expressions, record validation, structural validation,
// and drift detection.
func ValidateAll(ctx context.Context, root string, wheres []string) (*rules.BatchValidationResult, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	// Phase 1: Stem health checks.
	var results []*rules.ValidationResult
	stemHealth, stemErr := rules.ValidateStemHealth(ctx, absRoot)
	if stemErr == nil {
		results = append(results, rules.StemHealthToResults(stemHealth)...)
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

	records, err := index.Scan(ctx, absRoot, reg, index.WithScopeResolver(resolver))
	if err != nil {
		return nil, fmt.Errorf("scanning: %w", err)
	}

	derive.DeriveAllSimple(ctx, records, absRoot)
	derive.EnrichBuiltinsSimple(ctx, records, absRoot)
	derive.AggregateAllSimple(ctx, records, absRoot)

	// Apply where filters.
	filtered, err := filterRecords(ctx, records, wheres)
	if err != nil {
		return nil, fmt.Errorf("filtering records: %w", err)
	}

	visitedDirs := make(map[string]bool)

	for _, rec := range filtered {
		recAbsPath := filepath.Join(absRoot, rec.Path)
		dir := filepath.Dir(recAbsPath)
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
		relDir, _ := filepath.Rel(absRoot, dir)
		if relDir == "" || relDir == "." {
			relDir = ""
		}
		dirPath := relDir + "/"
		results = append(results, rules.NewValidationResult(dirPath, structErrs))
	}

	// Drift detection: group records by parent directory and detect drift
	// for each index file against its direct children.
	var driftWarnings []rules.DriftWarning
	parentChildren := rules.GroupByParentDir(filtered, absRoot)
	for dir, group := range parentChildren {
		if group.Parent == nil {
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
		driftWarnings = append(driftWarnings, rules.DetectDrift(*group.Parent, group.Children, effective.Schema)...)
	}

	return rules.NewBatchValidationResultWithDrift(results, driftWarnings), nil
}

// filterRecords applies where expressions to records, returning only those that match.
func filterRecords(ctx context.Context, records []*extract.Record, wheres []string) ([]*extract.Record, error) {
	var cleaned []string
	for _, w := range wheres {
		if w != "" {
			cleaned = append(cleaned, w)
		}
	}
	if len(cleaned) == 0 {
		return records, nil
	}

	whereExpr := ""
	for i, w := range cleaned {
		if i > 0 {
			whereExpr += " && "
		}
		whereExpr += "(" + w + ")"
	}

	program, err := query.CompileWhere(whereExpr)
	if err != nil {
		return nil, err
	}

	var filtered []*extract.Record
	for _, rec := range records {
		match, err := query.MatchRecord(ctx, program, rec, nil)
		if err != nil {
			return nil, err
		}
		if match {
			filtered = append(filtered, rec)
		}
	}

	if filtered == nil {
		filtered = []*extract.Record{}
	}
	return filtered, nil
}
