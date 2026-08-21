package rules

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func evaluateBoth(t *testing.T, root string) (*StemHealthResult, *StemHealthResult) {
	t.Helper()
	direct, err := ValidateStemHealth(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	state, err := DiscoverStemState(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	pure, err := EvaluateStemState(context.Background(), state)
	if err != nil {
		t.Fatal(err)
	}
	return direct, pure
}

func TestEvaluateStemStateDiagnosticsParity(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, root string)
	}{
		{
			name: "valid stems",
			setup: func(t *testing.T, root string) {
				mustWriteStemTestFile(t, filepath.Join(root, ".stem"), []byte(`version: 2
root: true
scope:
  match: "*.md"
schema:
  title:
    type: string
    required: true
`))
				mustWriteStemTestFile(t, filepath.Join(root, "readme.md"), []byte("# Readme"))
			},
		},
		{
			name: "malformed YAML",
			setup: func(t *testing.T, root string) {
				mustWriteStemTestFile(t, filepath.Join(root, ".stem"), []byte("version: [broken\n"))
			},
		},
		{
			name: "orphan scope",
			setup: func(t *testing.T, root string) {
				mustWriteStemTestFile(t, filepath.Join(root, ".stem"), []byte(`version: 2
scope:
  match: "*.missing"
`))
			},
		},
		{
			name: "rule field references",
			setup: func(t *testing.T, root string) {
				mustWriteStemTestFile(t, filepath.Join(root, ".stem"), []byte(`version: 2
schema:
  title:
    type: string
validate:
  - rule: non_empty
    field: missing
    severity: warn
`))
			},
		},
		{
			name: "aggregates",
			setup: func(t *testing.T, root string) {
				mustWriteStemTestFile(t, filepath.Join(root, ".stem"), []byte(`version: 2
schema:
  status:
    type: enum
    values: [todo, done]
    required: true
aggregate:
  status: 'count_if(status == "todo")'
`))
			},
		},
		{
			name: "unknown link keys",
			setup: func(t *testing.T, root string) {
				mustWriteStemTestFile(t, filepath.Join(root, ".stem"), []byte(`version: 2
links:
  checks:
    resolv: true
`))
			},
		},
		{
			name: "nested roots",
			setup: func(t *testing.T, root string) {
				mustWriteStemTestFile(t, filepath.Join(root, ".stem"), []byte("version: 2\nroot: true\n"))
				mustWriteStemTestFile(t, filepath.Join(root, "docs", ".stem"), []byte("version: 2\nroot: true\n"))
			},
		},
		{
			name: "monotonic conflicts",
			setup: func(t *testing.T, root string) {
				mustWriteStemTestFile(t, filepath.Join(root, ".stem"), []byte(`version: 2
schema:
  status:
    type: enum
    values: [pending, done]
`))
				mustWriteStemTestFile(t, filepath.Join(root, "child", ".stem"), []byte(`version: 2
schema:
  status:
    type: enum
    values: [pending, blocked, done]
`))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			tt.setup(t, root)

			direct, pure := evaluateBoth(t, root)
			assertStemDiagnosticsDeepEqual(t, StemHealthDiagnostics(direct), StemHealthDiagnostics(pure))
		})
	}
}

