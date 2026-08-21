package main

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const task11FixResolutionStem = `version: 2
root: true
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Completed]
  id:
    type: sequence
    match:
      "BAD*": {prefix: BAD, digits: 2.0}
aggregate:
  estado: |
    all(descendants, {.estado == "Completed"}) ? "Completed" : "Pending"
`

func setupTask11FixResolutionProject(t *testing.T, invalidPaths ...string) string {
	t.Helper()
	files := map[string]string{
		".stem":          task11FixResolutionStem,
		"A.md":           "---\nestado: Pendng\n---\n# Independent correction\n",
		"S001/README.md": "---\nestado: Pending\n---\n# Stale aggregate\n",
		"S001/T001.md":   "---\nestado: Completed\n---\n# Completed child\n",
	}
	for _, path := range invalidPaths {
		files[path] = "---\nestado: Pendng\n---\n# Unresolved\n"
	}
	root := setupValidateProject(t, files)
	declareTestBoundary(t, root)
	return root
}

func TestTask11FixAllDryRunAllInvalidEmitsCompleteFailureClassificationWithoutWrites(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":     task11FixResolutionStem,
		"BAD001.md": "---\nestado: Pendng\n---\n# Unresolved one\n",
		"BAD002.md": "---\nestado: Pendng\n---\n# Unresolved two\n",
	})
	declareTestBoundary(t, root)
	before := task11FixFiles(t, root, "BAD001.md", "BAD002.md")
	mustChdir(t, root)

	stdout, err := runCmd(t, "fix", "--all", "--dry-run", "-o", "json")
	if err != ErrValidationFailed {
		t.Fatalf("err = %v, want ErrValidationFailed\nstdout=%s", err, stdout)
	}
	payload := task11DecodeFixPayload(t, stdout)
	task11AssertIncompleteFixEnvelope(t, payload, []string{"BAD001.md", "BAD002.md"}, map[string]string{
		"BAD001.md": `resolving .stem for BAD001.md: invalid match config for field "id" pattern "BAD*": digits must be a positive integer`,
		"BAD002.md": `resolving .stem for BAD002.md: invalid match config for field "id" pattern "BAD*": digits must be a positive integer`,
	}, true)
	if got := task11FixFiles(t, root, "BAD001.md", "BAD002.md"); !reflect.DeepEqual(got, before) {
		t.Fatalf("dry-run wrote files\nbefore=%q\nafter=%q", before, got)
	}
}

func TestTask11FixAllDryRunMixedEmitsFailurePreviewsWithoutWrites(t *testing.T) {
	root := setupTask11FixResolutionProject(t, "BAD001.md")
	before := task11FixFiles(t, root, "A.md", "S001/README.md", "S001/T001.md", "BAD001.md")
	mustChdir(t, root)

	stdout, err := runCmd(t, "fix", "--all", "--dry-run", "-o", "json")
	if err != ErrValidationFailed {
		t.Fatalf("err = %v, want ErrValidationFailed\nstdout=%s", err, stdout)
	}
	payload := task11DecodeFixPayload(t, stdout)
	task11AssertIncompleteFixEnvelope(t, payload, []string{"A.md", "S001/README.md", "S001/T001.md", "BAD001.md"}, map[string]string{
		"BAD001.md": `resolving .stem for BAD001.md: invalid match config for field "id" pattern "BAD*": digits must be a positive integer`,
	}, true)
	results := task11FixResultByPath(t, payload)
	if results["A.md"]["fixed"] != false || !strings.Contains(strings.Join(task11StringSlice(t, results["A.md"]["changes"]), " "), `correct estado: "Pendng" -> "Pending"`) {
		t.Fatalf("valid correction = %#v, want unfixed preview", results["A.md"])
	}
	if results["S001/README.md"]["fixed"] != false || len(task11StringSlice(t, results["S001/README.md"]["changes"])) != 0 {
		t.Fatalf("aggregate correction leaked into incomplete preview: %#v", results["S001/README.md"])
	}
	if got := task11FixFiles(t, root, "A.md", "S001/README.md", "S001/T001.md", "BAD001.md"); !reflect.DeepEqual(got, before) {
		t.Fatalf("dry-run wrote files\nbefore=%q\nafter=%q", before, got)
	}
}

