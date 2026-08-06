// Package fix implements repair operations for data-only corrections.
package fix

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/fsx"
	"github.com/pablontiv/rootline/internal/proposal"
	"github.com/pablontiv/rootline/internal/rules"
)

// RolledBackFile records a file that was written and then restored because it
// failed post-validation, together with the errors that triggered the restore.
type RolledBackFile struct {
	Path   string   `json:"path"`
	Errors []string `json:"errors"`
}

// RepairResult represents the outcome of a repair operation.
type RepairResult struct {
	Version int    `json:"version"`
	Kind    string `json:"kind"`
	// Root is the directory the report's document paths were resolved against.
	// It is reported because the failure this field exists to expose is silent:
	// a report resolved against the wrong root produces a run that touches
	// nothing and, before this was surfaced, said nothing about why.
	Root string `json:"root,omitempty"`
	// Complete is the run's own verdict on whether it carried through everything
	// it accepted. It is true exactly when Errors and RolledBack are both empty,
	// which is also exactly when the command exits 0.
	//
	// It is not redundant with the exit status. A report saved as a CI artifact
	// is read long after the exit code is gone, and a consumer must not have to
	// re-implement the rule — or parse prose — to learn whether the tree is in
	// the state it asked for. Rejections and skips do not make a run incomplete:
	// the command refused or deferred those on purpose and said so.
	Complete bool     `json:"complete"`
	DryRun   bool     `json:"dry_run"`
	Changed  []string `json:"changed"`
	// RolledBack lists files whose write was reverted after post-validation
	// rejected the result. A caller cannot infer this from Changed and Errors
	// alone, so the two outcomes are reported separately.
	RolledBack []RolledBackFile `json:"rolled_back,omitempty"`
	Skipped    []string         `json:"skipped"`
	Rejected   []string         `json:"rejected"` // schema proposals and containment violations
	Errors     []string         `json:"errors"`

	// changedPaths runs parallel to Changed and names the file each entry
	// describes, so a rollback can retract exactly its own entries without
	// pattern-matching the human-readable message. Unexported: internal
	// bookkeeping, not part of the JSON contract.
	changedPaths []string

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
	// original holds the bytes read before any write, so a failed
	// post-validation can restore them. They are captured during the read pass
	// that already loads every file, so tracking them costs no extra I/O.
	original []byte
	// written records whether a handler actually wrote this target. Files that
	// were only read must not be post-validated: reporting a pre-existing
	// validation failure as a rollback would be a false accusation.
	written bool
}

