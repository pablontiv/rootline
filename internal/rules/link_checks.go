package rules

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/pablontiv/picokit/fuzzy"
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
// file; relative targets resolve against its directory. Links whose style is
// not in the schema's effective styles are skipped, as are absolute targets
// (root-relative ADO form is out of scope).
func CheckLinks(links []extract.Link, schema LinkSchema, sourceAbsPath string, cache *HeadingCache) []ValidationError {
	if schema.Checks == nil {
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

		if schema.Checks.Encoding && strings.Contains(link.Target, " ") {
			errs = append(errs, ValidationError{
				Rule:     "link_encoding",
				Field:    "links",
				Message:  fmt.Sprintf("link target %q contains unencoded spaces (use %%20)", link.Target),
				Source:   "links.checks",
				Severity: "error",
			})
		}

		if strings.HasPrefix(link.Target, "/") {
			continue
		}

		if schema.Checks.Resolve || schema.Checks.Anchors {
			resolved, suggestion, ok := resolveCaseSensitive(filepath.Dir(sourceAbsPath), link.Target)
			if !ok {
				if schema.Checks.Resolve {
					msg := fmt.Sprintf("link target %q does not resolve to an existing file (case-sensitive)", link.Target)
					if suggestion != "" {
						msg += fmt.Sprintf(" (did you mean %q?)", suggestion)
					}
					errs = append(errs, ValidationError{
						Rule:       "link_resolve",
						Field:      "links",
						Message:    msg,
						Source:     "links.checks",
						Severity:   "error",
						Suggestion: suggestion,
					})
				}
				continue
			}
			if schema.Checks.Anchors && link.Anchor != "" {
				errs = append(errs, checkAnchor(link, resolved, cache)...)
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

// resolveCaseSensitive resolves a relative link target against baseDir,
// requiring exact-case matches for every target path component (APFS is
// case-insensitive; ADO and git are not). Directory targets resolve to their
// README.md. Returns the resolved absolute path, a fuzzy suggestion for the
// first unmatched component (may be empty), and whether resolution succeeded.
func resolveCaseSensitive(baseDir, target string) (string, string, bool) {
	decoded, err := url.PathUnescape(target)
	if err != nil {
		decoded = target
	}

	cur := baseDir
	for _, comp := range strings.Split(filepath.ToSlash(filepath.Clean(decoded)), "/") {
		switch comp {
		case "", ".":
			continue
		case "..":
			cur = filepath.Dir(cur)
			continue
		}
		entry, suggestion, ok := findEntry(cur, comp)
		if !ok {
			return "", suggestion, false
		}
		cur = filepath.Join(cur, entry)
	}

	info, err := os.Stat(cur)
	if err != nil {
		return "", "", false
	}
	if info.IsDir() {
		entry, suggestion, ok := findEntry(cur, "README.md")
		if !ok {
			return "", suggestion, false
		}
		cur = filepath.Join(cur, entry)
	}
	return cur, "", true
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
	suggestion := fuzzy.Match(name, names)
	return "", suggestion, false
}
