package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/fix"
	"github.com/pablontiv/rootline/internal/proposal"
	"github.com/pablontiv/rootline/internal/rules"
	"github.com/spf13/cobra"
)

var (
	setDryRun     bool
	setCreate     bool
	setNoValidate bool
)

var setCmd = &cobra.Command{
	Use:   "set <file> <field=value>...",
	Short: "Set frontmatter fields or document sections",
	Long: `Set one or more frontmatter fields or document sections.

Syntax:
  field=value        Set frontmatter field (or replace section content)
  field+=@file       Append file content to a section
  field=@file        Set value from file content
  field=""           Clear a field

Use --create to add sections that don't yet exist in the document.
Use --dry-run to preview changes without modifying files.`,
	Args: cobra.MinimumNArgs(2),
	RunE: runSet,
}

func init() {
	setCmd.Flags().BoolVar(&setDryRun, "dry-run", false, "show what would change without modifying files")
	setCmd.Flags().BoolVar(&setCreate, "create", false, "allow creating new sections")
	setCmd.Flags().BoolVar(&setNoValidate, "no-validate", false, "skip post-mutation validation (emergency only)")
	rootCmd.AddCommand(setCmd)
}

// fieldOp represents a parsed field=value operation.
type fieldOp struct {
	Field  string // field name
	Value  string // resolved value
	Append bool   // true for += syntax
}

// parseFieldOps parses arguments like "field=value", "field+=@file", "field=@file".
func parseFieldOps(args []string) ([]fieldOp, error) {
	var ops []fieldOp
	for _, arg := range args {
		// Check for += first
		if idx := strings.Index(arg, "+="); idx > 0 {
			field := arg[:idx]
			raw := arg[idx+2:]
			value, err := resolveValue(raw)
			if err != nil {
				return nil, fmt.Errorf("resolving value for %s: %w", field, err)
			}
			ops = append(ops, fieldOp{Field: field, Value: value, Append: true})
			continue
		}

		// Regular = assignment
		idx := strings.Index(arg, "=")
		if idx <= 0 {
			return nil, fmt.Errorf("invalid field assignment %q (expected field=value)", arg)
		}
		field := arg[:idx]
		raw := arg[idx+1:]
		value, err := resolveValue(raw)
		if err != nil {
			return nil, fmt.Errorf("resolving value for %s: %w", field, err)
		}
		ops = append(ops, fieldOp{Field: field, Value: value, Append: false})
	}
	return ops, nil
}

// resolveValue handles =@file syntax (read from file) or returns the literal value.
func resolveValue(raw string) (string, error) {
	if strings.HasPrefix(raw, "@") {
		filePath := raw[1:]
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("reading %s: %w", filePath, err)
		}
		return string(data), nil
	}
	return raw, nil
}

// findRootMarker walks up from startPath using WalkUp to find the schema boundary.
// Returns the directory containing the root marker (or the filesystem root if no marker).
// Uses marker-based discovery instead of Git.
func findRootMarker(startPath string) (string, error) {
	entries, err := rules.WalkUp(startPath)
	if err != nil {
		return "", err
	}
	if len(entries) == 0 {
		return "", rules.ErrNoSchemaFound
	}

	// The root-most entry (first in the chain) is the boundary marker's directory
	rootEntry := entries[0]
	return filepath.Dir(rootEntry.Path), nil
}

