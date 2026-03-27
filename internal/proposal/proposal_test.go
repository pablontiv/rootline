package proposal

import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/fuzzy"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestDetectExtendEnum(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Completed"}, Required: true},
		},
	}

	errs := map[string][]rules.ValidationError{
		"a.md": {{Rule: "enum", Field: "estado", Message: `value "Obsoleto" not in allowed values: [Pending, Completed]`}},
		"b.md": {{Rule: "enum", Field: "estado", Message: `value "Obsoleto" not in allowed values: [Pending, Completed]`}},
		"c.md": {{Rule: "enum", Field: "estado", Message: `value "Obsoleto" not in allowed values: [Pending, Completed]`}},
	}

	proposals := detectExtendEnum(stem, errs)
	if len(proposals) != 1 {
		t.Fatalf("got %d proposals, want 1", len(proposals))
	}
	if proposals[0].Type != ExtendEnum {
		t.Errorf("type = %q, want extend_enum", proposals[0].Type)
	}
	if proposals[0].Value != "Obsoleto" {
		t.Errorf("value = %q, want Obsoleto", proposals[0].Value)
	}
	if len(proposals[0].Paths) != 3 {
		t.Errorf("paths = %d, want 3", len(proposals[0].Paths))
	}
}

func TestDetectCorrectValue(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Completed", "In Progress"}, Required: true},
		},
	}

	errs := map[string][]rules.ValidationError{
		"a.md": {{Rule: "enum", Field: "estado", Message: `value "Completo" not in allowed values: [Pending, Completed, In Progress]`}},
	}

	proposals := detectCorrectValue(stem, errs)
	if len(proposals) != 1 {
		t.Fatalf("got %d proposals, want 1", len(proposals))
	}
	if proposals[0].Type != CorrectValue {
		t.Errorf("type = %q, want correct_value", proposals[0].Type)
	}
	if proposals[0].From != "Completo" {
		t.Errorf("from = %q, want Completo", proposals[0].From)
	}
	if proposals[0].To != "Completed" {
		t.Errorf("to = %q, want Completed", proposals[0].To)
	}
}

func TestDetectAddField(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Completed"}, Required: true},
		},
	}

	errs := map[string][]rules.ValidationError{
		"a.md": {{Rule: "required", Field: "estado", Message: `required field "estado" is missing`}},
	}

	proposals := detectAddField(stem, errs)
	if len(proposals) != 1 {
		t.Fatalf("got %d proposals, want 1", len(proposals))
	}
	if proposals[0].Type != AddField {
		t.Errorf("type = %q, want add_field", proposals[0].Type)
	}
	if proposals[0].Value != "Pending" {
		t.Errorf("value = %q, want Pending (first enum value)", proposals[0].Value)
	}
}

func TestDetectMigrateValue(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Blocked", "Completed"}, Required: true},
		},
	}

	errs := map[string][]rules.ValidationError{
		"a.md": {{Rule: "enum", Field: "estado", Message: `value "Pending (blocked by T001)" not in allowed values: [Pending, Blocked, Completed]`}},
	}

	proposals := detectMigrateValue(stem, errs)
	if len(proposals) != 1 {
		t.Fatalf("got %d proposals, want 1", len(proposals))
	}
	p := proposals[0]
	if p.Type != MigrateValue {
		t.Errorf("type = %q, want migrate_value", p.Type)
	}
	if p.From != "Pending (blocked by T001)" {
		t.Errorf("from = %q, want full original value", p.From)
	}
	if p.To != "Pending" {
		t.Errorf("to = %q, want Pending", p.To)
	}
	if len(p.WikiLinks) != 1 || p.WikiLinks[0] != "[[blocks:T001]]" {
		t.Errorf("wiki_links = %v, want [[blocks:T001]]", p.WikiLinks)
	}
}

func TestDetectExtractBody_Completada(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Completed", "In Progress"}, Required: true},
		},
	}
	records := []*extract.Record{
		{Path: "a.md", Body: "# Title\n\n**Estado**: Completada\n**Tipo**: lxc"},
	}
	errs := map[string][]rules.ValidationError{
		"a.md": {{Rule: "required", Field: "estado", Message: `required field "estado" is missing`}},
	}

	proposals := detectExtractBody(records, stem, errs)
	if len(proposals) != 1 {
		t.Fatalf("got %d proposals, want 1", len(proposals))
	}
	if proposals[0].Type != ExtractBody {
		t.Errorf("type = %q, want extract_body", proposals[0].Type)
	}
	if proposals[0].To != "Completed" {
		t.Errorf("to = %q, want Completed (mapped from Completada)", proposals[0].To)
	}
}

