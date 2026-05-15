package rules

import (
	"context"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func makeRecord(frontmatter map[string]any) *extract.Record {
	return &extract.Record{
		Path:        "test.md",
		Type:        "markdown",
		Frontmatter: frontmatter,
	}
}

func TestValidate_RequiredFieldMissing(t *testing.T) {
	stem := &StemFile{
		Schema: map[string]SchemaField{
			"title": {Type: "string", Required: true, Source: "root/.stem"},
		},
	}
	record := makeRecord(map[string]any{})

	errs := Validate(context.Background(), record, stem)
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1", len(errs))
	}
	if errs[0].Rule != "required" {
		t.Errorf("rule = %q, want required", errs[0].Rule)
	}
	if errs[0].Field != "title" {
		t.Errorf("field = %q, want title", errs[0].Field)
	}
	if errs[0].Source != "root/.stem" {
		t.Errorf("source = %q, want root/.stem", errs[0].Source)
	}
}

func TestValidate_RequiredFieldPresent(t *testing.T) {
	stem := &StemFile{
		Schema: map[string]SchemaField{
			"title": {Type: "string", Required: true, Source: "root/.stem"},
		},
	}
	record := makeRecord(map[string]any{"title": "Hello"})

	errs := Validate(context.Background(), record, stem)
	if len(errs) != 0 {
		t.Errorf("got %d errors, want 0", len(errs))
	}
}

func TestValidate_EnumValidValue(t *testing.T) {
	stem := &StemFile{
		Schema: map[string]SchemaField{
			"estado": {
				Type:   "enum",
				Values: []string{"Pending", "Completed"},
				Source: "tasks/.stem",
			},
		},
	}
	record := makeRecord(map[string]any{"estado": "Pending"})

	errs := Validate(context.Background(), record, stem)
	if len(errs) != 0 {
		t.Errorf("got %d errors, want 0", len(errs))
	}
}

func TestValidate_EnumInvalidValue(t *testing.T) {
	stem := &StemFile{
		Schema: map[string]SchemaField{
			"estado": {
				Type:     "enum",
				Values:   []string{"Pending", "Completed"},
				Required: true,
				Source:   "tasks/.stem",
			},
		},
	}
	record := makeRecord(map[string]any{"estado": "invalido"})

	errs := Validate(context.Background(), record, stem)
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1", len(errs))
	}
	if errs[0].Rule != "enum" {
		t.Errorf("rule = %q, want enum", errs[0].Rule)
	}
	if errs[0].Source != "tasks/.stem" {
		t.Errorf("source = %q, want tasks/.stem", errs[0].Source)
	}
}

func TestValidate_NonEmpty_EmptyString(t *testing.T) {
	stem := &StemFile{
		Validate: []ValidationRule{
			{Field: "title", Rule: "non_empty", Source: "root/.stem"},
		},
	}
	record := makeRecord(map[string]any{"title": ""})

	errs := Validate(context.Background(), record, stem)
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1", len(errs))
	}
	if errs[0].Rule != "non_empty" {
		t.Errorf("rule = %q, want non_empty", errs[0].Rule)
	}
}

func TestValidate_NonEmpty_MissingField(t *testing.T) {
	stem := &StemFile{
		Validate: []ValidationRule{
			{Field: "title", Rule: "non_empty", Source: "root/.stem"},
		},
	}
	record := makeRecord(map[string]any{})

	errs := Validate(context.Background(), record, stem)
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1", len(errs))
	}
	if errs[0].Rule != "non_empty" {
		t.Errorf("rule = %q, want non_empty", errs[0].Rule)
	}
}

func TestValidate_NonEmpty_Present(t *testing.T) {
	stem := &StemFile{
		Validate: []ValidationRule{
			{Field: "title", Rule: "non_empty", Source: "root/.stem"},
		},
	}
	record := makeRecord(map[string]any{"title": "Hello"})

	errs := Validate(context.Background(), record, stem)
	if len(errs) != 0 {
		t.Errorf("got %d errors, want 0", len(errs))
	}
}

func TestValidate_Exists_Present(t *testing.T) {
	stem := &StemFile{
		Validate: []ValidationRule{
			{Field: "status", Rule: "exists", Source: "root/.stem"},
		},
	}
	record := makeRecord(map[string]any{"status": ""})

	errs := Validate(context.Background(), record, stem)
	if len(errs) != 0 {
		t.Errorf("got %d errors, want 0 (exists allows empty)", len(errs))
	}
}

