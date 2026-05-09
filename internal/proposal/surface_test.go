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
	if got != SurfaceMigration {
		t.Errorf("MigrateValue.Surface() = %q, want migration", got)
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

func TestSurfaceClassificationMigration(t *testing.T) {
	// Only MigrateValue triggers migration surface
	p := &Proposal{
		Type:  MigrateValue,
		Field: "estado",
		From:  "old",
		To:    "new",
		Paths: []string{"a.md"},
	}
	if surface := p.Surface(); surface != SurfaceMigration {
		t.Errorf("MigrateValue should be migration, got %q", surface)
	}
}
