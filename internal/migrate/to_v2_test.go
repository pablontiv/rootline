package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpgradeToV2_V1ToV2(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	content := "version: 1\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n"
	if err := os.WriteFile(stemPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := UpgradeToV2(dir, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Total != 1 {
		t.Errorf("total = %d, want 1", result.Total)
	}
	if len(result.Updated) != 1 {
		t.Errorf("updated = %d, want 1", len(result.Updated))
	}
	if result.Skipped != 0 {
		t.Errorf("skipped = %d, want 0", result.Skipped)
	}

	data, err := os.ReadFile(stemPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "version: 2") {
		t.Errorf("file should contain 'version: 2', got:\n%s", data)
	}
	if strings.Contains(string(data), "version: 1") {
		t.Errorf("file should not contain 'version: 1', got:\n%s", data)
	}
}

func TestUpgradeToV2_V0ToV2(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	// No version field = version 0
	content := "scope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n"
	if err := os.WriteFile(stemPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := UpgradeToV2(dir, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Updated) != 1 {
		t.Errorf("updated = %d, want 1", len(result.Updated))
	}

	data, err := os.ReadFile(stemPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "version: 2") {
		t.Errorf("file should contain 'version: 2', got:\n%s", data)
	}
}

func TestUpgradeToV2_AlreadyV2(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	content := "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n"
	if err := os.WriteFile(stemPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := UpgradeToV2(dir, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Skipped != 1 {
		t.Errorf("skipped = %d, want 1", result.Skipped)
	}
	if len(result.Updated) != 0 {
		t.Errorf("updated = %d, want 0", len(result.Updated))
	}
}

func TestUpgradeToV2_CommentPreservation(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	content := "# Root stem\nversion: 1\nscope:\n  match: \"*.md\"\n# Schema section\nschema:\n  title:\n    type: string\n"
	if err := os.WriteFile(stemPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := UpgradeToV2(dir, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Updated) != 1 {
		t.Fatalf("updated = %d, want 1", len(result.Updated))
	}

	data, err := os.ReadFile(stemPath)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "# Root stem") {
		t.Errorf("head comment lost:\n%s", s)
	}
	if !strings.Contains(s, "# Schema section") {
		t.Errorf("inline comment lost:\n%s", s)
	}
	if !strings.Contains(s, "version: 2") {
		t.Errorf("version not upgraded:\n%s", s)
	}
}

func TestUpgradeToV2_DryRun(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	content := "version: 1\nscope:\n  match: \"*.md\"\n"
	if err := os.WriteFile(stemPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := UpgradeToV2(dir, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Updated) != 1 {
		t.Errorf("updated = %d, want 1", len(result.Updated))
	}

	// File should not be modified in dry-run mode.
	data, err := os.ReadFile(stemPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "version: 1") {
		t.Errorf("file was modified in dry-run mode")
	}
}

func TestUpgradeToV2_ResultFields(t *testing.T) {
	dir := t.TempDir()

	result, err := UpgradeToV2(dir, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Version != 1 {
		t.Errorf("result version = %d, want 1", result.Version)
	}
	if result.Kind != "rootline/migrate-to-v2" {
		t.Errorf("result kind = %q, want rootline/migrate-to-v2", result.Kind)
	}
}
