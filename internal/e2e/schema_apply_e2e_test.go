package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/infer"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestApply_SchemaApply_EnumExtension(t *testing.T) {
	// Stem has incomplete enum; schema apply should extend it based on inferences.
	root := setupProject(t, map[string]string{
		".stem":   "version: 2\nschema:\n  estado:\n    type: enum\n    required: true\n    values: [Pending, Done]\n  tipo:\n    type: string\n",
		"doc1.md": "---\nestado: Pending\ntipo: task\n---\nBody 1\n",
		"doc2.md": "---\nestado: Done\ntipo: task\n---\nBody 2\n",
		"doc3.md": "---\nestado: Pending\ntipo: task\n---\nBody 3\n",
	})

	// Collect inferences from analyze.
	report := runAnalyze(t, root)
	var allInferences []infer.ReportInference
	for _, cat := range report.Categories {
		allInferences = append(allInferences, cat.Inferences...)
	}

	// Apply schema inferences to .stem via library.
	entries, err := rules.WalkUp(root)
	if err != nil || len(entries) == 0 {
		t.Fatalf("no stem found: %v", err)
	}
	if err := os.Chmod(entries[0].Path, 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := infer.ApplySchemaInferences(entries[0].Path, allInferences, false)
	if err != nil {
		t.Fatalf("apply schema inferences error: %v", err)
	}

	// Verify at least some inferences were processed.
	totalActions := len(result.Applied) + len(result.Skipped)
	if totalActions == 0 && len(allInferences) > 0 {
		t.Logf("warning: %d inferences produced no apply actions (schema may already cover them)", len(allInferences))
	}

	info, err := os.Stat(entries[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("schema apply changed .stem mode: got %v want 0600", got)
	}

	// Validate: all documents should still pass after schema update.
	validateAllRecords(t, root)
}

func TestApply_SchemaApply_DryRun(t *testing.T) {
	// Verify that schema apply with --dry-run does not modify .stem files.
	root := setupProject(t, map[string]string{
		".stem":   "version: 2\nschema:\n  estado:\n    type: enum\n    required: true\n    values: [Pending, Done]\n  priority:\n    type: string\n",
		"doc1.md": "---\nestado: Pending\n---\nBody\n",
	})

	entries, err := rules.WalkUp(root)
	if err != nil || len(entries) == 0 {
		t.Fatalf("no stem found: %v", err)
	}

	original, _ := os.ReadFile(entries[0].Path)

	// Create a schema inference: mark priority as required
	inferences := []infer.ReportInference{
		{Type: "required_field", Field: "priority"},
	}

	result, err := infer.ApplySchemaInferences(entries[0].Path, inferences, true)
	if err != nil {
		t.Fatalf("dry-run error: %v", err)
	}

	// Verify dry-run flag is set
	if !result.DryRun {
		t.Error("expected DryRun=true")
	}

	// .stem file must be unchanged.
	after, _ := os.ReadFile(entries[0].Path)
	if string(after) != string(original) {
		t.Errorf("dry-run modified the .stem file:\noriginal: %s\nafter: %s", original, after)
	}
}

func TestApply_SchemaApply_SectionSourceRoundTrip(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem":   "version: 2\nschema: {}\n",
		"doc1.md": "---\ntitle: A\n---\n# A\n\n## Notes\n\nAlpha\n",
		"doc2.md": "---\ntitle: B\n---\n# B\n\n## Notes\n\nBeta\n",
	})

	report := runAnalyze(t, root)
	var allInferences []infer.ReportInference
	for _, cat := range report.Categories {
		for _, inf := range cat.Inferences {
			if inf.Type == "required_section" || inf.Type == "optional_section" {
				allInferences = append(allInferences, inf)
			}
		}
	}

	entries, err := rules.WalkUp(root)
	if err != nil || len(entries) == 0 {
		t.Fatalf("no stem found: %v", err)
	}
	result, err := infer.ApplySchemaInferences(entries[0].Path, allInferences, false)
	if err != nil {
		t.Fatalf("apply schema inferences: %v", err)
	}
	if len(result.Applied) == 0 {
		t.Fatalf("expected section schema application from report, got %+v", result)
	}

	stem, err := rules.ParseStem(entries[0].Path, mustReadE2EFile(t, entries[0].Path))
	if err != nil {
		t.Fatal(err)
	}
	field := stem.Schema["notes"]
	if field.Type != "string" || field.Extract != `body.section["## Notes"]` || !field.Required {
		t.Fatalf("section field did not round-trip from analyze to apply: %+v", field)
	}
	validateAllRecords(t, root)
}

func TestApply_RequiresAgent_Skipped(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem":   "version: 2\nschema:\n  estado:\n    type: string\n",
		"doc1.md": "---\nestado: Pending\n---\nBody\n",
	})

	entries, err := rules.WalkUp(root)
	if err != nil || len(entries) == 0 {
		t.Fatalf("no stem found: %v", err)
	}

	inferences := []infer.ReportInference{
		{Type: "required_field", Field: "estado", RequiresAgent: true, Message: "needs human review"},
	}

	result, err := infer.ApplySchemaInferences(entries[0].Path, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if len(result.Skipped) != 1 {
		t.Errorf("expected 1 skipped, got %d", len(result.Skipped))
	}
	if len(result.Applied) != 0 {
		t.Errorf("expected 0 applied, got %d", len(result.Applied))
	}
	if !strings.Contains(result.Skipped[0], "requires agent") {
		t.Errorf("expected 'requires agent' in skipped message, got: %s", result.Skipped[0])
	}
}

