package derive

import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestInjectLinkedFields_BlockedBy(t *testing.T) {
	// Record A has a [[blocks:B]] link. B has estado=Completed.
	// The stem defines link type "blocks" with field "blocked_by".
	// After injection, env["blocked_by"] should be ["Completed"].
	target := &extract.Record{
		Path:        "tasks/B.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"estado": "Completed"},
	}
	source := &extract.Record{
		Path:        "tasks/A.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"estado": "Pending"},
		Links:       []extract.Link{{Type: "blocks", Target: "B.md", Line: 1}},
	}
	stem := &rules.StemFile{
		Links: rules.LinkSchema{
			Rules: map[string]rules.LinkRule{
				"blocks": {Target: "*.md", Field: "blocked_by"},
			},
		},
	}
	resolver := NewMapResolver([]*extract.Record{source, target})

	env := BuildEnv(source.Frontmatter)
	InjectLinkedFields(env, source, stem, resolver)

	val, ok := env["blocked_by"]
	if !ok {
		t.Fatal("expected env to contain 'blocked_by'")
	}
	slice, ok := val.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", val)
	}
	if len(slice) != 1 {
		t.Fatalf("expected 1 value, got %d", len(slice))
	}
	if slice[0] != "Completed" {
		t.Errorf("blocked_by[0] = %v, want %q", slice[0], "Completed")
	}
}

func TestInjectLinkedFields_NoLinks(t *testing.T) {
	// Record with no links should leave env unchanged.
	source := &extract.Record{
		Path:        "tasks/A.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"estado": "Pending"},
	}
	stem := &rules.StemFile{
		Links: rules.LinkSchema{
			Rules: map[string]rules.LinkRule{
				"blocks": {Target: "*.md", Field: "blocked_by"},
			},
		},
	}
	resolver := NewMapResolver([]*extract.Record{source})

	env := BuildEnv(source.Frontmatter)
	InjectLinkedFields(env, source, stem, resolver)

	if _, ok := env["blocked_by"]; ok {
		t.Error("expected no 'blocked_by' in env for record without links")
	}
}

func TestInjectLinkedFields_NonExistentTarget(t *testing.T) {
	// Link to a target that doesn't exist should be silently skipped.
	source := &extract.Record{
		Path:        "tasks/A.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"estado": "Pending"},
		Links:       []extract.Link{{Type: "blocks", Target: "nonexistent.md", Line: 1}},
	}
	stem := &rules.StemFile{
		Links: rules.LinkSchema{
			Rules: map[string]rules.LinkRule{
				"blocks": {Target: "*.md", Field: "blocked_by"},
			},
		},
	}
	resolver := NewMapResolver([]*extract.Record{source})

	env := BuildEnv(source.Frontmatter)
	InjectLinkedFields(env, source, stem, resolver)

	if _, ok := env["blocked_by"]; ok {
		t.Error("expected no 'blocked_by' for non-existent target")
	}
}

func TestInjectLinkedFields_MultipleLinks(t *testing.T) {
	// Multiple links of the same type should produce a slice with all values.
	targetB := &extract.Record{
		Path:        "tasks/B.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"estado": "Completed"},
	}
	targetC := &extract.Record{
		Path:        "tasks/C.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"estado": "Pending"},
	}
	source := &extract.Record{
		Path:        "tasks/A.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"estado": "Pending"},
		Links: []extract.Link{
			{Type: "blocks", Target: "B.md", Line: 1},
			{Type: "blocks", Target: "C.md", Line: 2},
		},
	}
	stem := &rules.StemFile{
		Links: rules.LinkSchema{
			Rules: map[string]rules.LinkRule{
				"blocks": {Target: "*.md", Field: "blocked_by"},
			},
		},
	}
	resolver := NewMapResolver([]*extract.Record{source, targetB, targetC})

	env := BuildEnv(source.Frontmatter)
	InjectLinkedFields(env, source, stem, resolver)

	val, ok := env["blocked_by"]
	if !ok {
		t.Fatal("expected env to contain 'blocked_by'")
	}
	slice, ok := val.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", val)
	}
	if len(slice) != 2 {
		t.Fatalf("expected 2 values, got %d", len(slice))
	}
	// Values should be "Completed" and "Pending" (order matches link order).
	if slice[0] != "Completed" {
		t.Errorf("blocked_by[0] = %v, want %q", slice[0], "Completed")
	}
	if slice[1] != "Pending" {
		t.Errorf("blocked_by[1] = %v, want %q", slice[1], "Pending")
	}
}

