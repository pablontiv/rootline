package rules

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// setupMonotonicViolations builds a two-level monotonic chain whose child
// loosens every category the resolver detects: type, required, severity, enum
// values, and both structural bounds.
func setupMonotonicViolations(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
root: true
monotonic: true
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    values: [Pending, Done]
    required: true
    severity: error
  narrow:
    type: enum
    values: [a, b]
structural:
  subdirs:
    min_children: 2
    max_children: 5
`))
	mustWriteStemTestFile(t, filepath.Join(dir, "child", ".stem"), []byte(`version: 2
schema:
  estado:
    type: string
    required: false
    severity: warn
  narrow:
    type: enum
    values: [a, b, c]
structural:
  subdirs:
    min_children: 1
    max_children: 9
`))
	mustWriteStemTestFile(t, filepath.Join(dir, "x.md"), []byte("---\nestado: Pending\n---\n# x\n"))
	mustWriteStemTestFile(t, filepath.Join(dir, "child", "y.md"), []byte("---\nestado: Pending\n---\n# y\n"))
	return dir
}

func monotonicChecks(t *testing.T, dir string) []StemHealthCheck {
	t.Helper()
	res, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("ValidateStemHealth: %v", err)
	}
	var out []StemHealthCheck
	for _, c := range res.Checks {
		if c.Name == "monotonic-violations" {
			out = append(out, c)
		}
	}
	return out
}

func TestMonotonicViolations_DiscriminateCategories(t *testing.T) {
	checks := monotonicChecks(t, setupMonotonicViolations(t))
	if len(checks) == 0 {
		t.Fatal("no monotonic-violations checks emitted")
	}

	joined := make([]string, len(checks))
	for i, c := range checks {
		joined[i] = c.Field + " | " + c.Message
	}
	all := strings.Join(joined, "\n")

	// Every category must be named for what it is. Before the fix, four of the
	// five rendered as "(type change: ...)".
	wants := []struct {
		desc     string
		fragment string
	}{
		{"type widening", "widens type"},
		{"required loosening", "loosens required"},
		{"severity loosening", "loosens severity"},
		{"enum extension", "enum extended with disallowed value(s)"},
		{"min_children loosening", "structural.subdirs.min_children"},
		{"max_children loosening", "structural.subdirs.max_children"},
	}
	for _, w := range wants {
		if !strings.Contains(all, w.fragment) {
			t.Errorf("%s: no check mentions %q\ngot:\n%s", w.desc, w.fragment, all)
		}
	}
	if strings.Contains(all, "type change") {
		t.Errorf("stale \"type change\" wording still present:\n%s", all)
	}
}

func TestMonotonicViolations_StructuralFieldsAreDistinguishable(t *testing.T) {
	checks := monotonicChecks(t, setupMonotonicViolations(t))

	fields := map[string]int{}
	for _, c := range checks {
		fields[c.Field]++
	}
	if fields["structural"] != 0 {
		t.Errorf("field %q is the truncated form; want the full constraint path", "structural")
	}
	if fields["structural.subdirs.min_children"] != 1 {
		t.Errorf("min_children field count = %d, want 1 (fields: %v)", fields["structural.subdirs.min_children"], fields)
	}
	if fields["structural.subdirs.max_children"] != 1 {
		t.Errorf("max_children field count = %d, want 1 (fields: %v)", fields["structural.subdirs.max_children"], fields)
	}
	// Schema-field conflicts keep the bare field name, not the constraint suffix.
	if fields["estado"] < 3 {
		t.Errorf("estado field count = %d, want 3 (type, required, severity); fields: %v", fields["estado"], fields)
	}
	if fields["estado.type"] != 0 {
		t.Errorf("schema conflicts must not carry the %q constraint suffix", "estado.type")
	}
}

func TestNestedRootMarker_StaysInfoSeverity(t *testing.T) {
	dir := t.TempDir()
	stem := []byte(`version: 2
root: true
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    values: [Pending, Done]
`)
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), stem)
	mustWriteStemTestFile(t, filepath.Join(dir, "child", ".stem"), stem)
	mustWriteStemTestFile(t, filepath.Join(dir, "a.md"), []byte("---\nestado: Pending\n---\n# a\n"))
	mustWriteStemTestFile(t, filepath.Join(dir, "child", "b.md"), []byte("---\nestado: Done\n---\n# b\n"))

	res, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("ValidateStemHealth: %v", err)
	}
	diags := StemHealthDiagnostics(res)

	found := false
	for _, d := range diags {
		if d.Check != "nested-root-marker" {
			continue
		}
		found = true
		if d.Severity != "info" {
			t.Errorf("nested-root-marker severity = %q, want %q", d.Severity, "info")
		}
	}
	if !found {
		t.Fatalf("nested-root-marker diagnostic missing; got %+v", diags)
	}
}