func TestDetectExtractBody_Activa(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Completed", "In Progress"}, Required: true},
		},
	}
	records := []*extract.Record{
		{Path: "a.md", Body: "**Estado**: Activa"},
	}
	errs := map[string][]rules.ValidationError{
		"a.md": {{Rule: "required", Field: "estado", Message: `required field "estado" is missing`}},
	}

	proposals := detectExtractBody(records, stem, errs)
	if len(proposals) != 1 {
		t.Fatalf("got %d proposals, want 1", len(proposals))
	}
	if proposals[0].To != "In Progress" {
		t.Errorf("to = %q, want 'In Progress' (mapped from Activa)", proposals[0].To)
	}
}

func TestInferEstado_AllCompleted(t *testing.T) {
	got := InferEstado([]string{"Completed", "Completed"})
	if got != "Completed" {
		t.Errorf("got %q, want Completed", got)
	}
}

func TestInferEstado_Mixed(t *testing.T) {
	got := InferEstado([]string{"Pending", "Completed"})
	if got != "In Progress" {
		t.Errorf("got %q, want 'In Progress'", got)
	}
}

func TestInferEstado_AllPending(t *testing.T) {
	got := InferEstado([]string{"Pending", "Pending"})
	if got != "Pending" {
		t.Errorf("got %q, want Pending", got)
	}
}

func TestInferEstado_Empty(t *testing.T) {
	got := InferEstado([]string{})
	if got != "Pending" {
		t.Errorf("got %q, want Pending (default for empty)", got)
	}
}

func TestDetectInferFromChildren(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Completed", "In Progress"}, Required: true},
		},
	}
	records := []*extract.Record{
		{Path: "dir/README.md", Body: "# Dir", Frontmatter: map[string]any{}},
		{Path: "dir/a.md", Frontmatter: map[string]any{"estado": "Completed"}},
		{Path: "dir/b.md", Frontmatter: map[string]any{"estado": "Pending"}},
	}
	errs := map[string][]rules.ValidationError{
		"dir/README.md": {{Rule: "required", Field: "estado", Message: `required field "estado" is missing`}},
	}

	proposals := detectInferFromChildren(records, stem, errs)
	if len(proposals) != 1 {
		t.Fatalf("got %d proposals, want 1", len(proposals))
	}
	if proposals[0].Type != InferFromChildren {
		t.Errorf("type = %q, want infer_from_children", proposals[0].Type)
	}
	if proposals[0].Value != "In Progress" {
		t.Errorf("value = %q, want 'In Progress' (mixed children)", proposals[0].Value)
	}
}

func TestExtractEnumValue_Quoted(t *testing.T) {
	val := extractEnumValue(`value "Obsoleto" not in allowed values: [Pending, Completed]`, nil)
	if val != "Obsoleto" {
		t.Errorf("got %q, want Obsoleto", val)
	}
}

func TestExtractEnumValue_Unquoted(t *testing.T) {
	val := extractEnumValue(`value Compltado is not in allowed values: [Pending, Completed]`, nil)
	if val != "Compltado" {
		t.Errorf("got %q, want Compltado", val)
	}
}

func TestExtractEnumValue_NoMatch(t *testing.T) {
	val := extractEnumValue(`some other error message`, nil)
	if val != "" {
		t.Errorf("got %q, want empty", val)
	}
}

func TestMapValue(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"Completada", "Completed"},
		{"activa", "In Progress"},
		{"pendiente", "Pending"},
		{"Unknown", "Unknown"},
	}
	for _, tt := range tests {
		got := mapValue(tt.input)
		if got != tt.want {
			t.Errorf("mapValue(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestAnalyze_NilStem(t *testing.T) {
	report := Analyze([]*extract.Record{}, nil, map[string][]rules.ValidationError{
		"a.md": {{Rule: "required", Field: "estado"}},
	})
	if len(report.Proposals) != 0 {
		t.Errorf("got %d proposals with nil stem, want 0", len(report.Proposals))
	}
}

func TestAnalyze_NoErrors(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending"}, Required: true},
		},
	}

	report := Analyze([]*extract.Record{}, stem, map[string][]rules.ValidationError{})
	if len(report.Proposals) != 0 {
		t.Errorf("got %d proposals, want 0", len(report.Proposals))
	}
	if report.Summary.Total != 0 {
		t.Errorf("summary.total = %d, want 0", report.Summary.Total)
	}
}

