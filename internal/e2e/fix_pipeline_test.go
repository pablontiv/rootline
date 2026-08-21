package e2e

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/derive"
	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/fix"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/proposal"
	"github.com/pablontiv/rootline/internal/rules"
)

// --- Helpers ---

// collectErrors validates all records and returns a map of path → errors.
func collectErrors(ctx context.Context, root string, records []*extract.Record) map[string][]rules.ValidationError {
	allErrs := make(map[string][]rules.ValidationError)
	for _, rec := range records {
		absPath := filepath.Join(root, rec.Path)
		dir := filepath.Dir(absPath)
		effective, err := rules.ResolveForRecord(dir, rec.Path)
		if err != nil {
			continue
		}
		errs := rules.Validate(ctx, rec, effective)
		if len(errs) > 0 {
			allErrs[rec.Path] = errs
		}
	}
	return allErrs
}

// richestStem returns the effective stem with the most schema fields.
func richestStem(root string, records []*extract.Record) *rules.StemFile {
	var best *rules.StemFile
	for _, rec := range records {
		absPath := filepath.Join(root, rec.Path)
		dir := filepath.Dir(absPath)
		eff, err := rules.ResolveForRecord(dir, rec.Path)
		if err != nil {
			continue
		}
		if best == nil || len(eff.Schema) > len(best.Schema) {
			best = eff
		}
	}
	return best
}

// scanAndAggregate runs the full scan + derive + aggregate pipeline.
func scanAndAggregate(t *testing.T, ctx context.Context, root string) []*extract.Record {
	t.Helper()
	reg := extract.NewRegistry()
	resolver := buildScopeResolver()
	records, err := index.Scan(ctx, root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if err := derive.DeriveAllSimple(ctx, records, root); err != nil {
		t.Fatalf("DeriveAllSimple: %v", err)
	}
	if err := derive.EnrichBuiltinsSimple(ctx, records, root); err != nil {
		t.Fatalf("enrich builtins: %v", err)
	}
	if err := derive.AggregateAllSimple(ctx, records, root); err != nil {
		t.Fatalf("AggregateAllSimple: %v", err)
	}
	return records
}

// requireProposal asserts exactly one proposal of the given type and returns it.
func requireProposal(t *testing.T, report *proposal.Report, ptype proposal.Type) proposal.Proposal {
	t.Helper()
	var found []proposal.Proposal
	for _, p := range report.Proposals {
		if p.Type == ptype {
			found = append(found, p)
		}
	}
	if len(found) == 0 {
		var types []string
		for _, p := range report.Proposals {
			types = append(types, string(p.Type))
		}
		t.Fatalf("no %s proposal found; got types: %v", ptype, types)
	}
	if len(found) > 1 {
		t.Fatalf("got %d %s proposals, want 1", len(found), ptype)
	}
	return found[0]
}

// readFile reads a file from the test project.
func readFile(t *testing.T, root, relPath string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, relPath))
	if err != nil {
		t.Fatalf("reading %s: %v", relPath, err)
	}
	return string(content)
}

// --- Aggregate Consistency Tests ---

func TestValidateAggregateConsistency_Mismatch(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem":          "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: enum\n    values: [Pending, Completed, \"In Progress\"]\n    required: true\naggregate:\n  estado: |\n    all(descendants, {.estado == \"Completed\"}) ? \"Completed\" : \"Pending\"\n",
		"S001/README.md": "---\nestado: Pending\n---\n# Story\n",
		"S001/T001.md":   "---\nestado: Completed\n---\n# Task 1\n",
		"S001/T002.md":   "---\nestado: Completed\n---\n# Task 2\n",
		"S001/T003.md":   "---\nestado: Completed\n---\n# Task 3\n",
	})

	ctx := context.Background()
	records := scanAndAggregate(t, ctx, root)
	allErrs := collectErrors(ctx, root, records)

	// README should have an aggregate error.
	var aggErrors int
	for _, errs := range allErrs {
		for _, e := range errs {
			if e.Rule == "aggregate" {
				aggErrors++
			}
		}
	}
	if aggErrors != 1 {
		t.Errorf("got %d aggregate errors, want 1; allErrs: %v", aggErrors, allErrs)
	}
}

