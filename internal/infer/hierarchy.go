package infer

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

// dirPattern matches directory names like "E03-something", "F01-name", "S001-task".
var dirPattern = regexp.MustCompile(`^([A-Z]+)(\d+)-`)

// Level represents a detected hierarchy level based on directory naming patterns.
type Level struct {
	Prefix   string            // e.g. "E", "F", "S", "T"
	Digits   int               // number of digits, e.g. 2 for "E03"
	Depth    int               // depth relative to scan root (0-based)
	DirPaths []string          // actual directory paths at this level
	Records  []*extract.Record // records belonging to this level
}

// HierarchyResult is the output of hierarchical analysis.
type HierarchyResult struct {
	Root     *InferredSchema // fields common to all levels
	Levels   []LevelSchema   // per-level schema overrides
	Detected bool            // whether hierarchy was found (≥2 levels)
}

// LevelSchema pairs a level with its inferred schema.
type LevelSchema struct {
	Level    Level
	Schema   *InferredSchema              // full inferred schema for this level
	OnlyHere map[string]rules.SchemaField // fields with values only at this level
	StemPath string                       // relative path for .stem placement
}

// levelKey groups directories by (depth, prefix, digits).
type levelKey struct {
	depth  int
	prefix string
	digits int
}

// DetectLevels examines record paths to find recurring directory naming patterns.
// Also detects file-level patterns (e.g., T001-task.md) for leaf records.
// Returns levels ordered by depth (shallowest first). A group must have ≥2
// entries to qualify as a level.
func DetectLevels(records []*extract.Record, scanRoot string) []Level {
	// Track unique directories/files per level key.
	groups := make(map[levelKey]map[string]bool) // key → set of dir/file paths

	for _, rec := range records {
		relPath := toRelPath(rec.Path, scanRoot)

		dir := filepath.Dir(relPath)
		parts := strings.Split(filepath.ToSlash(dir), "/")
		if dir == "." {
			parts = nil
		}

		// Check directory components.
		for depth, part := range parts {
			m := dirPattern.FindStringSubmatch(part)
			if m == nil {
				continue
			}
			prefix := m[1]
			digits := len(m[2])
			key := levelKey{depth: depth, prefix: prefix, digits: digits}

			if groups[key] == nil {
				groups[key] = make(map[string]bool)
			}
			dirPath := strings.Join(parts[:depth+1], "/")
			groups[key][dirPath] = true
		}

		// Check filename for patterns (e.g., T001-task.md).
		baseName := filepath.Base(relPath)
		if m := dirPattern.FindStringSubmatch(baseName); m != nil {
			prefix := m[1]
			digits := len(m[2])
			depth := len(parts) // filename is one level deeper than its dir
			key := levelKey{depth: depth, prefix: prefix, digits: digits}

			if groups[key] == nil {
				groups[key] = make(map[string]bool)
			}
			groups[key][filepath.ToSlash(relPath)] = true
		}
	}

	// Build levels from groups with ≥2 directories.
	var levels []Level
	for key, dirs := range groups {
		if len(dirs) < 2 {
			continue
		}
		dirList := make([]string, 0, len(dirs))
		for d := range dirs {
			dirList = append(dirList, d)
		}
		sort.Strings(dirList)

		levels = append(levels, Level{
			Prefix:   key.prefix,
			Digits:   key.digits,
			Depth:    key.depth,
			DirPaths: dirList,
		})
	}

	// Sort by depth ascending, then prefix for stability.
	sort.Slice(levels, func(i, j int) bool {
		if levels[i].Depth != levels[j].Depth {
			return levels[i].Depth < levels[j].Depth
		}
		return levels[i].Prefix < levels[j].Prefix
	})

	// Assign records to deepest matching level.
	assignRecordsToLevels(records, levels, scanRoot)

	return levels
}

// toRelPath returns a relative path from scanRoot. If the path is already
// relative (as returned by index.Scan), it is returned as-is.
func toRelPath(path, scanRoot string) string {
	if filepath.IsAbs(path) {
		rel, err := filepath.Rel(scanRoot, path)
		if err != nil {
			return path
		}
		return rel
	}
	return path
}

// assignRecordsToLevels assigns each record to its deepest matching level.
func assignRecordsToLevels(records []*extract.Record, levels []Level, scanRoot string) {
	for _, rec := range records {
		relPath := toRelPath(rec.Path, scanRoot)

		dir := filepath.Dir(relPath)
		parts := strings.Split(filepath.ToSlash(dir), "/")
		if dir == "." {
			parts = nil
		}

		// Find deepest matching level (check dir components + filename).
		bestIdx := -1
		for i := range levels {
			// Check directory component at this depth.
			if levels[i].Depth < len(parts) {
				part := parts[levels[i].Depth]
				m := dirPattern.FindStringSubmatch(part)
				if m != nil && m[1] == levels[i].Prefix && len(m[2]) == levels[i].Digits {
					bestIdx = i
				}
			}
			// Check filename as a level (depth == len(parts)).
			if levels[i].Depth == len(parts) {
				baseName := filepath.Base(relPath)
				m := dirPattern.FindStringSubmatch(baseName)
				if m != nil && m[1] == levels[i].Prefix && len(m[2]) == levels[i].Digits {
					bestIdx = i
				}
			}
		}

		if bestIdx >= 0 {
			levels[bestIdx].Records = append(levels[bestIdx].Records, rec)
		}
	}
}

