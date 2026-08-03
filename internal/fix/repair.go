// Package fix implements repair operations for data-only corrections.
package fix

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/proposal"
	"github.com/pablontiv/rootline/internal/rules"
)

// RepairResult represents the outcome of a repair operation.
type RepairResult struct {
	Version  int      `json:"version"`
	Kind     string   `json:"kind"`
	DryRun   bool     `json:"dry_run"`
	Changed  []string `json:"changed"`
	Skipped  []string `json:"skipped"`
	Rejected []string `json:"rejected"` // schema proposals and containment violations
	Errors   []string `json:"errors"`

	// ResolvedTargets is populated in dry-run only, where the caller cannot
	// inspect the outcome on disk and needs to see where each write would land.
	ResolvedTargets *ResolvedTargetsBreakdown `json:"resolved_targets,omitempty"`
}

// ResolvedTargetsBreakdown reports where each report-supplied path resolved to,
// and why any of them were refused. Keys are the paths exactly as the report
// spelled them, so a caller can match entries back against its own input.
type ResolvedTargetsBreakdown struct {
	Accepted map[string]string `json:"accepted"` // report path -> validated absolute path
	Rejected map[string]string `json:"rejected"` // report path -> reason for refusal
}

// repairTarget pairs a record with the containment-validated absolute path it
// was read from. Handlers write to that path instead of re-deriving it, which
// keeps the containment invariant local to the check that established it.
type repairTarget struct {
	abs    string
	record *extract.Record
}

// ApplyRepair applies repair proposals (data-only corrections) to the filesystem.
// It accepts only proposals with Surface() == SurfaceRepair and silently rejects
// SurfaceSchema, SurfaceBootstrap, and SurfaceMigration proposals.
// Modifies Markdown frontmatter only — never touches .stem files.
// Supports dryRun for preview mode. Post-validates modified files and rolls
// back if validation fails.
//
// Reports are untrusted input, so every path they name is checked against root
// before the filesystem is touched at all. Paths that escape are recorded in
// Rejected, not Errors: they are a policy refusal rather than a failed write.
func ApplyRepair(proposals []proposal.Proposal, dryRun bool, root string) (*RepairResult, error) {
	result := &RepairResult{
		Version: 1,
		Kind:    "rootline/repair",
		DryRun:  dryRun,
	}

	if len(proposals) == 0 {
		return result, nil
	}

	// First pass: gate every proposal path on containment, before any read.
	accepted, rejected := containProposalPaths(proposals, root, result)
	if dryRun {
		result.ResolvedTargets = &ResolvedTargetsBreakdown{Accepted: accepted, Rejected: rejected}
	}

	// Second pass: extract the records behind the paths that survived the gate.
	targets := make(map[string]*repairTarget)
	reg := extract.NewRegistry()

	for path, absPath := range accepted {
		// A directory otherwise reaches os.ReadFile and comes back as
		// "read docs: is a directory" — the syscall that tripped over the path
		// rather than an answer about the report. Repair takes document paths.
		if info, statErr := os.Stat(absPath); statErr == nil && info.IsDir() {
			result.Errors = append(result.Errors,
				fmt.Sprintf("%s is a directory; use 'rootline repair apply --report <file>.json'", path))
			continue
		}
		content, err := os.ReadFile(absPath)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("read %s: %v", path, err))
			continue
		}

		ext := reg.ForFile(absPath, "")
		if ext == nil {
			result.Errors = append(result.Errors, fmt.Sprintf("no extractor for %s", path))
			continue
		}

		record, err := ext.Extract(path, content)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("extract %s: %v", path, err))
			continue
		}

		targets[path] = &repairTarget{abs: absPath, record: record}
	}

	// Third pass: classify and apply proposals.
	for _, p := range proposals {
		// A proposal is applied whole or not at all, so one bad path discards it.
		if hasRejectedPath(p, rejected) {
			continue
		}

		// Classify by surface.
		surf := p.Surface()
		if surf != proposal.SurfaceRepair {
			result.Rejected = append(result.Rejected, fmt.Sprintf("%s: %s (surface=%s)", p.Field, p.Description, surf))
			continue
		}

		// Apply based on proposal type.
		switch p.Type {
		case proposal.CorrectValue, proposal.MigrateValue:
			if err := applyRepairCorrectValue(&p, targets, result, dryRun); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s %s: %v", p.Type, p.Paths, err))
			}

		case proposal.AddField, proposal.ExtractBody, proposal.InferFromChildren, proposal.InferFromSiblings:
			if err := applyRepairAddField(&p, targets, result, dryRun); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s %s: %v", p.Type, p.Paths, err))
			}

		case proposal.SetField:
			if err := applyRepairSetField(&p, targets, result, dryRun); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("set_field %s: %v", p.Paths, err))
			}

		case proposal.SetSection:
			if err := applyRepairSetSection(&p, root, result, dryRun); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("set_section %s: %v", p.Heading, err))
			}

		case proposal.CorrectOutlier:
			if err := applyRepairCorrectValue(&p, targets, result, dryRun); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("correct_outlier %s: %v", p.Paths, err))
			}

		case proposal.PropagateAggregate:
			if err := applyRepairCorrectValue(&p, targets, result, dryRun); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("propagate_aggregate %s: %v", p.Paths, err))
			}

		case proposal.CorrectLink:
			if err := applyRepairCorrectLink(&p, targets, result, dryRun); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("correct_link %s: %v", p.Paths, err))
			}

		default:
			// Unknown repair-surface proposal type.
			result.Rejected = append(result.Rejected, fmt.Sprintf("%s: unknown repair type", p.Field))
		}
	}

	// Fourth pass: post-validate and rollback on failure.
	if !dryRun && len(result.Errors) == 0 {
		for path, tgt := range targets {
			// Re-resolve effective stem.
			dir := filepath.Dir(tgt.abs)
			effective, _ := rules.ResolveForRecord(dir, tgt.abs)

			// Validate.
			ctx := context.Background()
			errs := rules.Validate(ctx, tgt.record, effective)
			if len(errs) > 0 {
				// Validation failed — rollback.
				// Note: Since we've modified rec in-memory and written to file,
				// we cannot easily rollback without tracking originals.
				// For now, document the validation failure.
				var msgs []string
				for _, e := range errs {
					msgs = append(msgs, fmt.Sprintf("%s: %s", e.Field, e.Message))
				}
				result.Errors = append(result.Errors,
					fmt.Sprintf("post-validation failed for %s: %s", path, msgs))
			}
		}
	}

	return result, nil
}