func TestValidateAggregateConsistency_Match(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem":          "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: enum\n    values: [Pending, Completed]\n    required: true\naggregate:\n  estado: |\n    all(descendants, {.estado == \"Completed\"}) ? \"Completed\" : \"Pending\"\n",
		"S001/README.md": "---\nestado: Completed\n---\n# Story\n",
		"S001/T001.md":   "---\nestado: Completed\n---\n# Task 1\n",
		"S001/T002.md":   "---\nestado: Completed\n---\n# Task 2\n",
	})

	ctx := context.Background()
	records := scanAndAggregate(t, ctx, root)
	allErrs := collectErrors(ctx, root, records)

	for path, errs := range allErrs {
		for _, e := range errs {
			if e.Rule == "aggregate" {
				t.Errorf("unexpected aggregate error for %s: %s", path, e.Message)
			}
		}
	}
}

// --- Fix Pipeline Tests ---

func TestFixPipeline_DeriveInFix(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem":          "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: enum\n    values: [Pending, Completed, \"In Progress\"]\n    required: true\naggregate:\n  estado: |\n    all(descendants, {.estado == \"Completed\"}) ? \"Completed\" : \"Pending\"\n",
		"S001/README.md": "---\nestado: Pending\n---\n# Story\n",
		"S001/T001.md":   "---\nestado: Completed\n---\n# Task 1\n",
		"S001/T002.md":   "---\nestado: Completed\n---\n# Task 2\n",
	})

	ctx := context.Background()

	// Full pipeline: scan + derive + enrich + aggregate + validate.
	records := scanAndAggregate(t, ctx, root)
	allErrs := collectErrors(ctx, root, records)

	// After derive pipeline runs, aggregate should detect stale README estado.
	var aggErr bool
	for _, errs := range allErrs {
		for _, e := range errs {
			if e.Rule == "aggregate" {
				aggErr = true
			}
		}
	}
	if !aggErr {
		t.Errorf("expected aggregate error for stale README estado after derive pipeline, got none; allErrs: %v", allErrs)
	}
}

func TestFixPipeline_AddField(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: enum\n    values: [Pending, Completed]\n    required: true\n",
		"a.md":  "---\n\n---\n# Missing estado\n",
	})

	ctx := context.Background()
	records := scanAndAggregate(t, ctx, root)
	allErrs := collectErrors(ctx, root, records)
	effective := richestStem(root, records)

	report := proposal.Analyze(records, effective, allErrs)
	p := requireProposal(t, report, proposal.AddField)
	if p.Value != "Pending" {
		t.Errorf("add_field value = %q, want Pending", p.Value)
	}

	// The .stem declares estado without a default:, so the proposal's value is
	// engine-chosen. This pipeline test is about the apply mechanic, so it opts
	// in; the no-fabrication default is covered in internal/fix.
	if p.ValueSource != proposal.ValueSourceEnumFirst {
		t.Errorf("value_source = %q, want %q", p.ValueSource, proposal.ValueSourceEnumFirst)
	}
	if _, err := fix.ApplyProposals(ctx, report, root, records, true); err != nil {
		t.Fatalf("apply: %v", err)
	}

	// Re-validate: 0 errors.
	records2 := scanAndAggregate(t, ctx, root)
	allErrs2 := collectErrors(ctx, root, records2)
	if len(allErrs2) > 0 {
		t.Errorf("re-validate still has errors: %v", allErrs2)
	}
}

