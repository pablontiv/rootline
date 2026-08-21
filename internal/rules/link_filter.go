package rules

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/pablontiv/rootline/internal/extract"
)

// FilterLinksByStyles rewrites each record's Links to only those whose style
// is declared in the record's effective links.styles (resolved per record).
// Records that resolve no .stem keep the wikilink default, so markdown links
// never leak into style-unaware consumers like the graph.
func FilterLinksByStyles(records []*extract.Record, root string) error {
	for _, rec := range records {
		if len(rec.Links) == 0 {
			continue
		}
		styles := map[string]bool{extract.StyleWikilink: true}
		dir := filepath.Dir(filepath.Join(root, rec.Path))
		effective, err := ResolveForRecord(dir, rec.Path)
		if err != nil && !errors.Is(err, ErrNoSchemaFound) {
			return fmt.Errorf("resolve schema for %s: %w", rec.Path, err)
		}
		if effective != nil {
			styles = make(map[string]bool)
			for _, s := range effective.Links.EffectiveStyles() {
				styles[s] = true
			}
		}
		filtered := rec.Links[:0]
		for _, l := range rec.Links {
			if styles[linkStyle(l)] {
				filtered = append(filtered, l)
			}
		}
		rec.Links = filtered
	}
	return nil
}
