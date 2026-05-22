package fix

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/picokit/fuzzy"
	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/proposal"
	"github.com/pablontiv/rootline/internal/rules"
	"gopkg.in/yaml.v3"
)

// --- levenshtein (via fuzzy package) ---

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"kitten", "sitting", 3},
		{"Completed", "Completd", 1},
		{"same", "same", 0},
	}
	for _, tt := range tests {
		got := fuzzy.Distance(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("fuzzy.Distance(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

// --- ClosestMatch ---

func TestClosestMatch(t *testing.T) {
	candidates := []string{"Pending", "Completed"}
	got := ClosestMatch("Completd", candidates)
	if got != "Completed" {
		t.Errorf("ClosestMatch('Completd') = %q, want 'Completed'", got)
	}
}

func TestClosestMatch_EmptyCandidates(t *testing.T) {
	got := ClosestMatch("anything", nil)
	if got != "" {
		t.Errorf("expected empty string for empty candidates, got %q", got)
	}
}

func TestClosestMatch_ExactMatch(t *testing.T) {
	candidates := []string{"Pending", "Completed"}
	got := ClosestMatch("Pending", candidates)
	if got != "Pending" {
		t.Errorf("ClosestMatch('Pending') = %q, want 'Pending'", got)
	}
}

func TestClosestMatch_CaseInsensitive(t *testing.T) {
	candidates := []string{"Pending", "Completed"}
	got := ClosestMatch("pending", candidates)
	if got != "Pending" {
		t.Errorf("ClosestMatch('pending') = %q, want 'Pending'", got)
	}
}

// --- RewriteFrontmatter ---

func TestRewriteFrontmatter_NoPrior(t *testing.T) {
	original := "# Just a heading\nSome body.\n"
	fm := map[string]any{"estado": "Pending"}
	result := RewriteFrontmatter(original, fm)
	if !strings.HasPrefix(result, "---\n") {
		t.Error("expected frontmatter prepended")
	}
	if !strings.Contains(result, "estado: Pending") {
		t.Error("expected estado field in new frontmatter")
	}
	if !strings.Contains(result, "Just a heading") {
		t.Error("expected original body preserved")
	}
}

func TestRewriteFrontmatter_Malformed(t *testing.T) {
	original := "---\nestado: test\n# No closing\n"
	fm := map[string]any{"estado": "Pending"}
	result := RewriteFrontmatter(original, fm)
	if result != original {
		t.Errorf("expected original returned for malformed frontmatter, got: %s", result)
	}
}

func TestRewriteFrontmatter_ExistingFrontmatter(t *testing.T) {
	original := "---\nestado: old\n---\n# Title\n\nBody text.\n"
	fm := map[string]any{"estado": "Completed"}
	result := RewriteFrontmatter(original, fm)
	if !strings.Contains(result, "estado: Completed") {
		t.Errorf("expected updated frontmatter, got: %s", result)
	}
	if !strings.Contains(result, "Body text.") {
		t.Error("expected body preserved")
	}
	if strings.Contains(result, "old") {
		t.Error("expected old value replaced")
	}
}

func TestRewriteFrontmatter_PreservesBody(t *testing.T) {
	original := "---\ntipo: test\n---\n# Important\n\nBody here.\n"
	fm := map[string]any{"tipo": "test", "estado": "Pending"}
	result := RewriteFrontmatter(original, fm)
	if !strings.Contains(result, "Important") {
		t.Error("expected body heading preserved")
	}
	if !strings.Contains(result, "Body here.") {
		t.Error("expected body text preserved")
	}
}

// --- WriteFrontmatterFields ---

func TestWriteFrontmatterFields_YAMLQuoting(t *testing.T) {
	tests := []struct {
		name string
		fm   map[string]any
		want string
	}{
		{
			name: "plain string",
			fm:   map[string]any{"title": "Hello"},
			want: "title: Hello\n",
		},
		{
			name: "string with colon is quoted",
			fm:   map[string]any{"title": "Hello: World"},
			want: "'Hello: World'",
		},
		{
			name: "integer value",
			fm:   map[string]any{"count": 42},
			want: "count: 42\n",
		},
		{
			name: "boolean value",
			fm:   map[string]any{"draft": true},
			want: "draft: true\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b strings.Builder
			WriteFrontmatterFields(&b, tt.fm)
			out := b.String()
			if !strings.Contains(out, tt.want) {
				t.Errorf("output = %q\nwant fragment %q", out, tt.want)
			}
			var parsed map[string]any
			if err := yaml.Unmarshal([]byte(out), &parsed); err != nil {
				t.Errorf("output is not valid YAML: %v\noutput: %q", err, out)
			}
		})
	}
}

func TestWriteFrontmatterFields_SliceValue(t *testing.T) {
	fm := map[string]any{"tags": []any{"alpha", "beta"}}
	var b strings.Builder
	WriteFrontmatterFields(&b, fm)
	out := b.String()

	var parsed map[string]any
	if err := yaml.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid YAML: %v\noutput: %q", err, out)
	}
	tags, ok := parsed["tags"].([]any)
	if !ok {
		t.Fatalf("expected tags to be a list, got %T: %v", parsed["tags"], parsed["tags"])
	}
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
}

func TestWriteFrontmatterFields_SortedKeys(t *testing.T) {
	fm := map[string]any{"z_field": "z", "a_field": "a", "m_field": "m"}
	var b strings.Builder
	WriteFrontmatterFields(&b, fm)
	out := b.String()
	aIdx := strings.Index(out, "a_field")
	mIdx := strings.Index(out, "m_field")
	zIdx := strings.Index(out, "z_field")
	if aIdx >= mIdx || mIdx >= zIdx {
		t.Errorf("expected sorted order, got: %s", out)
	}
}

func TestWriteFrontmatterFields_Empty(t *testing.T) {
	var b strings.Builder
	WriteFrontmatterFields(&b, map[string]any{})
	if b.String() != "" {
		t.Errorf("expected empty output for empty map, got: %q", b.String())
	}
}

// --- InsertWikiLinksBeforeHeading ---

func TestInsertWikiLinksBeforeHeading(t *testing.T) {
	content := "---\nestado: Pending\n---\n# Title\n\n## Context\n\nSome text\n"
	links := []string{"[[blocks:T001]]", "[[blocks:T002]]"}
	result := InsertWikiLinksBeforeHeading(content, links)
	if !strings.Contains(result, "[[blocks:T001]]\n[[blocks:T002]]") {
		t.Errorf("expected wiki-links inserted, got: %s", result)
	}
	if !strings.Contains(result, "## Context") {
		t.Error("expected heading preserved")
	}
}

func TestInsertWikiLinksNoHeading(t *testing.T) {
	content := "---\nestado: Pending\n---\n# Title\n\nNo sub-headings here.\n"
	links := []string{"[[blocks:T001]]"}
	result := InsertWikiLinksBeforeHeading(content, links)
	if !strings.Contains(result, "[[blocks:T001]]") {
		t.Errorf("expected wiki-link appended, got: %s", result)
	}
}

// --- ApplyFixes ---

func TestApplyFixes_MissingRequired(t *testing.T) {
	record := &extract.Record{
		Path:        "test.md",
		Frontmatter: map[string]any{"tipo": "test"},
	}
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Required: true, Values: []string{"Pending", "Completed"}},
		},
	}
	errs := []rules.ValidationError{
		{Rule: "required", Field: "estado", Message: "field estado is required"},
	}

	added, corrected := ApplyFixes(context.Background(), record, effective, errs)
	if len(added) != 1 {
		t.Errorf("expected 1 field added, got %d", len(added))
	}
	if len(corrected) != 0 {
		t.Errorf("expected 0 corrections, got %d", len(corrected))
	}
	if record.Frontmatter["estado"] != "Pending" {
		t.Errorf("expected estado=Pending, got %v", record.Frontmatter["estado"])
	}
}