func TestValidate_Exists_Absent(t *testing.T) {
	stem := &StemFile{
		Validate: []ValidationRule{
			{Field: "status", Rule: "exists", Source: "root/.stem"},
		},
	}
	record := makeRecord(map[string]any{})

	errs := Validate(context.Background(), record, stem)
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1", len(errs))
	}
	if errs[0].Rule != "exists" {
		t.Errorf("rule = %q, want exists", errs[0].Rule)
	}
}

func TestValidate_Requires_ConditionTrue_FieldMissing(t *testing.T) {
	stem := &StemFile{
		Validate: []ValidationRule{
			{
				Rule:   "requires",
				If:     map[string]any{"Estado": "Completed"},
				Then:   map[string]any{"fields": []any{"Fecha"}},
				Source: "prd/.stem",
			},
		},
	}
	record := makeRecord(map[string]any{"Estado": "Completed"})

	errs := Validate(context.Background(), record, stem)
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1", len(errs))
	}
	if errs[0].Rule != "requires" {
		t.Errorf("rule = %q, want requires", errs[0].Rule)
	}
	if errs[0].Field != "Fecha" {
		t.Errorf("field = %q, want Fecha", errs[0].Field)
	}
}

func TestValidate_Requires_ConditionFalse_NoCheck(t *testing.T) {
	stem := &StemFile{
		Validate: []ValidationRule{
			{
				Rule:   "requires",
				If:     map[string]any{"Estado": "Completed"},
				Then:   map[string]any{"fields": []any{"Fecha"}},
				Source: "prd/.stem",
			},
		},
	}
	record := makeRecord(map[string]any{"Estado": "Pending"})

	errs := Validate(context.Background(), record, stem)
	if len(errs) != 0 {
		t.Errorf("got %d errors, want 0 (condition not met)", len(errs))
	}
}

func TestValidate_Requires_ConditionTrue_FieldPresent(t *testing.T) {
	stem := &StemFile{
		Validate: []ValidationRule{
			{
				Rule:   "requires",
				If:     map[string]any{"Estado": "Completed"},
				Then:   map[string]any{"fields": []any{"Fecha"}},
				Source: "prd/.stem",
			},
		},
	}
	record := makeRecord(map[string]any{
		"Estado": "Completed",
		"Fecha":  "2026-01-15",
	})

	errs := Validate(context.Background(), record, stem)
	if len(errs) != 0 {
		t.Errorf("got %d errors, want 0", len(errs))
	}
}

func TestValidate_ValidDocument_NoErrors(t *testing.T) {
	stem := &StemFile{
		Schema: map[string]SchemaField{
			"title":  {Type: "string", Required: true, Source: "root/.stem"},
			"estado": {Type: "enum", Values: []string{"Pending", "Completed"}, Required: true, Source: "tasks/.stem"},
		},
		Validate: []ValidationRule{
			{Field: "title", Rule: "non_empty", Source: "root/.stem"},
		},
	}
	record := makeRecord(map[string]any{
		"title":  "My Task",
		"estado": "Pending",
	})

	errs := Validate(context.Background(), record, stem)
	if len(errs) != 0 {
		t.Errorf("got %d errors, want 0 for valid doc", len(errs))
	}
}

func TestValidate_NilStem(t *testing.T) {
	record := makeRecord(map[string]any{"title": "Hello"})
	errs := Validate(context.Background(), record, nil)
	if errs != nil {
		t.Errorf("got %v, want nil for nil stem", errs)
	}
}

func TestValidate_MultipleErrors(t *testing.T) {
	stem := &StemFile{
		Schema: map[string]SchemaField{
			"title":  {Type: "string", Required: true, Source: "root/.stem"},
			"estado": {Type: "enum", Values: []string{"Pending", "Completed"}, Required: true, Source: "tasks/.stem"},
			"tipo":   {Type: "enum", Values: []string{"code", "docs"}, Source: "tasks/.stem"},
		},
	}
	// Missing title (required), bad estado (enum), bad tipo (enum)
	record := makeRecord(map[string]any{
		"estado": "Invalid",
		"tipo":   "unknown",
	})

	errs := Validate(context.Background(), record, stem)
	if len(errs) != 3 {
		t.Fatalf("got %d errors, want 3: %v", len(errs), errs)
	}
}

