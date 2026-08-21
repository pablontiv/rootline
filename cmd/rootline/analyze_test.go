package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/infer"
)

func writeAnalyzeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}

func executeAnalyze(t *testing.T, args ...string) []byte {
	t.Helper()
	buf, err := executeAnalyzeErr(t, args...)
	if err != nil {
		t.Fatalf("execute analyze: %v", err)
	}
	return buf
}

func executeAnalyzeErr(t *testing.T, args ...string) ([]byte, error) {
	t.Helper()
	resetFlags()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetArgs(append([]string{"analyze"}, args...))
	err := rootCmd.Execute()
	return bytes.Clone(buf.Bytes()), err
}

func TestSafeRunDetectorReturnsOrdinaryDetectorErrors(t *testing.T) {
	sentinel := errors.New("ordinary detector failure")
	got, err := safeRunDetector(context.Background(), "section_patterns", func() ([]infer.Inference, error) {
		return []infer.Inference{{Type: "partial"}}, sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("safeRunDetector error = %v, want sentinel", err)
	}
	if len(got) != 1 || got[0].Type == "detector_error" {
		t.Fatalf("safeRunDetector rewrote ordinary error result: %+v", got)
	}
}

func TestRunAnalyzePropagatesSectionCollisionError(t *testing.T) {
	root := t.TempDir()
	writeAnalyzeFile(t, filepath.Join(root, "doc.md"), "---\ntitle: Doc\n---\n# Doc\n\n## Notes\nA\n\n### Notes\nB\n")

	_, err := executeAnalyzeErr(t, root, "-o", "json")
	if err == nil {
		t.Fatal("expected analyze to return section collision error")
	}
	msg := err.Error()
	for _, want := range []string{"detector section_patterns", "section field name collision", "## Notes", "### Notes"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("analyze error %q missing %q", msg, want)
		}
	}
}

func TestAnalyzeJSONOutputIsDeterministic(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem": "version: 2\nroot: true\nschema:\n  title: {type: string}\n  status: {type: enum, values: [draft, published]}\n  priority: {type: enum, values: [low, high]}\n",
	})
	for i := 1; i <= 6; i++ {
		state, priority := "draft", "low"
		if i%2 == 0 {
			state, priority = "published", "high"
		}
		writeAnalyzeFile(t, filepath.Join(root, fmt.Sprintf("r%d.md", i)), fmt.Sprintf("---\ntitle: Record %d\nstatus: %s\npriority: %s\nowner: team-%d\n---\n# Record %d\n\nSee [[r%d]].\n", i, state, priority, i%3, i, i%6+1))
	}

	want := executeAnalyze(t, root, "-o", "json")
	var report infer.AnalyzeReport
	if err := json.Unmarshal(want, &report); err != nil {
		t.Fatalf("decode analyze JSON: %v", err)
	}
	if report.Version != 1 || report.Kind != "rootline/analyze" {
		t.Fatalf("contract identity = (%d, %q), want (1, %q)", report.Version, report.Kind, "rootline/analyze")
	}

	var fields []string
	for _, category := range report.Categories {
		if category.ID != "field_types" {
			continue
		}
		if len(category.Inferences) == 0 {
			t.Fatal("field_types inferences are empty; determinism assertion would be trivial")
		}
		for _, inference := range category.Inferences {
			fields = append(fields, inference.Field)
		}
	}
	if wantFields := []string{"owner", "priority", "status", "title"}; !slices.Equal(fields, wantFields) {
		t.Fatalf("field_types order = %v, want %v", fields, wantFields)
	}

	for run := 2; run <= 32; run++ {
		got := executeAnalyze(t, root, "-o", "json")
		if !bytes.Equal(got, want) {
			t.Fatalf("run %d changed analyze JSON\nfirst: %s\nrun %d: %s", run, want, run, got)
		}
	}
}
