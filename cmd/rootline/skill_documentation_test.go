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

func TestSkillGovernanceBoundaryDocumentationContract(t *testing.T) {
	repo := documentationContractRepoRoot(t)
	path := ".claude/skills/rootline/SKILL.md"
	text := string(documentationContractRead(t, repo, path))
	boundary := sectionFrom(text, "## Governance Boundary (root marker)")
	assertScopeContainsAll(t, path+" governance boundary", boundary, []string{
		"`.git` is not a boundary",
		"`query` without `--sort`, `tree`, `graph`, and `graph --check` work when no `.stem` files are discovered",
		"`validate`, `fix`, `describe`, `explain`, `new`, `set`, and `stats` require a `.stem` schema",
		"`stats` is not a schemaless counting alternative",
		"`query --sort` requires a `.stem` schema, even when sorting a built-in field such as `path`",
		"any discovered `.stem` chain must be parseable and terminate at `root: true`",
		"A malformed `.stem` always fails, including for exploration commands",
		"an existing chain without `root: true` is not schemaless mode",
		"Without a terminal (non-TTY), the command exits non-zero",
		"On a terminal (TTY), Rootline offers to add the marker",
		"accepting writes `root: true` to the proposed project root's `.stem` and continues",
		"Do not accept that prompt or write the marker without explicit user authorization",
		"Do **not** use `init --force`",
		"Bootstrap commands (`schema propose`, `analyze`) work without any `.stem`",
		"`init` and `migrate` also run without a marker",
		"temporary directory whose ancestors have no `.stem` files",
		"dir=\"$(mktemp -d)\"",
		"printf '%s\\n' '# First' '[[second]]' > \"$dir/first.md\"",
		"printf '%s\\n' '# Second' > \"$dir/second.md\"",
		"rootline query \"$dir\" --where \"type == 'markdown'\" --select path -o json",
		"rootline tree \"$dir\" -o json",
		"rootline graph \"$dir\" -o json",
		"rootline graph \"$dir\" --check",
	})
	for _, stale := range []string{
		"Governed commands (`validate`, `fix`, `query`, `tree`, `graph`, `describe`, `explain`, `set`, `stats`) fail on a tree with no `.stem`",
		"fail on a tree with no `.stem` or an unparseable one",
	} {
		if strings.Contains(text, stale) {
			t.Errorf("%s contains stale blanket schema requirement %q", path, stale)
		}
	}

	path = ".claude/skills/rootline/ref-query.md"
	text = string(documentationContractRead(t, repo, path))
	assertScopeContainsAll(t, path+" query", sectionFrom(text, "## query"), []string{
		"Without any discovered `.stem`, `query` works",
		"`query --sort` requires a `.stem` schema, even for built-in fields such as `path`",
		"Any discovered schema chain must be parseable and terminate at `root: true`",
	})
	assertScopeContainsAll(t, path+" tree", sectionFrom(text, "## tree"), []string{
		"`tree` works without any discovered `.stem`",
		"any discovered schema chain must be parseable and terminate at `root: true`",
	})
	assertScopeContainsAll(t, path+" stats", sectionFrom(text, "## stats"), []string{
		"`stats` requires a `.stem` schema with a declared `root: true` boundary",
		"it does not support schemaless counting",
		"rootline query <dir> --count -o json",
		"version 2 contract `{version, kind, total}` with kind `rootline/stats`",
		"it does not group counts by `estado` or `tipo`",
	})
	if strings.Contains(text, "JSON contains counts grouped by `estado`, `tipo`, and total records") {
		t.Errorf("%s contains stale grouped stats output contract", path)
	}
	assertScopeContainsAll(t, path+" explain", sectionFrom(text, "## explain"), []string{
		"`explain` requires a `.stem` schema with a declared `root: true` boundary",
	})
}
