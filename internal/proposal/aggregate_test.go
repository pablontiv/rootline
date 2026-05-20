package proposal

import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestDetectMissingAggregates_NilEffective(t *testing.T) {
	if got := DetectMissingAggregates(t.TempDir(), nil, nil); got != nil {
		t.Errorf("expected nil for nil effective, got %v", got)
	}
}

func TestDetectMissingAggregates_NoHierarchy(t *testing.T) {
	// Empty records → hierarchy not detected → returns nil.
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Done"}},
		},
	}
	got := DetectMissingAggregates(t.TempDir(), []*extract.Record{}, effective)
	if got != nil {
		t.Errorf("expected nil when no hierarchy detected, got %v", got)
	}
}
