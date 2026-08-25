package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// decodeEnvelope parses stdout as the validate envelope and fails the test if
// stdout is not a single JSON object.
func decodeEnvelope(t *testing.T, stdout string) map[string]any {
	t.Helper()
	var obj map[string]any
	if err := json.Unmarshal([]byte(stdout), &obj); err != nil {
		t.Fatalf("stdout is not a JSON envelope: %v\noutput: %q", err, stdout)
	}
	return obj
}

// firstResult returns results[0] of the envelope on stdout. Every validate
// invocation emits the envelope, so a single-file check reads its verdict here
// rather than from the top level.
func firstResult(t *testing.T, stdout string) map[string]any {
	t.Helper()
	env := decodeEnvelope(t, stdout)
	results, ok := env["results"].([]any)
	if !ok || len(results) == 0 {
		t.Fatalf("envelope has no results: %s", stdout)
	}
	return results[0].(map[string]any)
}

func envelopePaths(t *testing.T, env map[string]any) []string {
	t.Helper()
	results, ok := env["results"].([]any)
	if !ok {
		t.Fatalf("results missing or not an array: %v", env["results"])
	}
	paths := make([]string, 0, len(results))
	for _, r := range results {
		paths = append(paths, r.(map[string]any)["path"].(string))
	}
	return paths
}

func stemHealthChecks(t *testing.T, env map[string]any) []map[string]any {
	t.Helper()
	raw, ok := env["stem_health"].([]any)
	if !ok {
		t.Fatalf("stem_health missing or not an array: %v", env["stem_health"])
	}
	out := make([]map[string]any, 0, len(raw))
	for _, d := range raw {
		out = append(out, d.(map[string]any))
	}
	return out
}

// setupHealthProject builds the issue #68 fixture: three records plus a child
// .stem whose scope matches nothing and whose rule names a missing field, so
// stem health has something to report.
func setupHealthProject(t *testing.T) string {
	t.Helper()
	root := setupValidateProject(t, map[string]string{
		".stem":     "version: 2\nroot: true\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: enum\n    values: [Pending, Done]\n    required: true\n",
		"sub/.stem": "version: 2\nscope:\n  match: \"*.txt\"\nvalidate:\n  - rule: non_empty\n    field: nosuchfield\n",
		"a.md":      "---\nestado: Pending\n---\n# a\n",
		"b.md":      "---\nestado: Pending\n---\n# b\n",
		"c.md":      "---\nestado: Pending\n---\n# c\n",
	})
	mustChdir(t, root)
	return root
}

func TestValidateAll_StemHealthSeparatedFromRecords(t *testing.T) {
	setupHealthProject(t)

	stdout, err := executeValidate(t, "--all")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, stdout)
	}
	env := decodeEnvelope(t, stdout)

	paths := envelopePaths(t, env)
	for _, p := range paths {
		if strings.HasSuffix(p, ".stem") {
			t.Errorf(".stem entry %q leaked into results: %v", p, paths)
		}
	}

	summary := env["summary"].(map[string]any)
	if summary["total"].(float64) != 3 {
		t.Errorf("summary.total = %v, want 3 (record count); paths: %v", summary["total"], paths)
	}
	if summary["valid"].(float64) != 3 {
		t.Errorf("summary.valid = %v, want 3", summary["valid"])
	}

	health := stemHealthChecks(t, env)
	if len(health) == 0 {
		t.Fatal("stem_health is empty; the fixture has an orphan scope and a dangling rule field")
	}
	names := map[string]bool{}
	for _, h := range health {
		names[h["check"].(string)] = true
		if _, ok := h["severity"].(string); !ok {
			t.Errorf("stem_health entry missing severity: %v", h)
		}
	}
	if !names["scope-match"] && !names["rule-field-exists"] {
		t.Errorf("expected scope-match or rule-field-exists in stem_health, got %v", names)
	}
	if summary["stem_health_warnings_count"].(float64) == 0 {
		t.Error("summary.stem_health_warnings_count = 0, want > 0")
	}
}

