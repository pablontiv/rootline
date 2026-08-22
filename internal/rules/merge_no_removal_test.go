package rules

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A child .stem never removes anything a parent declared, in any section.
// Needing to remove a field means the structure is wrong; the fix belongs in
// the structure, not in a child that quietly drops a parent's guarantee.
func TestMergeStemFiles_ChildNeverRemovesInheritedKeys(t *testing.T) {
	parent := StemEntry{Path: "/p/.stem", Stem: &StemFile{
		Version:   2,
		Schema:    map[string]SchemaField{"titulo": {Type: "string"}, "removed": {Type: "string"}},
		Derive:    map[string]any{"slug": "slugify(titulo)", "gone": "upper(titulo)"},
		Aggregate: map[string]any{"total": "count(children)", "dropped": "sum(children.n)"},
	}}
	child := StemEntry{Path: "/p/c/.stem", Stem: &StemFile{
		Version:   2,
		Schema:    map[string]SchemaField{"removed": {declaration: schemaFieldDeclarationMetadata{NullField: true}}},
		Derive:    map[string]any{"gone": nil},
		Aggregate: map[string]any{"dropped": nil},
	}}

	merged := MergeStemFiles([]StemEntry{parent, child})

	if _, ok := merged.Schema["removed"]; !ok {
		t.Error("schema field \"removed\" was dropped by a child null; a child must never remove an inherited field")
	}
	if _, ok := merged.Derive["gone"]; !ok {
		t.Error("derive key \"gone\" was dropped by a child null")
	}
	if _, ok := merged.Aggregate["dropped"]; !ok {
		t.Error("aggregate key \"dropped\" was dropped by a child null")
	}
	if _, ok := merged.Schema["titulo"]; !ok {
		t.Error("untouched parent field disappeared")
	}
}

// The nested case: a child cannot reach into an inherited map and null one of
// its leaves either.
func TestMergeStemFiles_ChildNeverRemovesNestedKeys(t *testing.T) {
	parent := StemEntry{Path: "/p/.stem", Stem: &StemFile{
		Version:   2,
		Aggregate: map[string]any{"rollup": map[string]any{"keep": "1", "drop": "2"}},
	}}
	child := StemEntry{Path: "/p/c/.stem", Stem: &StemFile{
		Version:   2,
		Aggregate: map[string]any{"rollup": map[string]any{"drop": nil}},
	}}

	merged := MergeStemFiles([]StemEntry{parent, child})

	rollup, ok := merged.Aggregate["rollup"].(map[string]any)
	if !ok {
		t.Fatalf("rollup = %#v, want a map", merged.Aggregate["rollup"])
	}
	if _, ok := rollup["drop"]; !ok {
		t.Error("nested key \"drop\" was dropped by a child null")
	}
}

// Setting a schema field to null is refused where the stem is read, so a
// zero-valued declaration never reaches the pipeline. Letting it through was
// what produced three unrelated diagnostics — incomplete-type, a required
// loosening and a type change to "" — none of which named the actual mistake.
func TestParseStem_RejectsNullSchemaField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".stem")
	body := "version: 2\nschema:\n  removed: null\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ParseStem(path, raw)
	if err == nil {
		t.Fatal("ParseStem accepted a null schema field; a child cannot remove an inherited field")
	}
	msg := err.Error()
	for _, want := range []string{"removed", "null"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error %q does not mention %q", msg, want)
		}
	}
}

// The whole point is one accurate message instead of three misleading ones.
func TestValidateStemHealth_NullSchemaFieldReportsOneError(t *testing.T) {
	dir := t.TempDir()
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\nroot: true\nscope:\n  match: \"*.md\"\nschema:\n  titulo:\n    type: string\n    required: true\n  removed:\n    type: string\n    required: true\n"))
	child := filepath.Join(dir, "child")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteStemTestFile(t, filepath.Join(child, ".stem"), []byte("version: 2\nschema:\n  removed: null\n"))
	mustWriteStemTestFile(t, filepath.Join(child, "c.md"), []byte("---\ntitulo: C\nremoved: x\n---\n\n# C\n"))

	diags, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	var errs []StemHealthCheck
	for _, c := range diags.Checks {
		if c.Status == "fail" {
			errs = append(errs, c)
		}
	}
	if len(errs) != 1 {
		t.Fatalf("got %d error diagnostics, want exactly 1:\n%+v", len(errs), errs)
	}
	if !strings.Contains(errs[0].Message, "removed") {
		t.Errorf("message %q does not name the offending field", errs[0].Message)
	}
}