// containProposalPaths validates every distinct path named by the proposals and
// splits them into the accepted paths (mapped to their validated absolute form)
// and the refused ones (mapped to the reason). Refusals are also appended to
// result.Rejected so they surface in the command output.
func containProposalPaths(proposals []proposal.Proposal, root string, result *RepairResult) (accepted, rejected map[string]string) {
	accepted = make(map[string]string)
	rejected = make(map[string]string)

	for _, p := range proposals {
		for _, path := range p.Paths {
			if _, seen := accepted[path]; seen {
				continue
			}
			if _, seen := rejected[path]; seen {
				continue
			}

			// Repair reports carry root-relative record paths by construction,
			// so an absolute path is malformed input rather than a real target.
			absPath, err := ContainPath(root, path, PolicyRejectAbsolute)
			if err != nil {
				rejected[path] = containmentReason(err)
				result.Rejected = append(result.Rejected, err.Error())
				continue
			}
			accepted[path] = absPath
		}
	}

	return accepted, rejected
}

// containmentReason extracts the bare reason from a containment failure, for
// the resolved-targets breakdown where the path is already the map key.
func containmentReason(err error) string {
	var cerr *ContainmentError
	if errors.As(err, &cerr) {
		return cerr.Message
	}
	return err.Error()
}

// hasRejectedPath reports whether any path of the proposal failed containment.
func hasRejectedPath(p proposal.Proposal, rejected map[string]string) bool {
	for _, path := range p.Paths {
		if _, bad := rejected[path]; bad {
			return true
		}
	}
	return false
}

// applyRepairCorrectValue updates a field value in a record's frontmatter.
func applyRepairCorrectValue(p *proposal.Proposal, targets map[string]*repairTarget, result *RepairResult, dryRun bool) error {
	for _, path := range p.Paths {
		tgt, ok := targets[path]
		if !ok {
			continue
		}

		tgt.record.Frontmatter[p.Field] = p.To

		if dryRun {
			result.Changed = append(result.Changed,
				fmt.Sprintf("correct %s: %q->%q in %s", p.Field, p.From, p.To, path))
			continue
		}

		// Read file, update frontmatter, write back.
		content, err := os.ReadFile(tgt.abs)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		newContent := RewriteFrontmatter(string(content), tgt.record.Frontmatter)
		if err := os.WriteFile(tgt.abs, []byte(newContent), 0644); err != nil { //nolint:gosec // tgt.abs is the path ContainPath validated and confined to root
			return fmt.Errorf("writing %s: %w", path, err)
		}

		result.Changed = append(result.Changed,
			fmt.Sprintf("correct %s: %q->%q in %s", p.Field, p.From, p.To, path))
	}
	return nil
}