func TestTask11FixAllMixedApplyRepairsIndependentEnumAndSuppressesAggregate(t *testing.T) {
	completeRoot := setupTask11FixResolutionProject(t)
	mustChdir(t, completeRoot)
	completeStdout, err := runCmd(t, "fix", "--all", "--dry-run", "-o", "json")
	if err != nil {
		t.Fatalf("complete dry-run failed: %v\nstdout=%s", err, completeStdout)
	}
	if !strings.Contains(completeStdout, `"type":"propagate_aggregate"`) {
		t.Fatalf("complete fixture did not activate propagation:\n%s", completeStdout)
	}

	root := setupTask11FixResolutionProject(t, "BAD001.md")
	mustChdir(t, root)
	stdout, err := runCmd(t, "fix", "--all", "-o", "json")
	if err != ErrValidationFailed {
		t.Fatalf("err = %v, want ErrValidationFailed\nstdout=%s", err, stdout)
	}
	payload := task11DecodeFixPayload(t, stdout)
	task11AssertIncompleteFixEnvelope(t, payload, []string{"A.md", "S001/README.md", "S001/T001.md", "BAD001.md"}, map[string]string{
		"BAD001.md": `resolving .stem for BAD001.md: invalid match config for field "id" pattern "BAD*": digits must be a positive integer`,
	}, false)
	if got := string(mustReadFile(t, filepath.Join(root, "A.md"))); !strings.Contains(got, "estado: Pending") {
		t.Fatalf("independent enum correction was not written:\n%s", got)
	}
	if got := string(mustReadFile(t, filepath.Join(root, "S001/README.md"))); !strings.Contains(got, "estado: Pending") {
		t.Fatalf("aggregate correction was not suppressed:\n%s", got)
	}
}

func TestTask11FixAllDryRunFullyResolvedPreservesProposalProtocol(t *testing.T) {
	root := setupTask11FixResolutionProject(t)
	mustChdir(t, root)

	stdout, err := runCmd(t, "fix", "--all", "--dry-run", "-o", "json")
	if err != nil {
		t.Fatalf("fully resolved dry-run failed: %v\nstdout=%s", err, stdout)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("fully resolved dry-run is not JSON: %v\n%s", err, stdout)
	}
	if payload["version"] != float64(1) || payload["kind"] != "rootline/proposals" {
		t.Fatalf("fully resolved dry-run identity = %#v, want rootline/proposals v1", payload)
	}
	for _, forbidden := range []string{"complete", "errors", "results"} {
		if _, ok := payload[forbidden]; ok {
			t.Fatalf("fully resolved proposal payload contains failure field %q: %#v", forbidden, payload)
		}
	}
}

func task11DecodeFixPayload(t *testing.T, stdout string) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("fix output is not JSON: %v\n%s", err, stdout)
	}
	return payload
}

func task11AssertIncompleteFixEnvelope(t *testing.T, payload map[string]any, paths []string, wantErrors map[string]string, dryRun bool) {
	t.Helper()
	assertJSONKeys(t, payload, []string{"version", "kind", "complete", "results", "summary", "errors"})
	if payload["version"] != float64(1) || payload["kind"] != "rootline/fix-batch" || payload["complete"] != false {
		t.Fatalf("failure identity = %#v, want incomplete rootline/fix-batch v1", payload)
	}
	results := task11FixResultByPath(t, payload)
	if len(results) != len(paths) {
		t.Fatalf("result count = %d, want one result for each scanned path %v: %#v", len(results), paths, results)
	}
	for _, path := range paths {
		result, ok := results[path]
		if !ok {
			t.Fatalf("missing result for scanned path %q: %#v", path, results)
		}
		if dryRun && result["fixed"] != false {
			t.Fatalf("result for %q marked fixed during incomplete dry-run: %#v", path, result)
		}
	}
	gotErrors := task11StringSlice(t, payload["errors"])
	want := make([]string, 0, len(wantErrors))
	for _, path := range paths {
		if message, ok := wantErrors[path]; ok {
			want = append(want, message)
			result := results[path]
			if got := task11StringSlice(t, result["changes"]); !reflect.DeepEqual(got, []string{"skipped: schema resolution failed"}) {
				t.Fatalf("invalid result for %q = %#v, want schema-resolution classification", path, result)
			}
		}
	}
	if !reflect.DeepEqual(gotErrors, want) {
		t.Fatalf("resolution errors = %#v, want exact path-qualified causes %#v", gotErrors, want)
	}
}

func task11FixResultByPath(t *testing.T, payload map[string]any) map[string]map[string]any {
	t.Helper()
	results := make(map[string]map[string]any)
	for _, raw := range payload["results"].([]any) {
		result := raw.(map[string]any)
		path, _ := result["path"].(string)
		if _, exists := results[path]; exists {
			t.Fatalf("duplicate result for %q: %#v", path, payload["results"])
		}
		results[path] = result
	}
	return results
}

func task11StringSlice(t *testing.T, value any) []string {
	t.Helper()
	raw, ok := value.([]any)
	if !ok {
		t.Fatalf("value = %#v, want JSON string array", value)
	}
	values := make([]string, len(raw))
	for i, item := range raw {
		values[i], ok = item.(string)
		if !ok {
			t.Fatalf("item %d = %#v, want string", i, item)
		}
	}
	return values
}

func task11FixFiles(t *testing.T, root string, paths ...string) map[string]string {
	t.Helper()
	files := make(map[string]string, len(paths))
	for _, path := range paths {
		files[path] = string(mustReadFile(t, filepath.Join(root, path)))
	}
	return files
}