func TestValidateAll_DirectoryResultsSeparatedFromRecords(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":    "version: 2\nroot: true\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: string\nstructural:\n  subdirs:\n    max_children: 5\n",
		"a.md":     "---\nestado: a\n---\n# a\n",
		"b.md":     "---\nestado: b\n---\n# b\n",
		"sub/c.md": "---\nestado: c\n---\n# c\n",
	})
	mustChdir(t, root)

	stdout, err := executeValidate(t, "--all")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, stdout)
	}
	env := decodeEnvelope(t, stdout)

	// A directory is not a record either. Counting one made summary.total
	// disagree with `query --count` on the same path.
	paths := envelopePaths(t, env)
	for _, p := range paths {
		if strings.HasSuffix(p, "/") {
			t.Errorf("directory entry %q leaked into results: %v", p, paths)
		}
	}
	if got := env["summary"].(map[string]any)["total"].(float64); got != 3 {
		t.Errorf("summary.total = %v, want 3 (record count); paths: %v", got, paths)
	}

	structural, ok := env["structural"].([]any)
	if !ok {
		t.Fatalf("structural missing or not an array: %v", env["structural"])
	}
	if len(structural) == 0 {
		t.Error("expected directory results under structural")
	}
	for _, s := range structural {
		if p := s.(map[string]any)["path"].(string); !strings.HasSuffix(p, "/") {
			t.Errorf("structural entry %q is not a directory", p)
		}
	}
}

func TestValidateAll_StructuralViolationStillFails(t *testing.T) {
	// max_children: 1 with two subdirectories violates the structural rule.
	root := setupValidateProject(t, map[string]string{
		".stem":    "version: 2\nroot: true\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: string\nstructural:\n  subdirs:\n    max_children: 1\n",
		"a.md":     "---\nestado: a\n---\n# a\n",
		"one/b.md": "---\nestado: b\n---\n# b\n",
		"two/c.md": "---\nestado: c\n---\n# c\n",
	})
	mustChdir(t, root)

	stdout, err := executeValidate(t, "--all")
	if err != ErrValidationFailed {
		t.Fatalf("err = %v, want ErrValidationFailed — moving directories out of results must not drop them from the exit code\noutput: %s", err, stdout)
	}
	env := decodeEnvelope(t, stdout)
	if got := env["summary"].(map[string]any)["structural_errors_count"].(float64); got == 0 {
		t.Errorf("structural_errors_count = 0, want > 0; envelope: %s", stdout)
	}
}

func TestValidateAll_WhereFilterDoesNotLeakStemEntries(t *testing.T) {
	setupHealthProject(t)

	stdout, err := executeValidate(t, "--all", "--where", "estado == 'Done'")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, stdout)
	}
	env := decodeEnvelope(t, stdout)

	paths := envelopePaths(t, env)
	if len(paths) != 0 {
		t.Errorf("results = %v, want empty (no record matches estado == 'Done')", paths)
	}
	if got := env["summary"].(map[string]any)["total"].(float64); got != 0 {
		t.Errorf("summary.total = %v, want 0", got)
	}
}

func TestValidateAll_EmptyCorpusReportsZeroRecords(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem": "version: 2\nroot: true\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: string\n",
	})
	empty := filepath.Join(root, "empty")
	if err := os.MkdirAll(empty, 0o755); err != nil {
		t.Fatal(err)
	}
	mustChdir(t, root)

	stdout, _ := executeValidate(t, "--all", "empty")
	env := decodeEnvelope(t, stdout)

	summary := env["summary"].(map[string]any)
	if summary["total"].(float64) != 0 {
		t.Errorf("summary.total = %v, want 0 for an empty corpus", summary["total"])
	}
	if summary["valid"].(float64) != 0 {
		t.Errorf("summary.valid = %v, want 0 for an empty corpus", summary["valid"])
	}

	notices, ok := env["notices"].([]any)
	if !ok {
		t.Fatalf("notices missing or not an array: %v", env["notices"])
	}
	found := false
	for _, n := range notices {
		if n.(map[string]any)["code"] == "no_records" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a no_records notice for an empty corpus, got %v", notices)
	}
}