func TestApplyFixes_InvalidEnum(t *testing.T) {
	record := &extract.Record{
		Path:        "test.md",
		Frontmatter: map[string]any{"estado": "Completd"},
	}
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Required: true, Values: []string{"Pending", "Completed"}},
		},
	}
	errs := []rules.ValidationError{
		{Rule: "enum", Field: "estado", Message: `value "Completd" not in allowed values`},
	}

	added, corrected := ApplyFixes(context.Background(), record, effective, errs)
	if len(added) != 0 {
		t.Errorf("expected 0 added, got %d", len(added))
	}
	if len(corrected) != 1 {
		t.Errorf("expected 1 correction, got %d", len(corrected))
	}
	if record.Frontmatter["estado"] != "Completed" {
		t.Errorf("expected estado=Completed, got %v", record.Frontmatter["estado"])
	}
}

func TestApplyFixes_NilEffective(t *testing.T) {
	record := &extract.Record{
		Path:        "test.md",
		Frontmatter: map[string]any{},
	}
	errs := []rules.ValidationError{
		{Rule: "required", Field: "estado"},
	}
	added, corrected := ApplyFixes(context.Background(), record, nil, errs)
	if len(added) != 0 || len(corrected) != 0 {
		t.Error("expected no changes with nil effective")
	}
}

func TestApplyFixes_UnknownField(t *testing.T) {
	record := &extract.Record{
		Path:        "test.md",
		Frontmatter: map[string]any{},
	}
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{},
	}
	errs := []rules.ValidationError{
		{Rule: "required", Field: "unknown_field"},
	}
	added, corrected := ApplyFixes(context.Background(), record, effective, errs)
	if len(added) != 0 || len(corrected) != 0 {
		t.Error("expected no changes for unknown field")
	}
}

func TestApplyFixes_RequiredWithDefault(t *testing.T) {
	record := &extract.Record{
		Path:        "test.md",
		Frontmatter: map[string]any{},
	}
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "string", Required: true, Default: "Draft"},
		},
	}
	errs := []rules.ValidationError{
		{Rule: "required", Field: "estado"},
	}
	added, _ := ApplyFixes(context.Background(), record, effective, errs)
	if len(added) != 1 {
		t.Fatalf("expected 1 added, got %d", len(added))
	}
	if record.Frontmatter["estado"] != "Draft" {
		t.Errorf("expected estado=Draft, got %v", record.Frontmatter["estado"])
	}
}

// --- applyCorrectLink ---

func TestApplyCorrectLink_Unit(t *testing.T) {
	dir := t.TempDir()

	mustWriteFile(t, filepath.Join(dir, "T001-task.md"),
		[]byte("---\nestado: Pending\n---\n# Task\n\n[[blocks:E04]]\n"))

	p := proposal.Proposal{
		Type:  proposal.CorrectLink,
		Field: "links",
		From:  "[[blocks:E04]]",
		To:    "[[reference:E04]]",
		Paths: []string{"T001-task.md"},
	}

	err := applyCorrectLink(p, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "T001-task.md"))
	contentStr := string(content)
	if !strings.Contains(contentStr, "[[reference:E04]]") {
		t.Errorf("expected [[reference:E04]], got:\n%s", contentStr)
	}
	if strings.Contains(contentStr, "[[blocks:E04]]") {
		t.Errorf("expected [[blocks:E04]] replaced")
	}
}

func TestApplyCorrectLink_Expand(t *testing.T) {
	dir := t.TempDir()

	mustWriteFile(t, filepath.Join(dir, "T002-validate.md"),
		[]byte("---\nestado: Pending\n---\n# Validate\n\n[[blocks:T001]]\n"))

	p := proposal.Proposal{
		Type:  proposal.CorrectLink,
		Field: "links",
		From:  "[[blocks:T001]]",
		To:    "[[blocks:T001-add-feature]]",
		Paths: []string{"T002-validate.md"},
	}

	err := applyCorrectLink(p, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "T002-validate.md"))
	contentStr := string(content)
	if !strings.Contains(contentStr, "[[blocks:T001-add-feature]]") {
		t.Errorf("expected expanded link, got:\n%s", contentStr)
	}
}