func TestValidate_ExplicitEnumRule(t *testing.T) {
	stem := &StemFile{
		Schema: map[string]SchemaField{
			"priority": {Type: "enum", Values: []string{"high", "low"}, Source: "root/.stem"},
		},
		Validate: []ValidationRule{
			{Field: "priority", Rule: "enum", Source: "root/.stem"},
		},
	}
	record := makeRecord(map[string]any{"priority": "medium"})

	errs := Validate(context.Background(), record, stem)
	// Both schema auto-check and explicit rule fire — but they produce the same error.
	// We expect at least 1 enum error.
	hasEnum := false
	for _, e := range errs {
		if e.Rule == "enum" && e.Field == "priority" {
			hasEnum = true
		}
	}
	if !hasEnum {
		t.Error("expected enum error for priority")
	}
}

func TestValidate_RequiresMultipleFields(t *testing.T) {
	stem := &StemFile{
		Validate: []ValidationRule{
			{
				Rule:   "requires",
				If:     map[string]any{"Estado": "Completed"},
				Then:   map[string]any{"fields": []any{"Fecha", "Autor"}},
				Source: "prd/.stem",
			},
		},
	}
	record := makeRecord(map[string]any{"Estado": "Completed"})

	errs := Validate(context.Background(), record, stem)
	if len(errs) != 2 {
		t.Fatalf("got %d errors, want 2 (Fecha + Autor)", len(errs))
	}
}

func TestValidate_SourceTracking(t *testing.T) {
	stem := &StemFile{
		Schema: map[string]SchemaField{
			"title":  {Type: "string", Required: true, Source: "docs/.stem"},
			"estado": {Type: "enum", Values: []string{"Pending"}, Required: true, Source: "tasks/.stem"},
		},
	}
	record := makeRecord(map[string]any{})

	errs := Validate(context.Background(), record, stem)
	sources := map[string]bool{}
	for _, e := range errs {
		sources[e.Source] = true
	}
	if !sources["docs/.stem"] {
		t.Error("expected error from docs/.stem")
	}
	if !sources["tasks/.stem"] {
		t.Error("expected error from tasks/.stem")
	}
}

func TestValidate_RequiredAggregateOnIndexFile_Skipped(t *testing.T) {
	stem := &StemFile{
		Schema: map[string]SchemaField{
			"estado": {Type: "enum", Required: true, Source: "root/.stem", Severity: "error"},
		},
		Aggregate: map[string]any{
			"estado": "len(filter(descendants, {.estado == 'Completed'})) == len(descendants) ? 'Completed' : 'Pending'",
		},
	}
	record := &extract.Record{
		Path:        "docs/epics/E01/README.md",
		Type:        "markdown",
		Frontmatter: map[string]any{},
	}
	errs := Validate(context.Background(), record, stem)
	if len(errs) != 0 {
		t.Errorf("got %d errors, want 0 (aggregate field on index file)", len(errs))
	}
}

func TestValidate_RequiredAggregateOnNonIndexFile_Error(t *testing.T) {
	stem := &StemFile{
		Schema: map[string]SchemaField{
			"estado": {Type: "enum", Required: true, Source: "root/.stem", Severity: "error"},
		},
		Aggregate: map[string]any{
			"estado": "some expr",
		},
	}
	record := &extract.Record{
		Path:        "docs/epics/E01/T001-task.md",
		Type:        "markdown",
		Frontmatter: map[string]any{},
	}
	errs := Validate(context.Background(), record, stem)
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1 (non-index file)", len(errs))
	}
	if errs[0].Rule != "required" {
		t.Errorf("rule = %q, want required", errs[0].Rule)
	}
}

func TestValidate_RequiredNoAggregateOnIndexFile_Error(t *testing.T) {
	stem := &StemFile{
		Schema: map[string]SchemaField{
			"estado": {Type: "enum", Required: true, Source: "root/.stem", Severity: "error"},
		},
	}
	record := &extract.Record{
		Path:        "docs/epics/E01/README.md",
		Type:        "markdown",
		Frontmatter: map[string]any{},
	}
	errs := Validate(context.Background(), record, stem)
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1 (no aggregate)", len(errs))
	}
	if errs[0].Rule != "required" {
		t.Errorf("rule = %q, want required", errs[0].Rule)
	}
}