func TestAnalyze_Mixed(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Completed"}, Required: true},
			"tipo":   {Type: "enum", Values: []string{"feature", "bug"}, Required: true},
		},
	}

	errs := map[string][]rules.ValidationError{
		"a.md": {
			{Rule: "enum", Field: "estado", Message: `value "Completo" not in allowed values: [Pending, Completed]`},
		},
		"b.md": {
			{Rule: "required", Field: "tipo", Message: `required field "tipo" is missing`},
		},
	}

	report := Analyze([]*extract.Record{}, stem, errs)
	if report.Summary.Total != 2 {
		t.Errorf("summary.total = %d, want 2", report.Summary.Total)
	}
	if report.Summary.CorrectValue != 1 {
		t.Errorf("summary.correct_value = %d, want 1", report.Summary.CorrectValue)
	}
	if report.Summary.AddField != 1 {
		t.Errorf("summary.add_field = %d, want 1", report.Summary.AddField)
	}
}

func TestExtendEnum_SkipsParenthesizedValues(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Completed"}, Required: true},
		},
	}

	// Same parenthesized value in 3 records — should NOT generate extend_enum.
	errs := map[string][]rules.ValidationError{
		"a.md": {{Rule: "enum", Field: "estado", Message: `value "Pending (blocked by T001)" not in allowed values: [Pending, Completed]`}},
		"b.md": {{Rule: "enum", Field: "estado", Message: `value "Pending (blocked by T001)" not in allowed values: [Pending, Completed]`}},
		"c.md": {{Rule: "enum", Field: "estado", Message: `value "Pending (blocked by T001)" not in allowed values: [Pending, Completed]`}},
	}

	proposals := detectExtendEnum(stem, errs)
	if len(proposals) != 0 {
		t.Errorf("got %d proposals, want 0 (parenthesized values should be skipped)", len(proposals))
	}
}

func TestAnalyze_MigrateValueSuppressesCorrectValue(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Blocked", "Completed"}, Required: true},
		},
	}

	// This error triggers both migrate_value (has parentheses) and correct_value (base is close to "Pending").
	errs := map[string][]rules.ValidationError{
		"a.md": {{Rule: "enum", Field: "estado", Message: `value "Pending (blocked by T001)" not in allowed values: [Pending, Blocked, Completed]`}},
	}

	report := Analyze([]*extract.Record{}, stem, errs)

	if report.Summary.MigrateValue != 1 {
		t.Errorf("summary.migrate_value = %d, want 1", report.Summary.MigrateValue)
	}
	if report.Summary.CorrectValue != 0 {
		t.Errorf("summary.correct_value = %d, want 0 (suppressed by migrate_value)", report.Summary.CorrectValue)
	}
	if report.Summary.ExtendEnum != 0 {
		t.Errorf("summary.extend_enum = %d, want 0", report.Summary.ExtendEnum)
	}
}

func TestAnalyze_MigrateValueSuppressesExtendEnum(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Completed"}, Required: true},
		},
	}

	// Same parenthesized value in 2 records — should produce migrate_value but NOT extend_enum.
	errs := map[string][]rules.ValidationError{
		"a.md": {{Rule: "enum", Field: "estado", Message: `value "Pending (blocked by T001)" not in allowed values: [Pending, Completed]`}},
		"b.md": {{Rule: "enum", Field: "estado", Message: `value "Pending (blocked by T001)" not in allowed values: [Pending, Completed]`}},
	}

	report := Analyze([]*extract.Record{}, stem, errs)

	if report.Summary.MigrateValue != 2 {
		t.Errorf("summary.migrate_value = %d, want 2", report.Summary.MigrateValue)
	}
	if report.Summary.ExtendEnum != 0 {
		t.Errorf("summary.extend_enum = %d, want 0 (suppressed by migrate_value / parentheses)", report.Summary.ExtendEnum)
	}
	if report.Summary.CorrectValue != 0 {
		t.Errorf("summary.correct_value = %d, want 0 (suppressed by migrate_value)", report.Summary.CorrectValue)
	}
}

