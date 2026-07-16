package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupTraversalDir builds a corpus for link-traversal queries:
//
//	wiki/sources/witness-a.md (corroborated) --supports--> wiki/entities/tool-a.md
//	wiki/sources/witness-b.md (unverified)   --supports--> wiki/entities/tool-b.md
//	wiki/sources/witness-c.md (corroborated) --mentions--> wiki/entities/tool-b.md
//	raw/decoy.md              (corroborated) --supports--> wiki/entities/tool-b.md  (outside wiki/)
func setupTraversalDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"wiki/entities/tool-a.md":   "---\nkind: tool\n---\n# Tool A\n",
		"wiki/entities/tool-b.md":   "---\nkind: tool\n---\n# Tool B\n",
		"wiki/entities/lib-x.md":    "---\nkind: library\n---\n# Lib X\n",
		"wiki/sources/witness-a.md": "---\nverification: corroborated\n---\n# Witness A\n\n[[supports:tool-a.md]]\n",
		"wiki/sources/witness-b.md": "---\nverification: unverified\n---\n# Witness B\n\n[[supports:tool-b.md]]\n",
		"wiki/sources/witness-c.md": "---\nverification: corroborated\n---\n# Witness C\n\n[[mentions:tool-b.md]]\n",
		"raw/decoy.md":              "---\nverification: corroborated\n---\n# Decoy\n\n[[supports:tool-b.md]]\n",
	}
	for rel, content := range files {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	return dir
}