func TestValidate_RequiredAggregateCustomIndexName(t *testing.T) {
	stem := &StemFile{
		Schema: map[string]SchemaField{
			"estado": {Type: "enum", Required: true, Source: "root/.stem", Severity: "error"},
		},
		Aggregate: map[string]any{
			"estado": "some expr",
		},
		Structural: StructuralRules{
			Subdirs: SubdirRules{RequireIndex: "index.md"},
		},
	}
	// Custom index file — should skip
	record := &extract.Record{
		Path:        "docs/index.md",
		Type:        "markdown",
		Frontmatter: map[string]any{},
	}
	errs := Validate(context.Background(), record, stem)
	if len(errs) != 0 {
		t.Errorf("got %d errors, want 0 (custom index name match)", len(errs))
	}

	// README.md is NOT the index file when custom name is set — should error
	record2 := &extract.Record{
		Path:        "docs/README.md",
		Type:        "markdown",
		Frontmatter: map[string]any{},
	}
	errs2 := Validate(context.Background(), record2, stem)
	if len(errs2) != 1 {
		t.Errorf("got %d errors, want 1 (README.md is not the custom index)", len(errs2))
	}
}

func TestValidate_ExcludesMatchSkipsRequired(t *testing.T) {
	stem := &StemFile{
		Schema: map[string]SchemaField{
			"estado": {
				Type:     "enum",
				Required: true,
				Source:   "root/.stem",
				Severity: "error",
				Excludes: &ExcludeRule{Match: "*/README.md"},
			},
		},
	}
	record := &extract.Record{
		Path:        "docs/README.md",
		Type:        "markdown",
		Frontmatter: map[string]any{},
	}
	errs := Validate(context.Background(), record, stem)
	if len(errs) != 0 {
		t.Errorf("got %d errors, want 0 (excludes match)", len(errs))
	}
}

func TestValidate_ExcludesNoMatchKeepsRequired(t *testing.T) {
	stem := &StemFile{
		Schema: map[string]SchemaField{
			"estado": {
				Type:     "enum",
				Required: true,
				Source:   "root/.stem",
				Severity: "error",
				Excludes: &ExcludeRule{Match: "*/README.md"},
			},
		},
	}
	record := &extract.Record{
		Path:        "docs/T001-task.md",
		Type:        "markdown",
		Frontmatter: map[string]any{},
	}
	errs := Validate(context.Background(), record, stem)
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1 (excludes no match)", len(errs))
	}
	if errs[0].Rule != "required" {
		t.Errorf("rule = %q, want required", errs[0].Rule)
	}
}

func TestValidate_ExcludesNilNoEffect(t *testing.T) {
	stem := &StemFile{
		Schema: map[string]SchemaField{
			"estado": {
				Type:     "enum",
				Required: true,
				Source:   "root/.stem",
				Severity: "error",
			},
		},
	}
	record := &extract.Record{
		Path:        "docs/README.md",
		Type:        "markdown",
		Frontmatter: map[string]any{},
	}
	errs := Validate(context.Background(), record, stem)
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1 (nil excludes)", len(errs))
	}
}

func TestValidateLinks_AllowedType(t *testing.T) {
	stem := &StemFile{
		Links: LinkSchema{Allowed: []string{"blocks", "parent"}},
		Path:  "test/.stem",
	}
	record := &extract.Record{
		Path: "test.md", Type: "markdown",
		Frontmatter: map[string]any{},
		Links:       []extract.Link{{Target: "T003", Type: "blocks", Line: 1}},
	}
	errs := Validate(context.Background(), record, stem)
	if len(errs) != 0 {
		t.Errorf("got %d errors, want 0: %v", len(errs), errs)
	}
}

func TestValidateLinks_DisallowedType(t *testing.T) {
	stem := &StemFile{
		Links: LinkSchema{Allowed: []string{"blocks", "parent"}},
		Path:  "test/.stem",
	}
	record := &extract.Record{
		Path: "test.md", Type: "markdown",
		Frontmatter: map[string]any{},
		Links:       []extract.Link{{Target: "X", Type: "unknown", Line: 1}},
	}
	errs := Validate(context.Background(), record, stem)
	if len(errs) != 1 || errs[0].Rule != "link_type" {
		t.Errorf("got %v, want 1 link_type error", errs)
	}
}

