package infer

import (
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestGroupByScope_BucketsByClosestStem(t *testing.T) {
	conceptsStem := &rules.StemFile{Schema: map[string]rules.SchemaField{"kind": {Type: "enum"}}}
	sourcesStem := &rules.StemFile{Schema: map[string]rules.SchemaField{"tipo": {Type: "enum"}}}
	resolve := func(dir string) (*rules.StemFile, string) {
		if filepath.Base(dir) == "concepts" {
			return conceptsStem, "concepts/.stem"
		}
		return sourcesStem, "sources/.stem"
	}
	records := []*extract.Record{
		{Path: "concepts/a.md"}, {Path: "concepts/b.md"}, {Path: "sources/p.md"},
	}
	groups := GroupByScope(records, ".", resolve)
	if len(groups) != 2 {
		t.Fatalf("expected 2 scope groups, got %d", len(groups))
	}
	byKey := map[string]ScopeGroup{}
	for _, g := range groups {
		byKey[g.Key] = g
	}
	if len(byKey["concepts/.stem"].Records) != 2 || len(byKey["sources/.stem"].Records) != 1 {
		t.Errorf("wrong bucketing: %+v", byKey)
	}
}

func TestGroupByScope_NoStem(t *testing.T) {
	resolve := func(dir string) (*rules.StemFile, string) { return nil, "" }
	groups := GroupByScope([]*extract.Record{{Path: "x.md"}}, ".", resolve)
	if len(groups) != 1 || groups[0].Stem != nil || groups[0].Key != "" {
		t.Errorf("expected one nil-stem group, got %+v", groups)
	}
}