func TestApplyCorrectLink_FileNotFound(t *testing.T) {
	dir := t.TempDir()

	p := proposal.Proposal{
		Type:  proposal.CorrectLink,
		From:  "[[blocks:E04]]",
		To:    "[[reference:E04]]",
		Paths: []string{"nonexistent.md"},
	}

	err := applyCorrectLink(p, dir)
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

// --- applyMigrateValue ---

func TestApplyMigrateValue_Unit(t *testing.T) {
	dir := t.TempDir()

	mustWriteFile(t, filepath.Join(dir, "task.md"),
		[]byte("---\nestado: \"Pending (blocked by T001)\"\n---\n# Task\n\n## Context\n"))

	recordMap := map[string]*extract.Record{
		"task.md": {
			Path:        "task.md",
			Frontmatter: map[string]any{"estado": "Pending (blocked by T001)"},
		},
	}

	p := proposal.Proposal{
		Type:      proposal.MigrateValue,
		Field:     "estado",
		From:      "Pending (blocked by T001)",
		To:        "Pending",
		WikiLinks: []string{"[[blocks:T001]]"},
		Paths:     []string{"task.md"},
	}

	err := applyMigrateValue(p, dir, recordMap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "task.md"))
	contentStr := string(content)
	if !strings.Contains(contentStr, "estado: Pending") {
		t.Errorf("expected estado: Pending, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "[[blocks:T001]]") {
		t.Errorf("expected wiki-link inserted, got:\n%s", contentStr)
	}
}

func TestApplyMigrateValue_NoWikiLinks(t *testing.T) {
	dir := t.TempDir()

	mustWriteFile(t, filepath.Join(dir, "task.md"),
		[]byte("---\nestado: \"Pending (blocked by human)\"\n---\n# Task\n"))

	recordMap := map[string]*extract.Record{
		"task.md": {
			Path:        "task.md",
			Frontmatter: map[string]any{"estado": "Pending (blocked by human)"},
		},
	}

	p := proposal.Proposal{
		Type:  proposal.MigrateValue,
		Field: "estado",
		From:  "Pending (blocked by human)",
		To:    "Pending",
		Paths: []string{"task.md"},
	}

	err := applyMigrateValue(p, dir, recordMap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "task.md"))
	if !strings.Contains(string(content), "estado: Pending") {
		t.Errorf("expected estado: Pending, got:\n%s", string(content))
	}
}

func TestApplyMigrateValue_RecordNotInMap(t *testing.T) {
	dir := t.TempDir()
	recordMap := map[string]*extract.Record{}

	p := proposal.Proposal{
		Type:  proposal.MigrateValue,
		Field: "estado",
		Paths: []string{"missing.md"},
	}

	err := applyMigrateValue(p, dir, recordMap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- applyCorrectValue ---

func TestApplyCorrectValue_Unit(t *testing.T) {
	dir := t.TempDir()

	mustWriteFile(t, filepath.Join(dir, "task.md"),
		[]byte("---\nestado: Completd\n---\n# Task\n"))

	recordMap := map[string]*extract.Record{
		"task.md": {
			Path:        "task.md",
			Frontmatter: map[string]any{"estado": "Completd"},
		},
	}

	p := proposal.Proposal{
		Type:  proposal.CorrectValue,
		Field: "estado",
		From:  "Completd",
		To:    "Completed",
		Paths: []string{"task.md"},
	}

	err := applyCorrectValue(p, dir, recordMap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "task.md"))
	if !strings.Contains(string(content), "estado: Completed") {
		t.Errorf("expected estado: Completed, got:\n%s", string(content))
	}
}

// --- applySetField ---

func TestApplySetField_WithValue(t *testing.T) {
	dir := t.TempDir()

	mustWriteFile(t, filepath.Join(dir, "task.md"),
		[]byte("---\ntipo: test\n---\n# Task\n"))

	recordMap := map[string]*extract.Record{
		"task.md": {
			Path:        "task.md",
			Frontmatter: map[string]any{"tipo": "test"},
		},
	}

	p := proposal.Proposal{
		Type:  proposal.AddField,
		Field: "estado",
		Value: "Pending",
		Paths: []string{"task.md"},
	}

	err := applySetField(p, dir, recordMap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "task.md"))
	if !strings.Contains(string(content), "estado: Pending") {
		t.Errorf("expected estado: Pending, got:\n%s", string(content))
	}
}

func TestApplySetField_UsesToWhenValueEmpty(t *testing.T) {
	dir := t.TempDir()

	mustWriteFile(t, filepath.Join(dir, "task.md"),
		[]byte("---\ntipo: test\n---\n# Task\n"))

	recordMap := map[string]*extract.Record{
		"task.md": {
			Path:        "task.md",
			Frontmatter: map[string]any{"tipo": "test"},
		},
	}

	p := proposal.Proposal{
		Type:  proposal.ExtractBody,
		Field: "estado",
		To:    "Completed",
		Paths: []string{"task.md"},
	}

	err := applySetField(p, dir, recordMap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "task.md"))
	if !strings.Contains(string(content), "estado: Completed") {
		t.Errorf("expected estado: Completed, got:\n%s", string(content))
	}
}

// --- rewriteRecordFile ---

func TestRewriteRecordFile(t *testing.T) {
	dir := t.TempDir()

	mustWriteFile(t, filepath.Join(dir, "task.md"),
		[]byte("---\nestado: old\n---\n# Task\n"))

	fm := map[string]any{"estado": "new"}
	err := rewriteRecordFile(dir, "task.md", fm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "task.md"))
	if !strings.Contains(string(content), "estado: new") {
		t.Errorf("expected estado: new, got:\n%s", string(content))
	}
}

// --- addEnumValueToNode ---

func TestAddEnumValueToNode(t *testing.T) {
	stemYAML := `version: 2
schema:
  estado:
    type: enum
    values:
      - Pending
      - Completed
`
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(stemYAML), &doc); err != nil {
		t.Fatal(err)
	}

	err := addEnumValueToNode(doc.Content[0], "estado", "Obsoleto")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out, _ := yaml.Marshal(&doc)
	if !strings.Contains(string(out), "Obsoleto") {
		t.Errorf("expected Obsoleto in output, got:\n%s", string(out))
	}
}