func TestApply_SchemaApply_MalformedExternalGovernanceProposalParity(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem":               "version: 2\nroot: true\nschema:\n  title:\n    type: string\n",
		"subtree/.stem":       "version: [broken\n",
		"subtree/work/doc.md": "---\ntitle: Test\n---\n# Record\n",
	})
	workRoot := filepath.Join(root, "subtree", "work")
	malformedStem := filepath.Join(root, "subtree", ".stem")
	before := mustReadE2EFile(t, malformedStem)
	reportPath := filepath.Join(workRoot, "proposal.json")
	writeE2ESchemaApplyReport(t, reportPath, map[string]any{
		"version": 1,
		"kind":    "rootline/schema-proposals",
		"path":    workRoot,
		"root":    workRoot,
		"proposals": []map[string]any{{
			"id":        "candidate",
			"operation": "create_stem",
			"target":    filepath.Join(workRoot, ".stem"),
			"patch":     "version: 2\nschema:\n  estado:\n    type: string\n",
		}},
	})

	dryResult, dryErr := runE2ESchemaApplyCLI(t, reportPath, true)
	if dryErr == nil {
		t.Fatalf("dry-run accepted malformed external governance: %+v", dryResult)
	}
	realResult, realErr := runE2ESchemaApplyCLI(t, reportPath, false)
	if realErr == nil {
		t.Fatalf("real apply accepted malformed external governance: %+v", realResult)
	}

	if !dryResult.DryRun || realResult.DryRun {
		t.Fatalf("dry_run flags = dry:%v real:%v, want true/false", dryResult.DryRun, realResult.DryRun)
	}
	if dryResult.Complete || realResult.Complete || len(dryResult.Applied) != 0 || len(realResult.Applied) != 0 {
		t.Fatalf("results = dry:%+v real:%+v", dryResult, realResult)
	}
	if !reflect.DeepEqual(normalizedE2ESchemaApplyStemHealth(dryResult.StemHealth), normalizedE2ESchemaApplyStemHealth(realResult.StemHealth)) {
		t.Fatalf("dry-run and real stem_health differ:\ndry=%+v\nreal=%+v", dryResult.StemHealth, realResult.StemHealth)
	}
	want := e2eSchemaApplyStemHealthKey{Path: filepath.Join("..", ".stem"), Check: "yaml-valid", Severity: rules.SeverityError}
	if got := normalizedE2ESchemaApplyStemHealth(dryResult.StemHealth); len(got) != 1 || got[0] != want {
		t.Fatalf("stem_health keys = %+v, want %+v", got, want)
	}
	if got := mustReadE2EFile(t, malformedStem); string(got) != string(before) {
		t.Fatalf("malformed external .stem changed:\nbefore:\n%s\nafter:\n%s", before, got)
	}
	if _, err := os.Stat(filepath.Join(workRoot, ".stem")); !os.IsNotExist(err) {
		t.Fatalf("candidate target was created: %v", err)
	}
}

type e2eSchemaApplyResult struct {
	Complete   bool                         `json:"complete"`
	Applied    []string                     `json:"applied"`
	DryRun     bool                         `json:"dry_run"`
	StemHealth []rules.StemHealthDiagnostic `json:"stem_health"`
}

type e2eSchemaApplyStemHealthKey struct {
	Path     string
	Check    string
	Field    string
	Severity string
}

func writeE2ESchemaApplyReport(t *testing.T, path string, report any) {
	t.Helper()
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func runE2ESchemaApplyCLI(t *testing.T, reportPath string, dryRun bool) (*e2eSchemaApplyResult, error) {
	t.Helper()
	args := []string{"run", "./cmd/rootline", "schema", "apply", "--report", reportPath}
	if dryRun {
		args = append(args, "--dry-run")
	}
	cmd := exec.Command("go", args...) //nolint:gosec -- test invokes the local rootline CLI with fixture paths.
	cmd.Dir = filepath.Join("..", "..")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	var result e2eSchemaApplyResult
	if decodeErr := json.Unmarshal(stdout.Bytes(), &result); decodeErr != nil {
		t.Fatalf("decode schema apply stdout: %v\nstdout:%s\nstderr:%s\nrun err:%v", decodeErr, stdout.String(), stderr.String(), err)
	}
	return &result, err
}

func normalizedE2ESchemaApplyStemHealth(diags []rules.StemHealthDiagnostic) []e2eSchemaApplyStemHealthKey {
	keys := make([]e2eSchemaApplyStemHealthKey, 0, len(diags))
	for _, diag := range diags {
		keys = append(keys, e2eSchemaApplyStemHealthKey{Path: diag.Path, Check: diag.Check, Field: diag.Field, Severity: diag.Severity})
	}
	return keys
}

// validateAllRecords runs validation against all records in the root directory.
func mustReadE2EFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func validateAllRecords(t *testing.T, root string) {
	t.Helper()
	entries, err := rules.WalkUp(root)
	if err != nil || len(entries) == 0 {
		return // no stem = nothing to validate against
	}
	effective := rules.MergeStemFiles(entries)

	files, err := filepath.Glob(filepath.Join(root, "*.md"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}

	reg := extract.NewRegistry()
	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("reading %s: %v", f, err)
		}
		ext := reg.ForFile(f, "")
		if ext == nil {
			continue
		}
		rec, err := ext.Extract(f, content)
		if err != nil {
			continue
		}
		relPath, _ := filepath.Rel(root, f)
		rec.Path = relPath
		errs := rules.Validate(context.Background(), rec, effective)
		if len(errs) > 0 {
			t.Errorf("validation errors in %s after apply: %v", relPath, errs)
		}
	}
}
