package fix

import (
	"strings"
	"testing"
)

// These tests cover the format-preserving frontmatter rewrite required by
// issue #60 defect 3: a one-field mutation must produce a one-field diff.
//
// The guarantee is deliberately bounded to what a yaml.Node round-trip can
// deliver. gopkg.in/yaml.v3 rebuilds output from the node tree rather than from
// the source bytes, so key order, comments and each scalar's quoting style
// survive, but inter-token whitespace and nested indentation are normalized.
// Nothing here asserts byte equality of the whole block.

func TestRewriteFrontmatter_PreservesKeyOrder(t *testing.T) {
	original := "---\nzzz: last\nestado: Pending\naaa: first\n---\n\n# Doc\n\nbody\n"
	fm := map[string]any{"zzz": "last", "estado": "Done", "aaa": "first"}

	result := RewriteFrontmatter(original, fm)

	zIdx := strings.Index(result, "zzz")
	eIdx := strings.Index(result, "estado")
	aIdx := strings.Index(result, "aaa")
	if zIdx == -1 || eIdx == -1 || aIdx == -1 {
		t.Fatalf("expected all three keys present, got:\n%s", result)
	}
	if zIdx >= eIdx || eIdx >= aIdx {
		t.Errorf("expected original order zzz < estado < aaa, got:\n%s", result)
	}
}

func TestRewriteFrontmatter_PreservesComments(t *testing.T) {
	original := "---\nzzz: last # comment A\nestado: Pending\n---\n\n# Doc\n"
	fm := map[string]any{"zzz": "last", "estado": "Done"}

	result := RewriteFrontmatter(original, fm)

	if !strings.Contains(result, "# comment A") {
		t.Errorf("expected YAML comment retained, got:\n%s", result)
	}
}

func TestRewriteFrontmatter_PreservesCommentOnMutatedKey(t *testing.T) {
	original := "---\nestado: Pending # current phase\n---\n\n# Doc\n"
	fm := map[string]any{"estado": "Done"}

	result := RewriteFrontmatter(original, fm)

	if !strings.Contains(result, "estado: Done") {
		t.Errorf("expected value updated, got:\n%s", result)
	}
	if !strings.Contains(result, "# current phase") {
		t.Errorf("expected comment on the mutated key retained, got:\n%s", result)
	}
}

func TestRewriteFrontmatter_AppendsNewKeyAfterExisting(t *testing.T) {
	original := "---\ntitle: A\nstatus: open\n---\n\n# Doc\n"
	fm := map[string]any{"title": "A", "status": "open", "labels": "x"}

	result := RewriteFrontmatter(original, fm)

	tIdx := strings.Index(result, "title")
	sIdx := strings.Index(result, "status")
	lIdx := strings.Index(result, "labels")
	if lIdx == -1 {
		t.Fatalf("expected new key appended, got:\n%s", result)
	}
	if tIdx >= sIdx || sIdx >= lIdx {
		t.Errorf("expected new key after existing ones, got:\n%s", result)
	}
}

func TestRewriteFrontmatter_RemovesDroppedKey(t *testing.T) {
	original := "---\ntitle: A\nobsolete: gone\nstatus: open\n---\n\n# Doc\n"
	fm := map[string]any{"title": "A", "status": "open"}

	result := RewriteFrontmatter(original, fm)

	if strings.Contains(result, "obsolete") {
		t.Errorf("expected key absent from the map to be removed, got:\n%s", result)
	}
	if !strings.Contains(result, "title: A") || !strings.Contains(result, "status: open") {
		t.Errorf("expected remaining keys intact, got:\n%s", result)
	}
}

func TestRewriteFrontmatter_PreservesQuotingStyle(t *testing.T) {
	original := "---\nstatus: \"pending\"\nother: plain\n---\n\n# Doc\n"
	fm := map[string]any{"status": "pending", "other": "changed"}

	result := RewriteFrontmatter(original, fm)

	if !strings.Contains(result, `status: "pending"`) {
		t.Errorf("expected double-quoted style retained on the untouched key, got:\n%s", result)
	}
}

func TestRewriteFrontmatter_UntouchedFieldsKeepValues(t *testing.T) {
	original := "---\na: 1\nb: two\nc: true\nd: Pending\n---\n\n# Doc\n"
	fm := map[string]any{"a": 1, "b": "two", "c": true, "d": "Done"}

	result := RewriteFrontmatter(original, fm)

	for _, want := range []string{"a: 1", "b: two", "c: true", "d: Done"} {
		if !strings.Contains(result, want) {
			t.Errorf("expected %q in output, got:\n%s", want, result)
		}
	}
}

func TestRewriteFrontmatter_MalformedYAMLFallsBack(t *testing.T) {
	// Frontmatter delimiters are well formed but the content is not valid YAML.
	// RewriteFrontmatter cannot return an error (its signature is load-bearing
	// for the six call sites), so it must degrade to the map-based rebuild
	// rather than lose the document.
	original := "---\nthis: is: not: valid: yaml\n---\n\n# Doc\n\nbody\n"
	fm := map[string]any{"estado": "Done"}

	result := RewriteFrontmatter(original, fm)

	if !strings.Contains(result, "estado: Done") {
		t.Errorf("expected fallback to still write the field, got:\n%s", result)
	}
	if !strings.Contains(result, "body") {
		t.Errorf("expected body preserved through fallback, got:\n%s", result)
	}
}

func TestRewriteFrontmatter_BodyUntouchedByNodeRewrite(t *testing.T) {
	// The body must survive verbatim, including content that looks like YAML.
	original := "---\nestado: Pending\n---\n\n# Doc\n\nkey: not frontmatter\n\n---\n\ntail\n"
	fm := map[string]any{"estado": "Done"}

	result := RewriteFrontmatter(original, fm)

	body := "\n# Doc\n\nkey: not frontmatter\n\n---\n\ntail\n"
	if !strings.HasSuffix(result, body) {
		t.Errorf("expected body preserved verbatim, got:\n%s", result)
	}
}