func TestFixPipeline_CorrectValue(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: enum\n    values: [Pending, Completed, \"In Progress\"]\n    required: true\n",
		"a.md":  "---\nestado: Completo\n---\n# Typo\n",
	})

	ctx := context.Background()
	records := scanAndAggregate(t, ctx, root)
	allErrs := collectErrors(ctx, root, records)
	effective := richestStem(root, records)

	report := proposal.Analyze(records, effective, allErrs)
	p := requireProposal(t, report, proposal.CorrectValue)
	if p.From != "Completo" || p.To != "Completed" {
		t.Errorf("correct_value: %q -> %q, want Completo -> Completed", p.From, p.To)
	}

	if _, err := fix.ApplyProposals(ctx, report, root, records, false); err != nil {
		t.Fatalf("apply: %v", err)
	}

	records2 := scanAndAggregate(t, ctx, root)
	allErrs2 := collectErrors(ctx, root, records2)
	if len(allErrs2) > 0 {
		t.Errorf("re-validate still has errors: %v", allErrs2)
	}
}

func TestFixPipeline_MigrateValue(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: enum\n    values: [Pending, Completed, Blocked]\n    required: true\nlinks:\n  allowed: [blocks]\n  blocks:\n    target: \"T.*\"\n",
		"a.md":  "---\nestado: Pending (blocked by T001)\n---\n# Blocked task\n",
	})

	ctx := context.Background()
	records := scanAndAggregate(t, ctx, root)
	allErrs := collectErrors(ctx, root, records)
	effective := richestStem(root, records)

	report := proposal.Analyze(records, effective, allErrs)
	p := requireProposal(t, report, proposal.MigrateValue)
	if p.To != "Pending" {
		t.Errorf("migrate to = %q, want Pending", p.To)
	}

	if _, err := fix.ApplyProposals(ctx, report, root, records, false); err != nil {
		t.Fatalf("apply: %v", err)
	}

	// Verify wiki-link was inserted.
	content := readFile(t, root, "a.md")
	if !strings.Contains(content, "[[blocks:T001]]") {
		t.Errorf("expected [[blocks:T001]] in body, got:\n%s", content)
	}
}

func TestFixPipeline_ExtractBody(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem":         "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: enum\n    values: [Pending, Completed]\n    required: true\n",
		"dir/README.md": "---\n\n---\n# Story\n\n**Estado**: Completed\n",
	})

	ctx := context.Background()
	records := scanAndAggregate(t, ctx, root)
	allErrs := collectErrors(ctx, root, records)
	effective := richestStem(root, records)

	report := proposal.Analyze(records, effective, allErrs)
	p := requireProposal(t, report, proposal.ExtractBody)
	if p.To != "Completed" {
		t.Errorf("extract_body to = %q, want Completed", p.To)
	}

	if _, err := fix.ApplyProposals(ctx, report, root, records, false); err != nil {
		t.Fatalf("apply: %v", err)
	}

	records2 := scanAndAggregate(t, ctx, root)
	allErrs2 := collectErrors(ctx, root, records2)
	if len(allErrs2) > 0 {
		t.Errorf("re-validate still has errors: %v", allErrs2)
	}
}

func TestFixPipeline_CorrectLink(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem":                 "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: enum\n    values: [Pending, Completed]\n    required: true\nlinks:\n  allowed: [blocks]\n  blocks:\n    target: \"^T\\\\d{3}-\"\n",
		"dir/a.md":              "---\nestado: Pending\n---\n# Task\n\n[[blocks:T001]]\n",
		"dir/T001-full-name.md": "---\nestado: Completed\n---\n# Target task\n",
	})

	ctx := context.Background()
	records := scanAndAggregate(t, ctx, root)
	allErrs := collectErrors(ctx, root, records)
	effective := richestStem(root, records)

	report := proposal.Analyze(records, effective, allErrs)
	p := requireProposal(t, report, proposal.CorrectLink)
	if !strings.Contains(p.To, "T001-full-name") {
		t.Errorf("correct_link to = %q, want to contain T001-full-name", p.To)
	}

	if _, err := fix.ApplyProposals(ctx, report, root, records, false); err != nil {
		t.Fatalf("apply: %v", err)
	}

	content := readFile(t, root, "dir/a.md")
	if !strings.Contains(content, "[[blocks:T001-full-name]]") {
		t.Errorf("expected [[blocks:T001-full-name]] in body, got:\n%s", content)
	}
}

