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

func TestComputeNextSequence_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	field := SchemaField{Prefix: "T", Digits: 3}
	got := computeNextSequence(dir, field)
	if got != "T001" {
		t.Errorf("empty dir: got %q, want %q", got, "T001")
	}
}

func TestComputeNextSequence_ExistingFiles(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "T001-a.md"))
	writeTestFile(t, filepath.Join(dir, "T002-b.md"))

	field := SchemaField{Prefix: "T", Digits: 3}
	got := computeNextSequence(dir, field)
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
	got := computeNextSequence(dir, field)
	if got != "T002" {
		t.Errorf("ignoring non-matching: got %q, want %q", got, "T002")
	}
}

func TestComputeNextSequence_DifferentPrefix(t *testing.T) {
	dir := t.TempDir()
	mkTestDir(t, filepath.Join(dir, "E01-infra"))
	mkTestDir(t, filepath.Join(dir, "E02-apps"))

	field := SchemaField{Prefix: "E", Digits: 2}
	got := computeNextSequence(dir, field)
	if got != "E03" {
		t.Errorf("prefix E digits 2: got %q, want %q", got, "E03")
	}
}

func TestComputeNextSequence_NonExistentDir(t *testing.T) {
	field := SchemaField{Prefix: "T", Digits: 3}
	got := computeNextSequence("/nonexistent/path", field)
	if got != "T001" {
		t.Errorf("nonexistent dir: got %q, want %q", got, "T001")
	}
}

func TestComputeNextSequence_EmptyPrefix(t *testing.T) {
	dir := t.TempDir()
	field := SchemaField{Prefix: "", Digits: 3}
	got := computeNextSequence(dir, field)
	if got != "" {
		t.Errorf("empty prefix: got %q, want empty", got)
	}
}

func TestComputeNextSequence_ZeroDigits(t *testing.T) {
	dir := t.TempDir()
	field := SchemaField{Prefix: "T", Digits: 0}
	got := computeNextSequence(dir, field)
	if got != "" {
		t.Errorf("zero digits: got %q, want empty", got)
	}
}
