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
		entry, suggestion, ok := findEntry(cur, comp)
		if !ok {
			// Only the final component may gain an inferred extension:
			// [[b]] means b.md, but a missing intermediate directory is a
			// missing directory, not a file to guess at.
			if inferred, ok2 := inferMarkdown(cur, comp, i == len(comps)-1, req.Style); ok2 {
				cur = filepath.Join(cur, inferred)
				continue
			}
			return LinkResolution{Suggestion: suggestion}
		}
		cur = filepath.Join(cur, entry)
	}

	info, err := os.Stat(cur)
	if err != nil {
		return LinkResolution{}
	}
	if info.IsDir() {
		entry, suggestion, ok := findEntry(cur, "README.md")
		if !ok {
			return LinkResolution{Suggestion: suggestion}
		}
		cur = filepath.Join(cur, entry)
	}
	return LinkResolution{Path: cur, OK: true}
}

// inferMarkdown retries a missing final path component with a ".md" suffix.
//
// This is what makes [[b]] resolve to b.md. It is deliberately limited to
// wikilinks: a markdown destination carries its extension by convention and
// ADO resolves it literally, so inferring one there would accept links the
// published wiki rejects.
func inferMarkdown(dir, comp string, isFinal bool, style string) (string, bool) {
	if !isFinal || style != extract.StyleWikilink || strings.HasSuffix(comp, ".md") {
		return "", false
	}
	entry, _, ok := findEntry(dir, comp+".md")
	if !ok {
		return "", false
	}
	return entry, true
}

// findEntry looks for an exact-case directory entry, returning a fuzzy
// suggestion from the directory's entries when absent.
func findEntry(dir, name string) (string, string, bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", "", false
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.Name() == name {
			return name, "", true
		}
		names = append(names, e.Name())
	}
	return "", fuzzy.Match(name, names), false
}
