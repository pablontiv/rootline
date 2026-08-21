package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pablontiv/rootline/internal/fsx"
	"github.com/pablontiv/rootline/internal/rules"
)

type stemWritePlan struct {
	reportTarget string
	targetPath   string
	target       *fsx.AtomicTarget
	content      []byte
	action       string
}

type schemaApplyBatchPlan struct {
	writes         []stemWritePlan
	actionsByWrite [][]string
}

type stemWriteFunc func(*fsx.AtomicTarget, []byte, fs.FileMode) error

func closeSchemaApplyBatch(plan schemaApplyBatchPlan) error {
	seen := map[*fsx.AtomicTarget]struct{}{}
	var errs []error
	for _, write := range plan.writes {
		if write.target == nil {
			continue
		}
		if _, ok := seen[write.target]; ok {
			continue
		}
		seen[write.target] = struct{}{}
		if err := write.target.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func validateProspectiveStemWrites(ctx context.Context, root string, plan schemaApplyBatchPlan) ([]rules.StemHealthDiagnostic, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	physicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("resolving validation root %s: %w", root, err)
	}
	state, err := rules.DiscoverStemState(ctx, physicalRoot)
	if err != nil {
		return nil, err
	}

	for _, item := range sortedSchemaApplyBatch(plan) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if item.write.target == nil {
			return nil, fmt.Errorf("validating target %s: target capability unavailable", item.write.reportTarget)
		}
		next, err := state.Overlay(item.write.target.PhysicalPath(), item.write.content)
		if err != nil {
			return nil, fmt.Errorf("planning proposed .stem %s: %w", item.write.reportTarget, err)
		}
		state = next
	}

	result, err := rules.EvaluateStemState(ctx, state)
	if err != nil {
		return nil, err
	}

	diags := rules.StemHealthDiagnostics(result)
	sortStemHealthDiagnostics(diags)
	return diags, nil
}

func executeStemWrites(ctx context.Context, plan schemaApplyBatchPlan, dryRun bool, write stemWriteFunc) (applied []string, errs []string) {
	for _, item := range sortedSchemaApplyBatch(plan) {
		if err := ctx.Err(); err != nil {
			errs = append(errs, fmt.Sprintf("schema apply canceled: %v", err))
			return applied, errs
		}

		if dryRun {
			applied = append(applied, item.actions...)
			continue
		}

		if item.write.target == nil {
			errs = append(errs, fmt.Sprintf("write %s: target capability unavailable", item.write.reportTarget))
			continue
		}
		if write == nil {
			errs = append(errs, fmt.Sprintf("write %s: writer unavailable", item.write.reportTarget))
			continue
		}
		if err := write(item.write.target, item.write.content, 0o644); err != nil {
			errs = append(errs, fmt.Sprintf("write %s: %v", item.write.reportTarget, err))
			continue
		}
		applied = append(applied, item.actions...)
	}
	return applied, errs
}

func blockingStemHealth(diags []rules.StemHealthDiagnostic) []rules.StemHealthDiagnostic {
	blocking := make([]rules.StemHealthDiagnostic, 0, len(diags))
	for _, diag := range diags {
		if diag.Severity == rules.SeverityError {
			blocking = append(blocking, diag)
		}
	}
	return blocking
}

type schemaApplyBatchItem struct {
	write   stemWritePlan
	actions []string
}

func sortedSchemaApplyBatch(plan schemaApplyBatchPlan) []schemaApplyBatchItem {
	items := make([]schemaApplyBatchItem, 0, len(plan.writes))
	byTarget := map[string]int{}
	for i, write := range plan.writes {
		// The legacy per-write action label is retained on stemWritePlan for the
		// planner shape, but publishing is intentionally driven only by the
		// index-aligned actionsByWrite slice so duplicate targets cannot inherit
		// another write's actions.
		_ = write.action
		var actions []string
		if i < len(plan.actionsByWrite) {
			actions = append([]string(nil), plan.actionsByWrite[i]...)
		}
		key := normalizedStemWriteTarget(write)
		if existing, ok := byTarget[key]; ok {
			// Multiple approved public actions can describe one physical target. The
			// filesystem must see only the final bytes, and the public actions are
			// published only after that single final write succeeds.
			items[existing].write = write
			items[existing].actions = append(items[existing].actions, actions...)
			continue
		}
		byTarget[key] = len(items)
		items = append(items, schemaApplyBatchItem{write: write, actions: actions})
	}

	slices.SortStableFunc(items, func(a, b schemaApplyBatchItem) int {
		return strings.Compare(normalizedStemWriteTarget(a.write), normalizedStemWriteTarget(b.write))
	})
	return items
}

func isNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist) || os.IsNotExist(err)
}

func normalizedStemWriteTarget(write stemWritePlan) string {
	if write.target != nil {
		return write.target.PhysicalPath()
	}
	if write.targetPath != "" {
		abs, err := filepath.Abs(write.targetPath)
		if err != nil {
			return filepath.Clean(write.targetPath)
		}
		return filepath.Clean(abs)
	}
	return filepath.Clean(write.reportTarget)
}

func sortStemHealthDiagnostics(diags []rules.StemHealthDiagnostic) {
	slices.SortFunc(diags, func(a, b rules.StemHealthDiagnostic) int {
		if c := strings.Compare(a.Path, b.Path); c != 0 {
			return c
		}
		if c := strings.Compare(a.Check, b.Check); c != 0 {
			return c
		}
		if c := strings.Compare(a.Field, b.Field); c != 0 {
			return c
		}
		if c := strings.Compare(a.Severity, b.Severity); c != 0 {
			return c
		}
		return strings.Compare(a.Message, b.Message)
	})
}
