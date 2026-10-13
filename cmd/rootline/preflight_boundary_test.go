package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/proposal"
	"github.com/spf13/cobra"
)

// TestValidateSingleFileNoBoundaryStemEmitsEnvelope verifies that validate
// emits a proper JSON envelope when a single file target has no .stem anywhere.
// Issue #149: validate <file> should emit envelope, not raw error.
func TestValidateSingleFileNoBoundaryStemEmitsEnvelope(t *testing.T) {
	tmpdir := t.TempDir()
	mdFile := filepath.Join(tmpdir, "test.md")
	if err := os.WriteFile(mdFile, []byte("---\n---\n# Test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Save current dir and change to tmpdir with no .stem
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(oldCwd)
	}()

	if err := os.Chdir(tmpdir); err != nil {
		t.Fatal(err)
	}

	// Run validate <file> -o json
	var stdout bytes.Buffer
	cmd := rootCmd
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"validate", "test.md", "-o", "json"})

	err = cmd.Execute()
	if err == nil {
		t.Fatal("validate should fail with no .stem")
	}

	// Check that stdout has valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("stdout should be valid JSON, got: %s, error: %v", stdout.String(), err)
	}

	// Check envelope structure
	if version, ok := result["version"]; !ok || version != float64(2) {
		t.Errorf("envelope should have version 2, got: %v", version)
	}
	if kind, ok := result["kind"]; !ok || kind != "rootline/validate-batch" {
		t.Errorf("envelope should have kind rootline/validate-batch, got: %v", kind)
	}

	// Check that notices contain scan_failed or schema_resolution_failed
	notices, ok := result["notices"].([]interface{})
	if !ok {
		t.Errorf("envelope should have notices array, got: %v", result["notices"])
	}
	if len(notices) == 0 {
		t.Error("envelope should have at least one notice for the missing schema")
	}
}

// TestValidateAllNoBoundaryStemEmitsEnvelope verifies that validate --all
// emits a proper JSON envelope when there's no declared boundary.
// Issue #149: validate --all should emit envelope, not raw error.
func TestValidateAllNoBoundaryStemEmitsEnvelope(t *testing.T) {
	tmpdir := t.TempDir()

	// Create a .stem without root: true
	stemPath := filepath.Join(tmpdir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nscope:\n  match: \"*.md\"\nschema:\n  titulo:\n    type: string\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a test markdown file
	mdFile := filepath.Join(tmpdir, "test.md")
	if err := os.WriteFile(mdFile, []byte("---\ntitulo: Test\n---\n# Test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Save current dir and change to tmpdir
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(oldCwd)
	}()

	if err := os.Chdir(tmpdir); err != nil {
		t.Fatal(err)
	}

	// Run validate --all . -o json
	var stdout bytes.Buffer
	cmd := rootCmd
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"validate", "--all", ".", "-o", "json"})

	err = cmd.Execute()
	if err == nil {
		t.Fatal("validate should fail with no declared boundary")
	}

	// Check that stdout has valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("stdout should be valid JSON, got: %s, error: %v", stdout.String(), err)
	}

	// Check envelope structure
	if version, ok := result["version"]; !ok || version != float64(2) {
		t.Errorf("envelope should have version 2, got: %v", version)
	}
	if kind, ok := result["kind"]; !ok || kind != "rootline/validate-batch" {
		t.Errorf("envelope should have kind rootline/validate-batch, got: %v", kind)
	}

	// Check that notices contain schema_resolution_failed or similar error
	notices, ok := result["notices"].([]interface{})
	if !ok {
		t.Errorf("envelope should have notices array, got: %v", result["notices"])
	}
	if len(notices) == 0 {
		t.Error("envelope should have at least one notice for the boundary issue")
	}
}

func effectiveTargetTree(t *testing.T, declared bool) string {
	t.Helper()
	dir := t.TempDir()
	stem := "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  titulo:\n    type: string\n"
	if declared {
		stem += "root: true\n"
	}
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stem), 0o644)
	mustWriteFile(t, filepath.Join(dir, "task.md"), []byte("---\ntitulo: Old\n---\n# Task\n"), 0o644)
	return dir
}

func effectiveTargetReport(t *testing.T, reportFile, root, path, value string) {
	t.Helper()
	report := proposal.Report{
		Version: 1,
		Kind:    "rootline/proposals",
		Root:    root,
		Path:    path,
		Proposals: []proposal.Proposal{{
			Type:  proposal.CorrectValue,
			Field: "titulo",
			From:  "Old",
			To:    value,
			Paths: []string{"task.md"},
		}},
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, reportFile, data, 0o644)
}

