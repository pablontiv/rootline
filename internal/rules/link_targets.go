package rules

import (
	"path/filepath"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
)

// ResolveMarkdownTargets rewrites markdown-style link targets to the
// root-relative paths used as graph node keys, applying the same resolution
// as validate's link checks: %20 decoding, case-sensitive component walk,
// and directory targets resolving to their README.md. Root-anchored targets
// ("/x.md") resolve against root. Targets that fail to resolve are left
// verbatim so the graph reports them as broken links. Wikilinks are never
// touched.
func ResolveMarkdownTargets(records []*extract.Record, root string) {
	for _, rec := range records {
		dir := filepath.Dir(filepath.Join(root, rec.Path))
		for i, link := range rec.Links {
			if link.Style != extract.StyleMarkdown {
				continue
			}
			base, target := dir, link.Target
			if strings.HasPrefix(target, "/") {
				base, target = root, strings.TrimPrefix(target, "/")
			}
			resolved, _, ok := resolveCaseSensitive(base, target)
			if !ok {
				continue
			}
			if rel, err := filepath.Rel(root, resolved); err == nil {
				rec.Links[i].Target = rel
			}
		}
	}
}
