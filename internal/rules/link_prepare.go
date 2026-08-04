package rules

import (
	"path/filepath"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
)

// PrepareLinks resolves every link through the canonical resolver and rewrites
// resolvable targets to the root-relative form used as a graph node key.
//
// This is the bridge between the resolver, which answers in filesystem paths,
// and the graph, which is keyed by root-relative record paths. Running it means
// graph, query traversal and validate all ask the same question of the same
// resolver, which is what issue #62 was about: the two used to run disjoint
// engines and could not agree on whether a link was broken.
//
// Each link is annotated with the outcome. A target that fails to resolve keeps
// its verbatim text so the reported broken link matches what the author wrote.
// A target that resolves but names a file outside the governed record set is
// still marked resolved — the schema declares what is governed, not what
// exists, so such a link is an edge to a non-node, not a broken link.
//
// It replaces ResolveMarkdownTargets, which handled only markdown targets.
func PrepareLinks(records []*extract.Record, root string) {
	index := basenameIndex(records)
	for _, rec := range records {
		dir := filepath.Dir(filepath.Join(root, rec.Path))
		fallback := false
		if eff, err := ResolveForRecord(dir, rec.Path); err == nil && eff != nil {
			fallback = eff.Links.BasenameFallback
		}
		for i, link := range rec.Links {
			res := ResolveLinkTarget(ResolveRequest{
				BaseDir: dir,
				Root:    root,
				Target:  link.Target,
				Style:   linkStyle(link),
			})
			// Resolving is only half the job: without a root-relative key the
			// graph has nothing to match against, so a target that resolves on
			// disk but cannot be expressed relative to root stays unresolved
			// rather than being marked resolved with an unusable target.
			if rel, err := filepath.Rel(root, res.Path); res.OK && err == nil {
				rec.Links[i].Resolution = extract.LinkResolved
				rec.Links[i].Target = rel
				continue
			}
			if match, ok := index.lookup(link.Target); ok && fallback {
				rec.Links[i].Resolution = extract.LinkResolved
				rec.Links[i].Target = match
				continue
			}
			rec.Links[i].Resolution = extract.LinkUnresolved
		}
	}
}

// basenames maps a bare name to the record paths carrying it.
type basenames map[string][]string

// basenameIndex indexes every record by its basename, with and without the
// ".md" extension, so a target naming no path can be matched against the
// record set.
func basenameIndex(records []*extract.Record) basenames {
	idx := make(basenames, len(records))
	for _, rec := range records {
		base := filepath.Base(rec.Path)
		idx[base] = append(idx[base], rec.Path)
		if noExt := strings.TrimSuffix(base, ".md"); noExt != base {
			idx[noExt] = append(idx[noExt], rec.Path)
		}
	}
	return idx
}

// lookup resolves a bare target to a record path when exactly one record
// carries that basename. An ambiguous name resolves to nothing: guessing
// between two equally-plausible records would be worse than reporting the
// link unresolved.
func (b basenames) lookup(target string) (string, bool) {
	matches, ok := b[target]
	if !ok || len(matches) != 1 {
		return "", false
	}
	return matches[0], true
}
