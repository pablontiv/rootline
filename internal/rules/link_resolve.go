package rules

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/pablontiv/picokit/fuzzy"
	"github.com/pablontiv/rootline/internal/extract"
)

// ResolveRequest describes one link target to resolve.
//
// BaseDir is the directory of the record that carries the link; relative
// targets resolve against it. Root is the scan root and is only consulted for
// root-anchored ("/x.md") targets — leave it empty when no root is in scope,
// in which case a root-anchored target cannot be judged and stays unresolved.
// Style selects style-aware behavior: only wikilinks infer a ".md" extension.
type ResolveRequest struct {
	BaseDir string
	Root    string
	Target  string
	Style   string
}

// LinkResolution is the outcome of resolving one link target.
//
// Path is the absolute filesystem path the target resolved to, set only when
// OK. Suggestion is a fuzzy match for the first unmatched path component and
// may be empty; it is populated on failure so callers can offer "did you
// mean?" without re-walking the directory.
type LinkResolution struct {
	Path       string
	Suggestion string
	OK         bool
}

type linkTargetInfo struct {
	IsDir bool
}

type linkTargetProvider interface {
	readDir(dir string) ([]string, error)
	stat(path string) (linkTargetInfo, error)
}

type diskLinkTargetProvider struct{}

func (diskLinkTargetProvider) readDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names, nil
}

func (diskLinkTargetProvider) stat(path string) (linkTargetInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return linkTargetInfo{}, err
	}
	return linkTargetInfo{IsDir: info.IsDir()}, nil
}

// ResolveLinkTarget is the single authority for "does this link target exist?".
//
// Every command resolves through here so that validate, graph, query traversal
// and fix cannot disagree about whether a link is broken — issue #62 existed
// precisely because validate and graph each had their own resolver. Resolution
// is filesystem-backed, not node-set-backed: a target that exists on disk
// resolves even when the schema does not govern it, because the schema
// declares what is governed, not what exists.
//
// Semantics, in order:
//   - a root-anchored target ("/x.md") rebases onto Root
//   - "%20" and other percent escapes decode
//   - every path component must match case-exactly (APFS is case-insensitive;
//     ADO and git are not)
//   - a wikilink whose final component does not exist retries it with ".md"
//   - a directory target resolves to its README.md
func ResolveLinkTarget(req ResolveRequest) LinkResolution {
	return resolveLinkTarget(req, diskLinkTargetProvider{})
}

func resolveLinkTargetWithProspectiveTarget(req ResolveRequest, overlay *prospectiveLinkOverlay) LinkResolution {
	if overlay == nil {
		return ResolveLinkTarget(req)
	}
	return resolveLinkTarget(req, overlay)
}

func resolveLinkTarget(req ResolveRequest, provider linkTargetProvider) LinkResolution {
	base, target := req.BaseDir, req.Target
	if strings.HasPrefix(target, "/") {
		if req.Root == "" {
			return LinkResolution{}
		}
		base, target = req.Root, strings.TrimPrefix(target, "/")
	}

	decoded, err := url.PathUnescape(target)
	if err != nil {
		decoded = target
	}

	comps := strings.Split(filepath.ToSlash(filepath.Clean(decoded)), "/")
	cur := base
	for i, comp := range comps {
		switch comp {
		case "", ".":
			continue
		case "..":
			cur = filepath.Dir(cur)
			continue
		}
		entry, suggestion, ok := findEntry(provider, cur, comp)
		if !ok {
			// Only the final component may gain an inferred extension:
			// [[b]] means b.md, but a missing intermediate directory is a
			// missing directory, not a file to guess at.
			if inferred, ok2 := inferMarkdown(provider, cur, comp, i == len(comps)-1, req.Style); ok2 {
				cur = filepath.Join(cur, inferred)
				continue
			}
			return LinkResolution{Suggestion: suggestion}
		}
		cur = filepath.Join(cur, entry)
	}

	if !withinRoot(req.Root, cur) {
		return LinkResolution{}
	}

	info, err := provider.stat(cur)
	if err != nil {
		return LinkResolution{}
	}
	if info.IsDir {
		entry, suggestion, ok := findEntry(provider, cur, "README.md")
		if !ok {
			return LinkResolution{Suggestion: suggestion}
		}
		cur = filepath.Join(cur, entry)
	}
	return LinkResolution{Path: cur, OK: true}
}