func TestMigrateValue_NotesOnlyHasWikiLinks(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Completed"}, Required: true},
		},
	}

	// "human" is a note (not an ID-like target) but should still produce wiki_links.
	errs := map[string][]rules.ValidationError{
		"a.md": {{Rule: "enum", Field: "estado", Message: `value "Pending (blocked by human)" not in allowed values: [Pending, Completed]`}},
	}

	proposals := detectMigrateValue(stem, errs)
	if len(proposals) != 1 {
		t.Fatalf("got %d proposals, want 1", len(proposals))
	}
	p := proposals[0]
	if len(p.WikiLinks) == 0 {
		t.Errorf("wiki_links is empty, want non-empty for notes-only migrate_value")
	}
	if p.WikiLinks[0] != "[[note:human]]" {
		t.Errorf("wiki_links[0] = %q, want [[note:human]]", p.WikiLinks[0])
	}
}

func TestMigrateValue_MixedTargetsAndNotes(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Completed"}, Required: true},
		},
	}

	errs := map[string][]rules.ValidationError{
		"a.md": {{Rule: "enum", Field: "estado", Message: `value "Pending (blocked by E04 + human)" not in allowed values: [Pending, Completed]`}},
	}

	proposals := detectMigrateValue(stem, errs)
	if len(proposals) != 1 {
		t.Fatalf("got %d proposals, want 1", len(proposals))
	}
	p := proposals[0]
	if len(p.WikiLinks) != 2 {
		t.Fatalf("wiki_links = %v, want 2 entries", p.WikiLinks)
	}
	if p.WikiLinks[0] != "[[blocks:E04]]" {
		t.Errorf("wiki_links[0] = %q, want [[blocks:E04]]", p.WikiLinks[0])
	}
	if p.WikiLinks[1] != "[[note:human]]" {
		t.Errorf("wiki_links[1] = %q, want [[note:human]]", p.WikiLinks[1])
	}
}

func TestAnalyze_ExtractBodySuppressesAddField(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Completed", "In Progress"}, Required: true},
		},
	}
	records := []*extract.Record{
		{Path: "a.md", Body: "# Title\n\n**Estado**: Completada"},
	}
	errs := map[string][]rules.ValidationError{
		"a.md": {{Rule: "required", Field: "estado", Message: `required field "estado" is missing`}},
	}

	report := Analyze(records, stem, errs)

	// extract_body should be present, add_field should be suppressed.
	if report.Summary.ExtractBody != 1 {
		t.Errorf("summary.extract_body = %d, want 1", report.Summary.ExtractBody)
	}
	if report.Summary.AddField != 0 {
		t.Errorf("summary.add_field = %d, want 0 (suppressed by extract_body)", report.Summary.AddField)
	}
	if report.Summary.Total != 1 {
		t.Errorf("summary.total = %d, want 1", report.Summary.Total)
	}
}

func TestAnalyze_InferFromChildrenSuppressesAddField(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Completed", "In Progress"}, Required: true},
		},
	}
	records := []*extract.Record{
		{Path: "dir/README.md", Body: "# Dir", Frontmatter: map[string]any{}},
		{Path: "dir/a.md", Frontmatter: map[string]any{"estado": "Completed"}},
		{Path: "dir/b.md", Frontmatter: map[string]any{"estado": "Pending"}},
	}
	errs := map[string][]rules.ValidationError{
		"dir/README.md": {{Rule: "required", Field: "estado", Message: `required field "estado" is missing`}},
	}

	report := Analyze(records, stem, errs)

	// infer_from_children should be present, add_field should be suppressed.
	if report.Summary.InferFromChildren != 1 {
		t.Errorf("summary.infer_from_children = %d, want 1", report.Summary.InferFromChildren)
	}
	if report.Summary.AddField != 0 {
		t.Errorf("summary.add_field = %d, want 0 (suppressed by infer_from_children)", report.Summary.AddField)
	}
	if report.Summary.Total != 1 {
		t.Errorf("summary.total = %d, want 1", report.Summary.Total)
	}
}

func TestAnalyze_AddFieldSurvivesWithoutCoverage(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Completed"}, Required: true},
		},
	}
	// No body hints, not a README with children — add_field should survive.
	records := []*extract.Record{
		{Path: "standalone.md", Body: "# No hints here", Frontmatter: map[string]any{}},
	}
	errs := map[string][]rules.ValidationError{
		"standalone.md": {{Rule: "required", Field: "estado", Message: `required field "estado" is missing`}},
	}

	report := Analyze(records, stem, errs)

	if report.Summary.AddField != 1 {
		t.Errorf("summary.add_field = %d, want 1 (no other detector covers this)", report.Summary.AddField)
	}
	if report.Summary.ExtractBody != 0 {
		t.Errorf("summary.extract_body = %d, want 0", report.Summary.ExtractBody)
	}
	if report.Summary.InferFromChildren != 0 {
		t.Errorf("summary.infer_from_children = %d, want 0", report.Summary.InferFromChildren)
	}
}

