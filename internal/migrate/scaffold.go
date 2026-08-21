package migrate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/fsx"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/rules"
)

// ScaffoldResult holds the outcome of a scaffold operation.
type ScaffoldResult struct {
	// FilesScaffolded is the number of files that had at least one section added.
	FilesScaffolded int `json:"files_scaffolded"`
	// SectionsAdded is the total count of sections inserted across all files.
	SectionsAdded int `json:"sections_added"`
	// Details lists each section insertion that was performed (or would be in dry-run).
	Details []ScaffoldDetail `json:"details,omitempty"`
}

// ScaffoldDetail describes a single section insertion.
type ScaffoldDetail struct {
	File    string `json:"file"`
	Heading string `json:"heading"`
	DryRun  bool   `json:"dry_run,omitempty"`
}

// ScaffoldValidationInput is the prospective record supplied to the command
// layer for validation before scaffold writes replace the record on disk.
type ScaffoldValidationInput struct {
	Path      string
	AbsPath   string
	Content   []byte
	Effective *rules.StemFile
}

// ScaffoldValidator validates a prospective scaffolded record.
type ScaffoldValidator func(context.Context, ScaffoldValidationInput) (*rules.ValidationResult, error)

// ScaffoldOperation detects documents missing required section fields and
// scaffolds the missing sections with default or placeholder content.
type ScaffoldOperation struct {
	RootPath  string
	DryRun    bool
	Validator ScaffoldValidator
}

// Execute performs the scaffold operation, returning which files and sections
// were affected. In dry-run mode it reports what would happen without modifying
// any files.
func (op *ScaffoldOperation) Execute() (*ScaffoldResult, error) {
	absRoot, err := filepath.Abs(op.RootPath)
	if err != nil {
		return nil, err
	}

	// Use AST registry so that BodySections and links are populated for source resolution.
	reg := extract.NewASTRegistry()

	// Build a scope resolver so that index.Scan respects .stem scope rules.
	resolver := func(dir string) (*rules.StemFile, error) {
		entries, walkErr := rules.WalkUp(dir)
		if walkErr != nil {
			return nil, walkErr
		}
		return rules.MergeStemFiles(entries), nil
	}

	records, err := index.Scan(context.Background(), absRoot, reg, index.WithScopeResolver(resolver))
	if err != nil {
		return nil, err
	}

	result := &ScaffoldResult{}

	for _, rec := range records {
		if extractionErrs := rules.ExtractionErrors(rec); len(extractionErrs) > 0 {
			return nil, fmt.Errorf("scanning %s: %s", rec.Path, validationMessages(rules.NewValidationResult(rec.Path, extractionErrs)))
		}

		absPath, err := absoluteRecordPath(absRoot, rec.Path)
		if err != nil {
			return nil, fmt.Errorf("resolving path for %s: %w", rec.Path, err)
		}

		effective, resolveErr := rules.ResolveForRecord(filepath.Dir(absPath), absPath)
		if resolveErr != nil {
			return nil, fmt.Errorf("resolving schema for %s: %w", rec.Path, resolveErr)
		}
		if effective == nil {
			return nil, fmt.Errorf("no .stem schema found for %s", absPath)
		}

		sections, err := rules.RequiredSectionMaterializations(rec, effective)
		if err != nil {
			return nil, fmt.Errorf("materializing sections for %s: %w", rec.Path, err)
		}
		if len(sections) == 0 {
			continue
		}

		prospective, err := renderScaffoldedContent(absPath, sections)
		if err != nil {
			return nil, fmt.Errorf("rendering scaffold for %s: %w", rec.Path, err)
		}
		if err := op.validateProspective(rec.Path, absPath, prospective, effective); err != nil {
			return nil, fmt.Errorf("validating scaffold for %s: %w", rec.Path, err)
		}

		for _, section := range sections {
			result.SectionsAdded++
			result.Details = append(result.Details, ScaffoldDetail{
				File:    rec.Path,
				Heading: section.Heading,
				DryRun:  op.DryRun,
			})
		}

		if !op.DryRun {
			if writeErr := fsx.WriteFileAtomic(absPath, prospective, 0o644); writeErr != nil {
				return nil, fmt.Errorf("writing scaffold for %s: %w", rec.Path, writeErr)
			}
		}

		result.FilesScaffolded++
	}

	return result, nil
}

func absoluteRecordPath(absRoot, recordPath string) (string, error) {
	if filepath.IsAbs(recordPath) {
		return filepath.Abs(recordPath)
	}
	return filepath.Abs(filepath.Join(absRoot, recordPath))
}

func renderScaffoldedContent(absPath string, sections []rules.SectionMaterialization) ([]byte, error) {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	return renderScaffoldedContentFromBytes(data, sections), nil
}

func renderScaffoldedContentFromBytes(data []byte, sections []rules.SectionMaterialization) []byte {
	content := string(data)
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	var sb strings.Builder
	sb.WriteString(content)
	for _, section := range sections {
		body := strings.TrimRight(section.Content, "\n")
		sb.WriteString("\n")
		sb.WriteString(section.Heading)
		sb.WriteString("\n\n")
		sb.WriteString(body)
		sb.WriteString("\n")
	}
	return []byte(sb.String())
}

func (op *ScaffoldOperation) validateProspective(path, absPath string, content []byte, effective *rules.StemFile) error {
	if op.Validator == nil {
		return fmt.Errorf("prospective scaffold validation unavailable for %s", absPath)
	}
	result, err := op.Validator(context.Background(), ScaffoldValidationInput{
		Path:      path,
		AbsPath:   absPath,
		Content:   content,
		Effective: effective,
	})
	if err != nil {
		return err
	}
	if result == nil {
		return fmt.Errorf("prospective scaffold validation returned nil result for %s", absPath)
	}
	if !result.Valid {
		return fmt.Errorf("prospective scaffold validation failed for %s: %s", absPath, validationMessages(result))
	}
	return nil
}

func validationMessages(result *rules.ValidationResult) string {
	msgs := make([]string, 0, len(result.Errors))
	for _, err := range result.Errors {
		if err.Field != "" {
			msgs = append(msgs, fmt.Sprintf("%s: %s", err.Field, err.Message))
			continue
		}
		msgs = append(msgs, err.Message)
	}
	if len(msgs) == 0 {
		return "invalid record"
	}
	return strings.Join(msgs, "; ")
}
