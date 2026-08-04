package rules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

// A bare wikilink is the canonical Rootline form: [[b]] next to b.md must
// resolve. This is the headline case of issue #62 sub-defect 1.
func TestResolveLinkTarget_WikilinkInfersMarkdownExtension(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "b.md"), "# B\n")
	writeFile(t, filepath.Join(dir, "t.md"), "body")

	got := ResolveLinkTarget(ResolveRequest{
		BaseDir: dir, Target: "b", Style: extract.StyleWikilink,
	})
	if !got.OK {
		t.Fatalf("[[b]] should resolve to b.md, got %+v", got)
	}
	if want := filepath.Join(dir, "b.md"); got.Path != want {
		t.Errorf("Path = %q, want %q", got.Path, want)
	}
}

// The .md inference must survive a directory prefix. Issue #62 sub-defect 10
// is exactly this asymmetry: [[b]] worked while [[sub/README]] did not.
func TestResolveLinkTarget_WikilinkInferenceWithDirectoryPrefix(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "sub", "README.md"), "# Sub\n")
	writeFile(t, filepath.Join(dir, "t.md"), "body")

	got := ResolveLinkTarget(ResolveRequest{
		BaseDir: dir, Target: "sub/README", Style: extract.StyleWikilink,
	})
	if !got.OK {
		t.Fatalf("[[sub/README]] should resolve, got %+v", got)
	}
	if want := filepath.Join(dir, "sub", "README.md"); got.Path != want {
		t.Errorf("Path = %q, want %q", got.Path, want)
	}
}

// Inference is style-aware: markdown destinations carry their extension by
// convention, and ADO resolves them literally. Keep that strict.
func TestResolveLinkTarget_MarkdownDoesNotInferExtension(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "b.md"), "# B\n")

	got := ResolveLinkTarget(ResolveRequest{
		BaseDir: dir, Target: "b", Style: extract.StyleMarkdown,
	})
	if got.OK {
		t.Errorf("markdown target %q must not infer .md, got %+v", "b", got)
	}
}

// Behavior inherited from resolveCaseSensitive must survive the extension.
func TestResolveLinkTarget_PreservesStrictSemantics(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "guides", "README.md"), "# Guides\n")
	writeFile(t, filepath.Join(dir, "Setup.md"), "# Setup\n")
	writeFile(t, filepath.Join(dir, "my page.md"), "# P\n")

	t.Run("directory resolves to README", func(t *testing.T) {
		got := ResolveLinkTarget(ResolveRequest{BaseDir: dir, Target: "guides/", Style: extract.StyleMarkdown})
		if !got.OK {
			t.Errorf("directory with README should resolve, got %+v", got)
		}
	})
	t.Run("case mismatch stays broken", func(t *testing.T) {
		got := ResolveLinkTarget(ResolveRequest{BaseDir: dir, Target: "setup.md", Style: extract.StyleMarkdown})
		if got.OK {
			t.Errorf("case mismatch must stay broken (ADO/git are case-sensitive), got %+v", got)
		}
	})
	t.Run("percent-20 decodes", func(t *testing.T) {
		got := ResolveLinkTarget(ResolveRequest{BaseDir: dir, Target: "my%20page.md", Style: extract.StyleMarkdown})
		if !got.OK {
			t.Errorf("%%20 target should resolve, got %+v", got)
		}
	})
	t.Run("suggestion offered for near miss", func(t *testing.T) {
		got := ResolveLinkTarget(ResolveRequest{BaseDir: dir, Target: "Setpu.md", Style: extract.StyleMarkdown})
		if got.OK || got.Suggestion != "Setup.md" {
			t.Errorf("expected broken with suggestion Setup.md, got %+v", got)
		}
	})
}

// A wikilink that already carries .md must not become b.md.md.
func TestResolveLinkTarget_WikilinkWithExplicitExtension(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "b.md"), "# B\n")
	got := ResolveLinkTarget(ResolveRequest{BaseDir: dir, Target: "b.md", Style: extract.StyleWikilink})
	if !got.OK || got.Path != filepath.Join(dir, "b.md") {
		t.Errorf("explicit .md wikilink should resolve as-is, got %+v", got)
	}
}

// Root-anchored targets are the idiomatic form in an ADO code wiki — the
// stated target of links.checks. validate used to skip them with a bare
// `continue`, so a dangling /x.md passed validation while graph flagged it
// (issue #62 sub-defect 3).
func TestResolveLinkTarget_RootAnchored(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "docs", "README.md"), "# Docs\n")
	writeFile(t, filepath.Join(root, "deep", "src.md"), "body")

	got := ResolveLinkTarget(ResolveRequest{
		BaseDir: filepath.Join(root, "deep"), Root: root,
		Target: "/docs/README.md", Style: extract.StyleMarkdown,
	})
	if !got.OK {
		t.Fatalf("/docs/README.md should resolve against root, got %+v", got)
	}
	if want := filepath.Join(root, "docs", "README.md"); got.Path != want {
		t.Errorf("Path = %q, want %q", got.Path, want)
	}
}

