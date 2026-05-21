package main

import (
	"bytes"
	"testing"

	"github.com/pablontiv/rootline/internal/infer"
	"github.com/spf13/cobra"
)

// TestRenderApplyTable_EmptyResult tests renderApplyTable with empty result
func TestRenderApplyTable_EmptyResult(t *testing.T) {
	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	result := &infer.ApplyResult{
		DryRun:  false,
		Applied: []string{},
		Skipped: []string{},
	}

	err := renderApplyTable(cmd, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("No modifications")) {
		t.Errorf("expected 'No modifications' message, got: %s", output)
	}
}

// TestRenderApplyTable_WithApplied tests renderApplyTable with applied items
func TestRenderApplyTable_WithApplied(t *testing.T) {
	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	result := &infer.ApplyResult{
		DryRun:  false,
		Applied: []string{"sub/.stem", "sub/README.md"},
		Skipped: []string{},
	}

	err := renderApplyTable(cmd, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Applied")) {
		t.Errorf("expected 'Applied' label, got: %s", output)
	}
	if !bytes.Contains([]byte(output), []byte("sub/.stem")) {
		t.Errorf("expected 'sub/.stem' in output, got: %s", output)
	}
}

// TestRenderApplyTable_DryRun tests renderApplyTable with dry-run flag
func TestRenderApplyTable_DryRun(t *testing.T) {
	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	result := &infer.ApplyResult{
		DryRun:  true,
		Applied: []string{"sub/.stem"},
		Skipped: []string{},
	}

	err := renderApplyTable(cmd, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Dry run")) {
		t.Errorf("expected 'Dry run' message, got: %s", output)
	}
	if !bytes.Contains([]byte(output), []byte("Would apply")) {
		t.Errorf("expected 'Would apply' label, got: %s", output)
	}
}

// TestRenderApplyTable_WithSkipped tests renderApplyTable with skipped items
func TestRenderApplyTable_WithSkipped(t *testing.T) {
	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	result := &infer.ApplyResult{
		DryRun:  false,
		Applied: []string{},
		Skipped: []string{"complex.stem"},
	}

	err := renderApplyTable(cmd, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Skipped")) {
		t.Errorf("expected 'Skipped' section, got: %s", output)
	}
	if !bytes.Contains([]byte(output), []byte("requires agent")) {
		t.Errorf("expected 'requires agent' text, got: %s", output)
	}
}

// TestRenderApplyTable_MixedResults tests renderApplyTable with both applied and skipped
func TestRenderApplyTable_MixedResults(t *testing.T) {
	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	result := &infer.ApplyResult{
		DryRun:  false,
		Applied: []string{"simple.stem"},
		Skipped: []string{"complex.stem"},
	}

	err := renderApplyTable(cmd, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Applied")) {
		t.Errorf("expected 'Applied' section")
	}
	if !bytes.Contains([]byte(output), []byte("Skipped")) {
		t.Errorf("expected 'Skipped' section")
	}
}

// TestRenderApplyTable_DryRunWithNoChanges tests dry-run with no applied or skipped
func TestRenderApplyTable_DryRunNoChanges(t *testing.T) {
	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	result := &infer.ApplyResult{
		DryRun:  true,
		Applied: []string{},
		Skipped: []string{},
	}

	err := renderApplyTable(cmd, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Dry run")) {
		t.Errorf("expected 'Dry run' message even with no changes, got: %s", output)
	}
}
