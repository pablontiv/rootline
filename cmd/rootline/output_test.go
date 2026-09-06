package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// setupFormatProject builds the smallest governed corpus that every output
// path can run against: two records, one broken link, one cycle.
func setupFormatProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	if err := os.MkdirAll(docs, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(docs, ".stem"),
		[]byte("version: 2\nroot: true\nschema:\n  estado:\n    type: enum\n    values: [Pending, Done]\n"), 0o644)
	mustWriteFile(t, filepath.Join(docs, "r1.md"),
		[]byte("---\nestado: Pending\n---\n# R1\n\nSee [[r2]].\n\nAlso [[r99-does-not-exist]].\n"), 0o644)
	mustWriteFile(t, filepath.Join(docs, "r2.md"),
		[]byte("---\nestado: Done\n---\n# R2\n\nSee [[r1]].\n"), 0o644)
	return docs
}

// walkCommands returns every command in the tree, root included.
func walkCommands(c *cobra.Command) []*cobra.Command {
	all := []*cobra.Command{c}
	for _, sub := range c.Commands() {
		all = append(all, walkCommands(sub)...)
	}
	return all
}

// Every command must declare which of the four advertised formats it can
// produce. A new command with no entry is the exact hole that let --output go
// unvalidated for 23 call sites; this test closes it at compile-and-test time
// rather than at the user's terminal.
func TestCommandOutputFormats_CoversEveryCommand(t *testing.T) {
	for _, c := range walkCommands(rootCmd) {
		path := c.CommandPath()
		if _, ok := commandOutputFormats[path]; !ok {
			t.Errorf("command %q has no entry in commandOutputFormats; declare its supported formats (or formatAgnostic)", path)
		}
	}
}

// Declared formats must themselves be legal values of --output.
func TestCommandOutputFormats_DeclaresOnlyAdvertisedValues(t *testing.T) {
	for path, formats := range commandOutputFormats {
		for _, f := range formats {
			if !isAdvertisedFormat(f) {
				t.Errorf("command %q declares unknown format %q", path, f)
			}
		}
	}
}

func TestOutput_RejectsUnknownValue(t *testing.T) {
	docs := setupFormatProject(t)

	for _, bad := range []string{"sdlkfj", "JSON", ""} {
		_, err := runCmd(t, "query", docs, "--count", "-o", bad)
		if err == nil {
			t.Fatalf("-o %q: expected an error, got none", bad)
		}
		if !strings.Contains(err.Error(), "unknown output format") {
			t.Errorf("-o %q: error = %v, want it to name the unknown format", bad, err)
		}
		for _, legal := range []string{"json", "jsonl", "csv", "table"} {
			if !strings.Contains(err.Error(), legal) {
				t.Errorf("-o %q: error %v does not list the legal value %q", bad, err, legal)
			}
		}
	}
}

// The four commands that fell through to JSON for jsonl/csv (issue #63 §2).
func TestOutput_RejectsUnsupportedFormatPerCommand(t *testing.T) {
	docs := setupFormatProject(t)
	skillFixture := newSkillCommandFixture(t)

	cases := []struct {
		name          string
		args          []string
		wantSupported []string
	}{
		{"stats csv", []string{"stats", docs, "-o", "csv"}, []string{"json", "table"}},
		{"stats jsonl", []string{"stats", docs, "-o", "jsonl"}, []string{"json", "table"}},
		{"describe jsonl", []string{"describe", docs, "-o", "jsonl"}, []string{"json", "table"}},
		{"describe csv", []string{"describe", docs, "-o", "csv"}, []string{"json", "table"}},
		{"explain csv", []string{"explain", filepath.Join(docs, "r1.md"), "-o", "csv"}, []string{"json", "table"}},
		{"validate csv", []string{"validate", "--all", docs, "-o", "csv"}, []string{"json", "table"}},
		{"validate jsonl", []string{"validate", "--all", docs, "-o", "jsonl"}, []string{"json", "table"}},
		// The two with an inverted test that fell through to a diagram (§3).
		{"tree csv", []string{"tree", docs, "-o", "csv"}, []string{"json", "table"}},
		{"tree jsonl", []string{"tree", docs, "-o", "jsonl"}, []string{"json", "table"}},
		{"graph jsonl", []string{"graph", docs, "-o", "jsonl"}, []string{"json", "table"}},
		{"graph csv", []string{"graph", docs, "-o", "csv"}, []string{"json", "table"}},
		{"skill status table", []string{"skill", "status", "--source", skillFixture.repo, "-o", "table"}, []string{"json"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := runCmd(t, tc.args...)
			if err == nil {
				t.Fatalf("expected an error, got none; output: %s", out)
			}
			if !strings.Contains(err.Error(), "does not support output format") {
				t.Errorf("error = %v, want it to say the command does not support the format", err)
			}
			for _, supported := range tc.wantSupported {
				if !strings.Contains(err.Error(), supported) {
					t.Errorf("error = %v, want it to list %s as a supported value", err, supported)
				}
			}
		})
	}
}