func TestPreflight_QueryStatsEffectiveTarget(t *testing.T) {
	for _, command := range []string{"query", "stats"} {
		for _, form := range []string{"from", "positional", "positional_over_from"} {
			for _, declared := range []bool{false, true} {
				direction := "governed_cwd_undeclared_target"
				if declared {
					direction = "undeclared_cwd_governed_target"
				}
				t.Run(command+"/"+form+"/"+direction, func(t *testing.T) {
					cwd := effectiveTargetTree(t, !declared)
					target := effectiveTargetTree(t, declared)
					t.Chdir(cwd)
					args := []string{command}
					switch form {
					case "from":
						args = append(args, "--from", target)
					case "positional":
						args = append(args, target)
					case "positional_over_from":
						args = append(args, target, "--from", cwd)
					}
					out, err := runCmd(t, args...)
					if !declared {
						if err == nil || !strings.Contains(err.Error(), "declared boundary") {
							t.Fatalf("expected effective-target boundary rejection, got %v\n%s", err, out)
						}
						return
					}
					if err != nil {
						t.Fatalf("governed effective target must run: %v\n%s", err, out)
					}
					if !strings.Contains(out, `"rootline/`+command+`"`) {
						t.Fatalf("missing command result: %s", out)
					}
				})
			}
		}
	}
}

func TestPreflight_RepairEffectiveTargetPrecedence(t *testing.T) {
	for _, source := range []string{"explicit_root", "report_root", "report_path", "report_directory"} {
		for _, declared := range []bool{false, true} {
			direction := "governed_cwd_undeclared_target"
			if declared {
				direction = "undeclared_cwd_governed_target"
			}
			t.Run(source+"/"+direction, func(t *testing.T) {
				cwd := effectiveTargetTree(t, !declared)
				target := effectiveTargetTree(t, declared)
				t.Chdir(cwd)
				reportFile := filepath.Join(cwd, "report.json")
				reportRoot, reportScanPath := "", ""
				args := []string{"repair", "apply"}
				switch source {
				case "explicit_root":
					args = append(args, "--root", target)
					reportRoot, reportScanPath = cwd, cwd
				case "report_root":
					reportRoot, reportScanPath = target, cwd
				case "report_path":
					var err error
					reportScanPath, err = filepath.Rel(cwd, target)
					if err != nil {
						t.Fatal(err)
					}
				case "report_directory":
					reportFile = filepath.Join(target, "report.json")
				}
				effectiveTargetReport(t, reportFile, reportRoot, reportScanPath, "New")
				args = append(args, "--report", reportFile)
				out, err := runCmd(t, args...)
				if !declared {
					if err == nil || !strings.Contains(err.Error(), "declared boundary") {
						t.Fatalf("expected effective-target boundary rejection, got %v\n%s", err, out)
					}
					if strings.Contains(string(mustReadFile(t, filepath.Join(target, "task.md"))), "titulo: New") {
						t.Fatal("rejected effective target was mutated")
					}
					return
				}
				if err != nil {
					t.Fatalf("governed effective target must run: %v\n%s", err, out)
				}
				var result struct {
					Root    string   `json:"root"`
					Changed []string `json:"changed"`
				}
				if err := json.Unmarshal([]byte(out), &result); err != nil {
					t.Fatal(err)
				}
				if result.Root != target || len(result.Changed) != 1 {
					t.Fatalf("mutation must share preflight root %q, got %s", target, out)
				}
				if !strings.Contains(string(mustReadFile(t, filepath.Join(target, "task.md"))), "titulo: New") {
					t.Fatal("authorized effective target was not mutated")
				}
				if !strings.Contains(string(mustReadFile(t, filepath.Join(cwd, "task.md"))), "titulo: Old") {
					t.Fatal("lower-priority root was mutated")
				}
			})
		}
	}
}

