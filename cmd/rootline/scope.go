package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/rules"
)

// stemScopeResolver returns the schema resolver every scanning command passes
// to index.Scan so they all see the same record set.
//
// validate --all and fix --all already filtered by scope.match while graph and
// query did not, so a file the schema explicitly declared out of governance
// still became a graph node and could fail graph --check (issue #62
// sub-defect 5). The closure was duplicated at every call site, which is how
// two of them ended up without it.
func stemScopeResolver() func(dir string) (*rules.StemFile, error) {
	return func(dir string) (*rules.StemFile, error) {
		entries, err := rules.WalkUp(dir)
		if err != nil || len(entries) == 0 {
			return nil, err
		}
		return rules.MergeStemFiles(entries), nil
	}
}

// excludedFromGovernance reports why a path is outside the governed record
// set, or "" when it is governed. It mirrors the two filters index.Scan
// applies: .stemignore and scope.match.
func excludedFromGovernance(absPath, commandScopeRoot string) string {
	dir := filepath.Dir(absPath)
	entries, err := rules.WalkUp(dir)
	if err != nil || len(entries) == 0 {
		if pathWithinScope(absPath, commandScopeRoot) && index.IsIgnored(commandScopeRoot, absPath) {
			return "skipped: excluded by .stemignore"
		}
		return "" // no schema governs it; ordinary validation reports that
	}
	root := filepath.Dir(entries[0].Path)
	if index.IsIgnored(root, absPath) {
		return "skipped: excluded by .stemignore"
	}
	if !index.MatchesScope(absPath, rules.MergeStemFiles(entries)) {
		return "skipped: out of scope for this .stem (scope.match)"
	}
	return ""
}

func pathWithinScope(absPath, root string) bool {
	rel, err := filepath.Rel(root, absPath)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// scanGoverned scans root with scope filtering, falling back to an ungoverned
// scan only when nothing under root resolved a schema at all.
//
// The retry is deliberately narrower than passing AllowUngoverned outright: as
// soon as one record IS governed the first scan keeps its scope filtering, so
// a file the schema excluded stays excluded. It exists because graph and query
// describe a tree that may carry no schema, while still honoring scope.match
// on trees that do.
func scanGoverned(ctx context.Context, root string, reg *extract.Registry) ([]*extract.Record, error) {
	resolver := stemScopeResolver()
	records, err := index.Scan(ctx, root, reg, index.WithScopeResolver(resolver))
	if errors.Is(err, rules.ErrNoSchemaFound) {
		return index.Scan(ctx, root, reg, index.WithScopeResolver(resolver), index.AllowUngoverned())
	}
	return records, err
}

func ensureRecordsResolve(ctx context.Context, records []*extract.Record, root string) error {
	for _, rec := range records {
		if err := ctx.Err(); err != nil {
			return err
		}
		absPath := filepath.Join(root, rec.Path)
		if _, err := rules.ResolveForRecord(filepath.Dir(absPath), rec.Path); err != nil && !errors.Is(err, rules.ErrNoSchemaFound) {
			return fmt.Errorf("resolving governed record %s: %w", rec.Path, err)
		}
	}
	return nil
}
