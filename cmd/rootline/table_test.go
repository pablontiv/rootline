package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderTable(t *testing.T) {
	buf := new(bytes.Buffer)
	headers := []string{"Name", "Age", "City"}
	rows := [][]string{
		{"Alice", "30", "NYC"},
		{"Bob", "25", "LA"},
	}

	renderTable(buf, headers, rows)
	out := buf.String()

	if !strings.Contains(out, "Name") {
		t.Errorf("expected header 'Name' in output")
	}
	if !strings.Contains(out, "Alice") {
		t.Errorf("expected row 'Alice' in output")
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 4 { // header + separator + 2 rows
		t.Errorf("expected 4 lines, got %d: %s", len(lines), out)
	}
}

func TestRenderTableEmpty(t *testing.T) {
	buf := new(bytes.Buffer)
	renderTable(buf, []string{"A", "B"}, nil)
	out := buf.String()
	if !strings.Contains(out, "A") {
		t.Errorf("expected header in empty table output")
	}
}
