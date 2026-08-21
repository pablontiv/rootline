package rules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func TestCheckLinksProspectiveTargetHardLinkAliasUsesProspectiveBytes(t *testing.T) {
	root := t.TempDir()
	realTarget := filepath.Join(root, "real.md")
	aliasTarget := filepath.Join(root, "alias.md")
	writeFile(t, realTarget, "# Real\n\n## Old Anchor\n")
	createHardLinkOrSkip(t, realTarget, aliasTarget)

	schema := wikiChecksSchema(LinkChecks{Resolve: boolPtr(true), Anchors: true})
	cache := NewHeadingCache()
	oldLink := extract.Link{Target: "real", Anchor: "old-anchor", Type: "reference", Style: extract.StyleWikilink, Line: 1}
	if errs := CheckLinks([]extract.Link{oldLink}, schema, aliasTarget, root, cache); len(errs) != 0 {
		t.Fatalf("fixture old anchor should validate and populate cache: %+v", errs)
	}

	overlay := ProspectiveLinkTarget{AbsPath: aliasTarget, Content: []byte("# Real\n\n## New Anchor\n")}
	newLink := oldLink
	newLink.Anchor = "new-anchor"
	if errs := CheckLinksWithProspectiveTarget([]extract.Link{newLink}, schema, aliasTarget, root, cache, overlay); len(errs) != 0 {
		t.Fatalf("hard-link alias must use prospective bytes selected through real path, got %+v", errs)
	}
	if errs := CheckLinksWithProspectiveTarget([]extract.Link{oldLink}, schema, aliasTarget, root, cache, overlay); len(errs) != 1 || errs[0].Rule != "link_anchor" {
		t.Fatalf("stale cache/disk anchor must not validate for hard-link alias, got %+v", errs)
	}
}

func createHardLinkOrSkip(t *testing.T, oldname, newname string) {
	t.Helper()
	if err := os.Link(oldname, newname); err != nil {
		t.Skipf("hard links unavailable: %v", err)
	}
	oldInfo, err := os.Stat(oldname)
	if err != nil {
		t.Fatalf("stat old hard-link name: %v", err)
	}
	newInfo, err := os.Stat(newname)
	if err != nil {
		t.Fatalf("stat new hard-link name: %v", err)
	}
	if !os.SameFile(oldInfo, newInfo) {
		t.Fatalf("fixture paths are not hard links to the same file")
	}
}