func TestInjectLinkedFields_NilResolver(t *testing.T) {
	source := &extract.Record{
		Path:        "tasks/A.md",
		Frontmatter: map[string]any{"estado": "Pending"},
		Links:       []extract.Link{{Type: "blocks", Target: "B.md", Line: 1}},
	}
	stem := &rules.StemFile{
		Links: rules.LinkSchema{
			Rules: map[string]rules.LinkRule{
				"blocks": {Target: "*.md", Field: "blocked_by"},
			},
		},
	}

	env := BuildEnv(source.Frontmatter)
	InjectLinkedFields(env, source, stem, nil)

	if _, ok := env["blocked_by"]; ok {
		t.Error("expected no injection with nil resolver")
	}
}

func TestInjectLinkedFields_NoFieldInRule(t *testing.T) {
	// Link rule without Field should not inject anything.
	target := &extract.Record{
		Path:        "tasks/B.md",
		Frontmatter: map[string]any{"estado": "Completed"},
	}
	source := &extract.Record{
		Path:        "tasks/A.md",
		Frontmatter: map[string]any{"estado": "Pending"},
		Links:       []extract.Link{{Type: "blocks", Target: "B.md", Line: 1}},
	}
	stem := &rules.StemFile{
		Links: rules.LinkSchema{
			Rules: map[string]rules.LinkRule{
				"blocks": {Target: "*.md"}, // no Field
			},
		},
	}
	resolver := NewMapResolver([]*extract.Record{source, target})

	env := BuildEnv(source.Frontmatter)
	InjectLinkedFields(env, source, stem, resolver)

	if _, ok := env["blocked_by"]; ok {
		t.Error("expected no injection when rule has no Field")
	}
}

func TestDeriveRecord_WithResolver(t *testing.T) {
	// Integration test: DeriveRecord with a resolver injects linked values
	// that can be used in derive expressions.
	target := &extract.Record{
		Path:        "tasks/B.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"estado": "Completed"},
	}
	source := &extract.Record{
		Path:        "tasks/A.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"titulo": "Task A"},
		Links:       []extract.Link{{Type: "blocks", Target: "B.md", Line: 1}},
	}
	stem := &rules.StemFile{
		Derive: map[string]any{
			"status": `blocked_by[0]`,
		},
		Links: rules.LinkSchema{
			Rules: map[string]rules.LinkRule{
				"blocks": {Target: "*.md", Field: "blocked_by"},
			},
		},
	}
	resolver := NewMapResolver([]*extract.Record{source, target})

	derived, err := DeriveRecord(source, stem, nil, WithResolver(resolver))
	if err != nil {
		t.Fatalf("DeriveRecord: %v", err)
	}
	if derived["status"] != "Completed" {
		t.Errorf("status = %v, want %q", derived["status"], "Completed")
	}
}

func TestMapResolver_ExactPath(t *testing.T) {
	r := &extract.Record{Path: "tasks/B.md", Frontmatter: map[string]any{}}
	resolver := NewMapResolver([]*extract.Record{r})

	if got := resolver.Resolve("tasks/B.md"); got != r {
		t.Error("expected exact path match")
	}
}

func TestMapResolver_BasenameFallback(t *testing.T) {
	r := &extract.Record{Path: "tasks/B.md", Frontmatter: map[string]any{}}
	resolver := NewMapResolver([]*extract.Record{r})

	if got := resolver.Resolve("B.md"); got != r {
		t.Error("expected basename match for B.md")
	}
	if got := resolver.Resolve("B"); got != r {
		t.Error("expected basename match for B (without .md)")
	}
}

func TestMapResolver_AmbiguousBasename(t *testing.T) {
	r1 := &extract.Record{Path: "a/B.md", Frontmatter: map[string]any{}}
	r2 := &extract.Record{Path: "b/B.md", Frontmatter: map[string]any{}}
	resolver := NewMapResolver([]*extract.Record{r1, r2})

	if got := resolver.Resolve("B.md"); got != nil {
		t.Error("expected nil for ambiguous basename")
	}
}

