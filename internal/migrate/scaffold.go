package migrate

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
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

// sectionCandidate holds a section field that needs to be scaffolded into a file.
type sectionCandidate struct {
	heading string
	content string
	ordered int // 0 means no ordering specified; positive values determine insertion order
}

// ScaffoldOperation detects documents missing required section fields and
// scaffolds the missing sections with default or placeholder content.
type ScaffoldOperation struct {
	RootPath string
	DryRun   bool
}

// Execute performs the scaffold operation, returning which files and sections
// were affected. In dry-run mode it reports what would happen without modifying
// any files.
func (op *ScaffoldOperation) Execute() (*ScaffoldResult, error) {
	absRoot, err := filepath.Abs(op.RootPath)
	if err != nil {
		return nil, err
	}

	// Use AST registry so that Record.Sections is populated.
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
		// Determine the directory the record lives in so we can resolve the
		// correct effective .stem (handles nested directories).
		recDir := filepath.Dir(filepath.Join(absRoot, rec.Path))

		effective, resolveErr := rules.ResolveForRecord(recDir, rec.Path)
		if resolveErr != nil || effective == nil {
			continue
		}

		candidates := missingSections(rec, effective)
		if len(candidates) == 0 {
			continue
		}

		for _, c := range candidates {
			result.SectionsAdded++
			result.Details = append(result.Details, ScaffoldDetail{
				File:    rec.Path,
				Heading: c.heading,
				DryRun:  op.DryRun,
			})
		}

		if !op.DryRun {
			absPath := filepath.Join(absRoot, rec.Path)
			if writeErr := appendSections(absPath, candidates); writeErr != nil {
				return nil, writeErr
			}
		}

		result.FilesScaffolded++
	}

	return result, nil
}

// missingSections returns section candidates that are required by the schema but
// absent from the record's Sections map.
func missingSections(rec *extract.Record, effective *rules.StemFile) []sectionCandidate {
	if effective == nil || len(effective.Schema) == 0 {
		return nil
	}

	// Collect all section fields that are required and missing.
	var candidates []sectionCandidate

	for _, sf := range effective.Schema {
		if sf.Type != "section" || !sf.Required {
			continue
		}
		heading := sf.Heading
		if heading == "" {
			continue
		}

		// Check whether the section already exists in the record.
		if _, present := rec.Sections[heading]; present {
			continue
		}

		// Determine the content to scaffold.
		content := sf.Default
		if content == "" {
			content = "<!-- TODO -->"
		}

		ordered := 0
		if sf.Ordered != nil {
			ordered = *sf.Ordered
		}

		candidates = append(candidates, sectionCandidate{
			heading: heading,
			content: content,
			ordered: ordered,
		})
	}

	if len(candidates) == 0 {
		return nil
	}

	// Sort candidates: ordered fields (ordered > 0) come first in ascending
	// order; unordered fields follow in their natural map iteration order.
	sort.SliceStable(candidates, func(i, j int) bool {
		oi, oj := candidates[i].ordered, candidates[j].ordered
		if oi == 0 && oj == 0 {
			return false // preserve natural order among unordered
		}
		if oi == 0 {
			return false // unordered goes after ordered
		}
		if oj == 0 {
			return true // ordered goes before unordered
		}
		return oi < oj
	})

	return candidates
}

// appendSections appends one or more sections to the end of a file.
// Each section is written as: \n<heading>\n\n<content>\n
func appendSections(absPath string, candidates []sectionCandidate) error {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return err
	}

	content := string(data)
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	var sb strings.Builder
	sb.WriteString(content)

	for _, c := range candidates {
		body := strings.TrimRight(c.content, "\n")
		sb.WriteString("\n")
		sb.WriteString(c.heading)
		sb.WriteString("\n\n")
		sb.WriteString(body)
		sb.WriteString("\n")
	}

	return os.WriteFile(absPath, []byte(sb.String()), 0644)
}