func TestFixPipeline_InferFromSiblings(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem":     "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  tipo:\n    type: enum\n    values: [a, b, c]\n    required: true\n",
		"dir/f1.md": "---\ntipo: a\n---\n# F1\n",
		"dir/f2.md": "---\ntipo: a\n---\n# F2\n",
		"dir/f3.md": "---\ntipo: a\n---\n# F3\n",
		"dir/f4.md": "---\n\n---\n# F4 missing tipo\n",
	})

	ctx := context.Background()
	records := scanAndAggregate(t, ctx, root)
	allErrs := collectErrors(ctx, root, records)
	effective := richestStem(root, records)

	report := proposal.Analyze(records, effective, allErrs)
	p := requireProposal(t, report, proposal.InferFromSiblings)
	if p.Value != "a" {
		t.Errorf("infer value = %q, want a", p.Value)
	}

	if _, err := fix.ApplyProposals(ctx, report, root, records, false); err != nil {
		t.Fatalf("apply: %v", err)
	}

	records2 := scanAndAggregate(t, ctx, root)
	allErrs2 := collectErrors(ctx, root, records2)
	if len(allErrs2) > 0 {
		t.Errorf("re-validate still has errors: %v", allErrs2)
	}
}

func TestFixPipeline_CorrectOutlier(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem":     "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  tipo:\n    type: enum\n    values: [a, b]\n    required: true\n",
		"dir/f1.md": "---\ntipo: a\n---\n# F1\n",
		"dir/f2.md": "---\ntipo: a\n---\n# F2\n",
		"dir/f3.md": "---\ntipo: a\n---\n# F3\n",
		"dir/f4.md": "---\ntipo: a\n---\n# F4\n",
		"dir/f5.md": "---\ntipo: a\n---\n# F5\n",
		"dir/f6.md": "---\ntipo: b\n---\n# F6 outlier\n",
	})

	ctx := context.Background()
	records := scanAndAggregate(t, ctx, root)
	allErrs := collectErrors(ctx, root, records)
	effective := richestStem(root, records)

	report := proposal.Analyze(records, effective, allErrs)
	p := requireProposal(t, report, proposal.CorrectOutlier)
	if p.From != "b" || p.To != "a" {
		t.Errorf("outlier: %q -> %q, want b -> a", p.From, p.To)
	}

	if _, err := fix.ApplyProposals(ctx, report, root, records, false); err != nil {
		t.Fatalf("apply: %v", err)
	}

	records2 := scanAndAggregate(t, ctx, root)
	allErrs2 := collectErrors(ctx, root, records2)
	if len(allErrs2) > 0 {
		t.Errorf("re-validate still has errors: %v", allErrs2)
	}
}

func TestFixPipeline_ExtendEnum(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  tipo:\n    type: enum\n    values: [a, b]\n    required: true\n",
		"f1.md": "---\ntipo: nuevo\n---\n# F1\n",
		"f2.md": "---\ntipo: nuevo\n---\n# F2\n",
	})

	ctx := context.Background()
	records := scanAndAggregate(t, ctx, root)
	allErrs := collectErrors(ctx, root, records)
	effective := richestStem(root, records)

	report := proposal.Analyze(records, effective, allErrs)
	p := requireProposal(t, report, proposal.ExtendEnum)
	if p.Value != "nuevo" {
		t.Errorf("extend value = %q, want nuevo", p.Value)
	}

	// ExtendEnum is schema-surface; apply schema proposals via ApplySchemaProposals.
	if err := fix.ApplySchemaProposals(ctx, report, root); err != nil {
		t.Fatalf("apply schema proposals: %v", err)
	}

	// Re-validate: the enum was extended so the value is now valid.
	records2 := scanAndAggregate(t, ctx, root)
	allErrs2 := collectErrors(ctx, root, records2)
	if len(allErrs2) > 0 {
		t.Errorf("re-validate still has errors: %v", allErrs2)
	}
}

