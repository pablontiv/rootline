package proposal

import (
	"github.com/pablontiv/rootline/internal/migrate"
)

// BreakingChangeToEvolutionProposal converts a breaking change from a migrate diff
// into a schema evolution proposal.
// This allows schema changes that would normally be rejected under monotonic mode
// to be explicitly represented as intentional migrations with rationale.
func BreakingChangeToEvolutionProposal(change migrate.Change, stemPath string) *Proposal {
	p := &Proposal{
		Field:       change.Field,
		Paths:       []string{stemPath},
		Description: change.Message,
	}

	// Map migrate ChangeKind to evolution proposal Type based on the breaking change
	switch change.Kind {
	case migrate.FieldRemoved:
		p.Type = RemoveField
		p.MigrationNote = "field removed from schema during migration"

	case migrate.RequiredRemoved:
		p.Type = LooseRequired
		p.From = change.Before
		p.To = change.After
		p.MigrationNote = "required constraint loosened to support legacy records"

	case migrate.TypeChanged:
		p.Type = ChangeType
		p.From = change.Before
		p.To = change.After
		p.MigrationNote = "field type changed during schema evolution"

	case migrate.EnumValueRemoved:
		p.Type = ReplaceEnumValues
		p.MigrationNote = "enum value removed during migration; may require data repair"

	case migrate.SeverityChanged:
		p.Type = LoosenSeverity
		p.From = change.Before
		p.To = change.After
		p.MigrationNote = "validation severity reduced"

	default:
		// Fallback: mark as generic schema evolution
		p.Type = SchemaEvolution
		p.MigrationNote = "breaking change requires explicit evolution approval"
	}

	return p
}

// DiffToEvolutionReport converts breaking changes from a migrate diff into a proposal Report
// containing schema evolution proposals for explicit review and approval.
func DiffToEvolutionReport(diff *migrate.DiffResult) *Report {
	var proposals []Proposal

	// Extract only breaking changes and convert to evolution proposals
	for _, change := range diff.Changes {
		if !change.Breaking {
			continue // skip non-breaking changes
		}

		prop := BreakingChangeToEvolutionProposal(change, diff.StemPath)
		proposals = append(proposals, *prop)
	}

	// Build summary
	summary := Summary{Total: len(proposals)}
	for _, p := range proposals {
		switch p.Type {
		case SchemaEvolution:
			summary.SchemaEvolution++
		case RemoveField:
			summary.RemoveField++
		case LooseRequired:
			summary.LooseRequired++
		case ChangeType:
			summary.ChangeType++
		case ReplaceEnumValues:
			summary.ReplaceEnumValues++
		case LoosenSeverity:
			summary.LoosenSeverity++
		}
	}

	return &Report{
		Version:   1,
		Kind:      "rootline/schema-evolution-proposals",
		Proposals: proposals,
		Summary:   summary,
	}
}
