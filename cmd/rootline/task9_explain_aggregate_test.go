package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestTask9ExplainAggregateUsesSourceBackedChildIndexInputs(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
root: true
schema:
  status:
    type: string
    source: body.section["## Status"]
aggregate:
  done_children: 'len(filter(children, {.status == "done"}))'
`), 0o644)
	mustWriteFile(t, filepath.Join(dir, "README.md"), []byte("# Root\n"), 0o644)
	if err := os.Mkdir(filepath.Join(dir, "child"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(dir, "child", "README.md"), []byte("# Child\n\n## Status\n\ndone\n"), 0o644)
	declareTestBoundary(t, dir)

	out, err := runCmd(t, "explain", filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatalf("explain aggregate failed: %v\n%s", err, out)
	}
	var explain struct {
		Fields []map[string]any `json:"fields"`
	}
	if err := json.Unmarshal([]byte(out), &explain); err != nil {
		t.Fatalf("invalid explain JSON: %v\n%s", err, out)
	}
	for _, field := range explain.Fields {
		if field["name"] == "done_children" {
			if field["value"] != float64(1) || field["origin"] != "aggregate" {
				t.Fatalf("done_children explain field = %#v, want aggregate count 1 from child body source", field)
			}
			return
		}
	}
	t.Fatalf("explain fields missing done_children: %#v", explain.Fields)
}
