package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/rules"
)

func TestValidateProspectiveStemWritesRejectsJointlyInvalidHierarchy(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, ".stem"), []byte("version: 2\nroot: true\nschema:\n  estado:\n    type: enum\n    values: [Pending, Done]\n"), 0o644)

	childDir := filepath.Join(root, "child")
	if err := os.MkdirAll(childDir, 0o755); err != nil {
		t.Fatal(err)
	}
	childTarget := filepath.Join(childDir, ".stem")
	plan := schemaApplyBatchPlan{
		writes: []stemWritePlan{{
			reportTarget: "reported-child/.stem",
			target:       childTarget,
			content:      []byte("version: 2\nschema:\n  estado:\n    type: string\n"),
		}},
		actionsByWrite: [][]string{{"create_stem: " + childTarget}},
	}

	diags, err := validateProspectiveStemWrites(context.Background(), root, plan)
	if err != nil {
		t.Fatalf("validateProspectiveStemWrites returned error: %v", err)
	}

	for _, diag := range diags {
		if diag.Path == filepath.Join("child", ".stem") && diag.Check == "type-consistency" && diag.Field == "estado" && diag.Severity == rules.SeverityError {
			return
		}
	}
	t.Fatalf("missing child-owned error-severity type-consistency diagnostic: %+v", diags)
}

func TestValidateProspectiveStemWritesAllowsWarningsAndInfo(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, ".stem"), []byte("version: 2\nroot: true\nschema:\n  estado:\n    type: string\n"), 0o644)

	childDir := filepath.Join(root, "child")
	if err := os.MkdirAll(childDir, 0o755); err != nil {
		t.Fatal(err)
	}
	childTarget := filepath.Join(childDir, ".stem")
	plan := schemaApplyBatchPlan{
		writes: []stemWritePlan{{
			reportTarget: childTarget,
			target:       childTarget,
			content:      []byte("version: 2\nroot: true\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: string\n"),
		}},
		actionsByWrite: [][]string{{"create_stem: " + childTarget}},
	}

	diags, err := validateProspectiveStemWrites(context.Background(), root, plan)
	if err != nil {
		t.Fatalf("validateProspectiveStemWrites returned error: %v", err)
	}
	if len(diags) == 0 {
		t.Fatal("diagnostics = empty, want warning/info diagnostics returned to the envelope")
	}
	seen := map[string]bool{}
	for _, diag := range diags {
		if diag.Severity == rules.SeverityError {
			t.Fatalf("unexpected blocking diagnostic in warning/info case: %+v", diag)
		}
		seen[diag.Severity] = true
	}
	if !seen[rules.SeverityWarn] || !seen[rules.SeverityInfo] {
		t.Fatalf("diagnostic severities = %#v, want both warn and info in %+v", seen, diags)
	}
	if blocking := blockingStemHealth(diags); len(blocking) != 0 {
		t.Fatalf("blockingStemHealth = %+v, want warnings/info allowed", blocking)
	}
}

func TestValidateProspectiveStemWritesUsesFinalDuplicateTargetContent(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, ".stem"), []byte("version: 2\nroot: true\nschema:\n  estado:\n    type: enum\n    values: [Pending, Done]\n"), 0o644)

	childDir := filepath.Join(root, "child")
	if err := os.MkdirAll(childDir, 0o755); err != nil {
		t.Fatal(err)
	}
	childTarget := filepath.Join(childDir, ".stem")
	plan := schemaApplyBatchPlan{
		writes: []stemWritePlan{
			{reportTarget: "first", target: childTarget, content: []byte("version: 2\nschema:\n  estado:\n    type: string\n")},
			{reportTarget: "second", target: childTarget, content: []byte("version: 2\nschema:\n  estado:\n    type: enum\n    values: [Pending]\n")},
		},
		actionsByWrite: [][]string{{"first action"}, {"second action"}},
	}

	diags, err := validateProspectiveStemWrites(context.Background(), root, plan)
	if err != nil {
		t.Fatalf("validateProspectiveStemWrites returned error: %v", err)
	}
	for _, diag := range diags {
		if diag.Check == "type-consistency" && diag.Field == "estado" && diag.Severity == rules.SeverityError {
			t.Fatalf("duplicate target did not use final content; got blocking diagnostic %+v in %+v", diag, diags)
		}
	}
}