// detectSequencePattern examines directory names to infer a sequence pattern.
// Returns (prefix, digits, detected).
func detectSequencePattern(dirNames []string) (string, int, bool) {
	if len(dirNames) == 0 {
		return "", 0, false
	}

	var prefix string
	var digits int

	for _, name := range dirNames {
		base := filepath.Base(name)
		m := dirPattern.FindStringSubmatch(base)
		if m == nil {
			continue
		}
		if prefix == "" {
			prefix = m[1]
			digits = len(m[2])
		} else if m[1] != prefix || len(m[2]) != digits {
			return "", 0, false // inconsistent
		}
	}

	if prefix == "" {
		return "", 0, false
	}
	return prefix, digits, true
}

// AnalyzeHierarchy detects hierarchy levels and runs per-level analysis.
// Returns Detected=false if fewer than 2 levels are found.
func AnalyzeHierarchy(records []*extract.Record, scanRoot string) *HierarchyResult {
	levels := DetectLevels(records, scanRoot)

	if len(levels) < 2 {
		return &HierarchyResult{Detected: false}
	}

	// Analyze each level separately.
	levelSchemas := make([]LevelSchema, len(levels))
	for i, level := range levels {
		var schema *InferredSchema
		if len(level.Records) > 0 {
			schema = Analyze(level.Records)
		} else {
			schema = &InferredSchema{
				Fields: make(map[string]*FieldStats),
				Schema: make(map[string]rules.SchemaField),
			}
		}

		levelSchemas[i] = LevelSchema{
			Level:    level,
			Schema:   schema,
			OnlyHere: make(map[string]rules.SchemaField),
		}
	}

	// Compute field distribution: which fields belong at root vs per-level.
	root := distributeFields(levelSchemas)

	// Add sequence id to each level's OnlyHere.
	for i := range levelSchemas {
		prefix, digits, ok := detectSequencePattern(levelSchemas[i].Level.DirPaths)
		if ok {
			levelSchemas[i].OnlyHere["id"] = rules.SchemaField{
				Type:   "sequence",
				Prefix: prefix,
				Digits: digits,
			}
		}
	}

	// Compute StemPath for each level.
	computeStemPaths(levelSchemas)

	return &HierarchyResult{
		Root:     root,
		Levels:   levelSchemas,
		Detected: true,
	}
}

// distributeFields determines which fields go to the root .stem and which
// are level-specific. A field goes to root if it has real values at ALL levels.
// Otherwise it goes to OnlyHere of the level(s) where it has values.
func distributeFields(levelSchemas []LevelSchema) *InferredSchema {
	root := &InferredSchema{
		Fields: make(map[string]*FieldStats),
		Schema: make(map[string]rules.SchemaField),
	}

	// Collect all field names across levels.
	fieldPresence := make(map[string][]int) // field → indices of levels that have it
	for i, ls := range levelSchemas {
		for name := range ls.Schema.Schema {
			fieldPresence[name] = append(fieldPresence[name], i)
		}
	}

	nLevels := len(levelSchemas)
	for name, indices := range fieldPresence {
		if name == "id" {
			// id is always per-level (handled separately as sequence).
			continue
		}

		if len(indices) == nLevels && fieldsCompatible(name, levelSchemas) {
			// Present at all levels with compatible definition → root.
			// Use the most complete definition (from the level with most records).
			bestIdx := indices[0]
			for _, idx := range indices[1:] {
				if len(levelSchemas[idx].Level.Records) > len(levelSchemas[bestIdx].Level.Records) {
					bestIdx = idx
				}
			}
			root.Schema[name] = levelSchemas[bestIdx].Schema.Schema[name]
		} else {
			// Present at some levels → goes to OnlyHere of those levels.
			for _, idx := range indices {
				levelSchemas[idx].OnlyHere[name] = levelSchemas[idx].Schema.Schema[name]
			}
		}
	}

	return root
}

// fieldsCompatible checks if a field has the same type across all levels.
// For enum fields, it additionally checks that value sets overlap — disjoint
// value sets indicate genuinely different fields (e.g. tipo=[feature,historia]
// at one level vs tipo=[software-module,ci-cd] at another).
func fieldsCompatible(name string, levelSchemas []LevelSchema) bool {
	var baseType string
	for _, ls := range levelSchemas {
		sf, ok := ls.Schema.Schema[name]
		if !ok {
			return false
		}
		if baseType == "" {
			baseType = sf.Type
		} else if sf.Type != baseType {
			return false
		}
	}

	// For enums, check that all pairs of levels share at least one value.
	// Disjoint value sets → incompatible (genuinely different fields).
	// Overlapping sets → compatible (same field, different samples).
	if baseType == "enum" {
		for i := 0; i < len(levelSchemas); i++ {
			for j := i + 1; j < len(levelSchemas); j++ {
				sfI, okI := levelSchemas[i].Schema.Schema[name]
				sfJ, okJ := levelSchemas[j].Schema.Schema[name]
				if !okI || !okJ {
					continue
				}
				if len(sfI.Values) > 0 && len(sfJ.Values) > 0 && !enumValuesOverlap(sfI.Values, sfJ.Values) {
					return false
				}
			}
		}
	}

	return true
}