func TestValidateLinks_TargetMatchesRegexp(t *testing.T) {
	stem := &StemFile{
		Links: LinkSchema{
			Allowed: []string{"blocks"},
			Rules:   map[string]LinkRule{"blocks": {Target: `^T\d{3}-`}},
		},
		Path: "test/.stem",
	}
	record := &extract.Record{
		Path: "test.md", Type: "markdown",
		Frontmatter: map[string]any{},
		Links:       []extract.Link{{Target: "T001-task-name", Type: "blocks", Line: 1}},
	}
	errs := Validate(context.Background(), record, stem)
	if len(errs) != 0 {
		t.Errorf("got %d errors, want 0: %v", len(errs), errs)
	}
}

func TestValidateLinks_TargetMismatchRegexp(t *testing.T) {
	stem := &StemFile{
		Links: LinkSchema{
			Allowed: []string{"blocks"},
			Rules:   map[string]LinkRule{"blocks": {Target: `^T\d{3}-`}},
		},
		Path: "test/.stem",
	}
	for _, target := range []string{"target", "A,B,C,A", "T99-short"} {
		record := &extract.Record{
			Path: "test.md", Type: "markdown",
			Frontmatter: map[string]any{},
			Links:       []extract.Link{{Target: target, Type: "blocks", Line: 1}},
		}
		errs := Validate(context.Background(), record, stem)
		if len(errs) != 1 || errs[0].Rule != "link_target" {
			t.Errorf("target %q: got %v, want 1 link_target error", target, errs)
		}
	}
}

func TestValidateLinks_InvalidRegexp(t *testing.T) {
	stem := &StemFile{
		Links: LinkSchema{
			Allowed: []string{"blocks"},
			Rules:   map[string]LinkRule{"blocks": {Target: "[invalid"}},
		},
		Path: "test/.stem",
	}
	record := &extract.Record{
		Path: "test.md", Type: "markdown",
		Frontmatter: map[string]any{},
		Links:       []extract.Link{{Target: "anything", Type: "blocks", Line: 1}},
	}
	errs := Validate(context.Background(), record, stem)
	if len(errs) != 1 || errs[0].Rule != "link_target" {
		t.Errorf("got %v, want 1 link_target error for invalid regexp", errs)
	}
}

func TestValidateLinks_NoSchema_Permissive(t *testing.T) {
	stem := &StemFile{Path: "test/.stem"}
	record := &extract.Record{
		Path: "test.md", Type: "markdown",
		Frontmatter: map[string]any{},
		Links:       []extract.Link{{Target: "anything", Type: "whatever", Line: 1}},
	}
	errs := Validate(context.Background(), record, stem)
	if len(errs) != 0 {
		t.Errorf("got %d errors, want 0 (no schema = permissive): %v", len(errs), errs)
	}
}

func TestValidate_AggregateConsistency_Mismatch(t *testing.T) {
	stem := &StemFile{
		Path: "test/.stem",
		Schema: map[string]SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Completed", "In Progress"}, Required: true},
		},
		Aggregate: map[string]any{
			"estado": `all(children, {.estado == "Completed"}) ? "Completed" : "In Progress"`,
		},
	}
	record := &extract.Record{
		Path:        "dir/README.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"estado": "Pending"},
		Derived:     map[string]any{"estado": "Completed"},
	}
	errs := Validate(context.Background(), record, stem)
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1: %v", len(errs), errs)
	}
	if errs[0].Rule != "aggregate" {
		t.Errorf("rule = %q, want aggregate", errs[0].Rule)
	}
	if errs[0].Field != "estado" {
		t.Errorf("field = %q, want estado", errs[0].Field)
	}
}

func TestValidate_AggregateConsistency_Match(t *testing.T) {
	stem := &StemFile{
		Path: "test/.stem",
		Schema: map[string]SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Completed"}, Required: true},
		},
		Aggregate: map[string]any{
			"estado": `"Completed"`,
		},
	}
	record := &extract.Record{
		Path:        "dir/README.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"estado": "Completed"},
		Derived:     map[string]any{"estado": "Completed"},
	}
	errs := Validate(context.Background(), record, stem)
	if len(errs) != 0 {
		t.Errorf("got %d errors, want 0 (values match): %v", len(errs), errs)
	}
}

