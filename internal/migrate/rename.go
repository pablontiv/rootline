// Package migrate implements bulk migration operations for rootline documents.
//
// It provides field renaming across frontmatter and .stem schemas,
// preserving file structure while atomically updating all affected records.
package migrate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/fsx"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/rules"
	"gopkg.in/yaml.v3"
)

// RenameResult holds the outcome of a rename operation.
type RenameResult struct {
	Version      int              `json:"version"`
	Kind         string           `json:"kind"`
	OldField     string           `json:"old_field"`
	NewField     string           `json:"new_field"`
	FilesUpdated []string         `json:"files_updated"`
	StemsUpdated []string         `json:"stems_updated"`
	Summary      RenameSummary    `json:"summary"`
	FileDetails  []RenameFileInfo `json:"file_details,omitempty"`
}

// RenameSummary holds aggregate counts.
type RenameSummary struct {
	FilesScanned int `json:"files_scanned"`
	FilesUpdated int `json:"files_updated"`
	StemsUpdated int `json:"stems_updated"`
}

// RenameFileInfo describes a single file affected by the rename.
type RenameFileInfo struct {
	Path    string `json:"path"`
	Updated bool   `json:"updated"`
}

// RenameOperation performs a field rename across documents and .stem schemas.
type RenameOperation struct {
	OldField string
	NewField string
	RootPath string
	DryRun   bool
}

// Execute performs the rename operation, returning the result.
func (op *RenameOperation) Execute() (*RenameResult, error) {
	absRoot, err := filepath.Abs(op.RootPath)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	// Verify target path exists.
	info, err := os.Stat(absRoot)
	if err != nil {
		return nil, fmt.Errorf("path %s: %w", op.RootPath, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path %s is not a directory", op.RootPath)
	}

	// Scan all records under the path.
	reg := extract.NewRegistry()
	resolver := func(dir string) (*rules.StemFile, error) {
		entries, walkErr := rules.WalkUp(dir)
		if walkErr != nil {
			return nil, walkErr
		}
		return rules.MergeStemFiles(entries), nil
	}

	records, err := index.Scan(context.Background(), absRoot, reg, index.WithScopeResolver(resolver))
	if err != nil {
		return nil, fmt.Errorf("scanning: %w", err)
	}

	// Preflight every affected document and schema before any writes.
	for _, rec := range records {
		if _, hasOld := rec.Frontmatter[op.OldField]; !hasOld {
			continue
		}
		if _, hasNew := rec.Frontmatter[op.NewField]; hasNew {
			return nil, fmt.Errorf("file %s already has field %q; rename would overwrite", rec.Path, op.NewField)
		}
	}

	stems, err := findStemsWithField(absRoot, op.OldField)
	if err != nil {
		return nil, fmt.Errorf("finding .stem files: %w", err)
	}
	for _, entry := range stems {
		if _, hasNew := entry.Stem.Schema[op.NewField]; hasNew {
			return nil, fmt.Errorf(".stem %s already has field %q; rename would overwrite", entry.Path, op.NewField)
		}
	}

	result := &RenameResult{
		Version:  1,
		Kind:     "rootline/migrate-rename",
		OldField: op.OldField,
		NewField: op.NewField,
		Summary:  RenameSummary{FilesScanned: len(records)},
	}

	// Phase 1: Rename field in frontmatter of each matching record.
	for _, rec := range records {
		detail := RenameFileInfo{Path: rec.Path, Updated: false}

		if _, hasOld := rec.Frontmatter[op.OldField]; !hasOld {
			result.FileDetails = append(result.FileDetails, detail)
			continue
		}

		absPath := filepath.Join(absRoot, rec.Path)
		changed, err := renameFieldInFile(absPath, op.OldField, op.NewField, op.DryRun)
		if err != nil {
			return nil, fmt.Errorf("renaming field in %s: %w", rec.Path, err)
		}
		if !changed {
			result.FileDetails = append(result.FileDetails, detail)
			continue
		}

		detail.Updated = true
		result.FilesUpdated = append(result.FilesUpdated, rec.Path)
		result.FileDetails = append(result.FileDetails, detail)
	}

	result.Summary.FilesUpdated = len(result.FilesUpdated)

	// Phase 2: Rename field in .stem schema files.
	for _, entry := range stems {
		stemPath := entry.Path
		if !op.DryRun {
			if err := renameFieldInStem(stemPath, op.OldField, op.NewField); err != nil {
				return nil, fmt.Errorf("renaming field in .stem %s: %w", stemPath, err)
			}
		}
		relStem, _ := filepath.Rel(absRoot, stemPath)
		if relStem == "" {
			relStem = stemPath
		}
		result.StemsUpdated = append(result.StemsUpdated, relStem)
	}

	result.Summary.StemsUpdated = len(result.StemsUpdated)

	return result, nil
}

// renameFieldInFile reads a markdown file and renames the field in its frontmatter.
// It reports whether the key changed and skips the write in dry-run mode.
func renameFieldInFile(absPath, oldField, newField string, dryRun bool) (bool, error) {
	content, err := os.ReadFile(absPath)
	if err != nil {
		return false, err
	}

	newContent, changed := renameFrontmatterField(string(content), oldField, newField)
	if !changed || dryRun {
		return changed, nil
	}

	if err := fsx.WriteFileAtomic(absPath, []byte(newContent), 0o644); err != nil {
		return false, err
	}
	return true, nil
}

// renameFrontmatterField renames a column-0 key in the leading YAML frontmatter.
// Only the key bytes change; all other document bytes are preserved.
func renameFrontmatterField(text, oldField, newField string) (string, bool) {
	// Extraction strips a UTF-8 BOM before locating frontmatter. Keep it in the
	// original document and translate the locator's offsets back to that document.
	offset := 0
	if strings.HasPrefix(text, "\xef\xbb\xbf") {
		offset = 3
	}
	start, end, _, ok := extract.FrontmatterBounds(text[offset:])
	if !ok {
		return text, false
	}
	start += offset
	end += offset

	prefix := oldField + ":"
	for pos := start; pos < end; {
		lineEnd := end
		if nl := strings.IndexByte(text[pos:end], '\n'); nl >= 0 {
			lineEnd = pos + nl
		}
		if strings.HasPrefix(text[pos:lineEnd], prefix) {
			return text[:pos] + newField + text[pos+len(oldField):], true
		}
		pos = lineEnd + 1
	}
	return text, false
}

// findStemsWithField finds declaring .stem files under and above the root,
// retaining their parsed schemas for collision preflight.
func findStemsWithField(rootPath, field string) ([]rules.StemEntry, error) {
	var stems []rules.StemEntry
	seen := make(map[string]bool)

	err := filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if d.Name() != ".stem" {
			return nil
		}

		absPath, _ := filepath.Abs(path)
		if seen[absPath] {
			return nil
		}
		seen[absPath] = true

		content, readErr := os.ReadFile(absPath) //nolint:gosec // path comes from filepath.WalkDir and is deduplicated before reading
		if readErr != nil {
			return nil
		}

		stem, parseErr := rules.ParseStem(path, content)
		if parseErr != nil {
			return nil
		}

		if _, ok := stem.Schema[field]; ok {
			stems = append(stems, rules.StemEntry{Path: absPath, Stem: stem})
		}
		return nil
	})

	// Also check stems above the rootPath (walk up).
	entries, walkErr := rules.WalkUp(rootPath)
	if walkErr == nil {
		for _, entry := range entries {
			absPath, _ := filepath.Abs(entry.Path)
			if seen[absPath] {
				continue
			}
			if entry.Stem != nil {
				if _, ok := entry.Stem.Schema[field]; ok {
					stems = append(stems, rules.StemEntry{Path: absPath, Stem: entry.Stem})
					seen[absPath] = true
				}
			}
		}
	}

	sort.Slice(stems, func(i, j int) bool { return stems[i].Path < stems[j].Path })
	return stems, err
}

