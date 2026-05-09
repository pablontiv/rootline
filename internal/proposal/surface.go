package proposal

// ProposalSurface classifies which system component handles a proposal.
// This allows engines to classify proposals without inspecting command context.
type ProposalSurface string

const (
	// SurfaceSchema proposals mutate .stem files (schema evolution).
	SurfaceSchema ProposalSurface = "schema"

	// SurfaceRepair proposals mutate Markdown document frontmatter only (data corrections).
	SurfaceRepair ProposalSurface = "repair"

	// SurfaceBootstrap proposals scaffold missing .stem files for new directories.
	SurfaceBootstrap ProposalSurface = "bootstrap"

	// SurfaceMigration proposals handle bulk field renames or type changes across records.
	SurfaceMigration ProposalSurface = "migration"

	// SurfaceDiagnostic proposals are read-only observations (e.g., governance detectors)
	// that should not be automatically applied without agent review.
	SurfaceDiagnostic ProposalSurface = "diagnostic"

	// SurfaceRequiresAgent proposals need human or agent decision before any mutation.
	SurfaceRequiresAgent ProposalSurface = "requires_agent"
)

// Surface classifies the proposal target surface.
// Returns one of the ProposalSurface constants based on the proposal Type.
func (p *Proposal) Surface() ProposalSurface {
	switch p.Type {
	// Schema proposals: extend enum, add aggregate, remove stem field, implicit schema
	case ExtendEnum, AddAggregate, RemoveStemField:
		return SurfaceSchema

	// Repair proposals: correct value, add field, extract from body, infer from children/siblings
	case CorrectValue, AddField, ExtractBody, InferFromChildren, InferFromSiblings, CorrectOutlier, CorrectLink:
		return SurfaceRepair

	// Migration proposals: migrate value to different field or type
	case MigrateValue:
		return SurfaceMigration

	// Aggregate propagation is a repair operation (updates frontmatter)
	case PropagateAggregate:
		return SurfaceRepair

	// Special: infer schema from body patterns
	case SetField, SetSection:
		return SurfaceRepair

	default:
		// Unknown types are marked as diagnostic (conservative default)
		return SurfaceDiagnostic
	}
}

// SurfaceString returns the string representation of a ProposalSurface.
func (s ProposalSurface) String() string {
	return string(s)
}