// canonicalPhysicalPath returns the path identity used for containment and
// prospective overlay matching. It resolves symlinks through the deepest
// existing parent, then appends the still-missing suffix, so absent files are
// compared by the physical directory they would be written into.
func canonicalPhysicalPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	clean := filepath.Clean(abs)
	if real, err := filepath.EvalSymlinks(clean); err == nil {
		return filepath.Clean(real), nil
	}

	var missing []string
	cur := clean
	for {
		if _, err := os.Lstat(cur); err == nil {
			real, err := filepath.EvalSymlinks(cur)
			if err != nil {
				return "", err
			}
			for i := len(missing) - 1; i >= 0; i-- {
				real = filepath.Join(real, missing[i])
			}
			return filepath.Clean(real), nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return clean, nil
		}
		missing = append(missing, filepath.Base(cur))
		cur = parent
	}
}

// PhysicalPathWithin reports whether path's physical identity is contained by
// root's physical identity. Both sides use canonicalPhysicalPath, so symlinked
// roots such as macOS /var -> /private/var and absent target files compare by
// the same authority.
func PhysicalPathWithin(root, path string) (bool, error) {
	if root == "" {
		return true, nil
	}
	realRoot, err := canonicalPhysicalPath(root)
	if err != nil {
		return false, err
	}
	realPath, err := canonicalPhysicalPath(path)
	if err != nil {
		return false, err
	}
	rel, err := filepath.Rel(realRoot, realPath)
	if err != nil {
		return false, err
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)), nil
}

// withinRoot reports whether a resolved path stayed inside root.
//
// Link targets are document-controlled text, and a target may contain ".."
// components, so resolution can otherwise walk above the tree being governed
// and report on files outside it — with links.checks.anchors enabled that
// means reading them. Rootline already treats path containment as an
// invariant elsewhere (internal/fix/contain.go), so resolution honors it too.
// An empty root means no boundary is known and nothing can be judged against
// it; callers that need containment must pass one.
func withinRoot(root, path string) bool {
	ok, err := PhysicalPathWithin(root, path)
	return err == nil && ok
}

// SchemaRoot returns the directory of the root-most .stem governing path — the
// declared governance boundary, or the top of the discovered chain when none
// declares `root: true`.
//
// Root-anchored links ("/x.md") are relative to the wiki root, and a
// single-file `validate` invocation has no scan root to use. The schema
// boundary is the closest thing the engine actually knows about, and it is the
// same directory `validate --all` would be pointed at. Returns "" when no
// schema governs the path, which leaves root-anchored targets unresolved
// rather than silently anchoring them somewhere arbitrary.
func SchemaRoot(path string) string {
	entries, err := WalkUp(path)
	if err != nil || len(entries) == 0 {
		return ""
	}
	return filepath.Dir(entries[0].Path)
}

// inferMarkdown retries a missing final path component with a ".md" suffix.
//
// This is what makes [[b]] resolve to b.md. It is deliberately limited to
// wikilinks: a markdown destination carries its extension by convention and
// ADO resolves it literally, so inferring one there would accept links the
// published wiki rejects.
func inferMarkdown(provider linkTargetProvider, dir, comp string, isFinal bool, style string) (string, bool) {
	if !isFinal || style != extract.StyleWikilink || strings.HasSuffix(comp, ".md") {
		return "", false
	}
	entry, _, ok := findEntry(provider, dir, comp+".md")
	if !ok {
		return "", false
	}
	return entry, true
}

// findEntry looks for an exact-case directory entry, returning a fuzzy
// suggestion from the directory's entries when absent.
func findEntry(provider linkTargetProvider, dir, name string) (string, string, bool) {
	names, err := provider.readDir(dir)
	if err != nil {
		return "", "", false
	}
	for _, entry := range names {
		if entry == name {
			return name, "", true
		}
	}
	return "", fuzzy.Match(name, names), false
}