func TestAddEnumValueToNode_FieldNotFound(t *testing.T) {
	stemYAML := `version: 2
schema:
  estado:
    type: enum
    values: [Pending]
`
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(stemYAML), &doc); err != nil {
		t.Fatal(err)
	}

	err := addEnumValueToNode(doc.Content[0], "nonexistent", "value")
	if err == nil {
		t.Error("expected error for nonexistent field")
	}
}

func TestAddEnumValueToNode_NotMapping(t *testing.T) {
	node := &yaml.Node{Kind: yaml.ScalarNode, Value: "scalar"}
	err := addEnumValueToNode(node, "estado", "value")
	if err == nil {
		t.Error("expected error for non-mapping node")
	}
}

// --- applyExtendEnum ---

func setupStemDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	stemContent := `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values:
      - Pending
      - Completed
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent))
	return dir
}

func TestApplyExtendEnum(t *testing.T) {
	dir := setupStemDir(t)

	p := proposal.Proposal{
		Type:  proposal.ExtendEnum,
		Field: "estado",
		Value: "Obsoleto",
	}

	err := applyExtendEnum(p, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, ".stem"))
	if !strings.Contains(string(content), "Obsoleto") {
		t.Errorf("expected Obsoleto in .stem, got:\n%s", string(content))
	}
}

func TestApplyExtendEnum_NoStemFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	p := proposal.Proposal{
		Type:  proposal.ExtendEnum,
		Field: "estado",
		Value: "Test",
	}

	err := applyExtendEnum(p, dir)
	if err == nil {
		t.Error("expected error when .stem file doesn't exist")
	}
}

// --- ApplyProposals ---

func TestApplyProposals_ExtendEnum(t *testing.T) {
	dir := setupStemDir(t)

	mustWriteFile(t, filepath.Join(dir, "a.md"),
		[]byte("---\nestado: Obsoleto\ntipo: test\n---\n# A\n"))
	mustWriteFile(t, filepath.Join(dir, "b.md"),
		[]byte("---\nestado: Obsoleto\ntipo: test\n---\n# B\n"))

	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"estado": "Obsoleto", "tipo": "test"}},
		{Path: "b.md", Frontmatter: map[string]any{"estado": "Obsoleto", "tipo": "test"}},
	}

	report := &proposal.Report{
		Version: 1,
		Kind:    "rootline/proposals",
		Proposals: []proposal.Proposal{
			{
				Type:  proposal.ExtendEnum,
				Field: "estado",
				Value: "Obsoleto",
				Paths: []string{"a.md", "b.md"},
			},
		},
	}

	err := ApplyProposals(context.Background(), report, dir, records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check .stem was NOT updated — extend_enum is a schema proposal and is now skipped.
	content, _ := os.ReadFile(filepath.Join(dir, ".stem"))
	if strings.Contains(string(content), "Obsoleto") {
		t.Errorf("extend_enum should NOT have been applied (schema proposal), got:\n%s", string(content))
	}

	// Report should have no applied proposals (schema proposals are skipped).
	if len(report.Proposals) != 0 {
		t.Errorf("expected 0 applied proposals (extend_enum is schema surface), got %d", len(report.Proposals))
	}
}

func TestApplyProposals_CorrectValue(t *testing.T) {
	dir := setupStemDir(t)

	mustWriteFile(t, filepath.Join(dir, "task.md"),
		[]byte("---\nestado: Completd\ntipo: test\n---\n# Task\n"))

	records := []*extract.Record{
		{Path: "task.md", Frontmatter: map[string]any{"estado": "Completd", "tipo": "test"}},
	}

	report := &proposal.Report{
		Version: 1,
		Kind:    "rootline/proposals",
		Proposals: []proposal.Proposal{
			{
				Type:  proposal.CorrectValue,
				Field: "estado",
				From:  "Completd",
				To:    "Completed",
				Paths: []string{"task.md"},
			},
		},
	}

	err := ApplyProposals(context.Background(), report, dir, records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "task.md"))
	if !strings.Contains(string(content), "estado: Completed") {
		t.Errorf("expected estado: Completed, got:\n%s", string(content))
	}
}

func TestApplyProposals_SkipCorrectValueAfterExtendEnum(t *testing.T) {
	dir := setupStemDir(t)

	mustWriteFile(t, filepath.Join(dir, "a.md"),
		[]byte("---\nestado: Obsoleto\ntipo: test\n---\n# A\n"))

	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"estado": "Obsoleto", "tipo": "test"}},
	}

	report := &proposal.Report{
		Version: 1,
		Kind:    "rootline/proposals",
		Proposals: []proposal.Proposal{
			{
				Type:  proposal.CorrectValue,
				Field: "estado",
				From:  "Obsoleto",
				To:    "Completed",
				Paths: []string{"a.md"},
			},
		},
	}

	err := ApplyProposals(context.Background(), report, dir, records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// CorrectValue should be applied (it's a repair proposal).
	if len(report.Proposals) != 1 {
		t.Errorf("expected 1 applied proposal, got %d", len(report.Proposals))
	}
	if report.Proposals[0].Type != proposal.CorrectValue {
		t.Errorf("expected correct_value proposal, got %s", report.Proposals[0].Type)
	}

	// File should have been corrected to Completed.
	content, _ := os.ReadFile(filepath.Join(dir, "a.md"))
	if !strings.Contains(string(content), "Completed") {
		t.Error("file should have been corrected to Completed")
	}
}

func TestApplyProposals_AddField(t *testing.T) {
	dir := setupStemDir(t)

	mustWriteFile(t, filepath.Join(dir, "task.md"),
		[]byte("---\ntipo: test\n---\n# Task\n"))

	records := []*extract.Record{
		{Path: "task.md", Frontmatter: map[string]any{"tipo": "test"}},
	}

	report := &proposal.Report{
		Version: 1,
		Kind:    "rootline/proposals",
		Proposals: []proposal.Proposal{
			{
				Type:  proposal.AddField,
				Field: "estado",
				Value: "Pending",
				Paths: []string{"task.md"},
			},
		},
	}

	err := ApplyProposals(context.Background(), report, dir, records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "task.md"))
	if !strings.Contains(string(content), "estado: Pending") {
		t.Errorf("expected estado: Pending, got:\n%s", string(content))
	}
}

func TestApplyProposals_MigrateValue(t *testing.T) {
	dir := setupStemDir(t)

	mustWriteFile(t, filepath.Join(dir, "task.md"),
		[]byte("---\nestado: \"Pending (blocked by T001)\"\n---\n# Task\n\n## Context\n"))

	records := []*extract.Record{
		{Path: "task.md", Frontmatter: map[string]any{"estado": "Pending (blocked by T001)"}},
	}

	report := &proposal.Report{
		Version: 1,
		Kind:    "rootline/proposals",
		Proposals: []proposal.Proposal{
			{
				Type:      proposal.MigrateValue,
				Field:     "estado",
				From:      "Pending (blocked by T001)",
				To:        "Pending",
				WikiLinks: []string{"[[blocks:T001]]"},
				Paths:     []string{"task.md"},
			},
		},
	}

	err := ApplyProposals(context.Background(), report, dir, records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "task.md"))
	contentStr := string(content)
	if !strings.Contains(contentStr, "estado: Pending") {
		t.Errorf("expected estado: Pending, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "[[blocks:T001]]") {
		t.Errorf("expected wiki-link, got:\n%s", contentStr)
	}
}

func TestApplyProposals_CorrectLink(t *testing.T) {
	dir := setupStemDir(t)

	mustWriteFile(t, filepath.Join(dir, "task.md"),
		[]byte("---\nestado: Pending\n---\n# Task\n\n[[blocks:E04]]\n"))

	records := []*extract.Record{
		{Path: "task.md", Frontmatter: map[string]any{"estado": "Pending"}},
	}

	report := &proposal.Report{
		Version: 1,
		Kind:    "rootline/proposals",
		Proposals: []proposal.Proposal{
			{
				Type:  proposal.CorrectLink,
				Field: "links",
				From:  "[[blocks:E04]]",
				To:    "[[reference:E04]]",
				Paths: []string{"task.md"},
			},
		},
	}

	err := ApplyProposals(context.Background(), report, dir, records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "task.md"))
	if !strings.Contains(string(content), "[[reference:E04]]") {
		t.Errorf("expected corrected link, got:\n%s", string(content))
	}
}

func TestApplyProposals_EmptyReport(t *testing.T) {
	dir := setupStemDir(t)
	records := []*extract.Record{}
	report := &proposal.Report{
		Version:   1,
		Kind:      "rootline/proposals",
		Proposals: nil,
	}

	err := ApplyProposals(context.Background(), report, dir, records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplyProposals_ExtractBody(t *testing.T) {
	dir := setupStemDir(t)

	mustWriteFile(t, filepath.Join(dir, "task.md"),
		[]byte("---\ntipo: test\n---\n# Task\n"))

	records := []*extract.Record{
		{Path: "task.md", Frontmatter: map[string]any{"tipo": "test"}},
	}

	report := &proposal.Report{
		Version: 1,
		Kind:    "rootline/proposals",
		Proposals: []proposal.Proposal{
			{
				Type:  proposal.ExtractBody,
				Field: "estado",
				To:    "Completed",
				Paths: []string{"task.md"},
			},
		},
	}

	err := ApplyProposals(context.Background(), report, dir, records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "task.md"))
	if !strings.Contains(string(content), "estado: Completed") {
		t.Errorf("expected estado: Completed, got:\n%s", string(content))
	}
}

// --- removeStemSchemaField ---

func TestRemoveStemSchemaField_RemovesField(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	mustWriteFile(t, stemPath, []byte(`schema:
  id:
    type: sequence
    prefix: T
    digits: 3
  estado:
    type: enum
    required: true
    values: [Pending, Completed]
  tipo:
    type: enum
    values: [a, b]