func TestDetectCorrectLink_Retype(t *testing.T) {
	stem := &rules.StemFile{
		Links: rules.LinkSchema{
			Allowed: []string{"blocks", "reference"},
			Rules: map[string]rules.LinkRule{
				"blocks": {Target: `^T\d{3}-`},
			},
		},
	}
	records := []*extract.Record{
		{
			Path: "dir/T001-task.md",
			Links: []extract.Link{
				{Type: "blocks", Target: "E04", Line: 5},
			},
		},
	}
	errs := map[string][]rules.ValidationError{
		"dir/T001-task.md": {{
			Rule:    "link_target",
			Field:   "links",
			Message: `link target "E04" does not match pattern "^T\d{3}-"`,
		}},
	}

	proposals := detectCorrectLink(records, stem, errs)
	if len(proposals) != 1 {
		t.Fatalf("got %d proposals, want 1", len(proposals))
	}
	p := proposals[0]
	if p.Type != CorrectLink {
		t.Errorf("type = %q, want correct_link", p.Type)
	}
	if p.From != "[[blocks:E04]]" {
		t.Errorf("from = %q, want [[blocks:E04]]", p.From)
	}
	if p.To != "[[reference:E04]]" {
		t.Errorf("to = %q, want [[reference:E04]]", p.To)
	}
}

func TestDetectCorrectLink_Expand(t *testing.T) {
	stem := &rules.StemFile{
		Links: rules.LinkSchema{
			Allowed: []string{"blocks"},
			Rules: map[string]rules.LinkRule{
				"blocks": {Target: `^T\d{3}-`},
			},
		},
	}
	records := []*extract.Record{
		{
			Path: "dir/T002-validate.md",
			Links: []extract.Link{
				{Type: "blocks", Target: "T001", Line: 5},
			},
		},
		{
			Path:  "dir/T001-add-feature.md",
			Links: []extract.Link{},
		},
	}
	errs := map[string][]rules.ValidationError{
		"dir/T002-validate.md": {{
			Rule:    "link_target",
			Field:   "links",
			Message: `link target "T001" does not match pattern "^T\d{3}-"`,
		}},
	}

	proposals := detectCorrectLink(records, stem, errs)
	if len(proposals) != 1 {
		t.Fatalf("got %d proposals, want 1", len(proposals))
	}
	p := proposals[0]
	if p.From != "[[blocks:T001]]" {
		t.Errorf("from = %q, want [[blocks:T001]]", p.From)
	}
	if p.To != "[[blocks:T001-add-feature]]" {
		t.Errorf("to = %q, want [[blocks:T001-add-feature]]", p.To)
	}
}

func TestDetectCorrectLink_ExpandPriorityOverRetype(t *testing.T) {
	// When both expand and retype are possible, expand wins (preserves link semantics).
	stem := &rules.StemFile{
		Links: rules.LinkSchema{
			Allowed: []string{"blocks", "reference"},
			Rules: map[string]rules.LinkRule{
				"blocks": {Target: `^T\d{3}-`},
			},
		},
	}
	records := []*extract.Record{
		{
			Path: "dir/T002-validate.md",
			Links: []extract.Link{
				{Type: "blocks", Target: "T001", Line: 5},
			},
		},
		{
			Path: "dir/T001-add-feature.md",
		},
	}
	errs := map[string][]rules.ValidationError{
		"dir/T002-validate.md": {{
			Rule:    "link_target",
			Field:   "links",
			Message: `link target "T001" does not match pattern "^T\d{3}-"`,
		}},
	}

	proposals := detectCorrectLink(records, stem, errs)
	if len(proposals) != 1 {
		t.Fatalf("got %d proposals, want 1", len(proposals))
	}
	// Expand wins: [[blocks:T001]] → [[blocks:T001-add-feature]]
	if proposals[0].To != "[[blocks:T001-add-feature]]" {
		t.Errorf("to = %q, want [[blocks:T001-add-feature]] (expand has priority)", proposals[0].To)
	}
}

