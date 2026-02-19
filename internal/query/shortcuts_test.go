package query

import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func TestResolveShortcut_Estado(t *testing.T) {
	if got := ResolveShortcut("estado"); got != "frontmatter.estado" {
		t.Errorf("got %q, want frontmatter.estado", got)
	}
}

func TestResolveShortcut_Tipo(t *testing.T) {
	if got := ResolveShortcut("tipo"); got != "frontmatter.tipo" {
		t.Errorf("got %q, want frontmatter.tipo", got)
	}
}

func TestResolveShortcut_Path(t *testing.T) {
	if got := ResolveShortcut("path"); got != "path" {
		t.Errorf("got %q, want path", got)
	}
}

func TestResolveShortcut_Body(t *testing.T) {
	if got := ResolveShortcut("body"); got != "body" {
		t.Errorf("got %q, want body", got)
	}
}

func TestResolveShortcut_Type(t *testing.T) {
	if got := ResolveShortcut("type"); got != "type" {
		t.Errorf("got %q, want type", got)
	}
}

func TestResolveShortcut_DotPathPassthrough(t *testing.T) {
	if got := ResolveShortcut("frontmatter.custom"); got != "frontmatter.custom" {
		t.Errorf("got %q, want frontmatter.custom", got)
	}
}

func TestResolveShortcut_UnknownBareField(t *testing.T) {
	if got := ResolveShortcut("priority"); got != "frontmatter.priority" {
		t.Errorf("got %q, want frontmatter.priority", got)
	}
}

func TestResolveConditionShortcuts_Recursive(t *testing.T) {
	cond := &Condition{
		Op: OpAnd,
		Children: []Condition{
			{Op: OpEq, Field: "estado", Value: "Pending"},
			{Op: OpEq, Field: "tipo", Value: "lxc"},
		},
	}
	resolveConditionShortcuts(cond)

	if cond.Children[0].Field != "frontmatter.estado" {
		t.Errorf("children[0].field = %q", cond.Children[0].Field)
	}
	if cond.Children[1].Field != "frontmatter.tipo" {
		t.Errorf("children[1].field = %q", cond.Children[1].Field)
	}
}

func TestResolveConditionShortcuts_Nil(t *testing.T) {
	// Should not panic
	resolveConditionShortcuts(nil)
}

// Integration: shortcuts work through Execute
func TestExecute_WithShortcuts(t *testing.T) {
	records := []*extract.Record{
		{
			Path: "t1.md", Type: "markdown",
			Frontmatter: map[string]any{"estado": "Pending", "tipo": "lxc"},
		},
		{
			Path: "t2.md", Type: "markdown",
			Frontmatter: map[string]any{"estado": "Completado", "tipo": "vm"},
		},
	}

	// Use shortcut "estado" instead of "frontmatter.estado"
	result, err := Execute(records, &Query{
		Where: &Condition{Op: OpEq, Field: "estado", Value: "Pending"},
	})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	qr := result.(*QueryResult)
	if qr.Meta.Count != 1 {
		t.Errorf("count = %d, want 1", qr.Meta.Count)
	}
}

func TestExecute_ShortcutInAnd(t *testing.T) {
	records := []*extract.Record{
		{
			Path: "t1.md", Type: "markdown",
			Frontmatter: map[string]any{"estado": "Pending", "tipo": "servicio-docker"},
		},
		{
			Path: "t2.md", Type: "markdown",
			Frontmatter: map[string]any{"estado": "Pending", "tipo": "lxc"},
		},
	}

	result, _ := Execute(records, &Query{
		Where: &Condition{
			Op: OpAnd,
			Children: []Condition{
				{Op: OpEq, Field: "tipo", Value: "servicio-docker"},
				{Op: OpEq, Field: "estado", Value: "Pending"},
			},
		},
	})
	qr := result.(*QueryResult)
	if qr.Meta.Count != 1 {
		t.Errorf("count = %d, want 1", qr.Meta.Count)
	}
}
