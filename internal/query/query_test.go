package query

import (
	"encoding/json"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func makeRecords() []*extract.Record {
	return []*extract.Record{
		{
			Path: "tasks/T001.md", Type: "markdown",
			Frontmatter: map[string]any{"estado": "Pending", "tipo": "servicio-docker", "title": "Deploy Redis"},
			Body:        "# Deploy Redis\n\nSet up Redis container.",
		},
		{
			Path: "tasks/T002.md", Type: "markdown",
			Frontmatter: map[string]any{"estado": "Completado", "tipo": "modulo-sistema", "title": "Auth Module"},
			Body:        "# Auth Module\n\nImplement authentication.",
		},
		{
			Path: "tasks/T003.md", Type: "markdown",
			Frontmatter: map[string]any{"estado": "Pending", "tipo": "lxc", "title": "LXC Container"},
			Body:        "# LXC Container\n\nProvision infrastructure.",
		},
		{
			Path: "tasks/T004.md", Type: "markdown",
			Frontmatter: map[string]any{"estado": "Pending", "tipo": "vm", "tags": []any{"infra", "docker"}},
			Body:        "# VM Setup\n\nMigration required.",
		},
	}
}

func TestExecute_Eq(t *testing.T) {
	records := makeRecords()
	result, err := Execute(records, &Query{
		Where: &Condition{Op: OpEq, Field: "estado", Value: "Pending"},
	})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	qr := result.(*QueryResult)
	if qr.Meta.Count != 3 {
		t.Errorf("count = %d, want 3", qr.Meta.Count)
	}
}

func TestExecute_Ne(t *testing.T) {
	records := makeRecords()
	result, err := Execute(records, &Query{
		Where: &Condition{Op: OpNe, Field: "estado", Value: "Completado"},
	})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	qr := result.(*QueryResult)
	if qr.Meta.Count != 3 {
		t.Errorf("count = %d, want 3", qr.Meta.Count)
	}
}

func TestExecute_In(t *testing.T) {
	records := makeRecords()
	result, err := Execute(records, &Query{
		Where: &Condition{Op: OpIn, Field: "tipo", Values: []string{"lxc", "vm"}},
	})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	qr := result.(*QueryResult)
	if qr.Meta.Count != 2 {
		t.Errorf("count = %d, want 2", qr.Meta.Count)
	}
}

func TestExecute_Contains_Field(t *testing.T) {
	records := makeRecords()
	result, err := Execute(records, &Query{
		Where: &Condition{Op: OpContains, Field: "title", Value: "Redis"},
	})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	qr := result.(*QueryResult)
	if qr.Meta.Count != 1 {
		t.Errorf("count = %d, want 1", qr.Meta.Count)
	}
	if qr.Rows[0].Path != "tasks/T001.md" {
		t.Errorf("path = %q", qr.Rows[0].Path)
	}
}

func TestExecute_Contains_Body(t *testing.T) {
	records := makeRecords()
	result, err := Execute(records, &Query{
		Where: &Condition{Op: OpContains, Field: "body", Value: "migration"},
	})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	qr := result.(*QueryResult)
	// Only T004 body contains "migration" (case-sensitive)
	// Actually "Migration" is capitalized in T004
	if qr.Meta.Count != 0 {
		t.Errorf("count = %d, want 0 (case-sensitive)", qr.Meta.Count)
	}

	// Try with correct case
	result2, _ := Execute(records, &Query{
		Where: &Condition{Op: OpContains, Field: "body", Value: "Migration"},
	})
	qr2 := result2.(*QueryResult)
	if qr2.Meta.Count != 1 {
		t.Errorf("count = %d, want 1", qr2.Meta.Count)
	}
}

func TestExecute_Exists(t *testing.T) {
	records := makeRecords()
	result, err := Execute(records, &Query{
		Where: &Condition{Op: OpExists, Field: "title"},
	})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	qr := result.(*QueryResult)
	// T001, T002, T003 have title. T004 does not.
	if qr.Meta.Count != 3 {
		t.Errorf("count = %d, want 3", qr.Meta.Count)
	}
}

func TestExecute_And(t *testing.T) {
	records := makeRecords()
	result, err := Execute(records, &Query{
		Where: &Condition{
			Op: OpAnd,
			Children: []Condition{
				{Op: OpEq, Field: "tipo", Value: "servicio-docker"},
				{Op: OpEq, Field: "estado", Value: "Pending"},
			},
		},
	})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	qr := result.(*QueryResult)
	if qr.Meta.Count != 1 {
		t.Errorf("count = %d, want 1", qr.Meta.Count)
	}
	if qr.Rows[0].Path != "tasks/T001.md" {
		t.Errorf("path = %q", qr.Rows[0].Path)
	}
}

func TestExecute_Count(t *testing.T) {
	records := makeRecords()
	result, err := Execute(records, &Query{
		Where: &Condition{Op: OpEq, Field: "estado", Value: "Pending"},
		Count: true,
	})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	cr := result.(*CountResult)
	if cr.Count != 3 {
		t.Errorf("count = %d, want 3", cr.Count)
	}
	if cr.Kind != "rootline/count" {
		t.Errorf("kind = %q", cr.Kind)
	}
	if cr.Version != 1 {
		t.Errorf("version = %d", cr.Version)
	}
}

func TestExecute_Limit(t *testing.T) {
	records := makeRecords()
	result, err := Execute(records, &Query{Limit: 2})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	qr := result.(*QueryResult)
	if qr.Meta.Count != 2 {
		t.Errorf("count = %d, want 2", qr.Meta.Count)
	}
	if len(qr.Rows) != 2 {
		t.Errorf("rows = %d, want 2", len(qr.Rows))
	}
}

func TestExecute_MissingFieldEq_NoMatch(t *testing.T) {
	records := makeRecords()
	result, _ := Execute(records, &Query{
		Where: &Condition{Op: OpEq, Field: "nonexistent", Value: "x"},
	})
	qr := result.(*QueryResult)
	if qr.Meta.Count != 0 {
		t.Errorf("count = %d, want 0", qr.Meta.Count)
	}
}

func TestExecute_MissingFieldNe_Matches(t *testing.T) {
	records := makeRecords()
	result, _ := Execute(records, &Query{
		Where: &Condition{Op: OpNe, Field: "nonexistent", Value: "x"},
	})
	qr := result.(*QueryResult)
	if qr.Meta.Count != 4 {
		t.Errorf("count = %d, want 4 (all match since field is absent)", qr.Meta.Count)
	}
}

func TestExecute_ArrayFieldEq(t *testing.T) {
	records := makeRecords()
	result, _ := Execute(records, &Query{
		Where: &Condition{Op: OpEq, Field: "tags", Value: "docker"},
	})
	qr := result.(*QueryResult)
	if qr.Meta.Count != 1 {
		t.Errorf("count = %d, want 1", qr.Meta.Count)
	}
	if qr.Rows[0].Path != "tasks/T004.md" {
		t.Errorf("path = %q", qr.Rows[0].Path)
	}
}

func TestExecute_ArrayFieldIn(t *testing.T) {
	records := makeRecords()
	result, _ := Execute(records, &Query{
		Where: &Condition{Op: OpIn, Field: "tags", Values: []string{"infra", "networking"}},
	})
	qr := result.(*QueryResult)
	if qr.Meta.Count != 1 {
		t.Errorf("count = %d, want 1 (T004 has infra tag)", qr.Meta.Count)
	}
}

func TestExecute_ArrayFieldContains(t *testing.T) {
	records := makeRecords()
	result, _ := Execute(records, &Query{
		Where: &Condition{Op: OpContains, Field: "tags", Value: "dock"},
	})
	qr := result.(*QueryResult)
	if qr.Meta.Count != 1 {
		t.Errorf("count = %d, want 1 (T004 tag 'docker' contains 'dock')", qr.Meta.Count)
	}
}

func TestExecute_FrontmatterDotPath(t *testing.T) {
	records := makeRecords()
	result, _ := Execute(records, &Query{
		Where: &Condition{Op: OpEq, Field: "frontmatter.estado", Value: "Pending"},
	})
	qr := result.(*QueryResult)
	if qr.Meta.Count != 3 {
		t.Errorf("count = %d, want 3", qr.Meta.Count)
	}
}

func TestExecute_NoWhere_AllRecords(t *testing.T) {
	records := makeRecords()
	result, _ := Execute(records, &Query{})
	qr := result.(*QueryResult)
	if qr.Meta.Count != 4 {
		t.Errorf("count = %d, want 4", qr.Meta.Count)
	}
}

func TestExecute_EmptyRecords(t *testing.T) {
	result, _ := Execute(nil, &Query{})
	qr := result.(*QueryResult)
	if qr.Meta.Count != 0 {
		t.Errorf("count = %d, want 0", qr.Meta.Count)
	}
	if len(qr.Rows) != 0 {
		t.Errorf("rows should be empty, not nil")
	}
}

func TestQueryResult_JSON(t *testing.T) {
	records := makeRecords()[:1]
	result, _ := Execute(records, &Query{})

	data, err := ToJSON(result)
	if err != nil {
		t.Fatalf("ToJSON error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}

	if parsed["version"].(float64) != 1 {
		t.Errorf("version = %v", parsed["version"])
	}
	if parsed["kind"] != "rootline/query" {
		t.Errorf("kind = %v", parsed["kind"])
	}
}

func TestCountResult_JSON(t *testing.T) {
	records := makeRecords()
	result, _ := Execute(records, &Query{Count: true})

	data, err := ToJSON(result)
	if err != nil {
		t.Fatalf("ToJSON error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}

	if parsed["kind"] != "rootline/count" {
		t.Errorf("kind = %v", parsed["kind"])
	}
	if parsed["count"].(float64) != 4 {
		t.Errorf("count = %v", parsed["count"])
	}
}

func TestExecute_PathField(t *testing.T) {
	records := makeRecords()
	result, _ := Execute(records, &Query{
		Where: &Condition{Op: OpContains, Field: "path", Value: "T001"},
	})
	qr := result.(*QueryResult)
	if qr.Meta.Count != 1 {
		t.Errorf("count = %d, want 1", qr.Meta.Count)
	}
}

func TestExecute_UnknownOperator(t *testing.T) {
	records := makeRecords()
	result, _ := Execute(records, &Query{
		Where: &Condition{Op: "unknown", Field: "x", Value: "y"},
	})
	qr := result.(*QueryResult)
	if qr.Meta.Count != 0 {
		t.Errorf("unknown op should match nothing, got %d", qr.Meta.Count)
	}
}

func TestExecute_LimitGreaterThanResults(t *testing.T) {
	records := makeRecords()
	result, _ := Execute(records, &Query{Limit: 100})
	qr := result.(*QueryResult)
	if qr.Meta.Count != 4 {
		t.Errorf("count = %d, want 4", qr.Meta.Count)
	}
}
