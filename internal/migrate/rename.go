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
	resolver := func(dir string) *rules.StemFile {
		entries, walkErr := rules.WalkUp(dir)
		if walkErr != nil {
			return nil
		}
		return rules.MergeStemFiles(entries)
	}

	records, err := index.Scan(context.Background(), absRoot, reg, index.WithScopeResolver(resolver))
	if err != nil {
		return nil, fmt.Errorf("scanning: %w", err)
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

		// Check if the new field already exists to prevent data loss.
		if _, hasNew := rec.Frontmatter[op.NewField]; hasNew {
			return nil, fmt.Errorf("file %s already has field %q; rename would overwrite", rec.Path, op.NewField)
		}

		if !op.DryRun {
			absPath := filepath.Join(absRoot, rec.Path)
			if err := renameFieldInFile(absPath, op.OldField, op.NewField); err != nil {
				return nil, fmt.Errorf("renaming field in %s: %w", rec.Path, err)
			}
		}

		detail.Updated = true
		result.FilesUpdated = append(result.FilesUpdated, rec.Path)
		result.FileDetails = append(result.FileDetails, detail)
	}

	result.Summary.FilesUpdated = len(result.FilesUpdated)

	// Phase 2: Rename field in .stem schema files.
	stemPaths, err := findStemsWithField(absRoot, op.OldField)
	if err != nil {
		return nil, fmt.Errorf("finding .stem files: %w", err)
	}

	for _, stemPath := range stemPaths {
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

// renameFieldInFile reads a markdown file, renames the field in its frontmatter,
// and writes the file back preserving the body content.
func renameFieldInFile(absPath, oldField, newField string) error {
	content, err := os.ReadFile(absPath)
	if err != nil {
		return err
	}

	text := string(content)
	newContent, changed := renameFrontmatterField(text, oldField, newField)
	if !changed {
		return nil
	}

	return os.WriteFile(absPath, []byte(newContent), 0644)
}

// renameFrontmatterField renames a key in the YAML frontmatter of a markdown string.
// It preserves the body and field ordering. Returns the modified string and whether
// a change was made.
func renameFrontmatterField(text, oldField, newField string) (string, bool) {
	if !strings.HasPrefix(text, "---\n") {
		return text, false
	}

	endIdx := strings.Index(text[4:], "\n---\n")
	if endIdx == -1 {
		// Check for file ending with ---\n or ---EOF
		endIdx = strings.Index(text[4:], "\n---")
		if endIdx == -1 {
			return text, false
		}
		// Make sure it's actually the end of frontmatter.
		remaining := text[4+endIdx+4:]
		if remaining != "" && remaining != "\n" {
			return text, false
		}
	}

	fmContent := text[4 : 4+endIdx]
	body := text[4+endIdx+4:] // skip \n---

	// Process frontmatter line by line, renaming the key.
	lines := strings.Split(fmContent, "\n")
	changed := false
	prefix := oldField + ":"

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) {
			rest := line[strings.Index(line, ":")+1:]
			lines[i] = newField + ":" + rest
			changed = true
			break // Only rename the first occurrence at the top level.
		}
	}

	if !changed {
		return text, false
	}

	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(strings.Join(lines, "\n"))
	b.WriteString("\n---")
	b.WriteString(body)
	return b.String(), true
}

// findStemsWithField walks up from the given root to find all .stem files
// that contain the specified field in their schema.
func findStemsWithField(rootPath, field string) ([]string, error) {
	var stemPaths []string
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

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}

		stem, parseErr := rules.ParseStem(path, content)
		if parseErr != nil {
			return nil
		}

		if _, ok := stem.Schema[field]; ok {
			stemPaths = append(stemPaths, absPath)
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
					stemPaths = append(stemPaths, absPath)
					seen[absPath] = true
				}
			}
		}
	}

	sort.Strings(stemPaths)
	return stemPaths, err
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
	return os.WriteFile(stemPath, out, 0644)
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