// renameFieldInStem renames a field key within a .stem file's schema section,
// preserving the YAML structure and comments via AST manipulation.
func renameFieldInStem(stemPath, oldField, newField string) error {
	content, err := os.ReadFile(stemPath)
	if err != nil {
		return err
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return fmt.Errorf("parsing %s: %w", stemPath, err)
	}

	if len(doc.Content) == 0 {
		return fmt.Errorf("empty .stem document")
	}

	rootNode := doc.Content[0]
	if !renameSchemaKey(rootNode, oldField, newField) {
		return fmt.Errorf("field %q not found in schema of %s", oldField, stemPath)
	}

	out, err := yaml.Marshal(&doc)
	if err != nil {
		return err
	}
	return fsx.WriteFileAtomic(stemPath, out, 0o644)
}

// renameSchemaKey renames a key under the "schema" mapping in a YAML AST node.
func renameSchemaKey(node *yaml.Node, oldField, newField string) bool {
	if node.Kind != yaml.MappingNode {
		return false
	}

	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == "schema" {
			schemaNode := node.Content[i+1]
			if schemaNode.Kind != yaml.MappingNode {
				continue
			}
			for j := 0; j+1 < len(schemaNode.Content); j += 2 {
				if schemaNode.Content[j].Value == oldField {
					schemaNode.Content[j].Value = newField
					return true
				}
			}
		}
	}
	return false
}