func TestMapResolver_NotFound(t *testing.T) {
	resolver := NewMapResolver([]*extract.Record{})

	if got := resolver.Resolve("nonexistent.md"); got != nil {
		t.Error("expected nil for non-existent path")
	}
}

func TestLinkedValues_AllPattern(t *testing.T) {
	// all(blocked_by, {# == "Completed"}) returns true when all blockers are Completed.
	targetB := &extract.Record{
		Path:        "tasks/B.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"estado": "Completed"},
	}
	targetC := &extract.Record{
		Path:        "tasks/C.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"estado": "Completed"},
	}
	source := &extract.Record{
		Path:        "tasks/A.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"titulo": "Task A"},
		Links: []extract.Link{
			{Type: "blocks", Target: "B.md", Line: 1},
			{Type: "blocks", Target: "C.md", Line: 2},
		},
	}
	stem := &rules.StemFile{
		Derive: map[string]any{
			"all_done": `all(blocked_by, {# == "Completed"})`,
		},
		Links: rules.LinkSchema{
			Rules: map[string]rules.LinkRule{
				"blocks": {Target: "*.md", Field: "blocked_by"},
			},
		},
	}
	resolver := NewMapResolver([]*extract.Record{source, targetB, targetC})

	derived, err := DeriveRecord(source, stem, nil, WithResolver(resolver))
	if err != nil {
		t.Fatalf("DeriveRecord: %v", err)
	}
	if derived["all_done"] != true {
		t.Errorf("all_done = %v, want true", derived["all_done"])
	}

	// Now test with mixed states — all() should return false.
	targetC.Frontmatter["estado"] = "Pending"
	source2 := &extract.Record{
		Path:        "tasks/A2.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"titulo": "Task A2"},
		Links: []extract.Link{
			{Type: "blocks", Target: "B.md", Line: 1},
			{Type: "blocks", Target: "C.md", Line: 2},
		},
	}
	resolver2 := NewMapResolver([]*extract.Record{source2, targetB, targetC})
	derived2, err := DeriveRecord(source2, stem, nil, WithResolver(resolver2))
	if err != nil {
		t.Fatalf("DeriveRecord: %v", err)
	}
	if derived2["all_done"] != false {
		t.Errorf("all_done = %v, want false (mixed states)", derived2["all_done"])
	}
}

func TestLinkedValues_AnyPattern(t *testing.T) {
	// any(blocked_by, {# == "Blocked"}) returns true if any blocker is Blocked.
	targetB := &extract.Record{
		Path:        "tasks/B.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"estado": "Completed"},
	}
	targetC := &extract.Record{
		Path:        "tasks/C.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"estado": "Blocked"},
	}
	source := &extract.Record{
		Path:        "tasks/A.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"titulo": "Task A"},
		Links: []extract.Link{
			{Type: "blocks", Target: "B.md", Line: 1},
			{Type: "blocks", Target: "C.md", Line: 2},
		},
	}
	stem := &rules.StemFile{
		Derive: map[string]any{
			"has_blocked": `any(blocked_by, {# == "Blocked"})`,
		},
		Links: rules.LinkSchema{
			Rules: map[string]rules.LinkRule{
				"blocks": {Target: "*.md", Field: "blocked_by"},
			},
		},
	}
	resolver := NewMapResolver([]*extract.Record{source, targetB, targetC})

	derived, err := DeriveRecord(source, stem, nil, WithResolver(resolver))
	if err != nil {
		t.Fatalf("DeriveRecord: %v", err)
	}
	if derived["has_blocked"] != true {
		t.Errorf("has_blocked = %v, want true", derived["has_blocked"])
	}

	// With no Blocked states, any() should return false.
	targetC.Frontmatter["estado"] = "Completed"
	source2 := &extract.Record{
		Path:        "tasks/A2.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"titulo": "Task A2"},
		Links: []extract.Link{
			{Type: "blocks", Target: "B.md", Line: 1},
			{Type: "blocks", Target: "C.md", Line: 2},
		},
	}
	resolver2 := NewMapResolver([]*extract.Record{source2, targetB, targetC})
	derived2, err := DeriveRecord(source2, stem, nil, WithResolver(resolver2))
	if err != nil {
		t.Fatalf("DeriveRecord: %v", err)
	}
	if derived2["has_blocked"] != false {
		t.Errorf("has_blocked = %v, want false (no Blocked)", derived2["has_blocked"])
	}
}