func TestValidate_AggregateConsistency_NonIndexFile_Skipped(t *testing.T) {
	stem := &StemFile{
		Path: "test/.stem",
		Schema: map[string]SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Completed"}, Required: true},
		},
		Aggregate: map[string]any{
			"estado": `"Completed"`,
		},
	}
	record := &extract.Record{
		Path:        "dir/task.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"estado": "Pending"},
		Derived:     map[string]any{"estado": "Completed"},
	}
	errs := Validate(context.Background(), record, stem)
	if len(errs) != 0 {
		t.Errorf("got %d errors, want 0 (non-index file skipped): %v", len(errs), errs)
	}
}

func TestValidate_Requires_DerivedFieldCondition(t *testing.T) {
	// Bug fix: requires rule with condition on a derived field should fire.
	stem := &StemFile{
		Validate: []ValidationRule{
			{
				Rule:   "requires",
				If:     map[string]any{"estado": "Completed"},
				Then:   map[string]any{"fields": []any{"fecha_fin"}},
				Source: "root/.stem",
			},
		},
	}
	// estado is derived (not in frontmatter), condition should still match.
	record := &extract.Record{
		Path:        "task.md",
		Type:        "markdown",
		Frontmatter: map[string]any{},
		Derived:     map[string]any{"estado": "Completed"},
	}

	errs := Validate(context.Background(), record, stem)
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1 (derived condition match)", len(errs))
	}
	if errs[0].Rule != "requires" {
		t.Errorf("rule = %q, want requires", errs[0].Rule)
	}
	if errs[0].Field != "fecha_fin" {
		t.Errorf("field = %q, want fecha_fin", errs[0].Field)
	}
}

func TestValidate_Requires_DerivedFieldSatisfies(t *testing.T) {
	// A derived field should satisfy a "then.fields" requirement.
	stem := &StemFile{
		Validate: []ValidationRule{
			{
				Rule:   "requires",
				If:     map[string]any{"tipo": "software-module"},
				Then:   map[string]any{"fields": []any{"ejecutable_en"}},
				Source: "root/.stem",
			},
		},
	}
	record := &extract.Record{
		Path:        "task.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"tipo": "software-module"},
		Derived:     map[string]any{"ejecutable_en": "docker"},
	}

	errs := Validate(context.Background(), record, stem)
	if len(errs) != 0 {
		t.Errorf("got %d errors, want 0 (derived field satisfies requires): %v", len(errs), errs)
	}
}

func TestValidate_Exists_DerivedField(t *testing.T) {
	// A derived field should satisfy an "exists" rule.
	stem := &StemFile{
		Validate: []ValidationRule{
			{Field: "estado", Rule: "exists", Source: "root/.stem"},
		},
	}
	record := &extract.Record{
		Path:        "task.md",
		Type:        "markdown",
		Frontmatter: map[string]any{},
		Derived:     map[string]any{"estado": "Pending"},
	}

	errs := Validate(context.Background(), record, stem)
	if len(errs) != 0 {
		t.Errorf("got %d errors, want 0 (derived field satisfies exists): %v", len(errs), errs)
	}
}

func TestValidate_NonEmpty_DerivedField(t *testing.T) {
	// A derived field should satisfy a "non_empty" rule.
	stem := &StemFile{
		Validate: []ValidationRule{
			{Field: "estado", Rule: "non_empty", Source: "root/.stem"},
		},
	}
	record := &extract.Record{
		Path:        "task.md",
		Type:        "markdown",
		Frontmatter: map[string]any{},
		Derived:     map[string]any{"estado": "Pending"},
	}

	errs := Validate(context.Background(), record, stem)
	if len(errs) != 0 {
		t.Errorf("got %d errors, want 0 (derived field satisfies non_empty): %v", len(errs), errs)
	}
}