// ApplyRepair applies repair proposals (data-only corrections) to the filesystem.
// It accepts only proposals with Surface() == SurfaceRepair and silently rejects
// SurfaceSchema, SurfaceBootstrap, and SurfaceMigration proposals.
// Modifies Markdown frontmatter only — never touches .stem files.
// Supports dryRun for preview mode.
//
// Every file it writes is post-validated on its own and restored to its
// pre-write bytes when validation rejects the result, so a run never leaves a
// document in a state its own schema refuses. The check is per file by
// construction: an error against one path — including a path that could not be
// read at all — has no bearing on whether any other file is validated.
// Reverted files are reported in RolledBack and withdrawn from Changed.
//
// This covers every handler that writes through a repairTarget. set_section is
// the one exception: it resolves its own paths inside applySetSection and holds
// no target, so its writes carry no captured original and are not post-validated
// here. Stated rather than glossed over, because a doc comment promising a
// rollback that does not happen is the defect this function was fixing.
//
// Reports are untrusted input, so every path they name is checked against root
// before the filesystem is touched at all. Paths that escape are recorded in
// Rejected, not Errors: they are a policy refusal rather than a failed write.
func ApplyRepair(proposals []proposal.Proposal, dryRun bool, root string, fillMissing bool) (*RepairResult, error) {
	result := &RepairResult{
		Version: 1,
		Kind:    "rootline/repair",
		Root:    root,
		DryRun:  dryRun,
	}

	if len(proposals) == 0 {
		result.seal()
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

		targets[path] = &repairTarget{abs: absPath, record: record, original: content}
	}

	// Third pass: classify and apply proposals.
	for _, p := range proposals {
		// A proposal is applied whole or not at all, so one bad path discards it.
		if hasRejectedPath(p, rejected) {
			continue
		}

		// A value the engine picked because the schema declared none stays out
		// of the document unless the caller explicitly asked for it.
		if p.Type == proposal.AddField && proposal.IsEngineChosen(p.ValueSource) && !fillMissing {
			result.Skipped = append(result.Skipped,
				fmt.Sprintf("%s: engine-chosen value (%s); pass --fill-missing to apply", p.Field, p.ValueSource))
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

	// Fourth pass: post-validate every written file and roll back the failures.
	//
	// The gate is per file and deliberately does NOT consult result.Errors.
	// Gating the whole pass on a clean error list meant one unrelated read
	// failure disabled post-validation for every other file in the run.
	if !dryRun {
		postValidateWrittenTargets(targets, result)
	}

	result.seal()
	return result, nil
}

// seal records the run's completeness verdict. It is called once, after every
// pass has had its say, so Complete can never disagree with the fields it
// summarizes — and so it stays in step with the exit-status rule by
// construction rather than by two places remembering the same thing.
func (r *RepairResult) seal() {
	r.Complete = len(r.Errors) == 0 && len(r.RolledBack) == 0
}

// postValidateWrittenTargets re-validates each written file and restores its
// original bytes when validation rejects the result, so repair apply never
// leaves a document in a state its own schema refuses.
func postValidateWrittenTargets(targets map[string]*repairTarget, result *RepairResult) {
	ctx := context.Background()

	// Iterate in path order: map ranging is randomized and the result slices
	// are user-visible output.
	paths := make([]string, 0, len(targets))
	for path := range targets {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		tgt := targets[path]
		if !tgt.written {
			continue
		}

		msgs := validateWrittenTarget(ctx, tgt)
		if len(msgs) == 0 {
			continue
		}

		// The restore is atomic for the same reason the write was, and it
		// matters more here: a revert that half-succeeded would leave the
		// document in a state worse than the write it was undoing.
		if err := fsx.WriteFileAtomic(tgt.abs, tgt.original, newFileMode); err != nil {
			// The mutation stands and could not be undone. That is strictly
			// worse than a rollback, so it is reported as an error rather than
			// as a successful revert.
			result.Errors = append(result.Errors,
				fmt.Sprintf("post-validation failed for %s and rollback failed: %v", path, err))
			continue
		}

		result.RolledBack = append(result.RolledBack, RolledBackFile{Path: path, Errors: msgs})
		result.retractChanges(path)
	}
}

// validateWrittenTarget re-reads the file from disk and validates it, returning
// the formatted validation errors. Reading back rather than validating the
// in-memory record is what makes this a real check: it also catches a write
// that serialized to something other than the record it came from.
func validateWrittenTarget(ctx context.Context, tgt *repairTarget) []string {
	content, err := os.ReadFile(tgt.abs)
	if err != nil {
		return []string{fmt.Sprintf("re-reading after write: %v", err)}
	}

	reg := extract.NewRegistry()
	ext := reg.ForFile(tgt.abs, "")
	if ext == nil {
		return nil
	}
	record, err := ext.Extract(tgt.record.Path, content)
	if err != nil {
		return []string{fmt.Sprintf("re-extracting after write: %v", err)}
	}

	effective, err := rules.ResolveForRecord(filepath.Dir(tgt.abs), tgt.abs)
	if err != nil {
		return nil
	}

	var msgs []string
	for _, e := range rules.Validate(ctx, record, effective) {
		msgs = append(msgs, fmt.Sprintf("%s: %s", e.Field, e.Message))
	}
	return msgs
}

// retractChanges drops the Changed entries belonging to path. A file that was
// reverted was not changed, and leaving it listed would tell the caller a
// mutation landed when the opposite is true.
func (r *RepairResult) retractChanges(path string) {
	kept := make([]string, 0, len(r.Changed))
	keptPaths := make([]string, 0, len(r.changedPaths))
	for i, entry := range r.Changed {
		if i < len(r.changedPaths) && r.changedPaths[i] == path {
			continue
		}
		kept = append(kept, entry)
		if i < len(r.changedPaths) {
			keptPaths = append(keptPaths, r.changedPaths[i])
		}
	}
	r.Changed = kept
	r.changedPaths = keptPaths
}

// recordChange appends a Changed entry together with the path it describes.
func (r *RepairResult) recordChange(path, msg string) {
	r.Changed = append(r.Changed, msg)
	r.changedPaths = append(r.changedPaths, path)
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
				rejected[path] = ContainmentReason(err)
				result.Rejected = append(result.Rejected, err.Error())
				continue
			}
			accepted[path] = absPath
		}
	}

	return accepted, rejected
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
		if p.Type == proposal.MigrateValue && len(p.WikiLinks) > 0 {
			newContent = InsertWikiLinksBeforeHeading(newContent, p.WikiLinks)
		}
		if err := fsx.WriteFileAtomic(tgt.abs, []byte(newContent), newFileMode); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
		tgt.written = true

		result.recordChange(path,
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
		if err := fsx.WriteFileAtomic(tgt.abs, []byte(newContent), newFileMode); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
		tgt.written = true

		result.recordChange(path,
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
		if err := fsx.WriteFileAtomic(tgt.abs, []byte(newContent), newFileMode); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
		tgt.written = true

		result.recordChange(path,
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

		if err := fsx.WriteFileAtomic(tgt.abs, []byte(newContent), newFileMode); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
		tgt.written = true

		result.recordChange(path,
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
