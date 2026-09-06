package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/fix"
	"github.com/pablontiv/rootline/internal/proposal"
)

const emptyFrontmatterStem = `version: 2
root: true
scope:
  match: "*.md"
schema:
  alpha:
    type: string
    required: true
    default: alpha-default
`

const emptyFrontmatterDocument = "---\n---\n# Empty frontmatter\n\nBody\n\n---\n\ntail\n"
const repairedEmptyFrontmatterDocument = "---\nalpha: alpha-default\n---\n# Empty frontmatter\n\nBody\n\n---\n\ntail\n"

func newEmptyFrontmatterProject(t *testing.T) (root, target string, stemBefore []byte) {
	t.Helper()

	root = t.TempDir()
	target = filepath.Join(root, "a.md")
	mustWriteFile(t, filepath.Join(root, ".stem"), []byte(emptyFrontmatterStem), 0o644)
	mustWriteFile(t, target, []byte(emptyFrontmatterDocument), 0o600)
	return root, target, mustReadFile(t, filepath.Join(root, ".stem"))
}

func assertEmptyFrontmatterRepair(t *testing.T, root, target string, stemBefore []byte) {
	t.Helper()

	if got := string(mustReadFile(t, target)); got != repairedEmptyFrontmatterDocument {
		t.Errorf("repaired document = %q, want %q", got, repairedEmptyFrontmatterDocument)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat target: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("target mode = %04o, want 0600", info.Mode().Perm())
	}
	if got := mustReadFile(t, filepath.Join(root, ".stem")); string(got) != string(stemBefore) {
		t.Errorf(".stem changed:\nwant:\n%s\ngot:\n%s", stemBefore, got)
	}
	if out, err := runCmd(t, "validate", target, "--output", "json"); err != nil {
		t.Fatalf("post-repair validate failed: %v\n%s", err, out)
	}
}

func TestFixSingleRepairsEmptyFrontmatter(t *testing.T) {
	root, target, stemBefore := newEmptyFrontmatterProject(t)
	original := mustReadFile(t, target)

	dryOut, err := runCmd(t, "fix", "--dry-run", target)
	if err != nil {
		t.Fatalf("fix dry-run: %v\n%s", err, dryOut)
	}
	if !strings.Contains(dryOut, `would add alpha="alpha-default"`) {
		t.Errorf("dry-run did not preview the field addition: %s", dryOut)
	}
	if got := mustReadFile(t, target); string(got) != string(original) {
		t.Errorf("dry-run changed target:\nwant:\n%s\ngot:\n%s", original, got)
	}

	out, err := runCmd(t, "fix", target)
	if err != nil {
		t.Fatalf("fix apply: %v\n%s", err, out)
	}
	if !strings.Contains(out, `added alpha="alpha-default"`) || !strings.Contains(out, "Fixed: 1 fields added") {
		t.Errorf("fix did not report the applied field: %s", out)
	}
	assertEmptyFrontmatterRepair(t, root, target, stemBefore)

	beforeSecond := mustReadFile(t, target)
	secondOut, err := runCmd(t, "fix", target)
	if err != nil {
		t.Fatalf("second fix: %v\n%s", err, secondOut)
	}
	if !strings.Contains(secondOut, "no errors to fix") {
		t.Errorf("second fix was not reported as a no-op: %s", secondOut)
	}
	if got := mustReadFile(t, target); string(got) != string(beforeSecond) {
		t.Errorf("second fix changed target:\nwant:\n%s\ngot:\n%s", beforeSecond, got)
	}
}

