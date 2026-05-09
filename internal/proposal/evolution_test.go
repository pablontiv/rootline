package proposal

import (
	"testing"

	"github.com/pablontiv/rootline/internal/migrate"
)

func TestBreakingChangeToEvolutionProposal_FieldRemoved(t *testing.T) {
	change := migrate.Change{
		Kind:    migrate.FieldRemoved,
		Field:   "legacy_field",
		Before:  "string",
		Message: "field legacy_field removed",
	}

	prop := BreakingChangeToEvolutionProposal(change, ".stem")

	if prop.Type != RemoveField {
		t.Errorf("expected RemoveField, got %q", prop.Type)
	}
	if prop.Field != "legacy_field" {
		t.Errorf("expected field legacy_field, got %q", prop.Field)
	}
	if prop.Surface() != SurfaceMigration {
		t.Errorf("RemoveField proposal should be migration surface, got %q", prop.Surface())
	}
	if len(prop.Paths) != 1 || prop.Paths[0] != ".stem" {
		t.Errorf("expected paths [.stem], got %v", prop.Paths)
	}
}

func TestBreakingChangeToEvolutionProposal_RequiredRemoved(t *testing.T) {
	change := migrate.Change{
		Kind:    migrate.RequiredRemoved,
		Field:   "status",
		Before:  "true",
		After:   "false",
		Message: "field status is no longer required",
	}

	prop := BreakingChangeToEvolutionProposal(change, ".stem")

	if prop.Type != LooseRequired {
		t.Errorf("expected LooseRequired, got %q", prop.Type)
	}
	if prop.From != "true" || prop.To != "false" {
		t.Errorf("expected from=true to=false, got from=%q to=%q", prop.From, prop.To)
	}
	if prop.Surface() != SurfaceMigration {
		t.Errorf("LooseRequired proposal should be migration surface, got %q", prop.Surface())
	}
}

func TestBreakingChangeToEvolutionProposal_TypeChanged(t *testing.T) {
	change := migrate.Change{
		Kind:    migrate.TypeChanged,
		Field:   "priority",
		Before:  "enum",
		After:   "integer",
		Message: "field priority type changed from enum to integer",
	}

	prop := BreakingChangeToEvolutionProposal(change, ".stem")

	if prop.Type != ChangeType {
		t.Errorf("expected ChangeType, got %q", prop.Type)
	}
	if prop.From != "enum" || prop.To != "integer" {
		t.Errorf("expected from=enum to=integer, got from=%q to=%q", prop.From, prop.To)
	}
	if prop.Surface() != SurfaceMigration {
		t.Errorf("ChangeType proposal should be migration surface, got %q", prop.Surface())
	}
}

func TestBreakingChangeToEvolutionProposal_EnumValueRemoved(t *testing.T) {
	change := migrate.Change{
		Kind:    migrate.EnumValueRemoved,
		Field:   "estado",
		Before:  "pending",
		Message: "enum value pending removed from estado",
	}

	prop := BreakingChangeToEvolutionProposal(change, ".stem")

	if prop.Type != ReplaceEnumValues {
		t.Errorf("expected ReplaceEnumValues, got %q", prop.Type)
	}
	if prop.Surface() != SurfaceMigration {
		t.Errorf("ReplaceEnumValues proposal should be migration surface, got %q", prop.Surface())
	}
}

func TestBreakingChangeToEvolutionProposal_SeverityChanged(t *testing.T) {
	change := migrate.Change{
		Kind:    migrate.SeverityChanged,
		Field:   "critical",
		Before:  "critical",
		After:   "warning",
		Message: "field critical severity changed from critical to warning",
	}

	prop := BreakingChangeToEvolutionProposal(change, ".stem")

	if prop.Type != LoosenSeverity {
		t.Errorf("expected LoosenSeverity, got %q", prop.Type)
	}
	if prop.From != "critical" || prop.To != "warning" {
		t.Errorf("expected from=critical to=warning, got from=%q to=%q", prop.From, prop.To)
	}
	if prop.Surface() != SurfaceMigration {
		t.Errorf("LoosenSeverity proposal should be migration surface, got %q", prop.Surface())
	}
}

func TestDiffToEvolutionReport_NoBreakingChanges(t *testing.T) {
	diff := &migrate.DiffResult{
		Version:  1,
		Kind:     "rootline/migrate-diff",
		StemPath: ".stem",
		Changes: []migrate.Change{
			{
				Kind:     migrate.FieldAdded,
				Field:    "new_field",
				Breaking: false,
				After:    "string",
				Message:  "field new_field added",
			},
		},
		BreakingCount: 0,
		TotalCount:    1,
	}

	report := DiffToEvolutionReport(diff)

	if len(report.Proposals) != 0 {
		t.Errorf("expected 0 proposals for non-breaking changes, got %d", len(report.Proposals))
	}
	if report.Summary.Total != 0 {
		t.Errorf("expected summary total 0, got %d", report.Summary.Total)
	}
}

