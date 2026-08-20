package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type fencedBlock struct{ lang, body string }

func activeDocumentationContractFiles() []string {
	return []string{"README.md", "CLAUDE.md", "docs/analyze.md", "docs/init.md", "docs/validate.md", "docs/migrate.md", "docs/new.md", "docs/query.md", "docs/describe.md", "docs/explain.md", "docs/derivation.md", "docs/set.md", "docs/extensibility.md", "docs/levels.md", ".claude/skills/rootline/SKILL.md", ".claude/skills/rootline/ref-schema.md", ".claude/skills/rootline/ref-validate.md", ".claude/skills/rootline/ref-query.md", ".claude/skills/rootline/ref-advanced.md"}
}

func fencedBlocks(text string) []fencedBlock {
	var blocks []fencedBlock
	lines := strings.Split(text, "\n")
	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimLeft(lines[i], " \t")
		if !strings.HasPrefix(trimmed, "```") {
			continue
		}
		info := strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
		lang := ""
		if fields := strings.Fields(info); len(fields) > 0 {
			lang = strings.ToLower(fields[0])
		}
		var body []string
		i++
		for ; i < len(lines); i++ {
			if strings.HasPrefix(strings.TrimLeft(lines[i], " \t"), "```") {
				break
			}
			body = append(body, lines[i])
		}
		blocks = append(blocks, fencedBlock{lang: lang, body: strings.Join(body, "\n")})
	}
	return blocks
}

func contentOutsideFences(text string) string {
	var out []string
	lines := strings.Split(text, "\n")
	inFence := false
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimLeft(line, " \t"), "```") {
			inFence = !inFence
			continue
		}
		if !inFence {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

var inlineLegacyDeclarationRE = regexp.MustCompile("(?i)(^|[`{,\\s])(([\"']?type[\"']?\\s*:\\s*['\"]?(bool|section)['\"]?([`,}\\s]|$))|([\"']?(heading|ordered|enum)[\"']?\\s*:))")

func TestDocumentationContractInlineLegacyDetectorCoversQuotedKeys(t *testing.T) {
	for _, sample := range []string{`{"type":"section"}`, `{'type':'bool'}`, `"enum": ["old"]`, `'heading': "## Old"`} {
		if !inlineLegacyDeclarationRE.MatchString(sample) {
			t.Fatalf("inline legacy detector missed %s", sample)
		}
	}
}

func assertNoInlineLegacyDeclarations(t *testing.T, path, text string) {
	t.Helper()
	for _, match := range inlineLegacyDeclarationRE.FindAllString(text, -1) {
		t.Errorf("%s active guidance contains legacy inline declaration %q", path, strings.TrimSpace(match))
	}
}

func assertYAMLBlockHasNoLegacyDeclarations(t *testing.T, path string, block fencedBlock) {
	t.Helper()
	if strings.TrimSpace(block.body) == "" {
		return
	}
	node, err := parseDocumentationYAML(block.body)
	if err != nil {
		t.Errorf("%s malformed %s fence: %v\n%s", path, block.lang, err, block.body)
		return
	}
	if len(node.Content) == 0 {
		return
	}
	walkYAML(t, path, node.Content[0], "")
}

func walkYAML(t *testing.T, path string, node *yaml.Node, parentKey string) {
	if node == nil {
		return
	}
	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i].Value
			if key == "heading" || key == "ordered" || key == "enum" {
				t.Errorf("%s YAML example uses legacy key %q", path, key)
			}
			walkYAML(t, path, node.Content[i+1], key)
		}
	case yaml.SequenceNode:
		for _, child := range node.Content {
			walkYAML(t, path, child, parentKey)
		}
	case yaml.ScalarNode:
		if isLegacyTypeScalar(parentKey, node.Value) {
			t.Errorf("%s YAML example uses legacy type %q", path, node.Value)
		}
	}
}

func assertJSONBlockHasCanonicalProvenance(t *testing.T, path string, block fencedBlock) any {
	t.Helper()
	if strings.TrimSpace(block.body) == "" {
		return nil
	}
	var value any
	if err := json.Unmarshal([]byte(block.body), &value); err != nil {
		t.Errorf("%s malformed json fence: %v\n%s", path, err, block.body)
		return nil
	}
	walkJSON(t, path, value, "")
	return value
}

func walkJSON(t *testing.T, path string, value any, parentKey string) {
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			if key == "heading" || key == "ordered" || key == "enum" {
				t.Errorf("%s JSON example uses legacy key %q", path, key)
			}
			if key == "source" {
				if s, ok := child.(string); ok && isPhysicalStemFieldSource(v, s) {
					t.Errorf("%s JSON field uses source for physical stem %q", path, s)
				}
			}
			walkJSON(t, path, child, key)
		}
	case []any:
		for _, child := range v {
			walkJSON(t, path, child, parentKey)
		}
	case string:
		if isLegacyTypeScalar(parentKey, v) {
			t.Errorf("%s JSON example uses legacy scalar %q", path, v)
		}
	}
}

func parseDocumentationYAML(body string) (*yaml.Node, error) {
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(body), &node); err != nil {
		return nil, err
	}
	return &node, nil
}

func isLegacyTypeValue(value string) bool { return value == "section" || value == "bool" }

func isLegacyTypeScalar(parentKey, value string) bool {
	return parentKey == "type" && isLegacyTypeValue(value)
}

func isPhysicalStemFieldSource(obj map[string]any, source string) bool {
	return strings.HasSuffix(source, ".stem") && isFieldObject(obj) && !isValidationDiagnostic(obj)
}

func isFieldObject(obj map[string]any) bool {
	_, hasType := obj["type"]
	_, hasName := obj["name"]
	return hasType || hasName
}

func isValidationDiagnostic(obj map[string]any) bool {
	_, hasMessage := obj["message"]
	_, hasSeverity := obj["severity"]
	_, hasRule := obj["rule"]
	return hasMessage && hasSeverity && hasRule
}

func documentationExactEqual(got, want any) bool { return reflect.DeepEqual(got, want) }

func documentationContractRepoRoot(t *testing.T) string {
	t.Helper()
	path, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
			return path
		}
		next := filepath.Dir(path)
		if next == path {
			t.Fatal("repository root with go.mod not found")
		}
		path = next
	}
}

func documentationContractRead(t *testing.T, repo, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repo, path))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func documentationContractSchema(t *testing.T, repo, path string) map[string]any {
	t.Helper()
	var value map[string]any
	if err := yaml.Unmarshal(documentationContractRead(t, repo, path), &value); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return value
}

func documentationContractMap(t *testing.T, value any, name string) map[string]any {
	t.Helper()
	mapped, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("%s = %#v, want mapping", name, value)
	}
	return mapped
}