func TestPreflight_RepairUsesAuthorizedSnapshot(t *testing.T) {
	cwd := effectiveTargetTree(t, false)
	target := effectiveTargetTree(t, true)
	other := effectiveTargetTree(t, false)
	t.Chdir(cwd)
	reportFile := filepath.Join(cwd, "report.json")
	effectiveTargetReport(t, reportFile, target, "", "New")

	oldPreRun := repairApplyCmd.PreRunE
	t.Cleanup(func() { repairApplyCmd.PreRunE = oldPreRun })
	repairApplyCmd.PreRunE = func(cmd *cobra.Command, _ []string) error {
		// This hook runs after successful persistent preflight but before RunE.
		effectiveTargetReport(t, reportFile, other, "", "Unauthorized")
		return cmd.Flags().Set("root", other)
	}
	out, err := runCmd(t, "repair", "apply", "--report", reportFile)
	if err != nil {
		t.Fatalf("applying authorized snapshot: %v\n%s", err, out)
	}
	var result struct {
		Root string `json:"root"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatal(err)
	}
	if result.Root != target {
		t.Fatalf("root changed after preflight: %q, want %q", result.Root, target)
	}
	if !strings.Contains(string(mustReadFile(t, filepath.Join(target, "task.md"))), "titulo: New") {
		t.Fatal("original authorized proposal was not applied")
	}
	if !strings.Contains(string(mustReadFile(t, filepath.Join(other, "task.md"))), "titulo: Old") {
		t.Fatal("replacement report escaped preflight authorization")
	}
	if invocation, _ := repairApplyCmd.Context().Value(repairInvocationKey{}).(*repairInvocation); invocation != nil {
		t.Fatal("RunE retained consumed authorization")
	}
}

func TestPreflight_RepairRepeatedInvocationRefreshesAuthorization(t *testing.T) {
	cwd := effectiveTargetTree(t, true)
	t.Chdir(cwd)
	first := effectiveTargetTree(t, true)
	undeclared := effectiveTargetTree(t, false)
	last := effectiveTargetTree(t, true)
	reportFile := filepath.Join(cwd, "report.json")
	for _, target := range []string{first, undeclared, last} {
		effectiveTargetReport(t, reportFile, target, "", "New")
		out, err := runCmd(t, "repair", "apply", "--report", reportFile)
		if target == undeclared {
			if err == nil || !strings.Contains(err.Error(), "declared boundary") {
				t.Fatalf("reused command accepted undeclared report root: %v\n%s", err, out)
			}
		} else if err != nil {
			t.Fatalf("reused command rejected governed root: %v\n%s", err, out)
		}
		if invocation, _ := repairApplyCmd.Context().Value(repairInvocationKey{}).(*repairInvocation); invocation != nil {
			t.Fatal("reused command retained authorization after execution")
		}
	}
	if !strings.Contains(string(mustReadFile(t, filepath.Join(undeclared, "task.md"))), "titulo: Old") {
		t.Fatal("stale authorization allowed mutation")
	}
	if !strings.Contains(string(mustReadFile(t, filepath.Join(last, "task.md"))), "titulo: New") {
		t.Fatal("reused command did not refresh report root")
	}
}

func TestPreflight_RepairDryRunUsesEffectiveTarget(t *testing.T) {
	for _, declared := range []bool{false, true} {
		direction := "governed_cwd_undeclared_target"
		if declared {
			direction = "undeclared_cwd_governed_target"
		}
		t.Run(direction, func(t *testing.T) {
			cwd := effectiveTargetTree(t, !declared)
			target := effectiveTargetTree(t, declared)
			t.Chdir(cwd)
			reportFile := filepath.Join(cwd, "report.json")
			effectiveTargetReport(t, reportFile, target, "", "New")
			out, err := runCmd(t, "repair", "apply", "--report", reportFile, "--dry-run")
			if !declared {
				if err == nil || !strings.Contains(err.Error(), "declared boundary") {
					t.Fatalf("dry-run accepted undeclared effective target: %v\n%s", err, out)
				}
			} else {
				if err != nil {
					t.Fatalf("dry-run rejected governed effective target: %v\n%s", err, out)
				}
				var result struct {
					Root   string `json:"root"`
					DryRun bool   `json:"dry_run"`
				}
				if err := json.Unmarshal([]byte(out), &result); err != nil {
					t.Fatal(err)
				}
				if result.Root != target || !result.DryRun {
					t.Fatalf("dry-run did not share authorized root: %s", out)
				}
			}
			if !strings.Contains(string(mustReadFile(t, filepath.Join(target, "task.md"))), "titulo: Old") {
				t.Fatal("dry-run mutated the effective target")
			}
		})
	}
}

func TestPreflight_RepairReportErrorsBeforeBoundary(t *testing.T) {
	for _, tc := range []struct {
		name   string
		report string
		want   string
	}{
		{"invalid_json", "not JSON", "parsing report"},
		{"unsupported_version", `{"version":2,"kind":"rootline/proposals"}`, "unsupported report version"},
		{"wrong_kind", `{"version":1,"kind":"rootline/analyze"}`, "wrong report kind"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cwd := effectiveTargetTree(t, false)
			t.Chdir(cwd)
			reportFile := filepath.Join(cwd, "report.json")
			mustWriteFile(t, reportFile, []byte(tc.report), 0o644)
			out, err := runCmd(t, "repair", "apply", "--report", reportFile)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected report error before boundary error %q, got %v\n%s", tc.want, err, out)
			}
		})
	}
}