`))

	err := removeStemSchemaField(stemPath, "estado")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(stemPath)
	if strings.Contains(string(content), "estado") {
		t.Errorf("expected estado to be removed, got:\n%s", string(content))
	}
	if !strings.Contains(string(content), "id") {
		t.Error("expected id to remain")
	}
	if !strings.Contains(string(content), "tipo") {
		t.Error("expected tipo to remain")
	}
}

func TestRemoveStemSchemaField_RemovesSchemaWhenEmpty(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	mustWriteFile(t, stemPath, []byte(`schema:
  estado:
    type: string
validate:
  - rule: non_empty
    field: estado
`))

	err := removeStemSchemaField(stemPath, "estado")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(stemPath)
	if strings.Contains(string(content), "schema") {
		t.Errorf("expected schema section to be removed, got:\n%s", string(content))
	}
	if !strings.Contains(string(content), "validate") {
		t.Error("expected validate section to remain")
	}
}

func TestRemoveStemSchemaField_ErrorOnMissingField(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	mustWriteFile(t, stemPath, []byte(`schema:
  id:
    type: string
`))

	err := removeStemSchemaField(stemPath, "nonexistent")
	if err == nil {
		t.Error("expected error for missing field")
	}
}

func TestApplyProposals_RemoveStemField(t *testing.T) {
	dir := setupStemDir(t)

	// Add a child .stem that overrides estado
	childDir := filepath.Join(dir, "sub")
	if err := os.MkdirAll(childDir, 0755); err != nil {
		t.Fatal(err)
	}
	childStem := filepath.Join(childDir, ".stem")
	mustWriteFile(t, childStem, []byte(`schema:
  estado:
    type: enum
    required: true
    values: [Pending, Completed]
  ejecutable_en:
    type: string