func runSet(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Step 1: Parse file path and field ops.
	filePath := args[0]
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("resolving path %s: %w", filePath, err)
	}

	ops, err := parseFieldOps(args[1:])
	if err != nil {
		return err
	}

	// Step 2: Read file.
	content, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s (use 'rootline new' to scaffold a new document)", filePath)
		}
		return fmt.Errorf("reading %s: %w", filePath, err)
	}

	// Step 3: Get the invocation root (current working directory) for path computation.
	// This is the namespace that scanning already uses for Record.Path.
	invocationRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	// Step 3b: Resolve schema via marker-based discovery (for schema only, not paths).
	dir := filepath.Dir(absPath)
	_, err = findRootMarker(dir)
	if err != nil {
		return fmt.Errorf("resolving schema boundary: %w", err)
	}

	// Compute relative path from the invocation root (not the marker root).
	// record.Path and proposal.Paths must use the same namespace.
	relPath, err := filepath.Rel(invocationRoot, absPath)
	if err != nil {
		relPath = absPath
	}

	// Step 2b: Extract record using relative path (fix.ApplyProposals joins root+path).
	parseAST := true
	ext := &extract.MarkdownExtractor{ParseAST: &parseAST}
	record, err := ext.Extract(relPath, content)
	if err != nil {
		return fmt.Errorf("extracting %s: %w", filePath, err)
	}

	// Step 4: Load effective schema.
	effective, err := rules.ResolveForRecord(dir, absPath)
	if err != nil {
		return fmt.Errorf("resolving .stem for %s: %w", filePath, err)
	}

	var proposals []proposal.Proposal
	for _, op := range ops {
		p, pErr := buildProposal(op, relPath, effective)
		if pErr != nil {
			return pErr
		}
		proposals = append(proposals, p)
	}

	// Step 6: Pre-validate enum constraints.
	if effective != nil {
		for _, op := range ops {
			if op.Append {
				continue // append is always for sections, no enum check
			}
			sf, ok := effective.Schema[op.Field]
			if !ok {
				continue
			}
			if sf.Type == "section" {
				continue
			}
			if sf.Type == "enum" && len(sf.Values) > 0 && op.Value != "" {
				valid := false
				for _, v := range sf.Values {
					if v == op.Value {
						valid = true
						break
					}
				}
				if !valid {
					return fmt.Errorf("value %q not in allowed values for %s: [%s]",
						op.Value, op.Field, strings.Join(sf.Values, ", "))
				}
			}
		}
	}

	// Step 7: Dry-run mode.
	if setDryRun {
		for _, p := range proposals {
			switch p.Type {
			case proposal.SetField:
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "would set %s = %q\n", p.Field, p.Value)
			case proposal.SetSection:
				mode := p.Mode
				if mode == "" {
					mode = "replace"
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "would %s section %s (%s)\n", mode, p.Heading, p.Field)
			}
		}
		return nil
	}

	// Step 8: Save original content for rollback.
	originalContent := make([]byte, len(content))
	copy(originalContent, content)

	// Step 9: Apply proposals via fix.ApplyProposals.
	report := &proposal.Report{
		Version:   1,
		Kind:      "rootline/proposals",
		Proposals: proposals,
	}

	recordMap := []*extract.Record{record}
	if err := fix.ApplyProposals(ctx, report, invocationRoot, recordMap); err != nil {
		return fmt.Errorf("applying changes: %w", err)
	}

	// Step 10: Post-validate and rollback on failure.
	if !setNoValidate {
		if rollbackErr := postValidateOrRollback(ctx, absPath, dir, originalContent, effective); rollbackErr != nil {
			return rollbackErr
		}
	}

	// Print summary.
	for _, p := range report.Proposals {
		switch p.Type {
		case proposal.SetField:
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "set %s = %q\n", p.Field, p.Value)
		case proposal.SetSection:
			mode := p.Mode
			if mode == "" {
				mode = "replace"
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s section %s\n", mode, p.Heading)
		}
	}

	return nil
}

// buildProposal creates a Proposal from a fieldOp, consulting the schema to determine type.
func buildProposal(op fieldOp, relPath string, effective *rules.StemFile) (proposal.Proposal, error) {
	if op.Append {
		return proposal.Proposal{}, fmt.Errorf("append (+=) is no longer supported")
	}

	// Frontmatter field.
	return proposal.Proposal{
		Type:        proposal.SetField,
		Field:       op.Field,
		Value:       op.Value,
		Paths:       []string{relPath},
		Description: fmt.Sprintf("set %s to %q", op.Field, op.Value),
	}, nil
}

// postValidateOrRollback re-extracts the file and validates it.
// If validation fails, the original content is restored (rollback).
func postValidateOrRollback(ctx context.Context, absPath, dir string, originalContent []byte, effective *rules.StemFile) error {
	newContent, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("re-reading %s after mutation: %w", absPath, err)
	}

	parseAST := true
	ext := &extract.MarkdownExtractor{ParseAST: &parseAST}
	newRecord, err := ext.Extract(absPath, newContent)
	if err != nil {
		// Rollback
		_ = os.WriteFile(absPath, originalContent, 0644) //nolint:gosec // rollback intentionally restores the user-selected validated document path
		return fmt.Errorf("re-extracting %s after mutation (rolled back): %w", absPath, err)
	}

	// Re-resolve effective (could have changed if we modified .stem, but typically not).
	freshEffective, err := rules.ResolveForRecord(dir, absPath)
	if err != nil {
		freshEffective = effective
	}

	errs := rules.Validate(ctx, newRecord, freshEffective)
	if len(errs) > 0 {
		// Rollback
		_ = os.WriteFile(absPath, originalContent, 0644) //nolint:gosec // rollback intentionally restores the user-selected validated document path
		var msgs []string
		for _, e := range errs {
			msgs = append(msgs, fmt.Sprintf("%s: %s", e.Field, e.Message))
		}
		return fmt.Errorf("post-mutation validation failed (rolled back): %s", strings.Join(msgs, "; "))
	}
	return nil
}