func TestDiffToEvolutionReport_MultipleBreakingChanges(t *testing.T) {
	diff := &migrate.DiffResult{
		Version:  1,
		Kind:     "rootline/migrate-diff",
		StemPath: ".stem",
		Changes: []migrate.Change{
			{
				Kind:     migrate.FieldRemoved,
				Field:    "old_status",
				Breaking: true,
				Before:   "enum",
				Message:  "field old_status removed",
			},
			{
				Kind:     migrate.RequiredRemoved,
				Field:    "version",
				Breaking: false,
				Before:   "true",
				After:    "false",
				Message:  "field version is no longer required",
			},
			{
				Kind:     migrate.TypeChanged,
				Field:    "priority",
				Breaking: true,
				Before:   "string",
				After:    "integer",
				Message:  "field priority type changed",
			},
			{
				Kind:     migrate.FieldAdded,
				Field:    "new_id",
				Breaking: false,
				After:    "string",
				Message:  "field new_id added",
			},
		},
		BreakingCount: 2,
		TotalCount:    4,
	}

	report := DiffToEvolutionReport(diff)

	// Should have 3 proposals: 2 breaking + 1 non-breaking that's non-breaking
	// Wait, no — the function only extracts breaking changes
	if len(report.Proposals) != 2 {
		t.Errorf("expected 2 proposals for 2 breaking changes, got %d", len(report.Proposals))
	}

	if report.Summary.Total != 2 {
		t.Errorf("expected summary total 2, got %d", report.Summary.Total)
	}

	if report.Summary.RemoveField != 1 {
		t.Errorf("expected 1 RemoveField proposal, got %d", report.Summary.RemoveField)
	}

	if report.Summary.ChangeType != 1 {
		t.Errorf("expected 1 ChangeType proposal, got %d", report.Summary.ChangeType)
	}
}

func TestDiffToEvolutionReport_OnlyExtractsBreakingChanges(t *testing.T) {
	diff := &migrate.DiffResult{
		Version:  1,
		Kind:     "rootline/migrate-diff",
		StemPath: ".stem",
		Changes: []migrate.Change{
			{
				Kind:     migrate.FieldRemoved,
				Field:    "deprecated",
				Breaking: true,
				Before:   "string",
				Message:  "field deprecated removed",
			},
			{
				Kind:     migrate.FieldAdded,
				Field:    "new_field",
				Breaking: false,
				After:    "string",
				Message:  "field new_field added",
			},
			{
				Kind:     migrate.EnumValueAdded,
				Field:    "status",
				Breaking: false,
				After:    "archived",
				Message:  "enum value archived added",
			},
			{
				Kind:     migrate.EnumValueRemoved,
				Field:    "status",
				Breaking: true,
				Before:   "unknown",
				Message:  "enum value unknown removed",
			},
		},
		BreakingCount: 2,
		TotalCount:    4,
	}

	report := DiffToEvolutionReport(diff)

	// Should have exactly 2 proposals (both breaking changes)
	if len(report.Proposals) != 2 {
		t.Errorf("expected 2 proposals for 2 breaking changes, got %d", len(report.Proposals))
	}

	// Verify types
	if report.Proposals[0].Type != RemoveField {
		t.Errorf("expected first proposal to be RemoveField, got %q", report.Proposals[0].Type)
	}
	if report.Proposals[1].Type != ReplaceEnumValues {
		t.Errorf("expected second proposal to be ReplaceEnumValues, got %q", report.Proposals[1].Type)
	}
}

func TestDiffToEvolutionReport_ValidJSON(t *testing.T) {
	diff := &migrate.DiffResult{
		Version:  1,
		Kind:     "rootline/migrate-diff",
		StemPath: ".stem",
		Changes: []migrate.Change{
			{
				Kind:     migrate.FieldRemoved,
				Field:    "legacy",
				Breaking: true,
				Before:   "string",
				Message:  "field legacy removed",
			},
		},
		BreakingCount: 1,
		TotalCount:    1,
	}

	report := DiffToEvolutionReport(diff)

	if report.Version != 1 {
		t.Errorf("expected version 1, got %d", report.Version)
	}

	if report.Kind != "rootline/schema-evolution-proposals" {
		t.Errorf("expected kind rootline/schema-evolution-proposals, got %q", report.Kind)
	}

	if len(report.Proposals) != 1 {
		t.Errorf("expected 1 proposal, got %d", len(report.Proposals))
	}

	if report.Proposals[0].MigrationNote == "" {
		t.Errorf("expected migration note to be populated")
	}
}

func TestEvolutionProposal_RejectsSchemaApply(t *testing.T) {
	// Schema evolution proposals should NOT be applied by `schema apply`
	// because they are migration-class, not schema-class.
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

		// Verify surface is migration, not schema
		if p.Surface() != SurfaceMigration {
			t.Errorf("type %q should be migration surface, got %q", typ, p.Surface())
		}

		// Verify surface is NOT schema (schema_apply should reject migration-class)
		if p.Surface() == SurfaceSchema {
			t.Errorf("type %q should not be schema surface", typ)
		}
	}
}

func TestEvolutionProposal_HasMigrationNote(t *testing.T) {
	// All evolution proposals should include migration rationale
	changes := []migrate.Change{
		{
			Kind:     migrate.FieldRemoved,
			Field:    "deprecated",
			Breaking: true,
			Before:   "string",
			Message:  "field removed",
		},
		{
			Kind:     migrate.RequiredRemoved,
			Field:    "status",
			Breaking: false,
			Before:   "true",
			After:    "false",
			Message:  "required loosened",
		},
		{
			Kind:     migrate.TypeChanged,
			Field:    "priority",
			Breaking: true,
			Before:   "enum",
			After:    "integer",
			Message:  "type changed",
		},
	}

	for _, change := range changes {
		prop := BreakingChangeToEvolutionProposal(change, ".stem")
		if prop.MigrationNote == "" {
			t.Errorf("proposal for change %q should have migration note", change.Field)
		}
	}
}
