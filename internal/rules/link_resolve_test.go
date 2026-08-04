package rules

import (
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
