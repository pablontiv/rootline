package main

import (
	"strings"
	"testing"
)

func TestSkillDistributionDocumentationContract(t *testing.T) {
	repo := documentationContractRepoRoot(t)
	required := map[string][]string{
		"CHANGELOG.md": {
			"**BREAKING**", "~/.config/opencode/skills/rootline", "~/.agents/skills/rootline", "rootline skill install", "rootline skill status",
		},
		"docs/skill.md": {
			"rootline skill install", "--approve", "rootline skill uninstall", "rootline skill restore", "receipts.jsonl",
		},
		".claude/skills/rootline/SKILL.md": {
			"CHANGELOG.md", "rootline skill status", "~/.agents/skills/rootline",
		},
		"README.md": {"docs/skill.md", "rootline skill install"},
		"CLAUDE.md": {"rootline skill", "internal/skilldist"},
	}
	for path, needles := range required {
		text := string(documentationContractRead(t, repo, path))
		for _, needle := range needles {
			if !strings.Contains(text, needle) {
				t.Errorf("%s missing %q", path, needle)
			}
		}
	}
}
