package e2e

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/proposal"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestFixAllSiblingInference(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  tipo:\n    type: enum\n    required: true\n    values:\n      - a\n      - b\n      - c\n",
		"f1.md": "---\ntipo: a\n---\n# File 1\n",
		"f2.md": "---\ntipo: a\n---\n# File 2\n",
		"f3.md": "---\ntipo: a\n---\n# File 3\n",
		"f4.md": "---\n---\n# File 4 (missing tipo)\n",
	})

	ctx := context.Background()
	reg := extract.NewRegistry()
	resolver := func(dir string) *rules.StemFile {
		entries, err := rules.WalkUp(dir)
		if err != nil {
			return nil
		}
		return rules.MergeStemFiles(entries)
	}

	records, err := index.Scan(ctx, root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	// Validate all records and collect errors.
	allErrs := make(map[string][]rules.ValidationError)
	var effective *rules.StemFile
	for _, rec := range records {
		absPath := filepath.Join(root, rec.Path)
		dir := filepath.Dir(absPath)
		stem, resolveErr := rules.ResolveForRecord(dir, rec.Path)
		if resolveErr != nil {
			continue
		}
		if effective == nil || len(stem.Schema) > len(effective.Schema) {
			effective = stem
		}
		errs := rules.Validate(ctx, rec, stem)
		if len(errs) > 0 {
			allErrs[rec.Path] = errs
		}
	}

	report := proposal.Analyze(records, effective, allErrs)

	// Expect at least one infer_from_siblings proposal.
	found := false
	for _, p := range report.Proposals {
		if p.Type == proposal.InferFromSiblings {
			found = true
			if p.Value != "a" {
				t.Errorf("infer_from_siblings value = %q, want %q", p.Value, "a")
			}
			if p.Field != "tipo" {
				t.Errorf("infer_from_siblings field = %q, want %q", p.Field, "tipo")
			}
		}
	}
	if !found {
		t.Errorf("expected infer_from_siblings proposal, got types: %v", proposalTypes(report))
	}

	// Should NOT have an add_field for the same path/field (sibling inference takes priority).
	for _, p := range report.Proposals {
		if p.Type == proposal.AddField && p.Field == "tipo" {
			for _, path := range p.Paths {
				if path == "f4.md" {
					t.Errorf("add_field for tipo on f4.md should be suppressed by infer_from_siblings")
				}
			}
		}
	}
}

func proposalTypes(report *proposal.Report) []string {
	var types []string
	for _, p := range report.Proposals {
		types = append(types, string(p.Type))
	}
	return types
}