func TestValidate_Requires_MatchCondition_Applies(t *testing.T) {
	// requires rule with match: "T*" should fire for T-level records
	stem := &StemFile{
		Validate: []ValidationRule{
			{
				Rule:   "requires",
				If:     map[string]any{"match": "T*", "tipo": "modulo-sistema"},
				Then:   map[string]any{"fields": []any{"ejecutable_en"}},
				Source: "test/.stem",
			},
		},
	}
	record := &extract.Record{
		Path:        "docs/epics/E01/F01/S001/T001-task.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"tipo": "modulo-sistema"},
	}

	errs := Validate(context.Background(), record, stem)
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1", len(errs))
	}
	if errs[0].Field != "ejecutable_en" {
		t.Errorf("field = %q, want ejecutable_en", errs[0].Field)
	}
}

func TestValidate_Requires_MatchCondition_NonMatchingDir(t *testing.T) {
	// Same rule should NOT fire for F-level records
	stem := &StemFile{
		Validate: []ValidationRule{
			{
				Rule:   "requires",
				If:     map[string]any{"match": "T*", "tipo": "modulo-sistema"},
				Then:   map[string]any{"fields": []any{"ejecutable_en"}},
				Source: "test/.stem",
			},
		},
	}
	record := &extract.Record{
		Path:        "docs/epics/E01/F01-feature/README.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"tipo": "modulo-sistema"},
	}

	errs := Validate(context.Background(), record, stem)
	if len(errs) != 0 {
		t.Errorf("got %d errors, want 0 (match T* should not apply to F-level): %v", len(errs), errs)
	}
}

func TestValidate_Requires_MatchOnly(t *testing.T) {
	// match condition without other field conditions
	stem := &StemFile{
		Validate: []ValidationRule{
			{
				Rule:   "requires",
				If:     map[string]any{"match": "T*"},
				Then:   map[string]any{"fields": []any{"ejecutable_en"}},
				Source: "test/.stem",
			},
		},
	}

	t.Run("T-level fires", func(t *testing.T) {
		record := &extract.Record{
			Path:        "docs/epics/E01/F01/S001/T001-task.md",
			Type:        "markdown",
			Frontmatter: map[string]any{},
		}
		errs := Validate(context.Background(), record, stem)
		if len(errs) != 1 {
			t.Errorf("got %d errors, want 1", len(errs))
		}
	})

	t.Run("F-level skips", func(t *testing.T) {
		record := &extract.Record{
			Path:        "docs/epics/E01/F01-feature/README.md",
			Type:        "markdown",
			Frontmatter: map[string]any{},
		}
		errs := Validate(context.Background(), record, stem)
		if len(errs) != 0 {
			t.Errorf("got %d errors, want 0", len(errs))
		}
	})
}

func TestValidate_Requires_NoMatchBackwardCompat(t *testing.T) {
	// Rules without match should work exactly as before
	stem := &StemFile{
		Validate: []ValidationRule{
			{
				Rule:   "requires",
				If:     map[string]any{"tipo": "modulo-sistema"},
				Then:   map[string]any{"fields": []any{"ejecutable_en"}},
				Source: "test/.stem",
			},
		},
	}
	record := &extract.Record{
		Path:        "docs/epics/E01/F01-feature/README.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"tipo": "modulo-sistema"},
	}

	errs := Validate(context.Background(), record, stem)
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1 (no match key = applies everywhere)", len(errs))
	}
	if errs[0].Field != "ejecutable_en" {
		t.Errorf("field = %q, want ejecutable_en", errs[0].Field)
	}
}

func TestValidate_LinkFieldWithoutWikilink(t *testing.T) {
	stem := &StemFile{
		Schema: map[string]SchemaField{
			"spec": {Type: "link", Required: false, Severity: "warn"},
		},
	}
	rec := &extract.Record{
		Path:        "test.md",
		Frontmatter: map[string]any{"spec": "plain-text-no-brackets"},
	}
	errs := Validate(context.Background(), rec, stem)
	found := false
	for _, e := range errs {
		if e.Rule == "link-format" && e.Field == "spec" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected link-format warning for plain text in link field")
	}
}

func TestValidate_LinkFieldWithWikilink(t *testing.T) {
	stem := &StemFile{
		Schema: map[string]SchemaField{
			"spec": {Type: "link", Required: false, Severity: "warn"},
		},
	}
	rec := &extract.Record{
		Path:        "test.md",
		Frontmatter: map[string]any{"spec": "[[specs/my-design]]"},
	}
	errs := Validate(context.Background(), rec, stem)
	for _, e := range errs {
		if e.Rule == "link-format" {
			t.Errorf("unexpected link-format error: %s", e.Message)
		}
	}
}

