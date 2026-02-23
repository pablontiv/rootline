package rules

import (
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

	errs := Validate(record, stem)
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

	errs := Validate(record, stem)
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

	errs := Validate(record, stem)
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

	errs := Validate(record, stem)
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

	errs := Validate(record, stem)
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

	errs := Validate(record, stem)
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

	errs := Validate(record, stem)
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

	errs := Validate(record, stem)
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

	errs := Validate(record, stem)
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

	errs := Validate(record, stem)
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

	errs := Validate(record, stem)
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

	errs := Validate(record, stem)
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

	errs := Validate(record, stem)
	if len(errs) != 0 {
		t.Errorf("got %d errors, want 0 for valid doc", len(errs))
	}
}

func TestValidate_NilStem(t *testing.T) {
	record := makeRecord(map[string]any{"title": "Hello"})
	errs := Validate(record, nil)
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

	errs := Validate(record, stem)
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

	errs := Validate(record, stem)
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

	errs := Validate(record, stem)
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

	errs := Validate(record, stem)
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
	errs := Validate(record, stem)
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
	errs := Validate(record, stem)
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
	errs := Validate(record, stem)
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
	errs := Validate(record, stem)
	if len(errs) != 0 {
		t.Errorf("got %d errors, want 0 (custom index name match)", len(errs))
	}

	// README.md is NOT the index file when custom name is set — should error
	record2 := &extract.Record{
		Path:        "docs/README.md",
		Type:        "markdown",
		Frontmatter: map[string]any{},
	}
	errs2 := Validate(record2, stem)
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
	errs := Validate(record, stem)
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
	errs := Validate(record, stem)
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
	errs := Validate(record, stem)
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
	errs := Validate(record, stem)
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
	errs := Validate(record, stem)
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
	errs := Validate(record, stem)
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
		errs := Validate(record, stem)
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
	errs := Validate(record, stem)
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
	errs := Validate(record, stem)
	if len(errs) != 0 {
		t.Errorf("got %d errors, want 0 (no schema = permissive): %v", len(errs), errs)
	}
}