`))

	records := []*extract.Record{}
	report := &proposal.Report{
		Version: 1,
		Kind:    "rootline/proposals",
		Proposals: []proposal.Proposal{
			{
				Type:        proposal.RemoveStemField,
				Field:       "estado",
				Description: "remove redundant field",
				Paths:       []string{"sub/.stem"},
			},
		},
	}

	err := ApplyProposals(context.Background(), report, dir, records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// RemoveStemField is a schema proposal and should NOT be applied.
	content, _ := os.ReadFile(childStem)
	if !strings.Contains(string(content), "estado") {
		t.Errorf("expected estado to remain (remove_stem_field is schema proposal, not applied), got:\n%s", string(content))
	}
	if !strings.Contains(string(content), "ejecutable_en") {
		t.Error("expected ejecutable_en to remain")
	}

	// Report should have no applied proposals.
	if len(report.Proposals) != 0 {
		t.Errorf("expected 0 applied proposals, got %d", len(report.Proposals))
	}
}

// --- TestApplySetField ---

func TestApplySetField(t *testing.T) {
	dir := t.TempDir()
	relPath := "test.md"
	absPath := filepath.Join(dir, relPath)
	original := "---\nestado: bloqueado\nbloqueador: \"some reason\"\n---\n\n# Title\n"
	mustWriteFile(t, absPath, []byte(original))

	rec := &extract.Record{
		Path:        relPath,
		Frontmatter: map[string]any{"estado": "bloqueado", "bloqueador": "some reason"},
	}

	report := &proposal.Report{
		Version: 1,
		Kind:    "rootline/proposals",
		Proposals: []proposal.Proposal{
			{
				Type:  proposal.SetField,
				Field: "estado",
				Value: "listo-para-implementar",
				Paths: []string{relPath},
			},
			{
				Type:  proposal.SetField,
				Field: "bloqueador",
				Value: "",
				Paths: []string{relPath},
			},
		},
	}

	err := ApplyProposals(context.Background(), report, dir, []*extract.Record{rec})
	if err != nil {
		t.Fatalf("ApplyProposals: %v", err)
	}

	data, _ := os.ReadFile(absPath)
	content := string(data)
	if !strings.Contains(content, "estado: listo-para-implementar") {
		t.Errorf("estado not updated in:\n%s", content)
	}
}

// --- applySetSection ---

func TestApplySetSection_Replace(t *testing.T) {
	dir := t.TempDir()
	relPath := "task.md"
	absPath := filepath.Join(dir, relPath)

	original := "---\nestado: Pending\n---\n\n## Contexto\n\nOld content.\n\n## Desbloqueo\n\nStuff.\n"
	mustWriteFile(t, absPath, []byte(original))

	p := proposal.Proposal{
		Type:    proposal.SetSection,
		Heading: "## Contexto",
		Value:   "New content.",
		Mode:    "replace",
		Paths:   []string{relPath},
	}

	err := applySetSection(p, dir, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(absPath)
	content := string(data)

	if strings.Contains(content, "Old content.") {
		t.Errorf("expected old content removed, got:\n%s", content)
	}
	if !strings.Contains(content, "New content.") {
		t.Errorf("expected new content present, got:\n%s", content)
	}
	if !strings.Contains(content, "## Contexto") {
		t.Errorf("expected heading preserved, got:\n%s", content)
	}
	if !strings.Contains(content, "## Desbloqueo") {
		t.Errorf("expected other section preserved, got:\n%s", content)
	}
	if !strings.Contains(content, "Stuff.") {
		t.Errorf("expected other section content preserved, got:\n%s", content)
	}
}

func TestApplySetSection_Append(t *testing.T) {
	dir := t.TempDir()
	relPath := "task.md"
	absPath := filepath.Join(dir, relPath)

	original := "---\nestado: Pending\n---\n\n## Contexto\n\nExisting content.\n"
	mustWriteFile(t, absPath, []byte(original))

	p := proposal.Proposal{
		Type:    proposal.SetSection,
		Heading: "## Contexto",
		Value:   "Appended content.",
		Mode:    "append",
		Paths:   []string{relPath},
	}

	err := applySetSection(p, dir, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(absPath)
	content := string(data)

	if !strings.Contains(content, "Existing content.") {
		t.Errorf("expected existing content preserved, got:\n%s", content)
	}
	if !strings.Contains(content, "Appended content.") {
		t.Errorf("expected appended content present, got:\n%s", content)
	}
	if !strings.Contains(content, "## Contexto") {
		t.Errorf("expected heading preserved, got:\n%s", content)
	}
}

func TestApplySetSection_Create(t *testing.T) {
	dir := t.TempDir()
	relPath := "task.md"
	absPath := filepath.Join(dir, relPath)

	original := "---\nestado: Pending\n---\n\n## Contexto\n\nSome text.\n"
	mustWriteFile(t, absPath, []byte(original))

	p := proposal.Proposal{
		Type:    proposal.SetSection,
		Heading: "## Investigación",
		Value:   "Research notes.",
		Mode:    "create",
		Paths:   []string{relPath},
	}

	err := applySetSection(p, dir, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(absPath)
	content := string(data)

	if !strings.Contains(content, "## Investigación") {
		t.Errorf("expected new heading at EOF, got:\n%s", content)
	}
	if !strings.Contains(content, "Research notes.") {
		t.Errorf("expected new content at EOF, got:\n%s", content)
	}
	if !strings.Contains(content, "## Contexto") {
		t.Errorf("expected original section preserved, got:\n%s", content)
	}
	// New section should be at the end.
	contextoIdx := strings.Index(content, "## Contexto")
	investigacionIdx := strings.Index(content, "## Investigación")
	if investigacionIdx < contextoIdx {
		t.Errorf("expected new section after existing section, got:\n%s", content)
	}
}

// --- applySetSection error paths ---

func TestApplySetSection_UnknownMode(t *testing.T) {
	dir := t.TempDir()
	relPath := "task.md"
	absPath := filepath.Join(dir, relPath)

	mustWriteFile(t, absPath, []byte("---\nestado: Pending\n---\n\n## Contexto\n\nSome text.\n"))

	p := proposal.Proposal{
		Type:    proposal.SetSection,
		Heading: "## Contexto",
		Value:   "value",
		Mode:    "unknown_mode",
		Paths:   []string{relPath},
	}

	err := applySetSection(p, dir, nil)
	if err == nil {
		t.Fatal("expected error for unknown mode")
	}
	if !strings.Contains(err.Error(), "unknown mode") {
		t.Errorf("expected 'unknown mode' error, got: %v", err)
	}
}

func TestApplySetSection_HeadingNotFoundNoCreate(t *testing.T) {
	dir := t.TempDir()
	relPath := "task.md"
	absPath := filepath.Join(dir, relPath)

	mustWriteFile(t, absPath, []byte("---\nestado: Pending\n---\n\n# Title\n\nSome text.\n"))

	p := proposal.Proposal{
		Type:    proposal.SetSection,
		Heading: "## Nonexistent",
		Value:   "value",
		Mode:    "replace",
		Paths:   []string{relPath},
	}

	err := applySetSection(p, dir, nil)
	if err == nil {
		t.Fatal("expected error for heading not found in replace mode")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestApplySetSection_FileNotFound(t *testing.T) {
	dir := t.TempDir()

	p := proposal.Proposal{
		Type:    proposal.SetSection,
		Heading: "## Contexto",
		Value:   "value",
		Mode:    "replace",
		Paths:   []string{"nonexistent.md"},
	}

	err := applySetSection(p, dir, nil)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestApplySetSection_HeadingAtStart(t *testing.T) {
	// Test heading at very first line (no leading newline)
	dir := t.TempDir()
	relPath := "task.md"
	absPath := filepath.Join(dir, relPath)

	mustWriteFile(t, absPath, []byte("## Heading\n\nOriginal content.\n"))

	p := proposal.Proposal{
		Type:    proposal.SetSection,
		Heading: "## Heading",
		Value:   "New content.",
		Mode:    "replace",
		Paths:   []string{relPath},
	}

	err := applySetSection(p, dir, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(absPath)
	content := string(data)
	if !strings.Contains(content, "New content.") {
		t.Errorf("expected new content, got:\n%s", content)
	}
	if strings.Contains(content, "Original content.") {
		t.Errorf("expected old content replaced, got:\n%s", content)
	}
}

func TestApplySetSection_CreateWhenFoundActsLikeAppend(t *testing.T) {
	// When mode=create and heading is found, it should act like append
	dir := t.TempDir()
	relPath := "task.md"
	absPath := filepath.Join(dir, relPath)

	mustWriteFile(t, absPath, []byte("---\nestado: Pending\n---\n\n## Contexto\n\nExisting content.\n"))

	p := proposal.Proposal{
		Type:    proposal.SetSection,
		Heading: "## Contexto",
		Value:   "Additional content.",
		Mode:    "create",
		Paths:   []string{relPath},
	}

	err := applySetSection(p, dir, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(absPath)
	content := string(data)
	if !strings.Contains(content, "Existing content.") {
		t.Errorf("expected original content preserved, got:\n%s", content)
	}
	if !strings.Contains(content, "Additional content.") {
		t.Errorf("expected additional content appended, got:\n%s", content)
	}
}

// --- ApplyProposals dispatch for remaining types ---

func TestApplyProposals_CorrectOutlier(t *testing.T) {
	dir := setupStemDir(t)

	mustWriteFile(t, filepath.Join(dir, "task.md"),
		[]byte("---\nestado: Outlier\ntipo: test\n---\n# Task\n"))

	records := []*extract.Record{
		{Path: "task.md", Frontmatter: map[string]any{"estado": "Outlier", "tipo": "test"}},
	}

	report := &proposal.Report{
		Version: 1,
		Kind:    "rootline/proposals",
		Proposals: []proposal.Proposal{
			{
				Type:  proposal.CorrectOutlier,
				Field: "estado",
				From:  "Outlier",
				To:    "Pending",
				Paths: []string{"task.md"},
			},
		},
	}

	err := ApplyProposals(context.Background(), report, dir, records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "task.md"))
	if !strings.Contains(string(content), "estado: Pending") {
		t.Errorf("expected estado: Pending, got:\n%s", string(content))
	}
}

func TestApplyProposals_PropagateAggregate(t *testing.T) {
	dir := setupStemDir(t)

	mustWriteFile(t, filepath.Join(dir, "README.md"),
		[]byte("---\nestado: Pending\n---\n# Root\n"))

	records := []*extract.Record{
		{Path: "README.md", Frontmatter: map[string]any{"estado": "Pending"}},
	}

	report := &proposal.Report{
		Version: 1,
		Kind:    "rootline/proposals",
		Proposals: []proposal.Proposal{
			{
				Type:  proposal.PropagateAggregate,
				Field: "estado",
				From:  "Pending",
				To:    "Completed",
				Paths: []string{"README.md"},
			},
		},
	}

	err := ApplyProposals(context.Background(), report, dir, records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	if !strings.Contains(string(content), "estado: Completed") {
		t.Errorf("expected estado: Completed, got:\n%s", string(content))
	}
}

func TestApplyProposals_AddAggregate(t *testing.T) {
	dir := setupStemDir(t)
	stemPath := filepath.Join(dir, ".stem")

	records := []*extract.Record{}

	report := &proposal.Report{
		Version: 1,
		Kind:    "rootline/proposals",
		Proposals: []proposal.Proposal{
			{
				Type:          proposal.AddAggregate,
				Field:         "progress",
				AggregateExpr: "count(estado == 'Completed') / count(*)",
				Paths:         []string{stemPath},
			},
		},
	}

	err := ApplyProposals(context.Background(), report, dir, records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// AddAggregate is a schema proposal and should NOT be applied.
	content, _ := os.ReadFile(stemPath)
	if strings.Contains(string(content), "progress") {
		t.Errorf("add_aggregate should NOT have been applied (schema proposal), got:\n%s", string(content))
	}
	if strings.Contains(string(content), "aggregate") {
		t.Errorf("aggregate section should NOT exist (not applied), got:\n%s", string(content))
	}

	// Report should have no applied proposals.
	if len(report.Proposals) != 0 {
		t.Errorf("expected 0 applied proposals, got %d", len(report.Proposals))
	}
}

func TestApplyProposals_AddAggregate_NoPaths(t *testing.T) {
	// AddAggregate with empty Paths should not error, just skip
	dir := setupStemDir(t)
	records := []*extract.Record{}

	report := &proposal.Report{
		Version: 1,
		Kind:    "rootline/proposals",
		Proposals: []proposal.Proposal{
			{
				Type:          proposal.AddAggregate,
				Field:         "progress",
				AggregateExpr: "count(*)",
				Paths:         []string{},
			},
		},
	}

	err := ApplyProposals(context.Background(), report, dir, records)
	if err != nil {
		t.Fatalf("unexpected error for empty paths: %v", err)
	}
	// AddAggregate is a schema proposal, so it should not be in applied list.
	if len(report.Proposals) != 0 {
		t.Errorf("expected 0 applied proposals (schema surface), got %d", len(report.Proposals))
	}
}

func TestApplyProposals_RemoveStemField_NoPaths(t *testing.T) {
	// RemoveStemField with empty Paths should not error
	dir := setupStemDir(t)
	records := []*extract.Record{}

	report := &proposal.Report{
		Version: 1,
		Kind:    "rootline/proposals",
		Proposals: []proposal.Proposal{
			{
				Type:  proposal.RemoveStemField,
				Field: "estado",
				Paths: []string{},
			},
		},
	}

	err := ApplyProposals(context.Background(), report, dir, records)
	if err != nil {
		t.Fatalf("unexpected error for empty paths: %v", err)
	}
}

func TestApplyProposals_SetSection(t *testing.T) {
	dir := setupStemDir(t)

	mustWriteFile(t, filepath.Join(dir, "task.md"),
		[]byte("---\nestado: Pending\n---\n\n## Contexto\n\nOld content.\n"))

	records := []*extract.Record{
		{Path: "task.md", Frontmatter: map[string]any{"estado": "Pending"}},
	}

	report := &proposal.Report{
		Version: 1,
		Kind:    "rootline/proposals",
		Proposals: []proposal.Proposal{
			{
				Type:    proposal.SetSection,
				Field:   "contexto",
				Heading: "## Contexto",
				Value:   "New content.",
				Mode:    "replace",
				Paths:   []string{"task.md"},
			},
		},
	}

	err := ApplyProposals(context.Background(), report, dir, records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "task.md"))
	if !strings.Contains(string(content), "New content.") {
		t.Errorf("expected new content, got:\n%s", string(content))
	}
}

// --- addAggregateToStem ---

func TestAddAggregateToStem_NewSection(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	mustWriteFile(t, stemPath, []byte(`version: 2
schema:
  estado:
    type: string