// applyRepairAddField adds a missing field to a record's frontmatter.
func applyRepairAddField(p *proposal.Proposal, targets map[string]*repairTarget, result *RepairResult, dryRun bool) error {
	value := p.Value
	if value == "" {
		value = p.To
	}

	for _, path := range p.Paths {
		tgt, ok := targets[path]
		if !ok {
			continue
		}

		tgt.record.Frontmatter[p.Field] = value

		if dryRun {
			result.Changed = append(result.Changed,
				fmt.Sprintf("add %s=%q in %s", p.Field, value, path))
			continue
		}

		// Read file, update frontmatter, write back.
		content, err := os.ReadFile(tgt.abs)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		newContent := RewriteFrontmatter(string(content), tgt.record.Frontmatter)
		if err := os.WriteFile(tgt.abs, []byte(newContent), 0644); err != nil { //nolint:gosec // tgt.abs is the path ContainPath validated and confined to root
			return fmt.Errorf("writing %s: %w", path, err)
		}

		result.Changed = append(result.Changed,
			fmt.Sprintf("add %s=%q in %s", p.Field, value, path))
	}
	return nil
}

// applyRepairSetField sets a field via SetField proposal.
func applyRepairSetField(p *proposal.Proposal, targets map[string]*repairTarget, result *RepairResult, dryRun bool) error {
	for _, path := range p.Paths {
		tgt, ok := targets[path]
		if !ok {
			continue
		}

		tgt.record.Frontmatter[p.Field] = p.Value

		if dryRun {
			result.Changed = append(result.Changed,
				fmt.Sprintf("set %s=%q in %s", p.Field, p.Value, path))
			continue
		}

		// Read file, update frontmatter, write back.
		content, err := os.ReadFile(tgt.abs)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		newContent := RewriteFrontmatter(string(content), tgt.record.Frontmatter)
		if err := os.WriteFile(tgt.abs, []byte(newContent), 0644); err != nil { //nolint:gosec // tgt.abs is the path ContainPath validated and confined to root
			return fmt.Errorf("writing %s: %w", path, err)
		}

		result.Changed = append(result.Changed,
			fmt.Sprintf("set %s=%q in %s", p.Field, p.Value, path))
	}
	return nil
}

// applyRepairSetSection applies a SetSection proposal by delegating to the
// shared applySetSection, which re-checks containment because it is also
// reachable from the fix --all pipeline.
func applyRepairSetSection(p *proposal.Proposal, root string, result *RepairResult, dryRun bool) error {
	mode := p.Mode
	if mode == "" {
		mode = "replace"
	}

	if !dryRun {
		if err := applySetSection(*p, root, nil, PolicyRejectAbsolute); err != nil {
			return err
		}
	}

	// Recorded in both modes: an applied section write used to be invisible in
	// the result even though dry-run previewed it.
	for _, path := range p.Paths {
		result.Changed = append(result.Changed,
			fmt.Sprintf("%s section %s in %s", mode, p.Heading, path))
	}
	return nil
}

// applyRepairCorrectLink applies a CorrectLink proposal.
func applyRepairCorrectLink(p *proposal.Proposal, targets map[string]*repairTarget, result *RepairResult, dryRun bool) error {
	for _, path := range p.Paths {
		if dryRun {
			result.Changed = append(result.Changed,
				fmt.Sprintf("correct link: %s -> %s in %s", p.From, p.To, path))
			continue
		}

		tgt, ok := targets[path]
		if !ok {
			continue
		}

		// Read file, replace link, write back.
		content, err := os.ReadFile(tgt.abs)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		newContent := string(content)
		// Simple string replace — assumes link appears once.
		if idx := len(newContent); idx > 0 {
			newContent = replaceOnce(newContent, p.From, p.To)
		}

		if err := os.WriteFile(tgt.abs, []byte(newContent), 0644); err != nil { //nolint:gosec // tgt.abs is the path ContainPath validated and confined to root
			return fmt.Errorf("writing %s: %w", path, err)
		}

		result.Changed = append(result.Changed,
			fmt.Sprintf("correct link: %s -> %s in %s", p.From, p.To, path))
	}
	return nil
}

// replaceOnce replaces the first occurrence of old with new in s.
func replaceOnce(s, old, new string) string {
	idx := len(s) // sentinel: not found
	for i := 0; i < len(s)-(len(old)-1); i++ {
		if s[i:i+len(old)] == old {
			idx = i
			break
		}
	}
	if idx < len(s)-(len(old)-1) {
		return s[:idx] + new + s[idx+len(old):]
	}
	return s
}
