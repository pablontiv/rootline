package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/infer"
)

func writeAnalyzeBodyFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	for i := 1; i <= 3; i++ {
		context := ""
		if i <= 2 {
			context = "## Context\n\nShared context.\n\n"
		}
		body := fmt.Sprintf(`---
title: Record %d
---
# Record %d

%s## Invariantes

- INV00%d: must hold.

## Dependencias

- Requires F01.
`, i, i, context, i)
		mustWriteFile(t, filepath.Join(dir, fmt.Sprintf("F%02d-record.md", i)), []byte(body), 0o644)
	}

	return dir
}

func analyzeCategories(t *testing.T, dir string, threshold string) map[string]infer.CategoryResult {
	t.Helper()
	out, err := runCmd(t, "analyze", dir, "--threshold", threshold, "-o", "json")
	if err != nil {
		t.Fatalf("analyze failed: %v\n%s", err, out)
	}
	var report infer.AnalyzeReport
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("decode analyze report: %v\n%s", err, out)
	}
	categories := make(map[string]infer.CategoryResult, len(report.Categories))
	for _, category := range report.Categories {
		categories[category.ID] = category
	}
	return categories
}

func TestAnalyzeRunsBodyAwareDetectors(t *testing.T) {
	dir := writeAnalyzeBodyFixture(t)
	categories := analyzeCategories(t, dir, "0.60")

	for _, id := range []string{"section_patterns", "invariants", "formal_dependencies"} {
		if categories[id].InferenceCount == 0 {
			t.Errorf("expected %s to produce inferences", id)
		}
	}
}

func TestAnalyzeThresholdControlsSectionPatterns(t *testing.T) {
	dir := writeAnalyzeBodyFixture(t)
	low := analyzeCategories(t, dir, "0.60")
	high := analyzeCategories(t, dir, "0.90")

	if low["section_patterns"].InferenceCount <= high["section_patterns"].InferenceCount {
		t.Fatalf("expected lower threshold to produce more section patterns: low=%d high=%d",
			low["section_patterns"].InferenceCount, high["section_patterns"].InferenceCount)
	}
}