func TestFixAllRepairsEmptyFrontmatter(t *testing.T) {
	root, target, stemBefore := newEmptyFrontmatterProject(t)
	original := mustReadFile(t, target)

	dryOut, err := runCmd(t, "fix", "--all", root, "--dry-run", "--output", "json")
	if err != nil {
		t.Fatalf("fix --all dry-run: %v\n%s", err, dryOut)
	}
	var preview proposal.Report
	if err := json.Unmarshal([]byte(dryOut), &preview); err != nil {
		t.Fatalf("decode fix preview: %v\n%s", err, dryOut)
	}
	if preview.Version != 1 || preview.Kind != "rootline/proposals" || len(preview.Proposals) != 1 {
		t.Fatalf("fix preview = %+v, want one v1 repair proposal", preview)
	}
	if got := mustReadFile(t, target); string(got) != string(original) {
		t.Errorf("fix --all dry-run changed target:\nwant:\n%s\ngot:\n%s", original, got)
	}

	out, err := runCmd(t, "fix", "--all", root, "--output", "json")
	if err != nil {
		t.Fatalf("fix --all apply: %v\n%s", err, out)
	}
	var batch BatchFixResult
	if err := json.Unmarshal([]byte(out), &batch); err != nil {
		t.Fatalf("decode fix result: %v\n%s", err, out)
	}
	if batch.Version != 1 || batch.Kind != "rootline/fix-batch" || batch.Summary.Fixed != 1 {
		t.Fatalf("fix result = %+v, want one fixed record in v1 envelope", batch)
	}
	if len(batch.Results) != 1 || !batch.Results[0].Fixed || batch.Results[0].FieldsAdded != 1 {
		t.Errorf("record result = %+v, want one applied field", batch.Results)
	}
	assertEmptyFrontmatterRepair(t, root, target, stemBefore)

	secondOut, err := runCmd(t, "fix", "--all", root, "--output", "json")
	if err != nil {
		t.Fatalf("second fix --all: %v\n%s", err, secondOut)
	}
	var second BatchFixResult
	if err := json.Unmarshal([]byte(secondOut), &second); err != nil {
		t.Fatalf("decode second fix result: %v\n%s", err, secondOut)
	}
	if second.Summary.Fixed != 0 {
		t.Errorf("second fix --all fixed %d records, want 0: %s", second.Summary.Fixed, secondOut)
	}
}

func TestRepairApplyRepairsEmptyFrontmatter(t *testing.T) {
	root, target, stemBefore := newEmptyFrontmatterProject(t)
	original := mustReadFile(t, target)

	previewOut, err := runCmd(t, "fix", "--all", root, "--dry-run", "--output", "json")
	if err != nil {
		t.Fatalf("generate repair report: %v\n%s", err, previewOut)
	}
	reportFile := filepath.Join(t.TempDir(), "repair.json")
	mustWriteFile(t, reportFile, []byte(previewOut), 0o644)

	dryOut, err := runCmd(t, "repair", "apply", "--report", reportFile, "--dry-run", "--output", "json")
	if err != nil {
		t.Fatalf("repair apply dry-run: %v\n%s", err, dryOut)
	}
	var dry fix.RepairResult
	if err := json.Unmarshal([]byte(dryOut), &dry); err != nil {
		t.Fatalf("decode repair dry-run: %v\n%s", err, dryOut)
	}
	if dry.Version != 1 || dry.Kind != "rootline/repair" || !dry.Complete || len(dry.Changed) != 1 {
		t.Fatalf("repair dry-run = %+v, want one complete v1 preview", dry)
	}
	if got := mustReadFile(t, target); string(got) != string(original) {
		t.Errorf("repair dry-run changed target:\nwant:\n%s\ngot:\n%s", original, got)
	}

	out, err := runCmd(t, "repair", "apply", "--report", reportFile, "--output", "json")
	if err != nil {
		t.Fatalf("repair apply: %v\n%s", err, out)
	}
	var result fix.RepairResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("decode repair result: %v\n%s", err, out)
	}
	if result.Version != 1 || result.Kind != "rootline/repair" || !result.Complete || len(result.Changed) != 1 || len(result.RolledBack) != 0 {
		t.Fatalf("repair result = %+v, want one complete applied change", result)
	}
	assertEmptyFrontmatterRepair(t, root, target, stemBefore)

	secondOut, err := runCmd(t, "repair", "apply", "--report", reportFile, "--output", "json")
	if err != nil {
		t.Fatalf("second repair apply: %v\n%s", err, secondOut)
	}
	var second fix.RepairResult
	if err := json.Unmarshal([]byte(secondOut), &second); err != nil {
		t.Fatalf("decode second repair result: %v\n%s", err, secondOut)
	}
	if !second.Complete || len(second.Changed) != 0 || len(second.Skipped) != 1 {
		t.Errorf("second repair result = %+v, want one idempotent skip", second)
	}
}
