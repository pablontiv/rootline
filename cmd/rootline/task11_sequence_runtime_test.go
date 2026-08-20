package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const task11InvalidIntegralFloatSequenceStem = `version: 2
root: true
scope:
  match: "*.md"
schema:
  id:
    type: sequence
    required: false
    match:
      "T*": {prefix: T, digits: 2.0}
`

func setupTask11InvalidSequenceProject(t *testing.T) string {
	t.Helper()
	root := setupValidateProject(t, map[string]string{
		".stem":        task11InvalidIntegralFloatSequenceStem,
		"T001-task.md": "---\n---\n# Task\n",
	})
	declareTestBoundary(t, root)
	return root
}

func TestTask11ValidateSingleRejectsIntegralFloatSequenceConfig(t *testing.T) {
	root := setupTask11InvalidSequenceProject(t)
	mustChdir(t, root)

	stdout, err := executeValidate(t, "T001-task.md", "-o", "json")
	if err != ErrValidationFailed {
		t.Fatalf("err = %v, want ErrValidationFailed\nstdout=%s", err, stdout)
	}
	env := decodeEnvelope(t, stdout)
	assertSkippedResultPaths(t, env, []string{"T001-task.md"})
	assertNoticeCodes(t, env, []string{"schema_resolution_failed"})
	assertTask11NoticeMentionsSequenceDigits(t, env)
}

func TestTask11ValidateAllRejectsIntegralFloatSequenceConfig(t *testing.T) {
	root := setupTask11InvalidSequenceProject(t)
	mustChdir(t, root)

	stdout, err := executeValidate(t, "--all", "-o", "json")
	if err != ErrValidationFailed {
		t.Fatalf("err = %v, want ErrValidationFailed\nstdout=%s", err, stdout)
	}
	env := decodeEnvelope(t, stdout)
	assertSkippedResultPaths(t, env, []string{"T001-task.md"})
	assertNoticeCodes(t, env, []string{"schema_resolution_failed"})
	assertTask11NoticeMentionsSequenceDigits(t, env)
}

func TestTask11ValidateStagedRejectsIntegralFloatSequenceConfig(t *testing.T) {
	root := makeStagedRepo(t, map[string]string{
		".stem":        task11InvalidIntegralFloatSequenceStem,
		"T001-task.md": "---\n---\n# Task\n",
	})
	mustChdir(t, root)

	stdout, err := executeValidate(t, "--staged", "-o", "json")
	if err != ErrValidationFailed {
		t.Fatalf("err = %v, want ErrValidationFailed\nstdout=%s", err, stdout)
	}
	env := decodeEnvelope(t, stdout)
	assertSkippedResultPaths(t, env, []string{"T001-task.md"})
	assertNoticeCodes(t, env, []string{"schema_resolution_failed"})
	assertTask11NoticeMentionsSequenceDigits(t, env)
}

func TestTask11DescribeRejectsIntegralFloatSequenceHints(t *testing.T) {
	root := setupTask11InvalidSequenceProject(t)
	mustChdir(t, root)

	stdout, _, err := runTask11Command(t, "describe", root, "-o", "json")
	if err == nil {
		t.Fatalf("describe accepted invalid sequence config and wrote stdout: %s", stdout)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("describe wrote partial stdout before rejecting invalid config: %q", stdout)
	}
	assertTask11ErrorMentionsSequenceDigits(t, err)
}

func TestTask11NewRejectsIntegralFloatSequenceBeforeWriteOrDryRun(t *testing.T) {
	root := setupTask11InvalidSequenceProject(t)
	target := filepath.Join(root, "T002-new.md")

	stdout, _, err := runTask11Command(t, "new", target, "--dry-run")
	if err == nil {
		t.Fatalf("new --dry-run accepted invalid sequence config and wrote stdout: %s", stdout)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("new --dry-run wrote partial stdout before rejecting invalid config: %q", stdout)
	}
	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Fatalf("new created target despite invalid sequence config: stat err=%v", statErr)
	}
	assertTask11ErrorMentionsSequenceDigits(t, err)
}

