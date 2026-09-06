package fix

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
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

func TestRewriteFrontmatter_AppendsToEmptyLeadingBlock(t *testing.T) {
	tests := []struct {
		name     string
		original string
	}{
		{
			name:     "adjacent delimiters",
			original: "---\n---\n# Doc\n\nbody\n",
		},
		{
			name:     "blank line between delimiters",
			original: "---\n\n---\n# Doc\n\nbody\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RewriteFrontmatter(tt.original, map[string]any{"alpha": "alpha-default"})

			want := "---\nalpha: alpha-default\n---\n# Doc\n\nbody\n"
			if result != want {
				t.Errorf("RewriteFrontmatter() = %q, want %q", result, want)
			}
		})
	}
}

func TestRewriteFrontmatter_PreservesStandaloneComments(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		comments []string
	}{
		{"one comment", "# Keep this metadata comment.\n", []string{"# Keep this metadata comment."}},
		{"blank and indented comments", "\n# First note.\n\n  # Second note: café.\n\n", []string{"# First note.", "# Second note: café."}},
		{"comment punctuation", "# ---\n# status: do not infer from this note\n", []string{"# ---", "# status: do not infer from this note"}},
	}
	body := "## Body\n\nKeep café and <literal> unchanged.\n\n---\n\nTail\n"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RewriteFrontmatter("---\n"+tt.text+"---\n"+body, map[string]any{"status": "ready"})
			// A comment containing --- is not a frontmatter delimiter.
			end := strings.Index(result, "\n---\n")
			if end < 0 || result[end+5:] != body || !strings.HasPrefix(result, "---\n") {
				t.Fatalf("body or frontmatter boundaries changed: %q", result)
			}
			frontmatter := result[4 : end+1]
			var values map[string]any
			if err := yaml.Unmarshal([]byte(frontmatter), &values); err != nil {
				t.Fatalf("invalid rewritten YAML: %v\n%s", err, result)
			}
			if len(values) != 1 || values["status"] != "ready" {
				t.Fatalf("requested field was not added: %v", values)
			}
			previous := -1
			for _, comment := range tt.comments {
				position := strings.Index(frontmatter, comment)
				if position <= previous || strings.Count(frontmatter, comment) != 1 {
					t.Errorf("comment %q lost, duplicated or reordered in %q", comment, frontmatter)
				}
				previous = position
			}
			// Once populated, a later field mutation must still retain the notes.
			next := RewriteFrontmatter(result, map[string]any{"status": "done"})
			for _, comment := range tt.comments {
				if strings.Count(next, comment) != 1 {
					t.Errorf("later mutation lost or duplicated comment %q: %q", comment, next)
				}
			}
		})
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