// --- Propagation Tests ---

func TestFixPipeline_PropagateSingleLevel(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem":          "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: enum\n    values: [Pending, Completed]\n    required: true\naggregate:\n  estado: |\n    all(descendants, {.estado == \"Completed\"}) ? \"Completed\" : \"Pending\"\n",
		"S001/README.md": "---\nestado: Pending\n---\n# Story\n",
		"S001/T001.md":   "---\nestado: Completed\n---\n# Task 1\n",
		"S001/T002.md":   "---\nestado: Completed\n---\n# Task 2\n",
	})

	ctx := context.Background()

	// Step 1: scan + derive + validate + propose + propagate.
	records := scanAndAggregate(t, ctx, root)
	allErrs := collectErrors(ctx, root, records)
	effective := richestStem(root, records)

	report := proposal.Analyze(records, effective, allErrs)
	propProposals := proposal.DetectPropagateAggregate(records, effective)
	report.Proposals = append(report.Proposals, propProposals...)

	if len(propProposals) != 1 {
		t.Fatalf("got %d propagate proposals, want 1", len(propProposals))
	}

	// Step 2: apply fix.
	if _, err := fix.ApplyProposals(ctx, report, root, records, false); err != nil {
		t.Fatalf("apply: %v", err)
	}

	// Step 3: re-read and verify README was updated.
	content := readFile(t, root, "S001/README.md")
	if !strings.Contains(content, "estado: Completed") {
		t.Errorf("README not updated to Completed:\n%s", content)
	}

	// Step 4: re-validate -> 0 aggregate errors.
	records2 := scanAndAggregate(t, ctx, root)
	allErrs2 := collectErrors(ctx, root, records2)
	for path, errs := range allErrs2 {
		for _, e := range errs {
			if e.Rule == "aggregate" {
				t.Errorf("still has aggregate error after fix: %s: %s", path, e.Message)
			}
		}
	}
}

func TestFixPipeline_PropagateDeepHierarchy(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: enum\n    values: [Pending, Completed]\n    required: true\naggregate:\n  estado: |\n    all(descendants, {.estado == \"Completed\"}) ? \"Completed\" : \"Pending\"\n",
		// Epic level
		"E01/README.md": "---\nestado: Pending\n---\n# Epic\n",
		// Feature level
		"E01/F01/README.md": "---\nestado: Pending\n---\n# Feature\n",
		// Story level
		"E01/F01/S001/README.md": "---\nestado: Pending\n---\n# Story\n",
		// Tasks — all completed
		"E01/F01/S001/T001.md": "---\nestado: Completed\n---\n# Task 1\n",
		"E01/F01/S001/T002.md": "---\nestado: Completed\n---\n# Task 2\n",
	})

	ctx := context.Background()

	// Aggregation is bottom-up, so we may need multiple passes.
	// Pass 1: Story README gets updated.
	for pass := 0; pass < 3; pass++ {
		records := scanAndAggregate(t, ctx, root)
		allErrs := collectErrors(ctx, root, records)
		effective := richestStem(root, records)

		report := proposal.Analyze(records, effective, allErrs)
		propProposals := proposal.DetectPropagateAggregate(records, effective)
		report.Proposals = append(report.Proposals, propProposals...)

		if len(propProposals) == 0 {
			break // no more stale values
		}

		if _, err := fix.ApplyProposals(ctx, report, root, records, false); err != nil {
			t.Fatalf("apply pass %d: %v", pass, err)
		}
	}

	// Verify all READMEs at Story, Feature, Epic levels are Completed.
	for _, path := range []string{"E01/F01/S001/README.md", "E01/F01/README.md", "E01/README.md"} {
		content := readFile(t, root, path)
		if !strings.Contains(content, "estado: Completed") {
			t.Errorf("%s not updated to Completed:\n%s", path, content)
		}
	}

	// Final validate: 0 aggregate errors.
	records := scanAndAggregate(t, ctx, root)
	allErrs := collectErrors(ctx, root, records)
	for path, errs := range allErrs {
		for _, e := range errs {
			if e.Rule == "aggregate" {
				t.Errorf("still has aggregate error after fix: %s: %s", path, e.Message)
			}
		}
	}
}

