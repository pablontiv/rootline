package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

// ToV2Result is the versioned JSON output for migrate --to-v2.
type ToV2Result struct {
	Version int      `json:"version"`
	Kind    string   `json:"kind"`
	Updated []string `json:"updated"`
	Skipped int      `json:"skipped"`
	Total   int      `json:"total"`
}

// UpgradeToV2 upgrades all .stem files under dir from version 0/1 to version 2.
// It preserves YAML comments and formatting by operating on the AST.
func UpgradeToV2(dir string, dryRun bool) (*ToV2Result, error) {
	stems, err := FindStemFiles(dir)
	if err != nil {
		return nil, fmt.Errorf("finding stem files: %w", err)
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolving dir: %w", err)
	}

	result := &ToV2Result{
		Version: 1,
		Kind:    "rootline/migrate-to-v2",
	}

	for _, stemPath := range stems {
		result.Total++

		data, err := os.ReadFile(stemPath)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", stemPath, err)
		}

		updated, newData, err := upgradeFileToV2(data)
		if err != nil {
			return nil, fmt.Errorf("upgrading %s: %w", stemPath, err)
		}

		if !updated {
			result.Skipped++
			continue
		}

		relPath, _ := filepath.Rel(absDir, stemPath)
		result.Updated = append(result.Updated, relPath)

		if !dryRun {
			if err := os.WriteFile(stemPath, newData, 0644); err != nil {
				return nil, fmt.Errorf("writing %s: %w", stemPath, err)
			}
		}
	}

	return result, nil
}

// upgradeFileToV2 parses YAML content and upgrades the version field to 2.
// Returns (true, newContent, nil) if upgraded, (false, nil, nil) if already v2.
func upgradeFileToV2(data []byte) (bool, []byte, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return false, nil, fmt.Errorf("parsing YAML: %w", err)
	}

	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return false, nil, fmt.Errorf("unexpected YAML structure")
	}

	mapping := doc.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return false, nil, fmt.Errorf("expected mapping at top level")
	}

	// Search for existing version key.
	for i := 0; i < len(mapping.Content)-1; i += 2 {
		key := mapping.Content[i]
		val := mapping.Content[i+1]

		if key.Value == "version" {
			ver, err := strconv.Atoi(val.Value)
			if err != nil {
				return false, nil, fmt.Errorf("non-integer version: %s", val.Value)
			}
			if ver >= 2 {
				return false, nil, nil // already v2+
			}
			val.Value = "2"
			out, err := yaml.Marshal(&doc)
			if err != nil {
				return false, nil, fmt.Errorf("marshaling: %w", err)
			}
			return true, out, nil
		}
	}

	// No version field found — insert as first key-value pair.
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "version", Tag: "!!str"}
	valNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "2", Tag: "!!int"}
	mapping.Content = append([]*yaml.Node{keyNode, valNode}, mapping.Content...)

	out, err := yaml.Marshal(&doc)
	if err != nil {
		return false, nil, fmt.Errorf("marshaling: %w", err)
	}
	return true, out, nil
}
