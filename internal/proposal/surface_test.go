package proposal

import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestProposalSurface_ExtendEnum(t *testing.T) {
	p := &Proposal{
		Type:  ExtendEnum,
		Field: "estado",
		Value: "Obsoleto",
		Paths: []string{"a.md", "b.md"},
	}
	got := p.Surface()
	if got != SurfaceSchema {
		t.Errorf("ExtendEnum.Surface() = %q, want schema", got)
	}
}

func TestProposalSurface_CorrectValue(t *testing.T) {
	p := &Proposal{
		Type:  CorrectValue,
		Field: "estado",
		From:  "Completo",
		To:    "Completed",
		Paths: []string{"a.md"},
	}
	got := p.Surface()
	if got != SurfaceRepair {
		t.Errorf("CorrectValue.Surface() = %q, want repair", got)
	}
}

func TestProposalSurface_AddField(t *testing.T) {
	p := &Proposal{
		Type:  AddField,
		Field: "estado",
		Value: "Pending",
		Paths: []string{"a.md"},
	}
	got := p.Surface()
	if got != SurfaceRepair {
		t.Errorf("AddField.Surface() = %q, want repair", got)
	}
}

func TestProposalSurface_MigrateValue(t *testing.T) {
	p := &Proposal{
		Type:  MigrateValue,
		Field: "estado",
		From:  "Pending (blocked by T001)",
		To:    "Pending",
		Paths: []string{"a.md"},
	}
	got := p.Surface()
	if got != SurfaceRepair {
		t.Errorf("MigrateValue.Surface() = %q, want repair", got)
	}
}

func TestProposalSurface_ExtractBody(t *testing.T) {
	p := &Proposal{
		Type:  ExtractBody,
		Field: "estado",
		From:  "Completada",
		To:    "Completed",
		Paths: []string{"a.md"},
	}
	got := p.Surface()
	if got != SurfaceRepair {
		t.Errorf("ExtractBody.Surface() = %q, want repair", got)
	}
}

func TestProposalSurface_InferFromChildren(t *testing.T) {
	p := &Proposal{
		Type:  InferFromChildren,
		Field: "estado",
		Value: "In Progress",
		Paths: []string{"dir/README.md"},
	}
	got := p.Surface()
	if got != SurfaceRepair {
		t.Errorf("InferFromChildren.Surface() = %q, want repair", got)
	}
}

func TestProposalSurface_InferFromSiblings(t *testing.T) {
	p := &Proposal{
		Type:  InferFromSiblings,
		Field: "tipo",
		Value: "feature",
		Paths: []string{"a.md"},
	}
	got := p.Surface()
	if got != SurfaceRepair {
		t.Errorf("InferFromSiblings.Surface() = %q, want repair", got)
	}
}

func TestProposalSurface_AddAggregate(t *testing.T) {
	p := &Proposal{
		Type:          AddAggregate,
		Field:         "estado",
		AggregateExpr: "max(child.estado)",
		Paths:         []string{".stem"},
	}
	got := p.Surface()
	if got != SurfaceSchema {
		t.Errorf("AddAggregate.Surface() = %q, want schema", got)
	}
}

func TestProposalSurface_RemoveStemField(t *testing.T) {
	p := &Proposal{
		Type:  RemoveStemField,
		Field: "estado",
		Paths: []string{"child/.stem"},
	}
	got := p.Surface()
	if got != SurfaceSchema {
		t.Errorf("RemoveStemField.Surface() = %q, want schema", got)
	}
}

func TestProposalSurface_CorrectLink(t *testing.T) {
	p := &Proposal{
		Type:  CorrectLink,
		Field: "links",
		From:  "[[blocks:T001]]",
		To:    "[[blocks:T001-add-feature]]",
		Paths: []string{"a.md"},
	}
	got := p.Surface()
	if got != SurfaceRepair {
		t.Errorf("CorrectLink.Surface() = %q, want repair", got)
	}
}

func TestProposalSurface_PropagateAggregate(t *testing.T) {
	p := &Proposal{
		Type:  PropagateAggregate,
		Field: "estado",
		From:  "Pending",
		To:    "In Progress",
		Paths: []string{"dir/README.md"},
	}
	got := p.Surface()
	if got != SurfaceRepair {
		t.Errorf("PropagateAggregate.Surface() = %q, want repair", got)
	}
}

func TestProposalSurface_CorrectOutlier(t *testing.T) {
	p := &Proposal{
		Type:  CorrectOutlier,
		Field: "estado",
		From:  "Unknown",
		To:    "Pending",
		Paths: []string{"a.md"},
	}
	got := p.Surface()
	if got != SurfaceRepair {
		t.Errorf("CorrectOutlier.Surface() = %q, want repair", got)
	}
}