func TestValidateAll_MissingStemStillEmitsEnvelope(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "x.md"), []byte("---\nestado: Pending\n---\n# x\n"), 0o644)
	mustChdir(t, root)

	stdout, err := executeValidate(t, "--all")
	if err != ErrValidationFailed {
		t.Fatalf("err = %v, want ErrValidationFailed", err)
	}
	env := decodeEnvelope(t, stdout)

	if env["kind"] != "rootline/validate-batch" {
		t.Errorf("kind = %v", env["kind"])
	}
	found := false
	for _, h := range stemHealthChecks(t, env) {
		if h["check"] == "stem-files-exist" {
			found = true
		}
	}
	if !found {
		t.Errorf("stem-files-exist missing from stem_health: %v", env["stem_health"])
	}
	if !hasNotice(t, env, "scan_failed") {
		t.Errorf("expected a scan_failed notice, got %v", env["notices"])
	}
}

func TestValidateAll_CorruptStemStillEmitsEnvelope(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, ".stem"), []byte("version: 2\nroot: true\nschema:\n  x: [broken\n"), 0o644)
	mustWriteFile(t, filepath.Join(root, "x.md"), []byte("---\nestado: Pending\n---\n# x\n"), 0o644)
	mustChdir(t, root)

	stdout, err := executeValidate(t, "--all")
	if err != ErrValidationFailed {
		t.Fatalf("err = %v, want ErrValidationFailed", err)
	}
	env := decodeEnvelope(t, stdout)

	var yamlValid map[string]any
	for _, h := range stemHealthChecks(t, env) {
		if h["check"] == "yaml-valid" {
			yamlValid = h
		}
	}
	if yamlValid == nil {
		t.Fatalf("yaml-valid missing from stem_health: %v", env["stem_health"])
	}
	if yamlValid["severity"] != "error" {
		t.Errorf("yaml-valid severity = %v, want error", yamlValid["severity"])
	}
	if got := env["summary"].(map[string]any)["stem_health_errors_count"].(float64); got != 1 {
		t.Errorf("stem_health_errors_count = %v, want 1", got)
	}
}

func TestValidateAll_UnknownLinkRuleKeysWarningStrictness(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem": "version: 2\nroot: true\nscope:\n  match: \"*.md\"\nlinks:\n  blocks:\n    target: \"../tasks/*.md\"\n    targte: \"../tasks/*.md\"\n",
		"a.md":  "---\ntitle: A\n---\n# A\n",
	})
	mustChdir(t, root)

	normalOut, normalErr := executeValidate(t, "--all")
	if normalErr != nil {
		t.Fatalf("normal validate must not fail on warning-only unknown-link-rule-keys: %v\noutput: %s", normalErr, normalOut)
	}
	normalEnv := decodeEnvelope(t, normalOut)
	if got := normalEnv["summary"].(map[string]any)["invalid"].(float64); got != 0 {
		t.Fatalf("normal invalid count = %v, want 0; output: %s", got, normalOut)
	}

	foundNormal := false
	for _, h := range stemHealthChecks(t, normalEnv) {
		if h["check"] != "unknown-link-rule-keys" {
			continue
		}
		foundNormal = true
		if h["field"] != "targte" {
			t.Errorf("normal unknown-link-rule-keys field = %v, want targte", h["field"])
		}
		if h["severity"] != "warn" {
			t.Errorf("normal unknown-link-rule-keys severity = %v, want warn", h["severity"])
		}
		msg := h["message"].(string)
		if !strings.Contains(msg, "links.blocks") {
			t.Errorf("normal unknown-link-rule-keys message = %q, want links.blocks context", msg)
		}
		if !strings.Contains(msg, `did you mean "target"?`) {
			t.Errorf("normal unknown-link-rule-keys message = %q, want fuzzy suggestion", msg)
		}
	}
	if !foundNormal {
		t.Fatalf("normal validate missing unknown-link-rule-keys warning: %v", normalEnv["stem_health"])
	}

	strictOut, strictErr := executeValidate(t, "--all", "--strict")
	if strictErr != ErrValidationFailed {
		t.Fatalf("--strict err = %v, want ErrValidationFailed\noutput: %s", strictErr, strictOut)
	}
	strictEnv := decodeEnvelope(t, strictOut)
	if got := strictEnv["summary"].(map[string]any)["invalid"].(float64); got != 0 {
		t.Fatalf("strict invalid count = %v, want 0; output: %s", got, strictOut)
	}

	foundStrict := false
	for _, h := range stemHealthChecks(t, strictEnv) {
		if h["check"] == "unknown-link-rule-keys" {
			foundStrict = true
		}
	}
	if !foundStrict {
		t.Fatalf("strict validate missing unknown-link-rule-keys warning: %v", strictEnv["stem_health"])
	}
}

