package rules

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mkTestDir(t *testing.T, path string) {
	t.Helper()
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustComputeNextSequence(t *testing.T, dir string, field SchemaField) string {
	t.Helper()
	got, err := computeNextSequence(dir, "id", field)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func mustComputeAllNextSequences(t *testing.T, dir string, field SchemaField) map[string]string {
	t.Helper()
	got, err := computeAllNextSequences(dir, "id", field)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestComputeNextSequence_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	field := SchemaField{Prefix: "T", Digits: 3}
	got := mustComputeNextSequence(t, dir, field)
	if got != "T001" {
		t.Errorf("empty dir: got %q, want %q", got, "T001")
	}
}

func TestComputeNextSequence_ExistingFiles(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "T001-a.md"))
	writeTestFile(t, filepath.Join(dir, "T002-b.md"))

	field := SchemaField{Prefix: "T", Digits: 3}
	got := mustComputeNextSequence(t, dir, field)
	if got != "T003" {
		t.Errorf("with T001,T002: got %q, want %q", got, "T003")
	}
}

func TestComputeNextSequence_IgnoresNonMatching(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "README.md"))
	writeTestFile(t, filepath.Join(dir, ".stem"))
	writeTestFile(t, filepath.Join(dir, "T001-task.md"))

	field := SchemaField{Prefix: "T", Digits: 3}
	got := mustComputeNextSequence(t, dir, field)
	if got != "T002" {
		t.Errorf("ignoring non-matching: got %q, want %q", got, "T002")
	}
}

func TestComputeNextSequence_DifferentPrefix(t *testing.T) {
	dir := t.TempDir()
	mkTestDir(t, filepath.Join(dir, "E01-infra"))
	mkTestDir(t, filepath.Join(dir, "E02-apps"))

	field := SchemaField{Prefix: "E", Digits: 2}
	got := mustComputeNextSequence(t, dir, field)
	if got != "E03" {
		t.Errorf("prefix E digits 2: got %q, want %q", got, "E03")
	}
}

func TestComputeNextSequence_NonExistentDir(t *testing.T) {
	field := SchemaField{Prefix: "T", Digits: 3}
	got := mustComputeNextSequence(t, "/nonexistent/path", field)
	if got != "T001" {
		t.Errorf("nonexistent dir: got %q, want %q", got, "T001")
	}
}

func TestComputeNextSequence_EmptyPrefix(t *testing.T) {
	dir := t.TempDir()
	field := SchemaField{Prefix: "", Digits: 3}
	got := mustComputeNextSequence(t, dir, field)
	if got != "" {
		t.Errorf("empty prefix: got %q, want empty", got)
	}
}

func TestComputeNextSequence_ZeroDigits(t *testing.T) {
	dir := t.TempDir()
	field := SchemaField{Prefix: "T", Digits: 0}
	got := mustComputeNextSequence(t, dir, field)
	if got != "" {
		t.Errorf("zero digits: got %q, want empty", got)
	}
}

func makeMultiPatternField(t *testing.T) (SchemaField, string) {
	t.Helper()
	dir := t.TempDir()
	mkTestDir(t, filepath.Join(dir, "O01-first-outcome"))
	mkTestDir(t, filepath.Join(dir, "O02-second-outcome"))
	writeTestFile(t, filepath.Join(dir, "T001-direct-task.md"))

	field := SchemaField{
		Type: "sequence",
		Match: &FieldMatch{
			Configs: map[string]any{
				"O*": map[string]any{"prefix": "O", "digits": 2},
				"T*": map[string]any{"prefix": "T", "digits": 3},
			},
		},
	}
	return field, dir
}

func TestComputeAllNextSequences_MultiPattern(t *testing.T) {
	field, dir := makeMultiPatternField(t)

	got := mustComputeAllNextSequences(t, dir, field)

	if got == nil {
		t.Fatal("computeAllNextSequences returned nil, want map")
	}
	if got["O*"] != "O03" {
		t.Errorf("O*: got %q, want O03", got["O*"])
	}
	if got["T*"] != "T002" {
		t.Errorf("T*: got %q, want T002", got["T*"])
	}
}

func TestComputeNextSequence_MultiPatternDeterministic(t *testing.T) {
	field, dir := makeMultiPatternField(t)

	// O* < T* alphabetically; both patterns have matching entries.
	// next must always return the O* value — deterministically.
	first := mustComputeNextSequence(t, dir, field)
	for i := 0; i < 10; i++ {
		got := mustComputeNextSequence(t, dir, field)
		if got != first {
			t.Errorf("non-deterministic: run %d got %q, first run got %q", i+1, got, first)
		}
	}
	if first != "O03" {
		t.Errorf("next = %q, want O03 (first alphabetical pattern that matches)", first)
	}
}

func TestComputeAllNextSequences_NilMatch(t *testing.T) {
	dir := t.TempDir()
	field := SchemaField{Type: "sequence"}
	got := mustComputeAllNextSequences(t, dir, field)
	if got != nil {
		t.Errorf("nil match: got %v, want nil", got)
	}
}