// --- Validation Rule Tests ---

func TestValidateRule_NonEmpty(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  titulo:\n    type: string\nvalidate:\n  - rule: non_empty\n    field: titulo\n",
		"a.md":  "---\ntitulo: \"\"\n---\n# Empty title\n",
	})

	ctx := context.Background()
	records := scanAndAggregate(t, ctx, root)
	allErrs := collectErrors(ctx, root, records)

	found := false
	for _, errs := range allErrs {
		for _, e := range errs {
			if e.Rule == "non_empty" && e.Field == "titulo" {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("expected non_empty error for titulo, got: %v", allErrs)
	}
}

func TestValidateRule_Requires(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  tipo:\n    type: enum\n    values: [software-module, documentation]\n    required: true\n  ejecutable_en:\n    type: string\nvalidate:\n  - rule: requires\n    if:\n      tipo: software-module\n    then:\n      fields: [ejecutable_en]\n",
		"a.md":  "---\ntipo: software-module\n---\n# Missing ejecutable_en\n",
	})

	ctx := context.Background()
	records := scanAndAggregate(t, ctx, root)
	allErrs := collectErrors(ctx, root, records)

	found := false
	for _, errs := range allErrs {
		for _, e := range errs {
			if e.Rule == "requires" && e.Field == "ejecutable_en" {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("expected requires error for ejecutable_en, got: %v", allErrs)
	}
}

// --- Round-Trip Test ---

func TestFixRoundTrip_AggregateStale(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem":          "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: enum\n    values: [Pending, Completed]\n    required: true\naggregate:\n  estado: |\n    all(descendants, {.estado == \"Completed\"}) ? \"Completed\" : \"Pending\"\n",
		"S001/README.md": "---\nestado: Pending\n---\n# Story\n",
		"S001/T001.md":   "---\nestado: Completed\n---\n# Task 1\n",
		"S001/T002.md":   "---\nestado: Completed\n---\n# Task 2\n",
	})

	ctx := context.Background()

	// Step 1: Validate → expect aggregate error.
	records := scanAndAggregate(t, ctx, root)
	allErrs := collectErrors(ctx, root, records)

	var aggErr bool
	for _, errs := range allErrs {
		for _, e := range errs {
			if e.Rule == "aggregate" {
				aggErr = true
			}
		}
	}
	if !aggErr {
		t.Fatal("expected aggregate error, got none")
	}

	// Step 2: Fix → correct the stale value.
	// The aggregate error is detected by validate, but fix uses proposal.Analyze.
	// For aggregate mismatches, we need infer_from_children proposal (README missing correct estado).
	// Since the README has estado but it's wrong, we manually correct it for this round-trip.
	// The aggregate validation catches it; the user/fix can then correct.
	// For now, manually fix to verify re-validate passes.
	readmePath := filepath.Join(root, "S001/README.md")
	if err := os.WriteFile(readmePath, []byte("---\nestado: Completed\n---\n# Story\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Step 3: Re-validate → 0 aggregate errors.
	records2 := scanAndAggregate(t, ctx, root)
	allErrs2 := collectErrors(ctx, root, records2)

	for path, errs := range allErrs2 {
		for _, e := range errs {
			if e.Rule == "aggregate" {
				t.Errorf("still has aggregate error after fix: %s: %s", path, e.Message)
			}
		}
	}
}
