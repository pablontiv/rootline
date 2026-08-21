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
	writeRoot    string
	target       string
	content      []byte
	action       string
}

type schemaApplyBatchPlan struct {
	writes         []stemWritePlan
	actionsByWrite [][]string
}

type stemWriteFunc func(string, string, []byte, fs.FileMode) error

func validateProspectiveStemWrites(ctx context.Context, root string, plan schemaApplyBatchPlan) ([]rules.StemHealthDiagnostic, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	state, err := rules.DiscoverStemState(ctx, root)
	if err != nil {
		return nil, err
	}

	for _, item := range sortedSchemaApplyBatch(plan) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if item.write.writeRoot != "" {
			if _, err := fsx.StatInRoot(item.write.writeRoot, item.write.target); err != nil && !isNotExist(err) {
				return nil, fmt.Errorf("validating rooted target %s: %w", item.write.reportTarget, err)
			}
		}
		next, err := state.Overlay(item.write.target, item.write.content)
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

		if write == nil {
			errs = append(errs, fmt.Sprintf("write %s: writer unavailable", item.write.reportTarget))
			continue
		}
		writeRoot := item.write.writeRoot
		if writeRoot == "" {
			writeRoot = filepath.Dir(item.write.target)
		}
		if err := write(writeRoot, item.write.target, item.write.content, 0o644); err != nil {
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
		key := normalizedStemWriteTarget(write.target)
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
		return strings.Compare(normalizedStemWriteTarget(a.write.target), normalizedStemWriteTarget(b.write.target))
	})
	return items
}

func isNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist) || os.IsNotExist(err)
}

func normalizedStemWriteTarget(target string) string {
	abs, err := filepath.Abs(target)
	if err != nil {
		return filepath.Clean(target)
	}
	return filepath.Clean(abs)
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
