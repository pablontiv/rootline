package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/proposal"
)

func TestSetPostValidationRollsBackOnSchemaResolutionFailure(t *testing.T) {
	root := setupTask11InvalidSequenceProject(t)
	path := filepath.Join(root, "T001-task.md")
	original := []byte("---\nstatus: Pending\n---\n# Original\n")
	if err := os.WriteFile(path, []byte("---\nstatus: Done\n---\n# Written\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := postValidateOrRollback(context.Background(), path, root, original)
	if err == nil || !strings.Contains(err.Error(), "rolled back") || !strings.Contains(err.Error(), "digits") {
		t.Fatalf("post-validation error = %v, want resolution rollback cause", err)
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != string(original) {
		t.Fatalf("bytes after rollback = %q, want %q", got, original)
	}
}

func TestEnsureRecordsResolveHonorsCancellationBeforeResolution(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := ensureRecordsResolve(ctx, []*extract.Record{{Path: "record.md"}}, t.TempDir())
	if err != context.Canceled {
		t.Fatalf("ensureRecordsResolve error = %v, want context.Canceled", err)
	}
}

func TestTask11StrictResolutionReadCommandsRejectBeforeOutput(t *testing.T) {
	root := setupTask11InvalidSequenceProject(t)
	target := filepath.Join(root, "T001-task.md")
	for _, tt := range []struct {
		name string
		args []string
	}{
		{name: "analyze", args: []string{"analyze", root}},
		{name: "schema propose", args: []string{"schema", "propose", root}},
		{name: "stats", args: []string{"stats", root}},
		{name: "tree", args: []string{"tree", root}},
		{name: "migrate scaffold", args: []string{"migrate", "--scaffold", root}},
		{name: "explain", args: []string{"explain", target}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			stdout, _, err := runTask11Command(t, tt.args...)
			if err == nil {
				t.Fatalf("%s accepted invalid sequence config and wrote stdout: %s", tt.name, stdout)
			}
			if strings.TrimSpace(stdout) != "" {
				t.Fatalf("%s wrote partial stdout before rejecting invalid config: %q", tt.name, stdout)
			}
			assertTask11ErrorMentionsSequenceDigits(t, err)
		})
	}
}

func TestTask11SchemaAndRepairApplyPreserveFailureEnvelopes(t *testing.T) {
	root := setupTask11InvalidSequenceProject(t)

	t.Run("schema apply", func(t *testing.T) {
		reportPath := filepath.Join(root, "analyze.json")
		writeTask11Report(t, reportPath, map[string]any{
			"version": 1,
			"kind":    "rootline/analyze",
			"root":    root,
			"categories": []any{
				map[string]any{"id": "schema", "inferences": []any{
					map[string]any{"type": "required_field", "field": "id"},
				}},
			},
		})
		stdout, _, err := runTask11Command(t, "schema", "apply", "--report", reportPath, "-o", "json")
		if err == nil {
			t.Fatalf("schema apply accepted invalid governance: %s", stdout)
		}
		if strings.TrimSpace(stdout) == "" {
			t.Fatalf("schema apply returned %v without its required failure envelope", err)
		}
		assertTask11Envelope(t, stdout, "rootline/schema-apply")
		assertTask11EnvelopeMentionsSequenceDigits(t, stdout)
	})

	t.Run("repair apply", func(t *testing.T) {
		reportPath := filepath.Join(root, "repair.json")
		data, err := json.Marshal(proposal.Report{
			Version: 1,
			Kind:    "rootline/proposals",
			Root:    root,
			Proposals: []proposal.Proposal{{
				Type: proposal.CorrectValue, Field: "status", Paths: []string{"T001-task.md"}, From: "Pending", To: "Done",
			}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(reportPath, data, 0o644); err != nil {
			t.Fatal(err)
		}
		stdout, _, err := runTask11Command(t, "repair", "apply", "--report", reportPath, "-o", "json")
		if err == nil {
			t.Fatalf("repair apply accepted invalid governance: %s", stdout)
		}
		assertTask11Envelope(t, stdout, "rootline/repair")
		assertTask11EnvelopeMentionsSequenceDigits(t, stdout)
	})
}

func writeTask11Report(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertTask11EnvelopeMentionsSequenceDigits(t *testing.T, stdout string) {
	t.Helper()
	for _, want := range []string{"id", "T*", "digits"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("envelope %q does not contain %q", stdout, want)
		}
	}
}

func assertTask11Envelope(t *testing.T, stdout, kind string) {
	t.Helper()
	var envelope map[string]any
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("output is not a failure envelope: %v\n%s", err, stdout)
	}
	if envelope["kind"] != kind || envelope["version"] != float64(1) {
		t.Fatalf("envelope = %#v, want %s v1", envelope, kind)
	}
}