// enumValuesOverlap returns true if two string slices share at least one element.
func enumValuesOverlap(a, b []string) bool {
	set := make(map[string]bool, len(a))
	for _, v := range a {
		set[v] = true
	}
	for _, v := range b {
		if set[v] {
			return true
		}
	}
	return false
}

// computeStemPaths sets the relative StemPath for each level.
// Level 0 gets "." (root). Level N gets the pattern for depth N-1 parent dirs.
func computeStemPaths(levelSchemas []LevelSchema) {
	for i := range levelSchemas {
		if i == 0 {
			// The first level's sequence belongs in the root .stem.
			levelSchemas[i].StemPath = "."
		} else {
			// Level N's .stem goes into the parent directories (level N-1 dirs).
			// Use the first dir as a representative pattern.
			prev := levelSchemas[i-1]
			if len(prev.Level.DirPaths) > 0 {
				levelSchemas[i].StemPath = fmt.Sprintf("%s*-*/", prev.Level.Prefix)
			}
		}
	}
}

// ToLevelsMap converts the HierarchyResult into a map[string]*rules.HierarchyLevel
// suitable for the .stem `levels:` field. Each detected level becomes a named entry
// with match pattern, children chain, and per-level schema fields.
func (h *HierarchyResult) ToLevelsMap() map[string]*rules.HierarchyLevel {
	if !h.Detected || len(h.Levels) == 0 {
		return nil
	}

	result := make(map[string]*rules.HierarchyLevel, len(h.Levels))
	names := make([]string, len(h.Levels))

	for i, ls := range h.Levels {
		name := levelName(ls.Level.Prefix)
		names[i] = name

		matchPattern := fmt.Sprintf("%s%s-*", ls.Level.Prefix, strings.Repeat("?", ls.Level.Digits))

		level := &rules.HierarchyLevel{
			Match:  matchPattern,
			Schema: make(map[string]rules.SchemaField),
		}

		for fieldName, field := range ls.OnlyHere {
			level.Schema[fieldName] = field
		}

		result[name] = level
	}

	// Build children chains: each level's children is the next level.
	for i := 0; i < len(names)-1; i++ {
		result[names[i]].Children = []string{names[i+1]}
	}

	return result
}

// ToMatchSchema converts the HierarchyResult into a flat map of SchemaFields
// using match-based field scoping (v2 format). Root fields have no Match set,
// per-level fields get Match patterns, and sequence id gets a map-form Match.
func (h *HierarchyResult) ToMatchSchema() map[string]rules.SchemaField {
	if !h.Detected || len(h.Levels) == 0 {
		return nil
	}

	result := make(map[string]rules.SchemaField)

	// Root fields: no match restriction (applies everywhere).
	for name, field := range h.Root.Schema {
		result[name] = field
	}

	// Per-level fields: set match to the level's glob pattern.
	// Collect per-level fields, merging across levels when the same field
	// appears at multiple (but not all) levels.
	perLevelFields := make(map[string][]string)        // field → patterns
	perLevelDefs := make(map[string]rules.SchemaField) // field → best definition

	for _, ls := range h.Levels {
		glob := fmt.Sprintf("%s*", ls.Level.Prefix)
		for name, field := range ls.OnlyHere {
			if name == "id" {
				continue // handled separately
			}
			perLevelFields[name] = append(perLevelFields[name], glob)
			// Keep the definition with the most values (most complete).
			if existing, ok := perLevelDefs[name]; !ok || len(field.Values) > len(existing.Values) {
				perLevelDefs[name] = field
			}
		}
	}

	for name, patterns := range perLevelFields {
		field := perLevelDefs[name]
		field.Match = &rules.FieldMatch{Patterns: patterns}
		result[name] = field
	}

	// Sequence id: map-form match with per-level prefix/digits.
	idConfigs := make(map[string]any)
	for _, ls := range h.Levels {
		idField, ok := ls.OnlyHere["id"]
		if !ok || idField.Type != "sequence" {
			continue
		}
		glob := fmt.Sprintf("%s*", ls.Level.Prefix)
		idConfigs[glob] = map[string]any{
			"prefix": idField.Prefix,
			"digits": idField.Digits,
		}
	}
	if len(idConfigs) > 0 {
		result["id"] = rules.SchemaField{
			Type:  "sequence",
			Match: &rules.FieldMatch{Configs: idConfigs},
		}
	}

	return result
}

// levelName converts a prefix letter to a human-readable level name.
func levelName(prefix string) string {
	conventions := map[string]string{
		"E": "epic",
		"F": "feature",
		"S": "story",
		"T": "task",
	}
	if name, ok := conventions[prefix]; ok {
		return name
	}
	return strings.ToLower(prefix)
}