// query is the one command that genuinely implements all four.
func TestOutput_QueryAcceptsEveryAdvertisedFormat(t *testing.T) {
	docs := setupFormatProject(t)

	for _, f := range []string{"json", "table"} {
		if _, err := runCmd(t, "query", docs, "-o", f); err != nil {
			t.Errorf("query -o %s: unexpected error: %v", f, err)
		}
	}
	for _, f := range []string{"jsonl", "csv"} {
		if _, err := runCmd(t, "query", docs, "--select", "path,estado", "-o", f); err != nil {
			t.Errorf("query --select -o %s: unexpected error: %v", f, err)
		}
	}
}

func TestOutput_QuerySelectTableUsesSelectedColumns(t *testing.T) {
	docs := setupFormatProject(t)

	out, err := runCmd(t, "query", docs, "--select", "nonexistent,path,estado", "-o", "table")
	if err != nil {
		t.Fatalf("query --select -o table: unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected header, separator, and two data rows; got %d lines:\n%s", len(lines), out)
	}
	if got := strings.Fields(lines[0]); !slices.Equal(got, []string{"nonexistent", "path", "estado"}) {
		t.Fatalf("header fields = %v, want selected order; output:\n%s", got, out)
	}
	pathColumn := strings.Index(lines[0], "path")
	estadoColumn := strings.Index(lines[0], "estado")
	if pathColumn <= 0 || estadoColumn <= pathColumn {
		t.Fatalf("could not locate selected columns in header %q", lines[0])
	}
	for _, line := range lines[2:] {
		if got := strings.TrimSpace(line[:pathColumn]); got != "" {
			t.Errorf("missing selected field rendered as %q, want empty cell; row %q", got, line)
		}
		if got := strings.TrimSpace(line[pathColumn:estadoColumn]); got == "" {
			t.Errorf("path cell is empty in row %q", line)
		}
		if got := strings.TrimSpace(line[estadoColumn:]); got == "" {
			t.Errorf("estado cell is empty in row %q", line)
		}
	}
	if strings.Contains(out, `"kind":"rootline/query"`) {
		t.Fatalf("table output fell back to JSON:\n%s", out)
	}
}

func TestOutput_QuerySelectTableEmptyResultKeepsHeader(t *testing.T) {
	docs := setupFormatProject(t)

	out, err := runCmd(t, "query", docs, "--where", "estado == 'Never'", "--select", "estado,path", "-o", "table")
	if err != nil {
		t.Fatalf("empty query --select -o table: unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected only header and separator, got %d lines:\n%s", len(lines), out)
	}
	if got := strings.Fields(lines[0]); !slices.Equal(got, []string{"estado", "path"}) {
		t.Fatalf("header fields = %v, want selected order; output:\n%s", got, out)
	}
}

// tree and graph bound their diagram to the wrong side of the test: anything
// that was not "json" rendered a diagram. A diagram belongs to -o table.
func TestOutput_TreeBindsDiagramToTable(t *testing.T) {
	docs := setupFormatProject(t)

	out, err := runCmd(t, "tree", docs, "-o", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "docs") || strings.Contains(out, `"kind"`) {
		t.Errorf("-o table should render the ASCII tree, got: %s", out)
	}

	out, err = runCmd(t, "tree", docs, "-o", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var tr TreeResult
	if err := json.Unmarshal([]byte(out), &tr); err != nil {
		t.Fatalf("-o json should render JSON, got %s", out)
	}
}

func TestOutput_GraphBindsDiagramToTable(t *testing.T) {
	docs := setupFormatProject(t)

	out, err := runCmd(t, "graph", docs, "-o", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "digraph") {
		t.Errorf("-o table should render DOT, got: %s", out)
	}

	out, err = runCmd(t, "graph", docs, "-o", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var gr GraphResult
	if err := json.Unmarshal([]byte(out), &gr); err != nil {
		t.Fatalf("-o json should render JSON, got %s", out)
	}
}

// graph --check is documented as a text-plus-exit-code validator. It must not
// silently discard an --output the caller explicitly set...
func TestGraphCheck_RejectsExplicitOutput(t *testing.T) {
	docs := setupFormatProject(t)

	for _, f := range []string{"json", "table"} {
		_, err := runCmd(t, "graph", docs, "--check", "-o", f)
		if err == nil {
			t.Fatalf("graph --check -o %s: expected an error, got none", f)
		}
		if !strings.Contains(err.Error(), "--check") || !strings.Contains(err.Error(), "--output") {
			t.Errorf("graph --check -o %s: error = %v, want it to name both flags", f, err)
		}
	}
}

// ...but the default --output is not an explicit request, and the documented
// invocation `rootline graph <path> --check` must keep working.
func TestGraphCheck_DefaultOutputStillRuns(t *testing.T) {
	docs := setupFormatProject(t)

	out, err := runCmd(t, "graph", docs, "--check")
	if err == nil {
		t.Fatalf("fixture has a broken link, expected a validation failure; output: %s", out)
	}
	if !strings.Contains(out, "Broken links") {
		t.Errorf("expected the text report, got: %s", out)
	}
}