func TestTask11FixAllRejectsIntegralFloatSequenceWithoutWriting(t *testing.T) {
	root := setupTask11InvalidSequenceProject(t)
	stemPath := filepath.Join(root, ".stem")
	before := mustReadFile(t, stemPath)
	mustChdir(t, root)

	stdout, err := runCmd(t, "fix", "--all", "-o", "json")
	if err != ErrValidationFailed {
		t.Fatalf("err = %v, want ErrValidationFailed\nstdout=%s", err, stdout)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("fix output is not JSON: %v\n%s", err, stdout)
	}
	if payload["complete"] != false {
		t.Fatalf("fix payload complete = %v, want false: %#v", payload["complete"], payload)
	}
	after := mustReadFile(t, stemPath)
	if string(after) != string(before) {
		t.Fatalf("fix rewrote .stem despite invalid sequence config\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestTask11QueryAndGraphRejectIntegralFloatSequenceConfig(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
	}{
		{name: "query", args: []string{"query", "--from"}},
		{name: "graph", args: []string{"graph"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			root := setupTask11InvalidSequenceProject(t)
			cmdArgs := append([]string{}, tt.args...)
			if tt.name == "query" {
				cmdArgs = append(cmdArgs, root)
			} else {
				cmdArgs = append(cmdArgs, root, "-o", "json")
			}
			stdout, _, err := runTask11Command(t, cmdArgs...)
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

func runTask11Command(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	resetFlags()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	return stdout.String(), stderr.String(), err
}

func assertTask11NoticeMentionsSequenceDigits(t *testing.T, env map[string]any) {
	t.Helper()
	notices := env["notices"].([]any)
	if len(notices) == 0 {
		t.Fatal("missing notices")
	}
	message, _ := notices[0].(map[string]any)["message"].(string)
	for _, want := range []string{"id", "T*", "digits"} {
		if !strings.Contains(message, want) {
			t.Fatalf("notice message %q does not contain %q", message, want)
		}
	}
}

func assertTask11ErrorMentionsSequenceDigits(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("missing error")
	}
	for _, want := range []string{"id", "T*", "digits"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not contain %q", err.Error(), want)
		}
	}
}

func TestTask11FixAllMixedInvalidGovernanceRepairsValidAndClassifiesInvalid(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem": `version: 2
root: true
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    values: [Pending, Done]
  id:
    type: sequence
    match:
      "BAD*": {prefix: BAD, digits: 2.0}
`,
		"A.md":      "---\nestado: Pendng\n---\n# A\n",
		"BAD001.md": "---\nestado: Pendng\n---\n# Bad\n",
	})
	declareTestBoundary(t, root)
	mustChdir(t, root)

	stdout, err := runCmd(t, "fix", "--all", "-o", "json")
	if err != ErrValidationFailed {
		t.Fatalf("err = %v, want ErrValidationFailed\nstdout=%s", err, stdout)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("fix output is not JSON: %v\n%s", err, stdout)
	}
	if payload["kind"] != "rootline/fix-batch" || payload["version"] != float64(1) {
		t.Fatalf("unexpected fix envelope: %#v", payload)
	}
	if payload["complete"] != false {
		t.Fatalf("complete = %v, want false", payload["complete"])
	}
	errs, _ := payload["errors"].([]any)
	if len(errs) != 1 || !strings.Contains(errs[0].(string), "BAD001.md") || !strings.Contains(errs[0].(string), "digits") {
		t.Fatalf("errors = %#v, want BAD001 digits classification", payload["errors"])
	}
	valid := string(mustReadFile(t, filepath.Join(root, "A.md")))
	if !strings.Contains(valid, "estado: Pending") {
		t.Fatalf("valid record was not repaired:\n%s", valid)
	}
	invalid := string(mustReadFile(t, filepath.Join(root, "BAD001.md")))
	if strings.Contains(invalid, "estado: Pending") {
		t.Fatalf("invalid record was repaired despite resolution failure:\n%s", invalid)
	}
}