func TestValidateProspectiveStemWritesSortsDiagnostics(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, ".stem"), []byte("version: 2\nroot: true\nschema:\n  title:\n    type: string\n"), 0o644)
	for _, dir := range []string{"b", "a"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	plan := schemaApplyBatchPlan{
		writes: []stemWritePlan{
			{reportTarget: "b/.stem", target: filepath.Join(root, "b", ".stem"), content: []byte("version: 2\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n")},
			{reportTarget: "a/.stem", target: filepath.Join(root, "a", ".stem"), content: []byte("version: 2\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n")},
		},
		actionsByWrite: [][]string{{"b action"}, {"a action"}},
	}

	diags, err := validateProspectiveStemWrites(context.Background(), root, plan)
	if err != nil {
		t.Fatalf("validateProspectiveStemWrites returned error: %v", err)
	}
	var gotPaths []string
	for _, diag := range diags {
		if diag.Check == "scope-match" {
			gotPaths = append(gotPaths, diag.Path)
		}
	}
	wantPaths := []string{filepath.Join("a", ".stem"), filepath.Join("b", ".stem")}
	if !reflect.DeepEqual(gotPaths, wantPaths) {
		t.Fatalf("scope-match diagnostic paths = %#v, want sorted %#v (all diagnostics: %+v)", gotPaths, wantPaths, diags)
	}
}

func TestValidateProspectiveStemWritesParseErrorNamesReportTarget(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, ".stem"), []byte("version: 2\nroot: true\n"), 0o644)
	target := filepath.Join(root, "child", ".stem")
	plan := schemaApplyBatchPlan{
		writes: []stemWritePlan{{
			reportTarget: "from-report/.stem",
			target:       target,
			content:      []byte("version: 2\nschema: [not-a-map]\n"),
		}},
		actionsByWrite: [][]string{{"create_stem: " + target}},
	}

	_, err := validateProspectiveStemWrites(context.Background(), root, plan)
	if err == nil {
		t.Fatal("validateProspectiveStemWrites accepted malformed candidate")
	}
	if !strings.Contains(err.Error(), "from-report/.stem") {
		t.Fatalf("parse error %q does not name reportTarget", err)
	}
}

func TestExecuteStemWritesContinuesAfterAtomicWriteFailure(t *testing.T) {
	root := t.TempDir()
	targets := []string{filepath.Join(root, "a.stem"), filepath.Join(root, "b.stem"), filepath.Join(root, "c.stem")}
	var calls []string
	writeErr := errors.New("disk full")
	writer := func(target string, content []byte, mode fs.FileMode) error {
		calls = append(calls, fmt.Sprintf("%s:%s:%o", filepath.Base(target), string(content), mode.Perm()))
		if target == targets[1] {
			return writeErr
		}
		return nil
	}
	plan := schemaApplyBatchPlan{
		writes: []stemWritePlan{
			{reportTarget: "c", target: targets[2], content: []byte("C")},
			{reportTarget: "a", target: targets[0], content: []byte("A")},
			{reportTarget: "b", target: targets[1], content: []byte("B")},
		},
		actionsByWrite: [][]string{{"create c"}, {"create a"}, {"create b"}},
	}

	applied, errs := executeStemWrites(context.Background(), plan, false, writer)

	wantApplied := []string{"create a", "create c"}
	if !reflect.DeepEqual(applied, wantApplied) {
		t.Fatalf("applied = %#v, want %#v", applied, wantApplied)
	}
	if len(errs) != 1 || !strings.Contains(errs[0], "write b") || !strings.Contains(errs[0], writeErr.Error()) {
		t.Fatalf("errs = %#v, want one contextual middle-target error", errs)
	}
	wantCalls := []string{"a.stem:A:644", "b.stem:B:644", "c.stem:C:644"}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("writer calls = %#v, want sorted best-effort calls %#v", calls, wantCalls)
	}
}

