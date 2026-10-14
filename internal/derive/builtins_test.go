package derive

import (
	"strings"
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
	tests := []struct {
		name       string
		expression string
		want       string
	}{
		{"strings", `concat("a", "-", "b")`, "a-b"},
		{"mixed scalars", `concat("task-", 42, "-", true, "-", false, "-", -7)`, "task-42-true-false--7"},
		{"string unchanged", `concat("  <int Value> café 🙂  ")`, "  <int Value> café 🙂  "},
		{"nil between strings", `concat("a", nil, "b")`, "ab"},
		{"nil alone", `concat(nil)`, ""},
		{"no arguments", `concat()`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalExpr(t, tt.expression, map[string]any{})
			if result != tt.want {
				t.Errorf("%s = %v, want %q", tt.expression, result, tt.want)
			}
		})
	}
}

func TestConcatIntegerTypes(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{"int", int(-42), "-42"},
		{"int8", int8(-128), "-128"},
		{"int16", int16(-32768), "-32768"},
		{"int32", int32(-2147483648), "-2147483648"},
		{"int64", int64(-1 << 63), "-9223372036854775808"},
		{"uint", uint(42), "42"},
		{"uint8", uint8(255), "255"},
		{"uint16", uint16(65535), "65535"},
		{"uint32", uint32(4294967295), "4294967295"},
		{"uint64", uint64(1<<64 - 1), "18446744073709551615"},
		{"uintptr", uintptr(42), "42"},
		{"zero", 0, "0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalExpr(t, `concat("value=", value)`, map[string]any{"value": tt.value})
			if want := "value=" + tt.want; result != want {
				t.Errorf("concat(value) = %v, want %q", result, want)
			}
		})
	}
}

func TestConcatRejectsUnsupportedTypes(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		value      any
		wantError  string
	}{
		{"list literal", `concat("prefix", [1, 2], "suffix")`, nil, "concat: unsupported argument type []interface {}"},
		{"map literal", `concat("prefix", {"a": 1}, "suffix")`, nil, "concat: unsupported argument type map[string]interface {}"},
		{"slice", `concat(value)`, []string{"a"}, "concat: unsupported argument type []string"},
		{"map", `concat(value)`, map[string]any{"a": 1}, "concat: unsupported argument type map[string]interface {}"},
		{"nil slice", `concat(value)`, []any(nil), "concat: unsupported argument type []interface {}"},
		{"nil map", `concat(value)`, map[string]any(nil), "concat: unsupported argument type map[string]interface {}"},
		{"float literal", `concat(3.14)`, nil, "concat: unsupported argument type float64"},
		{"float32", `concat(value)`, float32(1.5), "concat: unsupported argument type float32"},
		{"float64", `concat(value)`, float64(1.5), "concat: unsupported argument type float64"},
		{"complex64", `concat(value)`, complex64(1 + 2i), "concat: unsupported argument type complex64"},
		{"complex128", `concat(value)`, complex128(1 + 2i), "concat: unsupported argument type complex128"},
		{"array", `concat(value)`, [1]int{1}, "concat: unsupported argument type [1]int"},
		{"struct", `concat(value)`, struct{}{}, "concat: unsupported argument type struct {}"},
		{"pointer", `concat(value)`, new(int), "concat: unsupported argument type *int"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev := NewEvaluatorWithBuiltins()
			compiled, err := ev.Compile(tt.expression)
			if err != nil {
				t.Fatalf("Compile(%q): %v", tt.expression, err)
			}
			_, err = ev.Eval(compiled, map[string]any{"value": tt.value})
			if err == nil {
				t.Fatalf("Eval(%q): expected unsupported argument error", tt.expression)
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("Eval(%q): error = %q, want substring %q", tt.expression, err, tt.wantError)
			}
		})
	}
}

func TestBuiltinRejectsNonStringAtCompileTime(t *testing.T) {
	ev := newTestEval()
	// expr-lang enforces the type overload at compile time.
	// Passing an int to a func(string) builtin must fail compilation.
	badExprs := []string{
		"slugify(42)",
		"lower(true)",
		"upper(3.14)",
		"strlen(99)",
	}
	for _, expression := range badExprs {
		_, err := ev.Compile(expression)
		if err == nil {
			t.Errorf("Compile(%q): expected type error, got nil", expression)
		}
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
