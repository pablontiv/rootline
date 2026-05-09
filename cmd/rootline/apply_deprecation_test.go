package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestApplyDeprecationWarning verifies that rootline apply outputs a deprecation warning to stderr.
func TestApplyDeprecationWarning(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	stemContent := `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Completed]
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent), 0644)
	mustWriteFile(t, filepath.Join(dir, "task.md"), []byte("---\nestado: Pending\n---\n# Task\n"), 0644)

	// Create a minimal analyze report
	analyzeReport := map[string]any{
		"version": 1,
		"path":    dir,
		"categories": []map[string]any{
			{
				"id":   "field_types",
				"name": "Field Type Inference",
				"inferences": []map[string]any{
					{
						"type":           "field_type",
						"field":          "estado",
						"value":          "enum",
						"message":        "field 'estado' inferred as enum",
						"source":         "",
						"requires_agent": false,
					},
				},
				"inference_count": 1,
			},
		},
		"summary": map[string]any{
			"total_inferences": 1,
			"engine_resolved":  1,
			"agent_required":   0,
			"incremental":      false,
		},
	}

	reportData, err := json.Marshal(analyzeReport)
	if err != nil {
		t.Fatalf("failed to marshal report: %v", err)
	}

	reportFile := filepath.Join(dir, "report.json")
	if err := os.WriteFile(reportFile, reportData, 0644); err != nil {
		t.Fatalf("failed to write report: %v", err)
	}

	mustChdir(t, dir)

	// Run apply with the report and capture both stdout and stderr
	resetFlags()
	out, err := runCmd(t, "apply", reportFile)

	// The command should succeed (backward compatibility)
	if err != nil {
		t.Fatalf("apply command failed: %v", err)
	}

	// The output should contain the deprecation warning
	if !strings.Contains(out, "deprecated") {
		t.Errorf("expected deprecation warning in output, got:\n%s", out)
	}
	if !strings.Contains(out, "rootline schema apply") {
		t.Errorf("expected 'rootline schema apply' in deprecation message, got:\n%s", out)
	}
	if !strings.Contains(out, "rootline repair apply") {
		t.Errorf("expected 'rootline repair apply' in deprecation message, got:\n%s", out)
	}
	if !strings.Contains(out, "rootline fix --all") {
		t.Errorf("expected 'rootline fix --all' in deprecation message, got:\n%s", out)
	}
}

// TestApplyBackwardCompatibility verifies that rootline apply still functions despite deprecation.
func TestApplyBackwardCompatibility(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	stemContent := `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Completed, InvalidValue]
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent), 0644)
	mustWriteFile(t, filepath.Join(dir, "task.md"), []byte("---\nestado: Pending\n---\n# Task\n"), 0644)

	// Create a report with a data correction proposal (correct_value)
	analyzeReport := map[string]any{
		"version": 1,
		"path":    dir,
		"categories": []map[string]any{
			{
				"id":   "test_category",
				"name": "Test Category",
				"inferences": []map[string]any{
					{
						"type":           "correct_value",
						"field":          "estado",
						"value":          "Pending",
						"message":        "correcting estado value",
						"source":         filepath.Join(dir, "task.md"),
						"requires_agent": false,
					},
				},
				"inference_count": 1,
			},
		},
		"summary": map[string]any{
			"total_inferences": 1,
			"engine_resolved":  1,
			"agent_required":   0,
			"incremental":      false,
		},
	}

	reportData, err := json.Marshal(analyzeReport)
	if err != nil {
		t.Fatalf("failed to marshal report: %v", err)
	}

	reportFile := filepath.Join(dir, "report.json")
	if err := os.WriteFile(reportFile, reportData, 0644); err != nil {
		t.Fatalf("failed to write report: %v", err)
	}

	mustChdir(t, dir)

	resetFlags()
	out, err := runCmd(t, "apply", reportFile, "--output", "json")

	// The command should succeed
	if err != nil {
		t.Fatalf("apply command failed: %v (output: %s)", err, out)
	}

	// The warning is printed to stderr (via cmd.ErrOrStderr) but runCmd captures both.
	// Extract the JSON part (it comes after the warning message).
	// The warning message should be present (contains "deprecated")
	if !strings.Contains(out, "deprecated") {
		t.Errorf("expected deprecation warning in output")
	}

	// Find where the JSON starts
	jsonStartIdx := strings.Index(out, "{")
	if jsonStartIdx == -1 {
		t.Fatalf("no JSON found in output: %s", out)
	}

	jsonStr := out[jsonStartIdx:]

	// The output should be valid JSON (backward compatibility)
	var result map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Fatalf("expected JSON output from apply, got: %s", jsonStr)
	}

	// Verify the result has expected structure
	if _, ok := result["applied"]; !ok {
		t.Errorf("expected 'applied' field in result")
	}
}