func TestDetectCorrectLink_NoMatch(t *testing.T) {
	stem := &rules.StemFile{
		Links: rules.LinkSchema{
			Allowed: []string{"blocks"},
			Rules: map[string]rules.LinkRule{
				"blocks": {Target: `^T\d{3}-`},
			},
		},
	}
	records := []*extract.Record{
		{
			Path: "dir/T001-task.md",
			Links: []extract.Link{
				{Type: "blocks", Target: "UNKNOWN", Line: 5},
			},
		},
	}
	errs := map[string][]rules.ValidationError{
		"dir/T001-task.md": {{
			Rule:    "link_target",
			Field:   "links",
			Message: `link target "UNKNOWN" does not match pattern "^T\d{3}-"`,
		}},
	}

	proposals := detectCorrectLink(records, stem, errs)
	if len(proposals) != 0 {
		t.Errorf("got %d proposals, want 0 (no fix possible)", len(proposals))
	}
}

func TestExtractLinkTargetAndPattern(t *testing.T) {
	target, pattern := extractLinkTargetAndPattern(`link target "E04" does not match pattern "^T\d{3}-"`)
	if target != "E04" {
		t.Errorf("target = %q, want E04", target)
	}
	if pattern != `^T\d{3}-` {
		t.Errorf("pattern = %q, want ^T\\d{3}-", pattern)
	}
}

func TestAnalyze_MigrateValueSuppressesExtendAndCorrect(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Blocked", "Completed"}, Required: true},
		},
	}
	errs := map[string][]rules.ValidationError{
		"a.md": {{Rule: "enum", Field: "estado", Message: `value "Pending (blocked by T001)" not in allowed values: [Pending, Blocked, Completed]`}},
		"b.md": {{Rule: "enum", Field: "estado", Message: `value "Pending (blocked by T001)" not in allowed values: [Pending, Blocked, Completed]`}},
	}

	report := Analyze([]*extract.Record{}, stem, errs)

	// migrate_value should handle these, not extend_enum or correct_value.
	if report.Summary.MigrateValue != 2 {
		t.Errorf("summary.migrate_value = %d, want 2", report.Summary.MigrateValue)
	}
	if report.Summary.ExtendEnum != 0 {
		t.Errorf("summary.extend_enum = %d, want 0 (suppressed by migrate_value)", report.Summary.ExtendEnum)
	}
	if report.Summary.CorrectValue != 0 {
		t.Errorf("summary.correct_value = %d, want 0 (suppressed by migrate_value)", report.Summary.CorrectValue)
	}
}

func TestClosestMatch_NoCloseMatch(t *testing.T) {
	got := fuzzy.Match("xyz", []string{"Pending", "Completed"})
	if got != "" {
		t.Errorf("fuzzy.Match(\"xyz\", ...) = %q, want \"\" (too far from any candidate)", got)
	}
}

func TestClosestMatch_CloseMatch(t *testing.T) {
	got := fuzzy.Match("Completo", []string{"Pending", "Completed"})
	if got != "Completed" {
		t.Errorf("fuzzy.Match(\"Completo\", ...) = %q, want \"Completed\"", got)
	}
}

func TestClosestMatch_EmptyCandidates(t *testing.T) {
	got := fuzzy.Match("anything", []string{})
	if got != "" {
		t.Errorf("closestMatch with empty candidates = %q, want \"\"", got)
	}
}

func TestDetectCorrectValue_NoSuggestionWhenTooFar(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Completed", "In Progress"}, Required: true},
		},
	}

	errs := map[string][]rules.ValidationError{
		"a.md": {{Rule: "enum", Field: "estado", Message: `value "xyz" not in allowed values: [Pending, Completed, In Progress]`}},
	}

	proposals := detectCorrectValue(stem, errs)
	if len(proposals) != 0 {
		t.Errorf("got %d proposals, want 0 (value too far from any candidate)", len(proposals))
	}
}

func TestProposal_SetFieldType(t *testing.T) {
	p := Proposal{
		Type:  SetField,
		Field: "estado",
		Value: "listo-para-implementar",
		Paths: []string{"backlog/B023.md"},
	}
	if p.Type != "set_field" {
		t.Errorf("type = %q, want set_field", p.Type)
	}
}

func TestProposal_SetSectionType(t *testing.T) {
	p := Proposal{
		Type:    SetSection,
		Field:   "investigacion",
		Value:   "Decision: GO\nEvidence: ...",
		Heading: "## Investigación",
		Mode:    "replace",
		Paths:   []string{"backlog/B023.md"},
	}
	if p.Type != "set_section" {
		t.Errorf("type = %q, want set_section", p.Type)
	}
	if p.Heading != "## Investigación" {
		t.Errorf("heading = %q", p.Heading)
	}
	if p.Mode != "replace" {
		t.Errorf("mode = %q", p.Mode)
	}
}
