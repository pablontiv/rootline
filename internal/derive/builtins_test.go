package derive

import (
	"testing"
)

func newTestEval() *Evaluator {
	return NewEvaluatorWithBuiltins()
}

func evalExpr(t *testing.T, expression string, env map[string]any) any {
	t.Helper()
	ev := newTestEval()
	compiled, err := ev.Compile(expression)
	if err != nil {
		t.Fatalf("Compile(%q): %v", expression, err)
	}
	result, err := ev.Eval(compiled, env)
	if err != nil {
		t.Fatalf("Eval(%q): %v", expression, err)
	}
	return result
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello World!", "hello-world"},
		{"  Leading Trailing  ", "leading-trailing"},
		{"already-slug", "already-slug"},
		{"UPPER CASE", "upper-case"},
		{"special@#chars", "special-chars"},
		{"múltiple   spaces", "m-ltiple-spaces"},
		{"", ""},
	}
	for _, tt := range tests {
		result := evalExpr(t, "slugify(s)", map[string]any{"s": tt.input})
		if result != tt.want {
			t.Errorf("slugify(%q) = %q, want %q", tt.input, result, tt.want)
		}
	}
}

func TestLower(t *testing.T) {
	result := evalExpr(t, "lower(x)", map[string]any{"x": "HELLO"})
	if result != "hello" {
		t.Errorf("lower(HELLO) = %v, want %q", result, "hello")
	}
}

func TestUpper(t *testing.T) {
	result := evalExpr(t, "upper(x)", map[string]any{"x": "hello"})
	if result != "HELLO" {
		t.Errorf("upper(hello) = %v, want %q", result, "HELLO")
	}
}

func TestTrim(t *testing.T) {
	result := evalExpr(t, "trim(x)", map[string]any{"x": "  hello  "})
	if result != "hello" {
		t.Errorf("trim = %v, want %q", result, "hello")
	}
}

func TestStrlen(t *testing.T) {
	result := evalExpr(t, "strlen(x)", map[string]any{"x": "hello"})
	if result != 5 {
		t.Errorf("strlen(hello) = %v, want 5", result)
	}
}

func TestLenArray(t *testing.T) {
	// expr built-in len() works on arrays.
	result := evalExpr(t, "len(items)", map[string]any{
		"items": []any{1, 2, 3},
	})
	if result != 3 {
		t.Errorf("len([1,2,3]) = %v, want 3", result)
	}
}

func TestCountWithPredicate(t *testing.T) {
	// expr built-in count() with closure syntax.
	children := []any{
		map[string]any{"estado": "Completed"},
		map[string]any{"estado": "Pending"},
		map[string]any{"estado": "Completed"},
	}
	result := evalExpr(t, `count(children, {.estado == "Completed"})`, map[string]any{
		"children": children,
	})
	if result != 2 {
		t.Errorf("count(completado) = %v, want 2", result)
	}
}

func TestAnyWithPredicate(t *testing.T) {
	children := []any{
		map[string]any{"estado": "Completed"},
		map[string]any{"estado": "Pending"},
	}
	result := evalExpr(t, `any(children, {.estado == "Pending"})`, map[string]any{
		"children": children,
	})
	if result != true {
		t.Errorf("any(pending) = %v, want true", result)
	}
}

func TestAllWithPredicate(t *testing.T) {
	children := []any{
		map[string]any{"estado": "Completed"},
		map[string]any{"estado": "Completed"},
	}
	result := evalExpr(t, `all(children, {.estado == "Completed"})`, map[string]any{
		"children": children,
	})
	if result != true {
		t.Errorf("all(completado) = %v, want true", result)
	}

	// Negative case.
	children = append(children, map[string]any{"estado": "Pending"})
	result = evalExpr(t, `all(children, {.estado == "Completed"})`, map[string]any{
		"children": children,
	})
	if result != false {
		t.Errorf("all(mixed) = %v, want false", result)
	}
}

func TestConcat(t *testing.T) {
	result := evalExpr(t, `concat("a", "-", "b")`, map[string]any{})
	if result != "a-b" {
		t.Errorf("concat = %v, want %q", result, "a-b")
	}
}

func TestNewEvaluatorWithBuiltins(t *testing.T) {
	ev := NewEvaluatorWithBuiltins()
	compiled, err := ev.Compile(`slugify("Hello World")`)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	result, err := ev.Eval(compiled, map[string]any{})
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	if result != "hello-world" {
		t.Errorf("result = %v, want %q", result, "hello-world")
	}
}
