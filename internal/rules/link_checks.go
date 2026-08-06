package rules

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/pablontiv/rootline/internal/extract"
)

// HeadingCache caches heading slugs per target file within a validation run.
// Behavior is implemented alongside the anchors check.
type HeadingCache struct {
	slugs map[string][]string
}

// NewHeadingCache creates an empty heading cache.
func NewHeadingCache() *HeadingCache {
	return &HeadingCache{slugs: make(map[string][]string)}
}

// CheckLinks runs filesystem-backed link checks (links.checks in .stem)
// against a record's links. sourceAbsPath is the absolute path of the record
// file; relative targets resolve against its directory. root is the scan root
// that root-anchored ("/x.md") targets rebase onto — pass "" when no root is
// in scope and such targets stay unresolved. Links whose style is not in the
// schema's effective styles are skipped.
//
// Resolution runs through ResolveLinkTarget, the same entry point graph and
// query use, so the commands cannot disagree about whether a link is broken.
func CheckLinks(links []extract.Link, schema LinkSchema, sourceAbsPath, root string, cache *HeadingCache) []ValidationError {
	// Broken-target detection needs no opt-in, so a schema with no checks
	// block still resolves. Anchors and encoding remain opt-in and are
	// guarded individually below.
	resolve := schema.ShouldResolve()
	anchors := schema.Checks != nil && schema.Checks.Anchors
	encoding := schema.Checks != nil && schema.Checks.Encoding
	if !resolve && !anchors && !encoding {
		return nil
	}

	styles := make(map[string]bool)
	for _, s := range schema.EffectiveStyles() {
		styles[s] = true
	}

	var errs []ValidationError
	for _, link := range links {
		if !styles[linkStyle(link)] {
			continue
		}

		if encoding && strings.Contains(link.Target, " ") {
			errs = append(errs, ValidationError{
				Rule:     "link_encoding",
				Field:    "links",
				Message:  fmt.Sprintf("link target %q contains unencoded spaces (use %%20)", link.Target),
				Source:   "links.checks",
				Severity: "error",
			})
		}

		if resolve || anchors {
			res := ResolveLinkTarget(ResolveRequest{
				BaseDir: filepath.Dir(sourceAbsPath),
				Root:    root,
				Target:  link.Target,
				Style:   linkStyle(link),
			})
			if !res.OK {
				// Basename fallback matches a target against every record in
				// the tree, and CheckLinks sees one record at a time. Calling
				// the link broken would be wrong — graph may well resolve it —
				// and staying silent would put the two commands back into
				// disagreement, so say plainly that this one is undecidable
				// here and name the command that can decide it.
				if schema.BasenameFallback && resolve {
					errs = append(errs, ValidationError{
						Rule:     "link_unverifiable",
						Field:    "links",
						Message:  fmt.Sprintf("link target %q cannot be verified: links.basename_fallback matches against every record, which this command does not scan (check it with 'rootline graph --check')", link.Target),
						Source:   "links.checks",
						Severity: "warning",
					})
					continue
				}
				if resolve {
					msg := fmt.Sprintf("link target %q does not resolve to an existing file (case-sensitive)", link.Target)
					if res.Suggestion != "" {
						msg += fmt.Sprintf(" (did you mean %q?)", res.Suggestion)
					}
					errs = append(errs, ValidationError{
						Rule:       "link_resolve",
						Field:      "links",
						Message:    msg,
						Source:     "links.checks",
						Severity:   "error",
						Suggestion: res.Suggestion,
					})
				}
				continue
			}
			if anchors && link.Anchor != "" {
				errs = append(errs, checkAnchor(link, res.Path, cache)...)
			}
		}
	}
	return errs
}

// checkAnchor verifies the link's anchor matches a heading slug in the
// resolved target file.
func checkAnchor(link extract.Link, resolvedPath string, cache *HeadingCache) []ValidationError {
	if cache == nil {
		cache = NewHeadingCache()
	}
	slugs, err := cache.headingSlugs(resolvedPath)
	if err != nil {
		return nil // unreadable/non-markdown target: resolve check already covers existence
	}
	want, err := url.PathUnescape(link.Anchor)
	if err != nil {
		want = link.Anchor
	}
	want = strings.ToLower(want)
	for _, s := range slugs {
		if s == want {
			return nil
		}
	}
	return []ValidationError{{
		Rule:     "link_anchor",
		Field:    "links",
		Message:  fmt.Sprintf("anchor %q not found in %q", link.Anchor, filepath.Base(resolvedPath)),
		Source:   "links.checks",
		Severity: "error",
	}}
}

// headingSlugs returns the slugified headings of a markdown file, cached.
func (c *HeadingCache) headingSlugs(absPath string) ([]string, error) {
	if slugs, ok := c.slugs[absPath]; ok {
		return slugs, nil
	}
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	parseAST := true
	ext := &extract.MarkdownExtractor{ParseAST: &parseAST}
	rec, err := ext.Extract(absPath, content)
	if err != nil {
		return nil, err
	}
	var slugs []string
	if rec.AST != nil {
		for _, sec := range extract.ExtractSections(rec.AST, []byte(rec.Body)) {
			slugs = append(slugs, slugifyHeading(sec.Heading))
		}
	}
	c.slugs[absPath] = slugs
	return slugs, nil
}

// slugifyHeading converts a heading to its anchor slug: lowercase, spaces and
// hyphens become hyphens (not collapsed), punctuation is dropped, letters,
// digits and underscores are kept (GitHub/ADO code-wiki convention).
func slugifyHeading(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	var b strings.Builder
	for _, r := range h {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_':
			b.WriteRune(r)
		case r == ' ' || r == '-':
			b.WriteByte('-')
		}
	}
	return b.String()
}

// Resolution lives in link_resolve.go: every command shares one resolver so
// that validate, graph and query cannot disagree about a link's validity.
