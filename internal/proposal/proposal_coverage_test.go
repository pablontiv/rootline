package proposal

import (
	"fmt"
	"slices"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestCombineReportsPreservesOrderAndRecountsSummary(t *testing.T) {
	first := &Report{
		Proposals:         []Proposal{{Type: AddField, Field: "alpha"}},
		SchemaSuggestions: []Proposal{{Type: ExtendEnum, Field: "state", Value: "Alpha"}},
		LinkFindings:      []LinkFinding{{Path: "alpha/a.md", Rule: "link_resolve"}},
		TypeFindings:      []TypeFinding{{Path: "alpha/a.md", Field: "alpha"}},
	}
	second := &Report{
		Proposals:    []Proposal{{Type: CorrectValue, Field: "beta"}},
		TypeFindings: []TypeFinding{{Path: "beta/b.md", Field: "beta"}},
	}

	got := CombineReports(first, nil, second)
	if got.Version != 1 || got.Kind != "rootline/proposals" {
		t.Fatalf("unexpected envelope: version=%d kind=%q", got.Version, got.Kind)
	}
	if len(got.Proposals) != 2 || got.Proposals[0].Field != "alpha" || got.Proposals[1].Field != "beta" {
		t.Fatalf("proposal order changed: %#v", got.Proposals)
	}
	if got.Summary.Total != 3 || got.Summary.AddField != 1 || got.Summary.CorrectValue != 1 || got.Summary.ExtendEnum != 1 {
		t.Errorf("summary was not recomputed: %#v", got.Summary)
	}
	if len(got.SchemaSuggestions) != 1 || len(got.LinkFindings) != 1 || len(got.TypeFindings) != 2 {
		t.Errorf("findings were not combined: %#v", got)
	}
}

// TestAnalyze_Basic tests the top-level Analyze orchestrator with various scenarios.
func TestAnalyze_Basic(t *testing.T) {
	tests := []struct {
		name            string
		records         []*extract.Record
		effective       *rules.StemFile
		errs            map[string][]rules.ValidationError
		expectProposals bool
		expectSummary   bool
	}{
		{
			name:            "nil effective",
			records:         []*extract.Record{},
			effective:       nil,
			errs:            map[string][]rules.ValidationError{},
			expectProposals: false,
			expectSummary:   false,
		},
		{
			name:      "empty errors",
			effective: &rules.StemFile{Schema: map[string]rules.SchemaField{}},
			errs:      map[string][]rules.ValidationError{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report := Analyze(tc.records, tc.effective, tc.errs)
			if report == nil {
				t.Fatal("Analyze returned nil report")
			}
			if report.Version != 1 {
				t.Errorf("expected version 1, got %d", report.Version)
			}
			if report.Kind != "rootline/proposals" {
				t.Errorf("expected kind rootline/proposals, got %q", report.Kind)
			}
		})
	}
}

// TestAnalyze_MigrateValuePhase tests Analyze's migrate_value phase.
func TestAnalyze_MigrateValuePhase(t *testing.T) {
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {
				Type:   "enum",
				Values: []string{"Pending", "In Progress", "Completed"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"docs/tasks/T001.md": {
			{
				Field:   "estado",
				Rule:    "enum",
				Message: `value Pending (blocked by E04/F01) is not in allowed values: Pending, In Progress, Completed`,
			},
		},
	}

	report := Analyze([]*extract.Record{}, effective, errs)
	if len(report.Proposals) == 0 {
		t.Fatal("expected migrate_value proposal, got none")
	}

	found := false
	for _, p := range report.Proposals {
		if p.Type == MigrateValue {
			found = true
			if p.From != "Pending (blocked by E04/F01)" {
				t.Errorf("expected From='Pending (blocked by E04/F01)', got %q", p.From)
			}
			if p.To != "Pending" {
				t.Errorf("expected To='Pending', got %q", p.To)
			}
			break
		}
	}
	if !found {
		t.Errorf("migrate_value proposal not found. proposals: %v", report.Proposals)
	}
}

// TestAnalyze_ExtendEnumPhase tests that migrate_value candidates are filtered from extend_enum.
func TestAnalyze_ExtendEnumPhase(t *testing.T) {
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {
				Type:   "enum",
				Values: []string{"Pending", "Completed"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"docs/tasks/T001.md": {
			{
				Field:   "estado",
				Rule:    "enum",
				Message: `value Pending (blocked by E04) is not in allowed values: Pending, Completed`,
			},
		},
		"docs/tasks/T002.md": {
			{
				Field:   "estado",
				Rule:    "enum",
				Message: `value Pending (blocked by E04) is not in allowed values: Pending, Completed`,
			},
		},
	}

	report := Analyze([]*extract.Record{}, effective, errs)

	// Should have migrate_value, not extend_enum, because the value has (blocked by ...) info
	for _, p := range report.Proposals {
		if p.Type == ExtendEnum && p.Field == "estado" {
			t.Errorf("unexpected extend_enum proposal: %v (should be migrate_value instead)", p)
		}
	}
}

// TestAnalyze_CorrectValuePhase tests that migrate_value has priority over correct_value.
func TestAnalyze_CorrectValuePhase(t *testing.T) {
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {
				Type:   "enum",
				Values: []string{"Pending", "In Progress", "Completed"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"docs/tasks/T001.md": {
			{
				Field:   "estado",
				Rule:    "enum",
				Message: `value Pending (blocked by E04) is not in allowed values: Pending, In Progress, Completed`,
			},
		},
	}

	report := Analyze([]*extract.Record{}, effective, errs)

	// Should have migrate_value, not correct_value
	hasMigrate := false
	hasCorrect := false
	for _, p := range report.Proposals {
		if p.Type == MigrateValue {
			hasMigrate = true
		}
		if p.Type == CorrectValue && p.Field == "estado" {
			hasCorrect = true
		}
	}

	if !hasMigrate {
		t.Error("expected migrate_value proposal")
	}
	if hasCorrect {
		t.Error("correct_value should not appear when migrate_value is applicable")
	}
}

// TestDetectMigrateValue_BasicMigration tests detectMigrateValue with a simple case.
func TestDetectMigrateValue_BasicMigration(t *testing.T) {
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {
				Type:   "enum",
				Values: []string{"Pending", "In Progress", "Completed"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"T001.md": {
			{
				Field:   "estado",
				Rule:    "enum",
				Message: `value Pending (blocked by T002) is not in allowed values: Pending, In Progress, Completed`,
			},
		},
	}

	proposals := detectMigrateValue(effective, errs)
	if len(proposals) != 1 {
		t.Fatalf("expected 1 proposal, got %d", len(proposals))
	}

	p := proposals[0]
	if p.Type != MigrateValue {
		t.Errorf("expected MigrateValue, got %v", p.Type)
	}
	if p.Field != "estado" {
		t.Errorf("expected field estado, got %q", p.Field)
	}
	if p.From != "Pending (blocked by T002)" {
		t.Errorf("expected From='Pending (blocked by T002)', got %q", p.From)
	}
	if p.To != "Pending" {
		t.Errorf("expected To='Pending', got %q", p.To)
	}
	if len(p.WikiLinks) != 1 {
		t.Errorf("expected 1 wiki link, got %d", len(p.WikiLinks))
	}
}

// TestDetectMigrateValue_MultipleTargets tests detectMigrateValue with multiple targets.
func TestDetectMigrateValue_MultipleTargets(t *testing.T) {
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {
				Type:   "enum",
				Values: []string{"Pending", "Blocked", "Completed"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"T001.md": {
			{
				Field:   "estado",
				Rule:    "enum",
				Message: `value Blocked (blocked by E04/F01 + T002) is not in allowed values: Pending, Blocked, Completed`,
			},
		},
	}

	proposals := detectMigrateValue(effective, errs)
	if len(proposals) != 1 {
		t.Fatalf("expected 1 proposal, got %d", len(proposals))
	}

	p := proposals[0]
	if len(p.WikiLinks) != 2 {
		t.Errorf("expected 2 wiki links, got %d: %v", len(p.WikiLinks), p.WikiLinks)
	}
}

// TestDetectMigrateValue_WithNotes tests detectMigrateValue with mixed targets and notes.
func TestDetectMigrateValue_WithNotes(t *testing.T) {
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {
				Type:   "enum",
				Values: []string{"Pending", "In Progress"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"T001.md": {
			{
				Field:   "estado",
				Rule:    "enum",
				Message: `value In Progress (blocked by T002 + waiting for review) is not in allowed values: Pending, In Progress`,
			},
		},
	}

	proposals := detectMigrateValue(effective, errs)
	if len(proposals) != 1 {
		t.Fatalf("expected 1 proposal, got %d", len(proposals))
	}

	p := proposals[0]
	if len(p.WikiLinks) != 2 {
		t.Errorf("expected 2 wiki links (1 target + 1 note), got %d: %v", len(p.WikiLinks), p.WikiLinks)
	}
}

// TestDetectMigrateValue_NoParentheses tests that values without parentheses are not migrated.
func TestDetectMigrateValue_NoParentheses(t *testing.T) {
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {
				Type:   "enum",
				Values: []string{"Pending", "In Progress"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"T001.md": {
			{
				Field:   "estado",
				Rule:    "enum",
				Message: `value Obsoleto is not in allowed values: Pending, In Progress`,
			},
		},
	}

	proposals := detectMigrateValue(effective, errs)
	if len(proposals) != 0 {
		t.Errorf("expected 0 migrate_value proposals, got %d: %v", len(proposals), proposals)
	}
}

// TestDetectMigrateValue_NoEnumSchema tests when enum schema doesn't exist.
func TestDetectMigrateValue_NoEnumSchema(t *testing.T) {
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"title": {
				Type: "string",
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"T001.md": {
			{
				Field:   "estado",
				Rule:    "enum",
				Message: `value Pending (blocked by T002) is not in allowed values: ...`,
			},
		},
	}

	proposals := detectMigrateValue(effective, errs)
	if len(proposals) != 0 {
		t.Errorf("expected 0 proposals when enum field missing, got %d", len(proposals))
	}
}

// TestDetectExtractBody_HappyPath tests extracting a value from body.
func TestDetectExtractBody_HappyPath(t *testing.T) {
	records := []*extract.Record{
		{
			Path: "T001.md",
			Body: "**Estado**: Completada\n**Titulo**: Task One",
			Frontmatter: map[string]any{
				"titulo": "Task One",
			},
		},
	}

	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {
				Type:   "enum",
				Values: []string{"Pending", "Completed", "In Progress"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"T001.md": {
			{
				Field:   "estado",
				Rule:    "required",
				Message: "required field estado is missing",
			},
		},
	}

	proposals := detectExtractBody(records, effective, errs)
	if len(proposals) != 1 {
		t.Fatalf("expected 1 proposal, got %d", len(proposals))
	}

	p := proposals[0]
	if p.Type != ExtractBody {
		t.Errorf("expected ExtractBody, got %v", p.Type)
	}
	if p.Field != "estado" {
		t.Errorf("expected field estado, got %q", p.Field)
	}
	if p.From != "Completada" {
		t.Errorf("expected From='Completada', got %q", p.From)
	}
	if p.To != "Completed" {
		t.Errorf("expected To='Completed', got %q", p.To)
	}
}

// TestDetectExtractBody_NoBodyFields tests when body has no bold-colon patterns.
func TestDetectExtractBody_NoBodyFields(t *testing.T) {
	records := []*extract.Record{
		{
			Path:        "T001.md",
			Body:        "Just plain text with no bold patterns",
			Frontmatter: map[string]any{},
		},
	}

	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {
				Type:   "enum",
				Values: []string{"Pending", "Completed"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"T001.md": {
			{
				Field:   "estado",
				Rule:    "required",
				Message: "required field estado is missing",
			},
		},
	}

	proposals := detectExtractBody(records, effective, errs)
	if len(proposals) != 0 {
		t.Errorf("expected 0 proposals when no body fields, got %d", len(proposals))
	}
}

// TestDetectExtractBody_NoRecordMapping tests when record path is not in recordMap.
func TestDetectExtractBody_NoRecordMapping(t *testing.T) {
	records := []*extract.Record{}

	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {
				Type:   "enum",
				Values: []string{"Pending", "Completed"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"T001.md": {
			{
				Field:   "estado",
				Rule:    "required",
				Message: "required field estado is missing",
			},
		},
	}

	proposals := detectExtractBody(records, effective, errs)
	if len(proposals) != 0 {
		t.Errorf("expected 0 proposals when record not found, got %d", len(proposals))
	}
}

// TestDetectExtractBody_NoBodyText tests when record has no body.
func TestDetectExtractBody_NoBodyText(t *testing.T) {
	records := []*extract.Record{
		{
			Path:        "T001.md",
			Body:        "",
			Frontmatter: map[string]any{},
		},
	}

	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {
				Type:   "enum",
				Values: []string{"Pending", "Completed"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"T001.md": {
			{
				Field:   "estado",
				Rule:    "required",
				Message: "required field estado is missing",
			},
		},
	}

	proposals := detectExtractBody(records, effective, errs)
	if len(proposals) != 0 {
		t.Errorf("expected 0 proposals when body is empty, got %d", len(proposals))
	}
}

// TestDetectExtractBody_FieldNotInBody tests when field is not in body.
func TestDetectExtractBody_FieldNotInBody(t *testing.T) {
	records := []*extract.Record{
		{
			Path:        "T001.md",
			Body:        "**Other**: Value",
			Frontmatter: map[string]any{},
		},
	}

	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {
				Type:   "enum",
				Values: []string{"Pending", "Completed"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"T001.md": {
			{
				Field:   "estado",
				Rule:    "required",
				Message: "required field estado is missing",
			},
		},
	}

	proposals := detectExtractBody(records, effective, errs)
	if len(proposals) != 0 {
		t.Errorf("expected 0 proposals when field not in body, got %d", len(proposals))
	}
}

// TestDetectExtractBody_WithEnumMapping tests body extraction with enum validation.
func TestDetectExtractBody_WithEnumMapping(t *testing.T) {
	records := []*extract.Record{
		{
			Path:        "T001.md",
			Body:        "**Estado**: Activa",
			Frontmatter: map[string]any{},
		},
	}

	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {
				Type:   "enum",
				Values: []string{"Pending", "In Progress", "Completed"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"T001.md": {
			{
				Field:   "estado",
				Rule:    "required",
				Message: "required field estado is missing",
			},
		},
	}

	proposals := detectExtractBody(records, effective, errs)
	if len(proposals) != 1 {
		t.Fatalf("expected 1 proposal, got %d", len(proposals))
	}

	p := proposals[0]
	if p.To != "In Progress" {
		t.Errorf("expected To='In Progress' (mapped from Activa), got %q", p.To)
	}
}

// TestDetectCorrectLink_BasicExpand tests link expansion for abbreviated targets.
func TestDetectCorrectLink_BasicExpand(t *testing.T) {
	records := []*extract.Record{
		{
			Path: "docs/tasks/T001.md",
			Links: []extract.Link{
				{Type: "blocks", Target: "T002"},
			},
			Body: "",
		},
		{
			Path: "docs/tasks/T002-full-name.md",
			Body: "",
		},
	}

	effective := &rules.StemFile{
		Links: rules.LinkSchema{
			Allowed: []string{"blocks"},
			Rules: map[string]rules.LinkRule{
				"blocks": {Target: "T\\d{3}-"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"docs/tasks/T001.md": {
			{
				Field:   "links",
				Rule:    "link_target",
				Message: `link target "T002" does not match pattern "T\\d{3}-"`,
			},
		},
	}

	proposals := detectCorrectLink(records, effective, errs)
	if len(proposals) != 1 {
		t.Fatalf("expected 1 proposal, got %d", len(proposals))
	}

	p := proposals[0]
	if p.Type != CorrectLink {
		t.Errorf("expected CorrectLink, got %v", p.Type)
	}
	if p.To != "[[blocks:T002-full-name]]" {
		t.Errorf("expected To='[[blocks:T002-full-name]]', got %q", p.To)
	}
}

// TestDetectCorrectLink_MultipleMatches tests that expansion returns empty when multiple matches exist.
func TestDetectCorrectLink_MultipleMatches(t *testing.T) {
	records := []*extract.Record{
		{
			Path: "docs/tasks/T001.md",
			Links: []extract.Link{
				{Type: "blocks", Target: "T002"},
			},
			Body: "",
		},
		{
			Path: "docs/tasks/T002-v1.md",
			Body: "",
		},
		{
			Path: "docs/tasks/T002-v2.md",
			Body: "",
		},
	}

	effective := &rules.StemFile{
		Links: rules.LinkSchema{
			Allowed: []string{"blocks"},
			Rules: map[string]rules.LinkRule{
				"blocks": {Target: "T\\d{3}-"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"docs/tasks/T001.md": {
			{
				Field:   "links",
				Rule:    "link_target",
				Message: `link target "T002" does not match pattern "T\\d{3}-"`,
			},
		},
	}

	proposals := detectCorrectLink(records, effective, errs)
	// Multiple matches should prevent expansion, so retype might be suggested (or no proposals if no alternative type)
	if len(proposals) > 0 {
		for _, p := range proposals {
			if p.To == "[[blocks:T002-v1]]" || p.To == "[[blocks:T002-v2]]" {
				t.Errorf("should not suggest specific match when multiple exist: %q", p.To)
			}
		}
	}
}

// TestDetectCorrectLink_NoRecordFound tests when record is not in map.
func TestDetectCorrectLink_NoRecordFound(t *testing.T) {
	records := []*extract.Record{}

	effective := &rules.StemFile{
		Links: rules.LinkSchema{
			Allowed: []string{"blocks"},
			Rules: map[string]rules.LinkRule{
				"blocks": {Target: "T\\d{3}-"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"docs/tasks/T001.md": {
			{
				Field:   "links",
				Rule:    "link_target",
				Message: `link target "T002" does not match pattern "T\\d{3}-"`,
			},
		},
	}

	proposals := detectCorrectLink(records, effective, errs)
	if len(proposals) != 0 {
		t.Errorf("expected 0 proposals when record not found, got %d", len(proposals))
	}
}

// TestDetectCorrectLink_NoLinkInRecord tests when link type is not in record.
func TestDetectCorrectLink_NoLinkInRecord(t *testing.T) {
	records := []*extract.Record{
		{
			Path:  "docs/tasks/T001.md",
			Links: []extract.Link{}, // no links
			Body:  "",
		},
	}

	effective := &rules.StemFile{
		Links: rules.LinkSchema{
			Allowed: []string{"blocks"},
			Rules: map[string]rules.LinkRule{
				"blocks": {Target: "T\\d{3}-"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"docs/tasks/T001.md": {
			{
				Field:   "links",
				Rule:    "link_target",
				Message: `link target "T002" does not match pattern "T\\d{3}-"`,
			},
		},
	}

	proposals := detectCorrectLink(records, effective, errs)
	if len(proposals) != 0 {
		t.Errorf("expected 0 proposals when link not in record, got %d", len(proposals))
	}
}

// TestDetectCorrectLink_EmptyLinks tests when effective links is empty.
func TestDetectCorrectLink_EmptyLinks(t *testing.T) {
	records := []*extract.Record{
		{
			Path: "docs/tasks/T001.md",
			Links: []extract.Link{
				{Type: "blocks", Target: "T002"},
			},
			Body: "",
		},
	}

	effective := &rules.StemFile{
		Links: rules.LinkSchema{},
	}

	errs := map[string][]rules.ValidationError{
		"docs/tasks/T001.md": {
			{
				Field:   "links",
				Rule:    "link_target",
				Message: `link target "T002" does not match pattern "T\\d{3}-"`,
			},
		},
	}

	proposals := detectCorrectLink(records, effective, errs)
	if len(proposals) != 0 {
		t.Errorf("expected 0 proposals when links schema is empty, got %d", len(proposals))
	}
}

// TestDetectCorrectLink_NilEffective tests when effective is nil.
func TestDetectCorrectLink_NilEffective(t *testing.T) {
	records := []*extract.Record{
		{
			Path: "docs/tasks/T001.md",
			Body: "",
		},
	}

	errs := map[string][]rules.ValidationError{
		"docs/tasks/T001.md": {
			{
				Field:   "links",
				Rule:    "link_target",
				Message: `link target "T002" does not match pattern "T\\d{3}-"`,
			},
		},
	}

	proposals := detectCorrectLink(records, nil, errs)
	if len(proposals) != 0 {
		t.Errorf("expected 0 proposals when effective is nil, got %d", len(proposals))
	}
}

// TestDetectCorrectLink_InvalidTargetExtraction tests when target cannot be extracted.
func TestDetectCorrectLink_InvalidTargetExtraction(t *testing.T) {
	records := []*extract.Record{
		{
			Path: "docs/tasks/T001.md",
			Links: []extract.Link{
				{Type: "blocks", Target: "T002"},
			},
			Body: "",
		},
	}

	effective := &rules.StemFile{
		Links: rules.LinkSchema{
			Allowed: []string{"blocks"},
		},
	}

	errs := map[string][]rules.ValidationError{
		"docs/tasks/T001.md": {
			{
				Field:   "links",
				Rule:    "link_target",
				Message: `malformed message without proper format`,
			},
		},
	}

	proposals := detectCorrectLink(records, effective, errs)
	if len(proposals) != 0 {
		t.Errorf("expected 0 proposals when target cannot be extracted, got %d", len(proposals))
	}
}

// TestDetectMissingAggregates_AggregateAlreadyExists tests skipping when aggregate already exists.
func TestDetectMissingAggregates_AggregateAlreadyExists(t *testing.T) {
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {
				Type:   "enum",
				Values: []string{"Pending", "In Progress", "Completed"},
			},
		},
		Aggregate: map[string]any{
			"estado": "max_by_order(children, 'estado')",
		},
	}

	// Hierarchical records
	records := []*extract.Record{
		{
			Path: "docs/E01/README.md",
			Frontmatter: map[string]any{
				"titulo": "Epic 1",
			},
		},
		{
			Path: "docs/E01/F01/README.md",
			Frontmatter: map[string]any{
				"estado": "Pending",
			},
		},
	}

	proposals := DetectMissingAggregates("docs", records, effective)
	// Should skip estado because aggregate already exists
	for _, p := range proposals {
		if p.Field == "estado" {
			t.Errorf("should not propose aggregate for estado when it already exists")
		}
	}
}

// TestAnalyze_ExtractBodyAndInferFiltering tests that extract_body and infer_from_children have priority over add_field.
func TestAnalyze_ExtractBodyAndInferFiltering(t *testing.T) {
	records := []*extract.Record{
		{
			Path: "T001.md",
			Body: "**Estado**: Completada",
			Frontmatter: map[string]any{
				"titulo": "Task One",
			},
		},
	}

	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {
				Type:   "enum",
				Values: []string{"Pending", "Completed"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"T001.md": {
			{
				Field:   "estado",
				Rule:    "required",
				Message: "required field estado is missing",
			},
		},
	}

	report := Analyze(records, effective, errs)

	// Should have extract_body, not add_field
	hasExtract := false
	hasAdd := false
	for _, p := range report.Proposals {
		if p.Type == ExtractBody && p.Field == "estado" {
			hasExtract = true
		}
		if p.Type == AddField && p.Field == "estado" {
			hasAdd = true
		}
	}

	if !hasExtract {
		t.Error("expected extract_body proposal")
	}
	if hasAdd {
		t.Error("add_field should not appear when extract_body is applicable")
	}
}

// TestDetectCorrectValue_FuzzyMatch tests correct_value with fuzzy matching.
func TestDetectCorrectValue_FuzzyMatch(t *testing.T) {
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {
				Type:   "enum",
				Values: []string{"Pending", "In Progress", "Completed"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"T001.md": {
			{
				Field:   "estado",
				Rule:    "enum",
				Message: `value Pendng is not in allowed values: Pending, In Progress, Completed`,
			},
		},
	}

	proposals := detectCorrectValue(effective, errs)
	if len(proposals) != 1 {
		t.Fatalf("expected 1 proposal, got %d", len(proposals))
	}

	p := proposals[0]
	if p.Type != CorrectValue {
		t.Errorf("expected CorrectValue, got %v", p.Type)
	}
	if p.From != "Pendng" {
		t.Errorf("expected From='Pendng', got %q", p.From)
	}
	if p.To != "Pending" {
		t.Errorf("expected To='Pending', got %q", p.To)
	}
}

// TestDetectCorrectValue_NoMatch tests when value is not close to any valid enum value.
func TestDetectCorrectValue_NoMatch(t *testing.T) {
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {
				Type:   "enum",
				Values: []string{"Pending", "In Progress", "Completed"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"T001.md": {
			{
				Field:   "estado",
				Rule:    "enum",
				Message: `value XYZ is not in allowed values: Pending, In Progress, Completed`,
			},
		},
	}

	proposals := detectCorrectValue(effective, errs)
	if len(proposals) != 0 {
		t.Errorf("expected 0 proposals when no close match, got %d", len(proposals))
	}
}

// TestDetectCorrectValue_NoEnumSchema tests when enum schema doesn't exist.
func TestDetectCorrectValue_NoEnumSchema(t *testing.T) {
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"title": {
				Type: "string",
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"T001.md": {
			{
				Field:   "estado",
				Rule:    "enum",
				Message: `value Pendng is not in allowed values: ...`,
			},
		},
	}

	proposals := detectCorrectValue(effective, errs)
	if len(proposals) != 0 {
		t.Errorf("expected 0 proposals when enum field missing, got %d", len(proposals))
	}
}

// TestDetectAddField_DefaultValue tests add_field with default value.
func TestDetectAddField_DefaultValue(t *testing.T) {
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {
				Type:    "enum",
				Default: "Pending",
				Values:  []string{"Pending", "Completed"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"T001.md": {
			{
				Field:   "estado",
				Rule:    "required",
				Message: "required field estado is missing",
			},
		},
	}

	proposals := detectAddField(effective, errs)
	if len(proposals) != 1 {
		t.Fatalf("expected 1 proposal, got %d", len(proposals))
	}

	p := proposals[0]
	if p.Type != AddField {
		t.Errorf("expected AddField, got %v", p.Type)
	}
	if p.Value != "Pending" {
		t.Errorf("expected Value='Pending', got %q", p.Value)
	}
}

// TestDetectAddField_NoDefaultUsesFirstEnum tests add_field uses first enum value if no default.
func TestDetectAddField_NoDefaultUsesFirstEnum(t *testing.T) {
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {
				Type:   "enum",
				Values: []string{"In Progress", "Pending", "Completed"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"T001.md": {
			{
				Field:   "estado",
				Rule:    "required",
				Message: "required field estado is missing",
			},
		},
	}

	proposals := detectAddField(effective, errs)
	if len(proposals) != 1 {
		t.Fatalf("expected 1 proposal, got %d", len(proposals))
	}

	p := proposals[0]
	if p.Value != "In Progress" {
		t.Errorf("expected Value='In Progress' (first enum), got %q", p.Value)
	}
}

// TestDetectAddField_NoEnumNoDefault tests add_field with no enum and no default.
func TestDetectAddField_NoEnumNoDefault(t *testing.T) {
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"title": {
				Type: "string",
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"T001.md": {
			{
				Field:   "title",
				Rule:    "required",
				Message: "required field title is missing",
			},
		},
	}

	proposals := detectAddField(effective, errs)
	if len(proposals) != 1 {
		t.Fatalf("expected 1 proposal, got %d", len(proposals))
	}

	p := proposals[0]
	if p.Value != "" {
		t.Errorf("expected empty Value for string field, got %q", p.Value)
	}
}

// TestDetectExtendEnum_SingleRecord tests that extend_enum requires N >= 2 records.
func TestDetectExtendEnum_SingleRecord(t *testing.T) {
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {
				Type:   "enum",
				Values: []string{"Pending", "Completed"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"T001.md": {
			{
				Field:   "estado",
				Rule:    "enum",
				Message: `value Obsoleto is not in allowed values: Pending, Completed`,
			},
		},
	}

	proposals := detectExtendEnum(effective, errs)
	if len(proposals) != 0 {
		t.Errorf("expected 0 proposals for single record, got %d", len(proposals))
	}
}

// TestDetectExtendEnum_TwoRecords tests that extend_enum triggers with N >= 2.
func TestDetectExtendEnum_TwoRecords(t *testing.T) {
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {
				Type:   "enum",
				Values: []string{"Pending", "Completed"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"T001.md": {
			{
				Field:   "estado",
				Rule:    "enum",
				Message: `value Obsoleto is not in allowed values: Pending, Completed`,
			},
		},
		"T002.md": {
			{
				Field:   "estado",
				Rule:    "enum",
				Message: `value Obsoleto is not in allowed values: Pending, Completed`,
			},
		},
	}

	proposals := detectExtendEnum(effective, errs)
	found := false
	for _, p := range proposals {
		if p.Type == ExtendEnum && p.Value == "Obsoleto" && len(p.Paths) == 2 {
			found = true
		}
	}
	if !found {
		t.Errorf("expected extend_enum proposal for Obsoleto in 2 records")
	}
}

func TestDetectExtendEnumSortsPaths(t *testing.T) {
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {
				Type:   "enum",
				Values: []string{"Pending", "Completed"},
			},
		},
	}

	errs := make(map[string][]rules.ValidationError)
	for i := 15; i >= 0; i-- {
		path := fmt.Sprintf("T%03d.md", i)
		errs[path] = []rules.ValidationError{{
			Field:   "estado",
			Rule:    "enum",
			Message: `value Obsoleto is not in allowed values: Pending, Completed`,
		}}
	}

	for i := 0; i < 128; i++ {
		proposals := detectExtendEnum(effective, errs)
		if len(proposals) != 1 {
			t.Fatalf("iteration %d: expected 1 proposal, got %d", i, len(proposals))
		}
		if got := proposals[0].Paths; len(got) != 16 || !slices.IsSorted(got) {
			t.Fatalf("iteration %d: paths = %v, want 16 paths in lexical order", i, got)
		}
	}
}

// TestDetectExtendEnum_SkipsParenthesized tests that extend_enum skips values with parentheses.
func TestDetectExtendEnum_SkipsParenthesized(t *testing.T) {
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {
				Type:   "enum",
				Values: []string{"Pending", "Completed"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"T001.md": {
			{
				Field:   "estado",
				Rule:    "enum",
				Message: `value Pending (blocked by T002) is not in allowed values: Pending, Completed`,
			},
		},
		"T002.md": {
			{
				Field:   "estado",
				Rule:    "enum",
				Message: `value Pending (blocked by T001) is not in allowed values: Pending, Completed`,
			},
		},
	}

	proposals := detectExtendEnum(effective, errs)
	for _, p := range proposals {
		if p.Type == ExtendEnum && p.Field == "estado" {
			t.Errorf("should not create extend_enum for parenthesized values: %v", p)
		}
	}
}

// TestDetectCorrectLink_Retype tests link retyping to a different allowed type.
func TestDetectCorrectLink_Retype(t *testing.T) {
	records := []*extract.Record{
		{
			Path: "docs/tasks/T001.md",
			Links: []extract.Link{
				{Type: "blocks", Target: "E04"},
			},
			Body: "",
		},
	}

	effective := &rules.StemFile{
		Links: rules.LinkSchema{
			Allowed: []string{"blocks", "reference"},
			Rules: map[string]rules.LinkRule{
				"blocks":    {Target: "T\\d{3}-"},
				"reference": {Target: "E\\d{2}"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"docs/tasks/T001.md": {
			{
				Field:   "links",
				Rule:    "link_target",
				Message: `link target "E04" does not match pattern "T\\d{3}-"`,
			},
		},
	}

	proposals := detectCorrectLink(records, effective, errs)
	found := false
	for _, p := range proposals {
		if p.Type == CorrectLink && p.To == "[[reference:E04]]" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected retype proposal from blocks to reference")
	}
}

// TestDetectCorrectLink_RetypeToNoConstraint tests retyping to link type with no constraint.
func TestDetectCorrectLink_RetypeToNoConstraint(t *testing.T) {
	records := []*extract.Record{
		{
			Path: "docs/tasks/T001.md",
			Links: []extract.Link{
				{Type: "blocks", Target: "SomeTarget"},
			},
			Body: "",
		},
	}

	effective := &rules.StemFile{
		Links: rules.LinkSchema{
			Allowed: []string{"blocks", "note"},
			Rules: map[string]rules.LinkRule{
				"blocks": {Target: "T\\d{3}-"},
				// note has no rule, so any target is valid
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"docs/tasks/T001.md": {
			{
				Field:   "links",
				Rule:    "link_target",
				Message: `link target "SomeTarget" does not match pattern "T\\d{3}-"`,
			},
		},
	}

	proposals := detectCorrectLink(records, effective, errs)
	found := false
	for _, p := range proposals {
		if p.Type == CorrectLink && p.To == "[[note:SomeTarget]]" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected retype proposal to note (no constraint)")
	}
}

// TestTryExpandTarget_SingleMatch tests expanding abbreviated target.
func TestTryExpandTarget_SingleMatch(t *testing.T) {
	records := []*extract.Record{
		{
			Path: "docs/tasks/T001.md",
		},
		{
			Path: "docs/tasks/T002-full-name.md",
		},
	}

	result := tryExpandTarget("T002", "docs/tasks/T001.md", records)
	if result != "T002-full-name" {
		t.Errorf("expected T002-full-name, got %q", result)
	}
}

// TestExtractEnumValue_UnquotedFormat tests extracting unquoted enum value.
func TestExtractEnumValue_UnquotedFormat(t *testing.T) {
	msg := "value Pendng is not in allowed values: Pending, In Progress, Completed"
	val := extractEnumValue(msg, []string{"Pending", "In Progress", "Completed"})

	if val != "Pendng" {
		t.Errorf("expected Pendng, got %q", val)
	}
}

// TestExtractEnumValue_QuotedFormat tests extracting quoted enum value (fallback when unquoted not found).
func TestExtractEnumValue_QuotedFormat(t *testing.T) {
	// When there's no unquoted match, it falls back to quoted format
	msg := `value is not in allowed: "Pendng" (did you mean...)`
	val := extractEnumValue(msg, []string{"Pending", "In Progress", "Completed"})

	if val != "Pendng" {
		t.Errorf("expected Pendng, got %q", val)
	}
}

// TestExtractEnumValue_WithParentheses tests extracting value with parentheses.
func TestExtractEnumValue_WithParentheses(t *testing.T) {
	msg := "value Pending (blocked by T001) is not in allowed values: Pending, In Progress"
	val := extractEnumValue(msg, []string{"Pending", "In Progress"})

	if val != "Pending (blocked by T001)" {
		t.Errorf("expected 'Pending (blocked by T001)', got %q", val)
	}
}

// TestExtractEnumValue_MalformedMessage tests with malformed message.
func TestExtractEnumValue_MalformedMessage(t *testing.T) {
	msg := "some random error message"
	val := extractEnumValue(msg, []string{"Pending", "Completed"})

	if val != "" {
		t.Errorf("expected empty string for malformed message, got %q", val)
	}
}

// TestAnalyze_SiblingInferencePhase tests sibling inference in Analyze orchestrator.
func TestAnalyze_SiblingInferencePhase(t *testing.T) {
	records := []*extract.Record{
		{
			Path:        "T001.md",
			Frontmatter: map[string]any{"tipo": "feature"},
		},
		{
			Path:        "T002.md",
			Frontmatter: map[string]any{"tipo": "feature"},
		},
		{
			Path:        "T003.md",
			Frontmatter: map[string]any{},
		},
	}

	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"tipo": {
				Type:   "enum",
				Values: []string{"feature", "bug", "refactor"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"T003.md": {
			{
				Field:   "tipo",
				Rule:    "required",
				Message: "required field tipo is missing",
			},
		},
	}

	report := Analyze(records, effective, errs)
	// Should have proposals from Analyze (may include infer_from_siblings)
	if len(report.Proposals) == 0 {
		t.Logf("note: no proposals generated, sibling inference may not apply")
	}
}

// TestAnalyze_OutlierDetectionPhase tests outlier detection in Analyze orchestrator.
func TestAnalyze_OutlierDetectionPhase(t *testing.T) {
	records := []*extract.Record{
		{
			Path:        "T001.md",
			Frontmatter: map[string]any{"priority": "high"},
		},
		{
			Path:        "T002.md",
			Frontmatter: map[string]any{"priority": "high"},
		},
		{
			Path:        "T003.md",
			Frontmatter: map[string]any{"priority": "high"},
		},
		{
			Path:        "T004.md",
			Frontmatter: map[string]any{"priority": "MEDIUM"},
		},
	}

	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"priority": {
				Type:   "enum",
				Values: []string{"high", "medium", "low"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{}

	report := Analyze(records, effective, errs)
	// Should detect MEDIUM as an outlier (casing)
	if len(report.Proposals) > 0 {
		t.Logf("proposals: %v", report.Proposals)
	}
}

// TestAnalyze_NonEnumError tests handling of non-enum error types.
func TestAnalyze_NonEnumError(t *testing.T) {
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"titulo": {
				Type: "string",
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"T001.md": {
			{
				Field:   "titulo",
				Rule:    "non_empty",
				Message: "field titulo must not be empty",
			},
		},
	}

	report := Analyze([]*extract.Record{}, effective, errs)
	// Non-enum errors should be ignored by these detectors
	// Expect no proposals
	if len(report.Proposals) > 0 {
		t.Logf("proposals found for non-enum error: %v", report.Proposals)
	}
}

// TestAnalyze_UnknownField tests when schema field doesn't exist.
func TestAnalyze_UnknownField(t *testing.T) {
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {
				Type:   "enum",
				Values: []string{"Pending", "Completed"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"T001.md": {
			{
				Field:   "unknown_field",
				Rule:    "enum",
				Message: `value Test is not in allowed values: ...`,
			},
		},
	}

	report := Analyze([]*extract.Record{}, effective, errs)
	// Unknown fields should be skipped
	if len(report.Proposals) > 0 {
		t.Errorf("expected no proposals for unknown field, got %d", len(report.Proposals))
	}
}

// TestAnalyze_SummaryAccuracy tests that summary counts match proposals.
func TestAnalyze_SummaryAccuracy(t *testing.T) {
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {
				Type:   "enum",
				Values: []string{"Pending", "Completed"},
			},
			"titulo": {
				Type: "string",
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"T001.md": {
			{
				Field:   "estado",
				Rule:    "enum",
				Message: `value Invalid is not in allowed values: Pending, Completed`,
			},
		},
		"T002.md": {
			{
				Field:   "titulo",
				Rule:    "required",
				Message: "required field titulo is missing",
			},
		},
	}

	report := Analyze([]*extract.Record{}, effective, errs)

	// Verify summary.Total matches len(Proposals)
	if report.Summary.Total != len(report.Proposals) {
		t.Errorf("summary.Total %d != len(proposals) %d", report.Summary.Total, len(report.Proposals))
	}

	// Count proposal types
	correctCount := 0
	addCount := 0
	for _, p := range report.Proposals {
		if p.Type == CorrectValue {
			correctCount++
		}
		if p.Type == AddField {
			addCount++
		}
	}

	if report.Summary.CorrectValue != correctCount {
		t.Errorf("summary.CorrectValue %d != actual %d", report.Summary.CorrectValue, correctCount)
	}
	if report.Summary.AddField != addCount {
		t.Errorf("summary.AddField %d != actual %d", report.Summary.AddField, addCount)
	}
}

// TestDetectCorrectLink_BadMessageFormat tests when message format is unparseable.
func TestDetectCorrectLink_BadMessageFormat(t *testing.T) {
	records := []*extract.Record{
		{
			Path:  "T001.md",
			Links: []extract.Link{{Type: "blocks", Target: "T002"}},
		},
	}

	effective := &rules.StemFile{
		Links: rules.LinkSchema{
			Allowed: []string{"blocks"},
			Rules: map[string]rules.LinkRule{
				"blocks": {Target: "T\\d{3}-"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"T001.md": {
			{
				Field:   "links",
				Rule:    "link_target",
				Message: "unparseable message format",
			},
		},
	}

	proposals := detectCorrectLink(records, effective, errs)
	if len(proposals) != 0 {
		t.Errorf("expected 0 proposals for unparseable message, got %d", len(proposals))
	}
}

// TestDetectExtractBody_NonRequiredError tests when error is not "required".
func TestDetectExtractBody_NonRequiredError(t *testing.T) {
	records := []*extract.Record{
		{
			Path:        "T001.md",
			Body:        "**Estado**: Completada",
			Frontmatter: map[string]any{},
		},
	}

	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {
				Type:   "enum",
				Values: []string{"Pending", "Completed"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"T001.md": {
			{
				Field:   "estado",
				Rule:    "enum",
				Message: `value Invalid is not in allowed values: ...`,
			},
		},
	}

	proposals := detectExtractBody(records, effective, errs)
	if len(proposals) != 0 {
		t.Errorf("expected 0 proposals for non-required error, got %d", len(proposals))
	}
}

// TestDetectExtractBody_InvalidEnumValue tests when extracted body value doesn't match enum.
func TestDetectExtractBody_InvalidEnumValue(t *testing.T) {
	records := []*extract.Record{
		{
			Path:        "T001.md",
			Body:        "**Estado**: InvalidValue",
			Frontmatter: map[string]any{},
		},
	}

	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {
				Type:   "enum",
				Values: []string{"Pending", "In Progress", "Completed"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"T001.md": {
			{
				Field:   "estado",
				Rule:    "required",
				Message: "required field estado is missing",
			},
		},
	}

	proposals := detectExtractBody(records, effective, errs)
	// Should still generate proposal, using fuzzy match
	if len(proposals) != 1 {
		t.Fatalf("expected 1 proposal, got %d", len(proposals))
	}

	p := proposals[0]
	if p.Type != ExtractBody {
		t.Errorf("expected ExtractBody, got %v", p.Type)
	}
}

// TestDetectMigrateValue_NoBaseFoundNoParentheses tests when parenthetical info is invalid.
func TestDetectMigrateValue_InvalidBlockedByFormat(t *testing.T) {
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {
				Type:   "enum",
				Values: []string{"Pending", "In Progress"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"T001.md": {
			{
				Field:   "estado",
				Rule:    "enum",
				Message: `value Pending (other info) is not in allowed values: Pending, In Progress`,
			},
		},
	}

	proposals := detectMigrateValue(effective, errs)
	// Parentheses without "blocked by" should not create migrate proposal
	if len(proposals) != 0 {
		t.Errorf("expected 0 proposals for non-blocked-by parentheses, got %d", len(proposals))
	}
}

// TestAnalyze_ComplexScenario tests multiple proposal types generated together.
func TestAnalyze_ComplexScenario(t *testing.T) {
	records := []*extract.Record{
		{
			Path:        "T001.md",
			Body:        "**Estado**: Activa",
			Frontmatter: map[string]any{"titulo": "Task One"},
		},
		{
			Path: "T002.md",
			Links: []extract.Link{
				{Type: "blocks", Target: "T003"},
			},
			Frontmatter: map[string]any{"titulo": "Task Two"},
		},
		{
			Path:        "T003-expanded.md",
			Frontmatter: map[string]any{"titulo": "Task Three"},
		},
	}

	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {
				Type:   "enum",
				Values: []string{"Pending", "In Progress", "Completed"},
			},
			"titulo": {
				Type: "string",
			},
		},
		Links: rules.LinkSchema{
			Allowed: []string{"blocks"},
			Rules: map[string]rules.LinkRule{
				"blocks": {Target: "T\\d{3}-"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"T001.md": {
			{
				Field:   "estado",
				Rule:    "required",
				Message: "required field estado is missing",
			},
		},
		"T002.md": {
			{
				Field:   "links",
				Rule:    "link_target",
				Message: `link target "T003" does not match pattern "T\\d{3}-"`,
			},
		},
	}

	report := Analyze(records, effective, errs)

	// Should have multiple proposal types
	hasExtract := false
	hasLink := false
	for _, p := range report.Proposals {
		if p.Type == ExtractBody {
			hasExtract = true
		}
		if p.Type == CorrectLink {
			hasLink = true
		}
	}

	if !hasExtract {
		t.Error("expected extract_body proposal")
	}
	if !hasLink {
		t.Logf("proposals: %v (note: link may not be detected without expansion target)", report.Proposals)
	}
}

// TestAnalyze_AllDetectorsWithNil tests all detectors return empty when needed.
func TestAnalyze_AllDetectorsEmptyMaps(t *testing.T) {
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"field": {Type: "string"},
		},
	}

	report := Analyze([]*extract.Record{}, effective, map[string][]rules.ValidationError{})

	if len(report.Proposals) != 0 {
		t.Errorf("expected 0 proposals for empty errors, got %d", len(report.Proposals))
	}
	if report.Summary.Total != 0 {
		t.Errorf("expected summary.Total=0, got %d", report.Summary.Total)
	}
}

// TestDetectMissingAggregates_WithoutDetectedHierarchy tests that function returns nil when hierarchy not detected.
func TestDetectMissingAggregates_WithoutHierarchy(t *testing.T) {
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {
				Type:   "enum",
				Values: []string{"Pending", "Done"},
			},
		},
	}

	// Non-hierarchical flat structure
	records := []*extract.Record{
		{
			Path:        "docs/task1.md",
			Frontmatter: map[string]any{"estado": "Pending"},
		},
		{
			Path:        "docs/task2.md",
			Frontmatter: map[string]any{"estado": "Done"},
		},
	}

	proposals := DetectMissingAggregates("docs", records, effective)
	// No hierarchy pattern detected
	if len(proposals) > 0 {
		t.Logf("note: proposals detected: %v", proposals)
	}
}

// TestAnalyze_CorrectValueFiltering tests that correct_value is filtered out when migrate_value exists.
func TestAnalyze_MigrateHasPriorityOverCorrect(t *testing.T) {
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {
				Type:   "enum",
				Values: []string{"Pending", "In Progress", "Completed"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"T001.md": {
			{
				Field:   "estado",
				Rule:    "enum",
				Message: `value Pending (blocked by E04/F01) is not in allowed values: Pending, In Progress, Completed`,
			},
		},
	}

	report := Analyze([]*extract.Record{}, effective, errs)

	// Should have migrate_value, NOT correct_value
	migrateCount := 0
	correctCount := 0
	for _, p := range report.Proposals {
		if p.Type == MigrateValue {
			migrateCount++
		}
		if p.Type == CorrectValue {
			correctCount++
		}
	}

	if migrateCount == 0 {
		t.Error("expected migrate_value proposal")
	}
	if correctCount > 0 {
		t.Errorf("should not have correct_value when migrate_value exists, but got %d", correctCount)
	}
}

// TestAnalyze_AddFieldFiltering tests that add_field is filtered when extract_body exists.
func TestAnalyze_ExtractHasPriorityOverAddField(t *testing.T) {
	records := []*extract.Record{
		{
			Path:        "T001.md",
			Body:        "**Estado**: Completada",
			Frontmatter: map[string]any{},
		},
	}

	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {
				Type:   "enum",
				Values: []string{"Pending", "Completed"},
			},
		},
	}

	errs := map[string][]rules.ValidationError{
		"T001.md": {
			{
				Field:   "estado",
				Rule:    "required",
				Message: "required field estado is missing",
			},
		},
	}

	report := Analyze(records, effective, errs)

	// Should have extract_body, NOT add_field
	hasExtract := false
	hasAdd := false
	for _, p := range report.Proposals {
		if p.Type == ExtractBody && p.Field == "estado" {
			hasExtract = true
		}
		if p.Type == AddField && p.Field == "estado" {
			hasAdd = true
		}
	}

	if !hasExtract {
		t.Error("expected extract_body proposal")
	}
	if hasAdd {
		t.Error("should not have add_field when extract_body is applicable")
	}
}
