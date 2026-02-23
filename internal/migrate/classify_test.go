package migrate

import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestClassifier_FieldRemoved_ListsFilesWithField(t *testing.T) {
	before := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "string"},
			"tipo":   {Type: "string"},
		},
	}
	after := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "string"},
		},
	}

	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"estado": "Done", "tipo": "task"}},
		{Path: "b.md", Frontmatter: map[string]any{"estado": "Pending", "tipo": "bug"}},
		{Path: "c.md", Frontmatter: map[string]any{"estado": "Done"}},
	}

	result := Diff(".stem", before, after)
	Classify(result, records)

	if len(result.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(result.Changes))
	}
	c := result.Changes[0]
	if c.Kind != FieldRemoved {
		t.Errorf("expected field_removed, got %s", c.Kind)
	}
	if !c.Breaking {
		t.Errorf("field_removed should be breaking")
	}
	if c.AffectedFiles != 2 {
		t.Errorf("expected 2 affected files (a.md, b.md have tipo), got %d", c.AffectedFiles)
	}
}

func TestClassifier_EnumValueRemoved_ListsFilesUsingValue(t *testing.T) {
	before := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Done", "Cancelled"}},
		},
	}
	after := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Done"}},
		},
	}

	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"estado": "Cancelled"}},
		{Path: "b.md", Frontmatter: map[string]any{"estado": "Pending"}},
		{Path: "c.md", Frontmatter: map[string]any{"estado": "Cancelled"}},
		{Path: "d.md", Frontmatter: map[string]any{"estado": "Done"}},
	}

	result := Diff(".stem", before, after)
	Classify(result, records)

	if len(result.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(result.Changes))
	}
	c := result.Changes[0]
	if c.Kind != EnumValueRemoved {
		t.Errorf("expected enum_value_removed, got %s", c.Kind)
	}
	if c.AffectedFiles != 2 {
		t.Errorf("expected 2 affected files (a.md, c.md use Cancelled), got %d", c.AffectedFiles)
	}
}

func TestClassifier_TypeChanged_Breaking(t *testing.T) {
	before := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"tipo": {Type: "string"},
		},
	}
	after := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"tipo": {Type: "enum", Values: []string{"A", "B"}},
		},
	}

	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"tipo": "A"}},
		{Path: "b.md", Frontmatter: map[string]any{"tipo": "C"}},
		{Path: "c.md", Frontmatter: map[string]any{}},
	}

	result := Diff(".stem", before, after)
	Classify(result, records)

	// Find the TypeChanged change.
	var found *Change
	for i := range result.Changes {
		if result.Changes[i].Kind == TypeChanged {
			found = &result.Changes[i]
			break
		}
	}
	if found == nil {
		t.Fatal("expected type_changed change")
	}
	if !found.Breaking {
		t.Errorf("type_changed should be breaking")
	}
	if found.AffectedFiles != 2 {
		t.Errorf("expected 2 affected files (a.md, b.md have tipo), got %d", found.AffectedFiles)
	}
}

func TestClassifier_RequiredFalseToTrue_ListsFilesWithoutField(t *testing.T) {
	before := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"tipo": {Type: "string", Required: false},
		},
	}
	after := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"tipo": {Type: "string", Required: true},
		},
	}

	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"tipo": "task"}},
		{Path: "b.md", Frontmatter: map[string]any{}},
		{Path: "c.md", Frontmatter: map[string]any{"estado": "Done"}},
	}

	result := Diff(".stem", before, after)
	Classify(result, records)

	if len(result.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(result.Changes))
	}
	c := result.Changes[0]
	if c.Kind != RequiredAdded {
		t.Errorf("expected required_added, got %s", c.Kind)
	}
	if !c.Breaking {
		t.Errorf("required false->true should be breaking")
	}
	if c.AffectedFiles != 2 {
		t.Errorf("expected 2 affected files (b.md, c.md lack tipo), got %d", c.AffectedFiles)
	}
}

func TestClassifier_FieldAdded_NonBreaking(t *testing.T) {
	before := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "string"},
		},
	}
	after := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "string"},
			"tipo":   {Type: "string"},
		},
	}

	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"estado": "Done"}},
		{Path: "b.md", Frontmatter: map[string]any{"estado": "Pending"}},
	}

	result := Diff(".stem", before, after)
	Classify(result, records)

	if len(result.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(result.Changes))
	}
	c := result.Changes[0]
	if c.Kind != FieldAdded {
		t.Errorf("expected field_added, got %s", c.Kind)
	}
	if c.Breaking {
		t.Errorf("field_added should be non-breaking")
	}
	if c.AffectedFiles != 0 {
		t.Errorf("expected 0 affected files for non-breaking change, got %d", c.AffectedFiles)
	}
}

func TestClassifier_EnumValueAdded_NonBreaking(t *testing.T) {
	before := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Done"}},
		},
	}
	after := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Done", "InProgress"}},
		},
	}

	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"estado": "Pending"}},
	}

	result := Diff(".stem", before, after)
	Classify(result, records)

	if len(result.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(result.Changes))
	}
	c := result.Changes[0]
	if c.Breaking {
		t.Errorf("enum_value_added should be non-breaking")
	}
	if c.AffectedFiles != 0 {
		t.Errorf("expected 0 affected files for non-breaking change, got %d", c.AffectedFiles)
	}
}