func TestProposalSurface_SetField(t *testing.T) {
	p := &Proposal{
		Type:  SetField,
		Field: "investigacion",
		Value: "DECISION: GO",
		Paths: []string{"a.md"},
	}
	got := p.Surface()
	if got != SurfaceRepair {
		t.Errorf("SetField.Surface() = %q, want repair", got)
	}
}

func TestProposalSurface_SetSection(t *testing.T) {
	p := &Proposal{
		Type:    SetSection,
		Field:   "investigacion",
		Value:   "New content",
		Heading: "## Investigación",
		Mode:    "replace",
		Paths:   []string{"a.md"},
	}
	got := p.Surface()
	if got != SurfaceRepair {
		t.Errorf("SetSection.Surface() = %q, want repair", got)
	}
}

func TestProposalSurface_UnknownType(t *testing.T) {
	p := &Proposal{
		Type:  Type("unknown_future_type"),
		Field: "something",
		Paths: []string{"a.md"},
	}
	got := p.Surface()
	if got != SurfaceDiagnostic {
		t.Errorf("unknown type.Surface() = %q, want diagnostic (conservative default)", got)
	}
}

// Coverage test: ensure all defined proposal types map to a surface
func TestAllProposalTypesCovered(t *testing.T) {
	types := []Type{
		ExtendEnum,
		MigrateValue,
		CorrectValue,
		ExtractBody,
		InferFromChildren,
		AddField,
		CorrectLink,
		InferFromSiblings,
		CorrectOutlier,
		AddAggregate,
		RemoveStemField,
		PropagateAggregate,
		SetField,
		SetSection,
		SchemaEvolution,
		RemoveField,
		LooseRequired,
		ChangeType,
		ReplaceEnumValues,
		LoosenSeverity,
	}

	for _, typ := range types {
		p := &Proposal{Type: typ, Field: "test", Paths: []string{"test.md"}}
		surface := p.Surface()

		// Verify that surface is one of the expected values
		if surface != SurfaceSchema && surface != SurfaceRepair && surface != SurfaceBootstrap &&
			surface != SurfaceMigration && surface != SurfaceDiagnostic && surface != SurfaceRequiresAgent {
			t.Errorf("type %q returned invalid surface %q", typ, surface)
		}

		// Verify non-diagnostic classification for known types
		if surface == SurfaceDiagnostic {
			t.Errorf("type %q should not default to diagnostic", typ)
		}
	}
}

func TestSurfaceString(t *testing.T) {
	tests := []struct {
		s    ProposalSurface
		want string
	}{
		{SurfaceSchema, "schema"},
		{SurfaceRepair, "repair"},
		{SurfaceBootstrap, "bootstrap"},
		{SurfaceMigration, "migration"},
		{SurfaceDiagnostic, "diagnostic"},
		{SurfaceRequiresAgent, "requires_agent"},
	}

	for _, tt := range tests {
		got := tt.s.String()
		if got != tt.want {
			t.Errorf("%q.String() = %q, want %q", tt.s, got, tt.want)
		}
	}
}

// Integration test: verify surface classification in analysis context
func TestAnalyzedProposals_HaveSurfaceAssigned(t *testing.T) {
	// After Analyze(), proposals should be classified by surface
	// when Surface() is called.
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Completed"}, Required: true},
		},
	}

	errs := map[string][]rules.ValidationError{
		"a.md": {
			{Rule: "enum", Field: "estado", Message: `value "Obsoleto" not in allowed values: [Pending, Completed]`},
		},
		"b.md": {
			{Rule: "required", Field: "estado", Message: `required field "estado" is missing`},
		},
	}

	report := Analyze([]*extract.Record{}, stem, errs)

	if len(report.Proposals) == 0 {
		t.Fatalf("expected proposals, got 0")
	}

	for _, p := range report.Proposals {
		surface := p.Surface()
		if surface == "" {
			t.Errorf("proposal of type %q has empty surface", p.Type)
		}
	}
}

// Test specific inference types from the analyze command
// These represent governance and data inference categories
func TestGoverneranceInference_ImplicitSchema_DiagnosticSurface(t *testing.T) {
	// implicit_schema is a governance inference (needs agent review)
	// Even though it's not a Proposal type, we verify the Surface enum
	// can handle unknown types conservatively
	unknownType := Type("implicit_schema")
	p := &Proposal{
		Type:  unknownType,
		Field: "campo",
		Paths: []string{".stem"},
	}
	surface := p.Surface()
	if surface != SurfaceDiagnostic {
		t.Errorf("implicit_schema should map to diagnostic, got %q", surface)
	}
}

