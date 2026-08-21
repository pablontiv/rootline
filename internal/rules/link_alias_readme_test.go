package rules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func TestCheckLinksProspectiveTargetFinalFileAliasUsesProspectiveBytes(t *testing.T) {
	root := t.TempDir()
	realTarget := filepath.Join(root, "real.md")
	aliasTarget := filepath.Join(root, "alias.md")
	writeFile(t, realTarget, "# Real\n\n## Old Anchor\n")
	if err := os.Symlink(realTarget, aliasTarget); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

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
		t.Fatalf("final-file alias must use prospective bytes selected through real path, got %+v", errs)
	}
	if errs := CheckLinksWithProspectiveTarget([]extract.Link{oldLink}, schema, aliasTarget, root, cache, overlay); len(errs) != 1 || errs[0].Rule != "link_anchor" {
		t.Fatalf("stale cache/disk anchor must not validate for final-file alias, got %+v", errs)
	}
}

func TestResolveLinkTargetDirectoryReadmePreservesDiskParity(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, root string)
	}{
		{
			name: "normal README entry",
			setup: func(t *testing.T, root string) {
				writeFile(t, filepath.Join(root, "guides", "README.md"), "# Guides\n")
			},
		},
		{
			name: "dangling README symlink entry",
			setup: func(t *testing.T, root string) {
				if err := os.MkdirAll(filepath.Join(root, "guides"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(root, "missing.md"), filepath.Join(root, "guides", "README.md")); err != nil {
					t.Skipf("symlink unavailable: %v", err)
				}
			},
		},
		{
			name: "README symlink entry outside root",
			setup: func(t *testing.T, root string) {
				outside := t.TempDir()
				writeFile(t, filepath.Join(outside, "README.md"), "# Outside\n")
				if err := os.MkdirAll(filepath.Join(root, "guides"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(outside, "README.md"), filepath.Join(root, "guides", "README.md")); err != nil {
					t.Skipf("symlink unavailable: %v", err)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			tc.setup(t, root)

			got := ResolveLinkTarget(ResolveRequest{BaseDir: root, Root: root, Target: "guides/", Style: extract.StyleMarkdown})
			wantPath := filepath.Join(root, "guides", "README.md")
			if !got.OK || got.Path != wantPath {
				t.Fatalf("directory README entry should preserve base disk resolver outcome, got %+v want OK path %q", got, wantPath)
			}
		})
	}
}