func hasNotice(t *testing.T, env map[string]any, code string) bool {
	t.Helper()
	notices, ok := env["notices"].([]any)
	if !ok {
		t.Fatalf("notices missing or not an array: %v", env["notices"])
	}
	for _, n := range notices {
		if n.(map[string]any)["code"] == code {
			return true
		}
	}
	return false
}

func TestValidateSingleFile_UsesTheSameEnvelope(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":  "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n    required: true\n",
		"doc.md": "---\ntitle: Hello\n---\n# Hello",
	})

	stdout, err := executeValidate(t, filepath.Join(root, "doc.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	env := decodeEnvelope(t, stdout)

	if env["kind"] != "rootline/validate-batch" {
		t.Errorf("kind = %v, want rootline/validate-batch for a single file", env["kind"])
	}
	if env["version"].(float64) != 2 {
		t.Errorf("version = %v, want 2", env["version"])
	}
	for _, key := range []string{"results", "stem_health", "drift_warnings", "notices", "summary"} {
		if _, ok := env[key]; !ok {
			t.Errorf("key %q missing from single-file envelope", key)
		}
	}
	if got := env["summary"].(map[string]any)["total"].(float64); got != 1 {
		t.Errorf("summary.total = %v, want 1", got)
	}
}

func TestValidateStrict_NestedRootMarkerDoesNotFail(t *testing.T) {
	stem := "version: 2\nroot: true\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: enum\n    values: [Pending, Done]\n"
	root := setupValidateProject(t, map[string]string{
		".stem":       stem,
		"child/.stem": stem,
		"a.md":        "---\nestado: Pending\n---\n# a\n",
		"child/b.md":  "---\nestado: Done\n---\n# b\n",
	})
	mustChdir(t, root)

	stdout, err := executeValidate(t, "--all", "--strict")
	if err != nil {
		t.Fatalf("--strict failed on an info-level diagnostic: %v\noutput: %s", err, stdout)
	}
	env := decodeEnvelope(t, stdout)

	found := false
	for _, h := range stemHealthChecks(t, env) {
		if h["check"] == "nested-root-marker" {
			found = true
			if h["severity"] != "info" {
				t.Errorf("nested-root-marker severity = %v, want info", h["severity"])
			}
		}
	}
	if !found {
		t.Errorf("nested-root-marker missing from stem_health: %v", env["stem_health"])
	}
	if got := env["summary"].(map[string]any)["stem_health_info_count"].(float64); got == 0 {
		t.Error("summary.stem_health_info_count = 0, want > 0")
	}
}