func TestValidate_LinkFieldWithSlice(t *testing.T) {
	stem := &StemFile{
		Schema: map[string]SchemaField{
			"backlog_ids": {Type: "link", Required: false, Severity: "warn"},
		},
	}
	rec := &extract.Record{
		Path: "test.md",
		Frontmatter: map[string]any{
			"backlog_ids": []any{"[[B039]]", "[[B040]]"},
		},
	}
	errs := Validate(context.Background(), rec, stem)
	for _, e := range errs {
		if e.Rule == "link-format" {
			t.Errorf("unexpected link-format error: %s", e.Message)
		}
	}
}

func TestValidate_RequiredLinkFieldMissing(t *testing.T) {
	stem := &StemFile{
		Schema: map[string]SchemaField{
			"spec": {Type: "link", Required: true, Severity: "error"},
		},
	}
	rec := &extract.Record{
		Path:        "test.md",
		Frontmatter: map[string]any{},
	}
	errs := Validate(context.Background(), rec, stem)
	found := false
	for _, e := range errs {
		if e.Rule == "required" && e.Field == "spec" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected required error for missing link field")
	}
}

func TestValidate_PerRecordMatchFiltering_RequiredField(t *testing.T) {
	// Test that per-record schema with match filtering works correctly.
	// A field with RequiredMatch should only be required if the record path matches.
	stem := &StemFile{
		Path: "docs/tasks/.stem",
		Schema: map[string]SchemaField{
			"title": {Type: "string", Required: true, Source: "root/.stem"},
			// ejecutable_en is only required for T* (task) records
			"ejecutable_en": {
				Type:     "string",
				Required: false,
				RequiredMatch: &FieldMatch{
					Patterns: []string{"T*"},
				},
				Source: "docs/tasks/.stem",
			},
		},
	}

	t.Run("T-record has match-scoped field required", func(t *testing.T) {
		rec := &extract.Record{
			Path:        "docs/epics/E01/F01/T001-task.md",
			Type:        "markdown",
			Frontmatter: map[string]any{"title": "Test"},
		}
		// Simulate ResolveForRecord by applying match filtering
		filtered := FilterSchemaByMatch(pointerSchema(stem.Schema), rec.Path)
		effectiveStem := &StemFile{
			Path:   stem.Path,
			Schema: valueSchema(filtered),
		}

		errs := Validate(context.Background(), rec, effectiveStem)
		// Should have an error for missing ejecutable_en since it matches T*
		found := false
		for _, e := range errs {
			if e.Rule == "required" && e.Field == "ejecutable_en" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected required error for ejecutable_en in T-record")
		}
	})

	t.Run("F-record does not have match-scoped field required", func(t *testing.T) {
		rec := &extract.Record{
			Path:        "docs/epics/E01/F01-feature/README.md",
			Type:        "markdown",
			Frontmatter: map[string]any{"title": "Feature"},
		}
		// Simulate ResolveForRecord by applying match filtering
		filtered := FilterSchemaByMatch(pointerSchema(stem.Schema), rec.Path)
		effectiveStem := &StemFile{
			Path:   stem.Path,
			Schema: valueSchema(filtered),
		}

		errs := Validate(context.Background(), rec, effectiveStem)
		// Should NOT have an error for ejecutable_en since it doesn't match T*
		for _, e := range errs {
			if e.Rule == "required" && e.Field == "ejecutable_en" {
				t.Errorf("unexpected required error for ejecutable_en in F-record: %s", e.Message)
			}
		}
	})
}

// Helper functions for test support
func pointerSchema(schema map[string]SchemaField) map[string]*SchemaField {
	ptrSchema := make(map[string]*SchemaField, len(schema))
	for name, field := range schema {
		f := field
		ptrSchema[name] = &f
	}
	return ptrSchema
}

func valueSchema(ptrSchema map[string]*SchemaField) map[string]SchemaField {
	schema := make(map[string]SchemaField, len(ptrSchema))
	for name, field := range ptrSchema {
		schema[name] = *field
	}
	return schema
}