func TestResolveLinkTarget_RootAnchoredMissingIsBroken(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "src.md"), "body")
	got := ResolveLinkTarget(ResolveRequest{
		BaseDir: root, Root: root, Target: "/nonexistent.md", Style: extract.StyleMarkdown,
	})
	if got.OK {
		t.Errorf("missing root-anchored target must not resolve, got %+v", got)
	}
}

// Without a root there is nothing to anchor to, so the target cannot be
// judged. Better unresolved than anchored somewhere arbitrary.
func TestResolveLinkTarget_RootAnchoredWithoutRootIsUnresolved(t *testing.T) {
	dir := t.TempDir()
	got := ResolveLinkTarget(ResolveRequest{
		BaseDir: dir, Target: "/docs/README.md", Style: extract.StyleMarkdown,
	})
	if got.OK {
		t.Errorf("root-anchored target with no root must not resolve, got %+v", got)
	}
}

func TestSchemaRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".stem"), "version: 2\nroot: true\n")
	writeFile(t, filepath.Join(root, "sub", ".stem"), "version: 2\n")
	writeFile(t, filepath.Join(root, "sub", "a.md"), "body")

	if got := SchemaRoot(filepath.Join(root, "sub", "a.md")); got != root {
		t.Errorf("SchemaRoot = %q, want the boundary %q", got, root)
	}
	if got := SchemaRoot(filepath.Join(t.TempDir(), "orphan.md")); got != "" {
		t.Errorf("SchemaRoot with no governing schema = %q, want \"\"", got)
	}
}

// A link target must never resolve outside the root it is anchored to.
// Removing validate's blanket skip of "/"-prefixed targets newly exposes this
// path to document-controlled text, and rootline already treats path
// containment as an invariant (see internal/fix/contain.go, issue #69).
func TestResolveLinkTarget_CannotEscapeRoot(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(filepath.Dir(root), "outside-"+filepath.Base(root)+".md")
	if err := os.WriteFile(outside, []byte("# Outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(outside) }()
	writeFile(t, filepath.Join(root, "deep", "src.md"), "body")
	base := filepath.Join(root, "deep")
	escape := "../" + filepath.Base(outside)

	for _, tc := range []struct{ name, target string }{
		{"root-anchored", "/" + escape},
		{"relative", "../" + escape},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveLinkTarget(ResolveRequest{
				BaseDir: base, Root: root, Target: tc.target, Style: extract.StyleMarkdown,
			})
			if got.OK {
				t.Errorf("target %q escaped the root and resolved to %q", tc.target, got.Path)
			}
		})
	}
}

// Containment must not reject legitimate parent traversal that stays inside.
func TestResolveLinkTarget_ParentTraversalInsideRootIsAllowed(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.md"), "# A\n")
	writeFile(t, filepath.Join(root, "deep", "src.md"), "body")

	got := ResolveLinkTarget(ResolveRequest{
		BaseDir: filepath.Join(root, "deep"), Root: root, Target: "../a.md", Style: extract.StyleMarkdown,
	})
	if !got.OK {
		t.Errorf("../a.md stays inside root and must resolve, got %+v", got)
	}
}

// A lexical containment check cannot see through a symlink inside the tree
// that points out of it, so containment compares real paths.
func TestResolveLinkTarget_SymlinkCannotEscapeRoot(t *testing.T) {
	root := t.TempDir()
	outsideDir := t.TempDir()
	writeFile(t, filepath.Join(outsideDir, "secret.md"), "# Secret\n")
	writeFile(t, filepath.Join(root, "src.md"), "body")
	if err := os.Symlink(outsideDir, filepath.Join(root, "escape")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	got := ResolveLinkTarget(ResolveRequest{
		BaseDir: root, Root: root, Target: "escape/secret.md", Style: extract.StyleMarkdown,
	})
	if got.OK {
		t.Errorf("symlink escaping the root must not resolve, got %q", got.Path)
	}
}

// A symlink that stays inside the root is legitimate and must still resolve.
func TestResolveLinkTarget_SymlinkInsideRootResolves(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "real", "page.md"), "# Page\n")
	writeFile(t, filepath.Join(root, "src.md"), "body")
	if err := os.Symlink(filepath.Join(root, "real"), filepath.Join(root, "alias")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	got := ResolveLinkTarget(ResolveRequest{
		BaseDir: root, Root: root, Target: "alias/page.md", Style: extract.StyleMarkdown,
	})
	if !got.OK {
		t.Errorf("symlink inside the root must resolve, got %+v", got)
	}
}
