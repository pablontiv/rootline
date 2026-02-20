package query

import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func makeExprRecords() []*extract.Record {
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
			Body:        "# LXC Container\n\nMigration required.",
		},
		{
			Path: "tasks/T004.md", Type: "markdown",
			Frontmatter: map[string]any{"estado": "Especificado", "tipo": "vm", "tags": []any{"infra", "docker"}},
			Body:        "# VM Setup",
		},
	}
}

func TestMatchRecord_Eq(t *testing.T) {
	rec := makeExprRecords()[0]
	prog, err := CompileWhere("estado == 'Pending'")
	if err != nil {
		t.Fatal(err)
	}
	match, err := MatchRecord(prog, rec)
	if err != nil {
		t.Fatal(err)
	}
	if !match {
		t.Error("expected match for estado == 'Pending'")
	}
}

func TestMatchRecord_Ne(t *testing.T) {
	rec := makeExprRecords()[0]
	prog, err := CompileWhere("estado != 'Completado'")
	if err != nil {
		t.Fatal(err)
	}
	match, err := MatchRecord(prog, rec)
	if err != nil {
		t.Fatal(err)
	}
	if !match {
		t.Error("expected match for estado != 'Completado'")
	}
}

func TestMatchRecord_In(t *testing.T) {
	rec := makeExprRecords()[0]
	prog, err := CompileWhere("estado in ['Pending', 'Especificado']")
	if err != nil {
		t.Fatal(err)
	}
	match, err := MatchRecord(prog, rec)
	if err != nil {
		t.Fatal(err)
	}
	if !match {
		t.Error("expected match for estado in ['Pending', 'Especificado']")
	}
}

func TestMatchRecord_And(t *testing.T) {
	rec := makeExprRecords()[0]
	prog, err := CompileWhere("tipo == 'servicio-docker' && estado == 'Pending'")
	if err != nil {
		t.Fatal(err)
	}
	match, err := MatchRecord(prog, rec)
	if err != nil {
		t.Fatal(err)
	}
	if !match {
		t.Error("expected match for AND condition")
	}
}

func TestMatchRecord_Or(t *testing.T) {
	rec := makeExprRecords()[2] // tipo: lxc
	prog, err := CompileWhere("tipo == 'lxc' || tipo == 'vm'")
	if err != nil {
		t.Fatal(err)
	}
	match, err := MatchRecord(prog, rec)
	if err != nil {
		t.Fatal(err)
	}
	if !match {
		t.Error("expected match for OR condition")
	}
}

func TestMatchRecord_Contains(t *testing.T) {
	rec := makeExprRecords()[2] // body has "Migration"
	prog, err := CompileWhere("body contains 'Migration'")
	if err != nil {
		t.Fatal(err)
	}
	match, err := MatchRecord(prog, rec)
	if err != nil {
		t.Fatal(err)
	}
	if !match {
		t.Error("expected match for body contains 'Migration'")
	}
}

func TestMatchRecord_NoMatch(t *testing.T) {
	rec := makeExprRecords()[1] // estado: Completado
	prog, err := CompileWhere("estado == 'Pending'")
	if err != nil {
		t.Fatal(err)
	}
	match, err := MatchRecord(prog, rec)
	if err != nil {
		t.Fatal(err)
	}
	if match {
		t.Error("expected no match for Completado record")
	}
}

func TestMatchRecord_UndefinedField(t *testing.T) {
	rec := makeExprRecords()[0] // no 'tags' field
	prog, err := CompileWhere("nonexistent == 'x'")
	if err != nil {
		t.Fatal(err)
	}
	match, err := MatchRecord(prog, rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if match {
		t.Error("expected no match for undefined field")
	}
}

func TestMatchRecord_NilCheck(t *testing.T) {
	rec := makeExprRecords()[0] // has 'tipo'
	prog, err := CompileWhere("tipo != nil")
	if err != nil {
		t.Fatal(err)
	}
	match, err := MatchRecord(prog, rec)
	if err != nil {
		t.Fatal(err)
	}
	if !match {
		t.Error("expected match for tipo != nil")
	}
}

func TestCompileWhere_InvalidExpr(t *testing.T) {
	_, err := CompileWhere("== invalid syntax")
	if err == nil {
		t.Error("expected error for invalid expression")
	}
}

func TestExecuteExpr_FiltersPending(t *testing.T) {
	records := makeExprRecords()
	result, err := ExecuteExpr(records, "estado == 'Pending'", &Query{})
	if err != nil {
		t.Fatal(err)
	}
	qr := result.(*QueryResult)
	if qr.Meta.Count != 2 {
		t.Errorf("count = %d, want 2", qr.Meta.Count)
	}
}

func TestExecuteExpr_InOperator(t *testing.T) {
	records := makeExprRecords()
	result, err := ExecuteExpr(records, "estado in ['Pending', 'Especificado']", &Query{})
	if err != nil {
		t.Fatal(err)
	}
	qr := result.(*QueryResult)
	if qr.Meta.Count != 3 {
		t.Errorf("count = %d, want 3 (2 Pending + 1 Especificado)", qr.Meta.Count)
	}
}

func TestExecuteExpr_Count(t *testing.T) {
	records := makeExprRecords()
	result, err := ExecuteExpr(records, "estado == 'Pending'", &Query{Count: true})
	if err != nil {
		t.Fatal(err)
	}
	cr := result.(*CountResult)
	if cr.Count != 2 {
		t.Errorf("count = %d, want 2", cr.Count)
	}
}

func TestExecuteExpr_Limit(t *testing.T) {
	records := makeExprRecords()
	result, err := ExecuteExpr(records, "", &Query{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	qr := result.(*QueryResult)
	if len(qr.Rows) != 2 {
		t.Errorf("rows = %d, want 2", len(qr.Rows))
	}
}

func TestExecuteExpr_NoWhere(t *testing.T) {
	records := makeExprRecords()
	result, err := ExecuteExpr(records, "", &Query{})
	if err != nil {
		t.Fatal(err)
	}
	qr := result.(*QueryResult)
	if qr.Meta.Count != 4 {
		t.Errorf("count = %d, want 4", qr.Meta.Count)
	}
}

func TestExecuteExpr_ComplexAnd(t *testing.T) {
	records := makeExprRecords()
	result, err := ExecuteExpr(records, "estado == 'Pending' && tipo == 'lxc'", &Query{})
	if err != nil {
		t.Fatal(err)
	}
	qr := result.(*QueryResult)
	if qr.Meta.Count != 1 {
		t.Errorf("count = %d, want 1", qr.Meta.Count)
	}
	if qr.Rows[0].Path != "tasks/T003.md" {
		t.Errorf("path = %q, want tasks/T003.md", qr.Rows[0].Path)
	}
}

func TestExecuteExpr_PathContains(t *testing.T) {
	records := makeExprRecords()
	result, err := ExecuteExpr(records, "path contains 'T001'", &Query{})
	if err != nil {
		t.Fatal(err)
	}
	qr := result.(*QueryResult)
	if qr.Meta.Count != 1 {
		t.Errorf("count = %d, want 1", qr.Meta.Count)
	}
}
