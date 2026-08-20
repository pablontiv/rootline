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

// ProspectiveLinkTarget overlays one exact file path during pre-write link
// validation. It is intentionally single-file: unrelated targets retain the
// ordinary filesystem semantics used by validate, graph, and query.
type ProspectiveLinkTarget struct {
	AbsPath string
	Content []byte
}

type prospectiveLinkOverlay struct {
	disk         diskLinkTargetProvider
	physicalPath string
	physicalDir  string
	baseName     string
	content      []byte
	targetInfo   os.FileInfo
}

func newProspectiveLinkOverlay(target ProspectiveLinkTarget, root string) *prospectiveLinkOverlay {
	if target.AbsPath == "" {
		return nil
	}
	if ok, err := PhysicalPathWithin(root, target.AbsPath); err != nil || !ok {
		return nil
	}
	absPath, err := filepath.Abs(target.AbsPath)
	if err != nil {
		return nil
	}
	physicalPath, err := canonicalPhysicalPath(absPath)
	if err != nil {
		return nil
	}
	physicalDir, err := canonicalPhysicalPath(filepath.Dir(absPath))
	if err != nil {
		return nil
	}
	var targetInfo os.FileInfo
	if info, err := os.Stat(absPath); err == nil {
		targetInfo = info
	} else if !os.IsNotExist(err) {
		return nil
	}
	return &prospectiveLinkOverlay{
		physicalPath: physicalPath,
		physicalDir:  physicalDir,
		baseName:     filepath.Base(absPath),
		content:      target.Content,
		targetInfo:   targetInfo,
	}
}

func (o *prospectiveLinkOverlay) readDir(dir string) ([]string, error) {
	names, err := o.disk.readDir(dir)
	if !o.dirMatches(dir) {
		return names, err
	}
	if err != nil {
		names = nil
	}
	caseFoldEquivalent := false
	for _, name := range names {
		if name == o.baseName {
			return names, nil
		}
		if strings.EqualFold(name, o.baseName) {
			caseFoldEquivalent = true
		}
	}
	if caseFoldEquivalent {
		if _, err := o.disk.stat(filepath.Join(dir, o.baseName)); err == nil {
			return names, nil
		}
	}
	return append(names, o.baseName), nil
}

func (o *prospectiveLinkOverlay) stat(path string) (linkTargetInfo, error) {
	if o.physicalPathMatches(path) {
		return linkTargetInfo{}, nil
	}
	return o.disk.stat(path)
}

func (o *prospectiveLinkOverlay) dirMatches(dir string) bool {
	if o == nil {
		return false
	}
	physicalDir, err := canonicalPhysicalPath(dir)
	return err == nil && physicalDir == o.physicalDir
}

func (o *prospectiveLinkOverlay) physicalPathMatches(path string) bool {
	if o == nil {
		return false
	}
	if o.targetInfo != nil {
		info, err := os.Stat(path)
		return err == nil && os.SameFile(o.targetInfo, info)
	}
	physicalPath, err := canonicalPhysicalPath(path)
	return err == nil && physicalPath == o.physicalPath
}

func (o *prospectiveLinkOverlay) contentFor(path string) ([]byte, bool) {
	if !o.physicalPathMatches(path) {
		return nil, false
	}
	return o.content, true
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
	return checkLinks(links, schema, sourceAbsPath, root, cache, nil)
}

// CheckLinksWithProspectiveTarget is the pre-write counterpart to CheckLinks.
// It overlays exactly one prospective file before disk existence and anchor
// reads, while preserving normal filesystem resolution for every other target.
func CheckLinksWithProspectiveTarget(links []extract.Link, schema LinkSchema, sourceAbsPath, root string, cache *HeadingCache, target ProspectiveLinkTarget) []ValidationError {
	return checkLinks(links, schema, sourceAbsPath, root, cache, newProspectiveLinkOverlay(target, root))
}

func checkLinks(links []extract.Link, schema LinkSchema, sourceAbsPath, root string, cache *HeadingCache, overlay *prospectiveLinkOverlay) []ValidationError {
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
			res := resolveLinkTargetWithProspectiveTarget(ResolveRequest{
				BaseDir: filepath.Dir(sourceAbsPath),
				Root:    root,
				Target:  link.Target,
				Style:   linkStyle(link),
			}, overlay)
			if !res.OK {
				// Basename fallback matches a target against every record in
				// the tree, and CheckLinks sees one record at a time. Calling
				// the link broken would be wrong — graph may well resolve it —
				// and staying silent would put the two commands back into
				// disagreement, so say plainly that this one is undecidable
				// here and name the command that can decide it.
				if schema.BasenameFallback && resolve {
					errs = append(errs, ValidationError{
						Rule:    "link_unverifiable",
						Field:   "links",
						Message: fmt.Sprintf("link target %q cannot be verified: links.basename_fallback matches against every record, which this command does not scan (check it with 'rootline graph --check')", link.Target),
						Source:  "links.checks",
						// "warn" is the severity NewValidationResult routes to
						// warnings; anything else counts as an error and would
						// fail the run for a link that may well be fine.
						Severity: "warn",
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
				errs = append(errs, checkAnchor(link, res.Path, cache, overlay)...)
			}
		}
	}
	return errs
}

// checkAnchor verifies the link's anchor matches a heading slug in the
// resolved target file.
func checkAnchor(link extract.Link, resolvedPath string, cache *HeadingCache, overlay *prospectiveLinkOverlay) []ValidationError {
	var slugs []string
	var err error
	if content, ok := overlay.contentFor(resolvedPath); ok {
		slugs, err = headingSlugsFromContent(resolvedPath, content)
	} else {
		if cache == nil {
			cache = NewHeadingCache()
		}
		slugs, err = cache.headingSlugs(resolvedPath)
	}
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
	slugs, err := headingSlugsFromContent(absPath, content)
	if err != nil {
		return nil, err
	}
	c.slugs[absPath] = slugs
	return slugs, nil
}

func headingSlugsFromContent(absPath string, content []byte) ([]string, error) {
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