func TestQueryHasInbound(t *testing.T) {
	dir := setupTraversalDir(t)
	out, err := runCmd(t, "query", filepath.Join(dir, "wiki/entities"),
		"--has-inbound", "verification == 'corroborated'",
		"--graph-root", filepath.Join(dir, "wiki"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// tool-a via witness-a (supports), tool-b via witness-c (mentions).
	if !strings.Contains(out, "tool-a.md") || !strings.Contains(out, "tool-b.md") {
		t.Errorf("expected tool-a.md and tool-b.md, got: %s", out)
	}
	if strings.Contains(out, "lib-x.md") {
		t.Errorf("expected lib-x.md (no inbound) to be filtered out, got: %s", out)
	}
}

func TestQueryHasInboundTypeFilter(t *testing.T) {
	dir := setupTraversalDir(t)
	out, err := runCmd(t, "query", filepath.Join(dir, "wiki/entities"),
		"--has-inbound", "verification == 'corroborated'",
		"--inbound-type", "supports",
		"--graph-root", filepath.Join(dir, "wiki"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "tool-a.md") {
		t.Errorf("expected tool-a.md, got: %s", out)
	}
	// tool-b's corroborated witness links via mentions, not supports.
	if strings.Contains(out, "tool-b.md") {
		t.Errorf("expected tool-b.md to be filtered out by --inbound-type, got: %s", out)
	}
}

func TestQueryHasOutbound(t *testing.T) {
	dir := setupTraversalDir(t)
	out, err := runCmd(t, "query", filepath.Join(dir, "wiki/sources"),
		"--has-outbound", "kind == 'tool'",
		"--graph-root", filepath.Join(dir, "wiki"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"witness-a.md", "witness-b.md", "witness-c.md"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %s in results, got: %s", want, out)
		}
	}
}

func TestQueryTraversalComposesWithWhere(t *testing.T) {
	dir := setupTraversalDir(t)
	// The motivating verified-tools one-liner.
	out, err := runCmd(t, "query", filepath.Join(dir, "wiki/entities"),
		"--where", "kind == 'tool'",
		"--has-inbound", "verification == 'corroborated'",
		"--inbound-type", "supports",
		"--graph-root", filepath.Join(dir, "wiki"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "tool-a.md") {
		t.Errorf("expected tool-a.md, got: %s", out)
	}
	for _, notWant := range []string{"tool-b.md", "lib-x.md", "witness"} {
		if strings.Contains(out, notWant) {
			t.Errorf("expected %s to be absent, got: %s", notWant, out)
		}
	}
	if !strings.Contains(out, `"kind":"rootline/query"`) {
		t.Errorf("expected versioned query envelope, got: %s", out)
	}
}

func TestQueryTraversalNoMatch(t *testing.T) {
	dir := setupTraversalDir(t)
	out, err := runCmd(t, "query", filepath.Join(dir, "wiki/entities"),
		"--has-inbound", "verification == 'retracted'",
		"--graph-root", filepath.Join(dir, "wiki"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"count":0`) {
		t.Errorf("expected count 0 envelope, got: %s", out)
	}
}

func TestQueryGraphRootBoundsEdgeUniverse(t *testing.T) {
	dir := setupTraversalDir(t)

	// Wide root (repo root): the raw/ decoy corroborates tool-b via supports.
	out, err := runCmd(t, "query", filepath.Join(dir, "wiki/entities"),
		"--has-inbound", "verification == 'corroborated'",
		"--inbound-type", "supports",
		"--graph-root", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "tool-b.md") {
		t.Errorf("wide graph-root: expected decoy to corroborate tool-b.md, got: %s", out)
	}

	// Bounded root: raw/ is outside the universe, tool-b must drop out.
	out, err = runCmd(t, "query", filepath.Join(dir, "wiki/entities"),
		"--has-inbound", "verification == 'corroborated'",
		"--inbound-type", "supports",
		"--graph-root", filepath.Join(dir, "wiki"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, "tool-b.md") {
		t.Errorf("bounded graph-root: expected raw/ decoy excluded, got: %s", out)
	}
}

func TestQueryGraphRootDefaultsToQueryPath(t *testing.T) {
	dir := setupTraversalDir(t)
	// Default graph-root = query path: sources are outside, so no inbound edges exist.
	out, err := runCmd(t, "query", filepath.Join(dir, "wiki/entities"),
		"--has-inbound", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"count":0`) {
		t.Errorf("expected count 0 with default graph-root, got: %s", out)
	}
}

func TestQueryTraversalPathMatchesPlainQueryFormat(t *testing.T) {
	dir := setupTraversalDir(t)

	plain, err := runCmd(t, "query", filepath.Join(dir, "wiki/entities"),
		"--where", "kind == 'tool'")
	if err != nil {
		t.Fatalf("plain query: unexpected error: %v", err)
	}
	traversal, err := runCmd(t, "query", filepath.Join(dir, "wiki/entities"),
		"--where", "kind == 'tool'",
		"--has-inbound", "",
		"--graph-root", filepath.Join(dir, "wiki"))
	if err != nil {
		t.Fatalf("traversal query: unexpected error: %v", err)
	}

	// Both modes must emit path in the same canonical format: relative to
	// the query path, without the graph-root prefix.
	for mode, out := range map[string]string{"plain": plain, "traversal": traversal} {
		if !strings.Contains(out, `"path":"tool-a.md"`) {
			t.Errorf("%s: expected query-path-relative path \"tool-a.md\", got: %s", mode, out)
		}
		if strings.Contains(out, `"path":"entities/`) {
			t.Errorf("%s: expected no graph-root prefix on path, got: %s", mode, out)
		}
	}
}

func TestQueryTraversalFlagValidation(t *testing.T) {
	dir := setupTraversalDir(t)
	cases := []struct {
		name string
		args []string
	}{
		{"inbound-type without has-inbound", []string{
			"query", filepath.Join(dir, "wiki/entities"), "--inbound-type", "supports"}},
		{"outbound-type without has-outbound", []string{
			"query", filepath.Join(dir, "wiki/sources"), "--outbound-type", "supports"}},
		{"graph-root without traversal", []string{
			"query", filepath.Join(dir, "wiki/entities"), "--graph-root", filepath.Join(dir, "wiki")}},
		{"query path outside graph-root", []string{
			"query", filepath.Join(dir, "wiki/entities"), "--has-inbound", "",
			"--graph-root", filepath.Join(dir, "wiki/sources")}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := runCmd(t, tc.args...); err == nil {
				t.Errorf("expected error for %s", tc.name)
			}
		})
	}
}