func TestLinkedValues_NilCheck(t *testing.T) {
	// blocked_by == nil returns true when there are no links of type blocks.
	source := &extract.Record{
		Path:        "tasks/A.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"titulo": "Task A"},
	}
	stem := &rules.StemFile{
		Derive: map[string]any{
			"is_unblocked": `blocked_by == nil`,
		},
		Links: rules.LinkSchema{
			Rules: map[string]rules.LinkRule{
				"blocks": {Target: "*.md", Field: "blocked_by"},
			},
		},
	}
	resolver := NewMapResolver([]*extract.Record{source})

	derived, err := DeriveRecord(source, stem, nil, WithResolver(resolver))
	if err != nil {
		t.Fatalf("DeriveRecord: %v", err)
	}
	if derived["is_unblocked"] != true {
		t.Errorf("is_unblocked = %v, want true (no links)", derived["is_unblocked"])
	}

	// With links present, blocked_by != nil.
	target := &extract.Record{
		Path:        "tasks/B.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"estado": "Completed"},
	}
	sourceWithLinks := &extract.Record{
		Path:        "tasks/A2.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"titulo": "Task A2"},
		Links:       []extract.Link{{Type: "blocks", Target: "B.md", Line: 1}},
	}
	resolver2 := NewMapResolver([]*extract.Record{sourceWithLinks, target})
	derived2, err := DeriveRecord(sourceWithLinks, stem, nil, WithResolver(resolver2))
	if err != nil {
		t.Fatalf("DeriveRecord: %v", err)
	}
	if derived2["is_unblocked"] != false {
		t.Errorf("is_unblocked = %v, want false (has links)", derived2["is_unblocked"])
	}
}

func TestLinkedValues_LenPattern(t *testing.T) {
	// len(blocked_by) returns the correct number of links.
	targetB := &extract.Record{
		Path:        "tasks/B.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"estado": "Completed"},
	}
	targetC := &extract.Record{
		Path:        "tasks/C.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"estado": "Pending"},
	}
	targetD := &extract.Record{
		Path:        "tasks/D.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"estado": "Blocked"},
	}
	source := &extract.Record{
		Path:        "tasks/A.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"titulo": "Task A"},
		Links: []extract.Link{
			{Type: "blocks", Target: "B.md", Line: 1},
			{Type: "blocks", Target: "C.md", Line: 2},
			{Type: "blocks", Target: "D.md", Line: 3},
		},
	}
	stem := &rules.StemFile{
		Derive: map[string]any{
			"blocker_count": `len(blocked_by)`,
		},
		Links: rules.LinkSchema{
			Rules: map[string]rules.LinkRule{
				"blocks": {Target: "*.md", Field: "blocked_by"},
			},
		},
	}
	resolver := NewMapResolver([]*extract.Record{source, targetB, targetC, targetD})

	derived, err := DeriveRecord(source, stem, nil, WithResolver(resolver))
	if err != nil {
		t.Fatalf("DeriveRecord: %v", err)
	}
	if derived["blocker_count"] != 3 {
		t.Errorf("blocker_count = %v, want 3", derived["blocker_count"])
	}
}

func TestResolveTarget(t *testing.T) {
	tests := []struct {
		source, target, want string
	}{
		{"tasks/A.md", "B.md", "B.md"},                // no path separator
		{"tasks/A.md", "../other/B.md", "other/B.md"}, // relative path
		{"tasks/A.md", "sub/B.md", "tasks/sub/B.md"},  // path with separator
		{"A.md", "B.md", "B.md"},                      // root-level
	}
	for _, tt := range tests {
		got := resolveTarget(tt.source, tt.target)
		if got != tt.want {
			t.Errorf("resolveTarget(%q, %q) = %q, want %q", tt.source, tt.target, got, tt.want)
		}
	}
}