func TestSurfaceClassificationMatrixRepair(t *testing.T) {
	// Test matrix of repair proposals (data surface only)
	repairTypes := []Type{
		CorrectValue,
		MigrateValue,
		AddField,
		ExtractBody,
		InferFromChildren,
		InferFromSiblings,
		CorrectOutlier,
		CorrectLink,
		PropagateAggregate,
		SetField,
		SetSection,
	}

	for _, typ := range repairTypes {
		p := &Proposal{Type: typ, Field: "test", Paths: []string{"a.md"}}
		if surface := p.Surface(); surface != SurfaceRepair {
			t.Errorf("type %q should be repair, got %q", typ, surface)
		}
	}
}

func TestSurfaceClassificationMatrixSchema(t *testing.T) {
	// Test matrix of schema proposals (.stem mutation)
	schemaTypes := []Type{
		ExtendEnum,
		AddAggregate,
		RemoveStemField,
	}

	for _, typ := range schemaTypes {
		p := &Proposal{Type: typ, Field: "test", Paths: []string{".stem"}}
		if surface := p.Surface(); surface != SurfaceSchema {
			t.Errorf("type %q should be schema, got %q", typ, surface)
		}
	}
}

func TestSurfaceClassificationSchemaEvolution(t *testing.T) {
	// Test matrix of schema evolution proposals (migration surface, not schema surface)
	evolutionTypes := []Type{
		SchemaEvolution,
		RemoveField,
		LooseRequired,
		ChangeType,
		ReplaceEnumValues,
		LoosenSeverity,
	}

	for _, typ := range evolutionTypes {
		p := &Proposal{
			Type:  typ,
			Field: "test",
			Paths: []string{".stem"},
		}
		if surface := p.Surface(); surface != SurfaceMigration {
			t.Errorf("type %q should be migration, got %q", typ, surface)
		}
	}
}

func TestProposalSurface_SchemaEvolution(t *testing.T) {
	p := &Proposal{
		Type:          SchemaEvolution,
		Field:         "legacy_field",
		Description:   "explicit schema evolution: remove deprecated field",
		Paths:         []string{".stem"},
		MigrationNote: "field superseded by new_field, migration complete",
	}
	got := p.Surface()
	if got != SurfaceMigration {
		t.Errorf("SchemaEvolution.Surface() = %q, want migration", got)
	}
}

func TestProposalSurface_RemoveField(t *testing.T) {
	p := &Proposal{
		Type:          RemoveField,
		Field:         "deprecated_status",
		Description:   "remove field from schema",
		Paths:         []string{".stem"},
		MigrationNote: "deprecated field no longer used",
	}
	got := p.Surface()
	if got != SurfaceMigration {
		t.Errorf("RemoveField.Surface() = %q, want migration", got)
	}
}

func TestProposalSurface_LooseRequired(t *testing.T) {
	p := &Proposal{
		Type:          LooseRequired,
		Field:         "old_status",
		Description:   "loosen required constraint",
		Paths:         []string{".stem"},
		From:          "true",
		To:            "false",
		MigrationNote: "field becoming optional to support legacy records",
	}
	got := p.Surface()
	if got != SurfaceMigration {
		t.Errorf("LooseRequired.Surface() = %q, want migration", got)
	}
}

func TestProposalSurface_ChangeType(t *testing.T) {
	p := &Proposal{
		Type:          ChangeType,
		Field:         "priority",
		Description:   "change field type",
		Paths:         []string{".stem"},
		From:          "enum",
		To:            "integer",
		MigrationNote: "numeric priority system replacing enum",
	}
	got := p.Surface()
	if got != SurfaceMigration {
		t.Errorf("ChangeType.Surface() = %q, want migration", got)
	}
}

func TestProposalSurface_ReplaceEnumValues(t *testing.T) {
	p := &Proposal{
		Type:          ReplaceEnumValues,
		Field:         "estado",
		Description:   "replace enum values",
		Paths:         []string{".stem"},
		From:          "pending,active,done",
		To:            "queued,running,finished",
		MigrationNote: "replacing legacy state names with new standard",
	}
	got := p.Surface()
	if got != SurfaceMigration {
		t.Errorf("ReplaceEnumValues.Surface() = %q, want migration", got)
	}
}

func TestProposalSurface_LoosenSeverity(t *testing.T) {
	p := &Proposal{
		Type:          LoosenSeverity,
		Field:         "critical_status",
		Description:   "reduce severity level",
		Paths:         []string{".stem"},
		From:          "critical",
		To:            "warning",
		MigrationNote: "field no longer critical in current workflows",
	}
	got := p.Surface()
	if got != SurfaceMigration {
		t.Errorf("LoosenSeverity.Surface() = %q, want migration", got)
	}
}