`))

	err := addAggregateToStem(stemPath, "progress", "count(estado == 'Completed') / count(*)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(stemPath)
	if !strings.Contains(string(content), "aggregate") {
		t.Errorf("expected aggregate section, got:\n%s", string(content))
	}
	if !strings.Contains(string(content), "progress") {
		t.Errorf("expected progress field, got:\n%s", string(content))
	}
}

func TestAddAggregateToStem_ExistingSection(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	mustWriteFile(t, stemPath, []byte(`version: 2
schema:
  estado:
    type: string
aggregate:
  count_total: count(*)
`))

	err := addAggregateToStem(stemPath, "count_done", "count(estado == 'Completed')")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(stemPath)
	if !strings.Contains(string(content), "count_total") {
		t.Errorf("expected existing aggregate preserved, got:\n%s", string(content))
	}
	if !strings.Contains(string(content), "count_done") {
		t.Errorf("expected new aggregate added, got:\n%s", string(content))
	}
}

func TestAddAggregateToStem_FileNotFound(t *testing.T) {
	err := addAggregateToStem("/nonexistent/path/.stem", "field", "expr")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestAddAggregateToStem_EmptyDocument(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	mustWriteFile(t, stemPath, []byte(`~`))

	// An empty/null YAML doc has no content nodes
	// yaml.Unmarshal("~") gives a doc with a null scalar node as content
	// Let's use truly empty
	mustWriteFile(t, stemPath, []byte(``))

	err := addAggregateToStem(stemPath, "field", "expr")
	if err == nil {
		t.Fatal("expected error for empty document")
	}
}

// --- removeStemSchemaField error paths ---

func TestRemoveStemSchemaField_FileNotFound(t *testing.T) {
	err := removeStemSchemaField("/nonexistent/path/.stem", "field")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestRemoveStemSchemaField_EmptyDocument(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	mustWriteFile(t, stemPath, []byte(``))

	err := removeStemSchemaField(stemPath, "field")
	if err == nil {
		t.Fatal("expected error for empty document")
	}
}

func TestRemoveStemSchemaField_NoSchemaSection(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	mustWriteFile(t, stemPath, []byte(`version: 2
scope:
  match: "*.md"
