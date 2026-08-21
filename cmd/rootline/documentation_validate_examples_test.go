package main

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestValidateDocumentationDiagnosticExamplesMatchCLI(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":        "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: string\n",
		"a.md":         "---\nestado: Pending\n---\n# A\n",
		"sub/.stem":    "version: 2\nscope:\n  match: \"*.txt\"\n",
		"sub/other.md": "---\nestado: Pending\n---\n# Other\n",
	})

	allOutput, err := executeValidate(t, "--all", root, "-o", "json")
	if err != nil {
		t.Fatalf("validate --all fixture: %v\n%s", err, allOutput)
	}
	var allEnvelope struct {
		StemHealth []map[string]any `json:"stem_health"`
	}
	if err := json.Unmarshal([]byte(allOutput), &allEnvelope); err != nil {
		t.Fatalf("decode validate --all output: %v\n%s", err, allOutput)
	}
	actualHealth := diagnosticByKey(t, allEnvelope.StemHealth, "check", "scope-match")

	skipOutput, err := executeValidate(t, filepath.Join(root, "sub", "other.md"), "-o", "json")
	if err != nil {
		t.Fatalf("validate excluded file fixture: %v\n%s", err, skipOutput)
	}
	var skipEnvelope struct {
		Results []struct {
			Warnings []map[string]any `json:"warnings"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(skipOutput), &skipEnvelope); err != nil {
		t.Fatalf("decode excluded-file output: %v\n%s", err, skipOutput)
	}
	if len(skipEnvelope.Results) != 1 || len(skipEnvelope.Results[0].Warnings) != 1 {
		t.Fatalf("excluded-file warnings = %#v, want one result with one warning", skipEnvelope.Results)
	}
	actualSkip := skipEnvelope.Results[0].Warnings[0]

	repo := documentationContractRepoRoot(t)
	delete(actualHealth, "path")
	for _, path := range []string{"docs/validate.md", ".claude/skills/rootline/ref-validate.md"} {
		document := string(documentationContractRead(t, repo, path))
		documentedHealth := documentedJSONDiagnostic(t, document, "check", "scope-match")
		delete(documentedHealth, "path")
		if !reflect.DeepEqual(documentedHealth, actualHealth) {
			t.Fatalf("%s scope-match diagnostic = %#v, CLI emits %#v", path, documentedHealth, actualHealth)
		}
	}

	document := string(documentationContractRead(t, repo, "docs/validate.md"))
	documentedSkip := documentedConsoleJSON(t, document, `$ rootline validate scope/other.md --field "results[].warnings"`)
	documentedWarnings, ok := documentedSkip.([]any)
	if !ok || len(documentedWarnings) != 1 {
		t.Fatalf("documented excluded-file projection = %#v, want one outer result", documentedSkip)
	}
	inner, ok := documentedWarnings[0].([]any)
	if !ok || len(inner) != 1 {
		t.Fatalf("documented excluded-file projection = %#v, want one warning", documentedSkip)
	}
	documentedWarning, ok := inner[0].(map[string]any)
	if !ok {
		t.Fatalf("documented excluded-file warning = %#v, want object", inner[0])
	}
	if !reflect.DeepEqual(documentedWarning, actualSkip) {
		t.Fatalf("docs/validate.md excluded-file warning = %#v, CLI emits %#v", documentedWarning, actualSkip)
	}
}

func diagnosticByKey(t *testing.T, diagnostics []map[string]any, key, value string) map[string]any {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic[key] == value {
			return diagnostic
		}
	}
	t.Fatalf("diagnostic %s=%q not found in %#v", key, value, diagnostics)
	return nil
}

func documentedJSONDiagnostic(t *testing.T, document, key, value string) map[string]any {
	t.Helper()
	for _, block := range fencedBlocks(document) {
		if block.lang != "json" {
			continue
		}
		var documentValue any
		if err := json.Unmarshal([]byte(block.body), &documentValue); err != nil {
			continue
		}
		if diagnostic := nestedDiagnosticByKey(documentValue, key, value); diagnostic != nil {
			return diagnostic
		}
	}
	t.Fatalf("documented JSON diagnostic %s=%q not found", key, value)
	return nil
}

func nestedDiagnosticByKey(documentValue any, key, value string) map[string]any {
	switch typed := documentValue.(type) {
	case map[string]any:
		if typed[key] == value {
			return typed
		}
		for _, child := range typed {
			if found := nestedDiagnosticByKey(child, key, value); found != nil {
				return found
			}
		}
	case []any:
		for _, child := range typed {
			if found := nestedDiagnosticByKey(child, key, value); found != nil {
				return found
			}
		}
	}
	return nil
}

func documentedConsoleJSON(t *testing.T, document, command string) any {
	t.Helper()
	for _, block := range fencedBlocks(document) {
		if block.lang != "console" || !strings.Contains(block.body, command) {
			continue
		}
		_, output, ok := strings.Cut(block.body, "\n")
		if !ok {
			t.Fatalf("documented console command %q has no output", command)
		}
		var value any
		if err := json.Unmarshal([]byte(output), &value); err != nil {
			t.Fatalf("decode documented console output for %q: %v\n%s", command, err, output)
		}
		return value
	}
	t.Fatalf("documented console command %q not found", command)
	return nil
}