func TestEvaluateStemStatePromotesMalformedExternalAncestorThroughOverlay(t *testing.T) {
	grand := t.TempDir()
	parent := filepath.Join(grand, "parent")
	root := filepath.Join(parent, "child")
	mustWriteStemTestFile(t, filepath.Join(grand, ".stem"), []byte("version: 2\nroot: true\n"))
	mustWriteStemTestFile(t, filepath.Join(parent, ".stem"), []byte("version: [broken\n"))
	mustWriteStemTestFile(t, filepath.Join(root, "record.md"), []byte("# Record\n"))

	state, err := DiscoverStemState(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	clone, err := state.Overlay(filepath.Join(root, ".stem"), []byte("version: 2\n"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := EvaluateStemState(context.Background(), clone)
	if err != nil {
		t.Fatal(err)
	}

	want := StemHealthDiagnostic{
		Path:     "../.stem",
		Check:    "yaml-valid",
		Severity: "error",
	}
	if !containsStemHealthDiagnostic(StemHealthDiagnostics(result), want) {
		t.Fatalf("diagnostics did not include %+v; got %+v", want, StemHealthDiagnostics(result))
	}
}

func TestEvaluateStemStateRejectsEnumToStringWidening(t *testing.T) {
	root := t.TempDir()
	mustWriteStemTestFile(t, filepath.Join(root, ".stem"), []byte(`version: 2
schema:
  estado:
    type: enum
    values: [Pending, Done]
`))
	mustWriteStemTestFile(t, filepath.Join(root, "child", ".stem"), []byte(`version: 2
schema:
  estado:
    type: string
`))

	state, err := DiscoverStemState(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	result, err := EvaluateStemState(context.Background(), state)
	if err != nil {
		t.Fatal(err)
	}

	var got []StemHealthDiagnostic
	for _, diag := range StemHealthDiagnostics(result) {
		if diag.Check == "type-consistency" && diag.Field == "estado" {
			got = append(got, diag)
		}
	}
	assertStemDiagnostics(t, got, []StemHealthDiagnostic{{
		Path:     "child/.stem",
		Check:    "type-consistency",
		Field:    "estado",
		Severity: "error",
		Message:  `type changes from "enum" to "string"`,
	}})
}

func TestEvaluateStemStateWarningOnlyHasNoErrorSeverity(t *testing.T) {
	root := t.TempDir()
	mustWriteStemTestFile(t, filepath.Join(root, ".stem"), []byte(`version: 2
scope:
  match: "*.missing"
`))

	state, err := DiscoverStemState(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	result, err := EvaluateStemState(context.Background(), state)
	if err != nil {
		t.Fatal(err)
	}

	for _, diag := range StemHealthDiagnostics(result) {
		if diag.Severity == SeverityError {
			t.Fatalf("unexpected error-severity diagnostic in warning-only state: %+v", diag)
		}
	}
}

func TestEvaluateStemStateDiagnosticsAreDeterministic(t *testing.T) {
	root := filepath.Clean(t.TempDir())
	rootStemPath := filepath.Join(root, ".stem")
	childStemPath := filepath.Join(root, "child", ".stem")
	recordPath := filepath.Join(root, "child", "note.md")

	rootStem, err := ParseStem(rootStemPath, []byte(`version: 2
scope:
  match: "*.absent"
schema:
  status:
    type: enum
    values: [pending, done]
`))
	if err != nil {
		t.Fatal(err)
	}
	childStem, err := ParseStem(childStemPath, []byte(`version: 2
schema:
  status:
    type: enum
    values: [pending, blocked, done]
links:
  checks:
    resolv: true
`))
	if err != nil {
		t.Fatal(err)
	}

	stateA := &StemState{
		Root: root,
		Stems: map[string]*StemFile{
			rootStemPath:  rootStem,
			childStemPath: childStem,
		},
		ParseErrors: map[string]error{
			filepath.Join(root, "broken", ".stem"): errors.New("broken yaml"),
		},
		Entries: map[string]StemStateEntry{
			root:                         {IsDir: true},
			filepath.Join(root, "child"): {IsDir: true},
			recordPath:                   {IsDir: false},
			rootStemPath:                 {IsDir: false},
			childStemPath:                {IsDir: false},
		},
	}
	stateB := &StemState{
		Root: root,
		Stems: map[string]*StemFile{
			childStemPath: childStem,
			rootStemPath:  rootStem,
		},
		ParseErrors: map[string]error{
			filepath.Join(root, "broken", ".stem"): errors.New("broken yaml"),
		},
		Entries: map[string]StemStateEntry{
			childStemPath:                {IsDir: false},
			rootStemPath:                 {IsDir: false},
			recordPath:                   {IsDir: false},
			filepath.Join(root, "child"): {IsDir: true},
			root:                         {IsDir: true},
		},
	}

	resultA, err := EvaluateStemState(context.Background(), stateA)
	if err != nil {
		t.Fatal(err)
	}
	resultB, err := EvaluateStemState(context.Background(), stateB)
	if err != nil {
		t.Fatal(err)
	}
	assertStemDiagnosticsDeepEqual(t, StemHealthDiagnostics(resultA), StemHealthDiagnostics(resultB))
}

func TestEvaluateStemStateHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := EvaluateStemState(ctx, &StemState{
		Root:        filepath.Clean(t.TempDir()),
		Stems:       map[string]*StemFile{},
		ParseErrors: map[string]error{},
		Entries:     map[string]StemStateEntry{},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("EvaluateStemState() error = %v, want context.Canceled", err)
	}
}

func assertStemDiagnosticsDeepEqual(t *testing.T, got, want []StemHealthDiagnostic) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("diagnostics mismatch\ngot:  %+v\nwant: %+v", got, want)
	}
}

func containsStemHealthDiagnostic(diags []StemHealthDiagnostic, want StemHealthDiagnostic) bool {
	for _, diag := range diags {
		if diag.Path == want.Path && diag.Check == want.Check && diag.Severity == want.Severity && diag.Field == want.Field {
			return true
		}
	}
	return false
}