`))

	err := removeStemSchemaField(stemPath, "some_field")
	if err == nil {
		t.Fatal("expected error when no schema section")
	}
	if !strings.Contains(err.Error(), "no schema section") {
		t.Errorf("expected 'no schema section' error, got: %v", err)
	}
}

func TestRemoveStemSchemaField_NonMappingRoot(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	mustWriteFile(t, stemPath, []byte(`- item1
- item2
`))

	err := removeStemSchemaField(stemPath, "field")
	if err == nil {
		t.Fatal("expected error for non-mapping root")
	}
}

// --- rewriteRecordFile error paths ---

func TestRewriteRecordFile_FileNotFound(t *testing.T) {
	dir := t.TempDir()
	fm := map[string]any{"estado": "test"}
	err := rewriteRecordFile(dir, "nonexistent.md", fm)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

// --- applyCorrectValue with missing record ---

func TestApplyCorrectValue_MissingRecord(t *testing.T) {
	dir := t.TempDir()
	recordMap := map[string]*extract.Record{} // empty - path not in map

	p := proposal.Proposal{
		Type:  proposal.CorrectValue,
		Field: "estado",
		From:  "Wrong",
		To:    "Correct",
		Paths: []string{"missing.md"},
	}

	// Should not error - missing records are skipped
	err := applyCorrectValue(p, dir, recordMap)
	if err != nil {
		t.Fatalf("unexpected error for missing record: %v", err)
	}
}

// --- applySetField with missing record ---

func TestApplySetField_MissingRecord(t *testing.T) {
	dir := t.TempDir()
	recordMap := map[string]*extract.Record{}

	p := proposal.Proposal{
		Type:  proposal.SetField,
		Field: "estado",
		Value: "Pending",
		Paths: []string{"missing.md"},
	}

	err := applySetField(p, dir, recordMap)
	if err != nil {
		t.Fatalf("unexpected error for missing record: %v", err)
	}
}

// --- helper ---

func mustWriteFile(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}