func TestClassifier_RequiredTrueToFalse_NonBreaking(t *testing.T) {
	before := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"tipo": {Type: "string", Required: true},
		},
	}
	after := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"tipo": {Type: "string", Required: false},
		},
	}

	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"tipo": "task"}},
	}

	result := Diff(".stem", before, after)
	Classify(result, records)

	if len(result.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(result.Changes))
	}
	c := result.Changes[0]
	if c.Breaking {
		t.Errorf("required true->false should be non-breaking")
	}
	if c.AffectedFiles != 0 {
		t.Errorf("expected 0 affected files for non-breaking change, got %d", c.AffectedFiles)
	}
}

func TestClassifier_BreakingChangesFirst(t *testing.T) {
	before := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado":    {Type: "enum", Values: []string{"Pending", "Done", "Cancelled"}},
			"old_field": {Type: "string"},
		},
	}
	after := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado":    {Type: "enum", Values: []string{"Pending", "Done"}},
			"new_field": {Type: "string"},
		},
	}

	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"estado": "Cancelled", "old_field": "x"}},
		{Path: "b.md", Frontmatter: map[string]any{"estado": "Pending"}},
	}

	result := Diff(".stem", before, after)
	Classify(result, records)

	// Should have: enum_value_removed (breaking), field_removed (breaking), field_added (non-breaking)
	if result.TotalCount != 3 {
		t.Fatalf("expected 3 changes, got %d", result.TotalCount)
	}

	// Verify breaking changes come first.
	seenNonBreaking := false
	for _, c := range result.Changes {
		if !c.Breaking {
			seenNonBreaking = true
		}
		if c.Breaking && seenNonBreaking {
			t.Errorf("breaking change %s/%s appeared after non-breaking change", c.Kind, c.Field)
		}
	}
}

func TestClassifier_EmptyRecords(t *testing.T) {
	before := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"tipo": {Type: "string"},
		},
	}
	after := &rules.StemFile{
		Schema: map[string]rules.SchemaField{},
	}

	result := Diff(".stem", before, after)
	Classify(result, nil)

	if len(result.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(result.Changes))
	}
	if result.Changes[0].AffectedFiles != 0 {
		t.Errorf("expected 0 affected files with nil records, got %d", result.Changes[0].AffectedFiles)
	}
}

func TestClassifier_MultipleBreakingWithCounts(t *testing.T) {
	before := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado":        {Type: "enum", Values: []string{"Pending", "Done", "Cancelled"}, Required: true},
			"old_field":     {Type: "string"},
			"ejecutable_en": {Type: "string", Required: false},
		},
	}
	after := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado":        {Type: "enum", Values: []string{"Pending", "Done", "InProgress"}, Required: true},
			"new_field":     {Type: "string"},
			"ejecutable_en": {Type: "string", Required: true},
		},
	}

	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"estado": "Cancelled", "old_field": "x", "ejecutable_en": "1 session"}},
		{Path: "b.md", Frontmatter: map[string]any{"estado": "Pending", "old_field": "y"}},
		{Path: "c.md", Frontmatter: map[string]any{"estado": "Done"}},
		{Path: "d.md", Frontmatter: map[string]any{"estado": "Cancelled"}},
	}

	result := Diff(".stem", before, after)
	Classify(result, records)

	// Expected changes:
	// Breaking: enum_value_removed Cancelled (2 files: a,d), field_removed old_field (2 files: a,b),
	//           required_added ejecutable_en (3 files: b,c,d)
	// Non-breaking: enum_value_added InProgress (0), field_added new_field (0)
	if result.TotalCount != 5 {
		t.Errorf("expected 5 changes, got %d", result.TotalCount)
		for _, c := range result.Changes {
			t.Logf("  %s %s breaking=%v affected=%d", c.Kind, c.Field, c.Breaking, c.AffectedFiles)
		}
	}
	if result.BreakingCount != 3 {
		t.Errorf("expected 3 breaking, got %d", result.BreakingCount)
	}

	// Verify specific affected file counts.
	for _, c := range result.Changes {
		switch {
		case c.Kind == EnumValueRemoved && c.Before == "Cancelled":
			if c.AffectedFiles != 2 {
				t.Errorf("enum_value_removed Cancelled: expected 2 affected, got %d", c.AffectedFiles)
			}
		case c.Kind == FieldRemoved && c.Field == "old_field":
			if c.AffectedFiles != 2 {
				t.Errorf("field_removed old_field: expected 2 affected, got %d", c.AffectedFiles)
			}
		case c.Kind == RequiredAdded && c.Field == "ejecutable_en":
			if c.AffectedFiles != 3 {
				t.Errorf("required_added ejecutable_en: expected 3 affected, got %d", c.AffectedFiles)
			}
		case c.Kind == EnumValueAdded:
			if c.AffectedFiles != 0 {
				t.Errorf("enum_value_added: expected 0 affected, got %d", c.AffectedFiles)
			}
		case c.Kind == FieldAdded:
			if c.AffectedFiles != 0 {
				t.Errorf("field_added: expected 0 affected, got %d", c.AffectedFiles)
			}
		}
	}

	// Verify breaking-first ordering.
	seenNonBreaking := false
	for _, c := range result.Changes {
		if !c.Breaking {
			seenNonBreaking = true
		}
		if c.Breaking && seenNonBreaking {
			t.Errorf("breaking change %s/%s appeared after non-breaking", c.Kind, c.Field)
		}
	}
}