func TestExecuteStemWritesStopsAfterContextCancellation(t *testing.T) {
	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	var calls []string
	writer := func(target string, content []byte, mode fs.FileMode) error {
		calls = append(calls, filepath.Base(target))
		cancel()
		return nil
	}
	plan := schemaApplyBatchPlan{
		writes: []stemWritePlan{
			{reportTarget: "a", target: filepath.Join(root, "a.stem"), content: []byte("A")},
			{reportTarget: "b", target: filepath.Join(root, "b.stem"), content: []byte("B")},
		},
		actionsByWrite: [][]string{{"create a"}, {"create b"}},
	}

	applied, errs := executeStemWrites(ctx, plan, false, writer)

	if !reflect.DeepEqual(applied, []string{"create a"}) {
		t.Fatalf("applied = %#v, want first action only", applied)
	}
	if !reflect.DeepEqual(calls, []string{"a.stem"}) {
		t.Fatalf("writer calls = %#v, want stop before second write", calls)
	}
	if len(errs) != 1 || !strings.Contains(errs[0], context.Canceled.Error()) {
		t.Fatalf("errs = %#v, want one cancellation error", errs)
	}
}

func TestExecuteStemWritesDryRunDoesNotCallWriter(t *testing.T) {
	root := t.TempDir()
	called := false
	writer := func(string, []byte, fs.FileMode) error {
		called = true
		return nil
	}
	plan := schemaApplyBatchPlan{
		writes: []stemWritePlan{
			{reportTarget: "b", target: filepath.Join(root, "b.stem"), content: []byte("B")},
			{reportTarget: "a", target: filepath.Join(root, "a.stem"), content: []byte("A")},
		},
		actionsByWrite: [][]string{{"create b"}, {"create a"}},
	}

	applied, errs := executeStemWrites(context.Background(), plan, true, writer)

	if called {
		t.Fatal("dry-run called the injected writer")
	}
	if len(errs) != 0 {
		t.Fatalf("errs = %#v, want none", errs)
	}
	wantApplied := []string{"create a", "create b"}
	if !reflect.DeepEqual(applied, wantApplied) {
		t.Fatalf("applied = %#v, want sorted dry-run actions %#v", applied, wantApplied)
	}
}

func TestExecuteStemWritesUsesStableTargetOrder(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a.stem")
	b := filepath.Join(root, "b.stem")
	plan := schemaApplyBatchPlan{
		writes: []stemWritePlan{
			{reportTarget: "b", target: b, content: []byte("B")},
			{reportTarget: "a-first", target: a, content: []byte("A1")},
			{reportTarget: "a-second", target: a, content: []byte("A2")},
		},
		actionsByWrite: [][]string{{"write b"}, {"write a first"}, {"write a second"}},
	}
	var calls []string
	writer := func(target string, content []byte, mode fs.FileMode) error {
		calls = append(calls, fmt.Sprintf("%s:%s", filepath.Base(target), string(content)))
		return nil
	}

	applied, errs := executeStemWrites(context.Background(), plan, false, writer)

	if len(errs) != 0 {
		t.Fatalf("errs = %#v, want none", errs)
	}
	wantCalls := []string{"a.stem:A1", "a.stem:A2", "b.stem:B"}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("calls = %#v, want stable target order %#v", calls, wantCalls)
	}
	wantApplied := []string{"write a first", "write a second", "write b"}
	if !reflect.DeepEqual(applied, wantApplied) {
		t.Fatalf("applied = %#v, want index-aligned actions %#v", applied, wantApplied)
	}
}

func TestSchemaApplyResultEnvelopeInitializesStemHealth(t *testing.T) {
	result := newSchemaApplyResult("/tmp/root", true)
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"stem_health":[]`) {
		t.Fatalf("schema apply envelope = %s, want non-omitempty empty stem_health array", data)
	}
}
