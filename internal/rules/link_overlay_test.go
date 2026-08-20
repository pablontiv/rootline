package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func TestCheckLinksProspectiveTargetExistingLiteralOutranksInferredMarkdownForAnchors(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "foo"), "# Foo\n\n## Existing Anchor\n")
	target := filepath.Join(dir, "foo.md")
	schema := wikiChecksSchema(LinkChecks{Resolve: boolPtr(true), Anchors: true})
	overlay := ProspectiveLinkTarget{AbsPath: target, Content: []byte("# Foo\n\n## Prospective Anchor\n")}

	link := extract.Link{Target: "foo", Anchor: "existing-anchor", Type: "reference", Style: extract.StyleWikilink, Line: 1}
	if errs := CheckLinksWithProspectiveTarget([]extract.Link{link}, schema, target, dir, nil, overlay); len(errs) != 0 {
		t.Fatalf("literal existing foo must be selected before prospective foo.md, got %+v", errs)
	}

	link.Anchor = "prospective-anchor"
	if errs := CheckLinksWithProspectiveTarget([]extract.Link{link}, schema, target, dir, nil, overlay); len(errs) != 1 || errs[0].Rule != "link_anchor" {
		t.Fatalf("anchor must be checked against selected literal foo, not prospective foo.md, got %+v", errs)
	}
}

func TestCheckLinksProspectiveTargetExactCaseMismatchIsBroken(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.md")
	schema := wikiChecksSchema(LinkChecks{Resolve: boolPtr(true)})
	overlay := ProspectiveLinkTarget{AbsPath: target, Content: []byte("# Target\n")}

	errs := CheckLinksWithProspectiveTarget([]extract.Link{wikiLink("Target")}, schema, target, dir, nil, overlay)
	if len(errs) != 1 || errs[0].Rule != "link_resolve" {
		t.Fatalf("wrong-case prospective target must stay broken, got %+v", errs)
	}
}

func TestCheckLinksProspectiveTargetNormalizesInternalSymlinkAlias(t *testing.T) {
	root := t.TempDir()
	realDir := filepath.Join(root, "real")
	if err := os.Mkdir(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	aliasDir := filepath.Join(root, "alias")
	if err := os.Symlink(realDir, aliasDir); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	source := filepath.Join(root, "source.md")
	writeFile(t, source, "body")

	aliasTarget := filepath.Join(aliasDir, "target.md")
	schema := wikiChecksSchema(LinkChecks{Resolve: boolPtr(true), Anchors: true})
	overlay := ProspectiveLinkTarget{AbsPath: aliasTarget, Content: []byte("# Target\n\n## Alias Anchor\n")}
	link := extract.Link{Target: "real/target", Anchor: "alias-anchor", Type: "reference", Style: extract.StyleWikilink, Line: 1}

	if errs := CheckLinksWithProspectiveTarget([]extract.Link{link}, schema, source, root, nil, overlay); len(errs) != 0 {
		t.Fatalf("prospective target reached through an internal symlink alias should resolve by physical identity, got %+v", errs)
	}
}

func TestCheckLinksProspectiveTargetNormalizesVarPrivateVarAlias(t *testing.T) {
	physicalRoot, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Skipf("temp dir has no physical alias: %v", err)
	}
	if !strings.HasPrefix(physicalRoot, "/private/var/") {
		t.Skipf("platform does not expose /var -> /private/var temp dirs: %s", physicalRoot)
	}
	aliasRoot := strings.TrimPrefix(physicalRoot, "/private")
	if _, err := os.Stat(aliasRoot); err != nil {
		t.Skipf("/var alias unavailable: %v", err)
	}
	source := filepath.Join(physicalRoot, "source.md")
	writeFile(t, source, "body")

	schema := wikiChecksSchema(LinkChecks{Resolve: boolPtr(true), Anchors: true})
	overlay := ProspectiveLinkTarget{
		AbsPath: filepath.Join(aliasRoot, "target.md"),
		Content: []byte("# Target\n\n## Var Alias Anchor\n"),
	}
	link := extract.Link{Target: "target", Anchor: "var-alias-anchor", Type: "reference", Style: extract.StyleWikilink, Line: 1}

	if errs := CheckLinksWithProspectiveTarget([]extract.Link{link}, schema, source, physicalRoot, nil, overlay); len(errs) != 0 {
		t.Fatalf("/var and /private/var identities should resolve consistently, got %+v", errs)
	}
}
